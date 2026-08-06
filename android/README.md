# Mazzy VPN Android client — release audit area

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

Status: `foundation`. This directory now contains the native Android
foundation, but it is not an installable VPN product yet. The service has an
explicit lifecycle and refuses to establish a TUN interface until a pinned
engine adapter is present. The client is native `VpnService`, not a wrapped
Desktop UI and not a shell CLI inside the app.

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

Until the native engine and device gates pass, every protocol in
`protocols/v1/registry.json` keeps
`support.android: planned`. The nine proxy/transport protocols (`vless`,
`hysteria2`, `mieru`, `naive`, `tuic`, `shadowsocks2022`, `trojan`, `anytls`,
`shadowtls`) are explicitly unsupported on Android: catalog presence is not
connection support, and none of them may be presented as a device-wide VPN
before a pinned, reproducibly built engine passes `VpnService` TUN integration
tests.
