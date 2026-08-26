// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/mazurovn/mazzy-vpn/core/livecheck"
	"github.com/mazurovn/mazzy-vpn/core/runstatus"
)

// printDashboard renders a compact live status block for an active connection.
// It is called right after connect and on each refresh tick (foreground path).
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

// drawLiveDashboard renders the rich dashboard block for the menu header when a
// background daemon is publishing a heartbeat. It shows the connection badge,
// egress, a latency sparkline graph, latency stats, recent errors and an error
// frequency estimate — so the user watches the connection from inside the menu
// instead of a blocking log. Returns false when there is no fresh heartbeat, so
// the caller can fall back to a one-shot live probe.
func drawLiveDashboard() bool {
	snap, ok := daemonRunning()
	if !ok {
		return false
	}
	t := translator()

	// A left-rule layout (no fragile right border) so Unicode-width glyphs and
	// localized labels never break alignment.
	const rule = "────────────────────────────────────────────────────────"
	badge, label := dashBadge(snap.State)
	up := ""
	if snap.StartedAt > 0 {
		up = shortDur(time.Since(time.Unix(snap.StartedAt, 0)))
	}
	mode := "fg"
	if snap.Background {
		mode = "bg"
	}

	fmt.Println("┌" + rule)
	fmt.Println("│ Mazzy VPN — live dashboard")
	fmt.Println("├" + rule)

	line := fmt.Sprintf("│ %s %s", badge, label)
	if snap.Egress != "" {
		line += "   egress " + safeDisplay(snap.Egress)
	}
	// A daemon exists (PID alive) but its status is old: say so instead of
	// silently rendering stale numbers as if they were current.
	if age := snap.HeartbeatAge(); age > 15*time.Second {
		line += fmt.Sprintf("   ⚠ status %s old", shortDur(age))
	}
	fmt.Println(line)
	fmt.Printf("│ zone %s · %s %s · %s\n",
		trunc(safeDisplay(snap.Zone), 24), t.T("cli.dash.uptime"), up, mode)
	if lh := linkHealthLine(snap); lh != "" {
		fmt.Println("│ " + trunc(lh, 54))
	}

	// Latency graph + stats.
	series := snap.LatencySeries()
	spark := runstatus.Sparkline(series, 40)
	mn, avg, mx := runstatus.LatencyStats(series)
	fmt.Printf("│ %s %s  %d/%d/%d ms  (%d/%d ok)\n",
		t.T("cli.dash.graph"), spark, mn, avg, mx, snap.Checks-snap.Fails, snap.Checks)

	// Error rate + recent errors.
	rate := snap.ErrorRatePerMin(10 * time.Minute)
	fmt.Printf("│ %s: %d · %.1f %s · reconnects %d\n",
		t.T("cli.dash.errors"), len(snap.Errors), rate, t.T("cli.dash.errrate"), snap.Reconnects)
	recent := snap.RecentErrors(2)
	if len(recent) == 0 {
		fmt.Println("│   " + t.T("cli.dash.no_errors"))
	} else {
		for _, e := range recent {
			ts := time.Unix(e.TS, 0).Format("15:04:05")
			fmt.Println("│   " + trunc(ts+" "+safeDisplay(e.Reason), 52))
		}
	}
	fmt.Println("└" + rule)
	return true
}

// dashBadge maps a runstatus.State to a glyph + label.
func dashBadge(s runstatus.State) (string, string) {
	switch s {
	case runstatus.StateProtected:
		return "✔", "PROTECTED"
	case runstatus.StateConnecting:
		return "…", "CONNECTING"
	case runstatus.StateReconnect:
		return "⟳", "RECONNECTING"
	case runstatus.StateLinkUp:
		return "⚠", "LINK UP"
	case runstatus.StatePaused:
		return "⏸", "PAUSED"
	default:
		return "✖", "DISCONNECTED"
	}
}

// fmtBytes renders a byte count compactly (1.2 GiB, 340 MiB, 12 KiB).
func fmtBytes(n int64) string {
	switch {
	case n < 0:
		return "0 B"
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.0f KiB", float64(n)/1024)
	case n < 1024*1024*1024:
		return fmt.Sprintf("%.0f MiB", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1024*1024*1024))
	}
}

// linkHealthLine renders the link-facts row shared by the menu dashboard and
// the TUI header: handshake age, traffic, loss and egress identity/stealth.
// Empty when the heartbeat has none of these yet (old daemon / just started).
func linkHealthLine(snap runstatus.Snapshot) string {
	parts := []string{}
	if snap.HandshakeAgeS > 0 {
		parts = append(parts, "hs "+shortDur(time.Duration(snap.HandshakeAgeS)*time.Second))
	}
	if snap.RxBytes > 0 || snap.TxBytes > 0 {
		parts = append(parts, "↓"+fmtBytes(snap.RxBytes)+" ↑"+fmtBytes(snap.TxBytes))
	}
	if snap.Checks > 0 {
		parts = append(parts, fmt.Sprintf("loss %.1f%%", snap.LossPercent()))
	}
	if snap.EgressCountry != "" {
		geo := safeDisplay(snap.EgressCountry)
		if snap.EgressCity != "" {
			geo += " " + safeDisplay(snap.EgressCity)
		}
		parts = append(parts, geo)
	}
	if snap.StealthScore > 0 {
		parts = append(parts, fmt.Sprintf("stealth %d", snap.StealthScore))
	}
	return strings.Join(parts, " · ")
}

// shortDur formats a duration compactly (e.g. 3m, 2h05m, 45s).
func shortDur(d time.Duration) string {
	d = d.Round(time.Second)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		h := int(d.Hours())
		m := int(d.Minutes()) - h*60
		return fmt.Sprintf("%dh%02dm", h, m)
	}
}

// trunc shortens s to at most n DISPLAY COLUMNS (not runes), adding an ellipsis
// when cut. Using display width keeps dashboard boxes aligned when zone/egress
// names contain double-width CJK or emoji glyphs — the app ships zh/ja/ko
// locales, so rune-count truncation previously misaligned the borders (audit
// P2-5). The ellipsis costs one column, so the visible content fits in n.
func trunc(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	budget := n - 1 // reserve one column for the ellipsis
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw > budget {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String() + "…"
}
