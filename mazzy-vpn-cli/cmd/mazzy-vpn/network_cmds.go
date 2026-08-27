// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/catalog"
	"github.com/mazurovn/mazzy-vpn/core/measure"
	"github.com/mazurovn/mazzy-vpn/core/netadapter"
	"github.com/mazurovn/mazzy-vpn/core/profile"
)

// rankBudget bounds a whole test/rank pass so a large catalog can never look
// like a hang: even with every server dead, the probe wave is capped and the
// caller returns with whatever completed. Sized for ~50 profiles at the pooled
// concurrency (each probe ≤ the per-probe timeout).
func rankBudget(n int) time.Duration {
	switch {
	case n <= 8:
		return 12 * time.Second
	case n <= 24:
		return 20 * time.Second
	default:
		return 30 * time.Second
	}
}

// rankWithProgress ranks targets under a bounded deadline while printing a live
// single-line "probing k/N" indicator to stderr (interactive only), so the user
// always sees forward motion instead of a frozen screen. It returns the ranked
// results. Progress is suppressed in --json/non-TTY contexts by passing
// quiet=true.
func rankWithProgress(ctx context.Context, m *measure.Measurer, targets []measure.Target, quiet bool) []measure.Result {
	cctx, cancel := context.WithTimeout(ctx, rankBudget(len(targets)))
	defer cancel()
	var cb func(done, total int)
	if !quiet {
		cb = func(done, total int) {
			fmt.Fprintf(os.Stderr, "\r  probing servers %d/%d…", done, total)
			if done == total {
				fmt.Fprint(os.Stderr, "\r\033[K") // clear the line when finished
			}
		}
	}
	return m.RankBestProgress(cctx, targets, cb)
}

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
	results := rankWithProgress(ctx, newMeasurer(), targets, false)
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
	jsonOut := hasFlag(args, "--json")
	results := rankWithProgress(ctx, m, targets, jsonOut)

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
		return 0
	}
	fmt.Println(translator().Tf("cli.test.testing", len(targets)) + "\n")
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
		fmt.Printf("%-24s %-10s %s\n", safeDisplay(r.Name), lat, status)
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
	results := rankWithProgress(ctx, m, targets, hasFlag(args, "--json"))
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
		fmt.Println(translator().Tf("cli.up.best_alive", safeDisplay(best.Name), best.LatencyMS))
	} else {
		fmt.Println(translator().Tf("cli.up.best_noicmp", safeDisplay(best.Name)))
	}
	fmt.Println(translator().Tf("cli.best.connect_with", safeDisplay(best.Name)))
	return 0
}
