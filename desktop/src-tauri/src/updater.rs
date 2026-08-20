// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

use serde::Serialize;
use std::{sync::Mutex, time::Duration};
use tauri::{AppHandle, Emitter, State};
use tauri_plugin_opener::OpenerExt;
use tauri_plugin_updater::{Update, UpdaterExt};

const RELEASE_URL_PREFIX: &str = "https://github.com/mazurovn/mazzy-vpn/releases/tag/desktop-v";

#[derive(Default)]
pub struct PendingUpdate(Mutex<Option<Update>>);

#[derive(Clone, Debug, Serialize)]
pub struct UpdateInfo {
    current_version: String,
    version: String,
    install_supported: bool,
    installation_method: &'static str,
}

#[derive(Clone, Debug, Serialize)]
#[serde(tag = "event", rename_all = "kebab-case")]
enum UpdateProgress {
    Started { content_length: Option<u64> },
    Progress { chunk_length: usize },
    Finished,
    Installed,
}

fn can_install_in_app() -> bool {
    #[cfg(target_os = "linux")]
    {
        can_install_on_linux(std::env::var_os("APPIMAGE").as_deref())
    }
    #[cfg(not(target_os = "linux"))]
    {
        true
    }
}

#[cfg(target_os = "linux")]
fn can_install_on_linux(appimage: Option<&std::ffi::OsStr>) -> bool {
    appimage.is_some_and(|value| !value.is_empty())
}

fn safe_release_version(version: &str) -> bool {
    !version.is_empty()
        && version.len() <= 64
        && version
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || matches!(byte, b'.' | b'-' | b'+'))
}

fn release_url(version: &str) -> Result<String, String> {
    if !safe_release_version(version) {
        return Err("The update service returned an invalid release version".into());
    }
    Ok(format!("{RELEASE_URL_PREFIX}{version}"))
}

fn pending_lock(
    pending: &PendingUpdate,
) -> Result<std::sync::MutexGuard<'_, Option<Update>>, String> {
    pending
        .0
        .lock()
        .map_err(|_| "The pending update state is unavailable".into())
}

#[tauri::command]
pub async fn check_for_update(
    app: AppHandle,
    pending: State<'_, PendingUpdate>,
) -> Result<Option<UpdateInfo>, String> {
    log::info!(target: "mazzy_vpn_desktop::updater", "update.check_started");
    let update = app
        .updater_builder()
        .timeout(Duration::from_secs(15))
        .build()
        .map_err(|error| format!("Unable to initialize the signed updater: {error}"))?
        .check()
        .await
        .map_err(|error| format!("Unable to check for updates: {error}"))?;

    let info = update.as_ref().map(|update| {
        let install_supported = can_install_in_app();
        UpdateInfo {
            current_version: update.current_version.clone(),
            version: update.version.clone(),
            install_supported,
            installation_method: if install_supported {
                "signed-in-app"
            } else {
                "release-page"
            },
        }
    });
    log::info!(
        target: "mazzy_vpn_desktop::updater",
        "update.check_complete available={}",
        info.is_some()
    );
    *pending_lock(&pending)? = update;
    Ok(info)
}

#[tauri::command]
pub fn open_update_release(
    app: AppHandle,
    pending: State<'_, PendingUpdate>,
) -> Result<(), String> {
    let version = pending_lock(&pending)?
        .as_ref()
        .map(|update| update.version.clone())
        .ok_or_else(|| "There is no pending update".to_owned())?;
    app.opener()
        .open_url(release_url(&version)?, None::<&str>)
        .map_err(|error| format!("Unable to open the update release: {error}"))
}

#[tauri::command]
pub async fn install_update(
    app: AppHandle,
    pending: State<'_, PendingUpdate>,
) -> Result<(), String> {
    log::info!(target: "mazzy_vpn_desktop::updater", "update.install_started");
    if !can_install_in_app() {
        return Err(
            "This installation is managed by DEB/RPM; install the package from the release page"
                .into(),
        );
    }
    let update = pending_lock(&pending)?
        .as_ref()
        .cloned()
        .ok_or_else(|| "There is no pending update".to_owned())?;
    let update_version = update.version.clone();
    let progress_app = app.clone();
    let finished_app = app.clone();
    let mut started = false;
    if let Err(error) = update
        .download_and_install(
            move |chunk_length, content_length| {
                if !started {
                    let _ = progress_app.emit(
                        "update-progress",
                        UpdateProgress::Started { content_length },
                    );
                    started = true;
                }
                let _ =
                    progress_app.emit("update-progress", UpdateProgress::Progress { chunk_length });
            },
            move || {
                let _ = finished_app.emit("update-progress", UpdateProgress::Finished);
            },
        )
        .await
    {
        log::warn!(target: "mazzy_vpn_desktop::updater", "update.install_failed");
        return Err(format!("The signed update could not be installed: {error}"));
    }
    let mut pending_update = pending_lock(&pending)?;
    if pending_update
        .as_ref()
        .is_some_and(|candidate| candidate.version == update_version)
    {
        *pending_update = None;
    }
    let _ = app.emit("update-progress", UpdateProgress::Installed);
    log::info!(target: "mazzy_vpn_desktop::updater", "update.install_complete");
    Ok(())
}

#[tauri::command]
pub fn restart_application(app: AppHandle) {
    app.restart();
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn release_urls_are_fixed_to_the_project_repository() {
        assert_eq!(
            release_url("0.4.2").expect("release URL"),
            "https://github.com/mazurovn/mazzy-vpn/releases/tag/desktop-v0.4.2"
        );
        assert!(release_url("0.4.2/../../malicious").is_err());
        assert!(release_url("0.4.2?redirect=https://example.com").is_err());
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn package_managed_linux_install_cannot_self_update() {
        use std::ffi::OsStr;

        assert!(!can_install_on_linux(None));
        assert!(!can_install_on_linux(Some(OsStr::new(""))));
        assert!(can_install_on_linux(Some(OsStr::new(
            "/tmp/Mazzy VPN Desktop.AppImage"
        ))));
    }
}
