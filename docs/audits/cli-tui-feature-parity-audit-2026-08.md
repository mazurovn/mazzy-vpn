# Mazzy VPN CLI/TUI feature-parity audit and roadmap

Date: 2026-08-20  
Scope: `mazzy-vpn-cli/cmd/mazzy-vpn`, `core/runstatus`, `core/livecheck`  
Baseline: CLI 2.3.0, branch `feat/go-rewrite`

## Executive summary

The project has **two different interactive interfaces**:

1. the line menu (`mazzy-vpn menu` / `--plain`), which already exposes import,
   remove, language and most diagnostics;
2. the full-screen Bubble Tea TUI (`mazzy-vpn` on a terminal / `mazzy-vpn tui`),
   which exposes only a small subset.

This distinction explains the reported mismatch: running `mazzy-vpn` opens the
full-screen TUI, where profile import/removal, provider-folder workflow,
language selection and most commands are absent even though some exist in the
line menu and CLI.

The root help works (`mazzy-vpn -h`, `--help`, `help`), but **subcommand help is
not implemented**. `mazzy-vpn doctor --help` runs doctor; other commands may
interpret `--help` as input, print a misuse error, or start network work. Help
must be intercepted before any handler side effect.

The existing dashboard is useful but not yet a complete control surface. It
shows state, zone, egress, uptime, a sparkline, min/avg/max probe duration,
check/fail counts, error rate, reconnect count and one recent error in the
full-screen header. It does not show traffic, throughput, packet loss, jitter,
handshake age, DNS state, kill-switch state or a dedicated five-line status/error
panel. The value currently labelled latency is the duration of an HTTPS egress
check, not WireGuard peer RTT; the UI must label this accurately.

## Method and review limitation

This is a direct source/code-path audit plus test inventory. Anthropic models
were explicitly excluded. The project Mazzy/Pi Ops control plane is available,
but the required `pi-subagents` child runtime is not loaded in the current Pi
session, so no independent external-model verdict is claimed. A non-Anthropic
Codex/Gemini/Grok review wave is specified below and must be run after the child
runtime is enabled.

## Command inventory and interactive parity

Legend:

- **Yes**: directly available.
- **Partial**: related action exists but important flags/workflow are absent.
- **No**: no reachable action in that interface.
- **Display**: represented as dashboard state rather than a command button.

| CLI command / capability | Line menu | Full-screen TUI | Finding |
|---|---:|---:|---|
| `tui`, `menu`, `--plain` | N/A | N/A | Interface selectors, not actions. |
| `doctor` | Yes | No | TUI lacks system-health screen/action. |
| `list DIR` | No | No | Directory validation is CLI-only without an explicit reason. |
| `validate FILE` | No | No | Single-file validation absent from both menus. |
| `verify [--no-dns]` | Yes (`v`) | No | TUI has no catalog health audit. |
| `import FILE|DIR...` | Partial | No | Line menu accepts one path; CLI accepts multiple paths and recursive folders. |
| provider bundle/folder import | Partial | No | Recursive scan exists in CLI, but there is no discover/preview/provider workflow. |
| `profiles [--ping]` | Partial | No | Line menu lists without `--ping`; TUI only has ranked zones, not profile management. |
| `favorite NAME [--off]` | Yes | No | Missing in TUI. |
| `remove NAME` | Yes | No | Missing in TUI; no batch remove. |
| `test` | Yes | Partial | `t` starts ranking but main view does not present the resulting table. |
| `best` | Yes | Partial | Best-zone connect exists; standalone result/action is absent. |
| `adapters` | Yes | No | Missing in TUI. |
| `netdiag` | Yes | No | Missing in TUI. |
| `diagnose` | Yes | No | Missing in TUI. |
| `trace [ZONE]` | Yes | No | Missing in TUI. |
| `stealth` | Yes | No | Missing in TUI. |
| `mimic [--apply]` | Yes | No | Missing in TUI. |
| `dns-check [--dot]` | Yes (without DoT choice) | No | TUI absent; line menu does not expose `--dot`. |
| `control id|pair|list` | No | No | Control-plane identity/pairing absent from both menus. |
| `language [code|--list]` | Yes | No | Full-screen settings have no language selector. |
| `up [NAME|--best|--clean]` | Partial | Partial | TUI best/zone paths use foreground `up`, which suspends the TUI until disconnect. |
| `auto` | Partial | Partial | Best selection exists, but the command's failover plan is not exposed clearly. |
| `daemon NAME` | Partial | Partial | Background best exists; named daemon/session/persistence choices are incomplete. |
| `stop` | Yes (`k`) | Yes (`k`) | Present. |
| `connect FILE [--uplink]` | No | No | Raw profile + pinned-uplink workflow absent. |
| `disconnect` | Yes | Yes | Present. |
| `recover [--reset-catalog]` | Partial | Partial | TUI executes recover immediately with no confirmation; reset-catalog is absent. |
| `providers [--type]` | Yes | No | Missing in TUI. |
| `update [--apply]` | Partial | No | Line menu checks only; apply is not an explicit menu choice. |
| `status [--json]` | Display | Display | Dashboard covers human status; no JSON menu action is needed. |
| `version` | No | No | Should be shown in About/Help, not necessarily a main action. |
| root `-h`, `--help`, `help` | N/A | N/A | Works only at root level. |
| `COMMAND -h`, `COMMAND --help` | No | No | Missing for every subcommand. |
| `help COMMAND` | No | No | `help` ignores the requested command. |

## Confirmed high-priority defects

### P0 — full-screen connect blocks the TUI

`requestConnectCmd` executes `mazzy-vpn up` through `tea.ExecProcess`.
`cmdUp` delegates to foreground `cmdConnect`, which holds the process until
SIGINT/TERM. Therefore quick connect and zone connect suspend the full-screen UI
instead of returning to a live dashboard. The line menu already avoids this by
starting `daemon ... --session`.

**Required correction:** use one shared non-blocking connection action for both
interactive UIs: start/resume the session daemon, return immediately, and update
through heartbeat events.

### P0 — destructive recover has no TUI confirmation

Pressing `r` immediately invokes privileged recover. The line menu asks for
confirmation. The TUI needs a modal confirmation; `--reset-catalog` needs a
stronger typed confirmation and must never share a one-key shortcut.

### P1 — misleading menu parity test

`TestMenuCoversUserFacingCommands` only scans `menu.go`. It proves line-menu
coverage, not full-screen TUI coverage, although the default interactive entry
point is the full-screen TUI. This allowed the regression to pass.

### P1 — test/rank hotkey has weak/no result UX

On the main TUI, `t` starts `rankZonesCmd`, but does not switch to the zones
screen. Completion only appends a log line; the ranked results are not displayed.
The loading state is also not rendered by `viewMain`.

### P1 — advertised cancel does not cancel

The zones screen says “press esc to cancel”, but `rankZonesCmd` owns an internal
context and exposes no cancellation function to the model. Esc only changes the
screen; the probes continue and can later deliver a stale result.

### P1 — subcommand help can trigger work or misuse parsing

Only the first root token is checked for `-h`/`--help`. Handlers receive later
help flags. Help must be resolved before diagnostics, filesystem changes,
network probes, elevation or catalog mutation.

### P1 — language exists but the default TUI cannot change it

The line menu has item 22 and `cmdLanguage` persists the setting. The
full-screen settings view exposes only six boolean toggles. In addition, many
TUI and line-menu labels are hardcoded English, so changing language does not
fully localize the interface.

### P1 — import capability is stronger than its UI

The CLI already recursively scans folders, accepts multiple roots, classifies
AmneziaWG/WireGuard/OpenVPN and suppresses OpenVPN twins when a WG/AWG variant
exists. The line menu takes only one unstructured path and the TUI has no import
screen, preview, progress, error drill-down or provider-bundle summary.

### P2 — status refresh can overlap expensive probes

The TUI schedules an HTTPS egress check every two seconds with a six-second
bound. There is no explicit single-flight guard. A slow check can overlap later
refresh commands. When a daemon heartbeat is present, this active probing is
largely redundant because the daemon already updates status every ten seconds.

### P2 — “latency” is actually HTTPS egress-check duration

The daemon records `time.Since(livecheck.Check)` as `LatencyMS`. That includes
DNS/TCP/TLS/HTTP endpoint behavior and is not peer RTT. Rename it to “egress
check” or add a separate tunnel/endpoint RTT metric.

## Help and command-registry design

Do not add dozens of independent `if hasFlag(args, "--help")` checks. Introduce
a single declarative registry:

```go
type CommandSpec struct {
    Name, Summary, Usage string
    Aliases, Examples    []string
    Flags                []FlagSpec
    Privilege            Privilege
    JSON                  bool
    Menu                  MenuPolicy
    Handler               func(context.Context, []string) int
}
```

Required behavior:

- `mazzy-vpn -h`, `--help`, `help` → root help, stdout, exit 0;
- `mazzy-vpn help import` → import help, stdout, exit 0;
- `mazzy-vpn import -h` and `import --help` → same help, **no side effects**;
- malformed use → concise error + command usage on stderr, exit 2;
- unknown flag → exit 2 instead of silent ignore;
- help generated from the same registry used by dispatch and menu parity tests;
- aliases resolve to the canonical command help;
- localization keys live in the registry rather than hardcoded paragraphs.

Add tests for every registered command: both help flags must return 0 and must
not call its handler. Add golden tests for root and command help.

## Proposed full-screen information architecture

Use tabs/screens rather than placing 30 actions on one dashboard:

1. **Dashboard** — connection health, interactive graph, five-line status/error
   pane, quick actions.
2. **Connect** — best, cleanest, choose zone, raw file, pinned uplink,
   session/background mode, disconnect/reconnect/stop.
3. **Profiles** — import, provider-folder scan, profiles, verify, favorite,
   remove/batch remove, validate file/folder.
4. **Diagnostics** — doctor, test, adapters, netdiag, diagnose, trace, stealth,
   DNS, mimic, provider checks.
5. **Settings** — all current toggles, preferred zone, preferred uplink,
   language, refresh interval, graph window.
6. **Logs & errors** — activity log, structured recent errors, filters and copy.
7. **Help/About** — key map, command equivalents, version and license.

Global keys: `tab`/`shift+tab` switch sections, `?` opens contextual help,
`esc` closes modal/goes back, `/` filters lists, `q` quits only when no modal is
open. Avoid overloaded keys whose meaning changes invisibly.

## Profile-management feature design

### Import file/folder/provider bundle

- `i` opens an import modal with a text input.
- Input may be a file or directory; directories are recursive.
- Support an **add another path** queue instead of shell-style splitting, so
  spaces in paths are safe.
- First run a dry discovery pass: total files, AWG/WG/OVPN counts, duplicates,
  invalid/unreadable files and OpenVPN twins that will be skipped.
- Confirm import, show progress and retain per-file errors in the error pane.
- A “provider bundle” is a preset label for the same recursive engine, not a
  separate importer, until provider-specific adapters are actually required.
- After import, offer Verify now / View profiles / Back.

### Remove profiles

- Profile table supports single and multi-select.
- `delete` opens a confirmation modal listing names.
- Removing an active profile is blocked or requires disconnect first.
- “Reset catalog” lives in a danger zone and requires typing `RESET`; never bind
  it to the normal delete key.
- Record a structured UI event and show success/failure per profile.

### Language

- Language is a selectable row in Settings using `i18n.Supported`.
- Apply immediately by replacing the model translator and redrawing all labels.
- Remove hardcoded English from TUI/menu views; fallback to English only for
  missing translation keys.

## Dashboard metrics roadmap

### Already available

- coarse state, zone, interface, protocol, egress IP;
- uptime and foreground/background mode;
- egress-check sample series and min/avg/max;
- total checks/failures, errors, errors/minute, reconnect count;
- recent structured error reasons.

### Phase 1: derived metrics without new privileged APIs

- success rate and packet/check loss percentage;
- rolling p50/p95 and jitter (mean absolute delta or standard deviation);
- heartbeat age/staleness;
- current sample age and last successful check time;
- reconnects/hour and time since last reconnect;
- quality grade based on egress-check latency, loss and staleness;
- current settings badges: kill-switch, auto-reconnect, DNS/mimic state;
- distinguish “egress-check duration” from tunnel RTT.

### Phase 2: real tunnel and host metrics

- RX/TX cumulative bytes and current throughput;
- latest handshake age;
- peer endpoint and tunnel MTU;
- physical uplink name/type/MTU;
- route and DNS resolver health;
- egress country/ASN and stealth score (cached, not queried every redraw).

The embedded engine has internal peer counters but does not currently expose a
safe metrics method. Add a narrow `Engine.Metrics()`/UAPI snapshot abstraction
rather than reading vendor internals from the TUI.

### Interactive graph

- selectable windows: 1m / 5m / 20m / session;
- cursor mode with timestamp, value and success/failure for the selected sample;
- left/right pan and `+`/`-` zoom;
- separate visual marks for failed egress checks;
- responsive width and ASCII fallback;
- do not redraw faster than telemetry arrives;
- increase/roll up the 120-sample ring if a session window is offered.

## Five-line status and error pane

Reserve a stable pane of exactly five event rows below the graph:

- severity glyph, timestamp, subsystem and sanitized message;
- newest first or chronological mode, selectable in Settings;
- categories: connection, reconnect, DNS, route, import, profile, privilege;
- one-line details with a `enter` drill-down modal;
- persistent daemon events merged with in-session UI events;
- empty rows remain visible to prevent layout jumping;
- unread-error badge and `e` shortcut to the full Logs & errors screen.

The existing main `logPane` only shows in-session events while `viewLog` merges
persisted lines. Consolidate both through a single structured event model.

## Implementation plan

### Phase 0 — safety and contract tests (P0/P1)

1. Add the command registry and universal command help.
2. Add strict unknown-flag handling.
3. Add a registry-driven parity test with an explicit `CLIOnlyReason` escape
   hatch; cover both line menu and full-screen TUI.
4. Make TUI connect/zone actions use the non-blocking session daemon.
5. Add recover confirmation modal and real cancellation for ranking.

Acceptance: every command passes `COMMAND -h` and `COMMAND --help` without
side effects; full-screen connect returns to dashboard; no destructive one-key
action.

### Phase 1 — profiles and language

1. Add Profiles screen and reusable profile table.
2. Add recursive file/folder import with preview and progress.
3. Add verify, favorite, single/batch remove and validation actions.
4. Add language and preferred zone/uplink selectors.
5. Complete localization of visible TUI and line-menu strings.

Acceptance: reported missing actions are reachable from default `mazzy-vpn` TUI
and covered by update/view tests.

### Phase 2 — diagnostics and command parity

1. Add Diagnostics screen with async cancellable jobs.
2. Add doctor/update/provider/control-plane/help screens.
3. Add raw-file connect and pinned-uplink flow.
4. Show command equivalent for each action for discoverability.

Acceptance: every human-facing command is reachable or explicitly documented as
CLI-only with a reason.

### Phase 3 — telemetry and dashboard

1. Extend `runstatus.Snapshot` compatibly (`schema_version`, optional fields).
2. Add derived loss/jitter/p50/p95/staleness metrics.
3. Add real engine counters/handshake/throughput through a narrow abstraction.
4. Implement interactive graph and fixed five-line event pane.
5. Add responsive layouts for narrow/medium/wide terminals.

Acceptance: no active probe overlap, graph values are accurately labelled,
telemetry JSON remains backward-compatible, and stale heartbeat is obvious.

### Phase 4 — quality, accessibility and release

- keyboard-only test matrix and contextual help;
- no-color/ASCII mode, CJK width tests and small-terminal behavior;
- race tests for async jobs, stale-message/request-ID tests and cancellation;
- golden view tests at 80x24, 120x30 and 160x45;
- integration tests with fake privilege runner, catalog and heartbeat;
- documentation/screenshots updated with anonymized RFC 5737 data.

## Non-Anthropic independent review wave (pending runtime enablement)

Maximum wall-clock budget: 10 hours; stop earlier when all verdicts arrive.

1. **Codex reviewer:** command registry, parser/help safety, CLI↔UI parity and
   test architecture.
2. **Gemini reviewer:** Bubble Tea information architecture, accessibility,
   localization and responsive layout.
3. **Grok reviewer:** telemetry model, interactive graph, operational failure
   states and threat/safety review.
4. Parent synthesis: deduplicate findings, require evidence references, turn
   accepted items into implementation tasks. No reviewer edits the active
   worktree.

## Recommended first deliverable

Implement Phase 0 and Phase 1 as one focused release candidate. These directly
fix the user's observed gaps: non-blocking TUI connection, recursive provider
folder import, profile removal, language selection and reliable `-h/--help`.
Telemetry expansion should follow after the control surface and command contract
are stable.
