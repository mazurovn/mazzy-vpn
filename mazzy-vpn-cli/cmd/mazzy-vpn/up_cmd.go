// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mazurovn/mazzy-vpn/core/measure"
)

// cmdUp connects by managed profile NAME, or picks the best zone with --best/
// --auto. This is the ergonomic connect path (no file paths).
func cmdUp(ctx context.Context, args []string) int {
	cat := newCatalog()

	var name string
	auto := hasFlag(args, "--best") || hasFlag(args, "--auto")
	for _, a := range args {
		if a != "" && a[0] != '-' {
			name = a
			break
		}
	}

	if auto || name == "" {
		if name == "" && !auto {
			// No name given and not explicitly auto: fall back to auto-best for
			// convenience, but tell the user.
			if cat.Count() == 0 {
				fmt.Fprintln(os.Stderr, "no managed profiles; import first: mazzy-vpn import <DIR>")
				return 2
			}
			fmt.Println("No profile named; selecting the best reachable zone...")
			auto = true
		}
		if auto {
			targets, err := targetsFromCatalog(cat)
			if err != nil || len(targets) == 0 {
				fmt.Fprintln(os.Stderr, "no profiles with endpoints to rank")
				return 1
			}
			m := newMeasurer()
			ranked := m.RankBest(ctx, targets)
			best, ok := measure.BestAlive(ranked)
			if !ok {
				fmt.Fprintln(os.Stderr, "no reachable server found; check your connection (mazzy-vpn netdiag)")
				return 1
			}
			name = best.Name
			if best.ICMPAlive {
				fmt.Printf("Best zone: %s (%d ms, ✔ alive)\n", name, best.LatencyMS)
			} else {
				fmt.Printf("Best zone: %s (no ICMP reply; may still work)\n", name)
			}
		}
	}

	entry, err := cat.Get(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "profile %q not found (see: mazzy-vpn profiles)\n", name)
		return 1
	}
	// Reuse the file-based connect path with the managed file.
	return cmdConnect(ctx, []string{entry.File})
}

// cmdAuto ranks all reachable zones and returns them best-first so the caller
// (or a user) can connect with automatic failover. Without root it prints the
// ordered plan; with root it connects to the best zone.
func cmdAuto(ctx context.Context, args []string) int {
	cat := newCatalog()
	targets, err := targetsFromCatalog(cat)
	if err != nil || len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "no profiles with endpoints; import first: mazzy-vpn import <DIR>")
		return 1
	}
	fmt.Println("Ranking zones by reachability and latency...")
	ranked := newMeasurer().RankBest(ctx, targets)

	// Prefer ICMP-alive servers; they are proven reachable.
	reachable := make([]string, 0, len(ranked))
	for _, r := range ranked {
		if r.ICMPAlive {
			reachable = append(reachable, r.Name)
		}
	}
	if len(reachable) == 0 {
		// Fall back to UDP-reachable if nothing answered ICMP.
		for _, r := range ranked {
			if r.Reachable {
				reachable = append(reachable, r.Name)
			}
		}
	}
	if len(reachable) == 0 {
		fmt.Fprintln(os.Stderr, "no reachable server found; check your connection (mazzy-vpn netdiag)")
		return 1
	}

	fmt.Printf("Connection plan (best first): %v\n", reachable)
	if os.Geteuid() != 0 {
		fmt.Println("Run with sudo to auto-connect with failover:")
		fmt.Printf("  sudo mazzy-vpn auto\n")
		return 0
	}

	// With root: connect to the best zone. (Foreground; real failover across
	// zones needs the daemon mode planned for a later step.)
	best, _ := cat.Get(reachable[0])
	if best == nil {
		return 1
	}
	fmt.Printf("Auto-connecting to best zone: %s\n", best.Name)
	return cmdConnect(ctx, []string{best.File})
}
