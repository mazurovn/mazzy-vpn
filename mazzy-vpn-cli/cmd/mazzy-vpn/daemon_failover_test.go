// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"testing"
	"time"

	"github.com/mazurovn/mazzy-vpn/core/measure"
)

// TestFilterQuarantinedNoPingPong is the regression guard for gate finding 3:
// when every zone is quarantined the filter returns EMPTY (caller holds), never
// the unfiltered list — so failover cannot re-elect a just-failed zone and
// oscillate A→B→A, churning the kill-switch and routes each cycle.
func TestFilterQuarantinedNoPingPong(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	targets := []measure.Target{{Name: "A"}, {Name: "B"}}
	avoid := map[string]time.Time{
		"A": now.Add(5 * time.Minute),
		"B": now.Add(5 * time.Minute),
	}
	if got := filterQuarantined(targets, avoid, now); len(got) != 0 {
		t.Fatalf("all-quarantined must yield empty (hold), got %v", got)
	}

	// A partially-quarantined set keeps only the healthy zone.
	avoid = map[string]time.Time{"A": now.Add(5 * time.Minute)}
	got := filterQuarantined(targets, avoid, now)
	if len(got) != 1 || got[0].Name != "B" {
		t.Fatalf("expected only B, got %v", got)
	}

	// An expired cooldown is honored again.
	avoid = map[string]time.Time{"A": now.Add(-time.Minute)}
	if got := filterQuarantined(targets, avoid, now); len(got) != 2 {
		t.Fatalf("expired cooldown must restore the zone, got %v", got)
	}

	// No avoid map → unchanged.
	if got := filterQuarantined(targets, nil, now); len(got) != 2 {
		t.Fatalf("nil avoid must pass through, got %v", got)
	}
}
