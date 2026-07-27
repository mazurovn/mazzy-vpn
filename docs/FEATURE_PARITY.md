# Mazzy VPN capability parity / Паритет функций

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

Machine-readable source of truth:
[`capabilities.json`](capabilities.json). CI validates that every capability,
test/document reference and release gate remains consistent.

Status: **I** implemented · **P** partial · **R** planned · **—** not applicable.

| Capability ID | CLI | TUI | Linux | macOS | Windows | Android | iOS |
|---|---:|---:|---:|---:|---:|---:|---:|
| `connection-lifecycle` | I | I | I | R | R | R | R |
| `profile-import` | I | I | I | R | R | R | R |
| `profile-location-selection` | I | I | I | R | R | R | R |
| `validation-probe-test` | I | I | I | R | R | R | R |
| `operating-modes` | I | I | P | R | R | R | R |
| `service-control` | I | P | I | R | R | — | — |
| `dependency-bootstrap` | I | — | I | R | R | — | — |
| `dashboard-tray` | I | I | I | P | P | R | R |
| `localization-six-languages` | I | I | P | P | P | R | R |
| `automatic-recovery` | I | I | I | R | R | R | R |
| `self-contained-runtime` | I | I | I | R | R | R | R |
| `privilege-boundary` | I | I | P | R | R | R | R |
| `mobile-vpn-lifecycle` | — | — | — | — | — | R | R |

## Release gates

| Gate | Declared ready | Meaning |
|---|---:|---|
| `cli-tui-1.2` | yes | Current Linux CLI/TUI release gate |
| `desktop-linux-1.0` | **no** | Standalone Linux application; no separate CLI install |
| `desktop-macos-1.0` | **no** | Standalone signed macOS application with native backend |
| `desktop-windows-1.0` | **no** | Standalone signed Windows application with native backend |
| `mobile-android-1.0` | **no** | Native signed Android client using `VpnService` |
| `mobile-ios-1.0` | **no** | Native signed iOS client using Network Extension |

The validator calculates readiness from the matrix. A gate cannot be marked
ready while any required capability is `partial`, `planned` or
`not-applicable`.

## Русский

Desktop 0.2 — Linux control-center preview. Он уже включает установщик общего
движка, импорт и выбор профилей, validate/probe/live-test, Doctor с полным
выводом, журнал и управление службами. Gate Desktop 1.0 всё ещё закрыт:
не завершены fallback-policy UI, полный перевод новых экранов на шесть языков
и переход от typed `pkexec`-адаптера к локальному versioned daemon API.
Android и iOS пока являются только планом: UI preview или Desktop wrapper не
считаются мобильным VPN-клиентом.

Новая функция считается завершённой не после добавления одной кнопки, а после
обновления общего API/core, всех применимых интерфейсов, автоматических тестов,
матрицы и документации на русском и английском.

## English

Desktop 0.2 is a Linux control-center preview. It bundles the shared-engine
installer and now exposes profile import/selection, validation, probes, live
tests, full Doctor output, logs and service controls. The Desktop 1.0 gate
remains closed until fallback-policy UI, full six-language coverage for the new
screens and the versioned local daemon API replace the typed `pkexec` adapter.
Android and iOS are currently plans only: a UI preview or wrapped Desktop
frontend does not count as a mobile VPN client.

A feature is complete only after its shared API/core, every applicable
interface, automated tests, this registry and both Russian and English
documentation are updated.
