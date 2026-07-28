#!/usr/bin/env python3
"""Fail when documentation fixtures or fixed locations escape into runtime paths."""

from __future__ import annotations

import re
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
CORE = (ROOT / "mazzy-vpn").read_text(encoding="utf-8")
WEBVIEW = (ROOT / "desktop/ui/app.js").read_text(encoding="utf-8")
RUST = (ROOT / "desktop/src-tauri/src/backend.rs").read_text(encoding="utf-8")


def fail(message: str) -> None:
    raise AssertionError(message)


def section(source: str, start: str, end: str) -> str:
    start_offset = source.find(start)
    end_offset = source.find(end, start_offset + len(start))
    if start_offset < 0 or end_offset < 0:
        fail(f"Expected source boundary is missing: {start!r} .. {end!r}")
    return source[start_offset:end_offset]


profile_parser = section(CORE, "profile_file_stem() {", "profile_endpoint() {")
for required in (
    "profile_metadata_value",
    "nmconnection_profile_name",
    "profile_location",
    "profile_config_country_code",
):
    if required not in profile_parser:
        fail(f"Runtime profile parser is missing {required}")

endpoint_parser = section(CORE, "profile_endpoint() {", "endpoint_host() {")
for directive in ('"remote"', "Endpoint", "gateway"):
    if directive not in endpoint_parser:
        fail(f"Runtime endpoint parser no longer reads {directive} from profiles")

profile_catalog = section(CORE, "profiles_json_live() {", "profile_opaque_id() {")
if 'location="$(profile_location "$file")"' not in profile_catalog:
    fail("Profile catalog derives location without parsing the profile")
if 'country_code="$(profile_config_country_code "$file"' not in profile_catalog:
    fail("Profile catalog ignores explicit country metadata")
if re.search(r'"location":"%s".*profile_name', profile_catalog, flags=re.DOTALL):
    fail("Profile catalog aliases location to a fixed/display-only profile name")

country_inference = section(
    CORE, "profile_expected_country_code() {", "verify_geo_record() {"
)
if 'profile_config_country_code "$1"' not in country_inference:
    fail("Egress verification ignores the config country code")
for location_hint in (
    "belgium",
    "brussels",
    "germany",
    "berlin",
    "moscow",
    "russia",
):
    if location_hint in country_inference.lower():
        fail(f"Runtime country inference still hard-codes a location: {location_hint}")

dns_hook = section(CORE, "openvpn_dns_up() {", "openvpn_dns_down() {")
if "1.1.1.1" in dns_hook:
    fail("OpenVPN DNS hook silently hard-codes a public resolver")
if "OPENVPN_FALLBACK_DNS" not in dns_hook:
    fail("OpenVPN DNS hook has no explicit administrator override")
if 'OPENVPN_FALLBACK_DNS="${VPNCTL_OPENVPN_FALLBACK_DNS:-}"' not in CORE:
    fail("OpenVPN fallback DNS is not empty by default")

preview_fixture = section(
    WEBVIEW, "function documentationPreviewData()", "function renderDocumentationPreview()"
)
webview_runtime = WEBVIEW.replace(preview_fixture, "")
for fixture_value in (
    "203.0.113.7",
    "Belgium — Brussels",
    "Germany — Berlin",
    "Netherlands — Amsterdam",
):
    if fixture_value in webview_runtime:
        fail(f"Documentation fixture escaped into WebView runtime: {fixture_value}")
if 'if (!documentationPreview) throw new Error' not in preview_fixture:
    fail("Documentation fixture does not fail closed outside preview mode")
if (
    "!window.__TAURI_INTERNALS__" not in WEBVIEW
    or 'window.location.protocol === "http:"' not in WEBVIEW
    or "localDocumentationHosts.has(window.location.hostname)" not in WEBVIEW
):
    fail("Documentation preview can be activated in the packaged Tauri runtime")

rust_runtime = RUST.split("#[cfg(test)]", 1)[0]
for fixture_value in ("203.0.113.7", "Belgium", "Brussels"):
    if fixture_value in rust_runtime:
        fail(f"Test fixture escaped into Rust runtime: {fixture_value}")
if "sanitize_profile_cache" not in rust_runtime:
    fail("Desktop passes the root profile cache to the WebView without validation")
if "sanitize_status_cache" not in rust_runtime:
    fail("Desktop passes the root status cache to the WebView without validation")

probe_defaults = {
    "HEALTH_FALLBACK_PROBE_URL": "https://1.1.1.1/cdn-cgi/trace",
    "PROBE_URL": "https://api.ipify.org",
    "PROBE_V6_URL": "https://api6.ipify.org",
    "VERIFY_GEO_PRIMARY_URL": "https://ipapi.co/json/",
    "VERIFY_GEO_SECONDARY_URL": "https://ipwho.is/",
    "VERIFY_SPEED_URL": "https://speed.cloudflare.com/__down?bytes=5000000",
}
for variable, default in probe_defaults.items():
    expected = f'{variable}="${{VPNCTL_{variable}:-{default}}}"'
    if expected not in CORE:
        fail(f"External diagnostic default is no longer explicit/overrideable: {variable}")

print(
    "Runtime hard-code audit OK: config-derived profiles/endpoints, "
    "isolated docs fixtures, explicit diagnostic defaults"
)
