// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Command mazzy-core-smoke is the privileged autonomy proof for backlog C1-8.
// It brings a real AmneziaWG/WireGuard tunnel up from a .conf file using ONLY
// mazzy-core (embedded amneziawg-go + native routes/dns/guard) — no awg-quick,
// no wg-quick, no jq, no external VPN tools.
//
// Usage (requires root / CAP_NET_ADMIN):
//
//	sudo mazzy-core-smoke up   <profile.conf> [--protocol amneziawg|wireguard]
//	sudo mazzy-core-smoke check                 # print interface + routes
//
// The "up" mode brings the tunnel online, prints status, waits for SIGINT,
// then tears everything down cleanly.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/connect"
	"github.com/mazurovn/mazzy-vpn/core/engine/wireguard"
	"github.com/mazurovn/mazzy-vpn/core/profile"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "up":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		os.Exit(cmdUp(os.Args[2], parseProtocol(os.Args[3:])))
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: mazzy-core-smoke up <profile.conf> [--protocol amneziawg|wireguard]")
}

func parseProtocol(args []string) core.Protocol {
	proto := core.AmneziaWG
	for i := 0; i < len(args); i++ {
		if args[i] == "--protocol" && i+1 < len(args) {
			if p, ok := core.CanonicalProtocol(args[i+1]); ok {
				proto = p
			}
		}
	}
	return proto
}

func cmdUp(path string, proto core.Protocol) int {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "error: must run as root (needs CAP_NET_ADMIN to create TUN)")
		return 1
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read profile:", err)
		return 1
	}
	cfg, err := profile.Parse(string(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse profile:", err)
		return 1
	}
	if problems := profile.Validate(proto, cfg); len(problems) != 0 {
		fmt.Fprintln(os.Stderr, "invalid profile:", problems)
		return 1
	}

	ctx := context.Background()
	fmt.Printf("mazzy-core: bringing up %s from %s (no external *-quick tools)\n", proto.Title(), path)
	conn, err := connect.Up(ctx, proto, cfg, connect.Options{LogLevel: wireguard.LogError})
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect up:", err)
		return 1
	}
	fmt.Printf("UP: interface=%s protocol=%s\n", conn.Interface, conn.Protocol)
	fmt.Println("Tunnel is live. Press Ctrl+C to tear down.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nmazzy-core: tearing down...")
	if err := conn.Down(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "teardown error:", err)
		return 1
	}
	fmt.Println("DOWN: clean")
	return 0
}
