// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package zonescore

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRecordAndGet(t *testing.T) {
	c := NewAt(filepath.Join(t.TempDir(), "zs.json"))
	if err := c.Record(Score{Zone: "NL1", StealthScore: 85, EgressCC: "NL"}); err != nil {
		t.Fatal(err)
	}
	s, ok := c.Get("NL1")
	if !ok || s.StealthScore != 85 || s.UpdatedAt == 0 {
		t.Fatalf("bad score: %+v ok=%v", s, ok)
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zs.json")
	c1 := NewAt(path)
	c1.Record(Score{Zone: "DE1", StealthScore: 70})
	// Fresh cache reads from disk.
	c2 := NewAt(path)
	if s, ok := c2.Get("DE1"); !ok || s.StealthScore != 70 {
		t.Fatalf("not persisted: %+v", s)
	}
}

func TestFresh(t *testing.T) {
	now := time.Unix(1000, 0)
	s := Score{UpdatedAt: 1000 - 60} // 60s old
	if !s.Fresh(2*time.Minute, now) {
		t.Error("60s old should be fresh within 2m TTL")
	}
	if s.Fresh(30*time.Second, now) {
		t.Error("60s old should be stale within 30s TTL")
	}
}

func TestRankPrefersCleanFresh(t *testing.T) {
	c := NewAt(filepath.Join(t.TempDir(), "zs.json"))
	c.Record(Score{Zone: "clean", StealthScore: 90, IsDatacenter: false})
	c.Record(Score{Zone: "dc", StealthScore: 90, IsDatacenter: true})
	c.Record(Score{Zone: "low", StealthScore: 40, IsDatacenter: false})

	got := c.Rank([]string{"low", "dc", "clean", "unknown"}, time.Hour)
	// clean (non-dc, high) first; unknown last (preserved order among unknowns).
	if got[0] != "clean" {
		t.Errorf("clean should rank first, got %v", got)
	}
	if got[len(got)-1] != "unknown" {
		t.Errorf("unknown should rank last, got %v", got)
	}
	// dc (datacenter) should rank below the non-dc clean but above low? No:
	// among known, non-dc preferred then score. low is non-dc 40, dc is dc 90.
	// non-dc beats dc, so low(non-dc) ranks above dc.
	idxLow, idxDC := indexOf(got, "low"), indexOf(got, "dc")
	if idxLow > idxDC {
		t.Errorf("non-datacenter 'low' should rank above datacenter 'dc': %v", got)
	}
}

func TestRankStaleTreatedAsUnknown(t *testing.T) {
	c := NewAt(filepath.Join(t.TempDir(), "zs.json"))
	c.Record(Score{Zone: "old", StealthScore: 90})
	// Force it stale by using a tiny TTL.
	got := c.Rank([]string{"old", "fresh-unknown"}, time.Nanosecond)
	// Both effectively unknown → input order preserved.
	if got[0] != "old" {
		t.Errorf("stale should preserve input order, got %v", got)
	}
}

func TestRankNeverDropsZones(t *testing.T) {
	c := NewAt(filepath.Join(t.TempDir(), "zs.json"))
	in := []string{"a", "b", "c"}
	got := c.Rank(in, time.Hour)
	if len(got) != len(in) {
		t.Errorf("rank must keep all zones: %v", got)
	}
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}
