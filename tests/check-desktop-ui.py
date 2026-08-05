#!/usr/bin/env python3
"""Fail closed when Desktop HTML, JavaScript, Rust and versions drift apart."""

from __future__ import annotations

import json
import os
import re
import subprocess
import struct
import sys
import tempfile
from collections import Counter
from html.parser import HTMLParser
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
HTML = ROOT / "desktop/ui/index.html"
JS = ROOT / "desktop/ui/app.js"
RUST = ROOT / "desktop/src-tauri/src/backend.rs"
MAIN = ROOT / "desktop/src-tauri/src/main.rs"
AGENT = ROOT / "desktop/src-tauri/src/agent_control.rs"
UPDATER = ROOT / "desktop/src-tauri/src/updater.rs"
CARGO_MANIFEST = ROOT / "desktop/src-tauri/Cargo.toml"
TAURI_CONFIG = ROOT / "desktop/src-tauri/tauri.conf.json"
CAPABILITY = ROOT / "desktop/src-tauri/capabilities/default.json"
BUILD_RELEASE = ROOT / "desktop/scripts/build-release.mjs"
DESKTOP_WORKFLOW = ROOT / ".github/workflows/desktop.yml"
UPDATER_SIGNATURE_AUDIT = ROOT / "tests/prepare-updater-signature-audit.py"


def fail(message: str) -> None:
    raise AssertionError(message)


class UiParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.ids: list[str] = []
        self.i18n: set[str] = set()

    def handle_starttag(
        self, tag: str, attrs: list[tuple[str, str | None]]
    ) -> None:
        values = dict(attrs)
        if values.get("id"):
            self.ids.append(values["id"] or "")
        if values.get("data-i18n"):
            self.i18n.add(values["data-i18n"] or "")
        if values.get("data-i18n-placeholder"):
            self.i18n.add(values["data-i18n-placeholder"] or "")


def translations(javascript: str) -> dict[str, dict[str, str]]:
    prefix = javascript.split("const $ =", 1)[0]
    script = (
        "global.window = {};\n"
        f"{prefix}\nprocess.stdout.write(JSON.stringify(translations));"
    )
    result = subprocess.run(
        ["node"],
        check=True,
        capture_output=True,
        input=script,
        text=True,
        encoding="utf-8",
        errors="strict",
        timeout=10,
    )
    value = json.loads(result.stdout)
    if not isinstance(value, dict):
        fail("Desktop translations are not an object")
    return value


def desktop_version() -> str:
    package = json.loads((ROOT / "desktop/package.json").read_text())
    tauri = json.loads((ROOT / "desktop/src-tauri/tauri.conf.json").read_text())
    cargo = (ROOT / "desktop/src-tauri/Cargo.toml").read_text()
    match = re.search(r'^version = "([^"]+)"$', cargo, flags=re.MULTILINE)
    if match is None:
        fail("Cargo Desktop version is missing")
    versions = {package["version"], tauri["version"], match.group(1)}
    if len(versions) != 1:
        fail(f"Desktop versions drifted: {sorted(versions)}")
    return versions.pop()


def check_updater_signature_audit() -> None:
    with tempfile.TemporaryDirectory(prefix="mazzy-updater-audit-") as workspace_raw:
        workspace = Path(workspace_raw)
        metadata_dir = workspace / "updater-metadata"
        assets_dir = workspace / "updater-assets"
        metadata_dir.mkdir()
        assets_dir.mkdir()
        artifact = assets_dir / "Mazzy.AppImage.tar.gz"
        artifact.write_bytes(b"signed updater fixture")
        manifest = {
            "platforms": {
                "linux-x86_64": {
                    "url": (
                        "https://github.com/mazurovn/mazzy-vpn/releases/download/"
                        "desktop-v0.4.1/Mazzy.AppImage.tar.gz"
                    ),
                    "signature": "A" * 64,
                }
            }
        }
        (metadata_dir / "latest.json").write_text(
            json.dumps(manifest), encoding="utf-8"
        )

        result = subprocess.run(
            [sys.executable, str(UPDATER_SIGNATURE_AUDIT)],
            cwd=workspace,
            check=True,
            capture_output=True,
            timeout=10,
        )
        signature = workspace / "updater-signatures/signature-000.sig"
        expected = os.fsencode(artifact) + b"\0" + os.fsencode(signature) + b"\0"
        if (
            result.stdout != expected
            or signature.read_text(encoding="utf-8") != "A" * 64 + "\n"
        ):
            fail("Updater signature audit did not emit the expected safe arguments")

        (assets_dir / "unexpected-directory").mkdir()
        unsafe = subprocess.run(
            [sys.executable, str(UPDATER_SIGNATURE_AUDIT)],
            cwd=workspace,
            check=False,
            capture_output=True,
            timeout=10,
        )
        if unsafe.returncode == 0 or b"unsafe entry" not in unsafe.stderr:
            fail("Updater signature audit accepted an unsafe asset entry")


def main() -> None:
    check_updater_signature_audit()
    html = HTML.read_text(encoding="utf-8")
    javascript = JS.read_text(encoding="utf-8")
    rust = RUST.read_text(encoding="utf-8")
    main_rust = MAIN.read_text(encoding="utf-8")
    agent_rust = AGENT.read_text(encoding="utf-8")
    updater_rust = UPDATER.read_text(encoding="utf-8")
    cargo_manifest = CARGO_MANIFEST.read_text(encoding="utf-8")
    tauri_config = json.loads(TAURI_CONFIG.read_text(encoding="utf-8"))
    capability = json.loads(CAPABILITY.read_text(encoding="utf-8"))
    build_release = BUILD_RELEASE.read_text(encoding="utf-8")
    desktop_workflow = DESKTOP_WORKFLOW.read_text(encoding="utf-8")

    parser = UiParser()
    parser.feed(html)
    duplicates = sorted(identifier for identifier, count in Counter(parser.ids).items() if count > 1)
    if duplicates:
        fail(f"Desktop HTML contains duplicate IDs: {duplicates}")

    js_ids = set(re.findall(r'\$\("#([A-Za-z0-9_-]+)"\)', javascript))
    missing_ids = sorted(js_ids - set(parser.ids))
    if missing_ids:
        fail(f"Desktop JavaScript references missing IDs: {missing_ids}")

    values = translations(javascript)
    for language in ("ru", "en"):
        missing = sorted(parser.i18n - set(values.get(language, {})))
        if missing:
            fail(f"{language} Desktop translation is incomplete: {missing}")
    localized_verification = {
        "connectFastest",
        "checkAllLocations",
        "lastChecked",
        "sortLatency",
        "realVerificationTitle",
        "realVerificationHint",
        "verifyNow",
        "verifyWithSpeed",
        "observedLocation",
        "profileLocationMatch",
        "systemEgress",
        "ipv6Leak",
        "dnsRouting",
        "speedSample",
        "verificationDisclaimer",
        "partiallyVerified",
        "profileCountryMissing",
        "startService",
        "restartService",
        "stopService",
    }
    localized_updater = {
        "checkUpdates",
        "automaticUpdateChecks",
        "automaticUpdateChecksHint",
        "updateAvailable",
        "currentDesktopVersion",
        "availableDesktopVersion",
        "signedUpdate",
        "signedUpdateHint",
        "updateDialogInstallHint",
        "updateDialogPackageHint",
        "updateLater",
        "installUpdate",
        "openRelease",
        "checkingForUpdates",
        "upToDate",
        "updateDownloading",
        "updateInstalling",
        "updateInstalled",
        "restartNow",
        "updateCheckFailed",
    }
    for language in ("ru", "en", "de", "zh", "ja", "ko"):
        missing = sorted(
            (localized_verification | localized_updater)
            - set(values.get(language, {}))
        )
        if missing:
            fail(f"{language} safety-critical UI is not localized: {missing}")

    for dangerous in (".innerHTML", "insertAdjacentHTML", "eval(", "new Function"):
        if dangerous in javascript:
            fail(f"Desktop webview uses an unsafe DOM/code primitive: {dangerous}")
    if 'localStorage.getItem("mazzy-hide-ip") !== "false"' not in javascript:
        fail("Desktop privacy mode is not enabled by default")
    if 'invoke("verify_connection"' not in javascript:
        fail("Desktop real egress card bypasses the typed Rust command")
    if (
        "limitedConfidenceVerification" not in javascript
        or 'verify.geo.expected-country-unavailable' not in javascript
        or 'verify.geo.single-provider' not in javascript
        or 'result?.ipv4?.same_egress === true' not in javascript
        or 'result?.ipv6?.potential_leak === false' not in javascript
        or 'result?.dns?.state === "vpn-full-tunnel"' not in javascript
    ):
        fail("Desktop misclassifies limited GeoIP confidence as an egress risk")
    if (
        'key: "partiallyVerified", css: "partial", severity: "warning"'
        not in javascript
        or 'presentation.css === "partial"' not in javascript
    ):
        fail("Desktop partial verification is visually promoted to full success")
    if 'invoke("probe_profiles"' not in javascript:
        fail("Desktop location checks bypass the typed Rust command")
    if 'invoke("get_agent_integrations"' not in javascript:
        fail("Desktop agent view bypasses the typed Rust diagnostics command")
    forbidden_agent_authority = (
        "run_agent_operation",
        "runAgentOperation",
        "codex-remote-start",
        "codex-remote-pair",
        "codex-remote-stop",
        "pairingGrant",
    )
    for authority in forbidden_agent_authority:
        if authority in javascript or authority in html:
            fail(f"Desktop renderer retains executable Agent Control authority: {authority}")
    if "const readyProviders = 0;" in javascript:
        fail("Desktop provider-readiness badge is hard-coded")
    if (
        "provider.installed" not in javascript
        or "provider.remote_control_supported" not in javascript
        or 'provider.adapter_status === "ready"' not in javascript
        or 'provider.connection_model !== "diagnostics-only"' not in javascript
    ):
        fail("Desktop provider-readiness badge does not enforce runtime readiness")
    if 'previewParameters.get("preview") === "docs"' not in javascript:
        fail("Desktop documentation preview is missing")
    if 'previewParameters.get("update") === "available"' not in javascript:
        fail("Desktop signed-update documentation fixture is missing")
    if (
        "!window.__TAURI_INTERNALS__" not in javascript
        or 'window.location.protocol === "http:"' not in javascript
        or "localDocumentationHosts.has(window.location.hostname)" not in javascript
    ):
        fail("Desktop documentation preview can be activated in packaged Tauri")
    preview_start = javascript.find("function documentationPreviewData()")
    preview_end = javascript.find("function renderDocumentationPreview()", preview_start)
    if preview_start < 0 or preview_end < 0:
        fail("Desktop documentation fixture boundary is missing")
    preview_fixture = javascript[preview_start:preview_end]
    runtime_javascript = javascript[:preview_start] + javascript[preview_end:]
    if preview_fixture.count('"203.0.113.7"') < 4:
        fail("Desktop documentation preview does not consistently use RFC 5737 data")
    for fixture_only_value in (
        "203.0.113.7",
        "Belgium — Brussels",
        "Germany — Berlin",
        "Netherlands — Amsterdam",
    ):
        if fixture_only_value in runtime_javascript:
            fail(f"Documentation value escaped into Desktop runtime: {fixture_only_value}")
    for forbidden_preview_value in ("198.51.100.", "192.0.2."):
        if forbidden_preview_value in preview_fixture:
            fail(f"Desktop documentation preview mixes address fixtures: {forbidden_preview_value}")
    if 'if (!documentationPreview) throw new Error' not in preview_fixture:
        fail("Desktop documentation fixture does not fail closed outside preview mode")
    if javascript.count('if (sortMode === "status")') != 1:
        fail("Desktop profile status sorting is duplicated or missing")
    if javascript.count("function statusMatchesProfile(profile)") != 1:
        fail("Desktop active-profile state is not resolved through one exact identity helper")
    if 'data?.available !== false' not in javascript or 't("profilesUnavailable")' not in javascript:
        fail("Desktop hides an unavailable profile cache as an empty profile library")
    for identity_field in ("state.status.profile_id", "state.status.profile_file_name"):
        if identity_field not in javascript:
            fail(f"Desktop active-profile state ignores {identity_field}")
    if "deny_unknown_fields" not in rust or "tests.verify-egress" not in rust:
        fail("Desktop does not strictly sanitize the egress API contract")
    if "sanitize_status_cache" not in rust:
        fail("Desktop passes the privileged status cache to the WebView without validation")
    if "backend::sanitize_status_cache" not in main_rust:
        fail("Desktop status command bypasses the typed cache sanitizer")
    if 'backend::verify_connection' not in main_rust:
        fail("Desktop does not expose the typed egress verification command")
    if "run_agent_operation" in main_rust or "run_agent_operation" in agent_rust:
        fail("Desktop Tauri invoke surface retains Agent Control mutation authority")
    if "Command::new" in agent_rust:
        fail("Diagnostics-only Agent Control executes an untrusted discovered binary")
    if 'agent_control::get_agent_integrations' not in main_rust:
        fail("Desktop does not expose read-only Agent Control diagnostics")
    if '<dialog class="update-dialog" id="update-dialog"' not in html:
        fail("Desktop update consent is not presented in a modal dialog")
    for command in (
        'invoke("check_for_update")',
        'invoke("install_update")',
        'invoke("open_update_release")',
        'invoke("restart_application")',
    ):
        if command not in javascript:
            fail(f"Desktop update workflow is incomplete: {command}")
    install_call = 'invoke("install_update")'
    update_action_start = javascript.find("async function runPendingUpdateAction()")
    update_action_end = javascript.find("\nfunction handleUpdateProgress", update_action_start)
    if (
        javascript.count(install_call) != 1
        or update_action_start < 0
        or update_action_end < 0
        or not update_action_start
        < javascript.find(install_call)
        < update_action_end
    ):
        fail("Desktop can install an update outside the explicit consent action")
    if 'if (state.autoUpdateChecks) void checkForUpdates(false);' not in javascript:
        fail("Desktop does not automatically check for updates at startup")
    if (
        'localStorage.getItem("mazzy-auto-update-check") !== "false"'
        not in javascript
    ):
        fail("Desktop startup update checks are not enabled by default")
    if "info.install_supported" not in javascript:
        fail("Desktop does not distinguish package-managed updates")
    if "download_and_install" not in updater_rust:
        fail("Desktop updater does not verify and install signed artifacts")
    if (
        'std::env::var_os("APPIMAGE")' not in updater_rust
        or "DEB/RPM" not in updater_rust
    ):
        fail("Desktop updater can overwrite a package-managed Linux install")
    if (
        'https://github.com/mazurovn/mazzy-vpn/releases/tag/desktop-v'
        not in updater_rust
        or "safe_release_version" not in updater_rust
    ):
        fail("Desktop updater can open an untrusted release URL")
    if "PendingUpdate(Mutex<Option<Update>>)" not in updater_rust:
        fail("Desktop updater does not keep consent state in the trusted backend")
    if (
        ".as_ref()\n        .cloned()" not in updater_rust
        or "update.install_failed" not in updater_rust
        or "state.updateInfo = null" in javascript
    ):
        fail("Desktop cannot safely retry a failed signed update download")
    if "tauri_plugin_updater::Builder::new().build()" not in main_rust:
        fail("Desktop signed updater plugin is not initialized")
    if (
        'tauri-plugin-log = "2"' not in cargo_manifest
        or "tauri_plugin_log::Builder::new()" not in main_rust
        or ".max_file_size(1_000_000)" not in main_rust
        or "RotationStrategy::KeepOne" not in main_rust
        or 'target: "mazzy_vpn_desktop::operation"' not in rust
        or 'target: "mazzy_vpn_desktop::verification"' not in rust
    ):
        fail("Desktop persistent diagnostics logging is missing or unbounded")
    single_instance = main_rust.find(".plugin(tauri_plugin_single_instance::init")
    opener_plugin = main_rust.find(".plugin(tauri_plugin_opener::init())")
    if single_instance < 0 or opener_plugin < 0 or single_instance > opener_plugin:
        fail("Desktop single-instance protection is missing or not the first plugin")
    updater_config = tauri_config.get("plugins", {}).get("updater", {})
    if updater_config.get("endpoints") != [
        "https://github.com/mazurovn/mazzy-vpn/releases/download/desktop-updater/latest.json"
    ]:
        fail("Desktop updater endpoint is not pinned to the project update feed")
    if len(updater_config.get("pubkey", "")) < 100:
        fail("Desktop updater does not embed a release verification key")
    if tauri_config.get("bundle", {}).get("createUpdaterArtifacts") is not False:
        fail("Unsigned pull-request builds can create updater artifacts")
    renderer_permissions = capability.get("permissions", [])
    if any(
        isinstance(permission, str)
        and permission.startswith(("updater:", "opener:"))
        for permission in renderer_permissions
    ):
        fail("The renderer can bypass the trusted update consent commands")
    if (
        'if (process.env.TAURI_SIGNING_PRIVATE_KEY)' not in build_release
        or "createUpdaterArtifacts: true" not in build_release
        or "mkdtempSync" not in build_release
        or "writeFileSync" not in build_release
        or "shell: false" not in build_release
    ):
        fail("Release builds do not safely gate updater artifacts on the signing key")
    if (
        'spawnSync("cargo", [' not in build_release
        or "process.env.CARGO" in build_release
    ):
        fail("Release builds allow the cargo executable to be replaced through input")
    for workflow_boundary in (
        "TAURI_SIGNING_PRIVATE_KEY: ${{ secrets.TAURI_SIGNING_PRIVATE_KEY }}",
        "environment: desktop-release",
        "publish-update-feed:",
        "needs: preview-release",
        "gh release edit \"$RELEASE_TAG\"",
        "gh release upload desktop-updater",
        'startswith("linux-")',
        'startswith("windows-")',
        'startswith("darwin-")',
        "max-parallel: 1",
        "prepare-updater-signature-audit.py",
        "--example verify-updater-signatures",
    ):
        if workflow_boundary not in desktop_workflow:
            fail(f"Desktop signed release workflow is incomplete: {workflow_boundary}")
    if desktop_workflow.count("environment: desktop-release") < 2:
        fail("Desktop update-feed publication bypasses the protected environment")
    if "mapfile -d '' -t updater_signature_args < <(" in desktop_workflow:
        fail("Desktop signature-audit preparation can hide a failed subprocess")
    for screenshot_name in ("update-en.png", "update-ru.png"):
        screenshot = ROOT / "docs/images" / screenshot_name
        contents = screenshot.read_bytes()
        if contents[:8] != b"\x89PNG\r\n\x1a\n" or len(contents) < 24:
            fail(f"Desktop updater screenshot is not a valid PNG: {screenshot_name}")
        width, height = struct.unpack(">II", contents[16:24])
        if (width, height) != (1680, 951):
            fail(
                f"Desktop updater screenshot has unexpected dimensions: "
                f"{screenshot_name}={width}x{height}"
            )
    for tray_id in (
        "profiles",
        "agents",
        "diagnostics",
        "settings",
        "about",
        "verify",
        "probe-all",
        "autostart-on",
        "autostart-off",
        "monitor-on",
        "monitor-off",
    ):
        if f'"{tray_id}"' not in main_rust:
            fail(f"Desktop tray action is missing: {tray_id}")
    if 'const CLI_PATH: &str = "/usr/local/bin/mazzy-vpn"' in main_rust:
        fail("Desktop tray is pinned to a non-package-managed local engine")

    version = desktop_version()
    print(
        f"Desktop UI contract OK: v{version}, {len(parser.ids)} IDs, "
        f"{len(parser.i18n)} localized labels"
    )


if __name__ == "__main__":
    main()
