// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: AGPL-3.0-or-later
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod agent_control;
mod backend;
mod updater;

use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::process::Output;
#[cfg(target_os = "linux")]
use std::{fs, process::Command};
use tauri::{
    AppHandle, Emitter, Manager,
    menu::{Menu, MenuItem, PredefinedMenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
};

#[cfg(target_os = "linux")]
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

fn action_name(action: VpnAction) -> &'static str {
    match action {
        VpnAction::Quick => "quick",
        VpnAction::Reconnect => "reconnect",
        VpnAction::Disconnect => "disconnect",
        VpnAction::Verify => "verify",
        VpnAction::ProbeAll => "probe-all",
        VpnAction::Doctor => "doctor",
        VpnAction::Refresh => "refresh",
        VpnAction::AutostartOn => "autostart-on",
        VpnAction::AutostartOff => "autostart-off",
        VpnAction::MonitorOn => "monitor-on",
        VpnAction::MonitorOff => "monitor-off",
    }
}

fn output_text(output: &Output) -> String {
    backend::clean_output(output)
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
            // The socket-activated engine is the Desktop's primary backend.
            // A successful query also recreates the volatile /run cache after
            // reboot, without requiring any separately invoked CLI process.
            if backend::refresh_status_cache_from_api() {
                if let Ok(contents) = fs::read_to_string(STATUS_FILE) {
                    return backend::sanitize_status_cache(&contents)
                        .unwrap_or_else(fallback_status);
                }
            }
            let Some(cli_path) = backend::installed_cli_path() else {
                return fallback_status(format!(
                    "{cache_error}; Mazzy VPN engine is not installed"
                ));
            };
            let output = backend::bounded_output(
                Command::new(backend::TIMEOUT_PATH)
                    .args(["--kill-after=2s", "15s"])
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
    let result = tauri::async_runtime::spawn_blocking(move || execute_tray_action(&app, action))
        .await
        .map_err(|error| error.to_string());
    match &result {
        Ok(operation) => log::info!(
            target: "mazzy_vpn_desktop::tray",
            "tray.complete action={} success={}",
            operation.action,
            operation.success
        ),
        Err(_) => log::error!(target: "mazzy_vpn_desktop::tray", "tray.worker_failed"),
    }
    result
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
        VpnAction::Quick => backend::OperationRequest::Quick,
        VpnAction::Reconnect => backend::OperationRequest::Reconnect,
        VpnAction::Disconnect => backend::OperationRequest::Disconnect,
        VpnAction::Doctor => backend::OperationRequest::Doctor { fix: false },
        VpnAction::Refresh => backend::OperationRequest::Refresh,
        VpnAction::AutostartOn => backend::OperationRequest::Autostart { enabled: true },
        VpnAction::AutostartOff => backend::OperationRequest::Autostart { enabled: false },
        VpnAction::MonitorOn => backend::OperationRequest::Monitor { enabled: true },
        VpnAction::MonitorOff => backend::OperationRequest::Monitor { enabled: false },
        VpnAction::Verify | VpnAction::ProbeAll => unreachable!("handled above"),
    };
    let result = backend::execute_operation(app, operation);
    ActionResult {
        success: result.success,
        action: action_name(action),
        output: result.output,
        code: result.code,
        data: None,
    }
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
                action: action_name(action),
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

#[cfg(target_os = "linux")]
fn configure_linux_desktop_environment() {
    if std::env::var_os("NO_AT_BRIDGE").is_none() {
        // Some distributions ship an accessibility bridge that can segfault
        // during GTK startup. The UI remains usable without that bridge.
        unsafe {
            std::env::set_var("NO_AT_BRIDGE", "1");
        }
    }
}

fn main() {
    #[cfg(target_os = "linux")]
    configure_linux_desktop_environment();

    tauri::Builder::default()
        .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
            log::info!(target: "mazzy_vpn_desktop::lifecycle", "single_instance.focus_existing");
            show_window(app);
        }))
        .plugin(
            tauri_plugin_log::Builder::new()
                .level(log::LevelFilter::Info)
                .max_file_size(1_000_000)
                .rotation_strategy(tauri_plugin_log::RotationStrategy::KeepOne)
                .build(),
        )
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .manage(updater::PendingUpdate::default())
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
            agent_control::get_agent_integrations,
            show_main_window,
            hide_main_window,
            get_platform_info,
            updater::check_for_update,
            updater::open_update_release,
            updater::install_update,
            updater::restart_application
        ])
        .setup(|app| {
            log::info!(
                target: "mazzy_vpn_desktop::lifecycle",
                "desktop.start version={} os={}",
                env!("CARGO_PKG_VERSION"),
                std::env::consts::OS
            );
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
            let agents = MenuItem::with_id(app, "agents", "AI Agents", true, None::<&str>)?;
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
                    &agents,
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
                    "agents" => show_page(app, "agents"),
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
    fn tray_actions_have_stable_event_names() {
        assert_eq!(action_name(VpnAction::Quick), "quick");
        assert_eq!(action_name(VpnAction::Refresh), "refresh");
    }

    #[test]
    fn fallback_status_is_valid_and_marks_unavailable() {
        let status = fallback_status("test");
        assert_eq!(status["available"], false);
        assert_eq!(status["schema_version"], 1);
    }

    #[test]
    fn tray_output_is_sanitized_before_frontend_events() {
        let output = std::process::Command::new("/usr/bin/printf")
            .args(["\\033[31mFAIL\\033[0m\\n"])
            .output()
            .expect("printf");
        assert_eq!(output_text(&output), "FAIL");
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
