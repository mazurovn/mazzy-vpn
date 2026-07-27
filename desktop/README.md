# Mazzy VPN Desktop / Рабочий стол

Copyright © 2026 Nik m ([@mazurovn](https://github.com/mazurovn)).
Licensed under `AGPL-3.0-or-later`.

![Mazzy VPN Desktop Dashboard](../docs/images/dashboard-connected-preview.png)

## English

Mazzy VPN Desktop is a Tauri 2 Linux control center and system-tray client for
the shared Mazzy VPN engine.

- **Current status:** 0.2 Linux preview. It bundles the compatible engine
  installer, checks versions and dependencies, and no longer requires the user
  to install the CLI first.
- Linux: dashboard, profile/file/folder import, location selection,
  connect/disconnect, active-connection diagnostics, validate/probe/live-test
  with rollback, full Doctor output, bounded logs, service settings, dependency
  repair and tray.
- macOS and Windows: UI preview builds only. They do **not** provide a working
  VPN tunnel until native Network Extension/launchd and Windows service/Wintun
  backends are implemented.
- The GUI never reads VPN keys. It reads sanitized status/profile caches.
  Privileged actions use typed requests, validated arguments and `pkexec`; no
  arbitrary shell command is accepted.
- Desktop 1.0 remains gated on fallback-policy UI, complete six-language
  coverage for new screens and a versioned daemon API.

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
[docs/FEATURE_PARITY.md](../docs/FEATURE_PARITY.md).

## Русский

Mazzy VPN Desktop — Linux Control Center и системный tray на Tauri 2,
работающие с общим движком Mazzy VPN.

- **Текущий статус:** Linux preview 0.2. В пакет включён совместимый installer
  движка; предварительно устанавливать CLI пользователю больше не нужно.
- Linux: Dashboard, импорт файлов/папок, выбор локации, подключение,
  диагностика активного соединения, validate/probe/live-test с rollback, полный
  вывод Doctor, журнал, службы, проверка зависимостей и автоматическое
  исправление, а также экран «О программе» с версиями, автором, лицензией,
  приватностью и правилами.
- macOS и Windows: только preview интерфейса. VPN-туннель на этих ОС появится
  после реализации нативных Network Extension/launchd и Windows service/Wintun.
- GUI не читает VPN-ключи. Он использует очищенные cache состояния и профилей.
  Привилегированные действия принимаются как typed-запросы с проверенными
  аргументами и запускаются через `pkexec`; shell-строки не строятся.
- Gate Desktop 1.0 остаётся закрыт до реализации fallback-policy UI, полного
  перевода новых экранов на шесть языков и versioned daemon API.

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
[docs/FEATURE_PARITY.md](../docs/FEATURE_PARITY.md).
