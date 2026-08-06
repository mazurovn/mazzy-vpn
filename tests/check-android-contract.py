#!/usr/bin/env python3
"""Static Android foundation gate; it never substitutes for device tests."""
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ANDROID = ROOT / "android"
manifest = (ANDROID / "app/src/main/AndroidManifest.xml").read_text()
service = (ANDROID / "app/src/main/kotlin/com/mazzy/vpn/MazzyVpnService.kt").read_text()
validator = (ANDROID / "app/src/main/java/com/mazzy/vpn/core/ManagedProfileValidator.kt").read_text()
registry = json.loads((ROOT / "protocols/v1/registry.json").read_text())

required = [
    ANDROID / "settings.gradle.kts",
    ANDROID / "app/build.gradle.kts",
    ANDROID / "app/src/main/kotlin/com/mazzy/vpn/MainActivity.kt",
    ANDROID / "app/src/main/kotlin/com/mazzy/vpn/MazzyVpnService.kt",
    ANDROID / "app/src/main/java/com/mazzy/vpn/core/SecureSecretStore.kt",
    ANDROID / "app/src/main/java/com/mazzy/vpn/core/ProfileImportService.kt",
]
assert all(path.is_file() for path in required)
assert "android.permission.BIND_VPN_SERVICE" in manifest
assert "android.net.VpnService" in manifest
assert "android.permission.INTERNET" in manifest
assert "startForeground" in service
assert "establish()" not in service, "foundation must not fake a tunnel"
assert "insecure_tls" in validator and "unsupported_protocol" in validator
assert "protocol.lowercase(Locale.ROOT)" in validator
assert "profile.tls[\"insecure\"] is Boolean" in validator
assert (ANDROID / "app/src/test/java/com/mazzy/vpn/core/ManagedProfileTest.kt").exists()
assert "15" in (ROOT / "docs/ANDROID_ARCHITECTURE.en.md").read_text()
assert "15" in (ROOT / "docs/ANDROID_ARCHITECTURE.ru.md").read_text()
for item in registry["protocols"]:
    assert item["support"]["android"] == "planned", item["id"]
print("PASS: Android foundation contract; runtime/device gates remain open")
