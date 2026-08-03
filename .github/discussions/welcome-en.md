# Welcome to Mazzy VPN Discussions

Mazzy VPN is an open-source VPN profile and connection manager with CLI/TUI,
automatic recovery and a Desktop control center. The project does not sell VPN
access or provide servers: you use configurations from your provider or your
own infrastructure.

Author and maintainer: [Nik m (@mazurovn)](https://github.com/mazurovn).
License: GNU AGPL-3.0-or-later.

## Current status

- The current release source line is CLI/TUI 1.4.0 for Linux and the unsigned
  Desktop 0.4.0 preview. Linux Desktop is functional; Windows and macOS remain UI
  previews without native VPN backends. Issue #31 is closed with a verified
  `glib` backport and clean default-branch security checks.
- Windows/macOS — UI preview without a production VPN backend.
- Android/iOS — planned native clients; no working mobile packages yet.

The [Releases](https://github.com/mazurovn/mazzy-vpn/releases) page is the
source of truth: a version in `main`, the changelog or a PR is not released
without its corresponding tag and Release page.

## Where to post

- **Announcements** — releases and important maintainer updates.
- **Q&A** — installation, profiles, Doctor and troubleshooting questions.
- **Ideas** — feature proposals and UX improvements.
- **Polls** — votes that help order future user-facing features.
- **General** — architecture, documentation, localization and community.
- **Show and tell** — safe integrations without private configurations.

A reproducible defect belongs in
[Issues](https://github.com/mazurovn/mazzy-vpn/issues). Never publish a
vulnerability or secret; follow
[SECURITY.md](https://github.com/mazurovn/mazzy-vpn/blob/main/SECURITY.md).

## Rules

1. Include the OS, Mazzy VPN version, protocol and exact steps.
2. Remove keys, passwords, PSKs, endpoints, IPs and private paths from logs.
3. Do not publish working VPN profiles or another person's credentials.
4. Separate confirmed behavior from assumptions.
5. Be respectful and follow the laws in your jurisdiction.
6. Do not present modified builds as official releases; preserve authorship.

Poll results inform priority but cannot waive mandatory security, platform or
release gates.

Start with the [Wiki](https://github.com/mazurovn/mazzy-vpn/wiki),
[FAQ](https://github.com/mazurovn/mazzy-vpn/wiki/FAQ) and
[platform roadmap](https://github.com/mazurovn/mazzy-vpn/wiki/Platform-Roadmap).
