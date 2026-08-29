// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mazurovn/mazzy-vpn/core/runstatus"
)

// cmdReconnect forces a FRESH connection right now — the menu/TUI "Reconnect"
// button. Unlike `up` (which no-ops when the daemon believes the current zone
// is fine) it always drops the tunnel and reconnects, because the human
// pressing it can see the connection is not actually working even when the
// probes say otherwise.
//
//   - daemon running: send a "reconnect" intent. Zone defaults to --best so the
//     daemon re-ranks with its live cooldowns + the shared egress history and
//     lands on a proven-working server, not merely the fastest ping.
//   - no daemon: start the self-healing daemon on the requested zone.
//
// Usage: sudo mazzy-vpn reconnect [NAME]
func cmdReconnect(ctx context.Context, args []string) int {
	if !requireRoot("reconnect") {
		return 1
	}
	zone := firstNonFlagValueAware(args)
	if zone == "" {
		zone = "--best"
	}
	t := translator()
	if snap, running := daemonRunning(); running {
		if err := writeDesired(zone, "reconnect"); err != nil {
			fmt.Fprintln(os.Stderr, "reconnect:", err)
			return 1
		}
		fmt.Printf("reconnect sent to daemon pid %d (%s → %s)\n", snap.PID, safeDisplay(snap.Zone), safeDisplay(zone))
		return waitDaemonProtected(60 * time.Second)
	}
	fmt.Println(t.T("cli.up.selecting_best"))
	return cmdDaemon(ctx, []string{zone, "--background"})
}

// waitDaemonProtected polls the daemon heartbeat until it confirms real egress,
// so intent-forwarding commands give the caller an honest verdict instead of
// "request sent". Bounded: intent tick (10s) + connect + egress confirmation
// take ~40s worst-case.
func waitDaemonProtected(within time.Duration) int {
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		time.Sleep(3 * time.Second)
		s, ok := runstatus.Read()
		if !ok {
			continue
		}
		if s.State == runstatus.StateProtected && s.Fresh(15*time.Second) {
			fmt.Printf("✔ connected: %s egress %s\n", safeDisplay(s.Zone), safeDisplay(s.Egress))
			return 0
		}
	}
	fmt.Fprintf(os.Stderr, "daemon did not confirm egress within %s — check: mazzy-vpn status / doctor\n", within)
	return 1
}
