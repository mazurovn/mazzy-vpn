#!/usr/bin/env python3
"""Fail closed when Desktop HTML, JavaScript, Rust and versions drift apart."""

from __future__ import annotations

import json
import re
import subprocess
from collections import Counter
from html.parser import HTMLParser
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
HTML = ROOT / "desktop/ui/index.html"
JS = ROOT / "desktop/ui/app.js"
RUST = ROOT / "desktop/src-tauri/src/backend.rs"
MAIN = ROOT / "desktop/src-tauri/src/main.rs"


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
        ["node", "-e", script],
        check=True,
        capture_output=True,
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


def main() -> None:
    html = HTML.read_text(encoding="utf-8")
    javascript = JS.read_text(encoding="utf-8")
    rust = RUST.read_text(encoding="utf-8")
    main_rust = MAIN.read_text(encoding="utf-8")

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
        "startService",
        "restartService",
        "stopService",
    }
    for language in ("ru", "en", "de", "zh", "ja", "ko"):
        missing = sorted(localized_verification - set(values.get(language, {})))
        if missing:
            fail(f"{language} verification UI is not localized: {missing}")

    for dangerous in (".innerHTML", "insertAdjacentHTML", "eval(", "new Function"):
        if dangerous in javascript:
            fail(f"Desktop webview uses an unsafe DOM/code primitive: {dangerous}")
    if 'localStorage.getItem("mazzy-hide-ip") !== "false"' not in javascript:
        fail("Desktop privacy mode is not enabled by default")
    if 'invoke("verify_connection"' not in javascript:
        fail("Desktop real egress card bypasses the typed Rust command")
    if 'invoke("probe_profiles"' not in javascript:
        fail("Desktop location checks bypass the typed Rust command")
    if 'invoke("get_agent_integrations"' not in javascript:
        fail("Desktop agent view bypasses the typed Rust diagnostics command")
    if 'invoke("run_agent_operation"' not in javascript:
        fail("Desktop agent lifecycle bypasses the typed Rust operation command")
    if 'state.pairingGrant = pairing' not in javascript or "localStorage" in javascript[
        javascript.find("function retainPairingGrant") : javascript.find(
            "function renderAgentIntegrations"
        )
    ]:
        fail("Desktop pairing grant is missing or persisted in browser storage")
    if 'previewParameters.get("preview") === "docs"' not in javascript:
        fail("Desktop documentation preview is missing")
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
    if 'agent_control::run_agent_operation' not in main_rust:
        fail("Desktop does not expose the typed agent operation command")
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
