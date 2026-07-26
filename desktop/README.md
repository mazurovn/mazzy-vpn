# Mazzy VPN Desktop / Рабочий стол

Copyright © 2026 Nik m ([@mazurovn](https://github.com/mazurovn)).
Licensed under `AGPL-3.0-or-later`.

![Mazzy VPN Desktop Dashboard](../docs/images/dashboard-connected-preview.png)

## English

Mazzy VPN Desktop is a Tauri 2 dashboard and system-tray companion for the
Mazzy VPN CLI engine.

- Linux: functional dashboard, safe status cache, quick connect, reconnect,
  disconnect, diagnostics, and a tray menu.
- macOS and Windows: UI preview builds only. They do **not** provide a working
  VPN tunnel until native Network Extension/launchd and Windows service/Wintun
  backends are implemented.
- The GUI never reads VPN profiles or keys. It reads the sanitized
  `/run/mazzy-vpn/status.json` cache. Privileged actions use a fixed command
  allowlist and `pkexec`; no arbitrary shell command is accepted.

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

Full guide: [docs/DESKTOP.en.md](../docs/DESKTOP.en.md).

## Русский

Mazzy VPN Desktop — Dashboard и системный tray на Tauri 2, работающие поверх
CLI-движка Mazzy VPN.

- Linux: рабочий Dashboard, безопасный cache состояния, быстрое подключение,
  переподключение, отключение, диагностика и tray-меню.
- macOS и Windows: только preview интерфейса. VPN-туннель на этих ОС появится
  после реализации нативных Network Extension/launchd и Windows service/Wintun.
- GUI не читает VPN-конфиги и ключи. Он использует очищенный cache
  `/run/mazzy-vpn/status.json`. Действия с повышенными правами ограничены
  фиксированным списком и запускаются через `pkexec`; произвольные shell-команды
  не принимаются.

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

Полная инструкция: [docs/DESKTOP.ru.md](../docs/DESKTOP.ru.md).
