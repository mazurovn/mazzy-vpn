# Mazzy VPN CLI — Deep Audit (`mazzy-vpn-cli/cmd/mazzy-vpn`)

> **STATUS: ALL FINDINGS RESOLVED (2026-08).** Every item below was fixed and is
> covered by a regression test that does **not** rely on the `MAZZY_CONFIG_HOME`
> escape hatch that previously masked P0-A. `go build`, `go vet`, `gofmt -l`, and
> the full `go test ./...` pass clean. See `audit_fixes_test.go`,
> `update_test.go`, and the width test in `dashboard_test.go`.
>
> Fix map: P0-A/P0-B → `desired.go` (shared `runDir` path + `TS` staleness) ·
> P1-C → `connect.go` `profileDisplayName` · P1-D → `recover.go` honest counts ·
> P1-E → `connect.go` `killSwitchArmed` + `guard.RemoveFailClosed` fallback ·
> P1-F → `main.go` ldflags-stampable `version` + `displayVersion` ·
> P1-G → `up_cmd.go` live∩clean intersection · P1-H → `cmdAuto` delegates to the
> failover daemon · P2-1 → `isPrerelease` · P2-2 → `SHA256SUMS` verification ·
> P2-3 → `replaceBinary` single rollback path · P2-4 → `help`→stdout ·
> P2-5 → width-aware `trunc` · P2-7 → shared `firstNonFlagValueAware` parser.


Scope: ~5,600 LOC of non-vendor Go across 31 files. Baseline is healthy:
`go build`, `go vet`, `gofmt -l`, and `go test ./cmd/...` all pass clean.
This audit looks past the green tests for **logic bugs, client-path breakage,
cross-privilege correctness, and code smells the compiler cannot see.**

Severity: **P0** = user-visible broken behavior in the recommended flow ·
**P1** = correctness/leak/UX defect · **P2** = polish / maintainability.

---

## P0-A — Cross-privilege `desired.json` path mismatch (Disconnect/pause silently ignored in production)

**Files:** `desired.go:22-30`, `daemon.go:115,176`, `menu.go:380`, `tui.go:329-354`

`desiredPath()` resolves to:
```go
if d := os.Getenv("MAZZY_CONFIG_HOME"); d != "" { ... }      // tests only
if h, err := os.UserConfigDir(); err == nil { return .../mazzy-vpn/desired.json }
```
`os.UserConfigDir()` returns **`$HOME/.config`**.

- The **writer** is the unprivileged menu/TUI → `HOME=/home/<user>` →
  `/home/<user>/.config/mazzy-vpn/desired.json`.
- The **reader** is the daemon started via `sudo` (default `env_reset`) →
  `HOME=/root` → `/root/.config/mazzy-vpn/desired.json`.

The daemon polls a **different file** than the menu writes. Result in the
recommended flow (menu → Quick connect starts a `--session` daemon):
- **Disconnect** writes a down-intent the daemon never sees → the daemon
  reconnects on its next tick. The user's Disconnect is a no-op.
- **Zone switch** and **pause/resume** intents are likewise dropped.

Why the tests are green: `pause_resume_test.go` and `tui_action_test.go` set
`MAZZY_CONFIG_HOME` to a temp dir, so writer and reader agree **only in tests**.
This is a textbook case of a test-only env var masking a production defect.

**Fix options (pick one, deterministically):**
1. Put the intent file in the **world-shared runtime dir** already used for the
   heartbeat: `runDir()` (`/run/mazzy-vpn`, honored via `MAZZY_RUN_DIR`). The
   daemon writes the heartbeat there and both sides already agree on it. Make
   `desiredPath()` return `filepath.Join(runDir(), "desired.json")` and write it
   world-readable (0644). This is the smallest correct change.
2. Or capture the invoking user's `HOME`/`SUDO_USER` before elevation and pass
   `MAZZY_CONFIG_HOME` explicitly through `runInvoker`/`buildPrivilegedCmd`.

Option 1 is preferred: the heartbeat and the intent then live in one shared,
privilege-neutral location.

---

## P0-B — `TS` staleness field is written but never enforced → stale down-intent wedges a fresh daemon

**Files:** `desired.go:18,38`, `daemon.go:110-118,176-205`

`desiredIntent.TS` is set on every write but **never read**. `readDesired()`
returns any intent regardless of age. The daemon has a partial guard
(`daemon.go:113` clears intent to `up` on startup), but any later resume from
menu/TUI relies on the intent file — and a leftover `down` from a previous
session (different boot, crash, or the P0-A wrong-path file appearing later) is
honored forever with no freshness check.

**Fix:** in `readDesired()`, drop intents older than a bounded window (mirror
`runstatus.Snapshot.Fresh(30s)`), and compare `TS` against the daemon start time
so only intents *newer than the current run* are applied.

---

## P1-C — `status` shows a **stale profile name** from whichever writer ran last

**Files:** `connect.go:130`, `daemon.go:335`, `connect.go:236-260`

Two writers disagree on `state.State.Profile`:
- `cmdConnect` writes `filepath.Base(path)` → e.g. `Berlin.conf` (with extension).
- daemon `connectZone` writes the catalog `name` → e.g. `Berlin` (no extension).

`cmdStatus` reads `cur.Profile` and prints it as the profile label. So the same
connection is reported as `Berlin.conf` or `Berlin` depending on path taken, and
after a daemon failover to another zone the **persisted profile is never updated
back if a foreground connect later runs** — the label can lie. Also, `status`
derives live truth from `detectLiveInterface()` but the *name* from persisted
state, which can be from a completely different past session.

**Fix:** normalize to the catalog name in both writers
(`strings.TrimSuffix(base, filepath.Ext(base))`), and when `detectLiveInterface()`
is empty, do not present a persisted profile as if it were current.

---

## P1-D — `recover` prints "✔ cleared policy-routing rules" unconditionally (false success)

**File:** `recover.go:50-59`

The four `ip rule del ...` calls discard both output and error
(`_, _ = r.Run(...)`), then the code prints `✔ cleared policy-routing rules`
regardless of whether anything was cleared or the commands even exist. On a host
where `ip` is missing or the rules were never present, the panic button reports
success it did not achieve. Compare with the `run()` helper used elsewhere, which
only prints/counts on real success.

**Fix:** route these through the same `run()` accounting (or track a boolean) and
report "no rules to clear" vs. "cleared N" honestly. For a *panic button*, a
false "clean" is a safety claim the tool cannot back.

---

## P1-E — Foreground `connect` reconnect can leave the kill-switch **armed** on unrelated failures

**File:** `connect.go:186-235`

In the auto-reconnect branch: kill-switch is armed *before* teardown (good). But
if `connect.Up` fails (`rerr != nil`), the code logs, notifies, resets
`consecutiveFail`, and `continue`s — **leaving the kill-switch armed** and the
old `conn` reference replaced only on success. That is the intended fail-closed
posture *during* the gap, but there is **no ceiling**: if every reconnect keeps
failing, the loop keeps arming without a live `conn`, and the only disarm path on
exit is `conn.DisarmKillSwitch` on the *current* `conn` — which is the last
successfully-connected one, not necessarily reflecting the armed table. The
daemon path solved exactly this with an explicit `killSwitchArmed bool` +
`guard.RemoveFailClosed` fallback (`daemon.go:129-140`). The **foreground path
lacks that fallback**, so Ctrl-C after a series of failed reconnects can leave
the host fail-closed.

**Fix:** port the daemon's `killSwitchArmed` tracking + direct
`guard.New(...).RemoveFailClosed(ctx)` teardown fallback into `cmdConnect`.

---

## P1-F — Version is triplicated and already drifting

**Files:** `main.go:20` (`const version = "2.2.0"`), `RELEASE_NOTES.md`
(`Mazzy VPN 1.4.7 / Desktop 0.4.8`), `CHANGELOG.md` (`1.4.7`).

The CLI self-reports `2.2.0` while the shipping release notes/changelog say
`1.4.7`. `cmdUpdate.isNewer()` compares the GitHub tag against this constant, so a
wrong constant makes self-update either nag forever or refuse a real upgrade. A
single source of truth (ldflags `-X main.version=$(git describe)` at build) is
the durable fix; at minimum reconcile the constant with the release line.

---

## P1-G — `up --clean` re-ranks by stealth but ignores whether the pick is actually live

**File:** `up_cmd.go:82-95`

After `BestAlive` selects a live zone, `--clean` overwrites `name` with
`zonescore.New().Rank(live, 24h)[0]` — the *cleanest* zone by cached stealth
score. But the stealth cache can be **stale (24h window)** and the chosen
"cleanest" zone is not re-validated for liveness before connect. So `--clean` can
downgrade a proven-live pick to a cached-clean-but-now-dead one. Minor, but it
undoes the liveness guarantee the preceding rank just established.

**Fix:** intersect: pick the highest stealth score **among the currently
ICMP-alive** set only (it already builds `live`), and keep the `BestAlive`
fallback if the cache is empty.

---

## P1-H — `cmdAuto` with root only connects to `reachable[0]` with **no failover**, contradicting its own promise

**File:** `up_cmd.go:155-167` and its doc comment

The function comment says "connect ... with automatic failover", and the plan is
printed best-first, but the root branch does `cmdConnect(ctx, []string{best.File})`
— a single foreground connect with **no failover across the ranked list**. If
`reachable[0]` is ICMP-alive but not actually routing, the user is stuck. The
daemon has real failover; `auto` should either delegate to
`daemon <best> --best` (which fails over) or loop the ranked list.

**Fix:** make `cmdAuto` (root) start the failover-capable daemon, or iterate
`reachable` on egress-verify failure. Align code with the documented contract.

---

## P2 — Smaller issues and smells

1. **`isNewer` suffix logic is fragile** (`update.go:150-159`):
   `strings.ContainsAny(cur, "-")` treats `2.2.0-vpn.local` as "older than any
   clean tag", so a local dev build will always offer to "update" to an equal
   upstream tag. Use a real prerelease comparison (semver) or explicitly ignore
   `.local` builds in `cmdUpdate`.

2. **`replaceBinary` size gate is magic** (`update.go:246`): `fi.Size() <
   1_000_000` hard-codes "1 MB = valid". A stripped future binary or a different
   arch could legitimately be smaller; a corrupt 1.1 MB blob passes. Verify the
   **published sha256** from the release instead of a size heuristic (the code
   already computes the sha but only prints 16 hex chars — it never checks it).

3. **`copyFile` cross-device fallback drops the backup restore** (`update.go:225-233`):
   in the `os.Rename` failure branch it copies `newBin`→`target` but returns
   `nil` without ever restoring or removing the `.old` created earlier in
   `replaceBinary`; the two functions' rollback responsibilities overlap
   confusingly. Consolidate the atomic-replace logic in one place.

4. **`printUsage` writes to stderr for `help`** (`main.go:171` used by
   `case "help"` → returns 0). `help` is a success path; its text should go to
   **stdout** so `mazzy-vpn help | less` works. Only the *unknown command* branch
   should use stderr.

5. **Sparkline/`trunc` width math assumes rune-width == 1** (`dashboard.go`,
   `tui.go`): emoji and CJK egress/zone names (the app ships zh/ja/ko locales)
   are double-width, so `trunc(..., n)` and the box rules can misalign. Use a
   width-aware measure (the vendored `rivo/uniseg` is already present).

6. **`daemonRunning()` freshness is 30s but the ticker is 10s** — fine — but the
   **stealth ticker (5 min)** updates `lastStealth` from `gatherStealthSignal`,
   which can block; it runs in the same `select` loop as egress checks, so a slow
   stealth probe delays reconnect detection. Consider running it off-loop.

7. **`firstNonFlag` is defined and tested but `cmdUp` reimplements the same
   skip-flag-value parse inline** (`up_cmd.go:39-52`). Two parsers for one job;
   the inline one only special-cases `--uplink`. Any future value-flag (e.g.
   `--proto`) silently regresses to the "value read as zone name" bug the test
   claims to have fixed. Route both through one arg parser.

8. **`unknown command` returns 2 and prints full usage to stderr** (`main.go`):
   good, but flags typo'd as the *first* token (`mazzy-vpn --uplink eth0 up`)
   hit `default` and error out, because there is no global-flag handling before
   the subcommand switch. Minor CLI-ergonomics gap.

---

## Suggested fix order (highest client-path impact first)

| # | Finding | Effort | Impact |
|---|---------|--------|--------|
| 1 | **P0-A** shared intent path | S | Disconnect/pause actually work under sudo |
| 2 | **P0-B** enforce `TS` staleness | S | no wedged daemon from stale intent |
| 3 | **P1-E** foreground kill-switch fallback | M | no fail-closed host after failed reconnects |
| 4 | **P1-D** honest `recover` output | S | panic button tells the truth |
| 5 | **P1-C** profile-name normalization | S | `status` stops lying |
| 6 | **P1-H** `auto` failover contract | M | matches documented behavior |
| 7 | **P1-F** single version source | S | self-update correctness |
| 8 | **P1-G / P2** parser unification, sha verify, width-aware trunc | M | robustness |

All findings are logic/UX/security-posture issues that **pass the current test
suite** — every one is a candidate for a new regression test written *without*
the `MAZZY_CONFIG_HOME`/temp-dir escape hatch that currently hides P0-A.
