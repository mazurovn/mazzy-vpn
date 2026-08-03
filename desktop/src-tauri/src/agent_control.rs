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
    windows_command_candidates(name, None)
}

fn trusted_command_name(name: &str) -> Option<&'static str> {
    match name {
        "claude" => Some("claude"),
        "codex" => Some("codex"),
        "mazzy-agent" => Some("mazzy-agent"),
        "mazzy-agent-transport-iroh" => Some("mazzy-agent-transport-iroh"),
        "mazzy-agent-transport-libp2p" => Some("mazzy-agent-transport-libp2p"),
        "tailscale" => Some("tailscale"),
        _ => None,
    }
}

fn trusted_command_directories() -> Vec<PathBuf> {
    if cfg!(target_os = "windows") {
        [r"C:\Program Files\nodejs", r"C:\Program Files\Codex"]
            .into_iter()
            .map(PathBuf::from)
            .collect()
    } else {
        ["/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin"]
            .into_iter()
            .map(PathBuf::from)
            .collect()
    }
}

fn command_available(name: &str) -> bool {
    let Some(name) = trusted_command_name(name) else {
        return false;
    };
    let candidates = command_candidates(name);
    trusted_command_directories().into_iter().any(|directory| {
        candidates
            .iter()
            .map(|candidate| directory.join(candidate))
            .any(|candidate| executable_file(&candidate))
    })
}

fn provider_state(id: &'static str, display_name: &'static str, command: &str) -> ProviderState {
    let installed = command_available(command);
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
                    .all(|probe| command_available(probe));
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
        assert!(executable_file(Path::new("/bin/sh")));
        let manifest = Path::new(env!("CARGO_MANIFEST_DIR")).join("Cargo.toml");
        assert!(!executable_file(&manifest));
    }

    #[test]
    fn discovery_rejects_commands_outside_the_static_allowlist() {
        assert!(!command_available("../codex"));
        assert!(!command_available("unregistered-agent"));
    }

    #[cfg(windows)]
    #[test]
    fn windows_discovery_requires_a_valid_pathext_suffix() {
        let candidates = windows_command_candidates("codex", Some(".EXE;.CMD;bad;.;.CMD"));
        assert_eq!(candidates, vec!["codex.cmd", "codex.exe"]);
        assert!(!candidates.iter().any(|candidate| candidate == "codex"));
    }
}
