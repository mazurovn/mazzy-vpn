// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package runstatus is the shared live-status heartbeat between the privileged
// connect/daemon writer and the unprivileged menu/TUI reader. The writer (root)
// records a compact snapshot — link state, egress, a rolling latency series for
// the dashboard graph, and a rolling error log with a frequency estimate — to a
// world-readable JSON file. The reader renders a dashboard header without ever
// needing root, so the user lands back in the menu instead of a blocking log.
package runstatus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// State is the coarse connection state for the dashboard badge.
type State string

const (
	StateConnecting State = "connecting"
	StateProtected  State = "protected"
	StateLinkUp     State = "link-up" // interface up but no confirmed egress
	StateDown       State = "down"
	StateReconnect  State = "reconnecting"
)

// maxSamples bounds the latency graph ring; maxErrors bounds the error ring.
const (
	maxSamples = 120
	maxErrors  = 50
)

// Sample is one latency reading for the dashboard graph.
type Sample struct {
	TS        int64 `json:"ts"`         // unix seconds
	LatencyMS int   `json:"latency_ms"` // egress probe latency (0 = no egress)
	OK        bool  `json:"ok"`         // egress confirmed at this tick
}

// ErrEvent is one recorded error/degradation for the "recent errors" panel.
type ErrEvent struct {
	TS     int64  `json:"ts"`
	Reason string `json:"reason"`
}

// Snapshot is the full on-disk heartbeat. All fields are safe to render as-is
// EXCEPT strings sourced from the network (Egress/Zone/Reason), which the
// reader must pass through its own safeDisplay before printing.
type Snapshot struct {
	State      State      `json:"state"`
	Zone       string     `json:"zone"`
	Interface  string     `json:"interface"`
	Protocol   string     `json:"protocol"`
	Egress     string     `json:"egress"`
	PID        int        `json:"pid"`
	Background bool       `json:"background"`
	StartedAt  int64      `json:"started_at"`
	UpdatedAt  int64      `json:"updated_at"`
	Checks     int        `json:"checks"`
	Fails      int        `json:"fails"`
	Reconnects int        `json:"reconnects"`
	Samples    []Sample   `json:"samples"`
	Errors     []ErrEvent `json:"errors"`
}

// Fresh reports whether the heartbeat was updated within the given window. A
// stale file (writer died) should be treated as "no live daemon".
func (s Snapshot) Fresh(within time.Duration) bool {
	if s.UpdatedAt == 0 {
		return false
	}
	return time.Since(time.Unix(s.UpdatedAt, 0)) <= within
}

// LatencySeries returns just the latency values, oldest→newest, for the graph.
func (s Snapshot) LatencySeries() []int {
	out := make([]int, len(s.Samples))
	for i, sm := range s.Samples {
		out[i] = sm.LatencyMS
	}
	return out
}

// ErrorRatePerMin estimates errors per minute over the trailing window.
func (s Snapshot) ErrorRatePerMin(window time.Duration) float64 {
	if len(s.Errors) == 0 || window <= 0 {
		return 0
	}
	cutoff := time.Now().Add(-window).Unix()
	n := 0
	for _, e := range s.Errors {
		if e.TS >= cutoff {
			n++
		}
	}
	mins := window.Minutes()
	if mins <= 0 {
		return 0
	}
	return float64(n) / mins
}

// RecentErrors returns up to n newest error events (newest first).
func (s Snapshot) RecentErrors(n int) []ErrEvent {
	if n <= 0 || len(s.Errors) == 0 {
		return nil
	}
	out := make([]ErrEvent, 0, n)
	for i := len(s.Errors) - 1; i >= 0 && len(out) < n; i-- {
		out = append(out, s.Errors[i])
	}
	return out
}

// Path returns the heartbeat file path. It honors MAZZY_RUN_DIR (tests/dev) and
// otherwise lives under /run/mazzy-vpn where any user can read it.
func Path() string {
	if d := os.Getenv("MAZZY_RUN_DIR"); d != "" {
		return filepath.Join(d, "status.json")
	}
	return filepath.Join("/run/mazzy-vpn", "status.json")
}

// Writer records heartbeats. It is created by the privileged daemon/connect
// path and owns the rolling rings, so callers only push events.
type Writer struct {
	path string
	snap Snapshot
}

// NewWriter starts a heartbeat for a session and immediately flushes it, so the
// unprivileged reader sees a "connecting" dashboard from the very first moment
// rather than a gap until the first tick. background marks a detached run.
func NewWriter(zone, iface, proto string, background bool) *Writer {
	now := time.Now().Unix()
	w := &Writer{
		path: Path(),
		snap: Snapshot{
			State: StateConnecting, Zone: zone, Interface: iface, Protocol: proto,
			PID: os.Getpid(), Background: background, StartedAt: now, UpdatedAt: now,
		},
	}
	w.flush()
	return w
}

// SetState updates the coarse state and egress details, then flushes.
func (w *Writer) SetState(st State, iface, egress string) {
	if w == nil {
		return
	}
	w.snap.State = st
	if iface != "" {
		w.snap.Interface = iface
	}
	if egress != "" {
		w.snap.Egress = egress
	}
	w.flush()
}

// SetZone records a zone switch (failover), then flushes.
func (w *Writer) SetZone(zone string) {
	if w == nil {
		return
	}
	w.snap.Zone = zone
	w.flush()
}

// Tick records one health sample (latencyMS, ok) and flushes. A zero or
// negative latency with ok=false denotes a missed egress at this tick.
func (w *Writer) Tick(latencyMS int, ok bool) {
	if w == nil {
		return
	}
	w.snap.Checks++
	if !ok {
		w.snap.Fails++
	}
	w.snap.Samples = append(w.snap.Samples, Sample{TS: time.Now().Unix(), LatencyMS: latencyMS, OK: ok})
	if len(w.snap.Samples) > maxSamples {
		w.snap.Samples = w.snap.Samples[len(w.snap.Samples)-maxSamples:]
	}
	w.flush()
}

// Error appends an error event (rate-panel input) and flushes.
func (w *Writer) Error(reason string) {
	if w == nil {
		return
	}
	w.snap.Errors = append(w.snap.Errors, ErrEvent{TS: time.Now().Unix(), Reason: reason})
	if len(w.snap.Errors) > maxErrors {
		w.snap.Errors = w.snap.Errors[len(w.snap.Errors)-maxErrors:]
	}
	w.flush()
}

// Reconnected increments the reconnect counter and flushes.
func (w *Writer) Reconnected() {
	if w == nil {
		return
	}
	w.snap.Reconnects++
	w.flush()
}

// Close marks the session down and removes the heartbeat file so a stale
// snapshot never lingers after a clean disconnect.
func (w *Writer) Close() {
	if w == nil {
		return
	}
	_ = os.Remove(w.path)
}

// flush writes the snapshot atomically with world-readable perms so the
// unprivileged menu can read a root-written heartbeat. Both the directory
// (0755, traversable) and the file (0644) are chmod'd explicitly to defeat a
// restrictive umask that would otherwise strip the read/traverse bits and hide
// the dashboard from the unprivileged reader.
func (w *Writer) flush() {
	w.snap.UpdatedAt = time.Now().Unix()
	dir := filepath.Dir(w.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	_ = os.Chmod(dir, 0o755)
	data, err := json.MarshalIndent(w.snap, "", "  ")
	if err != nil {
		return
	}
	tmp := w.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Chmod(tmp, 0o644)
	if err := os.Rename(tmp, w.path); err != nil {
		_ = os.Remove(tmp)
	}
}

// Read loads the current heartbeat. ok is false when absent/unreadable.
func Read() (Snapshot, bool) {
	data, err := os.ReadFile(Path())
	if err != nil {
		return Snapshot{}, false
	}
	var s Snapshot
	if json.Unmarshal(data, &s) != nil {
		return Snapshot{}, false
	}
	return s, true
}
