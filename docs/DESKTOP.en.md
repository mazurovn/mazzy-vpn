# Mazzy VPN Desktop: Dashboard and tray

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

[Русский](DESKTOP.ru.md) · [Architecture](ARCHITECTURE.en.md) ·
[Standalone Desktop 1.0 plan](DESKTOP_ROADMAP.en.md) ·
[Capability parity](FEATURE_PARITY.md) · [Project home](../README.md)

![Mazzy VPN Desktop Dashboard — preview data](images/dashboard-connected-preview.png)

Mazzy VPN Desktop is a compact Tauri 2 application built on the validated
`mazzy-vpn` CLI engine. It presents connection health in one window and keeps
the main actions available from the system tray.

> **0.1 status:** this is a functional Linux dashboard/companion, not yet a
> standalone VPN client. It requires the installed CLI engine. Desktop 1.0 will
> bundle the shared core/bootstrap and reach full parity without a separate CLI
> installation; it remains preview until the
> [release gates](FEATURE_PARITY.md) pass.

## Platform status

| Platform | Status | Bundles |
|---|---|---|
| Linux x86_64 | Functional Dashboard and tray for an installed Mazzy VPN CLI | AppImage, DEB, RPM |
| macOS | UI preview; VPN backend is not implemented yet | app, DMG |
| Windows | UI preview; VPN backend is not implemented yet | MSI, NSIS EXE |

Do not use the macOS or Windows previews as traffic-protection tools. Complete
support requires native Network Extension/launchd and Windows service/Wintun
backends, code signing and platform-specific integration tests.

## Dashboard data

- systemd service and real VPN-interface state;
- public Internet access through the selected tunnel;
- selected location/default profile, protocol and interface;
- latest WireGuard/AmneziaWG handshake age;
- public VPN IP with a local privacy toggle;
- autostart, health monitor and external fallback state;
- profile counts for every protocol;
- local UI-session activity;
- cache freshness, so stale data cannot look current.

The UI reads `/run/mazzy-vpn/status.json` every five seconds. The root
CLI/health monitor atomically recreates that file. It contains no endpoint,
profile path, private key, username, password or configuration directive.

## Window and tray actions

| Action | CLI command |
|---|---|
| Quick Connect | `mazzy-vpn quick` |
| Reconnect | `mazzy-vpn reconnect` |
| Disconnect | `mazzy-vpn disconnect` |
| Self-diagnostics | `mazzy-vpn doctor` |
| Refresh Status | `mazzy-vpn _refresh-dashboard-cache` |

The GUI never constructs a shell string. Its Rust backend accepts an enum and
maps it to a fixed argument array. On Linux, state-changing actions go through
system `pkexec`, so the OS may show its standard administrator prompt. Closing
the window hides the application to the tray; use **Quit Mazzy VPN** for a full
exit.

On Linux, open the tray context menu with the right mouse button. Plain-click
events depend on the desktop environment.

## Linux installation

Install and verify the CLI first:

```bash
git clone https://github.com/mazurovn/mazzy-vpn.git
cd mazzy-vpn
sudo ./install.sh --yes
sudo mazzy-vpn diagnose
```

Then install one Desktop bundle from the Releases page.

DEB:

```bash
sudo apt install "./Mazzy VPN Desktop_0.1.0_amd64.deb"
```

RPM:

```bash
sudo dnf install "./Mazzy VPN Desktop-0.1.0-1.x86_64.rpm"
```

AppImage:

```bash
chmod +x "Mazzy VPN Desktop_0.1.0_amd64.AppImage"
./"Mazzy VPN Desktop_0.1.0_amd64.AppImage"
```

Replace the sample version with the downloaded file name. Verify the published
SHA-256 checksum.

## Languages

Dashboard supports Russian, English, German, Chinese, Japanese and Korean. The
selection stays in local WebView storage and is not sent to a server. Change
the CLI language independently:

```bash
mazzy-vpn language ru
mazzy-vpn language en
```

## Troubleshooting

If Dashboard has no data:

```bash
sudo mazzy-vpn _refresh-dashboard-cache
mazzy-vpn status --json
systemctl status vpnctl-health.timer
```

If an action fails, check `pkexec` and then the CLI:

```bash
command -v pkexec
sudo mazzy-vpn doctor
sudo mazzy-vpn diagnose
```

If no tray icon appears, verify that the desktop environment supports
StatusNotifier/AppIndicator. The VPN continues to run under systemd even when
Desktop is closed.

## Build from source

```bash
cd desktop
npm ci
npm run test
cargo clippy --manifest-path src-tauri/Cargo.toml --all-targets -- -D warnings
npm run build:release
```

Linux builds place AppImage, DEB and RPM bundles under
`desktop/src-tauri/target/release/bundle/`. npm and Cargo dependencies are
locked. The release command remaps the builder's local home directory out of
Rust diagnostic strings. CI builds each OS on its matching GitHub runner; the
release workflow creates a preview release while artifacts remain unsigned.
