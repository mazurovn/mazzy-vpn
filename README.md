<p align="center">
  <img src="assets/mazzy-vpn-logo.svg" width="190" alt="Mazzy VPN logo">
</p>

<h1 align="center">Mazzy VPN</h1>

<p align="center">
  A safe, transactional VPN manager for Linux.<br>
  Created and maintained by <a href="https://github.com/mazurovn">Nik m (@mazurovn)</a>.
</p>

<p align="center">
  <a href="LICENSE"><img alt="License: AGPL-3.0-or-later" src="https://img.shields.io/badge/license-AGPL--3.0--or--later-8f7dff"></a>
  <a href="https://github.com/mazurovn/mazzy-vpn/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/mazurovn/mazzy-vpn/actions/workflows/ci.yml/badge.svg"></a>
  <img alt="Linux" src="https://img.shields.io/badge/platform-Linux-65e7ff">
  <img alt="Version 1.2.0" src="https://img.shields.io/badge/version-1.2.0-ef70ff">
</p>

Mazzy VPN is a Linux VPN manager with a guided terminal UI and an
automation-friendly CLI. It manages AmneziaWG, WireGuard, OpenVPN and
NetworkManager L2TP/IPsec profiles through one command, validates imported
files before root uses them, checks every endpoint, performs transactional
live tests and restores the previously working connection after a failure.

[Русский](README.ru.md) · [English](README.en.md) ·
[Deutsch](README.de.md) · [中文](README.zh.md) ·
[日本語](README.ja.md) · [한국어](README.ko.md)

[Architecture diagrams](docs/ARCHITECTURE.en.md) ·
[Схемы архитектуры](docs/ARCHITECTURE.ru.md) ·
[Desktop Dashboard](docs/DESKTOP.en.md) ·
[Desktop Dashboard на русском](docs/DESKTOP.ru.md) ·
[Standalone Desktop 1.0 plan](docs/DESKTOP_ROADMAP.en.md) ·
[План самостоятельного Desktop 1.0](docs/DESKTOP_ROADMAP.ru.md) ·
[Capability parity](docs/FEATURE_PARITY.md) ·
[Project Wiki](https://github.com/mazurovn/mazzy-vpn/wiki)

![Mazzy VPN Desktop Dashboard](docs/images/dashboard-connected-preview.png)

- Live CLI/TUI dashboard with connection checks, selected location, default
  config, handshake, public IP, autostart and health-monitor state.
- Tauri 2 Desktop control center and tray. Desktop 0.2 bundles the Linux engine
  and installer, checks dependencies, imports and manages profiles, exposes
  transactional tests, Doctor output, logs and service settings, and can repair
  its own engine after explicit authorization. AppImage, DEB and RPM remain
  previews until the versioned local API and all Desktop 1.0 release gates are
  complete; macOS and Windows still require native VPN backends.
- Quick connection through the saved default profile: `mazzy-vpn quick`.
- Two-layer unattended recovery: systemd restarts an exited process, while an
  independent 20-second health monitor reconnects a stalled tunnel.
- Interface and installer languages: Russian, English, German, Chinese,
  Japanese and Korean.
- [Security policy](SECURITY.md)
- [Contributing](CONTRIBUTING.md)

Quick start:

```bash
sudo ./install.sh
mazzy-vpn
mazzy-vpn dashboard
mazzy-vpn quick
mazzy-vpn self-test
```

Primary command: `mazzy-vpn`. Compatibility aliases installed automatically:
`vpnctl` and `mazzyvpn`.

The installer runs syntax checks, bundled regression tests, systemd validation,
profile validation and an offline post-install self-test. Use
`--config-dir /path/to/configs` to recursively auto-detect and import supported
VPN profiles. Choose the interface language at the beginning or pass
`--lang ru|en|de|zh|ja|ko`. Live tunnel testing is opt-in with `--live-test`.

Real keys and personal VPN profiles are intentionally excluded from Git.
Publish templates only; keep operational profiles in `/etc/vpnctl/profiles`.

## Author and license

Mazzy VPN was created by [Nik m (@mazurovn)](https://github.com/mazurovn).
Copyright © 2026 Nik m.

The project is open source under
[GNU Affero General Public License v3.0 or later](LICENSE). Modified versions
may be made and distributed, but the AGPL source-sharing obligations must be
preserved. The Mazzy VPN name and original authorship notices must not be
misrepresented.
