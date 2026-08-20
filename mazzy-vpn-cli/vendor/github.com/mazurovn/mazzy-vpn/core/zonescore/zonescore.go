// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package zonescore caches per-zone stealth quality (egress ASN cleanliness,
// stealth score, egress country) so the CLI can prefer the "cleanest" zones —
// those that look least like a VPN to detection systems — without re-probing
// every server. Scores are learned as zones are used and refreshed by age.
//
// The endpoint ASN of a VPN server is NOT a useful signal (see design doc 03):
// services see the server's EGRESS IP, which differs and is only observable
// after connecting. This cache records those post-connect observations.
package zonescore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Score is the cached stealth quality of one zone's egress.
type Score struct {
	Zone         string `json:"zone"`
	StealthScore int    `json:"stealth_score"` // 0..100, higher = cleaner
	IsDatacenter bool   `json:"is_datacenter"`
	EgressCC     string `json:"egress_country"`
	UpdatedAt    int64  `json:"updated_at"` // unix seconds
}

// Fresh reports whether the score is newer than ttl.
func (s Score) Fresh(ttl time.Duration, now time.Time) bool {
	return now.Sub(time.Unix(s.UpdatedAt, 0)) < ttl
}

// Cache is a file-backed map of zone -> Score.
type Cache struct {
	Path   string
	scores map[string]Score
}

// DefaultPath returns the per-user cache path (honors MAZZY_CONFIG_HOME).
func DefaultPath() string {
	if d := os.Getenv("MAZZY_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "zone-scores.json")
	}
	if h, err := os.UserConfigDir(); err == nil {
		return filepath.Join(h, "mazzy-vpn", "zone-scores.json")
	}
	return filepath.Join(os.TempDir(), "mazzy-vpn", "zone-scores.json")
}

// New returns a Cache at the default path (loaded lazily).
func New() *Cache { return &Cache{Path: DefaultPath()} }

// NewAt returns a Cache at a specific path (for tests).
func NewAt(path string) *Cache { return &Cache{Path: path} }

// load reads the cache file (empty map when absent).
func (c *Cache) load() {
	if c.scores != nil {
		return
	}
	c.scores = map[string]Score{}
	data, err := os.ReadFile(c.Path)
	if err != nil {
		return
	}
	var list []Score
	if json.Unmarshal(data, &list) == nil {
		for _, s := range list {
			c.scores[s.Zone] = s
		}
	}
}

// Get returns the cached score for a zone.
func (c *Cache) Get(zone string) (Score, bool) {
	c.load()
	s, ok := c.scores[zone]
	return s, ok
}

// Record stores/updates a zone's score and persists the cache.
func (c *Cache) Record(s Score) error {
	c.load()
	s.UpdatedAt = time.Now().Unix()
	c.scores[s.Zone] = s
	return c.save()
}

// save atomically writes the cache (temp + rename, 0600).
func (c *Cache) save() error {
	if err := os.MkdirAll(filepath.Dir(c.Path), 0o700); err != nil {
		return err
	}
	list := make([]Score, 0, len(c.scores))
	for _, s := range c.scores {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Zone < list[j].Zone })
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, c.Path)
}

// Rank orders the given zone names best-first using cached scores. Zones with a
// fresh, higher stealth score (and non-datacenter) come first; unknown/stale
// zones sort last so they get evaluated. It never drops a zone — it only
// reorders, so callers still have every candidate.
func (c *Cache) Rank(zones []string, ttl time.Duration) []string {
	c.load()
	now := time.Now()
	type ranked struct {
		name  string
		known bool
		score int
		dc    bool
	}
	rs := make([]ranked, len(zones))
	for i, z := range zones {
		s, ok := c.scores[z]
		if ok && s.Fresh(ttl, now) {
			rs[i] = ranked{z, true, s.StealthScore, s.IsDatacenter}
		} else {
			rs[i] = ranked{z, false, 0, false}
		}
	}
	sort.SliceStable(rs, func(i, j int) bool {
		// Known-good (fresh) zones first.
		if rs[i].known != rs[j].known {
			return rs[i].known
		}
		if rs[i].known {
			// Non-datacenter preferred, then higher score.
			if rs[i].dc != rs[j].dc {
				return !rs[i].dc
			}
			return rs[i].score > rs[j].score
		}
		return false // preserve input order for unknowns
	})
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.name
	}
	return out
}
