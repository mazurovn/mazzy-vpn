// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// desiredIntent is the UI→daemon control file (ADR-0006 D2). It is written
// ONLY by privileged code paths (cmdDaemon's forward, recordDownIntent inside
// disconnect/stop/recover): /run/mazzy-vpn is root-owned 0755, so an
// unprivileged process cannot write here — the TUI/menu reach it through the
// elevated commands. The daemon polls and applies it each tick.
type desiredIntent struct {
	Zone    string `json:"zone"`
	Desired string `json:"desired"` // "up" | "down"
	TS      int64  `json:"ts"`
}

// desiredMaxAge bounds how long a written intent stays actionable. Beyond this
// the daemon ignores it (P0-B: a leftover "down" from a previous session must
// not wedge a freshly-started daemon forever). It also gates cross-boot files.
const desiredMaxAge = 2 * time.Minute

// desiredPath returns the intent-file path. It lives in the SHARED runtime dir
// (runDir → /run/mazzy-vpn, honored via MAZZY_RUN_DIR) — the same location as
// the heartbeat — so every privileged writer and the daemon agree on ONE file,
// and the unprivileged UI can still READ it (0644).
//
// P0-A: the previous implementation used os.UserConfigDir() = $HOME/.config,
// which differs between the unprivileged writer (HOME=/home/user) and the root
// daemon started via sudo (HOME=/root), so Disconnect/pause intents were written
// to a file the daemon never read. Anchoring to runDir() removes that split.
func desiredPath() string {
	return filepath.Join(runDir(), "desired.json")
}

// writeDesired atomically records the desired connection intent in the shared
// runtime dir. The file is world-readable (0644) so a root daemon can read an
// intent written by the unprivileged UI; the directory is created 0755 to match
// the heartbeat's cross-privilege visibility.
func writeDesired(zone, desired string) error {
	p := desiredPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(desiredIntent{Zone: zone, Desired: desired, TS: time.Now().Unix()}, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	// Best-effort: relax perms even if a prior umask tightened the temp file, so
	// the root reader can always see an intent from the unprivileged writer.
	_ = os.Chmod(tmp, 0o644)
	return os.Rename(tmp, p)
}

// readDesired reads the intent file (used by the daemon). It returns ok=false
// for a missing, unparseable, OR STALE intent so the daemon never acts on an
// order older than desiredMaxAge (P0-B). A zero/negative TS is treated as stale.
func readDesired() (desiredIntent, bool) {
	data, err := os.ReadFile(desiredPath())
	if err != nil {
		return desiredIntent{}, false
	}
	var di desiredIntent
	if json.Unmarshal(data, &di) != nil {
		return desiredIntent{}, false
	}
	if !intentFresh(di, time.Now()) {
		return desiredIntent{}, false
	}
	return di, true
}

// intentFresh reports whether an intent is recent enough to act on. Extracted so
// the staleness rule is unit-testable without touching the clock indirectly.
func intentFresh(di desiredIntent, now time.Time) bool {
	if di.TS <= 0 {
		return false
	}
	age := now.Sub(time.Unix(di.TS, 0))
	return age >= 0 && age <= desiredMaxAge
}
