// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/netexec"
	"github.com/mazurovn/mazzy-vpn/core/runstatus"
)

// cmdDisarm is the HARD reset: it returns the host to plain, VPN-free
// networking no matter what state the daemon, kill-switch or leftover rules
// are in. Unlike `disconnect` (graceful) and `recover` (cleanup while a daemon
// may keep running paused), disarm:
//
//  1. records the down-intent and KILLS the daemon (SIGTERM → SIGKILL);
//  2. reverts per-link DNS and flushes resolver caches, so name resolution
//     never stays pointed at a dead tunnel;
//  3. runs the full recover sweep: interfaces, ALL our nftables tables
//     (including the fail-closed kill-switch), policy-routing rules, routing
//     table, legacy leftovers;
//  4. removes the runtime status files so no ghost daemon is displayed;
//  5. VERIFIES plain internet actually works and says so honestly.
//
// This is the "give me my Wi-Fi back, no questions" button.
func cmdDisarm(ctx context.Context, args []string) int {
	if !requireRoot("disarm") {
		return 1
	}
	fmt.Println("⛔ HARD RESET: returning the host to plain networking (no VPN, no guards)")

	// 1. Stop the daemon, escalating to SIGKILL. Intent first so any survivor
	// pauses instead of re-creating what we tear down.
	if err := recordDownIntent(); err != nil {
		fmt.Fprintln(os.Stderr, "  warning:", err)
	}
	if snap, running := daemonRunning(); running {
		fmt.Printf("  ⏹ stopping daemon pid %d...\n", snap.PID)
		if pid, ok := signalDaemonPID(); ok {
			if !waitDaemonExit(pid, 10*time.Second) {
				fmt.Printf("  ⚠ SIGTERM ignored — SIGKILL pid %d\n", pid)
				if p, err := os.FindProcess(pid); err == nil {
					_ = p.Signal(syscall.SIGKILL)
				}
				_ = waitDaemonExit(pid, 5*time.Second)
			}
		}
	}

	// 2. DNS back to normal before the interfaces disappear, then flush caches.
	r := netexec.ExecRunner{}
	for _, iface := range core.ManagedInterfaces() {
		_, _ = r.Run(ctx, "resolvectl", "revert", iface)
	}
	_, _ = r.Run(ctx, "resolvectl", "flush-caches")
	fmt.Println("  ✔ DNS reverted to system defaults")

	// 3. Full network-state sweep (interfaces, nft tables incl. kill-switch,
	// policy rules, routes, legacy leftovers). Forward --purge-legacy if given.
	recArgs := []string{}
	if hasFlag(args, "--purge-legacy") {
		recArgs = append(recArgs, "--purge-legacy")
	}
	if code := cmdRecover(ctx, recArgs); code != 0 {
		return code
	}

	// 4. No ghost daemon in the dashboards.
	_ = os.Remove(runstatus.Path())

	// 5. Prove the plain uplink works. This is the whole point: the user must
	// know their internet is BACK, not assume it.
	fmt.Print("  verifying plain internet")
	ok := false
	for i := 0; i < 3 && !ok; i++ {
		fmt.Print(".")
		ok = plainInternetOK(ctx)
		if !ok {
			time.Sleep(2 * time.Second)
		}
	}
	fmt.Println()
	if ok {
		fmt.Println("✔ DISARMED: host is on plain networking and the internet works")
		return 0
	}
	fmt.Println("⚠ disarmed, but plain internet is NOT confirmed — the problem is outside the VPN")
	fmt.Println("  Check Wi-Fi/router, then: mazzy-vpn netdiag")
	return 1
}
