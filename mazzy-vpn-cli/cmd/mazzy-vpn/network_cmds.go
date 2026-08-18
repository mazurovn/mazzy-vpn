// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/catalog"
	"github.com/mazurovn/mazzy-vpn/core/measure"
	"github.com/mazurovn/mazzy-vpn/core/netadapter"
	"github.com/mazurovn/mazzy-vpn/core/profile"
)

// newMeasurer builds a Measurer that pings through the physical uplink so an
// active VPN (e.g. AdGuard) cannot mask true server liveness. Falls back to the
// default route when no uplink is detected.
func newMeasurer() *measure.Measurer {
	uplink := ""
	if adapters, err := netadapter.List(); err == nil {
		if rec, _, ok := netadapter.Recommend(adapters); ok {
			uplink = rec.Name
		}
	}
	return measure.NewViaUplink(uplink)
}

// targetsFromCatalog builds measurement targets from managed profiles that
// expose an endpoint (WireGuard/AmneziaWG).
func targetsFromCatalog(cat *catalog.Catalog) ([]measure.Target, error) {
	entries, err := cat.List()
	if err != nil {
		return nil, err
	}
	var targets []measure.Target
	for _, e := range entries {
		// Only WireGuard/AmneziaWG profiles are supported by the engine and have
		// a parseable [Peer] Endpoint; skip OpenVPN for now.
		if e.Protocol == core.OpenVPN {
			continue
		}
		data, err := os.ReadFile(e.File)
		if err != nil {
			continue
		}
		cfg, err := profile.Parse(string(data))
		if err != nil {
			continue
		}
		if ep := cfg.Endpoint(); ep != "" {
			targets = append(targets, measure.Target{Name: e.Name, Endpoint: ep})
		}
	}
	return targets, nil
}

// measureCatalogPings probes every managed profile with an endpoint and returns
// a map name -> human ping string ("12 ms" or "—"). Used by the interactive
// zone picker to show latency next to each profile.
func measureCatalogPings(ctx context.Context, cat *catalog.Catalog) map[string]string {
	out := map[string]string{}
	targets, err := targetsFromCatalog(cat)
	if err != nil || len(targets) == 0 {
		return out
	}
	results := newMeasurer().RankBest(ctx, targets)
	for _, r := range results {
		if r.Reachable {
			out[r.Name] = fmt.Sprintf("%d ms", r.LatencyMS)
		} else {
			out[r.Name] = "—"
		}
	}
	return out
}

// cmdTest probes reachability + latency of managed profiles' servers WITHOUT
// connecting, so the user can see which configs actually work.
func cmdTest(ctx context.Context, args []string) int {
	cat := newCatalog()
	targets, err := targetsFromCatalog(cat)
	if err != nil {
		fmt.Fprintln(os.Stderr, "catalog error:", err)
		return 1
	}
	if len(targets) == 0 {
		fmt.Println(translator().T("cli.test.no_profiles"))
		return 1
	}
	m := newMeasurer()
	results := m.RankBest(ctx, targets)

	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
		return 0
	}
	fmt.Printf("Testing %d server(s)...\n\n", len(targets))
	fmt.Printf("%-24s %-10s %s\n", "NAME", "PING", "STATUS")
	alive := 0
	for _, r := range results {
		var status, lat string
		switch {
		case r.ICMPAlive:
			status = "✔ alive"
			lat = fmt.Sprintf("%d ms", r.LatencyMS)
			alive++
		case r.Reachable:
			status = "? no ICMP reply"
			lat = "-"
		default:
			status = "✖ unreachable"
			lat = "-"
		}
		fmt.Printf("%-24s %-10s %s\n", r.Name, lat, status)
	}
	fmt.Printf("\n%d/%d servers answered ICMP (alive).\n", alive, len(results))
	fmt.Println(measureNote)
	return 0
}

// measureNote explains the liveness signals.
const measureNote = "note: '✔ alive' = server answered ICMP ping (best signal). '? no ICMP reply' = endpoint resolves and a UDP socket opens, but the host did not answer ping (may still work, or may be down). Prefer alive servers."

// cmdBest probes all servers and prints the single best zone to connect to.
func cmdBest(ctx context.Context, args []string) int {
	cat := newCatalog()
	targets, err := targetsFromCatalog(cat)
	if err != nil || len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "no profiles with endpoints; import some first")
		return 1
	}
	m := newMeasurer()
	results := m.RankBest(ctx, targets)
	best, ok := measure.BestAlive(results)
	if !ok {
		fmt.Println(translator().T("cli.up.no_reachable"))
		return 1
	}
	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(best)
		return 0
	}
	if best.ICMPAlive {
		fmt.Printf("Best zone: %s (%d ms, ✔ alive)\n", best.Name, best.LatencyMS)
	} else {
		fmt.Printf("Best zone: %s (no ICMP reply; may still work)\n", best.Name)
	}
	fmt.Printf("Connect with: sudo mazzy-vpn up %s\n", best.Name)
	return 0
}
