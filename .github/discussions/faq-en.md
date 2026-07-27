# FAQ: releases, installation, profiles, security and support

## Is this a VPN provider?

No. Mazzy VPN manages your profiles and connections; it does not issue an
account or provide a VPN server. You need a profile from your provider or your
own infrastructure.

## Which version is actually released?

A version is published only when it has a tag and a page in
[GitHub Releases](https://github.com/mazurovn/mazzy-vpn/releases). A version in
`main`, the changelog or a draft PR is not a release.

The current release line is CLI/TUI 1.2.0 and Desktop 0.2.0 preview, prepared
in [PR #22](https://github.com/mazurovn/mazzy-vpn/pull/22). Treat it as a
release candidate while either tag `v1.2.0`/`desktop-v0.2.0` or its
corresponding Release page is missing.

## Does Desktop require a separate CLI install?

Linux Desktop 0.2 bundles a compatible engine and installer, so no prior manual
CLI install is required. System changes run only after standard OS
authorization.

## Does a successful build mean the VPN works on every OS?

No. CI proves that the UI builds on Linux, macOS and Windows; it does not prove
that a platform VPN backend works. Linux Desktop 0.2 is a functional
control-center preview. Windows/macOS artifacts remain UI previews until native
backends, signed installers and platform integration tests are complete.

## What can I import?

AmneziaWG, WireGuard, OpenVPN and NetworkManager L2TP/IPsec. Backend support is
platform-specific; recognizing a file does not make that protocol functional
on Windows, macOS or mobile.

## Where are profiles stored?

On Linux they are root-protected under `/etc/vpnctl/profiles`. The frontend
receives sanitized metadata without keys, endpoints or private paths. Never
publish a working profile, private key, password, PSK, endpoint or complete log.

## What does Doctor check?

Versions, dependencies, profiles, systemd, desired state, VPN interface and
connectivity. Desktop 0.2 shows the complete result and bounded logs. Repair is
a separate action that explains system changes and requires confirmation.

## Are live tests safe?

A live test temporarily changes the active VPN route, checks the profile and
performs transactional rollback after success, failure, timeout or a signal.
Save important network work and read the confirmation before running it.

## Is there telemetry?

There is no mandatory account, analytics or telemetry. Details:
[PRIVACY.md](https://github.com/mazurovn/mazzy-vpn/blob/main/PRIVACY.md).

## Where can I download Windows, macOS, Android or iOS?

Windows/macOS artifacts are UI previews and must not be used to protect
traffic. Android/iOS are planned. Follow the
[release gates](https://github.com/mazurovn/mazzy-vpn/wiki/Releases-and-Roadmap).

## Why is `preview-release` marked `skipped` on a PR?

This is an expected workflow condition, not a failure. The release job runs
only for a `desktop-v*` tag; ordinary pushes and PRs run tests and produce
temporary artifacts. Publishing requires a separate tag after the release PR
is accepted.

## How do I propose and vote for a future feature?

Use the
[Ideas](https://github.com/mazurovn/mazzy-vpn/discussions/categories/ideas) and
[Polls](https://github.com/mazurovn/mazzy-vpn/discussions/categories/polls)
categories. Voting informs user-facing priority but cannot waive mandatory
security, platform or release gates.

## How do I get help or report a defect?

Run `mazzy-vpn version`, `mazzy-vpn doctor` and
`mazzy-vpn self-test --offline`. Open a Q&A with the OS, version, protocol,
steps and redacted output. Move a reproducible defect to
[Issues](https://github.com/mazurovn/mazzy-vpn/issues). Report vulnerabilities
privately through
[SECURITY.md](https://github.com/mazurovn/mazzy-vpn/blob/main/SECURITY.md).

Full FAQ:
https://github.com/mazurovn/mazzy-vpn/wiki/FAQ
