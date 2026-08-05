// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: AGPL-3.0-or-later

use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
#[cfg(target_os = "linux")]
use std::io::Write;
#[cfg(target_os = "linux")]
use std::os::unix::net::UnixStream;
use std::{
    collections::HashSet,
    env, fs,
    io::{self, Read},
    net::IpAddr,
    path::{Path, PathBuf},
    process::{Command, Output, Stdio},
    sync::atomic::{AtomicU64, Ordering},
    thread,
    time::{SystemTime, UNIX_EPOCH},
};
use tauri::{AppHandle, Manager};

const SYSTEM_CLI_PATH: &str = "/usr/bin/mazzy-vpn";
const LOCAL_CLI_PATH: &str = "/usr/local/bin/mazzy-vpn";
pub(crate) const TIMEOUT_PATH: &str = "/usr/bin/timeout";
pub(crate) const PKEXEC_PATH: &str = "/usr/bin/pkexec";
const PROFILES_FILE: &str = "/run/mazzy-vpn/profiles.json";
const API_SOCKET: &str = "/run/mazzy-vpn/api-v1.sock";
const MAX_OUTPUT_BYTES: usize = 512 * 1024;
const MAX_OUTPUT_STREAM_BYTES: usize = MAX_OUTPUT_BYTES / 2;
const TRUNCATED_OUTPUT_MARKER: &[u8] = b"\n[output truncated]\n";
#[cfg(target_os = "linux")]
const MAX_API_RESPONSE_BYTES: usize = 64 * 1024;
#[cfg(target_os = "linux")]
const LOCAL_API_COMPLETION_GRACE_MS: u64 = 60_000;
#[cfg(target_os = "linux")]
const READ_ONLY_API_RETRY_DELAYS_MS: &[u64] = &[0, 250, 500, 1_000, 2_000, 4_000];
#[cfg(target_os = "linux")]
const API_STARTUP_RETRY_DELAYS_MS: &[u64] = &[0, 250, 500, 1_000, 2_000, 4_000, 8_000];
static API_SEQUENCE: AtomicU64 = AtomicU64::new(1);

#[cfg(target_os = "linux")]
#[derive(Debug)]
enum LocalApiError {
    Unavailable,
    Indeterminate(String),
}

#[derive(Clone, Debug, Deserialize)]
#[serde(tag = "kind", rename_all = "kebab-case")]
pub enum OperationRequest {
    Quick,
    Reconnect,
    Disconnect,
    Refresh,
    Connect {
        protocol: String,
        profile: String,
    },
    Validate {
        protocol: String,
    },
    Probe {
        protocol: String,
        timeout: u16,
    },
    Test {
        protocol: String,
        profile: String,
        timeout: u16,
    },
    TestAll {
        protocol: String,
        timeout: u16,
    },
    Emergency {
        protocol: Option<String>,
        timeout: u16,
    },
    SelfTest {
        live: bool,
        timeout: u16,
    },
    Doctor {
        fix: bool,
    },
    Diagnose,
    Autostart {
        enabled: bool,
    },
    Monitor {
        enabled: bool,
    },
    ImportFiles {
        paths: Vec<String>,
        force: bool,
    },
    ImportFolder {
        path: String,
        dry_run: bool,
        force: bool,
    },
    RemoveProfile {
        protocol: String,
        profile: String,
    },
    Logs {
        lines: u16,
    },
    Language {
        language: String,
    },
    Bootstrap,
}

#[derive(Clone, Debug, Serialize)]
pub struct OperationResult {
    pub success: bool,
    pub action: String,
    pub output: String,
    pub code: Option<i32>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct LocationProbeSummary {
    total: u64,
    reachable: u64,
    unknown: u64,
    unreachable: u64,
    invalid: u64,
    active: u64,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct LocationProbeEntry {
    profile_id: String,
    display_name: String,
    protocol: String,
    selected: bool,
    active: bool,
    transport: String,
    reachability: String,
    latency_ms: Option<u64>,
    latency_source: String,
    message_key: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct LocationProbeCollection {
    schema_version: u8,
    checked_at: String,
    scope: String,
    timeout_seconds: u16,
    concurrency: u8,
    duration_ms: u64,
    summary: LocationProbeSummary,
    results: Vec<LocationProbeEntry>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct EgressTunnel {
    active: bool,
    protocol: Option<String>,
    profile: Option<String>,
    interface: Option<String>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct EgressIpv4 {
    interface_ip: Option<String>,
    default_ip: Option<String>,
    same_egress: bool,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct EgressIpv6 {
    interface_ip: Option<String>,
    default_ip: Option<String>,
    potential_leak: bool,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct EgressGeoProvider {
    provider: String,
    ip: String,
    country_code: String,
    country: String,
    region: String,
    city: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct EgressGeo {
    expected_country_code: Option<String>,
    observed_country_code: Option<String>,
    country_match: String,
    providers_agree: Option<bool>,
    providers: Vec<EgressGeoProvider>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct EgressDns {
    state: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct EgressSpeed {
    requested: bool,
    measured: bool,
    mbps: Option<f64>,
    connect_ms: Option<u64>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct EgressVerification {
    schema_version: u8,
    checked_at: String,
    verdict: String,
    message_key: String,
    tunnel: EgressTunnel,
    ipv4: EgressIpv4,
    ipv6: EgressIpv6,
    geo: EgressGeo,
    dns: EgressDns,
    speed: EgressSpeed,
    findings: Vec<String>,
}

#[derive(Clone, Debug, Serialize)]
pub struct DependencyState {
    pub id: &'static str,
    pub label: &'static str,
    pub installed: bool,
    pub required_for: &'static str,
}

#[derive(Clone, Debug, Serialize)]
pub struct InstallationReport {
    pub engine_installed: bool,
    pub package_managed: bool,
    pub installed_version: Option<String>,
    pub bundled_version: Option<String>,
    pub bundled_installer: bool,
    pub needs_install: bool,
    pub service_installed: bool,
    pub monitor_installed: bool,
    pub api_installed: bool,
    pub api_socket_available: bool,
    pub dependencies_ready: bool,
    pub missing_dependencies: usize,
    pub dependencies: Vec<DependencyState>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct ProfileCacheEntry {
    #[serde(default)]
    profile_id: String,
    protocol: String,
    protocol_name: String,
    file_name: String,
    name: String,
    location: String,
    #[serde(default)]
    country_code: Option<String>,
    selected: bool,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct ProfileCache {
    schema_version: u8,
    generated_at: u64,
    profiles: Vec<ProfileCacheEntry>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct DashboardFallback {
    active: bool,
    name: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct DashboardProfileCounts {
    amneziawg: u64,
    wireguard: u64,
    openvpn: u64,
    l2tp: u64,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct DashboardStatus {
    schema_version: u8,
    generated_at: u64,
    product: String,
    version: String,
    language: String,
    selected: bool,
    service_state: String,
    desired: String,
    mode: String,
    tunnel_active: bool,
    internet: String,
    healthy: bool,
    protocol: String,
    protocol_name: String,
    profile: String,
    #[serde(default)]
    profile_id: Option<String>,
    #[serde(default)]
    profile_file_name: Option<String>,
    location: String,
    interface: String,
    handshake_age: Option<u64>,
    public_ip: String,
    autostart: bool,
    health_monitor: bool,
    fallback: DashboardFallback,
    health_failures: u64,
    profiles: DashboardProfileCounts,
}

fn select_cli_path(system_installed: bool, local_installed: bool) -> Option<&'static Path> {
    if system_installed {
        Some(Path::new(SYSTEM_CLI_PATH))
    } else if local_installed {
        Some(Path::new(LOCAL_CLI_PATH))
    } else {
        None
    }
}

pub(crate) fn installed_cli_path() -> Option<&'static Path> {
    select_cli_path(
        Path::new(SYSTEM_CLI_PATH).is_file(),
        Path::new(LOCAL_CLI_PATH).is_file(),
    )
}

fn systemd_unit_installed(name: &str) -> bool {
    ["/usr/lib/systemd/system", "/etc/systemd/system"]
        .iter()
        .any(|directory| Path::new(directory).join(name).is_file())
}

fn protocol(value: &str, allow_all: bool) -> Result<String, String> {
    let value = value.to_ascii_lowercase();
    let valid = matches!(
        value.as_str(),
        "amneziawg" | "wireguard" | "openvpn" | "l2tp"
    ) || (allow_all && value == "all");
    valid
        .then_some(value)
        .ok_or_else(|| "Unsupported VPN protocol".to_owned())
}

fn timeout(value: u16, minimum: u16, maximum: u16) -> Result<String, String> {
    (minimum..=maximum)
        .contains(&value)
        .then(|| value.to_string())
        .ok_or_else(|| format!("Timeout must be between {minimum} and {maximum} seconds"))
}

fn profile_name(value: &str) -> Result<String, String> {
    let valid = !value.is_empty()
        && value.len() <= 255
        && !value.contains('/')
        && !value.contains('\\')
        && !value.chars().any(unsafe_frontend_character);
    valid
        .then(|| value.to_owned())
        .ok_or_else(|| "Unsafe or empty profile name".to_owned())
}

fn import_path(value: &str, directory: bool) -> Result<String, String> {
    let path = fs::canonicalize(value).map_err(|error| format!("{value}: {error}"))?;
    let metadata = fs::metadata(&path).map_err(|error| error.to_string())?;
    let expected = if directory {
        metadata.is_dir()
    } else {
        metadata.is_file()
    };
    if !expected {
        return Err(if directory {
            "Selected path is not a directory".to_owned()
        } else {
            "Selected path is not a regular file".to_owned()
        });
    }
    path.into_os_string()
        .into_string()
        .map_err(|_| "Selected path is not valid UTF-8".to_owned())
}

fn build_cli_args(request: &OperationRequest) -> Result<(String, Vec<String>), String> {
    let mut args = Vec::new();
    let action = match request {
        OperationRequest::Quick => {
            args.push("quick".into());
            "quick"
        }
        OperationRequest::Reconnect => {
            args.push("reconnect".into());
            "reconnect"
        }
        OperationRequest::Disconnect => {
            args.push("disconnect".into());
            "disconnect"
        }
        OperationRequest::Refresh => {
            args.push("_refresh-dashboard-cache".into());
            "refresh"
        }
        OperationRequest::Connect {
            protocol: selected_protocol,
            profile,
        } => {
            args.extend([
                "connect".into(),
                protocol(selected_protocol, false)?,
                profile_name(profile)?,
            ]);
            "connect"
        }
        OperationRequest::Validate {
            protocol: selected_protocol,
        } => {
            args.extend(["validate".into(), protocol(selected_protocol, true)?]);
            "validate"
        }
        OperationRequest::Probe {
            protocol: selected_protocol,
            timeout: seconds,
        } => {
            args.extend([
                "probe".into(),
                protocol(selected_protocol, true)?,
                "--timeout".into(),
                timeout(*seconds, 1, 30)?,
            ]);
            "probe"
        }
        OperationRequest::Test {
            protocol: selected_protocol,
            profile,
            timeout: seconds,
        } => {
            args.extend([
                "test".into(),
                protocol(selected_protocol, false)?,
                profile_name(profile)?,
                "--timeout".into(),
                timeout(*seconds, 2, 600)?,
            ]);
            "test"
        }
        OperationRequest::TestAll {
            protocol: selected_protocol,
            timeout: seconds,
        } => {
            args.extend([
                "test-all".into(),
                protocol(selected_protocol, true)?,
                "--timeout".into(),
                timeout(*seconds, 2, 600)?,
            ]);
            "test-all"
        }
        OperationRequest::Emergency {
            protocol: selected_protocol,
            timeout: seconds,
        } => {
            args.extend([
                "emergency".into(),
                "--timeout".into(),
                timeout(*seconds, 2, 600)?,
            ]);
            if let Some(selected_protocol) = selected_protocol {
                args.extend(["--protocol".into(), protocol(selected_protocol, false)?]);
            }
            "emergency"
        }
        OperationRequest::SelfTest {
            live,
            timeout: seconds,
        } => {
            args.extend([
                "self-test".into(),
                if *live { "--live" } else { "--offline" }.into(),
                "--timeout".into(),
                timeout(*seconds, 1, 30)?,
            ]);
            "self-test"
        }
        OperationRequest::Doctor { fix } => {
            args.push("doctor".into());
            if *fix {
                args.push("--fix".into());
            }
            "doctor"
        }
        OperationRequest::Diagnose => {
            args.push("diagnose".into());
            "diagnose"
        }
        OperationRequest::Autostart { enabled } => {
            args.extend([
                "autostart".into(),
                if *enabled { "on" } else { "off" }.into(),
            ]);
            "autostart"
        }
        OperationRequest::Monitor { enabled } => {
            args.extend(["monitor".into(), if *enabled { "on" } else { "off" }.into()]);
            "monitor"
        }
        OperationRequest::ImportFiles { paths, force } => {
            if paths.is_empty() || paths.len() > 128 {
                return Err("Select between 1 and 128 profile files".into());
            }
            args.push("import-files".into());
            for path in paths {
                args.push(import_path(path, false)?);
            }
            if *force {
                args.push("--force".into());
            }
            "import-files"
        }
        OperationRequest::ImportFolder {
            path,
            dry_run,
            force,
        } => {
            args.extend(["import-dir".into(), import_path(path, true)?]);
            if *dry_run {
                args.push("--dry-run".into());
            }
            if *force {
                args.push("--force".into());
            }
            "import-folder"
        }
        OperationRequest::RemoveProfile {
            protocol: selected_protocol,
            profile,
        } => {
            args.extend([
                "remove".into(),
                protocol(selected_protocol, false)?,
                profile_name(profile)?,
            ]);
            "remove-profile"
        }
        OperationRequest::Logs { lines } => {
            args.extend(["logs".into(), "--lines".into(), timeout(*lines, 20, 1000)?]);
            "logs"
        }
        OperationRequest::Language { language } => {
            let language = language.to_ascii_lowercase();
            if !matches!(language.as_str(), "ru" | "en" | "de" | "zh" | "ja" | "ko") {
                return Err("Unsupported interface language".into());
            }
            args.extend(["language".into(), language]);
            "language"
        }
        OperationRequest::Bootstrap => {
            return Err("Bootstrap uses the embedded installer".into());
        }
    };
    Ok((action.into(), args))
}

fn normalize_privileged_error(cleaned: String) -> String {
    let lower = cleaned.to_ascii_lowercase();
    if lower.contains("error creating textual authentication agent") && lower.contains("/dev/tty") {
        return "PolicyKit authorization is unavailable for this desktop operation. Start Mazzy VPN Desktop from a graphical session with a PolicyKit authentication agent, or wait for the local Mazzy VPN API to become ready.".into();
    }
    cleaned
}

fn package_managed_cli(cli_path: &Path) -> bool {
    cli_path == Path::new(SYSTEM_CLI_PATH)
}

fn engine_not_ready_error() -> String {
    "Mazzy VPN engine is installed, but its local API is still starting. Wait a few seconds and retry; no privileged fallback was started from the desktop.".into()
}

pub(crate) fn clean_output(output: &Output) -> String {
    let mut bytes = output.stdout.clone();
    if !output.stderr.is_empty() {
        if !bytes.is_empty() {
            bytes.push(b'\n');
        }
        bytes.extend_from_slice(&output.stderr);
    }
    bytes.truncate(MAX_OUTPUT_BYTES);
    let raw = String::from_utf8_lossy(&bytes);
    let mut cleaned = String::with_capacity(raw.len());
    let mut chars = raw.chars().peekable();
    while let Some(character) = chars.next() {
        if character == '\u{1b}' {
            if chars.peek() == Some(&'[') {
                let _ = chars.next();
                for next in chars.by_ref() {
                    if next.is_ascii_alphabetic() {
                        break;
                    }
                }
            }
            continue;
        }
        if character == '\n' || character == '\t' || !character.is_control() {
            cleaned.push(character);
        }
    }
    normalize_privileged_error(cleaned.trim().to_owned())
}

fn drain_bounded<R: Read>(mut reader: R, limit: usize) -> io::Result<Vec<u8>> {
    let mut retained = Vec::with_capacity(limit.min(8192));
    let mut buffer = [0_u8; 8192];
    let mut truncated = false;

    loop {
        let read = reader.read(&mut buffer)?;
        if read == 0 {
            break;
        }
        let remaining = limit.saturating_sub(retained.len());
        let keep = remaining.min(read);
        retained.extend_from_slice(&buffer[..keep]);
        truncated |= keep < read;
    }

    if truncated {
        let content_limit = limit.saturating_sub(TRUNCATED_OUTPUT_MARKER.len());
        retained.truncate(content_limit);
        let marker_bytes = limit.saturating_sub(retained.len());
        retained.extend_from_slice(
            &TRUNCATED_OUTPUT_MARKER[..marker_bytes.min(TRUNCATED_OUTPUT_MARKER.len())],
        );
    }
    Ok(retained)
}

fn join_capture(
    handle: thread::JoinHandle<io::Result<Vec<u8>>>,
    stream: &str,
) -> io::Result<Vec<u8>> {
    handle.join().map_err(|_| {
        io::Error::other(format!(
            "Desktop output capture thread panicked while reading {stream}"
        ))
    })?
}

pub(crate) fn bounded_output(command: &mut Command) -> io::Result<Output> {
    command.stdout(Stdio::piped()).stderr(Stdio::piped());
    let mut child = command.spawn()?;
    let stdout = child
        .stdout
        .take()
        .ok_or_else(|| io::Error::other("Unable to capture child stdout"))?;
    let stderr = child
        .stderr
        .take()
        .ok_or_else(|| io::Error::other("Unable to capture child stderr"))?;
    let stdout_handle = thread::spawn(move || drain_bounded(stdout, MAX_OUTPUT_STREAM_BYTES));
    let stderr_handle = thread::spawn(move || drain_bounded(stderr, MAX_OUTPUT_STREAM_BYTES));

    let status_result = child.wait();
    let stdout = join_capture(stdout_handle, "stdout")?;
    let stderr = join_capture(stderr_handle, "stderr")?;
    let status = status_result?;
    Ok(Output {
        status,
        stdout,
        stderr,
    })
}

fn command_result(action: String, result: Result<Output, io::Error>) -> OperationResult {
    match result {
        Ok(output) => OperationResult {
            success: output.status.success(),
            action,
            output: clean_output(&output),
            code: output.status.code(),
        },
        Err(error) => OperationResult {
            success: false,
            action,
            output: error.to_string(),
            code: None,
        },
    }
}

fn operation_deadline_seconds(request: &OperationRequest) -> u64 {
    match request {
        OperationRequest::Quick
        | OperationRequest::Reconnect
        | OperationRequest::Disconnect
        | OperationRequest::Connect { .. }
        | OperationRequest::Autostart { .. }
        | OperationRequest::Monitor { .. }
        | OperationRequest::RemoveProfile { .. } => 300,
        OperationRequest::Refresh
        | OperationRequest::Logs { .. }
        | OperationRequest::Language { .. } => 120,
        OperationRequest::Validate { .. } | OperationRequest::Diagnose => 600,
        OperationRequest::Probe { .. } => 900,
        OperationRequest::Test { timeout, .. } => u64::from(*timeout).saturating_add(180),
        OperationRequest::TestAll { .. } => 7_200,
        OperationRequest::Emergency { .. } => 1_800,
        OperationRequest::SelfTest { live, .. } => {
            if *live {
                1_800
            } else {
                600
            }
        }
        OperationRequest::Doctor { fix } => {
            if *fix {
                1_800
            } else {
                600
            }
        }
        OperationRequest::ImportFiles { .. } | OperationRequest::ImportFolder { .. } => 1_800,
        OperationRequest::Bootstrap => 1_800,
    }
}

fn operation_changes_system(request: &OperationRequest) -> bool {
    !matches!(
        request,
        OperationRequest::Refresh
            | OperationRequest::Validate { .. }
            | OperationRequest::Probe { .. }
            | OperationRequest::SelfTest { live: false, .. }
            | OperationRequest::Doctor { fix: false }
            | OperationRequest::Diagnose
            | OperationRequest::Logs { .. }
    )
}

fn timed_privileged_output(
    program: &Path,
    args: &[String],
    deadline_seconds: u64,
) -> io::Result<Output> {
    bounded_output(
        Command::new(TIMEOUT_PATH)
            .arg("--kill-after=30s")
            .arg(format!("{deadline_seconds}s"))
            .arg(PKEXEC_PATH)
            .arg(program)
            .args(args),
    )
}

fn timed_operation_result(
    action: String,
    result: Result<Output, io::Error>,
    changes_system: bool,
) -> OperationResult {
    let mut operation = command_result(action, result);
    if matches!(operation.code, Some(124 | 137)) {
        operation.output = if changes_system {
            "The operation exceeded its safety deadline. Network state may have changed; \
             refresh status and run Doctor before retrying."
                .into()
        } else {
            "The read-only operation exceeded its safety deadline and was stopped.".into()
        };
    }
    operation
}

fn engine_root(app: &AppHandle) -> PathBuf {
    let bundled = app
        .path()
        .resource_dir()
        .unwrap_or_else(|_| PathBuf::new())
        .join("engine");
    if bundled.join("install.sh").is_file() {
        bundled
    } else {
        Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../..")
            .to_path_buf()
    }
}

fn api_identifier(kind: &str) -> String {
    let timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis();
    let sequence = API_SEQUENCE.fetch_add(1, Ordering::Relaxed);
    format!("{kind}-{timestamp}-{}-{sequence}", std::process::id())
}

fn valid_api_identifier(identifier: &str) -> bool {
    (8..=128).contains(&identifier.len())
        && identifier.chars().enumerate().all(|(index, character)| {
            character.is_ascii_alphanumeric()
                || (index > 0 && matches!(character, '.' | '_' | ':' | '-'))
        })
}

fn profile_id(catalog: &Value, protocol: &str, file_name: &str) -> Result<String, String> {
    catalog
        .get("profiles")
        .and_then(Value::as_array)
        .and_then(|profiles| {
            profiles.iter().find(|profile| {
                profile.get("protocol").and_then(Value::as_str) == Some(protocol)
                    && profile.get("file_name").and_then(Value::as_str) == Some(file_name)
            })
        })
        .and_then(|profile| profile.get("profile_id"))
        .and_then(Value::as_str)
        .filter(|identifier| valid_api_identifier(identifier))
        .map(str::to_owned)
        .ok_or_else(|| "The profile cache does not contain a safe API profile ID".to_owned())
}

fn local_api_request(
    request: &OperationRequest,
    catalog: Option<&Value>,
) -> Result<Option<(String, Value)>, String> {
    let (action, operation, deadline_ms, payload) = match request {
        OperationRequest::Connect {
            protocol: selected_protocol,
            profile,
        } => {
            let selected_protocol = protocol(selected_protocol, false)?;
            let profile = profile_name(profile)?;
            let catalog = catalog.ok_or_else(|| "The profile cache is not ready".to_owned())?;
            let profile_id = profile_id(catalog, &selected_protocol, &profile)?;
            (
                "connect",
                "lifecycle.connect",
                60_000,
                json!({"profile_id": profile_id}),
            )
        }
        OperationRequest::Reconnect => ("reconnect", "lifecycle.reconnect", 30_000, json!({})),
        OperationRequest::Disconnect => ("disconnect", "lifecycle.disconnect", 30_000, json!({})),
        _ => return Ok(None),
    };
    Ok(Some((
        action.into(),
        json!({
            "api_version": "1.0",
            "request_id": api_identifier("request"),
            "operation": operation,
            "action_id": api_identifier("action"),
            "authorization": "system-mutate",
            "deadline_ms": deadline_ms,
            "payload": payload
        }),
    )))
}

#[cfg(any(target_os = "linux", test))]
fn probe_deadline_ms(profile_count: usize, timeout_seconds: u16, concurrency: u8) -> u64 {
    let workers = usize::from(concurrency.max(1));
    let rounds = profile_count.max(1).div_ceil(workers);
    let per_round_ms = u64::from(timeout_seconds.saturating_add(1)) * 3_000;
    u64::try_from(rounds)
        .unwrap_or(u64::MAX)
        .saturating_mul(per_round_ms)
        .saturating_add(5_000)
        .clamp(5_000, 900_000)
}

#[cfg(any(target_os = "linux", test))]
fn cached_profile_count(protocol_filter: &str) -> usize {
    fs::read_to_string(PROFILES_FILE)
        .ok()
        .and_then(|contents| serde_json::from_str::<Value>(&contents).ok())
        .and_then(|catalog| catalog.get("profiles").and_then(Value::as_array).cloned())
        .map(|profiles| {
            profiles
                .iter()
                .filter(|profile| {
                    protocol_filter == "all"
                        || profile.get("protocol").and_then(Value::as_str) == Some(protocol_filter)
                })
                .count()
        })
        .unwrap_or(1)
}

#[cfg(any(target_os = "linux", test))]
fn probe_api_request(
    selected_protocol: &str,
    timeout_seconds: u16,
    concurrency: u8,
    profile_count: usize,
) -> Value {
    let mut payload = json!({
        "timeout_seconds": timeout_seconds,
        "concurrency": concurrency
    });
    if selected_protocol != "all" {
        payload["protocol"] = Value::String(selected_protocol.to_owned());
    }
    json!({
        "api_version": "1.0",
        "request_id": api_identifier("request"),
        "operation": "tests.probe",
        "deadline_ms": probe_deadline_ms(profile_count, timeout_seconds, concurrency),
        "payload": payload
    })
}

fn sanitize_probe_collection(result: Value) -> Result<Value, String> {
    let collection: LocationProbeCollection = serde_json::from_value(result)
        .map_err(|_| "The endpoint probe returned a malformed result".to_owned())?;
    let protocol_valid =
        |value: &str| matches!(value, "amneziawg" | "wireguard" | "openvpn" | "l2tp");
    if collection.schema_version != 1
        || !(1..=30).contains(&collection.timeout_seconds)
        || !(1..=8).contains(&collection.concurrency)
        || !(collection.scope == "all" || protocol_valid(&collection.scope))
        || collection.checked_at.is_empty()
        || collection.checked_at.len() > 64
        || collection.checked_at.chars().any(char::is_control)
        || collection.results.len() > 1_024
    {
        return Err("The endpoint probe returned an invalid result".to_owned());
    }

    let total = u64::try_from(collection.results.len())
        .map_err(|_| "The endpoint probe result is too large".to_owned())?;
    let reachable = collection
        .results
        .iter()
        .filter(|entry| entry.reachability == "reachable")
        .count() as u64;
    let unknown = collection
        .results
        .iter()
        .filter(|entry| entry.reachability == "unknown")
        .count() as u64;
    let unreachable = collection
        .results
        .iter()
        .filter(|entry| entry.reachability == "unreachable")
        .count() as u64;
    let invalid = collection
        .results
        .iter()
        .filter(|entry| entry.reachability == "invalid")
        .count() as u64;
    let active = collection
        .results
        .iter()
        .filter(|entry| entry.active)
        .count() as u64;
    let selected = collection
        .results
        .iter()
        .filter(|entry| entry.selected)
        .count();
    if collection.summary.total != total
        || collection.summary.reachable != reachable
        || collection.summary.unknown != unknown
        || collection.summary.unreachable != unreachable
        || collection.summary.invalid != invalid
        || collection.summary.active != active
        || selected > 1
        || active > 1
    {
        return Err("The endpoint probe summary is inconsistent".to_owned());
    }
    for entry in &collection.results {
        let latency_valid = match entry.latency_ms {
            Some(_) => matches!(entry.latency_source.as_str(), "icmp" | "tcp"),
            None => entry.latency_source == "none",
        };
        if !valid_api_identifier(&entry.profile_id)
            || !valid_api_identifier(&entry.message_key)
            || entry.display_name.is_empty()
            || entry.display_name.chars().count() > 255
            || entry.display_name.chars().any(char::is_control)
            || !protocol_valid(&entry.protocol)
            || !matches!(entry.transport.as_str(), "tcp" | "udp")
            || !matches!(
                entry.reachability.as_str(),
                "reachable" | "unknown" | "unreachable" | "invalid"
            )
            || !latency_valid
            || (entry.active && !entry.selected)
        {
            return Err("The endpoint probe contains an invalid profile result".to_owned());
        }
    }
    serde_json::to_value(collection)
        .map_err(|_| "The endpoint probe result could not be sanitized".to_owned())
}

#[cfg(any(target_os = "linux", test))]
fn probe_result_from_response(response: Value) -> Result<Value, String> {
    if response.get("status").and_then(Value::as_str) != Some("ok") {
        let code = response
            .pointer("/error/code")
            .and_then(Value::as_str)
            .unwrap_or("internal-error");
        let message = response
            .pointer("/error/message_key")
            .and_then(Value::as_str)
            .unwrap_or("api.response.malformed");
        return Err(format!("{code}: {message}"));
    }
    let result = response
        .get("result")
        .cloned()
        .ok_or_else(|| "The endpoint probe returned a malformed result".to_owned())?;
    sanitize_probe_collection(result)
}

fn safe_frontend_text(value: &str, maximum: usize, allow_empty: bool) -> bool {
    (allow_empty || !value.is_empty())
        && value.chars().count() <= maximum
        && !value.chars().any(unsafe_frontend_character)
}

fn unsafe_frontend_character(value: char) -> bool {
    value.is_control()
        || matches!(
            value,
            '\u{00ad}'
                | '\u{061c}'
                | '\u{200b}'..='\u{200f}'
                | '\u{2028}'..='\u{202e}'
                | '\u{2060}'..='\u{2064}'
                | '\u{2066}'..='\u{2069}'
                | '\u{feff}'
        )
}

fn valid_country_code(value: &str) -> bool {
    value.len() == 2 && value.bytes().all(|byte| byte.is_ascii_uppercase())
}

fn compatibility_profile_id(protocol: &str, file_name: &str) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut hasher = Sha256::new();
    hasher.update(protocol.as_bytes());
    hasher.update([0]);
    hasher.update(file_name.as_bytes());
    let digest = hasher.finalize();
    let mut identifier = String::with_capacity(40);
    identifier.push_str("profile-");
    for byte in digest.iter().take(16) {
        identifier.push(HEX[usize::from(byte >> 4)] as char);
        identifier.push(HEX[usize::from(byte & 0x0f)] as char);
    }
    identifier
}

fn sanitize_profile_cache(contents: &str) -> Result<Value, String> {
    let mut cache: ProfileCache =
        serde_json::from_str(contents).map_err(|_| "The profile cache is malformed".to_owned())?;
    if cache.schema_version != 1 || cache.profiles.len() > 1_024 {
        return Err("The profile cache has an unsupported shape".to_owned());
    }
    let selected_count = cache
        .profiles
        .iter()
        .filter(|profile| profile.selected)
        .count();
    if selected_count > 1 {
        return Err("The profile cache selects multiple profiles".to_owned());
    }
    let mut profile_ids = HashSet::with_capacity(cache.profiles.len());
    let mut profile_files = HashSet::with_capacity(cache.profiles.len());
    for profile in &mut cache.profiles {
        if profile.profile_id.is_empty() {
            profile.profile_id = compatibility_profile_id(&profile.protocol, &profile.file_name);
        }
        let expected_protocol_name = match profile.protocol.as_str() {
            "amneziawg" => "AmneziaWG",
            "wireguard" => "WireGuard",
            "openvpn" => "OpenVPN",
            "l2tp" => "L2TP/IPsec",
            _ => return Err("The profile cache contains an unsupported protocol".to_owned()),
        };
        let extension_valid = match profile.protocol.as_str() {
            "amneziawg" | "wireguard" => profile.file_name.ends_with(".conf"),
            "openvpn" => {
                profile.file_name.ends_with(".ovpn") || profile.file_name.ends_with(".conf")
            }
            "l2tp" => profile.file_name.ends_with(".nmconnection"),
            _ => false,
        };
        if !profile_ids.insert(profile.profile_id.as_str())
            || !profile_files.insert((profile.protocol.as_str(), profile.file_name.as_str()))
            || !valid_api_identifier(&profile.profile_id)
            || profile.protocol_name != expected_protocol_name
            || profile_name(&profile.file_name).is_err()
            || !extension_valid
            || !safe_frontend_text(&profile.name, 255, false)
            || !safe_frontend_text(&profile.location, 255, false)
            || profile
                .country_code
                .as_deref()
                .is_some_and(|value| !valid_country_code(value))
        {
            return Err("The profile cache contains unsafe profile metadata".to_owned());
        }
    }
    serde_json::to_value(cache).map_err(|_| "The profile cache could not be sanitized".to_owned())
}

pub(crate) fn sanitize_status_cache(contents: &str) -> Result<Value, String> {
    let status: DashboardStatus =
        serde_json::from_str(contents).map_err(|_| "The status cache is malformed".to_owned())?;
    let protocol_name = match status.protocol.as_str() {
        "" => "",
        "amneziawg" => "AmneziaWG",
        "wireguard" => "WireGuard",
        "openvpn" => "OpenVPN",
        "l2tp" => "L2TP/IPsec",
        _ => return Err("The status cache contains an unsupported protocol".to_owned()),
    };
    let selected_identity_is_complete =
        status.profile_id.is_some() == status.profile_file_name.is_some();
    let selected_metadata_is_safe = !status.protocol.is_empty()
        && safe_frontend_text(&status.profile, 255, false)
        && safe_frontend_text(&status.location, 255, false)
        && status
            .profile_id
            .as_deref()
            .is_none_or(valid_api_identifier)
        && status
            .profile_file_name
            .as_deref()
            .is_none_or(|name| profile_name(name).is_ok());
    let unselected_metadata_is_empty = status.protocol.is_empty()
        && status.protocol_name.is_empty()
        && status.profile.is_empty()
        && status.profile_id.is_none()
        && status.profile_file_name.is_none()
        && status.location.is_empty()
        && status.interface.is_empty()
        && status.handshake_age.is_none()
        && status.public_ip.is_empty();
    let public_ip_is_safe = status.public_ip.is_empty()
        || status
            .public_ip
            .parse::<IpAddr>()
            .is_ok_and(|address| !address.is_unspecified());
    let interface_is_safe = status.interface.is_empty()
        || (status.interface.len() <= 32
            && status
                .interface
                .bytes()
                .all(|byte| byte.is_ascii_alphanumeric() || b"_.:-".contains(&byte)));
    let profile_counts_are_bounded = [
        status.profiles.amneziawg,
        status.profiles.wireguard,
        status.profiles.openvpn,
        status.profiles.l2tp,
    ]
    .into_iter()
    .all(|count| count <= 1_024);
    let fallback_is_consistent = if status.fallback.active {
        safe_frontend_text(&status.fallback.name, 128, false)
    } else {
        status.fallback.name.is_empty()
    };

    if status.schema_version != 1
        || status.product != "Mazzy VPN"
        || !safe_frontend_text(&status.version, 64, false)
        || status.language.is_empty()
        || status.language.len() > 16
        || !status
            .language
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || byte == b'-')
        || !matches!(status.service_state.as_str(), "active" | "inactive")
        || !matches!(status.desired.as_str(), "up" | "down" | "unknown")
        || !matches!(status.mode.as_str(), "normal" | "test")
        || !matches!(status.internet.as_str(), "up" | "down" | "unknown")
        || status.protocol_name != protocol_name
        || !selected_identity_is_complete
        || !interface_is_safe
        || !public_ip_is_safe
        || !profile_counts_are_bounded
        || status.health_failures > 1_000_000
        || !fallback_is_consistent
        || (status.selected && !selected_metadata_is_safe)
        || (!status.selected && !unselected_metadata_is_empty)
        || (status.tunnel_active
            && (!status.selected
                || status.service_state != "active"
                || status.interface.is_empty()))
        || (status.healthy
            && (!status.tunnel_active || status.internet != "up" || status.health_failures != 0))
    {
        return Err("The status cache contains unsafe or inconsistent data".to_owned());
    }
    serde_json::to_value(status).map_err(|_| "The status cache could not be sanitized".to_owned())
}

fn valid_optional_ip_family(value: &Option<String>, ipv4: bool) -> bool {
    value.as_deref().is_none_or(|ip| {
        ip.len() <= 64
            && ip
                .parse::<IpAddr>()
                .is_ok_and(|address| address.is_ipv4() == ipv4)
    })
}

fn sanitize_egress_verification(result: Value) -> Result<Value, String> {
    let verification: EgressVerification = serde_json::from_value(result)
        .map_err(|_| "The VPN egress check returned a malformed result".to_owned())?;
    let protocol_valid =
        |value: &str| matches!(value, "amneziawg" | "wireguard" | "openvpn" | "l2tp");
    let verdict_valid = matches!(
        verification.verdict.as_str(),
        "verified" | "warning" | "failed"
    );
    if verification.schema_version != 1
        || !safe_frontend_text(&verification.checked_at, 64, false)
        || !verdict_valid
        || !valid_api_identifier(&verification.message_key)
        || verification
            .tunnel
            .protocol
            .as_deref()
            .is_some_and(|value| !protocol_valid(value))
        || verification
            .tunnel
            .profile
            .as_deref()
            .is_some_and(|value| !safe_frontend_text(value, 255, false))
        || verification
            .tunnel
            .interface
            .as_deref()
            .is_some_and(|value| {
                value.is_empty()
                    || value.len() > 32
                    || !value
                        .bytes()
                        .all(|byte| byte.is_ascii_alphanumeric() || b"_.:-".contains(&byte))
            })
        || !valid_optional_ip_family(&verification.ipv4.interface_ip, true)
        || !valid_optional_ip_family(&verification.ipv4.default_ip, true)
        || !valid_optional_ip_family(&verification.ipv6.interface_ip, false)
        || !valid_optional_ip_family(&verification.ipv6.default_ip, false)
        || !matches!(
            verification.geo.country_match.as_str(),
            "match" | "mismatch" | "unknown"
        )
        || verification
            .geo
            .expected_country_code
            .as_deref()
            .is_some_and(|value| !valid_country_code(value))
        || verification
            .geo
            .observed_country_code
            .as_deref()
            .is_some_and(|value| !valid_country_code(value))
        || verification.geo.providers.len() > 2
        || !matches!(
            verification.dns.state.as_str(),
            "vpn-full-tunnel" | "vpn-interface" | "unknown" | "unavailable"
        )
        || verification.findings.len() > 32
        || verification
            .findings
            .iter()
            .any(|finding| !valid_api_identifier(finding))
    {
        return Err("The VPN egress check returned an invalid result".to_owned());
    }

    let present_tunnel_fields = [
        verification.tunnel.protocol.is_some(),
        verification.tunnel.profile.is_some(),
        verification.tunnel.interface.is_some(),
    ];
    if verification.tunnel.active && present_tunnel_fields.iter().any(|present| !present) {
        return Err("The VPN egress check omitted active tunnel details".to_owned());
    }
    if verification.ipv4.same_egress
        && (verification.ipv4.interface_ip.is_none()
            || verification.ipv4.interface_ip != verification.ipv4.default_ip)
    {
        return Err("The VPN egress check reported inconsistent IPv4 routing".to_owned());
    }
    if verification.ipv6.potential_leak
        && (verification.ipv6.default_ip.is_none()
            || verification.ipv6.interface_ip == verification.ipv6.default_ip)
    {
        return Err("The VPN egress check reported an inconsistent IPv6 leak".to_owned());
    }

    for provider in &verification.geo.providers {
        let provider_ip = provider.ip.parse::<IpAddr>();
        if !matches!(provider.provider.as_str(), "ipapi" | "ipwho")
            || provider.ip.len() > 64
            || !provider_ip.is_ok_and(|address| address.is_ipv4())
            || !valid_country_code(&provider.country_code)
            || verification.ipv4.interface_ip.as_deref() != Some(provider.ip.as_str())
            || !safe_frontend_text(&provider.country, 96, true)
            || !safe_frontend_text(&provider.region, 96, true)
            || !safe_frontend_text(&provider.city, 96, true)
        {
            return Err("The VPN egress check returned unsafe location data".to_owned());
        }
    }
    let unique_providers: HashSet<&str> = verification
        .geo
        .providers
        .iter()
        .map(|provider| provider.provider.as_str())
        .collect();
    if unique_providers.len() != verification.geo.providers.len() {
        return Err("The VPN egress check duplicated a location provider".to_owned());
    }
    let observed = verification
        .geo
        .providers
        .first()
        .map(|provider| provider.country_code.as_str())
        .filter(|country| !country.is_empty());
    if observed != verification.geo.observed_country_code.as_deref() {
        return Err("The VPN egress check returned inconsistent location data".to_owned());
    }
    if verification.geo.providers_agree.is_some() != (verification.geo.providers.len() == 2) {
        return Err("The VPN egress check returned inconsistent provider agreement".to_owned());
    }
    if let Some(agree) = verification.geo.providers_agree {
        let countries_equal = verification.geo.providers[0].country_code
            == verification.geo.providers[1].country_code;
        if agree != countries_equal {
            return Err("The VPN egress check returned false provider agreement".to_owned());
        }
    }

    match verification.geo.country_match.as_str() {
        "match" => {
            if verification.geo.expected_country_code.is_none()
                || verification.geo.expected_country_code != verification.geo.observed_country_code
            {
                return Err("The VPN egress check returned a false location match".to_owned());
            }
        }
        "mismatch" => {
            if verification.geo.expected_country_code.is_none()
                || verification.geo.observed_country_code.is_none()
                || verification.geo.expected_country_code == verification.geo.observed_country_code
            {
                return Err("The VPN egress check returned a false location mismatch".to_owned());
            }
        }
        _ => {}
    }

    let speed_valid = if verification.speed.measured {
        verification.speed.requested
            && verification
                .speed
                .mbps
                .is_some_and(|value| value.is_finite() && (0.0..=100_000.0).contains(&value))
            && verification
                .speed
                .connect_ms
                .is_some_and(|value| value <= 120_000)
    } else {
        verification.speed.mbps.is_none() && verification.speed.connect_ms.is_none()
    };
    if !speed_valid {
        return Err("The VPN speed sample returned inconsistent data".to_owned());
    }

    let unique_findings: HashSet<&str> = verification.findings.iter().map(String::as_str).collect();
    if unique_findings.len() != verification.findings.len() {
        return Err("The VPN egress check duplicated findings".to_owned());
    }
    if verification.verdict != "verified" && verification.findings.is_empty() {
        return Err("The VPN egress check returned an unexplained verdict".to_owned());
    }
    if verification.verdict != "failed"
        && (!verification.tunnel.active || verification.ipv4.interface_ip.is_none())
    {
        return Err("The VPN egress verdict contradicts the tunnel state".to_owned());
    }
    if verification.verdict == "verified"
        && (!verification.tunnel.active
            || !verification.ipv4.same_egress
            || verification.ipv6.potential_leak
            || verification.geo.expected_country_code.is_none()
            || verification.geo.country_match != "match"
            || verification.geo.providers_agree != Some(true)
            || verification.dns.state != "vpn-full-tunnel"
            || !verification.findings.is_empty())
    {
        return Err("The VPN egress check returned a false verified verdict".to_owned());
    }

    serde_json::to_value(verification)
        .map_err(|_| "The VPN egress result could not be sanitized".to_owned())
}

#[cfg(any(target_os = "linux", test))]
fn verify_api_request(timeout_seconds: u16, include_speed: bool) -> Value {
    json!({
        "api_version": "1.0",
        "request_id": api_identifier("request"),
        "operation": "tests.verify-egress",
        "deadline_ms": 240_000,
        "payload": {
            "timeout_seconds": timeout_seconds,
            "include_speed": include_speed
        }
    })
}

#[cfg(any(target_os = "linux", test))]
fn verify_result_from_response(response: Value) -> Result<Value, String> {
    if response.get("status").and_then(Value::as_str) != Some("ok") {
        let code = response
            .pointer("/error/code")
            .and_then(Value::as_str)
            .unwrap_or("internal-error");
        let message = response
            .pointer("/error/message_key")
            .and_then(Value::as_str)
            .unwrap_or("api.response.malformed");
        return Err(format!("{code}: {message}"));
    }
    let result = response
        .get("result")
        .cloned()
        .ok_or_else(|| "The VPN egress check returned a malformed result".to_owned())?;
    sanitize_egress_verification(result)
}

#[cfg(target_os = "linux")]
pub(crate) fn verify_connection_sync(
    timeout_seconds: u16,
    include_speed: bool,
) -> Result<Value, String> {
    timeout(timeout_seconds, 3, 30)?;

    let request = verify_api_request(timeout_seconds, include_speed);
    match send_local_api_for_read_only(&request) {
        Ok(response) => return verify_result_from_response(response),
        Err(LocalApiError::Unavailable) => {}
        Err(LocalApiError::Indeterminate(error)) => {
            return Err(format!(
                "The local API verification outcome is unknown; it was not repeated \
                 through another privilege path: {error}"
            ));
        }
    }

    let cli_path = installed_cli_path().ok_or_else(|| {
        "Mazzy VPN engine is not installed. Open Settings and run Install / Repair.".to_owned()
    })?;
    if package_managed_cli(cli_path) {
        return Err(engine_not_ready_error());
    }
    let mut command = Command::new(TIMEOUT_PATH);
    command
        .args(["--kill-after=2s", "240s", PKEXEC_PATH])
        .arg(cli_path)
        .args(["verify", "--timeout"])
        .arg(timeout_seconds.to_string())
        .arg("--json");
    if include_speed {
        command.arg("--speed");
    }
    let output = bounded_output(&mut command).map_err(|error| error.to_string())?;
    if let Ok(result) = serde_json::from_slice::<Value>(&output.stdout) {
        if let Ok(sanitized) = sanitize_egress_verification(result) {
            return Ok(sanitized);
        }
    }
    Err(clean_output(&output))
}

#[cfg(not(target_os = "linux"))]
pub(crate) fn verify_connection_sync(
    timeout_seconds: u16,
    _include_speed: bool,
) -> Result<Value, String> {
    timeout(timeout_seconds, 3, 30)?;
    Err("Preview build: actual VPN egress verification is available on Linux only.".to_owned())
}

#[cfg(target_os = "linux")]
pub(crate) fn probe_profiles_sync(
    selected_protocol: String,
    timeout_seconds: u16,
    concurrency: u8,
) -> Result<Value, String> {
    let selected_protocol = protocol(&selected_protocol, true)?;
    timeout(timeout_seconds, 1, 30)?;
    if !(1..=8).contains(&concurrency) {
        return Err("Probe concurrency must be between 1 and 8".to_owned());
    }

    let request = probe_api_request(
        &selected_protocol,
        timeout_seconds,
        concurrency,
        cached_profile_count(&selected_protocol),
    );
    match send_local_api_for_read_only(&request) {
        Ok(response) => return probe_result_from_response(response),
        Err(LocalApiError::Unavailable) => {}
        Err(LocalApiError::Indeterminate(error)) => {
            return Err(format!(
                "The local API probe outcome is unknown; the request was not repeated \
                 through another privilege path: {error}"
            ));
        }
    }

    let cli_path = installed_cli_path().ok_or_else(|| {
        "Mazzy VPN engine is not installed. Open Settings and run Install / Repair.".to_owned()
    })?;
    if package_managed_cli(cli_path) {
        return Err(engine_not_ready_error());
    }
    let probe_deadline = probe_deadline_ms(
        cached_profile_count(&selected_protocol),
        timeout_seconds,
        concurrency,
    )
    .div_ceil(1_000);
    let mut command = Command::new(TIMEOUT_PATH);
    command
        .arg("--kill-after=2s")
        .arg(format!("{probe_deadline}s"))
        .arg(PKEXEC_PATH)
        .arg(cli_path)
        .arg("probe")
        .arg(&selected_protocol)
        .arg("--timeout")
        .arg(timeout_seconds.to_string())
        .arg("--jobs")
        .arg(concurrency.to_string())
        .arg("--json");
    let output = bounded_output(&mut command).map_err(|error| error.to_string())?;
    if let Ok(result) = serde_json::from_slice::<Value>(&output.stdout) {
        if let Ok(sanitized) = sanitize_probe_collection(result) {
            return Ok(sanitized);
        }
    }
    Err(clean_output(&output))
}

#[cfg(not(target_os = "linux"))]
pub(crate) fn probe_profiles_sync(
    selected_protocol: String,
    timeout_seconds: u16,
    concurrency: u8,
) -> Result<Value, String> {
    protocol(&selected_protocol, true)?;
    timeout(timeout_seconds, 1, 30)?;
    if !(1..=8).contains(&concurrency) {
        return Err("Probe concurrency must be between 1 and 8".to_owned());
    }
    Err("Preview build: VPN profile reachability checks are available on Linux only.".to_owned())
}

fn api_operation_result(action: String, action_id: &str, response: Value) -> OperationResult {
    if response.get("status").and_then(Value::as_str) == Some("ok") {
        let result = response.get("result").cloned().unwrap_or(Value::Null);
        let state = result
            .get("state")
            .and_then(Value::as_str)
            .unwrap_or("failed");
        let message = result
            .get("message_key")
            .and_then(Value::as_str)
            .unwrap_or("api.response.malformed");
        return OperationResult {
            success: state == "succeeded",
            action,
            output: if state == "succeeded" {
                message.into()
            } else {
                format!("{message}; action ID: {action_id}")
            },
            code: None,
        };
    }
    let error = response.get("error").cloned().unwrap_or(Value::Null);
    let code = error
        .get("code")
        .and_then(Value::as_str)
        .unwrap_or("internal-error");
    let message = error
        .get("message_key")
        .and_then(Value::as_str)
        .unwrap_or("api.response.malformed");
    OperationResult {
        success: false,
        action,
        output: format!("{code}: {message}; action ID: {action_id}"),
        code: None,
    }
}

#[cfg(target_os = "linux")]
fn local_api_response_timeout(request: &Value) -> std::time::Duration {
    let deadline_ms = request
        .get("deadline_ms")
        .and_then(Value::as_u64)
        .unwrap_or(5_000)
        .clamp(100, 900_000);
    let grace_ms = if request.get("action_id").is_some() {
        LOCAL_API_COMPLETION_GRACE_MS
    } else {
        5_000
    };
    std::time::Duration::from_millis(deadline_ms + grace_ms)
}

#[cfg(target_os = "linux")]
fn parse_local_api_response(response: &[u8], request: &Value) -> Result<Value, LocalApiError> {
    if response.is_empty() || response.len() > MAX_API_RESPONSE_BYTES {
        return Err(LocalApiError::Indeterminate(
            "Local API returned an empty or oversized response".into(),
        ));
    }
    let response: Value = serde_json::from_slice(response).map_err(|error| {
        LocalApiError::Indeterminate(format!("Invalid local API response: {error}"))
    })?;
    if response.get("api_version").and_then(Value::as_str) != Some("1.0")
        || response.get("request_id").and_then(Value::as_str)
            != request.get("request_id").and_then(Value::as_str)
    {
        return Err(LocalApiError::Indeterminate(
            "Local API response identity does not match the request".into(),
        ));
    }
    Ok(response)
}

#[cfg(target_os = "linux")]
fn send_local_api(request: &Value) -> Result<Value, LocalApiError> {
    let mut stream = UnixStream::connect(API_SOCKET).map_err(|_| LocalApiError::Unavailable)?;
    stream
        .set_read_timeout(Some(local_api_response_timeout(request)))
        .map_err(|error| LocalApiError::Indeterminate(error.to_string()))?;
    stream
        .set_write_timeout(Some(std::time::Duration::from_secs(5)))
        .map_err(|error| LocalApiError::Indeterminate(error.to_string()))?;
    serde_json::to_writer(&mut stream, request)
        .map_err(|error| LocalApiError::Indeterminate(error.to_string()))?;
    stream
        .write_all(b"\n")
        .map_err(|error| LocalApiError::Indeterminate(error.to_string()))?;
    stream
        .flush()
        .map_err(|error| LocalApiError::Indeterminate(error.to_string()))?;

    let mut response = Vec::new();
    stream
        .take((MAX_API_RESPONSE_BYTES + 1) as u64)
        .read_to_end(&mut response)
        .map_err(|error| LocalApiError::Indeterminate(error.to_string()))?;
    parse_local_api_response(&response, request)
}

#[cfg(target_os = "linux")]
fn retry_indeterminate<T>(
    mut attempt: impl FnMut() -> Result<T, LocalApiError>,
) -> Result<T, LocalApiError> {
    match attempt() {
        Err(LocalApiError::Indeterminate(first_error)) => match attempt() {
            Err(LocalApiError::Unavailable) => Err(LocalApiError::Indeterminate(format!(
                "{first_error}; identical retry could not reconnect"
            ))),
            Err(LocalApiError::Indeterminate(second_error)) => Err(LocalApiError::Indeterminate(
                format!("{first_error}; identical retry failed: {second_error}"),
            )),
            result => result,
        },
        result => result,
    }
}

#[cfg(target_os = "linux")]
fn send_local_api_with_retry(request: &Value) -> Result<Value, LocalApiError> {
    retry_indeterminate(|| send_local_api(request))
}

#[cfg(target_os = "linux")]
fn send_local_api_for_read_only(request: &Value) -> Result<Value, LocalApiError> {
    for (attempt, delay_ms) in READ_ONLY_API_RETRY_DELAYS_MS.iter().enumerate() {
        if attempt > 0 {
            thread::sleep(std::time::Duration::from_millis(*delay_ms));
        }
        match send_local_api_with_retry(request) {
            Err(LocalApiError::Unavailable) => {}
            result => return result,
        }
    }
    Err(LocalApiError::Unavailable)
}

#[cfg(target_os = "linux")]
fn wait_for_local_api() -> bool {
    for (attempt, delay_ms) in API_STARTUP_RETRY_DELAYS_MS.iter().enumerate() {
        if attempt > 0 {
            thread::sleep(std::time::Duration::from_millis(*delay_ms));
        }
        if UnixStream::connect(API_SOCKET).is_ok() {
            return true;
        }
    }
    false
}

#[cfg(not(target_os = "linux"))]
fn wait_for_local_api() -> bool {
    true
}

#[cfg(target_os = "linux")]
fn try_execute_local_api(request: &OperationRequest) -> Option<OperationResult> {
    if !Path::new(API_SOCKET).exists() {
        return None;
    }
    let catalog = fs::read_to_string(PROFILES_FILE)
        .ok()
        .and_then(|contents| serde_json::from_str(&contents).ok());
    let (action, envelope) = match local_api_request(request, catalog.as_ref()) {
        Ok(Some(specification)) => specification,
        Ok(None) => return None,
        Err(error) => {
            return Some(OperationResult {
                success: false,
                action: "validation".into(),
                output: error,
                code: None,
            });
        }
    };
    let action_id = envelope
        .get("action_id")
        .and_then(Value::as_str)
        .unwrap_or("unknown-action")
        .to_owned();
    match send_local_api_with_retry(&envelope) {
        Ok(response) => Some(api_operation_result(action, &action_id, response)),
        Err(LocalApiError::Unavailable) => None,
        Err(LocalApiError::Indeterminate(error)) => Some(OperationResult {
            success: false,
            action,
            output: format!(
                "Local API outcome is unknown; do not retry automatically. \
                 Inspect action ID {action_id}: {error}"
            ),
            code: None,
        }),
    }
}

#[cfg(not(target_os = "linux"))]
fn try_execute_local_api(_: &OperationRequest) -> Option<OperationResult> {
    None
}

pub(crate) fn execute_operation(app: &AppHandle, request: OperationRequest) -> OperationResult {
    if matches!(request, OperationRequest::Bootstrap) {
        if Path::new(SYSTEM_CLI_PATH).is_file() && !wait_for_local_api() {
            return OperationResult {
                success: false,
                action: "bootstrap".into(),
                output: engine_not_ready_error(),
                code: None,
            };
        }
        if Path::new(SYSTEM_CLI_PATH).is_file() {
            return timed_operation_result(
                "bootstrap".into(),
                bounded_output(
                    Command::new(TIMEOUT_PATH)
                        .args(["--kill-after=30s", "1800s", PKEXEC_PATH])
                        .arg(SYSTEM_CLI_PATH)
                        .args(["doctor", "--fix"]),
                ),
                true,
            );
        }
        let installer = engine_root(app).join("install.sh");
        if !installer.is_file() {
            return OperationResult {
                success: false,
                action: "bootstrap".into(),
                output: "The embedded Mazzy VPN installer is missing".into(),
                code: None,
            };
        }
        return timed_operation_result(
            "bootstrap".into(),
            bounded_output(
                Command::new(TIMEOUT_PATH)
                    .args(["--kill-after=30s", "1800s", PKEXEC_PATH])
                    .arg("/bin/bash")
                    .arg(installer)
                    .arg("--yes")
                    .arg("--skip-tests"),
            ),
            true,
        );
    }

    let (action, args) = match build_cli_args(&request) {
        Ok(specification) => specification,
        Err(error) => {
            return OperationResult {
                success: false,
                action: "validation".into(),
                output: error,
                code: None,
            };
        }
    };
    let Some(cli_path) = installed_cli_path() else {
        return OperationResult {
            success: false,
            action,
            output: "Mazzy VPN engine is not installed. Open Settings and run Install / Repair."
                .into(),
            code: None,
        };
    };
    if package_managed_cli(cli_path) && !wait_for_local_api() {
        return OperationResult {
            success: false,
            action,
            output: engine_not_ready_error(),
            code: None,
        };
    }
    if let Some(result) = try_execute_local_api(&request) {
        return result;
    }
    let deadline_seconds = operation_deadline_seconds(&request);
    let changes_system = operation_changes_system(&request);
    timed_operation_result(
        action,
        timed_privileged_output(cli_path, &args, deadline_seconds),
        changes_system,
    )
}

fn version_from_output(output: &str) -> Option<String> {
    output
        .split_whitespace()
        .find(|part| {
            part.chars()
                .next()
                .is_some_and(|character| character.is_ascii_digit())
                && part.contains('.')
        })
        .map(|part| {
            part.trim_matches(|character: char| {
                !character.is_ascii_alphanumeric() && character != '.'
            })
            .to_owned()
        })
}

fn installed_version(path: &Path) -> Option<String> {
    let output = bounded_output(
        Command::new(TIMEOUT_PATH)
            .args(["--kill-after=2s", "10s"])
            .arg(path)
            .arg("version"),
    )
    .ok()?;
    output
        .status
        .success()
        .then(|| version_from_output(&String::from_utf8_lossy(&output.stdout)))
        .flatten()
}

fn bundled_version(root: &Path) -> Option<String> {
    let source = fs::read_to_string(root.join("mazzy-vpn")).ok()?;
    source.lines().find_map(|line| {
        line.strip_prefix("VERSION=")
            .map(|value| value.trim_matches('"').to_owned())
    })
}

fn command_path(command: &str) -> Option<PathBuf> {
    let mut directories: Vec<PathBuf> = env::var_os("PATH")
        .map(|path| env::split_paths(&path).collect())
        .unwrap_or_default();
    directories.extend(
        [
            "/usr/local/bin",
            "/usr/local/sbin",
            "/usr/bin",
            "/usr/sbin",
            "/bin",
            "/sbin",
            "/usr/libexec",
        ]
        .into_iter()
        .map(PathBuf::from),
    );
    directories
        .into_iter()
        .map(|directory| directory.join(command))
        .find(|candidate| candidate.is_file())
}

fn command_available(command: &str) -> bool {
    command_path(command).is_some()
}

fn any_command_available(commands: &[&str]) -> bool {
    commands.iter().any(|command| command_available(command))
}

fn all_commands_available(commands: &[&str]) -> bool {
    commands.iter().all(|command| command_available(command))
}

fn any_path_available(paths: &[&str]) -> bool {
    paths.iter().any(|path| Path::new(path).is_file())
}

fn amnezia_backend_available() -> bool {
    Path::new("/sys/module/amneziawg").is_dir()
        || command_available("amneziawg-go")
        || command_path("modprobe").is_some_and(|modprobe| {
            Command::new(modprobe)
                .args(["-n", "amneziawg"])
                .status()
                .is_ok_and(|status| status.success())
        })
}

fn dependencies() -> Vec<DependencyState> {
    let definitions = [
        ("bash", "Bash", "core", command_available("bash")),
        ("ip", "iproute2", "core", command_available("ip")),
        ("curl", "curl", "core", command_available("curl")),
        ("ping", "ICMP ping", "core", command_available("ping")),
        ("timeout", "timeout", "core", command_available("timeout")),
        ("flock", "flock", "core", command_available("flock")),
        (
            "getent",
            "DNS resolver",
            "core",
            command_available("getent"),
        ),
        (
            "systemd",
            "systemd + transient units",
            "core",
            all_commands_available(&["systemctl", "systemd-run"]),
        ),
        (
            "journalctl",
            "system journal",
            "core",
            command_available("journalctl"),
        ),
        (
            "pkexec",
            "PolicyKit authorization",
            "Desktop",
            command_available("pkexec"),
        ),
        (
            "jq",
            "JSON API runtime",
            "local API",
            command_available("jq"),
        ),
        (
            "socat",
            "Unix-socket API client",
            "CLI / TUI local API",
            command_available("socat"),
        ),
        (
            "dns-integration",
            "DNS integration",
            "VPN DNS",
            any_command_available(&["resolvectl", "resolvconf"]),
        ),
        (
            "openvpn",
            "OpenVPN",
            "OpenVPN",
            command_available("openvpn"),
        ),
        (
            "wireguard-tools",
            "WireGuard tools",
            "WireGuard",
            all_commands_available(&["wg", "wg-quick"]),
        ),
        (
            "amneziawg-tools",
            "AmneziaWG tools",
            "AmneziaWG",
            all_commands_available(&["awg", "awg-quick"]),
        ),
        (
            "amneziawg-backend",
            "AmneziaWG kernel/userspace backend",
            "AmneziaWG",
            amnezia_backend_available(),
        ),
        (
            "networkmanager",
            "NetworkManager",
            "L2TP/IPsec",
            command_available("nmcli"),
        ),
        (
            "networkmanager-l2tp",
            "NetworkManager L2TP",
            "L2TP/IPsec",
            command_available("nm-l2tp-service")
                || any_path_available(&[
                    "/usr/lib/NetworkManager/nm-l2tp-service",
                    "/usr/libexec/nm-l2tp-service",
                ]),
        ),
        (
            "ipsec",
            "strongSwan/Libreswan",
            "L2TP/IPsec",
            any_command_available(&["charon", "charon-systemd", "ipsec", "pluto"])
                || any_path_available(&[
                    "/usr/lib/ipsec/charon",
                    "/usr/libexec/ipsec/charon",
                    "/usr/libexec/ipsec/pluto",
                ]),
        ),
        ("ppp", "PPP", "L2TP/IPsec", command_available("pppd")),
        (
            "l2tp-transport",
            "L2TP transport",
            "L2TP/IPsec",
            any_command_available(&["xl2tpd", "kl2tpd"]),
        ),
    ];
    definitions
        .into_iter()
        .map(|(id, label, required_for, installed)| DependencyState {
            id,
            label,
            installed,
            required_for,
        })
        .collect()
}

#[tauri::command]
pub fn get_profiles() -> Value {
    let response = profile_cache_response(fs::read_to_string(PROFILES_FILE));
    if profile_cache_is_available(&response) {
        let count = response
            .get("profiles")
            .and_then(Value::as_array)
            .map_or(0, Vec::len);
        log::info!(
            target: "mazzy_vpn_desktop::profiles",
            "profiles.loaded count={count}"
        );
    } else {
        let error_code = response
            .get("error_code")
            .and_then(Value::as_str)
            .unwrap_or("profile-cache-unavailable");
        log::warn!(
            target: "mazzy_vpn_desktop::profiles",
            "profiles.unavailable code={error_code}"
        );
    }
    response
}

fn profile_cache_is_available(response: &Value) -> bool {
    response
        .get("available")
        .and_then(Value::as_bool)
        .unwrap_or(true)
}

fn unavailable_profile_cache(error_code: &str, error: &str) -> Value {
    json!({
        "schema_version": 1,
        "generated_at": 0,
        "available": false,
        "error_code": error_code,
        "error": error,
        "profiles": []
    })
}

fn profile_cache_response(contents: io::Result<String>) -> Value {
    match contents {
        Ok(contents) => sanitize_profile_cache(&contents)
            .unwrap_or_else(|error| unavailable_profile_cache("profile-cache-invalid", &error)),
        Err(error) => {
            let (code, message) = match error.kind() {
                io::ErrorKind::NotFound => (
                    "profile-cache-missing",
                    "Profile cache is not ready. Refresh the engine cache.",
                ),
                io::ErrorKind::PermissionDenied => (
                    "profile-cache-permission-denied",
                    "Profile cache is not readable by the Desktop process.",
                ),
                _ => (
                    "profile-cache-unavailable",
                    "Profile cache could not be read. Refresh the engine cache.",
                ),
            };
            unavailable_profile_cache(code, message)
        }
    }
}

#[tauri::command]
pub async fn probe_profiles(
    protocol: String,
    timeout: u16,
    concurrency: u8,
) -> Result<Value, String> {
    let result = tauri::async_runtime::spawn_blocking(move || {
        probe_profiles_sync(protocol, timeout, concurrency)
    })
    .await
    .map_err(|error| error.to_string())?;
    match &result {
        Ok(report) => log::info!(
            target: "mazzy_vpn_desktop::probe",
            "probe.complete total={} reachable={} unknown={}",
            report.pointer("/summary/total").and_then(Value::as_u64).unwrap_or(0),
            report.pointer("/summary/reachable").and_then(Value::as_u64).unwrap_or(0),
            report.pointer("/summary/unknown").and_then(Value::as_u64).unwrap_or(0)
        ),
        Err(_) => log::warn!(target: "mazzy_vpn_desktop::probe", "probe.failed"),
    }
    result
}

#[tauri::command]
pub async fn verify_connection(timeout: u16, include_speed: bool) -> Result<Value, String> {
    let result = tauri::async_runtime::spawn_blocking(move || {
        verify_connection_sync(timeout, include_speed)
    })
    .await
    .map_err(|error| error.to_string())?;
    match &result {
        Ok(report) => log::info!(
            target: "mazzy_vpn_desktop::verification",
            "verification.complete verdict={} findings={} speed_requested={}",
            report.get("verdict").and_then(Value::as_str).unwrap_or("unknown"),
            report.get("findings").and_then(Value::as_array).map_or(0, Vec::len),
            report.pointer("/speed/requested").and_then(Value::as_bool).unwrap_or(false)
        ),
        Err(_) => log::warn!(
            target: "mazzy_vpn_desktop::verification",
            "verification.failed"
        ),
    }
    result
}

#[tauri::command]
pub fn get_installation_report(app: AppHandle) -> InstallationReport {
    let root = engine_root(&app);
    let cli_path = installed_cli_path();
    let installed_version = cli_path.and_then(installed_version);
    let package_managed = cli_path == Some(Path::new(SYSTEM_CLI_PATH));
    let bundled_version = bundled_version(&root);
    let engine_installed = installed_version.is_some();
    let dependencies = dependencies();
    let missing_dependencies = dependencies
        .iter()
        .filter(|dependency| !dependency.installed)
        .count();
    let dependencies_ready = missing_dependencies == 0;
    let service_installed = systemd_unit_installed("vpnctl.service");
    let monitor_installed = systemd_unit_installed("vpnctl-health.timer");
    let api_installed = systemd_unit_installed("mazzy-vpn-api.socket");
    let api_socket_available = Path::new(API_SOCKET).exists();
    let needs_install = !engine_installed
        || installed_version != bundled_version
        || !dependencies_ready
        || !service_installed
        || !monitor_installed
        || !api_installed;
    InstallationReport {
        engine_installed,
        package_managed,
        installed_version,
        bundled_version,
        bundled_installer: root.join("install.sh").is_file(),
        needs_install,
        service_installed,
        monitor_installed,
        api_installed,
        api_socket_available,
        dependencies_ready,
        missing_dependencies,
        dependencies,
    }
}

#[tauri::command]
pub async fn run_operation(
    app: AppHandle,
    request: OperationRequest,
) -> Result<OperationResult, String> {
    let result = tauri::async_runtime::spawn_blocking(move || execute_operation(&app, request))
        .await
        .map_err(|error| error.to_string());
    match &result {
        Ok(operation) => log::info!(
            target: "mazzy_vpn_desktop::operation",
            "operation.complete action={} success={} code={}",
            operation.action,
            operation.success,
            operation.code.map_or_else(|| "none".to_owned(), |code| code.to_string())
        ),
        Err(_) => log::error!(
            target: "mazzy_vpn_desktop::operation",
            "operation.worker_failed"
        ),
    }
    result
}

#[tauri::command]
pub async fn pick_profile_files() -> Result<Vec<String>, String> {
    tauri::async_runtime::spawn_blocking(|| {
        rfd::FileDialog::new()
            .set_title("Select Mazzy VPN profiles")
            .add_filter("VPN profiles", &["conf", "ovpn", "nmconnection"])
            .pick_files()
            .unwrap_or_default()
            .into_iter()
            .filter_map(|path| path.into_os_string().into_string().ok())
            .collect()
    })
    .await
    .map_err(|error| error.to_string())
}

#[tauri::command]
pub async fn pick_profile_folder() -> Result<Option<String>, String> {
    tauri::async_runtime::spawn_blocking(|| {
        rfd::FileDialog::new()
            .set_title("Select a folder with Mazzy VPN profiles")
            .pick_folder()
            .and_then(|path| path.into_os_string().into_string().ok())
    })
    .await
    .map_err(|error| error.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn typed_requests_map_to_fixed_cli_arguments() {
        let (_, args) = build_cli_args(&OperationRequest::Connect {
            protocol: "openvpn".into(),
            profile: "Server; still one argument".into(),
        })
        .expect("valid request");
        assert_eq!(
            args,
            ["connect", "openvpn", "Server; still one argument"]
                .map(str::to_owned)
                .to_vec()
        );
    }

    #[test]
    fn unsafe_profile_names_and_timeouts_are_rejected() {
        assert!(
            build_cli_args(&OperationRequest::Connect {
                protocol: "openvpn".into(),
                profile: "../secret".into(),
            })
            .is_err()
        );
        assert!(
            build_cli_args(&OperationRequest::Probe {
                protocol: "all".into(),
                timeout: 31,
            })
            .is_err()
        );
    }

    #[test]
    fn ansi_output_is_removed_before_reaching_the_webview() {
        let output = std::process::Command::new("/usr/bin/printf")
            .args(["\\033[31mFAIL\\033[0m\\n"])
            .output()
            .expect("printf");
        assert_eq!(clean_output(&output), "FAIL");
    }

    #[test]
    fn policykit_terminal_errors_are_actionable() {
        let message = normalize_privileged_error(
            "Error creating textual authentication agent: Error opening current controlling terminal for the process (/dev/tty): No such device or address".into(),
        );
        assert!(message.contains("PolicyKit authorization is unavailable"));
        assert!(!message.contains("/dev/tty"));
    }

    #[test]
    fn child_output_capture_is_bounded_and_marks_truncation() {
        let input = vec![b'x'; MAX_OUTPUT_STREAM_BYTES + 1];
        let output =
            drain_bounded(std::io::Cursor::new(input), MAX_OUTPUT_STREAM_BYTES).expect("capture");
        assert_eq!(output.len(), MAX_OUTPUT_STREAM_BYTES);
        assert!(output.ends_with(TRUNCATED_OUTPUT_MARKER));
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn timeout_kills_descendants_that_keep_output_pipes_open() {
        let started = std::time::Instant::now();
        let output = bounded_output(Command::new(TIMEOUT_PATH).args([
            "--kill-after=1s",
            "0.1s",
            "/bin/sh",
            "-c",
            "sleep 30 & wait",
        ]))
        .expect("bounded timeout");
        assert_eq!(output.status.code(), Some(124));
        assert!(started.elapsed() < std::time::Duration::from_secs(5));
    }

    #[test]
    fn dependency_report_covers_the_cli_and_desktop_runtime() {
        let states = dependencies();
        for required in [
            "bash",
            "ip",
            "curl",
            "systemd",
            "journalctl",
            "pkexec",
            "jq",
            "socat",
            "openvpn",
            "wireguard-tools",
            "amneziawg-tools",
            "amneziawg-backend",
            "networkmanager-l2tp",
            "ipsec",
            "ppp",
            "l2tp-transport",
        ] {
            assert!(
                states.iter().any(|state| state.id == required),
                "missing dependency state: {required}"
            );
        }
    }

    #[test]
    fn package_managed_engine_takes_precedence_over_local_installs() {
        assert_eq!(
            select_cli_path(true, true),
            Some(Path::new(SYSTEM_CLI_PATH))
        );
        assert_eq!(
            select_cli_path(false, true),
            Some(Path::new(LOCAL_CLI_PATH))
        );
        assert_eq!(select_cli_path(false, false), None);
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn package_startup_wait_has_a_finite_budget() {
        let budget_ms: u64 = API_STARTUP_RETRY_DELAYS_MS.iter().sum();
        assert_eq!(budget_ms, 15_750);
        assert!(package_managed_cli(Path::new(SYSTEM_CLI_PATH)));
        assert!(!package_managed_cli(Path::new(LOCAL_CLI_PATH)));
    }

    #[test]
    fn engine_not_ready_error_explains_the_safe_fallback() {
        let message = engine_not_ready_error();
        assert!(message.contains("local API is still starting"));
        assert!(message.contains("no privileged fallback"));
        assert!(!message.contains("pkexec"));
    }

    #[test]
    fn lifecycle_requests_use_opaque_local_api_identifiers() {
        let catalog = json!({
            "profiles": [{
                "profile_id": "profile-0123456789abcdef0123456789abcdef",
                "protocol": "openvpn",
                "file_name": "Server.ovpn"
            }]
        });
        let (_, request) = local_api_request(
            &OperationRequest::Connect {
                protocol: "openvpn".into(),
                profile: "Server.ovpn".into(),
            },
            Some(&catalog),
        )
        .expect("request mapping")
        .expect("supported operation");
        assert_eq!(request["operation"], "lifecycle.connect");
        assert_eq!(
            request["payload"]["profile_id"],
            "profile-0123456789abcdef0123456789abcdef"
        );
        let encoded = serde_json::to_string(&request).expect("JSON");
        assert!(!encoded.contains("Server.ovpn"));
        assert_eq!(request["authorization"], "system-mutate");
    }

    #[test]
    fn batch_probe_request_is_bounded_and_contains_no_endpoint() {
        let request = probe_api_request("openvpn", 3, 4, 25);
        assert_eq!(request["operation"], "tests.probe");
        assert_eq!(request["payload"]["protocol"], "openvpn");
        assert_eq!(request["payload"]["timeout_seconds"], 3);
        assert_eq!(request["payload"]["concurrency"], 4);
        assert_eq!(request["deadline_ms"], 89_000);
        let encoded = serde_json::to_string(&request).expect("JSON");
        assert!(!encoded.contains("endpoint"));
        assert!(!encoded.contains("file_name"));
    }

    #[test]
    fn probe_response_keeps_reachability_latency_and_active_state() {
        let response = json!({
            "status": "ok",
            "result": {
                "schema_version": 1,
                "checked_at": "2026-07-28T12:00:00Z",
                "scope": "openvpn",
                "timeout_seconds": 3,
                "concurrency": 4,
                "duration_ms": 20,
                "summary": {
                    "total": 1,
                    "reachable": 1,
                    "unknown": 0,
                    "unreachable": 0,
                    "invalid": 0,
                    "active": 1
                },
                "results": [{
                    "profile_id": "profile-01234567",
                    "display_name": "Test Server",
                    "protocol": "openvpn",
                    "selected": true,
                    "active": true,
                    "transport": "udp",
                    "reachability": "reachable",
                    "latency_ms": 14,
                    "latency_source": "icmp",
                    "message_key": "probe.reachable.icmp"
                }]
            }
        });
        let result = probe_result_from_response(response.clone()).expect("probe result");
        assert_eq!(result["results"][0]["latency_ms"], 14);
        assert_eq!(result["results"][0]["active"], true);

        let mut unsafe_response = response;
        unsafe_response["result"]["results"][0]["endpoint"] = json!("vpn.invalid:1194");
        assert!(probe_result_from_response(unsafe_response).is_err());
    }

    fn verified_egress_response() -> Value {
        json!({
            "status": "ok",
            "result": {
                "schema_version": 1,
                "checked_at": "2026-07-28T12:00:00Z",
                "verdict": "verified",
                "message_key": "verify.verified",
                "tunnel": {
                    "active": true,
                    "protocol": "openvpn",
                    "profile": "Belgium Brussels",
                    "interface": "vpnovpn0"
                },
                "ipv4": {
                    "interface_ip": "203.0.113.7",
                    "default_ip": "203.0.113.7",
                    "same_egress": true
                },
                "ipv6": {
                    "interface_ip": null,
                    "default_ip": null,
                    "potential_leak": false
                },
                "geo": {
                    "expected_country_code": "BE",
                    "observed_country_code": "BE",
                    "country_match": "match",
                    "providers_agree": true,
                    "providers": [
                        {
                            "provider": "ipapi",
                            "ip": "203.0.113.7",
                            "country_code": "BE",
                            "country": "Belgium",
                            "region": "Brussels",
                            "city": "Brussels"
                        },
                        {
                            "provider": "ipwho",
                            "ip": "203.0.113.7",
                            "country_code": "BE",
                            "country": "Belgium",
                            "region": "Brussels",
                            "city": "Brussels"
                        }
                    ]
                },
                "dns": {"state": "vpn-full-tunnel"},
                "speed": {
                    "requested": false,
                    "measured": false,
                    "mbps": null,
                    "connect_ms": null
                },
                "findings": []
            }
        })
    }

    #[test]
    fn egress_request_is_explicit_bounded_and_contains_no_profile_data() {
        let request = verify_api_request(10, true);
        assert_eq!(request["operation"], "tests.verify-egress");
        assert_eq!(request["deadline_ms"], 240_000);
        assert_eq!(request["payload"]["timeout_seconds"], 10);
        assert_eq!(request["payload"]["include_speed"], true);
        let encoded = serde_json::to_string(&request).expect("JSON");
        assert!(!encoded.contains("profile"));
        assert!(!encoded.contains("endpoint"));
        assert!(!encoded.contains("key"));
    }

    #[test]
    fn egress_response_rejects_false_verdicts_and_unknown_fields() {
        let response = verified_egress_response();
        let result = verify_result_from_response(response.clone()).expect("egress result");
        assert_eq!(result["verdict"], "verified");
        assert_eq!(result["geo"]["observed_country_code"], "BE");

        let mut false_verdict = response.clone();
        false_verdict["result"]["ipv6"]["potential_leak"] = json!(true);
        assert!(verify_result_from_response(false_verdict).is_err());

        let mut wrong_provider_ip = response.clone();
        wrong_provider_ip["result"]["geo"]["providers"][0]["ip"] = json!("198.51.100.22");
        assert!(verify_result_from_response(wrong_provider_ip).is_err());

        let mut wrong_ip_family = response.clone();
        wrong_ip_family["result"]["ipv4"]["interface_ip"] = json!("2001:db8::1");
        wrong_ip_family["result"]["ipv4"]["default_ip"] = json!("2001:db8::1");
        assert!(verify_result_from_response(wrong_ip_family).is_err());

        let mut duplicate_provider = response.clone();
        duplicate_provider["result"]["geo"]["providers"][1]["provider"] = json!("ipapi");
        assert!(verify_result_from_response(duplicate_provider).is_err());

        let mut unexplained_warning = response.clone();
        unexplained_warning["result"]["verdict"] = json!("warning");
        assert!(verify_result_from_response(unexplained_warning).is_err());

        let mut missing_expected_country = response.clone();
        missing_expected_country["result"]["geo"]["expected_country_code"] = Value::Null;
        missing_expected_country["result"]["geo"]["country_match"] = json!("unknown");
        assert!(verify_result_from_response(missing_expected_country).is_err());

        let mut unsafe_response = response;
        unsafe_response["result"]["endpoint"] = json!("vpn.invalid:1194");
        assert!(verify_result_from_response(unsafe_response).is_err());
    }

    #[test]
    fn profile_cache_accepts_config_metadata_and_rejects_runtime_injection() {
        let cache = json!({
            "schema_version": 1,
            "generated_at": 1,
            "profiles": [{
                "profile_id": "profile-0123456789abcdef0123456789abcdef",
                "protocol": "openvpn",
                "protocol_name": "OpenVPN",
                "file_name": "opaque-profile.ovpn",
                "name": "AI workspace",
                "location": "Belgium — Brussels",
                "country_code": "BE",
                "selected": true
            }]
        });
        let encoded = serde_json::to_string(&cache).expect("profile cache JSON");
        let sanitized = sanitize_profile_cache(&encoded).expect("sanitized profile cache");
        assert_eq!(sanitized["profiles"][0]["location"], "Belgium — Brussels");
        assert!(profile_cache_is_available(&sanitized));

        let legacy_cache = json!({
            "schema_version": 1,
            "generated_at": 1,
            "profiles": [{
                "protocol": "openvpn",
                "protocol_name": "OpenVPN",
                "file_name": "opaque-profile.ovpn",
                "name": "AI workspace",
                "location": "Belgium — Brussels",
                "selected": false
            }]
        });
        let sanitized_legacy = sanitize_profile_cache(
            &serde_json::to_string(&legacy_cache).expect("legacy cache JSON"),
        )
        .expect("sanitized legacy profile cache");
        assert_eq!(
            sanitized_legacy["profiles"][0]["profile_id"],
            "profile-8137e652e96544ff40ae4ddf305f874a"
        );
        assert!(sanitized_legacy["profiles"][0]["country_code"].is_null());

        let mut endpoint_injection = cache.clone();
        endpoint_injection["profiles"][0]["endpoint"] = json!("vpn.invalid:1194");
        assert!(
            sanitize_profile_cache(
                &serde_json::to_string(&endpoint_injection).expect("injected cache JSON")
            )
            .is_err()
        );

        let mut path_injection = cache;
        path_injection["profiles"][0]["file_name"] = json!("../escape.ovpn");
        assert!(
            sanitize_profile_cache(
                &serde_json::to_string(&path_injection).expect("path cache JSON")
            )
            .is_err()
        );

        let duplicate = json!({
            "schema_version": 1,
            "generated_at": 1,
            "profiles": [
                {
                    "profile_id": "profile-0123456789abcdef0123456789abcdef",
                    "protocol": "openvpn",
                    "protocol_name": "OpenVPN",
                    "file_name": "first.ovpn",
                    "name": "First",
                    "location": "First",
                    "country_code": null,
                    "selected": false
                },
                {
                    "profile_id": "profile-0123456789abcdef0123456789abcdef",
                    "protocol": "openvpn",
                    "protocol_name": "OpenVPN",
                    "file_name": "second.ovpn",
                    "name": "Second",
                    "location": "Second",
                    "country_code": null,
                    "selected": false
                }
            ]
        });
        assert!(
            sanitize_profile_cache(
                &serde_json::to_string(&duplicate).expect("duplicate cache JSON")
            )
            .is_err()
        );

        let mut bidi_injection = json!({
            "schema_version": 1,
            "generated_at": 1,
            "profiles": [{
                "profile_id": "profile-fedcba9876543210fedcba9876543210",
                "protocol": "wireguard",
                "protocol_name": "WireGuard",
                "file_name": "safe.conf",
                "name": "Safe",
                "location": "Safe",
                "country_code": null,
                "selected": false
            }]
        });
        bidi_injection["profiles"][0]["name"] = json!("office\u{202e}conf");
        assert!(
            sanitize_profile_cache(
                &serde_json::to_string(&bidi_injection).expect("bidi cache JSON")
            )
            .is_err()
        );
    }

    #[test]
    fn profile_cache_failures_are_reported_as_unavailable_not_empty() {
        let missing = profile_cache_response(Err(io::Error::new(
            io::ErrorKind::NotFound,
            "fixture missing",
        )));
        assert_eq!(missing["available"], false);
        assert!(!profile_cache_is_available(&missing));
        assert_eq!(missing["error_code"], "profile-cache-missing");
        assert_eq!(missing["profiles"], json!([]));

        let denied = profile_cache_response(Err(io::Error::new(
            io::ErrorKind::PermissionDenied,
            "fixture denied",
        )));
        assert_eq!(denied["error_code"], "profile-cache-permission-denied");

        let invalid = profile_cache_response(Ok("{}".to_owned()));
        assert_eq!(invalid["available"], false);
        assert_eq!(invalid["error_code"], "profile-cache-invalid");
        assert_ne!(invalid["error"], "");
    }

    #[test]
    fn status_cache_requires_safe_exact_profile_identity_and_invariants() {
        let status = json!({
            "schema_version": 1,
            "generated_at": 1,
            "product": "Mazzy VPN",
            "version": "1.4.1",
            "language": "en",
            "selected": true,
            "service_state": "active",
            "desired": "up",
            "mode": "normal",
            "tunnel_active": true,
            "internet": "up",
            "healthy": true,
            "protocol": "openvpn",
            "protocol_name": "OpenVPN",
            "profile": "AI workspace",
            "profile_id": "profile-0123456789abcdef0123456789abcdef",
            "profile_file_name": "opaque-profile.ovpn",
            "location": "Belgium — Brussels",
            "interface": "vpnovpn0",
            "handshake_age": null,
            "public_ip": "203.0.113.7",
            "autostart": true,
            "health_monitor": true,
            "fallback": {"active": false, "name": ""},
            "health_failures": 0,
            "profiles": {
                "amneziawg": 0,
                "wireguard": 0,
                "openvpn": 1,
                "l2tp": 0
            }
        });
        let encoded = serde_json::to_string(&status).expect("status cache JSON");
        let sanitized = sanitize_status_cache(&encoded).expect("sanitized status cache");
        assert_eq!(
            sanitized["profile_id"],
            "profile-0123456789abcdef0123456789abcdef"
        );

        let mut unknown_field = status.clone();
        unknown_field["endpoint"] = json!("vpn.invalid:1194");
        assert!(
            sanitize_status_cache(
                &serde_json::to_string(&unknown_field).expect("injected status JSON")
            )
            .is_err()
        );

        let mut mismatched_identity = status.clone();
        mismatched_identity["profile_file_name"] = Value::Null;
        assert!(
            sanitize_status_cache(
                &serde_json::to_string(&mismatched_identity).expect("partial identity JSON")
            )
            .is_err()
        );

        let mut false_health = status.clone();
        false_health["internet"] = json!("down");
        assert!(
            sanitize_status_cache(
                &serde_json::to_string(&false_health).expect("false health JSON")
            )
            .is_err()
        );

        let mut bidi_profile = status;
        bidi_profile["profile"] = json!("office\u{202e}conf");
        assert!(
            sanitize_status_cache(&serde_json::to_string(&bidi_profile).expect("bidi status JSON"))
                .is_err()
        );
    }

    #[test]
    fn compatibility_processes_have_bounded_deadlines_and_mutation_semantics() {
        assert_eq!(
            operation_deadline_seconds(&OperationRequest::Test {
                protocol: "openvpn".into(),
                profile: "Server".into(),
                timeout: 600,
            }),
            780
        );
        assert_eq!(
            operation_deadline_seconds(&OperationRequest::TestAll {
                protocol: "all".into(),
                timeout: 600,
            }),
            7_200
        );
        assert!(!operation_changes_system(&OperationRequest::Doctor {
            fix: false
        }));
        assert!(operation_changes_system(&OperationRequest::Doctor {
            fix: true
        }));
    }

    #[test]
    fn local_api_failures_keep_the_action_id_for_recovery() {
        let result = api_operation_result(
            "reconnect".into(),
            "action-01234567",
            json!({
                "status": "error",
                "error": {
                    "code": "internal-error",
                    "message_key": "api.audit.unavailable"
                }
            }),
        );
        assert!(!result.success);
        assert!(result.output.contains("action-01234567"));
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn local_api_parser_rejects_multiple_response_documents() {
        let request = json!({
            "api_version": "1.0",
            "request_id": "request-01234567"
        });
        let response = br#"{"api_version":"1.0","request_id":"request-01234567","status":"ok"}
{"api_version":"1.0","request_id":"request-01234567","status":"ok"}
"#;
        assert!(matches!(
            parse_local_api_response(response, &request),
            Err(LocalApiError::Indeterminate(_))
        ));
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn indeterminate_local_api_transport_retries_the_identical_request_once() {
        let mut attempts = 0;
        let result = retry_indeterminate(|| {
            attempts += 1;
            if attempts == 1 {
                Err(LocalApiError::Indeterminate("lost response".into()))
            } else {
                Ok(json!({"status": "ok"}))
            }
        });
        assert_eq!(attempts, 2);
        assert_eq!(result.expect("retry result")["status"], "ok");

        attempts = 0;
        let unavailable: Result<Value, LocalApiError> = retry_indeterminate(|| {
            attempts += 1;
            Err(LocalApiError::Unavailable)
        });
        assert!(matches!(unavailable, Err(LocalApiError::Unavailable)));
        assert_eq!(attempts, 1);
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn mutation_transport_reserves_time_for_bounded_rollback() {
        let request = json!({
            "deadline_ms": 30_000,
            "action_id": "action-01234567"
        });
        assert_eq!(
            local_api_response_timeout(&request),
            std::time::Duration::from_secs(90)
        );
        assert_eq!(
            local_api_response_timeout(&json!({"deadline_ms": 5_000})),
            std::time::Duration::from_secs(10)
        );
    }
}
