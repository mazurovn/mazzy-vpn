# Deep Audit — Mazzy VPN Go CLI (2026-08-18)

Scope: `mazzy-vpn-cli/` + `core/`. Method: full static scan, CI-gate
reproduction, behavioural code review, empirical grep-verification of every
claim. All findings below are confirmed, not speculative.

## How this passed the quality gate

The gates check **compilation and lint**, never **behaviour**:

- `go build/vet/staticcheck/govulncheck` are all green — they cannot detect
  that the TUI never connects, or that settings toggles are dead.
- There is **no end-to-end test** of the `menu → daemon → tunnel` path.
- Settings are written as a UI mock-up; the wiring into real logic was never
  added, and no test asserts settings *behaviour*.
- The red GitHub runs are the **old `CI` + `CodeQL` workflows**, not Go CI:
  `tests/check-codeql-boundary.py` hardcodes a codeql-action commit SHA, so
  every Dependabot bump of that action fails the guard (self-blocking test).

## Findings

### P0 — Menu does not connect

- **P0-1** Full-screen TUI (`tui.go`) "connect" only writes `desired.json`
  and prints "intent saved"; it never brings up a tunnel.
- **P0-2** The daemon that consumes `desired.json` is **not installed/enabled**
  (`mazzy-vpn@.service` absent). Intent is written but nothing applies it.
- **P0-3** Line menu `runPrivileged` discards the return code (`_ = run(args)`)
  and, when not root, only prints a hint instead of elevating (`menu.go:317`).
- **P0-4** TUI writes `writeDesired("--best","up")` but the daemon explicitly
  ignores `--best` (`daemon.go:124`) — dead path.
- **P0-5** `connect`/`up` are foreground-only; calling them in-process from the
  menu blocks the menu forever inside the dashboard loop.

### P1 — Dead settings (0 real uses; verified by grep)

- **P1-1** `KillSwitch` — advertised "fail-closed", default on, but
  `InstallFailClosed` is referenced **only in tests**; connect never arms it.
- **P1-2** `Notifications` — `notify` always fires; the toggle is ignored.
- **P1-3** `AutoReconnect` — `connect.go` honours only the `--no-reconnect`
  flag, not the setting.
- **P1-4** `AutoConnect` — never consumed; no auto-connect on start.
- **P1-5** `AutoDiagnostics` — never consumed.

### P2 — Missing menu entries (commands exist in CLI, absent from menu)

`mimic`, `favorite`, `remove`, `language`, `daemon`, `auto`, `validate`.

### P3 — CI gates

- **P3-1** `check-codeql-boundary.py` pins an exact action SHA → Dependabot
  bumps break it. Fix: validate the pin *format* (`@<40-hex>`), not a literal.

### P4 — Profiles

- **P4-1** 40/50 of the user's profiles are `.ovpn`, unsupported by the engine.
  "Choose a zone" lists them but connecting errors out; UX gives no signal.

## Fix log (implemented)

- **P0-1/P0-5** New `elevate.go`: `runPrivileged` re-execs the binary through
  sudo/pkexec when not root, surfaces the child exit code (no swallowed
  failures), and dispatches in-process when already root.
- **P0-1/P0-4** TUI (`tui.go`) now performs actions via `tea.ExecProcess`
  (suspends the alt-screen so sudo/pkexec can prompt), instead of only writing
  `desired.json`. The `--best` sentinel is no longer written as intent.
- **P0-3** Menu call sites capture the exit code and print an explicit
  localized success/failure line (`reportResult`).
- **P1-1** `connect.Conn.ArmKillSwitch/DisarmKillSwitch` wire the real
  fail-closed guard; `cmdConnect` arms it during the reconnect window and
  teardown removes any still-armed guard.
- **P1-2/P1-3** `Notifications` and `AutoReconnect` settings now drive behaviour
  (`resolveNotifications`, `resolveAutoReconnect`).
- **P1-4/P1-5** `AutoConnect` (menu start) and `AutoDiagnostics` (before `up`)
  are honoured.
- **P2** Menu gained: favorite, remove, mimic, language, daemon.
- **P3-1** `check-codeql-boundary.py` validates the pin *format* and init/analyze
  SHA *consistency*, so Dependabot bumps no longer self-block the gate.
- **P4-1** Choosing an OpenVPN zone shows a clear localized message instead of
  failing deep in connect.

Each fix ships with a Go test asserting the *behaviour* (not just compilation):
`elevate_test.go`, `killswitch_test.go`, `settings_wiring_test.go`,
`menu_coverage_test.go`, `tui_action_test.go`.

## Gate status after fixes (reproduced locally under CI conditions)

- Gate 1 build (core+cli, vendored, static): PASS
- Gate 2 test + race (core+cli): PASS
- Gate 3 vet + staticcheck + govulncheck (core+cli): PASS
- Gate 4 gofmt + audit-public + installer-autonomy + codeql-boundary: PASS
- Fuzz smoke (profile/control): PASS
