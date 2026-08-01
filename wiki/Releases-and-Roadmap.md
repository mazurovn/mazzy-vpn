# Релизы и roadmap

## 1.3.2 / Desktop 0.3.2 — protocol foundation и upgrade hotfix

- исправлена совместимость Desktop с legacy cache без `profile_id` и ложная
  надпись «Профили не найдены» при 24 существующих профилях;
- package update устраняет конфликт старого `/usr/local/bin` с новым
  `/usr/bin/mazzy-vpn`, сохраняя обратимый backup;
- real Unix-socket gate исправляет закрытие API response-half после request EOF;
- добавлен versioned каталог из 13 протоколов, redacted URI detection,
  runtime diagnostics и безопасная read-only операция API `protocols.list`;
- VLESS, Hysteria 2, Mieru, NaiveProxy и ещё пять современных направлений пока
  не объявляются готовыми подключениями: остаются TUN/engine adapters и
  integration gates, перечисленные в [[Protocol Orchestration]].

## 1.3.0 / Desktop 0.3.0 — предыдущая baseline

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
подписанного update/rollback, перехода с временного vendored Tauri/GTK `glib`
backport и закрытия platform release gates. Issue
[#31](https://github.com/mazurovn/mazzy-vpn/issues/31) закрыт точным
provenance-verified backport `glib`; RustSec, Dependabot и CodeQL release checks
прошли без suppressions.

Опубликованы [CLI/TUI `v1.3.0`](https://github.com/mazurovn/mazzy-vpn/releases/tag/v1.3.0)
и unsigned [Desktop `desktop-v0.3.0` preview](https://github.com/mazurovn/mazzy-vpn/releases/tag/desktop-v0.3.0).
Оба tag указывают на один проверенный release source; к artifacts приложены
SHA-256 manifests.

## 1.2.0 / Desktop 0.2.0 — предыдущая release line

Это предыдущие stable CLI/TUI и Linux Desktop preview.

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

Version 1.3.2 / Desktop 0.3.2 fixes legacy profile-cache compatibility and the
mixed `/usr/local` versus package `/usr/bin` upgrade conflict. It adds a
versioned 13-entry protocol catalog, credential-redacted URI detection, runtime
diagnostics and the read-only `protocols.list` API operation. The nine modern
proxy/transport entries remain gated adapter work rather than advertised
connections; see [[Protocol Orchestration]]. It also keeps the real local API
response half open after request EOF and tests that path through a delayed Unix
socket responder.

Version 1.3.0 / Desktop 0.3.0 expanded the Linux preview
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
signed update/rollback, migration away from the temporary vendored Tauri/GTK
`glib` backport and platform release gates are complete. [Issue
#31](https://github.com/mazurovn/mazzy-vpn/issues/31) is closed with the exact
provenance-verified backport; RustSec, Dependabot and CodeQL release checks pass
without suppressions.

Published pages: [CLI/TUI `v1.3.0`](https://github.com/mazurovn/mazzy-vpn/releases/tag/v1.3.0)
and the unsigned [Desktop `desktop-v0.3.0` preview](https://github.com/mazurovn/mazzy-vpn/releases/tag/desktop-v0.3.0).
Both tags identify the same audited source commit and include SHA-256 manifests.

Version 1.2.0 / Desktop 0.2.0 is the previous published release line.

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
