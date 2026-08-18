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
	"github.com/mazurovn/mazzy-vpn/core/settings"
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

// resolveUplink returns the physical uplink to pin egress to (U7): an explicit
// --uplink flag wins, else the PreferredUplink setting, else "" (default route).
func resolveUplink(args []string) string {
	if v := flagValue(args, "--uplink"); v != "" {
		return v
	}
	return settingsUplink()
}

// connectOpts builds connect.Options with the resolved uplink.
func connectOpts(uplink string) connect.Options {
	return connect.Options{LogLevel: wireguard.LogError, Uplink: uplink}
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

	t := translator()

	// Single-flight: only one mutation at a time (parity acquire_action_lock).
	mu, err := lock.Acquire(lockDir())
	if err != nil {
		fmt.Fprintln(os.Stderr, t.T("cli.err.lock"))
		return 1
	}
	defer mu.Unlock()

	fmt.Println(t.Tf("cli.connect.connecting", safeDisplay(filepath.Base(path)), proto.Title()))
	uplink := resolveUplink(args)
	if uplink != "" {
		fmt.Println(t.Tf("cli.connect.pin_uplink", safeDisplay(uplink)))
	}
	conn, err := connect.Up(ctx, proto, cfg, connectOpts(uplink))
	if err != nil {
		fmt.Fprintln(os.Stderr, t.T("cli.connect.failed"), err)
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
	fmt.Print(t.T("cli.connect.verifying"))
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
		fmt.Println(t.T("cli.connect.ok"))
		fmt.Println(t.Tf("cli.connect.interface", conn.Interface))
		fmt.Println(t.Tf("cli.connect.egress", snap.EgressIP))
		fmt.Println(t.Tf("cli.connect.protocol", proto.Title()))
		nfy.Connected(zoneName, snap.EgressIP)
		maybeAutoMimic(ctx)
		go recordZoneScore(ctx, strings.TrimSuffix(zoneName, filepath.Ext(zoneName)))
	} else {
		fmt.Println(t.Tf("cli.connect.not_confirmed", conn.Interface, snap.Reason))
		fmt.Println("  The tunnel may still be establishing, or this server is not routing traffic.")
		fmt.Println("  Try another zone (mazzy-vpn test) if this persists.")
		nfy.Failed(zoneName, snap.Reason)
	}

	autoReconnect := !hasFlag(args, "--no-reconnect")
	if autoReconnect {
		fmt.Println(t.T("cli.connect.dashboard_reconnect"))
	} else {
		fmt.Println(t.T("cli.connect.dashboard"))
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
			fmt.Println(t.Tf("cli.connect.egress_lost", consecutiveFail, safeDisplay(zoneName)))
			nfy.Reconnecting(zoneName, s.Reason)
			_ = conn.Down(ctx)
			newConn, rerr := connect.Up(ctx, proto, cfg, connectOpts(uplink))
			if rerr != nil {
				fmt.Fprintf(os.Stderr, "  reconnect failed: %v\n", rerr)
				nfy.Failed(zoneName, rerr.Error())
				consecutiveFail = 0
				continue
			}
			conn = newConn
			rs := lc.WaitProtected(ctx, conn.Interface, 15*time.Second)
			if rs.Protected() {
				fmt.Println(t.Tf("cli.connect.reconnected", rs.EgressIP))
				nfy.Reconnected(zoneName, rs.EgressIP)
			} else {
				fmt.Println(t.Tf("cli.connect.not_confirmed", conn.Interface, rs.Reason))
				nfy.Failed(zoneName, rs.Reason)
			}
			consecutiveFail = 0
		}
	}

	fmt.Println(t.T("cli.connect.disconnecting"))
	if err := conn.Down(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "teardown error:", err)
		return 1
	}
	nfy.Disconnected(zoneName)
	_ = st.SetDesired(core.DesiredDown)
	fmt.Println(t.T("cli.connect.disconnected"))
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

	t := translator()
	switch {
	case snap.Protected():
		fmt.Printf("%-10s %s\n", t.T("cli.status.state"), t.T("cli.status.protected"))
		fmt.Printf("%-10s %s\n", t.T("cli.status.interface"), liveIface)
		fmt.Printf("%-10s %s\n", t.T("cli.status.egress"), snap.EgressIP)
		if profileName != "" {
			fmt.Printf("%-10s %s\n", t.T("cli.status.profile"), safeDisplay(profileName))
		}
	case snap.LinkUp:
		fmt.Printf("%-10s %s\n", t.T("cli.status.state"), t.Tf("cli.status.linkup", snap.Reason))
		fmt.Printf("%-10s %s\n", t.T("cli.status.interface"), liveIface)
	default:
		fmt.Printf("%-10s %s\n", t.T("cli.status.state"), t.T("cli.status.down"))
		if profileName != "" {
			fmt.Printf("%-10s %s (%s)\n", t.T("cli.status.last"), safeDisplay(profileName), protoName)
		}
	}
	return 0
}

// detectLiveInterface returns the first present Mazzy VPN interface, or "".
func detectLiveInterface() string {
	for _, name := range core.ManagedInterfaces() {
		if _, err := net.InterfaceByName(name); err == nil {
			return name
		}
	}
	return ""
}

// settingsUplink returns the user's preferred uplink from settings ("" if none).
func settingsUplink() string {
	return settings.NewStore().Load().PreferredUplink
}
