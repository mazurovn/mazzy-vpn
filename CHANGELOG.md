# Changelog

All notable changes to Mazzy VPN are documented here.

## Unreleased

No changes yet.

## 1.3.1 / Desktop 0.3.1 - 2026-08-01

- Fixed Desktop profile-cache compatibility with CLI 1.2 entries that do not
  contain `profile_id`; compatible opaque IDs are derived before strict
  validation instead of dropping all profiles.
- Distinguished an unavailable/invalid profile cache from a valid empty
  library in the WebView, fixing “24 profiles” on Dashboard versus “Profiles
  not found” on the Profiles screen.
- Added reversible package migration for trusted legacy Mazzy VPN commands in
  `/usr/local/bin`, preventing an older manual engine from shadowing the
  package-owned `/usr/bin/mazzy-vpn` after DEB/RPM update.
- Added a validated 13-entry protocol registry covering AmneziaWG, WireGuard,
  OpenVPN, L2TP/IPsec, VLESS/REALITY, Hysteria 2, Mieru, NaiveProxy, TUIC v5,
  Shadowsocks 2022, Trojan, AnyTLS and ShadowTLS v3.
- Added credential-redacted URI detection and runtime diagnostics through
  `mazzy-vpn protocols`; control bytes, oversized inputs and unknown schemes
  are rejected without reflecting the input.
- Added the read-only API v1 `protocols.list` operation and strict catalog
  schema. The nine new proxy/transport entries remain explicitly planned for
  connection until their TUN/runtime/platform integration gates pass.
- Defined deterministic orchestration weights, hard safety constraints and an
  AI-agent boundary that excludes credentials, generated shell commands and
  direct root-run engine configurations.
- Updated architecture, Desktop and platform roadmaps, capability parity,
  installation instructions, Wiki, project status, deep audit and all
  documentation screenshots.

## 1.3.0 / Desktop 0.3.0 - 2026-08-01

- Remediated `RUSTSEC-2024-0429` in the Tauri/GTK3 Linux graph by vendoring the
  crates.io `glib` 0.18.5 source and applying the exact reviewed upstream
  `VariantStrIter` soundness fix from gtk-rs commit
  `b5a4071e439bef2b5eea76c3aa25e5ae84839e34`.
- Added a release gate that verifies the original crate archive checksum,
  compares every vendored file byte-for-byte, proves the two-line backport is
  the only upstream source change and confirms Cargo resolves the local crate.
  Cargo-deny continues with an empty advisory ignore list.
- Included the vendored crate, provenance gate, `deny.toml` and pinned Rust
  toolchain in AppImage/DEB/RPM corresponding-source payloads; package audits
  now require and byte-compare the patched implementation and verifier.
- Added isolated Xvfb launch-smoke checks for the assembled AppImage and the
  binaries extracted from DEB/RPM; an immediate GTK/WebKit/resource crash now
  blocks both pull requests and tagged Desktop preview releases.
- Updated `event-listener` from 5.4.1 to 5.4.2 after the 2026-08-01 RustSec
  refresh found `RUSTSEC-2026-0221` through the `rfd`/`ashpd` dependency chain.
- Updated `serde_with` from 3.17.0 to 3.21.0 after the post-merge Dependabot
  scan found `GHSA-7gcf-g7xr-8hxj`; the fixed version bounds collection
  allocation before serializing attacker-controlled empty `KeyValueMap` items.
- Removed `CARGO_HOME`, `HOME` and arbitrary archive-path inputs from the glib
  provenance verifier. It now downloads only the pinned crates.io URL and
  validates the fixed SHA-256 before parsing, closing two CodeQL path-injection
  alerts without suppressing them.
- Removed an invalid shell-style fallback from `advisories.db-path`; cargo-deny
  now uses its documented portable, `CARGO_HOME`-aware database location instead
  of creating a literal `~/` directory inside the checkout.
- Removed the duplicate direct tray-to-`pkexec` command table. Tray lifecycle,
  service and recovery actions now use the same typed backend/local-API adapter
  as the main Desktop UI.
- Stopped using GNU `timeout --foreground` for Desktop helpers, so a timed-out
  command's descendants cannot retain captured pipes and hang the GUI. A Linux
  regression test covers a child that leaves a background process holding
  stdout/stderr open.
- Closed the local API probe/verify serialization descriptor before executing
  bounded workers. Timed-out descendants can no longer inherit `.probe.lock`
  or `.verify.lock` and cause a later request to fail spuriously as `busy`.
- Added the repeat architecture, security, code and release audit dated
  2026-08-01 and synchronized release/security documentation for issue #31.
- Added `mazzy-vpn verify [--speed] [--json]` and the protected read-only API
  query `tests.verify-egress`.
- Compare interface-bound and default IPv4 egress, require two distinct
  geolocation providers to report that exact IP and agree on country, inspect
  full-tunnel DNS configuration and flag a potential IPv6 leak.
- Keep the five-megabyte speed sample explicit and bounded; geo and speed
  services never run from the background health monitor.
- Made the health monitor detect two confirmed default-egress mismatches for
  profiles that declare a full tunnel while leaving split-tunnel profiles in
  automatic endpoint-only mode. Failure of both bounded connectivity observers
  still counts as a health failure; inability to compare only the default
  egress does not.
- Parse WireGuard/AmneziaWG `AllowedIPs` as a comma-separated route list, so
  compact valid forms such as `AllowedIPs=0.0.0.0/0` cannot silently bypass
  full-tunnel health enforcement.
- Parse optional `mazzy-name`, `mazzy-location` and `mazzy-country-code`
  metadata from profile comments, use NetworkManager's connection ID where
  available and fall back to the profile filename only when the protocol has
  no standard location field.
- Removed country/city inference from profile names. Expected-country
  comparison now requires explicit `mazzy-country-code`; actual country always
  comes from the interface-bound egress checks. Missing expected-country
  metadata produces a warning instead of a false `verified` location.
- Removed the silent `1.1.1.1` OpenVPN DNS fallback. DNS now comes from the VPN
  server/profile or an explicit `VPNCTL_OPENVPN_FALLBACK_DNS` administrator
  setting, preserving corporate split-DNS behavior.
- Strictly validate the root-owned Desktop profile cache before exposing it to
  the WebView and added a runtime hard-code boundary audit separating
  loopback-only documentation fixtures from operational paths.
- Strictly validate the privileged Desktop status cache and identify the
  active profile by opaque ID or exact config filename. Duplicate display
  names can no longer create a false active row or ambiguous API status.
- Reject duplicate profile identities and Unicode direction/zero-width
  spoofing markers across the CLI cache and Desktop boundary; root runtime also
  requires root-owned profile files under a complete root-owned,
  non-group/world-writable directory chain.
- Made both release scripts ignore all caller arguments; the tag-only audited
  Tauri wrapper always invokes one fixed builder and that builder always runs
  `tauri build`, closing a CodeQL high-severity user-controlled security-gate
  finding.
- Completed the package-owned corresponding-source payload with npm/Cargo
  manifests and locks, build/release scripts, Tauri capabilities, icons and
  the SVG logo; assembled-package audits require their presence and byte
  identity.
- Added Desktop profile sorting by ping/status/name, connect-fastest, an actual
  egress card, IP-hidden-by-default behavior and clickable event details.
- Expanded the tray with direct page navigation, actual egress verification,
  whole-list location ping, Doctor, refresh and explicit auto-connect/monitor
  controls. Tray probes now update the same structured UI state as window
  actions.
- Added strict Desktop verification parsing that rejects unknown fields, false
  verdicts, wrong IP families, provider-IP mismatch, duplicate providers and
  unexplained warnings.
- Bounded all Desktop compatibility processes and use absolute system paths for
  the privilege/timeout helpers. A timed-out mutation is reported as
  indeterminate and still remains a migration gate for the native service.
- Kept Linux-only egress/probe adapters out of macOS and Windows builds; preview
  bundles now return an explicit unsupported-platform error instead of trying
  Linux paths. The cross-platform UI validator also decodes Node output as
  UTF-8 rather than the Windows system code page.
- Updated AI-ready positioning, architecture, privacy, installation,
  cross-platform dependency strategy, Wiki and bilingual release documentation.
- Added bounded parallel whole-location endpoint checks to CLI/TUI and Desktop,
  with structured local API `tests.probe` results, per-profile active state,
  ICMP/TCP latency and sanitized JSON that never exposes an endpoint.
- Distinguished `unknown` from `unreachable`, so a UDP VPN server that resolves
  in DNS but blocks ICMP is not falsely reported as broken; full VPN
  authentication/routing still requires the transactional live test.
- Serialized API batch probes with a global lock, bounded each worker and the
  entire request deadline, and added the Desktop stylesheet to package-owned
  corresponding source payloads.
- Made Linux package executable modes independent of the builder umask:
  release packaging now normalizes runtime scripts to `0755` while assembling
  artifacts and restores the checkout modes afterwards.
- Published the language-neutral local API contract `1.0` with frontend-safe
  request, response and event envelopes, stable operation/error codes and
  explicit authorization, audit, deadline and rollback metadata.
- Added unprivileged `mazzy-vpn api-info --json` contract discovery and the
  equivalent read-only Desktop command.
- Bundled and installed the API manifest/schema with CLI and Desktop packages.
- Added a stdlib-only contract validator that prevents operation drift and
  forbidden keys, credentials, endpoints, configurations or unrestricted paths
  from entering the frontend schema.
- Added `versioned-local-api` to the machine-validated capability registry. The
  shared dispatcher and protected local service remain explicitly incomplete.
- Added a systemd socket-activated Linux API transport protected by
  `0660 root:mazzy-vpn`, with `status.get`, `profiles.list` and lifecycle
  connect/reconnect/disconnect handlers.
- Added opaque profile IDs, persistent idempotent action records, serialized
  mutations, bounded deadlines, sanitized root-only audit and desired-state
  rollback after lifecycle failures.
- Routed Desktop lifecycle operations through the protected API when installed,
  retaining the typed `pkexec` adapter as a compatibility fallback and for
  operation domains not migrated yet; an indeterminate transport is retried
  once with the identical request and action ID.
- Routed unprivileged CLI/TUI status, profile listing, dashboard and lifecycle
  actions through the same protected API, using opaque profile IDs only.
- Added bounded `socat` transport with response identity validation and
  same-action retry after a lost response, without an unsafe post-send `sudo`
  fallback; installers now bootstrap `socat` on supported Linux families.
- Preserved the published schema-v1 output of `status --json` and
  `profiles --json` regardless of socket availability; raw API v1 envelopes
  are opt-in through `--api-json`.
- Added crash reconciliation for orphaned API actions, bounded the completed
  action journal and rotated the sanitized root-only audit log.
- Rejected every terminal control character in imported or manually installed
  profile filenames before the name can reach CLI/TUI output.
- Bounded Desktop child-process stdout and stderr while draining both streams
  concurrently, preventing unbounded GUI memory growth from verbose helpers.
- Replaced floating GitHub-hosted OS generations and Rust `stable` with
  versioned runner labels and the declared Rust 1.88.0 toolchain.
- Restricted sanitized runtime status/profile caches to `root:mazzy-vpn`
  instead of exposing public IP and profile labels to every local account.
- Enter persistent recovery-only mode when an interrupted API mutation cannot
  be rolled back, requiring explicit administrator acknowledgement.
- Bounded local API requests by bytes before JSON parsing and added a finite
  receive deadline, preventing a socket client from forcing an unbounded Bash
  read.
- Pinned the immutable upstream commits behind the AmneziaWG tools and Go tags
  and stop installation if a tag resolves to different source.
- Made contract and capability validation reject ambiguous duplicate JSON keys.
- Extended `status.get` with optional safe runtime detail for CLI/TUI parity:
  desired mode, interface, handshake age, public IP, autostart, health monitor,
  failure count and fallback state; VPN endpoints remain forbidden.
- Restored that safe runtime detail in API-backed human status and TUI
  dashboard output without reading protected profiles.
- Localized new local-API client selection, transport, mutation and status
  messages in Russian, English, German, Chinese, Japanese and Korean.
- Made DEB/RPM own the engine, public runtime, systemd units/drop-ins, tmpfiles
  policy and completion under distribution-managed `/usr` paths, with base
  runtime dependencies and recommended protocol packages declared in metadata.
- Added idempotent package install/upgrade/remove scripts that verify the
  engine/API contract, activate services only on a running systemd host and
  deliberately preserve `/etc/vpnctl` profiles and `/var/lib/vpnctl` state.
- Made Desktop prefer the package-managed `/usr/bin/mazzy-vpn`; package repair
  now runs `doctor --fix`, including local-API group enrollment, instead of
  copying embedded files into `/usr/local`.
- Completed the AppImage embedded source set required by installer preflight,
  including Desktop config/UI, package lifecycle sources and capability docs.
- Added assembled AppImage/DEB/RPM payload, dependency, scriptlet,
  byte-identity and staged-bootstrap audits, and clean stale bundle outputs and
  the previously patched Tauri executable before every release build.

## 1.2.0 / Desktop 0.2.0 — 2026-07-27

- Bumped the shared engine to 1.2.0 and the Desktop Linux preview to 0.2.0.
- Expanded Desktop from a dashboard companion into a Linux control center with
  Dashboard, Profiles, Diagnostics and Settings screens.
- Added an About screen with product/engine versions, author, license, privacy,
  operational rules and security guidance, plus a bilingual privacy document.
- Added Android/iOS capability gates and a bilingual cross-platform roadmap for
  CLI/TUI, Linux, Windows, macOS and native mobile releases.
- Bundled the engine installer and required public runtime resources in Desktop
  packages, with installed/bundled version and dependency readiness checks plus
  an explicitly authorized install, update and repair workflow.
- Added sanitized profile discovery, search and selection; safe single/multiple
  file and folder import; connect, validate, probe, transactional test,
  test-all, emergency and profile removal actions.
- Added complete retained output for Doctor, approved Doctor fixes, offline/live
  self-tests and bounded service logs.
- Added Desktop controls for engine autostart and the independent recovery
  monitor.
- Added a typed Rust operation adapter with fixed argument construction, path
  and value validation, output sanitization and unit tests; no UI text is
  converted into a shell command.
- Added CLI `profiles --json`, `import-files`, independent `monitor on|off` and
  bounded `logs --lines`, with sanitized caches and regression coverage.
- Defined the standalone Desktop 1.0 architecture: one shared core and local
  API for independently usable CLI, TUI and Desktop clients.
- Added a machine-validated cross-surface capability registry and release gates
  that prevent a preview from being labeled as a complete Desktop client.
- Added bilingual Desktop roadmap, feature-parity matrix, PR checklist and a
  capability issue template.

## 1.1.0 — 2026-07-26

- Added English and Russian architecture documentation with component,
  connection, recovery and transactional rollback diagrams.
- Updated the installer and post-install checks to preserve the architecture
  documentation in `/usr/local/lib/mazzy-vpn/docs`.
- Added a sanitized machine-readable `status --json` cache that exposes no VPN
  endpoint, profile path, private key or configuration directive.
- Added a Tauri 2 Desktop Dashboard with six UI languages, connection health,
  default location, protocol, interface, handshake, public-IP privacy toggle,
  profile counts and activity state.
- Added a system tray menu for Quick Connect, Reconnect, Disconnect, Refresh,
  Self-diagnostics and Quit.
- Added functional Linux AppImage, DEB and RPM bundle builds. macOS and Windows
  bundles are explicitly labeled as UI previews until native VPN backends are
  implemented.
- Added pinned multi-platform GitHub Actions, Rust unit tests, Clippy checks and
  npm dependency auditing for Desktop.
- Added bilingual Desktop guides, updated all six project guides and added
  Dashboard images for the repository and Wiki.

## 1.0.0 — 2026-07-26

- Introduced the `mazzy-vpn` CLI/TUI with `vpnctl` and `mazzyvpn` aliases.
- Added a live dashboard with connection checks, selected location, default
  config, handshake, public IP, autostart and health-monitor state.
- Added `mazzy-vpn quick` for one-command connection through the saved default.
- Added installer and TUI language selection for RU, EN, DE, ZH, JA and KO.
- Added AmneziaWG, WireGuard, OpenVPN and NetworkManager L2TP/IPsec profiles.
- Added recursive profile-folder detection, validation and safe import.
- Added endpoint probes, doctor, diagnose, self-test and live test-all.
- Added transactional rollback, independent timeout guard, boot recovery and
  health watchdog.
- Hardened unattended recovery: immediate restart of an inactive desired
  service, reconnect after two confirmed traffic failures, dual HTTPS health
  probes and a roughly 20-second monitoring interval.
- Fixed the interactive TUI action-lock lifetime so an idle menu cannot block
  the independent health monitor.
- Added safe handling of external VPN fallback conflicts.
- Added validated AdGuard PID-file detection and cleanup when its status command
  cannot access the already-running tunnel session.
- Added automatic deduplication of stale WireGuard/AmneziaWG policy rules.
- Fixed systemd start-limit exhaustion during large profile test batches.
- Added explicit OpenVPN server-halt, authentication and
  `Too many connections` diagnostics with retryable service failures.
- Added six-language documentation, Bash completion, CI and public-repository
  secret/PII audit.
