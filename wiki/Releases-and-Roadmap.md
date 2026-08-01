# Релизы и roadmap

## 1.3.0 / Desktop 0.3.0 — release candidate

- Linux control center с Dashboard, Profiles, Diagnostics и Settings;
- встроенный installer/engine, проверка версий и зависимостей, install/repair;
- импорт файлов/папок, поиск/выбор профиля и безопасные profile actions;
- validate, probe, transactional tests, test-all и emergency;
- полный вывод Doctor, self-test и bounded logs;
- фактическая проверка default/interface egress, двух geo providers, DNS и
  IPv6 с optional bounded speed sample;
- массовый ping списка, сортировка по latency/status/name и connect fastest;
- расширенный tray, кликабельные события и понятные service states;
- управление autostart и независимым health monitor;
- типизированный Rust adapter без shell-команд и расширенные parity tests;
- экран «О программе» с версиями, автором, лицензией, приватностью и правилами;
- Android/iOS release gates и общий platform roadmap.

Desktop 0.3 не требует предварительной ручной установки CLI, но остаётся
preview до versioned service API, полного паритета режимов/локализаций,
подписанного update/rollback, исправленного Tauri/GTK dependency graph и
закрытия platform release gates. Сейчас публикация Desktop 0.3 дополнительно
ждёт PR/default-branch checks: [issue #31](https://github.com/mazurovn/mazzy-vpn/issues/31)
исправлен в candidate точным provenance-verified backport `glib`, а локальный
RustSec gate проходит без suppressions.

Фактическая публикация определяется наличием tags `v1.3.0` и
`desktop-v0.3.0` и соответствующих страниц в GitHub Releases. Пока хотя бы
одного из них нет, release line считается кандидатом.

## 1.2.0 / Desktop 0.2.0 — опубликованная release line

Это текущие опубликованные stable CLI/TUI и Linux Desktop preview. Версия в
source, changelog или PR не заменяет их до появления нового tag и GitHub
Release.

## 1.1.0

- CLI/TUI и двухуровневое автовосстановление;
- безопасный JSON status cache;
- Tauri Desktop Dashboard и tray;
- рабочие Linux AppImage, DEB и RPM;
- macOS app/DMG и Windows MSI/NSIS как UI preview;
- документация, схемы и Wiki на русском и английском;
- CI, Rust tests, ShellCheck, Clippy, npm audit и leak audit.

Desktop 0.1 в составе 1.1.0 — companion preview. Он ещё требует установленный
CLI engine и не является самостоятельным Desktop VPN-клиентом.

## Desktop 1.0

Подробности: [самостоятельный Desktop 1.0](Desktop-Full-Application-Plan).

1. общий versioned core/API для CLI, TUI и Desktop;
2. full profile import/library/location/default selection;
3. normal/test/emergency/fallback modes и transactional rollback;
4. service/autostart/health/recovery controls и полный doctor;
5. self-contained Linux bootstrap без отдельной установки CLI;
6. Windows service/Wintun backend и signed installer;
7. macOS Network Extension/launchd backend и notarized app;
8. accessibility, upgrades/rollback и platform integration/fault tests.

## Windows, macOS, Android и iOS

- Windows: UI preview до Windows Service, WireGuard/Wintun backend и signed
  installer — [issue #7](https://github.com/mazurovn/mazzy-vpn/issues/7).
- macOS: UI preview до Network Extension, signing/notarization —
  [issue #10](https://github.com/mazurovn/mazzy-vpn/issues/10).
- Android: planned native `VpnService` client —
  [issue #13](https://github.com/mazurovn/mazzy-vpn/issues/13).
- iOS: planned native Network Extension client —
  [issue #14](https://github.com/mazurovn/mazzy-vpn/issues/14).
- CLI/TUI: machine output, help и полный service parity —
  [issue #12](https://github.com/mazurovn/mazzy-vpn/issues/12).

Полный порядок и критерии: [План всех платформ](Platform-Roadmap).

Готовность вычисляется release gates в `docs/capabilities.json`. Пока
соответствующий gate false, пакет остаётся preview.

---

<a id="english"></a>

# Releases and roadmap

Version 1.3.0 / Desktop 0.3.0 is the current release candidate. It expands the Linux preview
into a Dashboard/Profiles/Diagnostics/Settings control center with bundled
engine bootstrap, version/dependency checks, file/folder import, profile
actions, validation/probes/transactional tests, complete Doctor/self-test/log
output, service controls and a typed Rust adapter.
It also adds actual default/interface egress verification, two-provider
location agreement, DNS/IPv6 signals, an optional bounded speed sample,
whole-list ping sorting/connect-fastest, an expanded tray, clickable events,
an About screen, explicit privacy rules and Android/iOS release gates.

Desktop 0.3 does not require a prior manual CLI installation, but remains a
preview until the versioned service API, complete mode/localization parity,
signed update/rollback, a fixed Tauri/GTK dependency graph and platform release
gates are complete. Desktop 0.3 publication waits for the remaining PR/default-
branch checks: [issue #31](https://github.com/mazurovn/mazzy-vpn/issues/31) is
resolved in the candidate with the exact provenance-verified `glib` backport,
and the local RustSec gate passes without suppressions.

Publication is determined by tags `v1.3.0` and `desktop-v0.3.0` and their
corresponding GitHub Release pages. Treat the release line as a candidate while
either one is missing.

Version 1.2.0 / Desktop 0.2.0 remains the published release line. A source
version, changelog entry or PR does not replace it until a new tag and GitHub
Release exist.

Version 1.1.0 adds the sanitized status cache, Tauri Dashboard and tray,
functional Linux AppImage/DEB/RPM bundles, macOS/Windows UI previews, bilingual
architecture/Wiki and expanded CI/security checks.

Desktop 0.1 is a companion preview that still requires the installed CLI
engine. It is not the standalone Desktop VPN application.

The [Desktop 1.0 plan](Desktop-Full-Application-Plan#english) introduces one
shared versioned core/API, complete profile/mode/service workflows, a bundled
Linux bootstrap, native Windows and macOS backends, signing, upgrades/rollback
and platform integration/fault tests. Machine-validated capability gates keep
all Desktop platforms in preview until their full requirements pass.

Windows and macOS remain UI previews until their native backends, signing and
platform tests pass ([#7](https://github.com/mazurovn/mazzy-vpn/issues/7),
[#10](https://github.com/mazurovn/mazzy-vpn/issues/10)). Android and iOS are
planned native clients, not Desktop wrappers
([#13](https://github.com/mazurovn/mazzy-vpn/issues/13),
[#14](https://github.com/mazurovn/mazzy-vpn/issues/14)).
See the [all-platform roadmap](Platform-Roadmap#english).
