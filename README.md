<p align="center">
  <img src="assets/mazzy-vpn-logo.svg" width="190" alt="Mazzy VPN logo">
</p>

<h1 align="center">Mazzy VPN</h1>

<p align="center">
  AI-ready, self-healing VPN client for open access to AI, the web, video and work.<br>
  Created and maintained by <a href="https://github.com/mazurovn">Nik m (@mazurovn)</a>.
</p>

<p align="center">
  <a href="LICENSE"><img alt="License: PolyForm Noncommercial 1.0.0" src="https://img.shields.io/badge/license-PolyForm%20Noncommercial%201.0.0-8f7dff"></a>
  <a href="COMMERCIAL.md"><img alt="Commercial license available" src="https://img.shields.io/badge/commercial-license%20available-ef70ff"></a>
  <a href="https://github.com/mazurovn/mazzy-vpn/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/mazurovn/mazzy-vpn/actions/workflows/ci.yml/badge.svg"></a>
  <img alt="Linux" src="https://img.shields.io/badge/platform-Linux-65e7ff">
  <a href="https://github.com/mazurovn/mazzy-vpn/releases/tag/v2.4.4"><img alt="Release 2.4.4" src="https://img.shields.io/badge/release-2.4.4-ef70ff"></a>
</p>

Mazzy VPN is a source-available AI-ready VPN client for Linux with Desktop, tray,
guided terminal UI and an automation-friendly CLI. It is designed for stable
AI sessions and agents, learning, video, the open web, remote work and
corporate portals. It manages your AmneziaWG, WireGuard, OpenVPN and
NetworkManager L2TP/IPsec profiles, measures every location, verifies the
actual VPN egress/location/DNS/IPv6 path, performs transactional live tests and
restores the previously working connection after failures.

Mazzy VPN is a client and control plane, not a hosted VPN subscription. Bring
profiles from a VPN provider or your organization. No account, telemetry or
project-hosted server is required.

The current release line is the **Go native CLI**
[v2.4.0](https://github.com/mazurovn/mazzy-vpn/releases/tag/v2.4.0) (single
static binary, embedded engine, 6 UI languages). The legacy bash
[CLI/TUI 1.4.7](https://github.com/mazurovn/mazzy-vpn/releases/tag/v1.4.7) and
[Desktop 0.4.8](https://github.com/mazurovn/mazzy-vpn/releases/tag/desktop-v0.4.8)
remain available.
A version is published only when its linked tag and GitHub Release page exist.
Desktop 0.4.8 is a DEB-first release with consent-gated repair:
Linux provides a functional control center. AppImage, RPM, Windows, macOS and
Android artifacts are not part of this release line.
Issue #31 is closed with the reviewed upstream `glib` backport, an exact
source-provenance gate and a clean default-branch dependency/security scan.

The versioned catalog now covers 13 protocols. Linux connection backends remain
implemented for AmneziaWG, WireGuard, OpenVPN and L2TP/IPsec. VLESS/REALITY,
Hysteria 2, Mieru, NaiveProxy, TUIC v5, Shadowsocks 2022, Trojan, AnyTLS and
ShadowTLS v3 have a validated registry, redacted capability API and safe share
URI/JSON classification where unambiguous. All nine have a closed neutral
profile validator and atomic secret-safe Linux import; six have a closed
sing-box config renderer. Their connection adapters remain explicitly
`planned`, not advertised as working tunnels.

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
[Cross-platform roadmap](docs/PLATFORM_ROADMAP.en.md) ·
[Protocol and AI orchestration](docs/PROTOCOL_ORCHESTRATION.en.md) ·
[Reverse Agent Control architecture](docs/AGENT_CONTROL_ARCHITECTURE.en.md) ·
[Target architecture and delivery DAG (RU)](docs/TARGET_ARCHITECTURE_2026-08-02.ru.md) ·
[Release 1.4 deep audit (RU)](docs/AUDIT_2026-08-03_RELEASE.ru.md) ·
[R0a mutation single-flight specification (RU)](docs/R0_MUTATION_SINGLE_FLIGHT.ru.md) ·
[Agent remote-control research (RU)](docs/RESEARCH_AGENT_REMOTE_CONTROL_2026-08-02.ru.md) ·
[Project Wiki](https://github.com/mazurovn/mazzy-vpn/wiki)

![Mazzy VPN Desktop Dashboard in English](docs/images/dashboard-en.png)

- Live CLI/TUI dashboard with connection checks, selected location, default
  config, handshake, public IP, autostart and health-monitor state.
- Tauri 2 Desktop control center and expanded tray. Desktop 0.4 bundles the Linux engine
  and installer, checks dependencies, imports and manages profiles, exposes
  sortable whole-list ping, fastest-location selection, actual egress/location
  verification, transactional tests, clickable events, Doctor output, logs
  and clear service settings, and starts or repairs its embedded engine through
  native PolicyKit authorization without a separately installed or running CLI.
  An existing package-managed or `/usr/local` CLI remains compatible and shares
  the same protected local API and state. DEB remains
  previews until the versioned local API and all Desktop 1.0 release gates are
  complete; macOS and Windows still require native VPN backends.
- Quick connection through the saved default profile: `mazzy-vpn quick`.
- Two-layer unattended recovery: systemd restarts an exited process, while an
  independent roughly one-minute health monitor reconnects a stalled tunnel.
- Interface and installer languages: Russian, English, German, Chinese,
  Japanese and Korean.
- [Security policy](SECURITY.md)
- [Privacy principles](PRIVACY.md)
- [Contributing](CONTRIBUTING.md)

Quick start:

```bash
sudo ./install.sh
mazzy-vpn
mazzy-vpn dashboard
mazzy-vpn quick
mazzy-vpn verify
mazzy-vpn probe all --timeout 3 --jobs 4
mazzy-vpn self-test
mazzy-vpn protocols list --json
mazzy-vpn protocols diagnose --json
mazzy-vpn protocols adapters --json
mazzy-vpn protocols managed-validate --stdin --json < profile.json
mazzy-vpn protocols managed-import profile.json --dry-run --json
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

## Screenshots

<p align="center">
  <img src="docs/assets/cli/status.png" width="32%" alt="mazzy-vpn up --best + status">
  <img src="docs/assets/cli/test.png" width="32%" alt="mazzy-vpn test (live server ranking)">
  <img src="docs/assets/cli/doctor.png" width="32%" alt="mazzy-vpn doctor + verify">
</p>

<p align="center"><sub>Connect to the best live zone · rank servers by latency · host &amp; config diagnostics. (Screens use documentation-only data: RFC 5737 IP <code>203.0.113.42</code>, generic zone names.)</sub></p>

> Managing Mazzy VPN from inside [pi](https://pi.dev)? See the
> [`@mazurovn/mazzy-vpn-pi`](https://www.npmjs.com/package/@mazurovn/mazzy-vpn-pi)
> extension.

## Go native CLI (v2)

The v2 line introduces a **single statically-linked Go binary** (`mazzy-vpn`)
that **embeds the VPN engine** (AmneziaWG/WireGuard via a vendored
`amneziawg-go`). It needs **no** `awg`, `awg-quick`, `wg`, `jq` or any external
VPN backend, builds with `CGO_ENABLED=0`, and is fully portable across Linux
hosts. Sources live under [`core/`](core/) (the reusable engine, 35 packages)
and [`mazzy-vpn-cli/`](mazzy-vpn-cli/) (the CLI + full-screen TUI).

Highlights:

- **AI-native control plane** — self-authenticating Ed25519 identities, trust
  pairing (anti-impersonation), signed time-bounded grants, deny-by-default
  egress channels for agents/harnesses.
- **Six UI languages** — English (default), Russian, German, Chinese, Japanese,
  Korean. `mazzy-vpn language` opens a selection menu; the language is resolved
  from settings/`MAZZY_LANG`/`LC_*`/`LANG` with **no hardcoding**.
- **Stealth & mimicry** — detection score, timezone/locale alignment to the
  egress country, DNS-over-TLS in-country checks, AI-provider reachability.
- **Safety** — fail-closed IPv6 leak guard armed before the interface exists,
  `fwmark = routing table` parity, real egress verification, atomic durable
  state, single-writer lock.
- **Quality gates** — tests + `-race` + `vet` + `staticcheck` + `gofmt` +
  `govulncheck` (0 vulnerabilities) + parser/control fuzzing, all in CI.

```bash
sudo ./install.sh                    # static binary, no compilation/apt
mazzy-vpn language --list            # en/ru/de/zh/ja/ko (current marked *)
mazzy-vpn best                       # connect the cleanest live server
mazzy-vpn providers                  # AI provider reachability
mazzy-vpn control id                 # this node's control-plane identity
```

See the [Go CLI wiki page](https://github.com/mazurovn/mazzy-vpn/wiki/Go-CLI)
for the full command reference in all six languages.

## Author and license

Mazzy VPN was created by [Nik m (@mazurovn)](https://github.com/mazurovn).
Copyright © 2026 Nik m.

The project is **source-available** under the
[PolyForm Noncommercial License 1.0.0](LICENSE). It is **free for personal,
research/scientific, educational, and charitable (noncommercial) use**.
**Commercial use requires a separate commercial license** — see
[COMMERCIAL.md](COMMERCIAL.md). An **Enterprise edition** (centralized fleet
management, SSO/SCIM, audit, SLA support) is planned on top of this core.

This is source-available software, not an OSI-approved open-source license. The
Mazzy VPN name and original authorship notices must not be misrepresented.
