# Mazzy VPN Desktop / Рабочий стол

Copyright © 2026 Nik m ([@mazurovn](https://github.com/mazurovn)).
Licensed under `AGPL-3.0-or-later`.

![Mazzy VPN Desktop Dashboard](../docs/images/dashboard-en.png)

## English

Mazzy VPN Desktop is a Tauri 2 Linux control center and system-tray client for
the shared Mazzy VPN engine.

- **Current status:** 0.4 Linux preview. It bundles the compatible 1.4 engine
  and installer, starts them before the first data load, checks versions and
  dependencies, and no longer requires the user to install or launch the CLI
  first. Privileged first-run setup is authorized by the native PolicyKit
  dialog.
- Linux: dashboard, profile/file/folder import, location selection and
  whole-list reachability/latency/active checks, connect/disconnect,
  active-connection diagnostics, validate/probe/live-test
  with rollback, full Doctor output, bounded logs, service settings, dependency
  repair and tray.
- macOS and Windows: UI preview builds only. They do **not** provide a working
  VPN tunnel until native Network Extension/launchd and Windows service/Wintun
  backends are implemented.
- The GUI never reads VPN keys. It reads sanitized status/profile caches.
  Privileged actions use typed requests, validated arguments and `pkexec`; no
  arbitrary shell command is accepted.
- Desktop checks the fixed GitHub update feed on startup by default, but always
  asks in a modal before download or installation. Tauri signatures are
  verified in the Rust backend. AppImage supports in-app install; DEB/RPM opens
  the matching release to preserve package-manager ownership.
- Desktop is single-instance: launching it again focuses the existing window
  instead of creating another WebView and tray icon.
- Desktop writes a privacy-bounded, `1,000,000`-byte rotating lifecycle log to the
  platform application log directory. It records operation names and outcomes,
  never profile contents, endpoints, credentials or public IP addresses.
- Desktop 1.0 remains gated on native peer identity, migration of the remaining
  privileged domains to a shared service/API, complete six-language coverage,
  a fixed Tauri/GTK RustSec dependency graph and signed clean-device release
  gates.

Development:

```bash
cd desktop
npm ci
npm run test
npm run dev
```

Linux packages:

```bash
npm run build:release
```

Tauri produces AppImage, DEB, and RPM bundles on supported Linux build hosts.
The release command remaps the builder's home path out of Rust diagnostics.

Full guide: [docs/DESKTOP.en.md](../docs/DESKTOP.en.md). The standalone product
architecture and parity gates are in
[docs/DESKTOP_ROADMAP.en.md](../docs/DESKTOP_ROADMAP.en.md) and
[docs/FEATURE_PARITY.md](../docs/FEATURE_PARITY.md). Signed-update behavior and
failure boundaries are documented in
[docs/DESKTOP_UPDATES.md](../docs/DESKTOP_UPDATES.md).

## Русский

Mazzy VPN Desktop — Linux Control Center и системный tray на Tauri 2,
работающие с общим движком Mazzy VPN.

- **Текущий статус:** Linux preview 0.4. В пакет включён совместимый installer
  и CLI-движок. Desktop запускает их до первой загрузки данных; предварительно
  устанавливать или запускать CLI пользователю больше не нужно. Системные
  изменения первого запуска подтверждаются в штатном диалоге PolicyKit.
- Linux: Dashboard, импорт файлов/папок, выбор локации, массовая проверка
  reachability/latency/active, подключение, диагностика активного соединения,
  validate/probe/live-test с rollback, полный
  вывод Doctor, журнал, службы, проверка зависимостей и автоматическое
  исправление, а также экран «О программе» с версиями, автором, лицензией,
  приватностью и правилами.
- macOS и Windows: только preview интерфейса. VPN-туннель на этих ОС появится
  после реализации нативных Network Extension/launchd и Windows service/Wintun.
- GUI не читает VPN-ключи. Он использует очищенные cache состояния и профилей.
  Привилегированные действия принимаются как typed-запросы с проверенными
  аргументами и запускаются через `pkexec`; shell-строки не строятся.
- Desktop по умолчанию проверяет фиксированный GitHub feed при запуске, но
  всегда показывает диалог перед загрузкой или установкой. Tauri-подпись
  проверяется в Rust backend. AppImage обновляется внутри приложения, а
  DEB/RPM открывает соответствующий release, не обходя package manager.
- Desktop работает в одном экземпляре: повторный запуск показывает уже
  открытое окно, не создавая второй WebView и tray.
- Desktop пишет ограниченный одним мегабайтом ротационный lifecycle-журнал в
  системный каталог логов приложения. В нём есть только имена и результаты
  операций, без профилей, endpoint, credentials и публичных IP.
- Gate Desktop 1.0 остаётся закрыт до native peer identity, переноса остальных
  privileged domains в общий service/API, полного перевода на шесть языков,
  исправления Tauri/GTK RustSec dependency graph и подписанных clean-device
  release gates.

Разработка и сборка:

```bash
cd desktop
npm ci
npm run test
npm run dev
npm run build:release
```

Release-команда удаляет локальный домашний путь сборщика из диагностических
строк Rust.

Полная инструкция: [docs/DESKTOP.ru.md](../docs/DESKTOP.ru.md). Архитектура
самостоятельного продукта и release gates:
[docs/DESKTOP_ROADMAP.ru.md](../docs/DESKTOP_ROADMAP.ru.md) и
[docs/FEATURE_PARITY.md](../docs/FEATURE_PARITY.md). Trust boundary, границы
ошибок и ограничения ручного rollback обновлений описаны в
[docs/DESKTOP_UPDATES.md](../docs/DESKTOP_UPDATES.md).
