// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: AGPL-3.0-or-later

use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
#[cfg(target_os = "linux")]
use std::io::Write;
#[cfg(target_os = "linux")]
use std::os::unix::net::UnixStream;
use std::{
    env, fs,
    io::{self, Read},
    path::{Path, PathBuf},
    process::{Command, Output, Stdio},
    sync::atomic::{AtomicU64, Ordering},
    thread,
    time::{SystemTime, UNIX_EPOCH},
};
use tauri::{AppHandle, Manager};

const SYSTEM_CLI_PATH: &str = "/usr/bin/mazzy-vpn";
const LOCAL_CLI_PATH: &str = "/usr/local/bin/mazzy-vpn";
const PROFILES_FILE: &str = "/run/mazzy-vpn/profiles.json";
const API_SOCKET: &str = "/run/mazzy-vpn/api-v1.sock";
const MAX_OUTPUT_BYTES: usize = 512 * 1024;
const MAX_OUTPUT_STREAM_BYTES: usize = MAX_OUTPUT_BYTES / 2;
const TRUNCATED_OUTPUT_MARKER: &[u8] = b"\n[output truncated]\n";
#[cfg(target_os = "linux")]
const MAX_API_RESPONSE_BYTES: usize = 64 * 1024;
#[cfg(target_os = "linux")]
const LOCAL_API_COMPLETION_GRACE_MS: u64 = 60_000;
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

fn select_cli_path(system_installed: bool, local_installed: bool) -> Option<&'static Path> {
    if system_installed {
        Some(Path::new(SYSTEM_CLI_PATH))
    } else if local_installed {
        Some(Path::new(LOCAL_CLI_PATH))
    } else {
        None
    }
}

fn installed_cli_path() -> Option<&'static Path> {
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
        && !value.chars().any(char::is_control);
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

fn clean_output(output: &Output) -> String {
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
    cleaned.trim().to_owned()
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

fn probe_profiles_sync(
    selected_protocol: String,
    timeout_seconds: u16,
    concurrency: u8,
) -> Result<Value, String> {
    let selected_protocol = protocol(&selected_protocol, true)?;
    timeout(timeout_seconds, 1, 30)?;
    if !(1..=8).contains(&concurrency) {
        return Err("Probe concurrency must be between 1 and 8".to_owned());
    }

    #[cfg(target_os = "linux")]
    if Path::new(API_SOCKET).exists() {
        let request = probe_api_request(
            &selected_protocol,
            timeout_seconds,
            concurrency,
            cached_profile_count(&selected_protocol),
        );
        match send_local_api_with_retry(&request) {
            Ok(response) => return probe_result_from_response(response),
            Err(LocalApiError::Unavailable) => {}
            Err(LocalApiError::Indeterminate(error)) => {
                return Err(format!(
                    "The local API probe outcome is unknown; the request was not repeated \
                     through another privilege path: {error}"
                ));
            }
        }
    }

    let cli_path = installed_cli_path().ok_or_else(|| {
        "Mazzy VPN engine is not installed. Open Settings and run Install / Repair.".to_owned()
    })?;
    let mut command = Command::new("pkexec");
    command
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
        if Path::new(SYSTEM_CLI_PATH).is_file() {
            return command_result(
                "bootstrap".into(),
                bounded_output(
                    Command::new("pkexec")
                        .arg(SYSTEM_CLI_PATH)
                        .args(["doctor", "--fix"]),
                ),
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
        return command_result(
            "bootstrap".into(),
            bounded_output(
                Command::new("pkexec")
                    .arg("/bin/bash")
                    .arg(installer)
                    .arg("--yes")
                    .arg("--skip-tests"),
            ),
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
    if let Some(result) = try_execute_local_api(&request) {
        return result;
    }
    command_result(
        action,
        bounded_output(Command::new("pkexec").arg(cli_path).args(args)),
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
    let output = bounded_output(Command::new(path).arg("version")).ok()?;
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
    fs::read_to_string(PROFILES_FILE)
        .ok()
        .and_then(|contents| serde_json::from_str(&contents).ok())
        .unwrap_or_else(|| {
            json!({
                "schema_version": 1,
                "generated_at": 0,
                "available": false,
                "error": "Profile cache is not ready. Refresh the engine cache.",
                "profiles": []
            })
        })
}

#[tauri::command]
pub async fn probe_profiles(
    protocol: String,
    timeout: u16,
    concurrency: u8,
) -> Result<Value, String> {
    tauri::async_runtime::spawn_blocking(move || {
        probe_profiles_sync(protocol, timeout, concurrency)
    })
    .await
    .map_err(|error| error.to_string())?
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
    tauri::async_runtime::spawn_blocking(move || execute_operation(&app, request))
        .await
        .map_err(|error| error.to_string())
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
    fn child_output_capture_is_bounded_and_marks_truncation() {
        let input = vec![b'x'; MAX_OUTPUT_STREAM_BYTES + 1];
        let output =
            drain_bounded(std::io::Cursor::new(input), MAX_OUTPUT_STREAM_BYTES).expect("capture");
        assert_eq!(output.len(), MAX_OUTPUT_STREAM_BYTES);
        assert!(output.ends_with(TRUNCATED_OUTPUT_MARKER));
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
