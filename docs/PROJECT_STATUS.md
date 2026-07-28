# Mazzy VPN project status and handoff

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

Last synchronized: 2026-07-28.

This file is the persistent handoff after interrupted Codex sessions. GitHub
issues and release gates in [`capabilities.json`](capabilities.json) remain the
authoritative backlog; this page records the verified resumption point.

## Verified release baseline

- `main` resolves to `993f32b56ebeb7baca934f5d60b5998f32ff4bcd`
  after merging protected API PR #26, client PR #27 and Linux package-lifecycle
  PR #28.
- `v1.2.0` and `desktop-v0.2.0` remain at
  `0fe11a22b4e68f4c9106e4415ebaf5a2c281cb2c`.
- CLI/TUI 1.2.0 is the current stable release.
- Linux Desktop 0.2.0 is a functional control-center preview.
- Windows and macOS 0.2.0 artifacts are unsigned UI previews without native
  VPN backends.
- Android and iOS clients are planned and have no release artifacts.
- PR #22, API contract PR #25, protected API PR #26, client PR #27 and package
  PR #28 are merged. PR #28 passed all required Ubuntu 22.04, macOS 14 and
  Windows 2022 checks and was squash-merged as `993f32b`.
- The uncompromising code, security, packaging and cross-platform review is in
  [`AUDIT_2026-07-28.ru.md`](AUDIT_2026-07-28.ru.md). Its production blockers
  remain authoritative until replaced by newer evidence.

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

1. [#4 Linux Desktop package lifecycle](https://github.com/mazurovn/mazzy-vpn/issues/4)
2. [#5 shared core and versioned local API](https://github.com/mazurovn/mazzy-vpn/issues/5)
3. [#12 CLI/TUI parity and automation contract](https://github.com/mazurovn/mazzy-vpn/issues/12)
4. [#6 profiles](https://github.com/mazurovn/mazzy-vpn/issues/6),
   [#8 services/recovery](https://github.com/mazurovn/mazzy-vpn/issues/8) and
   [#9 connection modes](https://github.com/mazurovn/mazzy-vpn/issues/9)
5. [#7 Windows](https://github.com/mazurovn/mazzy-vpn/issues/7) and
   [#10 macOS](https://github.com/mazurovn/mazzy-vpn/issues/10)
6. [#13 Android](https://github.com/mazurovn/mazzy-vpn/issues/13) and
   [#14 iOS](https://github.com/mazurovn/mazzy-vpn/issues/14)
7. [#11 cross-platform production gates](https://github.com/mazurovn/mazzy-vpn/issues/11)

## Current implementation slice

Merged PR #25 completed the first incremental issue #5 slice:

- API contract `1.0` publishes 25 operations and 14 stable error codes;
- request, response, event and sanitized audit envelopes are defined;
- `mazzy-vpn api-info --json` exposes the installed manifest without root;
- Desktop embeds and exposes the same manifest through a read-only command;
- installer and Desktop bundles include the manifest, schema and documentation;
- CI validates operation IDs, authorization, audit, deadlines, rollback
  semantics and the frontend forbidden-field policy.

Merged PR #26 built the protected-service slice:

- systemd socket activation at `/run/mazzy-vpn/api-v1.sock`, protected by
  `0660 root:mazzy-vpn`;
- `status.get`, `profiles.list` and all three `lifecycle.*` operations;
- optional frontend-safe runtime status detail without VPN endpoints or paths;
- opaque profile IDs instead of paths or filenames in API requests;
- persistent action-ID idempotency, serialized mutations and bounded deadlines;
- crash recovery for orphaned actions, bounded persistent journals, rotating
  root-only sanitized audit and explicit desired-state rollback;
- Desktop lifecycle routing through the API with a compatibility fallback while
  the remaining operation domains still use the typed `pkexec` adapter;
- bounded, concurrently drained Desktop child-process output instead of
  accumulating an arbitrary amount of helper output in GUI memory;
- CI uses versioned Ubuntu 22.04/24.04, macOS 14 and Windows Server 2022 runner
  labels with the declared Rust 1.85.0 toolchain rather than `latest`/`stable`;
- status/profile caches are atomically published as `0640 root:mazzy-vpn`
  under a `0750 root:mazzy-vpn` runtime directory created consistently by
  systemd-tmpfiles;
- failed or missing crash-recovery snapshots persist a root-only marker that
  blocks later API mutations until explicit administrator acknowledgement;
- strict runtime JSON parsing, mandatory durable start audit, bounded query
  refresh and process-group termination close the ambiguity, unaudited
  execution and surviving-descendant faults found in the full audit.

Merged PR #27 added the client slice:

- unprivileged CLI/TUI `status`, `profiles`, `list`, dashboard, quick, connect,
  reconnect and disconnect routing through the same Unix socket;
- profile selection from sanitized display names and opaque profile IDs without
  reading root-only profile files;
- bounded `socat` transport, response identity validation and one automatic
  retry with the same action ID after a lost response;
- stable schema-v1 `status --json`/`profiles --json`, with raw API envelopes
  available only through explicit `--api-json`;
- API-backed human status/TUI detail for interface, public IP, handshake,
  autostart, monitor and fallback, while VPN endpoints remain hidden;
- localized local-API client errors and lifecycle feedback in all six CLI/TUI
  languages;
- byte-bounded API request reads, terminal-safe profile names and immutable
  commit verification for the AmneziaWG source fallback;
- no direct `sudo` fallback after a request may have been sent;
- automatic `socat` installation on Debian/Ubuntu, Fedora/RHEL, Arch and
  openSUSE;
- strict single-document responses, locale-independent byte limits, explicit
  query deadlines and visible action IDs for failed lifecycle outcomes;
- Desktop dependency diagnostics include `socat`; DEB requires `pkexec` and
  RPM requires `polkit`;
- tag releases run the Rust unit suite and Clippy before attaching bundles;
- the non-functional notifications preference was removed; the disabled
  Desktop control now labels the feature as unavailable in the preview.

Merged PR #28 completed the package-owned issue #4 slice:

- package-owned engine/runtime under `/usr/bin` and `/usr/lib/mazzy-vpn`;
- package-owned systemd units/drop-ins, tmpfiles policy and completion, with no
  package payload under `/usr/local` or `/etc/vpnctl`;
- base DEB/RPM runtime dependencies, including process-control tools, and
  weak/recommended protocol/TCP-probe packages;
- idempotent post-install/pre-remove/post-remove scripts that preserve profiles
  and state, verify the API contract and skip live activation in offline chroot;
- Desktop package-path preference and package-safe `doctor --fix` repair,
  including invoking-user local-API group enrollment;
- an assembled AppImage/DEB/RPM audit for payload, dependencies, scriptlets,
  byte-identity, embedded source completeness and staged AppImage bootstrap,
  plus stale-bundle and previously patched executable cleanup before releases.

The active `agent/location-health` issue #6 slice adds:

- bounded parallel `probe all` with configurable 1–8 workers and per-step
  DNS/ICMP/TCP timeouts;
- per-profile `reachable`/`unknown`/`unreachable`/`invalid`, measured
  ICMP/TCP latency and current active state without exposing endpoints;
- API v1 `tests.probe` with whole-request deadline, opaque profile IDs,
  structured batch summary and a global lock against concurrent amplification;
- CLI/TUI human output plus stable `--json`, with blocked ICMP for UDP kept as
  `unknown` instead of a false failure;
- Desktop whole-list/protocol checks and status/latency/active indicators on
  every profile row;
- complete package-owned Desktop CSS corresponding source.

Verified locally:

- shell regression suite: 68/68 on the location-health branch;
- Rust unit tests: 17/17;
- ShellCheck, Clippy, capability/API validators, public audit and gitleaks;
- `npm audit --audit-level=high` reported 0 known vulnerabilities on
  2026-07-28; Cargo and system-package advisory coverage is still absent;
- latest clean all-target release build produced AppImage, DEB and RPM without
  stale Tauri marker warnings; actual DEB/RPM metadata contains the declared
  base runtime, privilege helper, process tools, recommendations and
  auto-detected GTK/WebKit requirements;
- assembled AppImage/DEB/RPM engine and API content byte-match source;
  lifecycle scripts/drop-ins match audited sources, neither package owns
  `/etc/vpnctl`, AppImage contract/capability validators pass and its embedded
  installer completes an isolated staged install;
- clean-host install/upgrade/remove, rollback/fault injection and reproducible
  build checks remain required.

Do not mark issue #5 complete after this slice. The remaining API domains,
native caller identity, a full request/response deadline beyond the bounded
rollback completion grace, long-lived idempotency semantics and real-host
crash/concurrency tests remain separate acceptance criteria.

Do not mark issue #4 complete after this slice. AppImage trust/signing,
clean-device distro coverage, package rollback/fault/soak tests, complex legacy
migration and the production supply chain remain separate acceptance criteria.
