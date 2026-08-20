# Mazzy VPN CLI — UX / Feature Design & Roadmap

Scope: the interactive client path (menu + TUI) and the diagnostic surface.
This document captures (a) what was just implemented, and (b) the detailed
follow-on design for the remaining gaps, focused on the **client journey** —
from "I have a folder of configs" to "I am protected and can see why".

---

## 1. The client journey (target)

```
 IMPORT ──► DISTRIBUTE ──► VERIFY ──► RANK ──► CONNECT ──► MONITOR ──► RECOVER
  folder     by proto      health    live      best zone   live dash    panic
  of .conf   (auto-sort)   audit     probe     + failover  + warnings   button
  /.ovpn
```

Each arrow is a place the old CLI could strand the user. The principles:

1. **Never look frozen.** Every network wave shows live progress and is
   time-bounded. (Fixes the "test/rank hangs" report.)
2. **Sort work for the user.** A dumped provider folder is auto-classified by
   protocol; unusable duplicates are hidden, not piled on.
3. **Tell the truth, with a fix.** Every diagnostic states what is wrong AND the
   next action.
4. **One label everywhere.** A zone is named the same in status, menu, TUI, and
   logs.

---

## 2. Implemented in this pass

### 2.1 Fixed: `test` / `rank` "hang" (P0)
- Root cause: `RankBest` used a fixed ICMP pool of **4** with up to **3s** per
  probe and **no overall deadline or progress**. A 50-profile catalog serialized
  into ~40s of dead air with a blank screen.
- Fix (`core/measure`): new `RankBestProgress(ctx, targets, progress)`:
  - concurrency scales with catalog size (8, or 12 for >24 targets),
  - honors `ctx` cancellation (returns a full slice, remainder marked
    `cancelled`) so it can never hang,
  - reports `progress(done, total)` per completed probe.
- Fix (CLI `network_cmds.go`): `rankWithProgress` wraps every ranking call with a
  **bounded deadline** (`rankBudget`: 12/20/30s by size) and a live
  `probing k/N…` line on stderr (suppressed in `--json`/non-TTY). Wired into
  `test`, `best`, the zone picker, and the TUI overlay.
- TUI (`tui.go`): the "Measuring…" overlay now animates a spinner (fast tick
  while loading) and each alive zone shows a latency quality bar
  (`excellent/great/good/fair/slow`).
- Measured: real 50-profile catalog `test` now completes in ~3s (was ~19s+),
  with visible progress throughout.

### 2.2 Import auto-distribution + dedup
- `import` now scans directories **recursively** (nested provider bundles).
- Each profile is **classified by protocol** (AmneziaWG / WireGuard / OpenVPN)
  and the pass prints a distribution summary:
  `sorted: AmneziaWG N · WireGuard N · OpenVPN N`.
- **WG-over-OVPN preference:** when the same logical server ships as both a
  `.conf` (WG/AWG) and an `.ovpn`, the OpenVPN twin is skipped so the catalog is
  not doubled with unconnectable entries. Matching is by a normalized stem
  (case/separator-insensitive), so `Austria--Vienna-S1.ovpn` ≡
  `AustriaViennaS1.conf`.
- Clear note when OpenVPN-only entries remain (engine cannot connect them yet).

### 2.3 Config health diagnostics — `verify` (new)
- `mazzy-vpn verify [--no-dns] [--json]` audits **every** managed profile:
  parses, validates required fields, connectability (engine can bring it up),
  and endpoint **DNS resolvability** (cheap, high-signal liveness).
- Broken profiles sort first; each shows problems + a one-line fix. Exit code is
  non-zero if any profile is a hard FAIL (parse/validate), WARN otherwise.
- Menu: `v` → Verify configs. Also aliased as `audit`.

### 2.4 Doctor now includes catalog health
- `doctor` prints a one-line offline catalog summary:
  `catalog: N profiles (X connectable, Y OpenVPN-only)` — instantly diagnosing
  a folder of OpenVPN-only configs without running the full audit.

---

## 3. Menu — current and proposed

Current interactive menu is a flat numbered list (1–24 + keyed l/k/v). It works
but is long. Proposed refinement (backwards-compatible; numbers keep working):

```
  Mazzy VPN — <status header / live dashboard>

  CONNECT                          DIAGNOSE
   1 ⚡ Quick connect (best)         7 📶 Test servers (live)     ← progress now
   2 🌍 Choose a zone (ping+bar)     8 🏆 Best zone
   3 🔄 Reconnect w/ diagnostics    10 🩺 Analyze network
   4 ⏹  Disconnect                  11 🔍 Diagnose (what's wrong)
   6 🛰  Run in background           13 🕵️ Stealth check
   k ⏹  Stop background             14 🔒 DNS privacy
                                    23 🔧 Doctor (+ catalog health)
  PROFILES                          v 🪺 Verify configs (health)  ← NEW
  17 📥 Import folder (auto-sort)   16 🤖 AI providers
  18 📋 List profiles               l 📜 Activity log
  19 ★ Favorite   20 🗑 Remove
  21 ⚙️ Settings   22 🌐 Language    0 Quit
```

Design intent: three columns grouped by intent (CONNECT / DIAGNOSE / PROFILES),
the header carries the live dashboard, and the two highest-value new surfaces
(progressive Test, Verify) are one keystroke away.

---

## 4. Remaining gaps & detailed design (next passes)

### 4.1 OpenVPN connectability (biggest functional gap)
Today OpenVPN configs are catalogued but not connectable (no engine). Options,
in order of preference:
- **A. Shell out to `openvpn`** when present (detected by `doctor`): wrap it like
  the AmneziaWG engine, publishing the same `runstatus` heartbeat so the TUI/menu
  dashboard is identical. Lowest effort, immediate value.
- **B. Convert** OpenVPN→WireGuard where the provider offers both (already the
  dedup path); surface a "this server also has a WG variant" hint on import.
- Client-path: `verify` already flags OVPN as WARN with the exact advice; the
  connect path should refuse early with the same message (it does today via
  `loadProfile`).

### 4.2 Live connection visualization & warnings (TUI)
- Add a **throughput/latency history** panel to the connected dashboard (the
  `runstatus` heartbeat already stores a latency series + error events; the TUI
  renders a sparkline). Extend with:
  - a **leak/stealth badge** (IPv6/DNS/timezone) sampled every N minutes (the
    daemon already runs a stealth ticker — surface its score as a colored badge),
  - **desktop + in-TUI warnings** when egress country changes unexpectedly or
    stealth score drops (daemon emits the event; TUI shows a toast line).
- Add a **"why am I not protected?"** inline hint in the header when LINK-UP but
  no egress, linking to `diagnose`.

### 4.3 Provider/config diagnostics depth
- `verify` today does parse+validate+DNS. Add (opt-in `--deep`):
  - UDP endpoint reachability (reuse `measure.Probe`) so a config is checked
    end-to-end without connecting,
  - handshake dry-run for WG/AWG where possible,
  - duplicate/near-duplicate detection across the catalog (same endpoint, keys).
- `providers` (AI reachability) should gain a **before/after** mode: run once on
  the plain uplink and once on the tunnel, to prove the VPN unblocks a provider.

### 4.4 Import ergonomics
- Detect a **single archive** (`.zip`/`.tar.gz`) and offer to extract+import.
- On import, optionally **auto-favorite** the fastest server per country.
- Show a post-import **"run verify now?"** prompt in the menu path.

### 4.5 Robustness / clientpath correctness
- `auto`/`best` should cache the last successful ranking briefly so repeated menu
  entries are instant (invalidate on network change).
- The zone picker should let the user **filter by country** and **sort by
  latency vs. stealth** (data already available).

---

## 5. Test coverage added
- `core/measure`: progress callback fires per probe; cancellation returns a full
  slice without hanging.
- CLI: import stem normalization, recursive collection, OpenVPN WARN vs broken
  FAIL audit verdicts.
- All existing suites remain green; `go build/vet/gofmt` clean.
