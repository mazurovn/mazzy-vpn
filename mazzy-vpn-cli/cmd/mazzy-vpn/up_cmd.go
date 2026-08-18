// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"os"

	"time"

	"github.com/mazurovn/mazzy-vpn/core/measure"
	"github.com/mazurovn/mazzy-vpn/core/zonescore"
)

// cmdUp connects by managed profile NAME, or picks the best zone with --best/
// --auto. This is the ergonomic connect path (no file paths).
func cmdUp(ctx context.Context, args []string) int {
	cat := newCatalog()
	t := translator()

	var name string
	clean := hasFlag(args, "--clean")
	auto := hasFlag(args, "--best") || hasFlag(args, "--auto") || clean
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
				fmt.Fprintln(os.Stderr, t.T("cli.up.no_profiles"))
				return 2
			}
			fmt.Println(t.T("cli.up.selecting_best"))
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
				fmt.Fprintln(os.Stderr, t.T("cli.up.no_reachable"))
				return 1
			}
			name = best.Name
			// --clean: among the live zones, prefer the ones the stealth cache
			// knows are cleanest (non-datacenter, high stealth score).
			if clean {
				live := []string{}
				for _, r := range ranked {
					if r.ICMPAlive {
						live = append(live, r.Name)
					}
				}
				if len(live) > 0 {
					ranked2 := zonescore.New().Rank(live, 24*time.Hour)
					name = ranked2[0]
					fmt.Println(t.Tf("cli.up.cleanest", name))
				}
			} else if best.ICMPAlive {
				fmt.Println(t.Tf("cli.up.best_alive", name, best.LatencyMS))
			} else {
				fmt.Println(t.Tf("cli.up.best_noicmp", name))
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
	t := translator()
	targets, err := targetsFromCatalog(cat)
	if err != nil || len(targets) == 0 {
		fmt.Fprintln(os.Stderr, t.T("cli.up.no_profiles"))
		return 1
	}
	fmt.Println(t.T("cli.up.ranking"))
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
		fmt.Fprintln(os.Stderr, t.T("cli.up.no_reachable"))
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
	fmt.Println(t.Tf("cli.up.auto_connecting", best.Name))
	return cmdConnect(ctx, []string{best.File})
}
