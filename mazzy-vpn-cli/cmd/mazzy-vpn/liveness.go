// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"fmt"
	"time"

	"github.com/mazurovn/mazzy-vpn/core/measure"
	"github.com/mazurovn/mazzy-vpn/core/reachcache"
)

// egressHistoryTTL is how long a recorded egress verdict (daemon observation
// or deep-probe result) is considered current for display and ranking bias.
const egressHistoryTTL = 6 * time.Hour

// livenessLabel renders a zone's recent REAL-egress history for list views.
// This is the column that stops ping from lying: "✔ routes" means the tunnel
// recently carried actual internet traffic; "✖ no-route" means it recently
// accepted a ping/handshake but forwarded nothing.
func livenessLabel(zone string) string {
	v, streak := reachcache.New().Verdict(zone, egressHistoryTTL)
	switch v {
	case "ok":
		return "✔ routes"
	case "fail":
		if streak > 1 {
			return fmt.Sprintf("✖ no-route ×%d", streak)
		}
		return "✖ no-route"
	}
	return "· untested"
}

// reorderByEgressHistory applies the shared egress-history bias to ranked
// measurement results: proven-routing zones first, recently-failed ones last,
// preserving the latency order within each tier. Membership never changes.
func reorderByEgressHistory(rs []measure.Result) []measure.Result {
	if len(rs) < 2 {
		return rs
	}
	names := make([]string, 0, len(rs))
	byName := make(map[string]measure.Result, len(rs))
	for _, r := range rs {
		names = append(names, r.Name)
		byName[r.Name] = r
	}
	out := make([]measure.Result, 0, len(rs))
	for _, n := range reachcache.New().Reorder(names, egressHistoryTTL) {
		out = append(out, byName[n])
	}
	return out
}

// listWindow returns [start,end) bounds of a `visible`-row window over `total`
// rows that keeps `cursor` in view — so long zone lists scroll instead of
// overflowing the terminal.
func listWindow(cursor, total, visible int) (int, int) {
	if visible <= 0 || total <= visible {
		return 0, total
	}
	start := cursor - visible/2
	if start < 0 {
		start = 0
	}
	if start+visible > total {
		start = total - visible
	}
	return start, start + visible
}
