# pi ↔ Mazzy VPN integration (pi extension)

A pi extension that lets pi **connect to and manage Mazzy VPN** from inside a
coding session. It is a thin, typed driver over the audited `mazzy-vpn` Go CLI
(2.3.0+) — it never re-implements VPN logic, it orchestrates the CLI.

Location (local, auto-discovered): `.pi/extensions/mazzy-vpn/`
(`index.ts` + `client.ts` + `README.md`). `.pi/` is git-ignored harness state,
consistent with the repo policy that orchestration tooling stays local while Go
sources and design docs are published — this doc is the published record.

## What it does (maps to the request)

- **Connect automatically.** On an interactive session start, if the tunnel is
  not already up, it auto-connects to the best live zone via the CLI's
  self-healing background daemon (survives the pi session).
- **Manage the VPN.** `/vpn` commands + a Ctrl+Alt+V toggle + LLM-callable tools
  for status / connect / switch / disconnect / recover / verify.
- **"If there is no connection."** A bounded background monitor samples status
  every 30s (configurable) and, after two consecutive unprotected samples,
  reconnects to the best zone — complementing the CLI daemon's own fast retry.
- **"Block or bypass."** The model can call `vpn_connect` to route through the
  VPN (bypass a geo-block) or `vpn_disconnect`/`recover` to return to the plain
  uplink. `vpn_verify_configs` diagnoses why a zone will not connect.

## Surface

Tools (LLM-callable): `vpn_status`, `vpn_connect`, `vpn_disconnect`,
`vpn_verify_configs`, `vpn_best_zone`.
Commands: `/vpn status|connect [zone]|disconnect|recover|best|verify|doctor`.
Shortcut: `Ctrl+Alt+V` connect/disconnect toggle.
UI: live footer status + a widget above the editor (state · interface · egress).

## Design invariants

- **CLI is the single source of truth.** All state comes from
  `mazzy-vpn status --json`; ranking from `best --json`; audits from
  `verify --json`. No duplicated network logic.
- **Never start resources from the extension factory.** The monitor timer is
  created in `session_start` and torn down in `session_shutdown`; the timer is
  `unref`'d so it never keeps the process alive.
- **Do not fight the daemon.** The extension only reconnects after two
  consecutive misses and uses a `reconnecting` guard to avoid storms.
- **Survive the session.** Exiting pi does not drop the VPN (the background
  daemon persists); the user disconnects explicitly.
- **Graceful degradation.** If the CLI is absent (or `MAZZY_VPN_BIN` is unset and
  not on PATH), it warns once and disables actions instead of erroring.
- **Privilege boundary.** Read-only commands need no root; mutating ones elevate
  via the CLI's own sudo/pkexec (password prompt on first use).

## Validation

- Loads cleanly in real pi (`0.84.2`); all 5 tools register and are callable.
- Verified end-to-end against the live CLI 2.3.0: `vpn_status` (protected +
  egress), `vpn_best_zone` (fastest zone), `vpn_verify_configs`
  (50 total · 10 connectable · 40 OpenVPN-only · 0 broken).
- Auto-discovery works with no `-e` flag; auto-connect/monitor are gated to
  interactive/RPC (never one-shot `-p`).

## Config (env, optional)

`MAZZY_VPN_BIN`, `MAZZY_VPN_AUTOCONNECT`, `MAZZY_VPN_MONITOR`,
`MAZZY_VPN_MONITOR_SECS`, `MAZZY_VPN_AUTORECONNECT` — see the extension README.
