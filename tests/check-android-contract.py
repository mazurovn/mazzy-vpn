#!/usr/bin/env python3
"""Static embedded Android engine gate; it never substitutes for device tests."""
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ANDROID = ROOT / "android"
manifest = (ANDROID / "app/src/main/AndroidManifest.xml").read_text()
service = (ANDROID / "app/src/main/kotlin/com/mazzy/vpn/MazzyVpnService.kt").read_text()
profile_repository = (ANDROID / "app/src/main/java/com/mazzy/vpn/core/AwgProfileRepository.kt").read_text()
secret_store = (ANDROID / "app/src/main/java/com/mazzy/vpn/core/SecureSecretStore.kt").read_text()
native_cmake = (ANDROID / "amneziawg-tunnel/native/CMakeLists.txt").read_text()
native_makefile = (ANDROID / "amneziawg-tunnel/native/libwg-go/Makefile").read_text()
tunnel_build = (ANDROID / "amneziawg-tunnel/build.gradle.kts").read_text()
validator = (ANDROID / "app/src/main/java/com/mazzy/vpn/core/ManagedProfileValidator.kt").read_text()
registry = json.loads((ROOT / "protocols/v1/registry.json").read_text())

required = [
    ANDROID / "settings.gradle.kts",
    ANDROID / "app/build.gradle.kts",
    ANDROID / "app/src/main/kotlin/com/mazzy/vpn/MainActivity.kt",
    ANDROID / "app/src/main/kotlin/com/mazzy/vpn/MazzyVpnService.kt",
    ANDROID / "app/src/main/java/com/mazzy/vpn/core/SecureSecretStore.kt",
    ANDROID / "app/src/main/java/com/mazzy/vpn/core/ProfileImportService.kt",
    ANDROID / "app/src/main/java/com/mazzy/vpn/core/AwgProfileRepository.kt",
    ANDROID / "amneziawg-tunnel/native/libwg-go/Makefile",
    ANDROID / "gradlew",
    ANDROID / "gradle/wrapper/gradle-wrapper.jar",
]
assert all(path.is_file() for path in required)
assert "android.permission.BIND_VPN_SERVICE" in manifest
assert "android.net.VpnService" in manifest
assert "android.permission.INTERNET" in manifest
assert "android.permission.ACCESS_NETWORK_STATE" in manifest
assert manifest.count('android.permission.BIND_VPN_SERVICE') == 1
assert "android.net.VpnService.SUPPORTS_ALWAYS_ON" in manifest and 'android:value="false"' in manifest
assert "startForeground" in service
assert "Builder().setSession" in service and "builder.establish()" in service
assert 'System.loadLibrary("wg-go")' in service
assert "GoBackend.awgTurnOn" in service and "GoBackend.awgTurnOff" in service
assert "GoBackend.awgGetSocketV4" in service and "GoBackend.awgGetSocketV6" in service
assert "protectSocketIfPresent" in service and "check(protect(socket))" in service
assert "waitForHandshake" in service and "last_handshake_time_sec=" in service
assert "override fun onRevoke()" in service
assert "PROFILE_SCHEMA" in profile_repository and 'secrets.write(PROFILE_SECRET, envelope)' in profile_repository
assert "CodingErrorAction.REPORT" in profile_repository
assert "validateFullTunnelConfig" in profile_repository and "profile-has-no-ipv4-default-route" in profile_repository
assert "profile-has-no-endpoint" in profile_repository and "profile-requires-bounded-keepalive" in profile_repository
assert "AtomicLong" in service and "requireCurrentGeneration" in service
assert "stateTransitionLock" in service and "NET_CAPABILITY_NOT_VPN" in service
assert "secret-store-decryption-failed" in secret_store
assert "libwg-go.so" in native_cmake
assert "NATIVE_SOURCES :=" in native_makefile and "$(NATIVE_SOURCES)" in native_makefile
assert "libwg.so" not in native_cmake and "libwg-quick.so" not in native_cmake
assert 'java.exclude("org/amnezia/awg/backend/**")' in tunnel_build
assert "RootShell.java" in tunnel_build and "ToolsInstaller.java" in tunnel_build
assert "amneziawg-android-APACHE-2.0.txt" in tunnel_build and "amneziawg-go-MIT.txt" in tunnel_build
assert "insecure_tls" in validator and "unsupported_protocol" in validator
assert 'require(profile.protocol in protocols)' in validator
assert 'tls["insecure"] == false' in validator
assert (ANDROID / "app/src/test/java/com/mazzy/vpn/core/ManagedProfileTest.kt").exists()
assert "15" in (ROOT / "docs/ANDROID_ARCHITECTURE.en.md").read_text()
assert "15" in (ROOT / "docs/ANDROID_ARCHITECTURE.ru.md").read_text()
for item in registry["protocols"]:
    assert item["support"]["android"] == "planned", item["id"]
print("PASS: embedded Android AmneziaWG contract; runtime/device gates remain open")
