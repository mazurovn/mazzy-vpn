# Mazzy VPN project status and handoff

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

Last synchronized: 2026-07-27.

This file is the persistent handoff after interrupted Codex sessions. GitHub
issues and release gates in [`capabilities.json`](capabilities.json) remain the
authoritative backlog; this page records the verified resumption point.

## Verified release baseline

- `main`, `v1.2.0` and `desktop-v0.2.0` resolve to
  `0fe11a22b4e68f4c9106e4415ebaf5a2c281cb2c`.
- CLI/TUI 1.2.0 is the current stable release.
- Linux Desktop 0.2.0 is a functional control-center preview.
- Windows and macOS 0.2.0 artifacts are unsigned UI previews without native
  VPN backends.
- Android and iOS clients are planned and have no release artifacts.
- PR #22 is merged and there are no open pull requests at this checkpoint.

Release links:

- <https://github.com/mazurovn/mazzy-vpn/releases/tag/v1.2.0>
- <https://github.com/mazurovn/mazzy-vpn/releases/tag/desktop-v0.2.0>

## Community and documentation baseline

- Repository description, RU/EN Welcome and FAQ discussions, platform roadmap,
  support template and feature polls are published.
- Public Wiki and repository `wiki/` sources are synchronized.
- Polls:
  - <https://github.com/mazurovn/mazzy-vpn/discussions/23>
  - <https://github.com/mazurovn/mazzy-vpn/discussions/24>

## Active backlog order

1. [#5 shared core and versioned local API](https://github.com/mazurovn/mazzy-vpn/issues/5)
2. [#12 CLI/TUI parity and automation contract](https://github.com/mazurovn/mazzy-vpn/issues/12)
3. [#6 profiles](https://github.com/mazurovn/mazzy-vpn/issues/6),
   [#8 services/recovery](https://github.com/mazurovn/mazzy-vpn/issues/8) and
   [#9 connection modes](https://github.com/mazurovn/mazzy-vpn/issues/9)
4. [#4 Linux Desktop 1.0](https://github.com/mazurovn/mazzy-vpn/issues/4)
5. [#7 Windows](https://github.com/mazurovn/mazzy-vpn/issues/7) and
   [#10 macOS](https://github.com/mazurovn/mazzy-vpn/issues/10)
6. [#13 Android](https://github.com/mazurovn/mazzy-vpn/issues/13) and
   [#14 iOS](https://github.com/mazurovn/mazzy-vpn/issues/14)
7. [#11 cross-platform production gates](https://github.com/mazurovn/mazzy-vpn/issues/11)

## Current implementation slice

The active branch completes the first incremental issue #5 slice:

- API contract `1.0` publishes 25 operations and 14 stable error codes;
- request, response, event and sanitized audit envelopes are defined;
- `mazzy-vpn api-info --json` exposes the installed manifest without root;
- Desktop embeds and exposes the same manifest through a read-only command;
- installer and Desktop bundles include the manifest, schema and documentation;
- CI validates operation IDs, authorization, audit, deadlines, rollback
  semantics and the frontend forbidden-field policy.

Verified locally:

- shell regression suite: 52/52;
- Rust unit tests: 8/8;
- ShellCheck, Clippy, capability/API validators, public audit and gitleaks;
- npm audit: 0 vulnerabilities;
- staged installer reads the installed contract through the staged CLI.

Do not mark issue #5 complete after this slice. CLI/TUI migration, protected
local transport, state migration, crash/concurrency testing and the final
daemon contract remain separate acceptance criteria.
