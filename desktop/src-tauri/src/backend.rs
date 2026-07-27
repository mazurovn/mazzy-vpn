// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: AGPL-3.0-or-later

use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::{
    env, fs,
    path::{Path, PathBuf},
    process::{Command, Output},
};
use tauri::{AppHandle, Manager};

const CLI_PATH: &str = "/usr/local/bin/mazzy-vpn";
const PROFILES_FILE: &str = "/run/mazzy-vpn/profiles.json";
const MAX_OUTPUT_BYTES: usize = 512 * 1024;

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
    pub installed_version: Option<String>,
    pub bundled_version: Option<String>,
    pub bundled_installer: bool,
    pub needs_install: bool,
    pub service_installed: bool,
    pub monitor_installed: bool,
    pub dependencies_ready: bool,
    pub missing_dependencies: usize,
    pub dependencies: Vec<DependencyState>,
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

fn command_result(action: String, result: Result<Output, std::io::Error>) -> OperationResult {
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

pub(crate) fn execute_operation(app: &AppHandle, request: OperationRequest) -> OperationResult {
    if matches!(request, OperationRequest::Bootstrap) {
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
            Command::new("pkexec")
                .arg("/bin/bash")
                .arg(installer)
                .arg("--yes")
                .arg("--skip-tests")
                .output(),
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
    if !Path::new(CLI_PATH).is_file() {
        return OperationResult {
            success: false,
            action,
            output: "Mazzy VPN engine is not installed. Open Settings and run Install / Repair."
                .into(),
            code: None,
        };
    }
    command_result(
        action,
        Command::new("pkexec").arg(CLI_PATH).args(args).output(),
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
    let output = Command::new(path).arg("version").output().ok()?;
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
pub fn get_installation_report(app: AppHandle) -> InstallationReport {
    let root = engine_root(&app);
    let installed_version = installed_version(Path::new(CLI_PATH));
    let bundled_version = bundled_version(&root);
    let engine_installed = installed_version.is_some();
    let dependencies = dependencies();
    let missing_dependencies = dependencies
        .iter()
        .filter(|dependency| !dependency.installed)
        .count();
    let dependencies_ready = missing_dependencies == 0;
    let needs_install =
        !engine_installed || installed_version != bundled_version || !dependencies_ready;
    InstallationReport {
        engine_installed,
        installed_version,
        bundled_version,
        bundled_installer: root.join("install.sh").is_file(),
        needs_install,
        service_installed: Path::new("/etc/systemd/system/vpnctl.service").is_file(),
        monitor_installed: Path::new("/etc/systemd/system/vpnctl-health.timer").is_file(),
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
    fn dependency_report_covers_the_cli_and_desktop_runtime() {
        let states = dependencies();
        for required in [
            "bash",
            "ip",
            "curl",
            "systemd",
            "journalctl",
            "pkexec",
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
}
