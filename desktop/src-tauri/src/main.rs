// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: AGPL-3.0-or-later
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod backend;

use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::fs;
use std::process::{Command, Output};
use tauri::{
    AppHandle, Emitter, Manager,
    menu::{Menu, MenuItem, PredefinedMenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
};

const STATUS_FILE: &str = "/run/mazzy-vpn/status.json";
const API_MANIFEST: &str = include_str!("../../../api/v1/manifest.json");

#[derive(Clone, Copy, Debug, Deserialize)]
#[serde(rename_all = "kebab-case")]
enum VpnAction {
    Quick,
    Reconnect,
    Disconnect,
    Verify,
    ProbeAll,
    Doctor,
    Refresh,
    AutostartOn,
    AutostartOff,
    MonitorOn,
    MonitorOff,
}

#[derive(Clone, Debug, Serialize)]
struct ActionResult {
    success: bool,
    action: &'static str,
    output: String,
    code: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    data: Option<Value>,
}

#[derive(Clone, Debug, Serialize)]
struct PlatformInfo {
    os: &'static str,
    functional: bool,
    note: &'static str,
    desktop_version: &'static str,
    author: &'static str,
    license: &'static str,
}

fn action_spec(action: VpnAction) -> (&'static str, &'static [&'static str]) {
    match action {
        VpnAction::Quick => ("quick", &["quick"]),
        VpnAction::Reconnect => ("reconnect", &["reconnect"]),
        VpnAction::Disconnect => ("disconnect", &["disconnect"]),
        VpnAction::Verify => ("verify", &["verify", "--timeout", "10"]),
        VpnAction::ProbeAll => (
            "probe-all",
            &["probe", "all", "--timeout", "3", "--jobs", "4"],
        ),
        VpnAction::Doctor => ("doctor", &["doctor"]),
        VpnAction::Refresh => ("refresh", &["_refresh-dashboard-cache"]),
        VpnAction::AutostartOn => ("autostart-on", &["autostart", "on"]),
        VpnAction::AutostartOff => ("autostart-off", &["autostart", "off"]),
        VpnAction::MonitorOn => ("monitor-on", &["monitor", "on"]),
        VpnAction::MonitorOff => ("monitor-off", &["monitor", "off"]),
    }
}

fn output_text(output: &Output) -> String {
    let mut text = String::from_utf8_lossy(&output.stdout).trim().to_owned();
    let stderr = String::from_utf8_lossy(&output.stderr).trim().to_owned();
    if !stderr.is_empty() {
        if !text.is_empty() {
            text.push('\n');
        }
        text.push_str(&stderr);
    }
    text
}

#[cfg(target_os = "linux")]
fn execute_action(action: VpnAction) -> ActionResult {
    let (name, args) = action_spec(action);
    let Some(cli_path) = backend::installed_cli_path() else {
        return ActionResult {
            success: false,
            action: name,
            output: "Mazzy VPN engine is not installed. Open Settings and run Install / Repair."
                .into(),
            code: None,
            data: None,
        };
    };
    let result = backend::bounded_output(
        Command::new(backend::TIMEOUT_PATH)
            .args([
                "--foreground",
                "--kill-after=30s",
                "900s",
                backend::PKEXEC_PATH,
            ])
            .arg(cli_path)
            .args(args),
    );
    match result {
        Ok(output) => ActionResult {
            success: output.status.success(),
            action: name,
            output: output_text(&output),
            code: output.status.code(),
            data: None,
        },
        Err(error) => ActionResult {
            success: false,
            action: name,
            output: format!("Unable to start pkexec: {error}"),
            code: None,
            data: None,
        },
    }
}

#[cfg(not(target_os = "linux"))]
fn execute_action(action: VpnAction) -> ActionResult {
    let (name, _) = action_spec(action);
    ActionResult {
        success: false,
        action: name,
        output: "Preview build: the native VPN backend is not implemented on this OS yet.".into(),
        code: None,
        data: None,
    }
}

fn fallback_status(error: impl ToString) -> Value {
    json!({
        "schema_version": 1,
        "generated_at": 0,
        "product": "Mazzy VPN",
        "version": env!("CARGO_PKG_VERSION"),
        "available": false,
        "platform_preview": !cfg!(target_os = "linux"),
        "error": error.to_string(),
        "profiles": {
            "amneziawg": 0,
            "wireguard": 0,
            "openvpn": 0,
            "l2tp": 0
        }
    })
}

#[cfg(target_os = "linux")]
fn read_status() -> Value {
    let data = match fs::read_to_string(STATUS_FILE) {
        Ok(data) => data,
        Err(cache_error) => {
            let Some(cli_path) = backend::installed_cli_path() else {
                return fallback_status(format!(
                    "{cache_error}; Mazzy VPN engine is not installed"
                ));
            };
            let output = backend::bounded_output(
                Command::new(backend::TIMEOUT_PATH)
                    .args(["--foreground", "--kill-after=2s", "15s"])
                    .arg(cli_path)
                    .args(["status", "--json"]),
            );
            return match output {
                Ok(output) if output.status.success() => String::from_utf8(output.stdout)
                    .map_err(|_| "status command returned non-UTF-8 data".to_owned())
                    .and_then(|contents| backend::sanitize_status_cache(&contents))
                    .unwrap_or_else(fallback_status),
                Ok(output) => fallback_status(output_text(&output)),
                Err(cli_error) => fallback_status(format!("{cache_error}; {cli_error}")),
            };
        }
    };
    backend::sanitize_status_cache(&data).unwrap_or_else(fallback_status)
}

#[cfg(not(target_os = "linux"))]
fn read_status() -> Value {
    fallback_status("Native VPN backend is not implemented on this OS yet")
}

#[tauri::command]
fn get_status() -> Value {
    read_status()
}

fn api_manifest() -> Result<Value, String> {
    serde_json::from_str(API_MANIFEST)
        .map_err(|error| format!("Embedded API contract is invalid: {error}"))
}

#[tauri::command]
fn get_api_info() -> Result<Value, String> {
    api_manifest()
}

#[tauri::command]
async fn run_action(app: AppHandle, action: VpnAction) -> Result<ActionResult, String> {
    tauri::async_runtime::spawn_blocking(move || execute_tray_action(&app, action))
        .await
        .map_err(|error| error.to_string())
}

fn show_window(app: &AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.show();
        let _ = window.unminimize();
        let _ = window.set_focus();
    }
}

fn show_page(app: &AppHandle, page: &'static str) {
    show_window(app);
    let _ = app.emit("navigate-page", page);
}

#[tauri::command]
fn show_main_window(app: AppHandle) {
    show_window(&app);
}

#[tauri::command]
fn hide_main_window(app: AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.hide();
    }
}

#[tauri::command]
fn get_platform_info() -> PlatformInfo {
    platform_info()
}

fn execute_tray_action(app: &AppHandle, action: VpnAction) -> ActionResult {
    if matches!(action, VpnAction::Verify) {
        return match backend::verify_connection_sync(10, false) {
            Ok(result) => {
                let verdict = result
                    .get("verdict")
                    .and_then(Value::as_str)
                    .unwrap_or("failed");
                let country = result
                    .pointer("/geo/observed_country_code")
                    .and_then(Value::as_str)
                    .unwrap_or("unknown");
                let dns = result
                    .pointer("/dns/state")
                    .and_then(Value::as_str)
                    .unwrap_or("unknown");
                let findings = result
                    .get("findings")
                    .and_then(Value::as_array)
                    .map_or(0, Vec::len);
                ActionResult {
                    success: verdict != "failed",
                    action: "verify",
                    output: format!(
                        "verdict={verdict}; observed_country={country}; dns={dns}; findings={findings}"
                    ),
                    code: None,
                    data: Some(result),
                }
            }
            Err(error) => ActionResult {
                success: false,
                action: "verify",
                output: error,
                code: None,
                data: None,
            },
        };
    }
    if matches!(action, VpnAction::ProbeAll) {
        return match backend::probe_profiles_sync("all".into(), 3, 4) {
            Ok(result) => {
                let total = result
                    .pointer("/summary/total")
                    .and_then(Value::as_u64)
                    .unwrap_or(0);
                let reachable = result
                    .pointer("/summary/reachable")
                    .and_then(Value::as_u64)
                    .unwrap_or(0);
                let unknown = result
                    .pointer("/summary/unknown")
                    .and_then(Value::as_u64)
                    .unwrap_or(0);
                let hard_failures = ["unreachable", "invalid"]
                    .iter()
                    .filter_map(|field| {
                        result
                            .pointer(&format!("/summary/{field}"))
                            .and_then(Value::as_u64)
                    })
                    .sum::<u64>();
                ActionResult {
                    success: total > 0 && hard_failures == 0,
                    action: "probe-all",
                    output: format!(
                        "total={total}; reachable={reachable}; unknown={unknown}; hard_failures={hard_failures}"
                    ),
                    code: None,
                    data: Some(result),
                }
            }
            Err(error) => ActionResult {
                success: false,
                action: "probe-all",
                output: error,
                code: None,
                data: None,
            },
        };
    }
    let operation = match action {
        VpnAction::Quick => Some(backend::OperationRequest::Quick),
        VpnAction::Reconnect => Some(backend::OperationRequest::Reconnect),
        VpnAction::Disconnect => Some(backend::OperationRequest::Disconnect),
        VpnAction::Doctor => Some(backend::OperationRequest::Doctor { fix: false }),
        VpnAction::Refresh => Some(backend::OperationRequest::Refresh),
        VpnAction::AutostartOn => Some(backend::OperationRequest::Autostart { enabled: true }),
        VpnAction::AutostartOff => Some(backend::OperationRequest::Autostart { enabled: false }),
        VpnAction::MonitorOn => Some(backend::OperationRequest::Monitor { enabled: true }),
        VpnAction::MonitorOff => Some(backend::OperationRequest::Monitor { enabled: false }),
        _ => None,
    };
    if let Some(operation) = operation {
        let result = backend::execute_operation(app, operation);
        return ActionResult {
            success: result.success,
            action: action_spec(action).0,
            output: result.output,
            code: result.code,
            data: None,
        };
    }
    execute_action(action)
}

fn launch_tray_action(app: AppHandle, action: VpnAction) {
    tauri::async_runtime::spawn(async move {
        let operation_app = app.clone();
        let result = match tauri::async_runtime::spawn_blocking(move || {
            execute_tray_action(&operation_app, action)
        })
        .await
        {
            Ok(result) => result,
            Err(error) => ActionResult {
                success: false,
                action: action_spec(action).0,
                output: error.to_string(),
                code: None,
                data: None,
            },
        };
        let _ = app.emit("vpn-action-result", result);
    });
}

fn platform_info() -> PlatformInfo {
    PlatformInfo {
        os: std::env::consts::OS,
        functional: cfg!(target_os = "linux"),
        desktop_version: env!("CARGO_PKG_VERSION"),
        author: env!("CARGO_PKG_AUTHORS"),
        license: env!("CARGO_PKG_LICENSE"),
        note: if cfg!(target_os = "linux") {
            "Linux CLI backend is available"
        } else {
            "Desktop preview; native VPN backend is not implemented"
        },
    }
}

fn main() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            get_status,
            get_api_info,
            run_action,
            backend::get_profiles,
            backend::probe_profiles,
            backend::verify_connection,
            backend::get_installation_report,
            backend::run_operation,
            backend::pick_profile_files,
            backend::pick_profile_folder,
            show_main_window,
            hide_main_window,
            get_platform_info
        ])
        .setup(|app| {
            let description = MenuItem::with_id(
                app,
                "description",
                "AI-ready VPN · recovery and real egress checks",
                false,
                None::<&str>,
            )?;
            let open = MenuItem::with_id(app, "open", "Open Dashboard", true, None::<&str>)?;
            let profiles =
                MenuItem::with_id(app, "profiles", "Profiles & Locations", true, None::<&str>)?;
            let diagnostics = MenuItem::with_id(
                app,
                "diagnostics",
                "Diagnostics & Events",
                true,
                None::<&str>,
            )?;
            let settings =
                MenuItem::with_id(app, "settings", "Services & Settings", true, None::<&str>)?;
            let about = MenuItem::with_id(app, "about", "About Mazzy VPN", true, None::<&str>)?;
            let quick = MenuItem::with_id(app, "quick", "Quick Connect", true, None::<&str>)?;
            let reconnect = MenuItem::with_id(app, "reconnect", "Reconnect", true, None::<&str>)?;
            let disconnect =
                MenuItem::with_id(app, "disconnect", "Disconnect", true, None::<&str>)?;
            let verify = MenuItem::with_id(app, "verify", "Verify VPN Egress", true, None::<&str>)?;
            let probe_all =
                MenuItem::with_id(app, "probe-all", "Ping All Locations", true, None::<&str>)?;
            let refresh = MenuItem::with_id(app, "refresh", "Refresh Status", true, None::<&str>)?;
            let doctor = MenuItem::with_id(app, "doctor", "Self-diagnostics", true, None::<&str>)?;
            let autostart_on = MenuItem::with_id(
                app,
                "autostart-on",
                "Enable Auto-connect",
                true,
                None::<&str>,
            )?;
            let autostart_off = MenuItem::with_id(
                app,
                "autostart-off",
                "Disable Auto-connect",
                true,
                None::<&str>,
            )?;
            let monitor_on = MenuItem::with_id(
                app,
                "monitor-on",
                "Enable Health Monitor",
                true,
                None::<&str>,
            )?;
            let monitor_off = MenuItem::with_id(
                app,
                "monitor-off",
                "Disable Health Monitor",
                true,
                None::<&str>,
            )?;
            let separator_one = PredefinedMenuItem::separator(app)?;
            let separator_two = PredefinedMenuItem::separator(app)?;
            let separator_three = PredefinedMenuItem::separator(app)?;
            let separator_four = PredefinedMenuItem::separator(app)?;
            let quit = MenuItem::with_id(app, "quit", "Quit Mazzy VPN", true, None::<&str>)?;
            let menu = Menu::with_items(
                app,
                &[
                    &description,
                    &open,
                    &profiles,
                    &diagnostics,
                    &settings,
                    &about,
                    &separator_one,
                    &quick,
                    &reconnect,
                    &disconnect,
                    &separator_two,
                    &verify,
                    &probe_all,
                    &refresh,
                    &doctor,
                    &separator_three,
                    &autostart_on,
                    &autostart_off,
                    &monitor_on,
                    &monitor_off,
                    &separator_four,
                    &quit,
                ],
            )?;

            TrayIconBuilder::with_id("mazzy-main")
                .icon(app.default_window_icon().expect("window icon").clone())
                .tooltip("Mazzy VPN")
                .menu(&menu)
                .show_menu_on_left_click(false)
                .on_menu_event(|app, event| match event.id.as_ref() {
                    "open" => show_page(app, "dashboard"),
                    "profiles" => show_page(app, "profiles"),
                    "diagnostics" => show_page(app, "diagnostics"),
                    "settings" => show_page(app, "settings"),
                    "about" => show_page(app, "about"),
                    "quick" => launch_tray_action(app.clone(), VpnAction::Quick),
                    "reconnect" => launch_tray_action(app.clone(), VpnAction::Reconnect),
                    "disconnect" => launch_tray_action(app.clone(), VpnAction::Disconnect),
                    "verify" => launch_tray_action(app.clone(), VpnAction::Verify),
                    "probe-all" => launch_tray_action(app.clone(), VpnAction::ProbeAll),
                    "refresh" => launch_tray_action(app.clone(), VpnAction::Refresh),
                    "doctor" => launch_tray_action(app.clone(), VpnAction::Doctor),
                    "autostart-on" => launch_tray_action(app.clone(), VpnAction::AutostartOn),
                    "autostart-off" => launch_tray_action(app.clone(), VpnAction::AutostartOff),
                    "monitor-on" => launch_tray_action(app.clone(), VpnAction::MonitorOn),
                    "monitor-off" => launch_tray_action(app.clone(), VpnAction::MonitorOff),
                    "quit" => app.exit(0),
                    _ => {}
                })
                .on_tray_icon_event(|tray, event| {
                    if let TrayIconEvent::Click {
                        button: MouseButton::Left,
                        button_state: MouseButtonState::Up,
                        ..
                    } = event
                    {
                        show_window(tray.app_handle());
                    }
                })
                .build(app)?;

            Ok(())
        })
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                api.prevent_close();
                let _ = window.hide();
            }
        })
        .run(tauri::generate_context!())
        .expect("error while running Mazzy VPN Desktop");
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn actions_map_only_to_fixed_cli_arguments() {
        assert_eq!(action_spec(VpnAction::Quick), ("quick", &["quick"][..]));
        assert_eq!(
            action_spec(VpnAction::Refresh),
            ("refresh", &["_refresh-dashboard-cache"][..])
        );
    }

    #[test]
    fn fallback_status_is_valid_and_marks_unavailable() {
        let status = fallback_status("test");
        assert_eq!(status["available"], false);
        assert_eq!(status["schema_version"], 1);
    }

    #[test]
    fn about_metadata_comes_from_the_package_manifest() {
        let info = platform_info();
        assert_eq!(info.desktop_version, env!("CARGO_PKG_VERSION"));
        assert!(info.author.contains("Nik m"));
        assert_eq!(info.license, "AGPL-3.0-or-later");
    }

    #[test]
    fn embedded_api_contract_is_frontend_readable() {
        let manifest = api_manifest().expect("embedded API manifest");
        assert_eq!(manifest["api_version"], "1.0");
        assert_eq!(manifest["status"], "foundation");
        let transports = manifest["transports"]
            .as_array()
            .expect("transport registry");
        let protected = transports
            .iter()
            .find(|transport| transport["id"] == "protected-local-service")
            .expect("protected service transport");
        assert_eq!(protected["status"], "partial");
    }
}
