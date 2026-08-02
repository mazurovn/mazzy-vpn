// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: AGPL-3.0-or-later

use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{
    env, io,
    io::Read,
    path::{Path, PathBuf},
    process::{Command, ExitStatus, Stdio},
    thread,
    time::{Duration, Instant, SystemTime, UNIX_EPOCH},
};

const AGENT_CONTROL_REGISTRY: &str = include_str!("../../../agent-control/v1/registry.json");
const MAX_STREAM_BYTES: usize = 32 * 1024;
const COMMAND_TIMEOUT: Duration = Duration::from_secs(12);

#[derive(Clone, Copy, Debug, Deserialize, PartialEq, Eq)]
pub enum AgentOperation {
    #[serde(rename = "codex-remote-start")]
    Start,
    #[serde(rename = "codex-remote-pair")]
    Pair,
    #[serde(rename = "codex-remote-stop")]
    Stop,
}

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

#[derive(Clone, Debug, Serialize)]
pub struct PairingGrant {
    code: String,
    expires_at: String,
}

#[derive(Clone, Debug, Serialize)]
pub struct AgentOperationResult {
    success: bool,
    operation: &'static str,
    message_key: &'static str,
    running: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pairing: Option<PairingGrant>,
}

struct LimitedOutput {
    status: ExitStatus,
    stdout: Vec<u8>,
    timed_out: bool,
}

fn generated_at() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs()
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

    let candidates = if cfg!(target_os = "windows") {
        vec![name.to_owned(), format!("{name}.exe")]
    } else {
        vec![name.to_owned()]
    };
    directories.into_iter().find_map(|directory| {
        candidates
            .iter()
            .map(|candidate| directory.join(candidate))
            .find(|candidate| candidate.is_file())
    })
}

fn drain_bounded(mut reader: impl Read, limit: usize) -> io::Result<Vec<u8>> {
    let mut retained = Vec::with_capacity(limit.min(8192));
    let mut buffer = [0_u8; 8192];
    loop {
        let read = reader.read(&mut buffer)?;
        if read == 0 {
            break;
        }
        let keep = limit.saturating_sub(retained.len()).min(read);
        retained.extend_from_slice(&buffer[..keep]);
    }
    Ok(retained)
}

fn run_limited(program: &Path, args: &[&str]) -> io::Result<LimitedOutput> {
    let mut child = Command::new(program)
        .args(args)
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()?;
    let stdout = child
        .stdout
        .take()
        .ok_or_else(|| io::Error::other("agent adapter stdout is unavailable"))?;
    let stderr = child
        .stderr
        .take()
        .ok_or_else(|| io::Error::other("agent adapter stderr is unavailable"))?;
    let stdout_reader = thread::spawn(move || drain_bounded(stdout, MAX_STREAM_BYTES));
    let stderr_reader = thread::spawn(move || drain_bounded(stderr, MAX_STREAM_BYTES));
    let started = Instant::now();
    let mut timed_out = false;
    let status = loop {
        if let Some(status) = child.try_wait()? {
            break status;
        }
        if started.elapsed() >= COMMAND_TIMEOUT {
            timed_out = true;
            let _ = child.kill();
            break child.wait()?;
        }
        thread::sleep(Duration::from_millis(40));
    };
    let stdout = stdout_reader
        .join()
        .map_err(|_| io::Error::other("agent adapter stdout reader panicked"))??;
    let _stderr = stderr_reader
        .join()
        .map_err(|_| io::Error::other("agent adapter stderr reader panicked"))??;
    Ok(LimitedOutput {
        status,
        stdout,
        timed_out,
    })
}

fn first_clean_line(bytes: &[u8]) -> Option<String> {
    let text = std::str::from_utf8(bytes).ok()?;
    let line = text.lines().find(|line| !line.trim().is_empty())?.trim();
    if line.len() > 160 || line.chars().any(|character| character.is_control()) {
        return None;
    }
    Some(line.to_owned())
}

fn output_contains(output: io::Result<LimitedOutput>, needle: &str) -> bool {
    output.is_ok_and(|output| {
        output.status.success()
            && !output.timed_out
            && String::from_utf8_lossy(&output.stdout).contains(needle)
    })
}

fn provider_state(
    id: &'static str,
    display_name: &'static str,
    command: &str,
    remote_needle: &str,
) -> ProviderState {
    let Some(path) = command_path(command) else {
        return ProviderState {
            id,
            display_name,
            installed: false,
            version: None,
            adapter_status: "unavailable",
            connection_model: "none",
            remote_control_supported: false,
            running: None,
            actions: Vec::new(),
        };
    };
    let version = run_limited(&path, &["--version"])
        .ok()
        .filter(|output| output.status.success() && !output.timed_out)
        .and_then(|output| first_clean_line(&output.stdout));
    let remote_control_supported = output_contains(run_limited(&path, &["--help"]), remote_needle);

    if id == "codex" {
        let running = remote_control_supported.then(|| {
            run_limited(&path, &["app-server", "daemon", "version"])
                .is_ok_and(|output| output.status.success() && !output.timed_out)
        });
        ProviderState {
            id,
            display_name,
            installed: true,
            version,
            adapter_status: if remote_control_supported {
                "implemented"
            } else {
                "unsupported-version"
            },
            connection_model: "vendor-native",
            remote_control_supported,
            running,
            actions: if remote_control_supported {
                vec![
                    "codex-remote-start",
                    "codex-remote-pair",
                    "codex-remote-stop",
                ]
            } else {
                Vec::new()
            },
        }
    } else {
        ProviderState {
            id,
            display_name,
            installed: true,
            version,
            adapter_status: if remote_control_supported {
                "discovery-only"
            } else {
                "unsupported-version"
            },
            connection_model: "interactive-vendor-native",
            remote_control_supported,
            running: None,
            actions: Vec::new(),
        }
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
    let codex = provider_state("codex", "Codex", "codex", "remote-control");
    let claude = provider_state("claude-code", "Claude Code", "claude", "--remote-control");
    Ok(AgentIntegrationReport {
        schema_version: 1,
        generated_at: generated_at(),
        embedded_client_ready: true,
        first_party_gateway_ready: false,
        telegram_ready: false,
        providers: vec![codex, claude],
        transports: transport_states()?,
    })
}

fn operation_name(operation: AgentOperation) -> &'static str {
    match operation {
        AgentOperation::Start => "codex-remote-start",
        AgentOperation::Pair => "codex-remote-pair",
        AgentOperation::Stop => "codex-remote-stop",
    }
}

fn operation_args(operation: AgentOperation) -> &'static [&'static str] {
    match operation {
        AgentOperation::Start => &["remote-control", "--json", "start"],
        AgentOperation::Pair => &["remote-control", "--json", "pair"],
        AgentOperation::Stop => &["remote-control", "--json", "stop"],
    }
}

fn parse_pairing(stdout: &[u8]) -> Result<PairingGrant, String> {
    let value: Value = serde_json::from_slice(stdout)
        .map_err(|_| "Codex returned an invalid pairing response".to_owned())?;
    let code = value
        .get("manualPairingCode")
        .and_then(Value::as_str)
        .filter(|code| {
            (4..=128).contains(&code.len())
                && code
                    .bytes()
                    .all(|byte| byte.is_ascii_alphanumeric() || matches!(byte, b'-' | b'_'))
        })
        .ok_or_else(|| "Codex pairing response did not contain a safe manual code".to_owned())?;
    let expires_at = value
        .get("expiresAt")
        .and_then(Value::as_str)
        .filter(|expiry| expiry.len() <= 64 && !expiry.chars().any(char::is_control))
        .ok_or_else(|| "Codex pairing response did not contain an expiry".to_owned())?;
    Ok(PairingGrant {
        code: code.to_owned(),
        expires_at: expires_at.to_owned(),
    })
}

pub fn execute_operation(
    operation: AgentOperation,
    confirmed: bool,
) -> Result<AgentOperationResult, String> {
    if matches!(operation, AgentOperation::Start | AgentOperation::Pair) && !confirmed {
        return Err("Agent remote access requires explicit local confirmation".to_owned());
    }
    let codex = command_path("codex").ok_or_else(|| "Codex CLI is not installed".to_owned())?;
    if !output_contains(run_limited(&codex, &["remote-control", "--help"]), "pair") {
        return Err("Installed Codex CLI does not support Remote Control".to_owned());
    }
    let output = run_limited(&codex, operation_args(operation))
        .map_err(|error| format!("Unable to run the Codex adapter: {error}"))?;
    if output.timed_out {
        return Err("Codex Remote Control exceeded its safety deadline".to_owned());
    }
    if !output.status.success() {
        return Err(format!(
            "Codex Remote Control failed with exit code {}",
            output.status.code().unwrap_or(-1)
        ));
    }
    let pairing = if operation == AgentOperation::Pair {
        Some(parse_pairing(&output.stdout)?)
    } else {
        None
    };
    let (message_key, running) = match operation {
        AgentOperation::Start => ("agent.codex.started", Some(true)),
        AgentOperation::Pair => ("agent.codex.paired", Some(true)),
        AgentOperation::Stop => ("agent.codex.stopped", Some(false)),
    };
    Ok(AgentOperationResult {
        success: true,
        operation: operation_name(operation),
        message_key,
        running,
        pairing,
    })
}

#[tauri::command]
pub async fn get_agent_integrations() -> Result<AgentIntegrationReport, String> {
    tauri::async_runtime::spawn_blocking(integration_report)
        .await
        .map_err(|error| error.to_string())?
}

#[tauri::command]
pub async fn run_agent_operation(
    operation: AgentOperation,
    confirmed: bool,
) -> Result<AgentOperationResult, String> {
    tauri::async_runtime::spawn_blocking(move || execute_operation(operation, confirmed))
        .await
        .map_err(|error| error.to_string())?
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn operations_map_only_to_fixed_codex_arguments() {
        assert_eq!(
            operation_args(AgentOperation::Start),
            ["remote-control", "--json", "start"]
        );
        assert_eq!(
            operation_args(AgentOperation::Pair),
            ["remote-control", "--json", "pair"]
        );
        assert_eq!(
            operation_args(AgentOperation::Stop),
            ["remote-control", "--json", "stop"]
        );
    }

    #[test]
    fn pairing_response_exposes_only_manual_code_and_expiry() {
        let grant = parse_pairing(
            br#"{"pairingCode":"opaque-secret","manualPairingCode":"ABCD-1234","environmentId":"private","expiresAt":"2026-08-02T12:00:00Z"}"#,
        )
        .expect("valid pairing response");
        assert_eq!(grant.code, "ABCD-1234");
        assert_eq!(grant.expires_at, "2026-08-02T12:00:00Z");
    }

    #[test]
    fn pairing_response_rejects_opaque_secret_without_manual_code() {
        assert!(
            parse_pairing(br#"{"pairingCode":"opaque-secret","expiresAt":"2026-08-02T12:00:00Z"}"#)
                .is_err()
        );
    }

    #[test]
    fn pairing_response_rejects_control_characters() {
        assert!(
            parse_pairing(
                b"{\"manualPairingCode\":\"BAD\\nCODE\",\"expiresAt\":\"2026-08-02T12:00:00Z\"}"
            )
            .is_err()
        );
    }

    #[test]
    fn embedded_transport_registry_is_parseable() {
        let transports = transport_states().expect("embedded registry");
        assert_eq!(transports.len(), 7);
        assert!(transports.iter().all(|transport| !transport.runtime_ready));
    }
}
