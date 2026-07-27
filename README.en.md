# Mazzy VPN — English guide

Created and maintained by
[Nik m (@mazurovn)](https://github.com/mazurovn).

Mazzy VPN is a Linux VPN manager with an interactive terminal UI and an
automation-friendly CLI. It combines AmneziaWG, WireGuard, OpenVPN and
NetworkManager L2TP/IPsec, auto-detects and safely imports profiles, checks
every endpoint, maintains one managed tunnel, runs live tests as transactions
and rolls back to the previously active managed or external VPN after failure.

The primary command is `mazzy-vpn`. The installer also creates the compatibility
aliases `vpnctl` and `mazzyvpn`.

[Architecture and operation diagrams](docs/ARCHITECTURE.en.md) ·
[Архитектура на русском](docs/ARCHITECTURE.ru.md)

## Install

```bash
cd /path/to/mazzy-vpn
sudo ./install.sh --yes
mazzy-vpn
```

Interactive installation starts with a choice of Russian, English, German,
Chinese, Japanese or Korean. For unattended installation:

```bash
sudo ./install.sh --lang en --yes
```

Useful installer options:

```bash
sudo ./install.sh --config-dir ~/VPN-configs
sudo ./install.sh --config-dir ~/VPN-configs --force-configs
sudo ./install.sh --no-deps
sudo ./install.sh --live-test
./install.sh --dry-run --yes
./install.sh --destdir /tmp/mazzy-stage --no-deps
```

By default the installer:

1. verifies the source package and Bash syntax;
2. runs the bundled regression suite;
3. installs supported dependencies;
4. installs `mazzy-vpn`, aliases, Bash completion, profiles and systemd units;
5. validates systemd units and every installed profile;
6. runs `mazzy-vpn self-test --offline`.

`--live-test` is intentionally opt-in because it connects every valid profile
one by one. Each live test has a timeout and a rollback transaction.

## Import a folder of VPN profiles

Create a clean folder structure:

```bash
mazzy-vpn init-config-dir ~/MazzyConfigs
```

This creates:

```text
MazzyConfigs/
├── amneziawg/
├── wireguard/
├── openvpn/
└── l2tp/
```

Profiles may also be mixed or nested. Scan without changing the system:

```bash
mazzy-vpn import-dir ~/VPN-configs --dry-run
```

Import all recognized and valid profiles:

```bash
sudo mazzy-vpn import-dir ~/VPN-configs
```

Mazzy VPN recognizes:

- AmneziaWG and WireGuard `*.conf` files by their sections and directives;
- OpenVPN `*.ovpn` or `*.conf` files by their OpenVPN directives;
- NetworkManager L2TP `*.nmconnection` files by their VPN service type.

Every recognized file is validated before copying. Unsafe executable hooks,
OpenVPN plugins, nested OpenVPN configs and incomplete profiles are rejected.
Installed profiles receive mode `600`. Existing files with different content
are not overwritten unless `--force` is explicitly supplied.

The same scan/import/create actions are available from TUI menu item 15.

## Main commands

```bash
mazzy-vpn                         # interactive TUI
sudo mazzy-vpn quick              # connect the saved default profile
mazzy-vpn dashboard               # connection checks in one view
mazzy-vpn list
sudo mazzy-vpn connect amneziawg 1
sudo mazzy-vpn disconnect
sudo mazzy-vpn reconnect
mazzy-vpn status
mazzy-vpn status --json
mazzy-vpn diagnose
mazzy-vpn validate all
mazzy-vpn probe all --timeout 3
mazzy-vpn self-test
mazzy-vpn self-test --offline
sudo mazzy-vpn test amneziawg 1 --timeout 45
sudo mazzy-vpn test-all openvpn --timeout 30
sudo mazzy-vpn emergency --timeout 20
sudo mazzy-vpn autostart on
mazzy-vpn logs
mazzy-vpn language
mazzy-vpn language de
```

The dashboard appears above the interactive menu and is also available as
`mazzy-vpn dashboard`. It shows the live tunnel and Internet check, selected
location, protocol, default config, interface, handshake age, public IP,
autostart, health monitor, fallback and profile counts.

`mazzy-vpn quick` connects the saved default config without another selection.
If no default exists, the TUI asks for a profile and saves it as the default.
Change the language immediately with menu item 16 or
`mazzy-vpn language ru|en|de|zh|ja|ko`.

## Desktop control center and tray

![Mazzy VPN Desktop Dashboard](docs/images/dashboard-connected-preview.png)

The Tauri Desktop 0.2 Linux preview provides Dashboard, Profiles, Diagnostics
and Settings screens plus a system tray. It bundles the compatible engine and
installer, checks installed versions and dependencies, and can install, update
or repair the engine after explicit authorization. Profile file/folder import,
search, connect, validation, probes, transactional tests, Doctor fixes,
self-tests, bounded logs, autostart and recovery-monitor controls are available
without first installing the CLI by hand. Packages are available as AppImage,
DEB and RPM.

Desktop 0.2 is a functional Linux control-center preview, not the final
standalone Desktop 1.0 product. The remaining gates include a shared versioned
local service API, complete mode/fallback settings, full six-language coverage
for the new screens, signed update/rollback, and native macOS and Windows VPN
backends. Development is synchronized through the
[Desktop 1.0 plan](docs/DESKTOP_ROADMAP.en.md) and
[capability matrix](docs/FEATURE_PARITY.md).

The GUI never reads VPN files after import and never reads keys. It consumes
sanitized status/profile caches without endpoints or paths, and sends only
typed operations from a fixed allowlist to the privileged engine. See the
[Desktop guide](docs/DESKTOP.en.md) for installation, limitations and
troubleshooting.

## Safe live tests and rollback

`test` starts one profile in an isolated transaction, verifies its interface and
public Internet access through that interface, and restores the previous state
unless `--keep` is specified:

```bash
sudo mazzy-vpn test amneziawg 1 --timeout 45
sudo mazzy-vpn test openvpn 2 --timeout 60 --keep
```

Rollback runs after success, failure, timeout or termination signal. An
independent transient systemd guard and boot recovery unit cover a killed CLI or
reboot. When configured, Mazzy VPN also detects legacy `wg0` scripts and
AdGuard VPN. It stops active external tunnels before a managed test and restores
only the one that was active before the transaction.

If an OpenVPN server reports `Too many connections`, the test identifies the
server-side session limit, restores the original VPN immediately and stops a
batch run instead of producing a long chain of misleading timeouts.

## Validation and diagnostics

- `validate all` checks format, required fields, endpoint values, permissions
  and unsafe directives for every installed profile.
- `probe all` checks endpoint DNS and ICMP; TCP OpenVPN ports are checked when
  `nc` is installed.
- `diagnose` checks the default route, DNS, service, tunnel interface,
  WireGuard/AmneziaWG handshake and public Internet access through the tunnel.
- `doctor` checks dependencies, AmneziaWG backend, L2TP/IPsec stack, fallback
  handlers, systemd units, profiles and saved state.
- `self-test` combines validation, endpoint probes and doctor.
- `self-test --live` additionally tests every tunnel with rollback.

## Autostart and recovery

```bash
sudo mazzy-vpn autostart on
sudo mazzy-vpn autostart off
```

The existing internal unit names remain `vpnctl.service`,
`vpnctl-health.timer` and `vpnctl-test-recovery.service` for upgrade
compatibility. Their descriptions and executable use the Mazzy VPN brand and
`/usr/local/bin/mazzy-vpn`.

The service restarts after any unexpected process exit. Independently, the
health timer checks the desired state, service, VPN interface and real HTTPS
access through that interface about every 20 seconds. An inactive desired
service is started immediately; two consecutive traffic failures trigger a
reconnect. `doctor --fix` enables the monitor and repairs autostart when a valid
default profile has `DESIRED=up`.

The runtime cleans orphan `wg-quick` policy rules only when no WireGuard or
AmneziaWG interface is active; unrelated IPsec rule 220 is left untouched.

## Logs

```bash
mazzy-vpn logs
mazzy-vpn logs -f
systemctl status vpnctl.service
systemctl status vpnctl-health.timer
```

Private keys and long AmneziaWG concealment parameters are not written to the
extended test log.

## OpenVPN `Too many connections`

Some providers count AmneziaWG and OpenVPN against the same account session
limit. A server may continue to count a recently stopped WireGuard-like tunnel
for a short time. Its OpenVPN response is:

```text
Halt command was pushed by server ('Too many connections')
```

TLS and interface setup may already have completed at this point; the server is
rejecting the data channel. Mazzy VPN converts this otherwise successful
OpenVPN process exit into a retryable service failure, and systemd retries
after 15 seconds. A transactional `test` or `test-all` restores the previous
tunnel and reports the server rejection directly.

Stop other devices using the same VPN account or allow the previous server
session to expire, then retry:

```bash
sudo mazzy-vpn connect openvpn "Example Server"
mazzy-vpn diagnose
```

Do not manually start AdGuard during `test-all`: it is an external fallback and
therefore intentionally changes the state that the transaction must restore.

## Publishing

Never push real private keys or personal VPN profiles to GitHub. The repository
`.gitignore` excludes `conf/`, release archives and checksums. Publish safe
templates only and keep operational profiles under `/etc/vpnctl/profiles` with
mode `600`.

## Author and license

Mazzy VPN was created by [Nik m (@mazurovn)](https://github.com/mazurovn).
Copyright © 2026 Nik m.

The project is open source under the
[GNU Affero General Public License v3.0 or later](LICENSE). You may study, use,
modify and distribute it, provided that derivative versions comply with the
AGPL and preserve the original authorship notices. See
[AUTHORS.md](AUTHORS.md) and [CONTRIBUTING.md](CONTRIBUTING.md).
