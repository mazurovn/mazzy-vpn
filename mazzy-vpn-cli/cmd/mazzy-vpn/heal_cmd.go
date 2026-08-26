// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/mazurovn/mazzy-vpn/core/livecheck"
	"github.com/mazurovn/mazzy-vpn/core/runstatus"
)

// healConnection is the active rescue ladder behind `doctor --heal`: it climbs
// from the cheapest intervention to the heaviest, stopping at the first rung
// that restores a PROTECTED connection.
//
//  0. already protected            → nothing to do
//  1. daemon alive but paused      → resume via up-intent, wait
//  2. daemon alive but unhealthy   → stop it
//  3. leftover tunnels/guards      → recover (full clean, incl. legacy)
//  4. no working connection        → start a fresh daemon on the best zone
//
// Requires root: every rung beyond 0 mutates network state.
func healConnection(ctx context.Context) int {
	if !requireRoot("doctor --heal") {
		return 1
	}
	lc := livecheck.New()

	// Rung 0: healthy already?
	if iface := detectLiveInterface(); iface != "" {
		if s := lc.Check(ctx, iface); s.Protected() {
			fmt.Printf("✔ already protected (%s, egress %s) — nothing to heal\n", iface, safeDisplay(s.EgressIP))
			return 0
		}
	}

	snap, running := daemonRunning()

	// Rung 0.5: a daemon that is actively reconnecting/connecting is doing this
	// job already — killing it mid-recovery (e.g. from a cron heal) would force
	// a slower full recover+restart cycle and cause flapping. Give it a grace
	// window first; escalate only if it still cannot deliver egress.
	if running && (snap.State == runstatus.StateReconnect || snap.State == runstatus.StateConnecting) {
		fmt.Printf("⟳ daemon pid %d is mid-reconnect — giving it 90s before escalating...\n", snap.PID)
		if waitProtected(ctx, lc, 90*time.Second) {
			fmt.Println("✔ healed: daemon recovered on its own")
			return 0
		}
	}

	// Rung 1: a paused daemon just needs a resume order (we are root, so the
	// intent write reaches the root-owned runtime dir).
	if running && snap.State == runstatus.StatePaused {
		fmt.Println("⏸ daemon is paused — resuming...")
		if err := writeDesired("", "up"); err == nil {
			if waitProtected(ctx, lc, 60*time.Second) {
				fmt.Println("✔ healed: daemon resumed and egress confirmed")
				return 0
			}
			fmt.Println("… resume did not restore egress; escalating")
		}
	}

	// Rung 2: an alive-but-unhealthy daemon is stopped so recovery and a fresh
	// start cannot race its auto-reconnect.
	if running {
		fmt.Printf("⟳ daemon pid %d is not delivering a protected connection — stopping it...\n", snap.PID)
		_ = recordDownIntent()
		if pid, ok := signalDaemonPID(); ok {
			if !waitDaemonExit(pid, 35*time.Second) {
				// A wedged daemon may hold an armed kill-switch: do NOT abort here
				// (that would leave the host fail-closed). Escalate to SIGKILL and
				// fall through to recover, which clears the guard tables.
				fmt.Printf("⚠ daemon pid %d ignored SIGTERM — sending SIGKILL\n", pid)
				if p, ferr := os.FindProcess(pid); ferr == nil {
					_ = p.Signal(syscall.SIGKILL)
				}
				_ = waitDaemonExit(pid, 5*time.Second)
			}
		}
	}

	// Rung 3: force-clean any leftover tunnels/guards/rules (also clears a
	// stuck fail-closed kill-switch — the classic "no VPN and no internet").
	fmt.Println("🧹 cleaning leftover network state...")
	if code := cmdRecover(ctx, nil); code != 0 {
		return code
	}

	// Rung 4: fresh start on the best live zone, detached so the healed tunnel
	// survives this command exiting.
	fmt.Println("⚡ starting a fresh daemon on the best zone...")
	if code := cmdDaemon(ctx, []string{"--best", "--background"}); code != 0 {
		fmt.Println("✖ could not start a daemon — check: mazzy-vpn diagnose")
		return code
	}
	if waitProtected(ctx, lc, 75*time.Second) {
		fmt.Println("✔ healed: fresh daemon up and egress confirmed")
		return 0
	}
	fmt.Println("✖ daemon started but egress is not confirmed yet — watch: mazzy-vpn (dashboard) or mazzy-vpn diagnose")
	return 1
}

// waitProtected polls the live interface until egress is confirmed or the
// deadline passes. Heartbeat state alone is not trusted — the point of heal is
// verifying the connection actually works.
func waitProtected(ctx context.Context, lc *livecheck.Checker, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if iface := detectLiveInterface(); iface != "" {
			if s := lc.Check(ctx, iface); s.Protected() {
				return true
			}
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(3 * time.Second):
		}
	}
	return false
}
