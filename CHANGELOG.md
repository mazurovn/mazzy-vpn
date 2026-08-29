# Changelog

All notable changes to Mazzy VPN are documented here.

## CLI 2.4.2 - 2026-08-29

### Added
- **`sudo mazzy-vpn reconnect [NAME]`** (TUI: `R`, menu: `3`) — the missing
  "reconnect NOW" button: always drops the current tunnel (even one the daemon
  believes is healthy) and reconnects to the best PROVEN-working zone via a new
  daemon `reconnect` intent, clearing every backoff. With no daemon it starts
  one. `--best` re-ranks inside the daemon with live cooldowns + egress
  history.
- **EGRESS liveness column** in `test`, the TUI zone picker and the menu zone
  chooser: `✔ routes` (recently carried real traffic) · `✖ no-route ×N`
  (recently accepted ping/handshake but forwarded nothing) · `· untested`.
  Ping alone kept promoting fast-but-dead servers; the honest signal is now
  visible everywhere zones are listed, and ranking is biased by it.
- **`probe` feeds selection and restores the VPN**: every deep-test verdict is
  persisted to the shared egress history (WORKS → prefer, DEAD/NO-EGRESS →
  sink), a running daemon is stopped automatically for exclusive access and
  RESTARTED automatically after the sweep (original zone if it proved usable,
  else the best proven zone) — a diagnostic can no longer leave the machine
  offline.

### Fixed
- **Egress-history split-brain**: the reachability cache lived in per-user
  config (`os.UserConfigDir`), so the root daemon wrote `/root/.config` while
  the user's UI read `~/.config` — the picker never saw what the daemon had
  learned. It now lives in ONE shared root-written, world-readable file
  (`/var/lib/mazzy-vpn/reachcache.json`, labels + timestamps only), with the
  legacy file read once as a seed.
- **`up NAME` vs a running daemon**: the daemon holds the mutation lock for its
  lifetime, so `up` after `disconnect` failed with "another operation is in
  progress" and the host stayed offline (observed live). `up` now forwards the
  zone as an intent to the running daemon — which also resumes a paused one —
  and waits for confirmed egress before reporting success.
- **TUI zone list overflow**: the picker now scrolls in a window that follows
  the cursor (`↑/↓ N more` markers) instead of overflowing small terminals.

### Earlier in 2.4.2 (2026-08-26)
- **`sudo mazzy-vpn probe [NAME|--all]`** — HARD deep connectivity test: for
  each zone it validates the config, probes the endpoint + path MTU through the
  uplink, then ACTUALLY brings the tunnel up and measures real egress plus the
  WireGuard tx/rx byte counters. It distinguishes WORKS · SERVER NOT ROUTING
  (tunnel up, tx≫rx, no egress — the server accepts the handshake but does not
  forward internet traffic) · DEAD (no handshake) · BAD CONFIG. Needs exclusive
  tunnel access, so it requires root and refuses while a daemon is running.
- **`sudo mazzy-vpn disarm`** (menu: `!`) — the HARD reset: kills the daemon
  (SIGTERM→SIGKILL), removes ALL our firewall/routing state including the
  fail-closed kill-switch, reverts per-link DNS and flushes resolver caches,
  clears runtime status files, then VERIFIES plain internet actually works and
  reports honestly. The escape hatch for "everything is blocked".
- **Diagnose sees self-inflicted blocks**: an armed kill-switch with no
  working tunnel, leftover mazzy nftables tables, stale policy-routing rules,
  and resolv.conf still pointing at a dead tunnel are now detected (nft
  inspection under sudo) and diagnosed as the top root cause with the disarm
  fix — previously diagnose was blind to exactly the worst incident class.
- **`sudo mazzy-vpn doctor --heal`** — active connection rescuer with an
  escalation ladder: already protected → noop; daemon mid-reconnect → 90s
  grace; paused → resume and verify; unhealthy → SIGTERM (SIGKILL fallback) →
  full `recover` → fresh daemon on the best zone, each rung verified by a real
  egress check. Granted passwordless via `trust` scoped to exactly
  `doctor --heal` — designed for agents/cron keeping the link 24/7.
- **Honest latency graph**: the daemon pings the VPN server (ICMP via the
  physical uplink, asynchronously) each healthy tick; the dashboard graph,
  p50/p95 and jitter now use the real RTT instead of the HTTPS probe duration
  (which bundled TCP+TLS+HTTP and overstated latency several-fold).
- **systemd watchdog**: the daemon speaks sd_notify (READY/WATCHDOG/STOPPING
  from the 5s heartbeat pulse); the packaged unit is now `Type=notify` with
  `WatchdogSec=90` and `Restart=always` — systemd revives the daemon even
  after a hard wedge or OOM kill. Under systemd, `--background` self-forking
  is disabled and a foreign daemon owning the VPN fails the unit start
  honestly instead of hanging it.
- **Legacy cleanup in `recover`**: removes the defunct
  `/etc/modules-load.d/amneziawg.conf` (content-checked) that errored on every
  boot; `--purge-legacy` also deletes leftover awg/awg-quick binaries.
- Dashboard link-health row (menu + TUI): WireGuard **handshake age**, tunnel
  **traffic ↓/↑**, session **loss %**, **egress country/city** and the
  **stealth score** — all published by the daemon into the heartbeat, so the
  unprivileged dashboard shows real link facts without extra probes.

### Fixed
- **Duplicate policy-routing rules**: zone switches / unclean teardowns could
  leave multiple copies of the fwmark and suppress_prefixlength ip rules. Rule
  installation is now idempotent (delete-all-copies before add) and teardown
  removes every copy.
- Soft-fail log/dashboard messages now aggregate **every** probe endpoint's
  failure ("api.ipify.org: timeout; checkip.amazonaws.com: …") instead of
  showing only the last one, with the noisy Go `Get "…":` prefixes stripped.

## CLI 2.4.1 - 2026-08-26

Reliability release: the self-healing daemon no longer wedges, lies, or leaks.
Full analysis in `docs/AUDIT_2026-08-26_DAEMON_SELFHEAL.ru.md`.

### Fixed
- **Daemon deadlock**: a busy daemon (long connect/failover phase) looked dead
  to every reader — `stop` said "nothing to stop", the dashboard vanished, and a
  new `daemon` request crashed into the mutation lock. Existence is now
  PID-based (with /proc identity check); heartbeat freshness is a health signal
  only, kept alive by a dedicated 5s pulse goroutine.
- **Reconnect storm on probe outage**: egress health relied on a single probe
  URL (api.ipify.org); when the ISP blocked it, the daemon endlessly tore down a
  working tunnel. Now: three independent probe endpoints with one bounded time
  budget, plus WireGuard **handshake-age** (via UAPI) to distinguish "tunnel
  dead" from "probe endpoints unreachable".
- **Failover loop**: zones that repeatedly fail egress (ICMP-alive but not
  routing) are quarantined for 10 minutes; `--best` requests to a running
  daemon are re-ranked by the daemon itself so its quarantine map is honored.
- **Kill-switch leak**: the fail-closed guard was lifted on an UNVERIFIED
  reconnect; it now stays armed until egress is confirmed. Ctrl+C during
  connect no longer skips teardown (signal-aware root context), so guards can
  never be left behind.
- Daemon loop no longer sleeps inside the tick handler: SIGTERM, menu intents
  and zone switches apply within one tick even mid-backoff.
- **TUI/menu could not actually control the daemon**: connect/resume wrote the
  intent file directly from the unprivileged process, but /run/mazzy-vpn is
  root-owned — the write failed silently (EACCES discarded) and the UI reported
  "resumed" while the daemon saw nothing. All connect/resume/disconnect/recover
  requests now go through the elevated command path, which records the intent
  as root.
- Reconnect counter no longer increments for unconfirmed reconnects.

### Added
- **`sudo mazzy-vpn trust [--revoke]`**: one-command passwordless daemon
  control. Installs a visudo-validated sudoers drop-in that lets the invoking
  user run daemon/stop/disconnect/recover/up/auto/mimic without a password —
  so the menu/TUI stops prompting on every action. Escalation-safe: the rule
  pins the absolute binary path, refuses binaries or directories the user
  could overwrite, and excludes raw-path `connect` (catalog-only verbs cannot
  be turned into a root file-read oracle).
- Heartbeat state **paused** (⏸ in TUI/menu): a daemon holding the tunnel down
  after Disconnect is now distinguishable from a dead one.
- Dashboard: protocol name, "status Ns old" staleness warning, and the newest
  error inline in the TUI header.
- Daemon log: full date stamps, real failure reasons (probe error text), and
  5 MiB rotation to `daemon.log.1`.
- `diagnose` now reads the daemon heartbeat (paused / reconnect-looping /
  stale) and detects any foreign VPN interface (wg*, tailscale*, proton*, ...),
  not just tun0/tun1.

## Unreleased

### Changed
- **License: now PolyForm Noncommercial 1.0.0** (source-available). Free for
  personal, research/scientific, educational and charitable (noncommercial) use;
  **commercial use requires a separate commercial license** (see COMMERCIAL.md).
  An **Enterprise edition** is planned on top of this core. This replaces the
  previous AGPL-3.0-or-later licensing for all future versions; already-released
  AGPL versions remain available under AGPL. Updated LICENSE, all SPDX headers
  (238 files), package/Cargo/tauri metadata, READMEs (7 languages), CONTRIBUTING,
  wiki, and desktop About.

## CLI 2.4.0 - 2026-08-20

### Added
- Full-screen **Profiles** workflow: recursive file/folder/provider-bundle
  import, managed profile list, connect, favorite, verify and confirmed removal.
- Full-screen **Diagnostics** hub for doctor, verify, test/rank, adapters,
  network analysis, diagnosis, trace, stealth, DNS, provider and update checks.
- Language selector and contextual Help/About screen inside the default TUI.
- Universal side-effect-free help: `mazzy-vpn COMMAND -h`, `COMMAND --help`
  and `mazzy-vpn help COMMAND`.
- Interactive dashboard windows (1m/5m/20m/session), historical sample cursor,
  p50/p95, loss, jitter, heartbeat age, and a fixed five-row status/error pane.

### Fixed
- TUI connect now starts/resumes a detached session daemon and immediately
  returns to the dashboard instead of blocking in foreground `up`.
- Destructive recover now requires confirmation in the full-screen TUI.
- Test/rank opens the ranked zones view instead of completing invisibly.

## pi extension 0.1.1 - 2026-08-19

### Added
- Anonymized screenshot for the README and the pi.dev gallery (`pi.image`);
  bundled in the npm tarball. Data is neutral (RFC 5737 doc IP, generic zone).

## pi extension 0.1.0 - 2026-08-19

### Added
- **Mazzy VPN extension for pi** (`@mazurovn/mazzy-vpn-pi`, tag `pi-ext-v0.1.0`):
  connect to and manage Mazzy VPN from inside pi. Auto-connect, live status
  widget, monitor + auto-reconnect, `/mazzy-vpn` commands (alias `/vpn`), a
  Ctrl+Alt+V toggle, and LLM-callable tools (`vpn_status`, `vpn_connect`,
  `vpn_disconnect`, `vpn_verify_configs`, `vpn_best_zone`). Drives the mazzy-vpn
  Go CLI; never re-implements VPN logic. Published to npm and the pi.dev gallery
  (via the `pi-package` keyword). Source: `pi-extension/`.

## CLI 2.3.0 - 2026-08-19

### Fixed
- **`test`/`rank` no longer appears to hang.** Ranking a large catalog used a
  tiny ICMP pool (4) with no overall deadline or progress, so 50 profiles took
  ~40s on a blank screen. Ranking now scales concurrency, is time-bounded,
  honors cancellation, and shows a live `probing k/N…` indicator. Real 50-profile
  `test` completes in ~3s.

### Added
- **Import auto-distribution.** `import` scans folders recursively, classifies
  each profile by protocol (AmneziaWG / WireGuard / OpenVPN) and prints a
  distribution summary. When a server ships as both a WireGuard/AmneziaWG
  `.conf` and an OpenVPN `.ovpn`, the unconnectable OpenVPN twin is skipped so
  the catalog is not doubled.
- **`mazzy-vpn verify` (alias `audit`)** — health audit of every managed config:
  parses, validates, connectability, and endpoint DNS resolvability, with a
  problem+fix per profile and a machine `--json` mode. Wired into the menu (`v`).
- **Doctor catalog health** — `doctor` now shows a one-line offline summary:
  `catalog: N profiles (X connectable, Y OpenVPN-only)`.
- **TUI zone picker visualization** — animated "measuring" spinner and a latency
  quality bar (excellent/great/good/fair/slow) per alive zone, with an
  `alive/total` footer.
- Design/roadmap for the client journey: docs/design/cli-ux-roadmap-2026-08.md.

## CLI 2.2.1 - 2026-08-19

### Fixed (deep audit pass — see docs/audits/cli-audit-2026-08.md)
- **Disconnect/pause now work under a root daemon (P0-A):** the TUI→daemon
  intent file moved from `$HOME/.config` (which differed between the
  unprivileged UI writer and the root daemon reader under sudo) to the shared
  runtime dir alongside the heartbeat, so both sides agree on one file.
- **Stale intents no longer wedge a fresh daemon (P0-B):** intents older than
  2 minutes (or with a zero/future timestamp) are ignored.
- **`status` reports a consistent profile label (P1-C):** foreground connect and
  the daemon now normalize the zone name identically (no `Berlin.conf` vs
  `Berlin`).
- **`recover` reports honestly (P1-D):** counts real policy-routing deletions
  instead of always printing "cleared".
- **No fail-closed host after failed reconnects (P1-E):** the foreground connect
  path now tracks the kill-switch and removes the fail-closed table directly on
  teardown, matching the daemon.
- **Self-update integrity (P2-2):** the downloaded binary is verified against the
  release's published `SHA256SUMS` before install; refuses on mismatch.
- **`auto` honors its failover contract (P1-H):** delegates to the self-healing
  daemon instead of a single foreground connect.
- Version is now linker-stampable for a single source of truth (P1-F); `--clean`
  only re-ranks live zones (P1-G); local/fork builds are not falsely offered an
  "update" (P2-1); `help` prints to stdout (P2-4); dashboard truncation is
  display-width aware for CJK/emoji locales (P2-5); one shared value-flag-aware
  argument parser (P2-7); single rollback path in the binary replace (P2-3).

## CLI 2.2.0 - 2026-08-19

### Added
- Non-blocking interactive menu: connecting no longer drops you into a
  blocking log that only Ctrl+C can exit. Connect launches the self-healing
  daemon detached and returns you straight to the menu.
- Live dashboard header in the menu and TUI: connection badge, egress, a
  latency sparkline graph with min/avg/max, recent errors, an error-rate
  (errors/min) estimate and a reconnect counter — all rendered from a shared
  heartbeat without holding the terminal.
- Activity log viewer: press `l` in the menu/TUI to read the daemon log, and
  a key to return. Persisted across background runs.
- Optional background mode (menu option 6 / TUI `b`): runs the VPN in a
  detached session (`setsid`) that survives closing the terminal window;
  ordinary connects stay tied to the menu session and stop on quit.
- `stop` subcommand and menu/TUI `k` to terminate a running daemon.
- New `core/runstatus` package: world-readable status heartbeat written by the
  privileged daemon and read by the unprivileged menu/TUI, with bounded latency
  and error ring buffers and a Unicode sparkline renderer.

### Fixed
- Cross-privilege dashboard read: the runtime directory is now `0755`
  (traversable) so an unprivileged menu can read the root daemon's heartbeat
  and log; the `0600` mutation lock remains the exclusivity boundary.
- Liveness detection: a root-owned daemon PID probed from an unprivileged menu
  returns `EPERM` from `Signal(0)`, which still proves the process is alive.
  It is no longer misreported as dead (which had hidden the dashboard).
- Stop now works from the unprivileged menu: it routes through the elevated
  `stop` subcommand instead of a doomed unprivileged `SIGTERM` (`EPERM`).
- Disconnect is effective while a daemon runs: a durable down-intent pauses the
  daemon's auto-reconnect instead of the daemon immediately reviving the
  tunnel; Quick connect resumes it in place. A freshly started daemon clears a
  stale down-intent so it does not pause itself.
- The heartbeat is flushed on creation, so the dashboard shows a connecting
  state from the first moment; directory/file modes are chmod'd explicitly to
  defeat a restrictive umask.

## 1.4.7 / Desktop 0.4.8 - 2026-08-12

- Make the DEB Desktop autonomous: the package-owned engine, agent daemon,
  provider registry and systemd units are embedded under `/usr/lib` and work
  without a separately installed CLI; an optional public CLI remains included
  for terminal users and interoperates through the same protected local API.
- Remove the hard systemd recovery dependency that caused repeated
  `Dependency failed` boot failures. Recovery is now engine-gated and keeps
  status/profile queries available while mutations fail closed.
- Harden transition rollback, boot test recovery and health commit ordering so
  guards and durable markers are not cleared before verified restoration.
- Replace global route/DNS/firewall byte comparison with a canonical snapshot
  restricted to Mazzy-owned interfaces and firewall state.
- Add a DEB-only build and isolated payload gate; AppImage and RPM are not part
  of this release line.

## 1.4.6 / Desktop 0.4.7 - 2026-08-09

- Make Linux Desktop start its bundled engine before the first status/profile
  load. A clean AppImage now opens the native PolicyKit authorization flow,
  installs/repairs the shared backend, starts the protected local API and grants
  the current GUI session a per-user ACL; users no longer need to install or run
  the standalone CLI first or log out after bootstrap. Existing package-managed
  and `/usr/local` CLI clients keep using the same API and state.
- Make package-managed Desktop repair the installed engine in place instead of
  copying bundled files over distribution-owned paths.
- Report the engine source, embedded CLI, current-session API accessibility and
  cache readiness separately so startup recovery is based on usable runtime
  state rather than file presence alone.
- Add `acl` to Linux package dependencies and verify autonomous startup,
  package lifecycle and isolated single-instance GUI smoke tests.

## Desktop 0.4.6 - 2026-08-07

- Set a Linux-safe accessibility environment before Tauri/GTK startup to
  prevent a distribution `libatk-bridge` segmentation fault from crashing
  Desktop at launch.
- Canonicalize Tauri draft-release API asset URLs from the downloaded signed
  artifact inventory before updater verification and publication. The corrected
  manifest is uploaded while the versioned release is still a draft, and the
  SHA-256 manifest now covers that exact public feed without recursively
  including an older checksum manifest on a retried job.
- Synchronize the project status, installation guide, FAQ and roadmap with the
  published `v1.4.1` and `desktop-v0.4.1` releases.

## 1.4.5 / Desktop 0.4.5 - 2026-08-07

- Added an explicit first-run Desktop installation/repair dialog; the bundled
  engine is never installed silently.
- Made Desktop dependency readiness protocol-aware so a selected backend is
  checked instead of accepting an unrelated installed backend.
- Hardened Android managed-profile validation for mixed-case protocols and
  non-boolean `tls.insecure` values.
- Restored fail-closed validation for an empty profile catalog and completed
  Linux package, headless GUI and cross-platform release audits.

## 1.4.4 / Desktop 0.4.4 - 2026-08-06

- Fixed first-run self-test for users without profiles: `mazzy-vpn self-test`
  and `mazzy-vpn doctor` no longer fail when no VPN profiles are configured.
- Improved `mazzy-vpn diagnose` and `mazzy-vpn doctor` to report local API
  socket accessibility and `mazzy-vpn` group membership with actionable
  guidance.
- Made `doctor` dependency checks context-aware: protocol-specific backends
  (OpenVPN, WireGuard, AmneziaWG, L2TP/IPsec) are WARN when unused and FAIL
  only when a profile for that protocol is selected.
- Hardened `install.sh` on Ubuntu: if kernel headers for the running kernel
  are unavailable, the installer automatically falls back to the AmneziaWG
  userspace backend instead of aborting.
- Hardened `packaging/linux/post-install.sh` to add the current user to the
  `mazzy-vpn` group even when `SUDO_USER`/`PKEXEC_UID` are not set (e.g. GUI
  package managers).
- Prevented Desktop bootstrap from running the bundled installer over a
  package-managed engine; package-managed installs now receive an actionable
  error pointing to the distro package manager.

## 1.4.3 / Desktop 0.4.3 - 2026-08-06

- Fixed regression-suite execution under `sudo`: `tests/run.sh` now uses a
  root-owned `TMPDIR` so profile permission validation succeeds, and the package
  lifecycle test helpers (`post-install.sh --test-migrate` and
  `post-remove.sh --test-restore`) no longer require a non-root caller.
- Made Desktop self-contained on distributions without an AmneziaWG PPA: the
  bundled installer is re-run from the Desktop bootstrap flow when the engine is
  installed but a backend dependency is missing, so the AmneziaWG userspace
  fallback and other protocol backends are bootstrapped automatically.
- Improved the Desktop "local API is still starting" error: the socket is now
  treated as ready when it exists but the desktop process lacks the `mazzy-vpn`
  group, allowing `pkexec` fallback and telling the user to log out and back in
  if the group was just added.

## 1.4.2 / Desktop 0.4.2 - 2026-08-05

- Fixed Desktop startup recovery when the local API socket or profile cache is
  created after the graphical session: profile loading now uses bounded retry
  and no longer requires a manual refresh.
- Delayed package-managed Desktop fallbacks until the local API is ready,
  preventing boot-time `pkexec` `/dev/tty` errors and reporting an actionable
  engine-not-ready state instead.
- Preserved an explicitly enabled VPN engine across package upgrades without
  enabling or starting the engine unexpectedly on a fresh installation.

## 1.4.1 / Desktop 0.4.1 - 2026-08-05

- Fixed the systemd boot transaction cycle that could drop
  `mazzy-vpn-api.socket` after an upgrade. The early recovery unit now has an
  explicit dependency graph, and every mutating consumer requires successful
  recovery before activation.
- Propagated recovery ordering, bounded restart policy and permanent failure
  handling through package drop-ins so legacy `/etc/systemd/system` units
  cannot shadow the current safety contract.
- Added a consent-gated Desktop updater. It checks a fixed GitHub release feed
  at startup, displays a modal before any action, and verifies installable
  artifacts with an embedded Tauri updater public key. The private key exists
  only in release secrets.
- AppImage, Windows and macOS updater artifacts can be installed in-app after
  explicit confirmation. DEB/RPM installations open the matching release
  instead of replacing a package-managed executable with an AppImage.
- Added a three-platform release/feed gate: only a successful Linux, Windows
  and macOS signed build can publish the versioned preview and advance the
  fixed updater metadata. Operating-system code signing/notarization and
  native Windows/macOS VPN backends remain separate open gates.
- Hardened the release gate against command and path injection: the cargo
  executable is fixed, updater metadata can select only an inventoried
  downloaded artifact, signature outputs use local bounded names, and the
  helper has no caller-controlled filesystem paths. The cross-platform UI
  audit streams JavaScript over stdin instead of exceeding the Windows command
  line limit.
- Made the legacy-upgrade systemd regression independent of an already
  installed Mazzy VPN by verifying package units, drop-ins, executable and
  inert target fixtures inside an isolated root. The signed Desktop workflow
  now publishes a versioned SHA-256 manifest after updater-signature audit.
- Expanded systemd, package-payload, updater-consent and release supply-chain
  regressions. The shell suite now passes 103 cases.
- Prevented duplicate Desktop/WebView/tray instances; a repeated launch now
  focuses the existing dashboard.
- Reduced external health probes from roughly every 20 seconds to once per
  minute, including a reset drop-in for legacy `/etc/systemd/system` timers,
  and corrected
  recovery diagnostics to distinguish a rejected systemd action from an
  unverified post-restart egress.
- Classified missing optional profile-country metadata and single-provider
  GeoIP degradation as partial verification when route, DNS and leak checks
  pass, instead of reporting a healthy VPN route as a risk.
- Added a privacy-bounded, `1,000,000`-byte rotating Desktop lifecycle log with
  operation outcomes and diagnostic aggregates but no profile data, endpoints,
  credentials, configurations or IP addresses.
- Kept a verified pending Desktop update available after a download/install
  error so the consent dialog can retry without silently changing versions.

## 1.4.0 / Desktop 0.4.0 - 2026-08-03

- Added explicit credential-free `verify-service` and
  `tests.verify-service-egress` checks for fixed NotebookLM/OpenAI endpoints.
  Probes use bounded HEAD requests through the selected VPN interface and emit
  only strict sanitized egress-eligibility enums; they do not influence health,
  planning or automatic profile selection.
- Renamed the generic success label to `Network egress verified` without
  changing the strict `EgressVerification` v1 result shape, and made speed JSON
  numeric formatting independent of the caller locale.
- Bounded watchdog recovery to one restart exactly at the consecutive-failure
  threshold, followed by a threshold-plus-one pause while OpenVPN native retry
  continues. Automatic recovery no longer clears systemd failure state; a
  monotonic startup grace and realistic systemd start budget are covered by
  source/package parity tests.
- Added a dual-stack `nftables` output/forward transition guard around managed
  connect, reconnect, live test, emergency selection and health recovery.
  Direct egress is rejected before the old tunnel stops; only exact managed or
  fallback transport endpoints and exact resolver addresses are allowlisted.
  Readiness is bound to the selected interface and supports IPv4- and
  IPv6-only protected paths. If neither the new path nor rollback can be
  verified, the guard and a root-only recovery marker remain fail-closed.
  This closes the confirmed transition leak without claiming the still-planned
  persistent always-on kill switch.
- Added a root-only boot recovery entrypoint and hardened oneshot unit for
  interrupted local API actions. It takes the shared mutation lock before
  reconciliation and is ordered before test recovery, the managed tunnel,
  health remediation and API socket. Runtime-directory, permission and lock
  acquisition failures now durably preserve the root-only recovery marker.
  The tunnel service has an ordered `Requires=` dependency on recovery, so a
  marker-persistence failure also blocks boot activation; test recovery keeps
  its deadlock-safe boot path and bounded startup budget.
- Made API snapshots, running/completed action records, audit events and
  recovery markers durable with file and parent-directory synchronization
  before lifecycle mutation. Snapshot deletion is also persisted before a
  terminal response is acknowledged.
- Health remediation now checks that recovery marker while holding the shared
  lock and waits for the terminal `systemctl` result. It no longer releases
  mutation authority after an asynchronous `--no-block` request.
- Added a closed managed-profile v1 for VLESS, Hysteria 2, Mieru, NaiveProxy,
  TUIC, Shadowsocks 2022, Trojan, AnyTLS and ShadowTLS. Validation rejects
  duplicate keys, insecure TLS and user-controlled listeners, paths, marks and
  routing; responses never reflect endpoints or credentials.
- Tightened managed TLS validation with a closed uTLS fingerprint enum,
  unique ALPN and certificate-pin sets, and a minimum certificate-pin length.
- Added atomic `protocols managed-import` with dry-run, conflict protection,
  explicit force replacement, symlink rejection and root-only `0700/0600`
  storage. Modern-protocol import is now truthfully `partial`; connection
  remains `planned`.
- Added the packaged, version-pinned `mazzy-sing-box-adapter` foundation. It
  renders a closed TUN, proxy-routed DoH and final-route graph for six protocols
  but is not wired into service lifecycle. Mieru/Naive sidecars and the typed
  ShadowTLS inner chain fail closed.
- Added `runtime/v1/adapter-registry.json` and `protocols adapters --json` with
  explicit engine versions, process graphs and supply-chain, rollback, leak and
  crash-test gates.
- Added a separate reverse agent-control v1 contract for LAN WSS, iroh, libp2p,
  WebRTC, WebTransport, Tailscale/Headscale and reverse WSS, plus draft schemas
  that declare future signed E2EE envelopes and Desktop/Web/CLI/Telegram
  channel-risk policy. No E2EE runtime is claimed; all seven paths are planned.
- Bound every agent-control capability to a fixed risk class and restricted
  Telegram Bot commands to an argument-free low-risk allowlist, preventing a
  caller from downgrading command risk or bypassing channel policy.
- Contained the unreleased Desktop Agent Control screen to read-only
  Codex/Claude and transport discovery. The renderer and Tauri invoke surface
  no longer expose provider start/pair/stop, and diagnostics do not execute a
  discovered agent binary. Native command-bound approval, trusted executable
  resolution and process-tree containment remain mandatory before any local
  provider lifecycle authority can return.
- Hardened Desktop executable discovery: provider/probe names pass through a
  static allowlist and only fixed system directories are inspected. Unix
  candidates must be executable; Windows accepts only the fixed
  `.com`/`.exe`/`.bat`/`.cmd` suffix set. Mutable `PATH`, `HOME` and `PATHEXT`
  values cannot redirect diagnostics to an arbitrary file.
- Recovery markers make `_service-run` return systemd's permanent stop status
  `77`, preventing `Restart=always` from looping during manual recovery. The
  Agent Control provider badge is derived from installed, authorized runtime
  readiness instead of a fixed value.
- Documented the source-level Happy, Claude Bridge, Paseo and Yep Anywhere
  comparison and separated ingress, E2EE transport and provider-adapter
  boundaries. Typed Codex app-server/Claude/ACP adapters are preferred over a
  PTY protocol.
- Added the five-role target-architecture audit and executable delivery DAG for
  the privileged egress and unprivileged Agent Control planes. It records the
  split-lock/rollback P0 debt, selects reverse HTTPS/WSS as the durable first
  path, defines `mazzy-vpnd`/`mazzy-agentd` ownership, protocol/crypto/ACK/key
  requirements and defers optional transports behind measured release gates.
- Removed the Desktop pairing parser fallback that could expose an opaque
  vendor `pairingCode` when `manualPairingCode` was absent. Opaque-only
  responses now fail closed and are covered by a Rust regression.
- Added the transitional R0a single-flight boundary: API lifecycle, direct
  CLI, ordinary/managed profile import/remove, timeout/boot recovery, health
  remediation, policy cleanup, `doctor --fix`, autostart and monitor now
  contend on one runtime `.mutation.lock`. API child operations
  validate the inherited lock inode; invalid descriptors fail closed before a
  system mutation, and fallback subprocesses close API/direct lock descriptors
  before they may daemonize. This does not claim the planned `mazzy-vpnd` owner
  or full route/DNS/firewall rollback proof.
- Aligned the Agent Control registry with ADR-009: reverse WSS is the durable
  baseline, followed by LAN and iroh accelerators; optional/later transports
  no longer precede the baseline in executable path priority.

- Added the read-only API v1 `planner.evaluate` operation and
  `mazzy-vpn planner evaluate --stdin --json`. The backend enforces five
  non-overridable runtime/profile/storage/platform gates and applies the
  versioned 100-point protocol policy with stable opaque-ID tie-breaking. The
  rollback-storage gate is a prerequisite for future execution, not a claim
  that a candidate-specific rollback has already been proved.
- Planner inputs are strict, limited to 64 KiB and 128 unique candidates, and
  reject duplicate JSON keys at every object depth. Results are dry-run only,
  credential-free and bounded to a 1 MiB CLI response cap.
- The public schema now binds `planner.evaluate`, `PlannerRequest` and its
  required 100–30000 ms deadline in both directions, matching backend dispatch.
- Planner deadlines now reach the OpenVPN parser inside candidate validation;
  expired health evidence, including recent outcome, contributes no health
  score. The Python SDK example accepts legal JSON whitespace but rejects
  duplicate keys, non-finite numbers and multiple documents.
- Removed caller-assigned censorship/workload fit from planner evidence. The
  backend now derives both factors from the versioned protocol catalog,
  workload, protocol class and transports, while caller evidence is limited to
  observed health inputs.
- Extended credential-redacted protocol classification to bounded JSON for
  sing-box/Xray outbounds, official Mieru settings and NaiveProxy. Duplicate
  keys, multiple documents and ambiguous mixed-protocol configurations fail
  closed; classification never authorizes import or engine execution.
- Desktop profile-cache failures now retain safe missing, permission and
  invalid-shape reason codes instead of collapsing every failure into an
  unexplained empty library.
- Added deterministic, deadline, stale-evidence, unsafe-storage,
  unknown-profile, duplicate-input and local-transport regressions. Automatic
  switching, failover and mutation authorization remain tracked in issue #39.

## 1.3.2 / Desktop 0.3.2 - 2026-08-01

- Fixed the installed local API client closing the bidirectional `socat` STDIO
  channel as soon as request stdin reached EOF. The client now keeps the
  response half open with `STDIO,ignoreeof`, so socket-activated systemd workers
  can return their typed response instead of failing with `broken pipe`.
- Added a real Unix-socket integration regression with a delayed responder. It
  exercises the system `socat` binary rather than the existing fake dispatcher,
  and CI installs the dependency explicitly.
- Supersedes CLI/TUI 1.3.1. The hidden Desktop 0.3.1 draft and tag were removed;
  no Desktop 0.3.1 preview was published.

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
