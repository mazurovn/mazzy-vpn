# Mazzy VPN project status and handoff

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

Last synchronized: 2026-08-02.

This file is the persistent handoff after interrupted Codex sessions. GitHub
issues and release gates in [`capabilities.json`](capabilities.json) remain the
authoritative backlog; this page records the verified resumption point.

## Verified release baseline

- Release tags `v1.3.2` and `desktop-v0.3.2` resolve to audited commit
  `80fe281f7a8791b83d9f7d13cb58c84a6e06f2a5`, after PR #40 and PR #42 were
  merged with all required default-branch checks green.
- The published stable release is CLI/TUI `v1.3.2`. The published Desktop
  prerelease is `desktop-v0.3.2`; its Linux AppImage, DEB and RPM plus macOS
  and Windows previews are covered by the published SHA-256 manifest.
- Issue #31 is closed. The release carries the provenance-verified upstream
  `glib` backport, `event-listener` 5.4.2 and `serde_with` 3.21.0; current
  RustSec, Dependabot and CodeQL release scans have no open alerts.
- Windows and macOS 0.3.2 artifacts are unsigned UI previews without native
  VPN backends. They must not be described as functional VPN clients.
- Android and iOS clients are planned and have no application source or
  release artifacts.
- Open backlog issues #4–#14 and #36–#39 remain incremental roadmap slices,
  not proof that the corresponding production gates are complete. Issues #31,
  #35 and #41 are closed after default-branch and release verification.
- The uncompromising code, security, packaging and cross-platform review started
  in [`AUDIT_2026-07-28.ru.md`](AUDIT_2026-07-28.ru.md). The release audit is
  [`AUDIT_2026-08-01.ru.md`](AUDIT_2026-08-01.ru.md); it
  records the issue #31 backport, the new `event-listener` advisory update,
  subprocess hang fix, tray-path consolidation and remaining production risks.
- The installed-system, profile-cache and protocol review is
  [`AUDIT_2026-08-01_PROTOCOLS.ru.md`](AUDIT_2026-08-01_PROTOCOLS.ru.md).
- The repeat planner, architecture, secret and release-state review is
  [`AUDIT_2026-08-02_PLANNER.ru.md`](AUDIT_2026-08-02_PLANNER.ru.md).

Published release links:

- <https://github.com/mazurovn/mazzy-vpn/releases/tag/v1.3.2>
- <https://github.com/mazurovn/mazzy-vpn/releases/tag/desktop-v0.3.2>

## Community and documentation baseline

- Repository description, RU/EN Welcome and FAQ discussions, platform roadmap,
  support template and feature polls are published.
- Public Wiki and repository `wiki/` sources are synchronized.
- Polls:
  - <https://github.com/mazurovn/mazzy-vpn/discussions/23>
  - <https://github.com/mazurovn/mazzy-vpn/discussions/24>

## Active backlog order

1. [#36 custom-server import](https://github.com/mazurovn/mazzy-vpn/issues/36),
   [#37 sing-box/TUN adapters](https://github.com/mazurovn/mazzy-vpn/issues/37),
   [#38 Mieru/Naive adapters](https://github.com/mazurovn/mazzy-vpn/issues/38)
   and [#39 AI planner](https://github.com/mazurovn/mazzy-vpn/issues/39)
2. [#4 Linux Desktop package lifecycle](https://github.com/mazurovn/mazzy-vpn/issues/4)
3. [#5 shared core and versioned local API](https://github.com/mazurovn/mazzy-vpn/issues/5)
4. [#12 CLI/TUI parity and automation contract](https://github.com/mazurovn/mazzy-vpn/issues/12)
5. [#6 profiles](https://github.com/mazurovn/mazzy-vpn/issues/6),
   [#8 services/recovery](https://github.com/mazurovn/mazzy-vpn/issues/8) and
   [#9 connection modes](https://github.com/mazurovn/mazzy-vpn/issues/9)
6. [#7 Windows](https://github.com/mazurovn/mazzy-vpn/issues/7) and
   [#10 macOS](https://github.com/mazurovn/mazzy-vpn/issues/10)
7. [#13 Android](https://github.com/mazurovn/mazzy-vpn/issues/13) and
   [#14 iOS](https://github.com/mazurovn/mazzy-vpn/issues/14)
8. [#11 cross-platform production gates](https://github.com/mazurovn/mazzy-vpn/issues/11)

## Protocol-orchestration work after 1.3.2

The 1.3.2 patch fixes three blockers found on an installed 0.2/1.2 system and
ships the protocol registry/detection foundation. Draft PR #43 adds the first
read-only planner slice; it is not part of the published 1.3.2 packages:

- Desktop accepts legacy profile-cache entries without `profile_id`, derives
  the same opaque ID as the CLI and distinguishes an unavailable cache from a
  genuinely empty profile list. This is the cause of the observed “24 profiles”
  dashboard versus “Profiles not found” view.
- DEB/RPM installation migrates recognized root-owned legacy copies in
  `/usr/local/bin/{mazzy-vpn,vpnctl,mazzyvpn}` to package-owned symlinks and
  preserves reversible backups. Unrelated or unsafe files are not replaced.
- A validated registry catalogs 13 protocols, including nine modern
  censorship-resistant proxy/transport families. The CLI and API expose only
  redacted catalog, runtime-diagnostic and URI-detection data. Connection stays
  truthfully limited to the four existing Linux backends until audited adapters
  and TUN integration pass their gates.
- Stable API v1 has 27 operations and 14 stable error codes. Draft PR #43 adds
  `planner.evaluate` as operation 28 without changing the API version.
- The real `socat` client preserves its response half after request EOF. A
  delayed Unix-socket integration test covers the installed systemd transport,
  which the previous fake dispatcher test did not model.
- The draft deterministic read-only planner applies backend-owned hard gates
  and versioned scoring to opaque profile IDs. Censorship/workload fit is now
  derived from the trusted catalog and workload instead of caller assertions.
  Its OpenVPN parser shares the request deadline, stale observed health is
  ignored, and rollback readiness is explicitly limited to protected storage.
  Remaining issue #39 work is history,
  authorized execution/failover and non-CLI integration. Platform work still
  includes pinned sing-box-family, Mieru and Naive/Cronet adapters;
  custom-server secret import; Linux TUN lifecycle; Windows service/Wintun;
  Android `VpnService`; and real integration, rollback and leak tests.
- The unreleased detector also classifies bounded sing-box/Xray, official Mieru
  and NaiveProxy JSON shapes. It rejects duplicate keys, ambiguous mixed
  protocols and secret reflection; classification is not import or execution.
- The capability matrix tracks macOS separately, but strict protocol registry
  v1 has only Linux, Windows and Android fields. Adding per-protocol macOS
  status requires a versioned registry v2; mutating the published v1 shape
  would break strict API/CLI consumers.
- Desktop profile-cache failures now preserve redacted missing/permission/shape
  reason codes instead of collapsing every failure into an unexplained empty
  list. The installed 0.3.2 cache was verified readable with 24 entries (9
  AmneziaWG and 15 OpenVPN).
- The regression suite now pins its API socket inside the temporary test root;
  an active installed daemon can no longer replace fixture profiles with the
  host's live profile catalog during a local run.
- Local patch verification is green: 81 Bash/end-to-end tests, 25 Rust tests,
  Clippy, ShellCheck, npm audit, cargo-deny, public leak audit and the unpacked
  AppImage/DEB/RPM lifecycle/source/GUI audit. All 12 documentation screenshots
  were regenerated at 1680×951 from the localhost-only RFC 5737 fixture.

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

The published 1.3.0 / Desktop 0.3.0 release line adds:

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

The final audited release source also fixes a local API query-deadline hang
found after an interrupted run: `tests.probe` and `tests.verify-egress` capture
bounded worker output through root-only temporary files, clean them on timeout
and close the serialization descriptor before spawning workers. The regression
suite asserts both cleanup and an immediately successful request after timeout.

Verified for the published release source on 2026-08-01:

- shell regression suite: 75/75 in one uninterrupted 2026-08-01 run, including
  API query deadline cleanup, Unicode profile-spoofing and runtime hard-code
  boundary checks;
- Rust unit tests: 24/24, including a descendant-pipe timeout regression;
- all 12 documentation screenshots were already regenerated on the current PR
  branch and were rechecked at 1680×951 with RFC 5737 preview data; the issue
  #31, subprocess and packaging fixes have no visual UI delta;
- ShellCheck, Clippy, capability/API validators, public audit and runtime
  hard-code audit;
- local API probe/verify workers close the parent serialization descriptor
  before execution; a regression test requires an immediate successful probe
  after a timed-out worker so descendants cannot retain the lock;
- `npm audit --audit-level=high` reports 0 known vulnerabilities. Issue #31 is
  remediated in release source by vendoring crates.io `glib` 0.18.5 and
  applying the exact upstream soundness fix. The provenance gate verifies the
  archive checksum and all source deltas before cargo-deny; `ignore = []`.
  The 2026-08-01 RustSec refresh also found `RUSTSEC-2026-0221` in
  `event-listener` 5.4.1, now updated to 5.4.2. The post-merge Dependabot scan
  then found `GHSA-7gcf-g7xr-8hxj` in `serde_with` 3.17.0; PR #33 updated it to
  3.21.0. Cargo-deny 0.20.2 reports `advisories ok`. System-package advisory
  scanning is still absent. Cargo-deny uses its documented `CARGO_HOME`-aware
  database default rather than a shell expression in TOML;
- crates.io checks on 2026-07-30 found no simple semver update that removes
  `glib` 0.18: `tauri` 2.11.5 and `webkit2gtk` 2.0.2 are current, `gtk` 0.18.2
  remains the GTK3 binding line, and `cargo update --dry-run` only offered minor
  unrelated updates plus `tray-icon` 0.24.2;
- PR #32 RustSec CI also found and fixed `quick-xml` and `time` advisories.
  Desktop MSRV was raised to Rust 1.88 and `Cargo.lock` now uses `plist` 1.10.0,
  `quick-xml` 0.41.0 and `time` 0.3.54; Rust tests and Clippy pass locally with
  `cargo +1.88.0`;
- GitHub private vulnerability reporting is enabled. Explicit CodeQL advanced
  setup runs `security-extended` analysis for Actions, JavaScript/TypeScript,
  Python and Rust with local threat sources. It excludes only the byte-verified
  `desktop/src-tauri/vendor/**` dependency snapshot; CI separately proves the
  crates.io checksum and exact two-line glib backport. The verifier downloads
  only the fixed crates.io URL and accepts no environment-controlled filesystem
  path, closing the two post-merge CodeQL path-injection findings without
  dismissal;
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
