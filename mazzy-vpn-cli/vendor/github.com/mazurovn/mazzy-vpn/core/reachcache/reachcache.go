// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package reachcache remembers, per zone, whether the tunnel to it recently
// carried REAL internet egress — not just whether the server answered a ping.
//
// This is the missing signal in server selection: ICMP ranking picks the
// fastest-ping server, but a server can answer ICMP (and even complete the
// WireGuard handshake) while forwarding no traffic at all. Without a memory of
// "this zone actually routed", selection re-picks the same fast-but-dead server
// every time. reachcache lets ranking prefer zones that recently WORKED and
// sink zones that recently failed egress, across restarts and in the picker.
package reachcache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Record is one zone's most recent egress outcome.
type Record struct {
	Zone       string `json:"zone"`
	EgressOK   bool   `json:"egress_ok"`   // last attempt actually routed
	At         int64  `json:"at"`          // unix seconds of the last attempt
	FailStreak int    `json:"fail_streak"` // consecutive egress failures
	// Reason distinguishes HOW a zone failed: "dead" (no handshake — the
	// server is down/blocked at the crypto layer) vs "no-route" (tunnel up,
	// zero forwarded traffic). Empty on old records and successes.
	Reason string `json:"reason,omitempty"`
}

// Cache is a small JSON file of per-zone egress outcomes.
type Cache struct {
	Path string
	recs map[string]Record
}

// DefaultPath honors MAZZY_STATE_DIR (tests/dev), else uses ONE shared
// system location: /var/cache/mazzy-vpn/reachcache.json.
//
// Why shared and not per-user: every WRITER of this cache is root (the daemon
// observing egress, `probe` recording deep-test verdicts) while the READERS
// include the unprivileged UI (zone picker, `test`). The previous per-user
// path (os.UserConfigDir) meant the root daemon wrote /root/.config while the
// user's TUI read ~/.config — two divergent caches, so the picker showed
// fast-ping-but-dead zones as attractive even though the daemon had already
// learned they route nothing. One root-written, world-readable file removes
// the split-brain; it contains only zone labels and timestamps (no secrets).
//
// Why /var/cache and NOT /var/lib/mazzy-vpn: the state store deliberately
// chmods /var/lib/mazzy-vpn back to 0700 on every write (private state), which
// would lock the unprivileged UI out again. Regenerable, non-secret data gets
// its own world-readable cache directory.
func DefaultPath() string {
	if d := os.Getenv("MAZZY_STATE_DIR"); d != "" {
		return filepath.Join(d, "reachcache.json")
	}
	return sharedPath
}

// sharedPath is the canonical system-wide cache location.
const sharedPath = "/var/cache/mazzy-vpn/reachcache.json"

// legacyPaths are pre-shared-location files, read once as a seed so existing
// egress history is not lost when upgrading (best-effort; the /var/lib one is
// root-readable only and migrates when the root daemon first saves).
func legacyPaths() []string {
	out := []string{"/var/lib/mazzy-vpn/reachcache.json"}
	if h, err := os.UserConfigDir(); err == nil {
		out = append(out, filepath.Join(h, "mazzy-vpn", "reachcache.json"))
	}
	return out
}

// New returns a Cache at the default path.
func New() *Cache { return &Cache{Path: DefaultPath()} }

// NewAt returns a Cache at an explicit path (tests).
func NewAt(path string) *Cache { return &Cache{Path: path} }

func (c *Cache) load() {
	if c.recs != nil {
		return
	}
	c.recs = map[string]Record{}
	data, err := os.ReadFile(c.Path)
	if err != nil {
		// Seed from a legacy location so history written before the
		// shared-path change still informs ranking (best-effort, read-only).
		// Only for the canonical shared path — a test/dev cache (MAZZY_STATE_DIR
		// or NewAt) must stay isolated from the invoking user's real files.
		if c.Path == sharedPath {
			for _, lp := range legacyPaths() {
				if data, err = os.ReadFile(lp); err == nil {
					break
				}
			}
		}
		if err != nil {
			return
		}
	}
	var list []Record
	if json.Unmarshal(data, &list) == nil {
		for _, r := range list {
			c.recs[r.Zone] = r
		}
	}
}

func (c *Cache) save() error {
	list := make([]Record, 0, len(c.recs))
	for _, r := range c.recs {
		list = append(list, r)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Zone < list[j].Zone })
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	// Dir 0755 / file 0644: root writes, the unprivileged UI must be able to
	// read (the cache holds zone labels + timestamps only). A random temp name
	// (never a predictable "<path>.tmp") plus rename keeps the write atomic and
	// immune to a pre-planted symlink at the temp path.
	if err := os.MkdirAll(filepath.Dir(c.Path), 0o755); err != nil {
		return err
	}
	tf, err := os.CreateTemp(filepath.Dir(c.Path), ".reachcache-*")
	if err != nil {
		return err
	}
	tmp := tf.Name()
	if _, err := tf.Write(data); err != nil {
		_ = tf.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := tf.Chmod(0o644); err != nil {
		_ = tf.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := tf.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, c.Path)
}

// RecordOK marks that the zone routed real egress just now.
func (c *Cache) RecordOK(zone string) {
	if zone == "" {
		return
	}
	c.load()
	c.recs[zone] = Record{Zone: zone, EgressOK: true, At: time.Now().Unix()}
	_ = c.save()
}

// RecordFail marks that the zone failed to route egress (handshake-only),
// incrementing its failure streak.
func (c *Cache) RecordFail(zone string) { c.RecordFailKind(zone, "no-route") }

// RecordFailKind is RecordFail with an explicit failure kind ("dead" for
// no-handshake servers, "no-route" for tunnel-up-but-nothing-forwarded).
func (c *Cache) RecordFailKind(zone, reason string) {
	if zone == "" {
		return
	}
	c.load()
	prev := c.recs[zone]
	c.recs[zone] = Record{Zone: zone, EgressOK: false, At: time.Now().Unix(), FailStreak: prev.FailStreak + 1, Reason: reason}
	_ = c.save()
}

// Get returns the stored record for a zone.
func (c *Cache) Get(zone string) (Record, bool) {
	c.load()
	r, ok := c.recs[zone]
	return r, ok
}

// Reorder returns names reordered by egress reliability WITHOUT changing the
// membership: it is a stable bias, so the caller's upstream ordering (e.g. ICMP
// latency) is preserved within each reliability tier. Tiers, best first:
//
//  1. recently WORKED (EgressOK within `ttl`)
//  2. unknown (no record, or record older than ttl)
//  3. recently FAILED egress (EgressOK=false within `ttl`) — pushed last,
//     worst fail-streak deepest
//
// This makes selection prefer proven-working zones and avoid the fast-ping
// servers that accept the tunnel but route nothing, while never permanently
// excluding a zone (a failed zone still ranks, just last, so recovery is
// possible once the good zones are exhausted).
func (c *Cache) Reorder(names []string, ttl time.Duration) []string {
	c.load()
	now := time.Now()
	tier := func(n string) int {
		r, ok := c.recs[n]
		if !ok || now.Sub(time.Unix(r.At, 0)) > ttl {
			return 1 // unknown
		}
		if r.EgressOK {
			return 0 // worked
		}
		return 2 // failed
	}
	idx := map[string]int{}
	for i, n := range names {
		idx[n] = i
	}
	out := append([]string(nil), names...)
	sort.SliceStable(out, func(i, j int) bool {
		ti, tj := tier(out[i]), tier(out[j])
		if ti != tj {
			return ti < tj
		}
		if ti == 2 { // both failed: deeper fail-streak sinks further
			ri := c.recs[out[i]]
			rj := c.recs[out[j]]
			if ri.FailStreak != rj.FailStreak {
				return ri.FailStreak < rj.FailStreak
			}
		}
		return idx[out[i]] < idx[out[j]] // stable: preserve upstream order
	})
	return out
}

// Verdict classifies a zone's recent egress history for display:
//
//	"ok"      — routed real egress within ttl
//	"fail"    — failed egress within ttl (FailStreak deep = more certain)
//	"unknown" — no record, or the record is older than ttl
//
// This is the honest liveness signal the ping columns lack: a server can
// answer ICMP in 20 ms and still forward nothing.
func (c *Cache) Verdict(zone string, ttl time.Duration) (verdict string, streak int) {
	c.load()
	r, ok := c.recs[zone]
	if !ok || time.Since(time.Unix(r.At, 0)) > ttl {
		return "unknown", 0
	}
	if r.EgressOK {
		return "ok", 0
	}
	return "fail", r.FailStreak
}
