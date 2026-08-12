# Mazzy VPN Android client — release audit area

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

Status: embedded AmneziaWG/WireGuard `device-test candidate`, not production.
The APK builds a pinned userspace `libwg-go.so`, imports native `.conf`
profiles into one Android-Keystore-encrypted envelope, establishes its own TUN
through the single app `VpnService`, protects backend sockets and reports
connected only after a handshake. It does not invoke or install the Linux CLI,
systemd, root helpers, or a second VPN application.

## What lives here

- Architecture, limitations, risks and the audit protocol:
  [`docs/ANDROID_ARCHITECTURE.en.md`](../docs/ANDROID_ARCHITECTURE.en.md) and
  [`docs/ANDROID_ARCHITECTURE.ru.md`](../docs/ANDROID_ARCHITECTURE.ru.md).
- Contract assertions: `tests/check-android-contract.py` — stdlib-only,
  runs on every CI run and keeps the Android source, architecture docs,
  `protocols/v1/registry.json`, `docs/capabilities.json` and CI wiring
  synchronized.
- Build validation: `tests/check-android-build.sh` — always runs the contract
  check, then **honestly skips** the real Gradle build (explicit `SKIP:` line,
  exit 0) when `android/gradlew` or the Android SDK
  (`ANDROID_HOME`/`ANDROID_SDK_ROOT`) is unavailable. A skip is never reported
  as a build pass; when the wrapper and SDK exist, the build runs for real and
  its failure fails CI.

## Audit protocol in one paragraph

The release audit iterates over **15 gates**: the 13 capabilities required by
the `mobile-android-1.0` release gate in `docs/capabilities.json`, plus
`android-signed-build` and `android-contract-parity`. Every iteration records
findings per gate; any finding resets the clean counter. The audit pass stops
after **five consecutive clean audits** — only then can the
`mobile-android-1.0` gate be considered for promotion, and `planned` remains
the honest status until a real, instrumented build exists.

Until emulator and physical-device routing/leak gates pass, every protocol in
`protocols/v1/registry.json` keeps
`support.android: planned`. The nine proxy/transport protocols (`vless`,
`hysteria2`, `mieru`, `naive`, `tuic`, `shadowsocks2022`, `trojan`, `anytls`,
`shadowtls`) are explicitly unsupported on Android: catalog presence is not
connection support, and none of them may be presented as a device-wide VPN
before its own pinned engine passes `VpnService` TUN integration tests.

The current vertical slice supports native AmneziaWG/WireGuard configuration
syntax only. OpenVPN is the next embedded backend and is not silently delegated
to an external Android app.

## Physical-device gate

Enable USB debugging, connect and authorize exactly one phone, then run from
the repository root (the profile content is never printed):

```bash
scripts/android-device-smoke.sh --profile /absolute/path/to/test.conf
```

Import the staged file in the app, remove that plaintext staging copy using the
printed command, grant Android's VPN permission, connect and verify public IP,
DNS, IPv4/IPv6 leak behaviour, Wi-Fi/mobile handoff, revoke, process death and
reboot. Roll back the debug install with:

```bash
scripts/android-device-smoke.sh --uninstall
```
