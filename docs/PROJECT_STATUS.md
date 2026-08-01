# Mazzy VPN project status and handoff

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

Last synchronized: 2026-08-01.

This file is the persistent handoff after interrupted Codex sessions. GitHub
issues and release gates in [`capabilities.json`](capabilities.json) remain the
authoritative backlog; this page records the verified resumption point.

## Verified release baseline

- GitHub `main` and local `origin/main` resolve to
  `b47d0cd452264a355e4e2da48babba76a2180901` after merging PR #30 with the
  audited AI-ready diagnostics and Desktop 0.3 candidate.
- The published stable release is CLI/TUI `v1.2.0`. The published Desktop
  preview is `desktop-v0.2.0`.
- CLI/TUI 1.3.0 and Linux Desktop 0.3.0 remain release candidates in PR #32;
  neither tag nor GitHub Release page exists yet. The branch now carries the
  provenance-verified issue #31 `glib` backport and passes local RustSec checks
  without advisory suppressions. Publication still requires green PR checks,
  merge, the default-branch Dependabot scan and audited release artifacts.
- Windows and macOS 0.2.0 artifacts are unsigned UI previews without native
  VPN backends. They must not be described as functional VPN clients.
- Android and iOS clients are planned and have no application source or
  release artifacts.
- PR #32 is open as a draft. Open backlog issues are #4–#14 and #31; PRs
  #25–#30 are merged incremental slices, not proof that the corresponding
  production gates are complete. Issue #31 remains open until the default-
  branch dependency graph confirms the source-level remediation.
- The uncompromising code, security, packaging and cross-platform review started
  in [`AUDIT_2026-07-28.ru.md`](AUDIT_2026-07-28.ru.md). The latest PR #32
  resumption audit is [`AUDIT_2026-08-01.ru.md`](AUDIT_2026-08-01.ru.md); it
  records the issue #31 backport, the new `event-listener` advisory update,
  subprocess hang fix, tray-path consolidation and remaining production risks.

Published release links:

- <https://github.com/mazurovn/mazzy-vpn/releases/tag/v1.2.0>
- <https://github.com/mazurovn/mazzy-vpn/releases/tag/desktop-v0.2.0>

Candidate release links must not be advertised until both pages exist:
`v1.3.0` and `desktop-v0.3.0`.

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

- API contract `1.0` publishes 26 operations and 14 stable error codes;
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
  labels with the declared Rust 1.88.0 toolchain rather than `latest`/`stable`;
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

Merged PR #29 completed the first location-health issue #6 slice:

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

The merged 1.3.0 / Desktop 0.3.0 candidate adds:

- `mazzy-vpn verify [--speed] [--json]` and API query
  `tests.verify-egress`;
- conservative verification of interface-bound versus default IPv4 egress,
  two location providers tied to the same egress IP, country agreement,
  potential IPv6 leak and configured full-tunnel DNS;
- optional explicit five-megabyte speed sample, never a background transfer;
- Desktop verification card, profile sorting by ping/status/name, connect
  fastest, clickable event details and clearer service-toggle state;
- expanded tray navigation and actions for egress verification, whole-list
  ping, Doctor, refresh, auto-connect and monitor control;
- fixed `/usr/bin` package-engine preference in tray/status paths, absolute
  privilege-helper paths and bounded compatibility processes with explicit
  indeterminate mutation messaging;
- strict Desktop response parsing that rejects false verdicts, wrong IP
  families, provider-IP mismatch, duplicate providers and unexplained
  warnings;
- strict typed parsing of both privileged runtime caches and exact
  `profile_id`/filename active-profile identity, including fail-closed handling
  of duplicate display names.

The current `mazurovn/new-realise` continuation adds a local API query-deadline
fix after an interrupted audit found `tests.probe` could hang the regression
suite when the external deadline fired while nested probe workers still held the
captured stdout pipe. `tests.probe` and `tests.verify-egress` now capture bounded
worker output through root-only temporary files and clean them on timeout; the
regression suite asserts no `.probe-result.*` or `.verify-result.*` files remain.

Verified locally for the current candidate on 2026-08-01:

- shell regression suite: 74/74 in one uninterrupted 2026-08-01 run, including
  API query deadline cleanup, Unicode profile-spoofing and runtime hard-code
  boundary checks;
- Rust unit tests: 24/24, including a descendant-pipe timeout regression;
- all 12 documentation screenshots were already regenerated on the current PR
  branch and were rechecked at 1680×951 with RFC 5737 preview data; the issue
  #31, subprocess and packaging fixes have no visual UI delta;
- ShellCheck, Clippy, capability/API validators, public audit and runtime
  hard-code audit;
- `npm audit --audit-level=high` reports 0 known vulnerabilities. Issue #31 is
  remediated in candidate source by vendoring crates.io `glib` 0.18.5 and
  applying the exact upstream soundness fix. The provenance gate verifies the
  archive checksum and all source deltas before cargo-deny; `ignore = []`.
  The 2026-08-01 RustSec refresh also found `RUSTSEC-2026-0221` in
  `event-listener` 5.4.1, now updated to 5.4.2. Cargo-deny 0.20.2 reports
  `advisories ok`. System-package advisory scanning is still absent;
- crates.io checks on 2026-07-30 found no simple semver update that removes
  `glib` 0.18: `tauri` 2.11.5 and `webkit2gtk` 2.0.2 are current, `gtk` 0.18.2
  remains the GTK3 binding line, and `cargo update --dry-run` only offered minor
  unrelated updates plus `tray-icon` 0.24.2;
- PR #32 RustSec CI also found fixable `quick-xml` and `time` advisories.
  Desktop MSRV was raised to Rust 1.88 and `Cargo.lock` now uses `plist` 1.10.0,
  `quick-xml` 0.41.0 and `time` 0.3.54; Rust tests and Clippy pass locally with
  `cargo +1.88.0`;
- GitHub private vulnerability reporting is enabled. Explicit CodeQL advanced
  setup runs `security-extended` analysis for Actions, JavaScript/TypeScript,
  Python and Rust with local threat sources. It excludes only the byte-verified
  `desktop/src-tauri/vendor/**` dependency snapshot; CI separately proves the
  crates.io checksum and exact two-line glib backport. The high-severity release
  wrapper alert is fixed locally and still requires confirmation by PR scan;
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
