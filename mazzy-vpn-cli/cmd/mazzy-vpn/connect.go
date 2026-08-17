// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/connect"
	"github.com/mazurovn/mazzy-vpn/core/engine/wireguard"
	"github.com/mazurovn/mazzy-vpn/core/livecheck"
	"github.com/mazurovn/mazzy-vpn/core/lock"
	"github.com/mazurovn/mazzy-vpn/core/notify"
	"github.com/mazurovn/mazzy-vpn/core/profile"
	"github.com/mazurovn/mazzy-vpn/core/state"
)

// requireRoot returns an error string when not privileged. Creating a TUN and
// applying routes/nft needs CAP_NET_ADMIN.
func requireRoot(action string) bool {
	if os.Geteuid() != 0 {
		fmt.Fprintf(os.Stderr, "error: %s requires root (CAP_NET_ADMIN)\n", action)
		return false
	}
	return true
}

// loadProfile reads and parses a WireGuard/AmneziaWG profile, inferring the
// protocol from its obfuscation fields. OpenVPN is not yet supported by the
// embedded engine and is rejected with a clear message.
func loadProfile(path string) (core.Protocol, *profile.Config, error) {
	if strings.EqualFold(filepath.Ext(path), ".ovpn") {
		return "", nil, fmt.Errorf("OpenVPN (.ovpn) is not yet supported by the embedded engine; use an AmneziaWG/WireGuard (.conf) profile")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("read profile: %w", err)
	}
	cfg, err := profile.Parse(string(data))
	if err != nil {
		return "", nil, fmt.Errorf("parse profile: %w", err)
	}
	proto := core.WireGuard
	if cfg.HasAmneziaFields {
		proto = core.AmneziaWG
	}
	if problems := profile.Validate(proto, cfg); len(problems) != 0 {
		return "", nil, fmt.Errorf("invalid profile: %s", strings.Join(problems, "; "))
	}
	return proto, cfg, nil
}

// cmdConnect brings up a tunnel in the FOREGROUND, holding it until SIGINT/TERM,
// then tears it down cleanly. Daemonized/service mode lands in a later step.
func cmdConnect(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sudo mazzy-vpn connect PROFILE.conf")
		return 2
	}
	if !requireRoot("connect") {
		return 1
	}
	path := args[0]
	proto, cfg, err := loadProfile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// Single-flight: only one mutation at a time (parity acquire_action_lock).
	mu, err := lock.Acquire(lockDir())
	if err != nil {
		fmt.Fprintln(os.Stderr, "another mazzy-vpn operation is in progress")
		return 1
	}
	defer mu.Unlock()

	fmt.Printf("Connecting %s (%s)...\n", filepath.Base(path), proto.Title())
	conn, err := connect.Up(ctx, proto, cfg, connect.Options{LogLevel: wireguard.LogError})
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect failed:", err)
		return 1
	}

	// Persist the working intent so status/reconnect can find it.
	st := newStore()
	if err := st.Write(state.State{
		Protocol: proto, Profile: filepath.Base(path),
		Desired: core.DesiredUp, Mode: core.ModeNormal,
	}); err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not persist state:", err)
	}

	// Verify REAL egress before telling the user they are protected. This is
	// the difference between "an interface exists" and "the VPN actually works".
	fmt.Print("Verifying protected egress")
	lc := livecheck.New()
	spinDone := make(chan struct{})
	go func() {
		for {
			select {
			case <-spinDone:
				return
			case <-time.After(700 * time.Millisecond):
				fmt.Print(".")
			}
		}
	}()
	snap := lc.WaitProtected(ctx, conn.Interface, 20*time.Second)
	close(spinDone)
	fmt.Println()

	zoneName := filepath.Base(path)
	nfy := notify.New()
	if snap.Protected() {
		fmt.Printf("✔ CONNECTED and protected.\n")
		fmt.Printf("  interface : %s\n", conn.Interface)
		fmt.Printf("  egress IP : %s\n", snap.EgressIP)
		fmt.Printf("  protocol  : %s\n", proto.Title())
		nfy.Connected(zoneName, snap.EgressIP)
	} else {
		fmt.Printf("⚠ Interface %s is up, but egress is NOT confirmed: %s\n", conn.Interface, snap.Reason)
		fmt.Println("  The tunnel may still be establishing, or this server is not routing traffic.")
		fmt.Println("  Try another zone (mazzy-vpn test) if this persists.")
		nfy.Failed(zoneName, snap.Reason)
	}

	autoReconnect := !hasFlag(args, "--no-reconnect")
	if autoReconnect {
		fmt.Println("\nLive dashboard + auto-reconnect (Ctrl+C to disconnect):")
	} else {
		fmt.Println("\nLive dashboard (Ctrl+C to disconnect):")
	}

	// Live dashboard: refresh status every few seconds until interrupted, and
	// auto-reconnect if the egress drops for several consecutive checks.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	printDashboard(ctx, lc, conn.Interface, proto.Title(), snap)

	consecutiveFail := 0
	const failLimit = 3 // ~15s of no egress before reconnect
dashLoop:
	for {
		select {
		case <-sig:
			break dashLoop
		case <-ticker.C:
			s := lc.Check(ctx, conn.Interface)
			printDashboard(ctx, lc, conn.Interface, proto.Title(), s)
			if s.Protected() {
				consecutiveFail = 0
				continue
			}
			if !autoReconnect {
				continue
			}
			consecutiveFail++
			if consecutiveFail < failLimit {
				continue
			}
			// Egress lost: attempt an in-place reconnect.
			fmt.Printf("⟳ Egress lost (%d checks). Reconnecting %s...\n", consecutiveFail, zoneName)
			nfy.Reconnecting(zoneName, snap.Reason)
			_ = conn.Down(ctx)
			newConn, rerr := connect.Up(ctx, proto, cfg, connect.Options{LogLevel: wireguard.LogError})
			if rerr != nil {
				fmt.Fprintf(os.Stderr, "  reconnect failed: %v\n", rerr)
				nfy.Failed(zoneName, rerr.Error())
				consecutiveFail = 0
				continue
			}
			conn = newConn
			rs := lc.WaitProtected(ctx, conn.Interface, 15*time.Second)
			if rs.Protected() {
				fmt.Printf("✔ Reconnected. egress=%s\n", rs.EgressIP)
				nfy.Reconnected(zoneName, rs.EgressIP)
			} else {
				fmt.Printf("⚠ Reconnect did not confirm egress: %s\n", rs.Reason)
				nfy.Failed(zoneName, rs.Reason)
			}
			consecutiveFail = 0
		}
	}

	fmt.Println("\nDisconnecting...")
	if err := conn.Down(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "teardown error:", err)
		return 1
	}
	nfy.Disconnected(zoneName)
	_ = st.SetDesired(core.DesiredDown)
	fmt.Println("Disconnected.")
	return 0
}

// cmdStatus reports the persisted intent (foreground model has no daemon yet).
func cmdStatus(ctx context.Context, args []string) int {
	// Detect the REAL live interface first — the VPN may have been started by
	// the daemon, foreground connect, or manually. This is the truth source.
	liveIface := detectLiveInterface()
	lc := livecheck.New()
	var snap livecheck.Snapshot
	if liveIface != "" {
		snap = lc.Check(ctx, liveIface)
	}

	st := newStore()
	cur, _ := st.Read()
	profileName := ""
	protoName := ""
	if cur != nil {
		profileName = cur.Profile
		protoName = cur.Protocol.Title()
	}

	if hasFlag(args, "--json") {
		state := "down"
		if snap.Protected() {
			state = "protected"
		} else if snap.LinkUp {
			state = "link-up"
		}
		fmt.Printf(`{"state":%q,"interface":%q,"egress":%q,"profile":%q}`+"\n",
			state, liveIface, snap.EgressIP, profileName)
		return 0
	}

	switch {
	case snap.Protected():
		fmt.Printf("State:     ✔ PROTECTED\n")
		fmt.Printf("Interface: %s\n", liveIface)
		fmt.Printf("Egress IP: %s\n", snap.EgressIP)
		if profileName != "" {
			fmt.Printf("Profile:   %s\n", profileName)
		}
	case snap.LinkUp:
		fmt.Printf("State:     ⚠ LINK UP (egress not confirmed: %s)\n", snap.Reason)
		fmt.Printf("Interface: %s\n", liveIface)
	default:
		fmt.Println("State:     ✖ down (no active VPN interface)")
		if profileName != "" {
			fmt.Printf("Last:      %s (%s)\n", profileName, protoName)
		}
	}
	return 0
}

// detectLiveInterface returns the first present Mazzy VPN interface, or "".
func detectLiveInterface() string {
	for _, name := range []string{"vpnaw0", "vpnwg0"} {
		if _, err := net.InterfaceByName(name); err == nil {
			return name
		}
	}
	return ""
}
