# Mazzy VPN capability parity / Паритет функций

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

Machine-readable source of truth:
[`capabilities.json`](capabilities.json). CI validates that every capability,
test/document reference and release gate remains consistent.

Status: **I** implemented · **P** partial · **R** planned · **—** not applicable.

| Capability ID | CLI | TUI | Desktop Linux | macOS | Windows |
|---|---:|---:|---:|---:|---:|
| `connection-lifecycle` | I | I | P | R | R |
| `profile-import` | I | I | R | R | R |
| `profile-location-selection` | I | I | R | R | R |
| `validation-probe-test` | I | I | R | R | R |
| `operating-modes` | I | I | R | R | R |
| `service-control` | I | P | R | R | R |
| `dependency-bootstrap` | I | — | R | R | R |
| `dashboard-tray` | I | I | I | P | P |
| `localization-six-languages` | I | I | I | I | I |
| `automatic-recovery` | I | I | P | R | R |
| `self-contained-runtime` | I | I | R | R | R |
| `privilege-boundary` | I | I | P | R | R |

## Release gates

| Gate | Declared ready | Meaning |
|---|---:|---|
| `cli-tui-1.1` | yes | Current Linux CLI/TUI release gate |
| `desktop-linux-1.0` | **no** | Standalone Linux application; no separate CLI install |
| `desktop-macos-1.0` | **no** | Standalone signed macOS application with native backend |
| `desktop-windows-1.0` | **no** | Standalone signed Windows application with native backend |

The validator calculates readiness from the matrix. A gate cannot be marked
ready while any required capability is `partial`, `planned` or
`not-applicable`.

## Русский

Desktop 0.1 — preview интерфейса, а не самостоятельный VPN-клиент. Версия
Desktop 1.0 допускается только после реализации полного импорта и выбора
профилей, режимов, управления службами, bootstrap зависимостей, общего core,
автовосстановления и platform-specific backend.

Новая функция считается завершённой не после добавления одной кнопки, а после
обновления общего API/core, всех применимых интерфейсов, автоматических тестов,
матрицы и документации на русском и английском.

## English

Desktop 0.1 is a UI preview, not a standalone VPN client. Desktop 1.0 is
allowed only after profile import/selection, modes, service control, dependency
bootstrap, the shared core, recovery and the platform-native backend are
implemented.

A feature is complete only after its shared API/core, every applicable
interface, automated tests, this registry and both Russian and English
documentation are updated.
