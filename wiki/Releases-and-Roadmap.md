# Релизы и roadmap

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

Готовность вычисляется release gates в `docs/capabilities.json`. Пока
соответствующий gate false, пакет остаётся preview.

---

<a id="english"></a>

# Releases and roadmap

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
