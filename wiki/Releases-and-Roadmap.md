# Релизы и roadmap

## 1.2.0 / Desktop 0.2.0 — выпущен как preview

- Linux control center с Dashboard, Profiles, Diagnostics и Settings;
- встроенный installer/engine, проверка версий и зависимостей, install/repair;
- импорт файлов/папок, поиск/выбор профиля и безопасные profile actions;
- validate, probe, transactional tests, test-all и emergency;
- полный вывод Doctor, self-test и bounded logs;
- управление autostart и независимым health monitor;
- типизированный Rust adapter без shell-команд и расширенные parity tests;
- экран «О программе» с версиями, автором, лицензией, приватностью и правилами;
- Android/iOS release gates и общий platform roadmap.

Desktop 0.2 не требует предварительной ручной установки CLI, но остаётся
preview до versioned service API, полного паритета режимов/локализаций,
подписанного update/rollback и закрытия platform release gates.

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

Version 1.2.0 / Desktop 0.2.0 is released as a preview. It expands the Linux preview
into a Dashboard/Profiles/Diagnostics/Settings control center with bundled
engine bootstrap, version/dependency checks, file/folder import, profile
actions, validation/probes/transactional tests, complete Doctor/self-test/log
output, service controls and a typed Rust adapter.
It also adds an About screen, explicit privacy rules and Android/iOS release
gates.

Desktop 0.2 does not require a prior manual CLI installation, but remains a
preview until the versioned service API, complete mode/localization parity,
signed update/rollback and platform release gates are complete.

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
