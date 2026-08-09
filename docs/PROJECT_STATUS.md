# Mazzy VPN project status and handoff

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

Last synchronized: 2026-08-09.

This file is the persistent handoff after interrupted Codex sessions. GitHub
issues and release gates in [`capabilities.json`](capabilities.json) remain the
authoritative published-product backlog. Until dedicated issues/evidence gates
are filed, R0/R1 architecture work is additionally governed by the IDs and DAG
in [`TARGET_ARCHITECTURE_2026-08-02.ru.md`](TARGET_ARCHITECTURE_2026-08-02.ru.md).
This page records the verified resumption point.

## Verified release baseline

- The current release line is CLI/TUI `v1.4.6` and Tauri-signed Desktop preview
  `desktop-v0.4.7`. Linux Desktop contains and starts its compatible engine
  before the first data load, while an existing package-managed or local CLI
  remains compatible. Desktop remains a prerelease because Windows and macOS
  do not yet have native VPN backends or operating-system code signing.
- Patch source `1.4.6` / Desktop `0.4.7` makes the Linux Desktop startup path
  self-contained: PolicyKit-authorized bootstrap installs or repairs the shared
  backend, starts the protected local API, refreshes read-only caches and grants
  the current GUI session a minimal per-user ACL. Package-managed engines are
  repaired in place and are never overwritten by the bundled CLI.
- Linux CLI/TUI is functional. Linux Desktop remains a preview; AppImage, DEB
  and RPM are functional control-center bundles. Installable updater artifacts
  are Tauri-signed, while package-repository and OS code-signing gates remain
  open.
- Issue #31 is closed. The release carries the provenance-verified upstream
  `glib` backport, `event-listener` 5.4.2 and `serde_with` 3.21.0; current
  RustSec, Dependabot and CodeQL release scans have no open alerts.
- A fresh 2026-08-02 `cargo audit` reports no known vulnerability, but does
  report 16 allowed `unmaintained` warnings in the existing Linux Tauri/GTK3,
  proc-macro and `unic` transitive graph. They are maintenance debt, not proof
  of an exploitable issue; migration off the legacy GTK3 graph remains a
  Desktop production gate.
- Windows and macOS 0.4.7 artifacts remain UI previews without native VPN
  backends or OS code signing. They must not be described as functional VPN
  clients; a Tauri updater signature does not change that boundary.
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
- The independent Claude/Codex/Qwen/Kimi release audit and remediation map is
  [`AUDIT_2026-08-03_RELEASE.ru.md`](AUDIT_2026-08-03_RELEASE.ru.md).
- The reverse agent-control plane is specified separately in
  [`AGENT_CONTROL_ARCHITECTURE.ru.md`](AGENT_CONTROL_ARCHITECTURE.ru.md).
  The v1 draft contract catalogs seven paths and five ingress channels. The
  Desktop 0.4 implements diagnostics-only Codex/Claude
  discovery and catalog status. Provider start/pair/stop was removed from both
  renderer and Tauri invoke after the P0 review; native command-bound approval,
  trusted executable resolution and process-tree containment are prerequisites
  for restoring any such authority. The seven Mazzy transport runtimes,
  first-party gateway, Web and Telegram remain planned and are not in published
  releases.
- The five-role repeat review and normative implementation plan are recorded in
  [`TARGET_ARCHITECTURE_2026-08-02.ru.md`](TARGET_ARCHITECTURE_2026-08-02.ru.md).
  It identifies the original P0 split-lock/rollback debt in the egress plane and incomplete
  authorization, crypto, ACK and key-lifecycle contracts in Agent Control. The
  plan uses reverse HTTPS/WSS first, LAN and iroh as measured accelerators, and
  a release-gated delivery DAG instead of a seven-transport rollout.
- Release 1.4 R0a converges API lifecycle, direct CLI, profile
  import/remove, recovery, health remediation, policy cleanup, `doctor --fix`, autostart and monitor on
  `$RUN_DIR/.mutation.lock`. API children validate the inherited lock inode and
  fail closed on an invalid descriptor. This removes the confirmed split-lock
  race, but it is not the planned `mazzy-vpnd` owner and does not prove
  restoration of routes, DNS, firewall or leak state.
  Implementation scope and migration criteria are recorded in
  [`R0_MUTATION_SINGLE_FLIGHT.ru.md`](R0_MUTATION_SINGLE_FLIGHT.ru.md).
- The P0 reboot slice adds a hardened root oneshot before `vpnctl.service`,
  health remediation and the API socket. It reconciles interrupted API actions
  under the same lock and restores a minimal nftables output/forward deny guard
  when a transition marker survived reboot. Rollback failure keeps the
  root-only marker, while API/service/health paths refuse mutation behind that
  marker and health retains the lock until systemd reports a terminal result.
- The source-level comparison and implementation decision is recorded in
  [`RESEARCH_AGENT_REMOTE_CONTROL_2026-08-02.ru.md`](RESEARCH_AGENT_REMOTE_CONTROL_2026-08-02.ru.md).
- The repeat managed-runtime and agent-control audit is
  [`AUDIT_2026-08-02_RUNTIME_AND_AGENTS.ru.md`](AUDIT_2026-08-02_RUNTIME_AND_AGENTS.ru.md).
  All nine modern entries now have closed validation and atomic secret-safe
  Linux import. Six also have a tested closed sing-box config renderer. No new
  connection lifecycle or platform backend is claimed.

Published release links:

- <https://github.com/mazurovn/mazzy-vpn/releases/tag/v1.4.6>
- <https://github.com/mazurovn/mazzy-vpn/releases/tag/desktop-v0.4.7>

## Community and documentation baseline

- Repository description, RU/EN Welcome and FAQ discussions, platform roadmap,
  support template and feature polls are published.
- Repository `wiki/` sources and the public GitHub Wiki contain the synchronized
  Agent Control, R0a, target architecture and 1.4/0.4 publication updates.
- Polls:
  - <https://github.com/mazurovn/mazzy-vpn/discussions/23>
  - <https://github.com/mazurovn/mazzy-vpn/discussions/24>

## Active backlog order

R0 status synchronized with the target architecture:

| ID | Status | Remaining release gate |
|---|---|---|
| R0-1 | contained | Desktop provider actions removed; process-group/job-object deadline required before re-enabling |
| R0-2 | contained | Desktop provider actions removed; native command-bound approval required before re-enabling |
| R0-3 | contained | diagnostics do not execute discovered agents; trusted executable provenance required before re-enabling |
| R0-4 | partial | shared R0a lock is implemented; daemon ownership, common audit IDs and provider fixtures remain |
| R0-5 | partial | all UI/docs/registry readiness labels must remain evidence-backed |

1. Keep Desktop Agent Control diagnostics-only while implementing
   process-group/job-object deadlines, native command-bound confirmation,
   cross-platform executable provenance, mutation single-flight/audit and
   unambiguous readiness labels behind a new review gate.
2. Converge egress mutations from the transitional shared R0a lock on one
   `mazzy-vpnd` owner. Rollback still restores desired intent without proving
   routes/DNS/interfaces/firewall. This remains P0 consistency debt before a
   shared-daemon GA claim.
3. Bootstrap the separate `mazzy-agent-control` monorepo: complete protocol and
   conformance testkit, unprivileged `mazzy-agentd`, typed Codex/Claude/ACP
   adapters, pairing/E2EE, reverse HTTPS/WSS and read-only Desktop/Web slice.
   LAN and iroh follow as measured accelerators; Telegram follows the
   interactive security gate. A first-party Mazzy network runtime does not yet
   exist.
4. Finish [#36 custom-server import](https://github.com/mazurovn/mazzy-vpn/issues/36)
   beyond the implemented neutral profile/import foundation,
   [#37 sing-box/TUN adapters](https://github.com/mazurovn/mazzy-vpn/issues/37),
   [#38 Mieru/Naive adapters](https://github.com/mazurovn/mazzy-vpn/issues/38)
   and [#39 AI planner](https://github.com/mazurovn/mazzy-vpn/issues/39)
5. [#4 Linux Desktop package lifecycle](https://github.com/mazurovn/mazzy-vpn/issues/4)
6. [#5 shared core and versioned local API](https://github.com/mazurovn/mazzy-vpn/issues/5)
7. [#12 CLI/TUI parity and automation contract](https://github.com/mazurovn/mazzy-vpn/issues/12)
8. [#6 profiles](https://github.com/mazurovn/mazzy-vpn/issues/6),
   [#8 services/recovery](https://github.com/mazurovn/mazzy-vpn/issues/8) and
   [#9 connection modes](https://github.com/mazurovn/mazzy-vpn/issues/9)
9. [#7 Windows](https://github.com/mazurovn/mazzy-vpn/issues/7) and
   [#10 macOS](https://github.com/mazurovn/mazzy-vpn/issues/10)
10. [#13 Android](https://github.com/mazurovn/mazzy-vpn/issues/13) and
   [#14 iOS](https://github.com/mazurovn/mazzy-vpn/issues/14)
11. [#11 cross-platform production gates](https://github.com/mazurovn/mazzy-vpn/issues/11)

## Protocol-orchestration release 1.4.0

Release 1.4.0 preserves the 1.3.2 installed-system fixes and ships the first
read-only planner, managed-profile and runtime-adapter foundation:

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
- All nine modern protocols now accept only the closed
  `managed-profile.schema.json` and can be atomically imported under a
  protocol/profile ID with root-only permissions. This is `partial` import,
  not vendor-format conversion or connection support.
- The packaged adapter renders a fixed sing-box 1.13.12 TUN/DNS/routing graph
  for six protocols. `runtime/v1/adapter-registry.json` keeps lifecycle and
  integration tests planned. Mieru/Naive need sidecar supervisors; ShadowTLS
  needs a typed inner proxy chain.
- Stable API v1 has 29 operations and 14 stable error codes, including
  `planner.evaluate`, without changing the API version.
- The real `socat` client preserves its response half after request EOF. A
  delayed Unix-socket integration test covers the installed systemd transport,
  which the previous fake dispatcher test did not model.
- The deterministic read-only planner applies backend-owned hard gates
  and versioned scoring to opaque profile IDs. Censorship/workload fit is now
  derived from the trusted catalog and workload instead of caller assertions.
  Its OpenVPN parser shares the request deadline, stale observed health is
  ignored, and rollback readiness is explicitly limited to protected storage.
  Remaining issue #39 work is history,
  authorized execution/failover and non-CLI integration. Platform work still
  includes pinned sing-box-family, Mieru and Naive/Cronet adapters;
  custom-server secret import; Linux TUN lifecycle; Windows service/Wintun;
  Android `VpnService`; and real integration, rollback and leak tests.
- The release detector also classifies bounded sing-box/Xray, official Mieru
  and NaiveProxy JSON shapes. It rejects duplicate keys, ambiguous mixed
  protocols and secret reflection; classification alone is not import or
  execution. Managed import is a separate closed-format operation.
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
- Local release verification includes 103 Bash/end-to-end tests plus the
  managed adapter, Agent Control, Rust, ShellCheck, package and UI gates.
  GitHub CI remains authoritative for Clippy, cargo-deny, npm audit, CodeQL and
  the cross-platform Desktop build matrix.
  Fourteen documentation screenshots remain unchanged; two new signed-update
  dialog captures are 1680×951. All 16 use the localhost-only RFC 5737 fixture;
  the Agent Control pair remains 1680×975 and the other 14 are 1680×951.

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
