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
const CLI_PATH: &str = "/usr/local/bin/mazzy-vpn";

#[derive(Clone, Copy, Debug, Deserialize)]
#[serde(rename_all = "kebab-case")]
enum VpnAction {
    Quick,
    Reconnect,
    Disconnect,
    Doctor,
    Refresh,
}

#[derive(Clone, Debug, Serialize)]
struct ActionResult {
    success: bool,
    action: &'static str,
    output: String,
    code: Option<i32>,
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
        VpnAction::Doctor => ("doctor", &["doctor"]),
        VpnAction::Refresh => ("refresh", &["_refresh-dashboard-cache"]),
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
    let result = Command::new("pkexec").arg(CLI_PATH).args(args).output();
    match result {
        Ok(output) => ActionResult {
            success: output.status.success(),
            action: name,
            output: output_text(&output),
            code: output.status.code(),
        },
        Err(error) => ActionResult {
            success: false,
            action: name,
            output: format!("Unable to start pkexec: {error}"),
            code: None,
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
            let output = Command::new(CLI_PATH).args(["status", "--json"]).output();
            return match output {
                Ok(output) if output.status.success() => {
                    serde_json::from_slice(&output.stdout).unwrap_or_else(fallback_status)
                }
                Ok(output) => fallback_status(output_text(&output)),
                Err(cli_error) => fallback_status(format!("{cache_error}; {cli_error}")),
            };
        }
    };
    serde_json::from_str(&data).unwrap_or_else(fallback_status)
}

#[cfg(not(target_os = "linux"))]
fn read_status() -> Value {
    fallback_status("Native VPN backend is not implemented on this OS yet")
}

#[tauri::command]
fn get_status() -> Value {
    read_status()
}

#[tauri::command]
async fn run_action(action: VpnAction) -> Result<ActionResult, String> {
    tauri::async_runtime::spawn_blocking(move || execute_action(action))
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

fn launch_tray_action(app: AppHandle, action: VpnAction) {
    tauri::async_runtime::spawn(async move {
        let result =
            match tauri::async_runtime::spawn_blocking(move || execute_action(action)).await {
                Ok(result) => result,
                Err(error) => ActionResult {
                    success: false,
                    action: action_spec(action).0,
                    output: error.to_string(),
                    code: None,
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
            run_action,
            backend::get_profiles,
            backend::get_installation_report,
            backend::run_operation,
            backend::pick_profile_files,
            backend::pick_profile_folder,
            show_main_window,
            hide_main_window,
            get_platform_info
        ])
        .setup(|app| {
            let open = MenuItem::with_id(app, "open", "Open Dashboard", true, None::<&str>)?;
            let quick = MenuItem::with_id(app, "quick", "Quick Connect", true, None::<&str>)?;
            let reconnect = MenuItem::with_id(app, "reconnect", "Reconnect", true, None::<&str>)?;
            let disconnect =
                MenuItem::with_id(app, "disconnect", "Disconnect", true, None::<&str>)?;
            let refresh = MenuItem::with_id(app, "refresh", "Refresh Status", true, None::<&str>)?;
            let doctor = MenuItem::with_id(app, "doctor", "Self-diagnostics", true, None::<&str>)?;
            let separator = PredefinedMenuItem::separator(app)?;
            let quit = MenuItem::with_id(app, "quit", "Quit Mazzy VPN", true, None::<&str>)?;
            let menu = Menu::with_items(
                app,
                &[
                    &open,
                    &separator,
                    &quick,
                    &reconnect,
                    &disconnect,
                    &refresh,
                    &doctor,
                    &quit,
                ],
            )?;

            TrayIconBuilder::with_id("mazzy-main")
                .icon(app.default_window_icon().expect("window icon").clone())
                .tooltip("Mazzy VPN")
                .menu(&menu)
                .show_menu_on_left_click(false)
                .on_menu_event(|app, event| match event.id.as_ref() {
                    "open" => show_window(app),
                    "quick" => launch_tray_action(app.clone(), VpnAction::Quick),
                    "reconnect" => launch_tray_action(app.clone(), VpnAction::Reconnect),
                    "disconnect" => launch_tray_action(app.clone(), VpnAction::Disconnect),
                    "refresh" => launch_tray_action(app.clone(), VpnAction::Refresh),
                    "doctor" => launch_tray_action(app.clone(), VpnAction::Doctor),
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
}
