// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"os"

	"strconv"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/guard"
	"github.com/mazurovn/mazzy-vpn/core/netexec"
	"github.com/mazurovn/mazzy-vpn/core/routes"
)

// cmdRecover forcibly tears down ANY Mazzy VPN tunnel and guards, returning the
// host to a clean state (no tunnels, no leftover nftables/ip rules). This is
// the "panic button" to get back to plain Wi‑Fi. It is safe to run even when
// nothing is up. Requires root.
func cmdRecover(ctx context.Context, args []string) int {
	if !requireRoot("recover") {
		return 1
	}
	r := netexec.ExecRunner{}
	steps := 0
	run := func(desc, bin string, a ...string) {
		if _, err := r.Run(ctx, bin, a...); err == nil {
			fmt.Printf("  ✔ %s\n", desc)
			steps++
		}
	}

	t := translator()
	fmt.Println(t.T("cli.recover.restoring"))

	// 1. Remove our tunnel interfaces.
	for _, iface := range core.ManagedInterfaces() {
		if _, err := r.Run(ctx, "ip", "link", "show", iface); err == nil {
			run("removed interface "+iface, "ip", "link", "del", iface)
		}
	}

	// 2. Remove our nftables guard tables (including the CONNMARK table, N5).
	for _, tbl := range []string{guard.IPv6GuardTable, guard.TransitionGuardTable, guard.ConnmarkTable} {
		run("removed nft table "+tbl, "nft", "delete", "table", "inet", tbl)
	}

	// 3. Remove our policy-routing rules (may exist in duplicate). Each rule del
	// succeeds once per existing copy and then fails with ENOENT, so we count
	// only the real deletions and report honestly — a panic button must not claim
	// to have cleared rules it never touched (audit P1-D).
	mark := strconv.Itoa(routes.DefaultMark)
	rulesCleared := 0
	for i := 0; i < 4; i++ {
		for _, spec := range [][]string{
			{"-4", "rule", "del", "not", "fwmark", mark, "table", mark},
			{"-6", "rule", "del", "not", "fwmark", mark, "table", mark},
			{"-4", "rule", "del", "table", "main", "suppress_prefixlength", "0"},
			{"-6", "rule", "del", "table", "main", "suppress_prefixlength", "0"},
		} {
			if _, err := r.Run(ctx, "ip", spec...); err == nil {
				rulesCleared++
			}
		}
	}
	if rulesCleared > 0 {
		fmt.Printf("  ✔ cleared %d policy-routing rule(s)\n", rulesCleared)
		steps += rulesCleared
	} else {
		fmt.Println("  • no policy-routing rules to clear")
	}

	// 4. Flush our routing table.
	run("flushed table "+mark, "ip", "route", "flush", "table", mark)

	// 5. Clear the persisted intent so nothing tries to resume.
	_ = newStore().SetDesired("down")

	if hasFlag(args, "--reset-catalog") {
		fmt.Println("  ⚠ --reset-catalog: removing managed profiles")
		_ = os.RemoveAll(newCatalog().Dir)
		fmt.Println("  ✔ catalog reset")
	}

	fmt.Println(t.Tf("cli.recover.done", steps))
	fmt.Println("  Verify with: mazzy-vpn status")
	return 0
}

// cmdDisconnect brings down the active tunnel gracefully (lighter than recover).
func cmdDisconnect(ctx context.Context, _ []string) int {
	if !requireRoot("disconnect") {
		return 1
	}
	t := translator()
	iface := detectLiveInterface()
	if iface == "" {
		fmt.Println(t.T("cli.disconnect.none"))
		return 0
	}
	r := netexec.ExecRunner{}
	if _, err := r.Run(ctx, "ip", "link", "del", iface); err != nil {
		fmt.Fprintf(os.Stderr, "failed to remove %s: %v\n", iface, err)
		return 1
	}
	for _, tbl := range []string{guard.IPv6GuardTable, guard.TransitionGuardTable, guard.ConnmarkTable} {
		_, _ = r.Run(ctx, "nft", "delete", "table", "inet", tbl)
	}
	_ = newStore().SetDesired("down")
	fmt.Println(t.Tf("cli.disconnect.ok", iface))
	return 0
}
