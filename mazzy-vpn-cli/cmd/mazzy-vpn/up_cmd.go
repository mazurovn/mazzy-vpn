// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"os"

	"time"

	"github.com/mazurovn/mazzy-vpn/core/measure"
	"github.com/mazurovn/mazzy-vpn/core/settings"
	"github.com/mazurovn/mazzy-vpn/core/zonescore"
)

// cmdUp connects by managed profile NAME, or picks the best zone with --best/
// --auto. This is the ergonomic connect path (no file paths).
func cmdUp(ctx context.Context, args []string) int {
	cat := newCatalog()
	t := translator()

	// P1-5: when AutoDiagnostics is enabled, run a quick network analysis before
	// connecting so obvious problems (no uplink, captive portal) surface first.
	// Suppressed with --no-diagnostics or in --json/script contexts.
	if set := settings.NewStore().Load(); set.AutoDiagnostics && !hasFlag(args, "--no-diagnostics") {
		fmt.Println(t.T("cli.netdiag.running"))
		cmdNetdiag(ctx, nil)
	}

	var name string
	clean := hasFlag(args, "--clean")
	auto := hasFlag(args, "--best") || hasFlag(args, "--auto") || clean
	// Detect the zone NAME via the shared value-flag-aware parser, so a value
	// token like `--uplink eth0` is never misread as the zone name (audit P2-7:
	// one parser for every subcommand, no per-command reimplementation).
	name = firstNonFlagValueAware(args)

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
			// Auto-best with real failover: hand the pick to the daemon, which
			// verifies egress and WALKS to another zone if this one handshakes but
			// does not route (gate finding #1: the old foreground `up --best` path
			// had NO failover and looped forever on a fast-ping/non-routing pick).
			// A named `up <zone>` still uses the direct single-shot path below.
			if !clean && os.Geteuid() == 0 {
				fmt.Println(t.T("cli.up.selecting_best"))
				return cmdDaemon(ctx, []string{"--best"})
			}
			m := newMeasurer()
			ranked := m.RankBest(ctx, targets)
			best, ok := measure.BestAlive(ranked)
			if !ok {
				fmt.Fprintln(os.Stderr, t.T("cli.up.no_reachable"))
				return 1
			}
			name = best.Name
			// --clean: among the CURRENTLY-LIVE zones only, prefer the ones the
			// stealth cache knows are cleanest (non-datacenter, high stealth score).
			// We rank strictly within the ICMP-alive set and keep the BestAlive pick
			// if the cache yields nothing, so --clean can never downgrade a proven-
			// live choice to a cached-clean-but-now-dead one (audit P1-G).
			if clean {
				live := []string{}
				for _, r := range ranked {
					if r.ICMPAlive {
						live = append(live, r.Name)
					}
				}
				if len(live) > 0 {
					if ranked2 := zonescore.New().Rank(live, 24*time.Hour); len(ranked2) > 0 && ranked2[0] != "" {
						name = ranked2[0]
						fmt.Println(t.Tf("cli.up.cleanest", safeDisplay(name)))
					}
				}
			} else if best.ICMPAlive {
				fmt.Println(t.Tf("cli.up.best_alive", safeDisplay(name), best.LatencyMS))
			} else {
				fmt.Println(t.Tf("cli.up.best_noicmp", safeDisplay(name)))
			}
		}
	}

	entry, err := cat.Get(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "profile %q not found (see: mazzy-vpn profiles)\n", name)
		return 1
	}
	// A running daemon owns the tunnel AND the mutation lock for its lifetime,
	// so the direct connect path below could only fail with "another operation
	// is in progress". Forward the zone as an intent instead — this also
	// RESUMES a daemon paused by `disconnect` (previously `up` right after
	// `disconnect` errored on the lock and the host stayed offline until
	// doctor --heal; observed live 2026-08-29).
	if os.Geteuid() == 0 {
		if snap, running, ferr := forwardToActiveDaemon(name); ferr != nil {
			fmt.Fprintf(os.Stderr, "request active daemon: %v\n", ferr)
			return 1
		} else if running {
			fmt.Printf("active daemon pid %d: connecting %s\n", snap.PID, safeDisplay(name))
			return waitDaemonProtected(60 * time.Second)
		}
	}
	// Reuse the file-based connect path with the managed file, forwarding the
	// connection flags the user passed (audit: --uplink / --no-reconnect were
	// previously dropped here so pinning an uplink via `up` silently did nothing).
	connectArgs := []string{entry.File}
	if up := flagValue(args, "--uplink"); up != "" {
		connectArgs = append(connectArgs, "--uplink", up)
	}
	if hasFlag(args, "--no-reconnect") {
		connectArgs = append(connectArgs, "--no-reconnect")
	}
	return cmdConnect(ctx, connectArgs)
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

	// With root: hand the pick to the self-healing daemon via the --best
	// sentinel so IT selects with the egress-history bias (reachcache) and
	// walks across live zones on egress failure. Passing a fixed zone here
	// would bypass that bias and could strand the user on an ICMP-alive but
	// non-routing server (audit P1-H, gate finding #1/#2).
	fmt.Println(t.Tf("cli.up.auto_connecting", safeDisplay(reachable[0])))
	return cmdDaemon(ctx, []string{"--best"})
}
