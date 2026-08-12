# Mazzy VPN 1.4.7 / Desktop 0.4.8

Release candidate prepared 2026-08-12. Publishing remains a separate,
explicit step after review of the source commit and successful CI run.

## Supported artifact

This release is Linux DEB-only (`amd64`). AppImage, RPM, Windows, macOS and
Android artifacts are intentionally excluded. Package-managed Desktop updates
are installed through APT/dpkg; no Tauri self-update feed is published.

## Autonomous Desktop and standalone CLI

- The DEB contains its canonical engine at
  `/usr/lib/mazzy-vpn/mazzy-vpn`, `mazzy-agentd`, provider/API/protocol
  registries and all required systemd units.
- Desktop discovers and starts the package engine and protected local API. A
  separately installed or manually launched CLI is not required.
- `/usr/bin/mazzy-vpn` remains an optional, fully functional terminal entry
  point. It shares the protected API and state when Desktop is installed but
  does not require the Desktop process.
- Native PolicyKit authorization remains the privilege boundary for Desktop
  repair and other typed operations that are not yet exposed by the local API.
- OpenVPN is a mandatory DEB dependency. AmneziaWG readiness requires both its
  tools and backend rather than accepting an incomplete installation.

## Boot and rollback fixes

- Removed the hard recovery dependency that previously produced
  `Dependency failed for vpnctl.service` after reboot.
- Recovery safety is enforced inside the engine while read-only status and
  profile queries remain available.
- Boot test recovery has explicit `test-recovery`, `awaiting-egress` and
  `awaiting-cleanup` phases. Ordinary mutations require `ready`; only the
  managed executor has a narrow, verified resume path.
- Transition guards and durable markers are cleared only after rollback and
  owned-network restoration are verified. A failed service stop remains
  fail-closed.
- Network rollback snapshots are canonical and limited to Mazzy-owned
  interfaces, routes, DNS links and firewall tables. Unrelated host network
  changes no longer create false rollback failures.
- The health timer has both an initial boot trigger and a repeating interval,
  avoiding the systemd `NEXT=-`/`infinity` regression.

## Packaging and migration

- DEB systemd overrides execute the package-internal engine, never a public
  `/usr/bin` or `/usr/local/bin` CLI path.
- The installer migrates recognized legacy `/etc/systemd/system` shadows,
  including the socket-activated `mazzy-vpn-api@.service`, and preserves
  restorable backups for package removal.
- Post-install recovery is restarted safely; an opted-in VPN start deferred by
  recovery does not leave dpkg half-configured.
- The release workflow builds and publishes exactly one DEB plus SHA-256
  checksums as a draft GitHub Release.
- Project-local installation, rollback and pre/post-reboot acceptance scripts
  preserve the previous DEB and systemd intent without storing passwords,
  profiles or VPN credentials.

## Verification evidence

- `bash tests/run.sh`: `1..105`, all pass on the frozen release tree.
- API contract: `1.0`, 31 operations, 14 typed errors.
- Desktop UI contract: 110 IDs, 160 localized labels.
- Rust: `cargo check --locked` and 35 unit tests pass.
- Real Tauri release binary and real DEB built successfully.
- Extracted DEB audit passes: versions, executable modes, internal/public
  engine bytes, agentd, registries, dependencies, postinst and isolated
  systemd verification.
- Isolated unpacked-DEB GUI smoke remains alive for the full 10-second window.

Artifact:

`Mazzy VPN Desktop_0.4.8_amd64.deb`

SHA-256:

The exact package digest is published in the accompanying
`Mazzy.VPN.Desktop_0.4.8_SHA256SUMS` asset.

## Honest boundary

The source and package gates are green. Installation on a real host and an
actual cold reboot are operational acceptance steps and are not implied by the
offline package audit. Do not clear recovery markers blindly; inspect the
typed recovery state first.
