// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mazurovn/mazzy-vpn/core/livecheck"
)

// printDashboard renders a compact live status block for an active connection.
// It is called right after connect and on each refresh tick.
func printDashboard(_ context.Context, _ *livecheck.Checker, iface, proto string, s livecheck.Snapshot) {
	status := "DISCONNECTED"
	badge := "✖"
	switch {
	case s.Protected():
		status = "PROTECTED"
		badge = "✔"
	case s.LinkUp:
		status = "LINK UP (no egress)"
		badge = "⚠"
	}
	now := time.Now().Format("15:04:05")
	fmt.Printf("  [%s] %s %-20s iface=%s proto=%s", now, badge, status, iface, proto)
	if s.EgressIP != "" {
		fmt.Printf(" egress=%s", safeDisplay(s.EgressIP))
	}
	if s.Reason != "" && !s.Protected() {
		fmt.Printf(" (%s)", safeDisplay(s.Reason))
	}
	fmt.Println()
}
