// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: AGPL-3.0-or-later

use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{
    env,
    path::{Path, PathBuf},
    time::{SystemTime, UNIX_EPOCH},
};

#[cfg(unix)]
use std::os::unix::fs::PermissionsExt;

const AGENT_CONTROL_REGISTRY: &str = include_str!("../../../agent-control/v1/registry.json");

#[derive(Debug, Deserialize)]
struct Registry {
    transports: Vec<RegistryTransport>,
}

#[derive(Debug, Deserialize)]
struct RegistryTransport {
    id: String,
    display_name: String,
    runtime_probes: Vec<String>,
    support: Value,
}

#[derive(Clone, Debug, Serialize)]
pub struct TransportState {
    id: String,
    display_name: String,
    candidate_runtime_available: bool,
    runtime_ready: bool,
}

#[derive(Clone, Debug, Serialize)]
pub struct ProviderState {
    id: &'static str,
    display_name: &'static str,
    installed: bool,
    version: Option<String>,
    adapter_status: &'static str,
    connection_model: &'static str,
    remote_control_supported: bool,
    running: Option<bool>,
    actions: Vec<&'static str>,
}

#[derive(Clone, Debug, Serialize)]
pub struct AgentIntegrationReport {
    schema_version: u8,
    generated_at: u64,
    embedded_client_ready: bool,
    first_party_gateway_ready: bool,
    telegram_ready: bool,
    providers: Vec<ProviderState>,
    transports: Vec<TransportState>,
}

fn generated_at() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs()
}

#[cfg(unix)]
fn executable_file(path: &Path) -> bool {
    path.metadata()
        .map(|metadata| metadata.is_file() && metadata.permissions().mode() & 0o111 != 0)
        .unwrap_or(false)
}

#[cfg(windows)]
fn executable_file(path: &Path) -> bool {
    path.is_file()
}

#[cfg(unix)]
fn command_candidates(name: &str) -> Vec<String> {
    vec![name.to_owned()]
}

#[cfg(windows)]
fn windows_command_candidates(name: &str, pathext: Option<&str>) -> Vec<String> {
    let mut extensions: Vec<String> = pathext
        .unwrap_or(".COM;.EXE;.BAT;.CMD")
        .split(';')
        .filter_map(|extension| {
            let extension = extension.trim();
            let suffix = extension.strip_prefix('.')?;
            if suffix.is_empty() || !suffix.bytes().all(|byte| byte.is_ascii_alphanumeric()) {
                None
            } else {
                Some(format!(".{}", suffix.to_ascii_lowercase()))
            }
        })
        .collect();
    extensions.sort();
    extensions.dedup();
    extensions
        .into_iter()
        .map(|extension| format!("{name}{extension}"))
        .collect()
}

#[cfg(windows)]
fn command_candidates(name: &str) -> Vec<String> {
    let pathext = env::var("PATHEXT").ok();
    windows_command_candidates(name, pathext.as_deref())
}

fn command_path(name: &str) -> Option<PathBuf> {
    if name.is_empty()
        || name.contains('/')
        || name.contains('\\')
        || !name
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || matches!(byte, b'-' | b'_'))
    {
        return None;
    }

    let mut directories: Vec<PathBuf> = env::var_os("PATH")
        .map(|path| env::split_paths(&path).collect())
        .unwrap_or_default();
    if cfg!(target_os = "windows") {
        directories.extend(
            [r"C:\\Program Files\\nodejs", r"C:\\Program Files\\Codex"]
                .into_iter()
                .map(PathBuf::from),
        );
    } else {
        if let Some(home) = env::var_os("HOME")
            .map(PathBuf::from)
            .filter(|home| home.is_absolute())
        {
            directories.extend([home.join(".local/bin"), home.join(".linuxbrew/bin")]);
        }
        directories.extend(
            ["/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin"]
                .into_iter()
                .map(PathBuf::from),
        );
    }

    let candidates = command_candidates(name);
    directories
        .into_iter()
        .filter(|directory| directory.is_absolute())
        .find_map(|directory| {
            candidates
                .iter()
                .map(|candidate| directory.join(candidate))
                .find(|candidate| executable_file(candidate))
                .and_then(|candidate| candidate.canonicalize().ok())
        })
}

fn provider_state(id: &'static str, display_name: &'static str, command: &str) -> ProviderState {
    let installed = command_path(command).is_some();
    ProviderState {
        id,
        display_name,
        installed,
        version: None,
        adapter_status: if installed {
            "discovery-only"
        } else {
            "unavailable"
        },
        connection_model: "diagnostics-only",
        remote_control_supported: false,
        running: None,
        actions: Vec::new(),
    }
}

fn transport_states() -> Result<Vec<TransportState>, String> {
    let registry: Registry = serde_json::from_str(AGENT_CONTROL_REGISTRY)
        .map_err(|error| format!("Embedded agent-control registry is invalid: {error}"))?;
    let platform = env::consts::OS;
    Ok(registry
        .transports
        .into_iter()
        .map(|transport| {
            let candidate_runtime_available = !transport.runtime_probes.is_empty()
                && transport
                    .runtime_probes
                    .iter()
                    .all(|probe| command_path(probe).is_some());
            let platform_implemented =
                transport.support.get(platform).and_then(Value::as_str) == Some("implemented");
            TransportState {
                id: transport.id,
                display_name: transport.display_name,
                candidate_runtime_available,
                runtime_ready: platform_implemented && candidate_runtime_available,
            }
        })
        .collect())
}

pub fn integration_report() -> Result<AgentIntegrationReport, String> {
    Ok(AgentIntegrationReport {
        schema_version: 1,
        generated_at: generated_at(),
        embedded_client_ready: false,
        first_party_gateway_ready: false,
        telegram_ready: false,
        providers: vec![
            provider_state("codex", "Codex", "codex"),
            provider_state("claude-code", "Claude Code", "claude"),
        ],
        transports: transport_states()?,
    })
}

#[tauri::command]
pub async fn get_agent_integrations() -> Result<AgentIntegrationReport, String> {
    tauri::async_runtime::spawn_blocking(integration_report)
        .await
        .map_err(|error| error.to_string())?
}

#[cfg(test)]
mod tests {
    use super::*;
    #[cfg(unix)]
    use std::fs;

    #[test]
    fn provider_discovery_never_exposes_actions_or_runtime_readiness() {
        let provider = provider_state("test", "Test", "definitely-not-a-mazzy-agent-binary");
        assert!(!provider.remote_control_supported);
        assert!(provider.running.is_none());
        assert!(provider.actions.is_empty());
    }

    #[test]
    fn embedded_transport_registry_is_parseable() {
        let transports = transport_states().expect("embedded registry");
        assert_eq!(transports.len(), 7);
        assert!(transports.iter().all(|transport| !transport.runtime_ready));
    }

    #[cfg(unix)]
    #[test]
    fn unix_discovery_requires_an_executable_regular_file() {
        let root = env::temp_dir().join(format!(
            "mazzy-agent-discovery-{}-{}",
            std::process::id(),
            generated_at()
        ));
        fs::create_dir_all(&root).expect("temporary directory");
        let candidate = root.join("agent");
        fs::write(&candidate, b"#!/bin/sh\n").expect("temporary executable");
        fs::set_permissions(&candidate, fs::Permissions::from_mode(0o600))
            .expect("non-executable mode");
        assert!(!executable_file(&candidate));
        fs::set_permissions(&candidate, fs::Permissions::from_mode(0o700))
            .expect("executable mode");
        assert!(executable_file(&candidate));
        fs::remove_dir_all(root).expect("remove temporary directory");
    }

    #[cfg(windows)]
    #[test]
    fn windows_discovery_requires_a_valid_pathext_suffix() {
        let candidates = windows_command_candidates("codex", Some(".EXE;.CMD;bad;.;.CMD"));
        assert_eq!(candidates, vec!["codex.cmd", "codex.exe"]);
        assert!(!candidates.iter().any(|candidate| candidate == "codex"));
    }
}
