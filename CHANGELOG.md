# Changelog

All notable changes to Mazzy VPN are documented here.

## Unreleased

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
  versioned runner labels and the declared Rust 1.85.0 toolchain.
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
