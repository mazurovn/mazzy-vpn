// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package routes

import (
	"context"
	"strings"
	"testing"

	"github.com/mazurovn/mazzy-vpn/core/netexec/netexectest"
	"github.com/mazurovn/mazzy-vpn/core/profile"
)

func fullTunnelConf() *profile.Config {
	c, err := profile.Parse("[Interface]\n" +
		"PrivateKey = " + key(1) + "\n" +
		"Address = 10.8.0.2/32\n" +
		"[Peer]\n" +
		"PublicKey = " + key(2) + "\n" +
		"AllowedIPs = 0.0.0.0/0, ::/0\n" +
		"Endpoint = h:51820\n")
	if err != nil {
		panic(err)
	}
	return c
}

func key(b byte) string {
	// 32 bytes base64; value content irrelevant for routing tests.
	switch b {
	case 1:
		return "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="
	default:
		return "AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI="
	}
}

func TestUpInstallsPolicyRoutingForBothFamilies(t *testing.T) {
	fake := &netexectest.Fake{}
	cfg := fullTunnelConf()
	a := New(fake, "vpnaw0", cfg)
	if err := a.Up(context.Background(), cfg); err != nil {
		t.Fatalf("Up: %v", err)
	}
	joined := strings.Join(fake.Calls, "\n")

	// Link up + address.
	mustContain(t, joined, "ip link set up dev vpnaw0")
	mustContain(t, joined, "ip -4 address add 10.8.0.2/32 dev vpnaw0")

	// v4 policy routing (wg-quick add_default parity).
	mustContain(t, joined, "ip -4 rule add not fwmark 51820 table 51820")
	mustContain(t, joined, "ip -4 rule add table main suppress_prefixlength 0")
	mustContain(t, joined, "ip -4 route add 0.0.0.0/0 dev vpnaw0 table 51820")

	// v6 policy routing.
	mustContain(t, joined, "ip -6 rule add not fwmark 51820 table 51820")
	mustContain(t, joined, "ip -6 route add ::/0 dev vpnaw0 table 51820")
}

func TestUpUsesConfigFwMarkAsTable(t *testing.T) {
	fake := &netexectest.Fake{}
	cfg := fullTunnelConf()
	cfg.FwMark = 0xca6c // 51820... choose different
	cfg.FwMark = 1234
	a := New(fake, "vpnaw0", cfg)
	if err := a.Up(context.Background(), cfg); err != nil {
		t.Fatalf("Up: %v", err)
	}
	mustContain(t, strings.Join(fake.Calls, "\n"), "table 1234")
}

func TestDownRevertsPolicyRules(t *testing.T) {
	fake := &netexectest.Fake{}
	cfg := fullTunnelConf()
	a := New(fake, "vpnaw0", cfg)
	_ = a.Up(context.Background(), cfg)
	fake.Calls = nil
	if err := a.Down(context.Background()); err != nil {
		t.Fatalf("Down: %v", err)
	}
	joined := strings.Join(fake.Calls, "\n")
	mustContain(t, joined, "rule del not fwmark 51820 table 51820")
	mustContain(t, joined, "rule del table main suppress_prefixlength 0")
}

// TestG1EngineMarkMatchesRoutingTable locks the P0 fix: the effective fwmark
// used for the wireguard socket must equal the routing table number.
func TestG1EngineMarkMatchesRoutingTable(t *testing.T) {
	cfg := fullTunnelConf()
	mark := EffectiveMark(cfg)
	if mark != DefaultMark {
		t.Fatalf("default mark = %d, want %d", mark, DefaultMark)
	}
	a := New(&netexectest.Fake{}, "vpnaw0", cfg)
	if a.Table != mark {
		t.Fatalf("routes table %d != engine mark %d (G1 regression)", a.Table, mark)
	}
	// And with an explicit config fwmark both must track it.
	cfg.FwMark = 777
	if EffectiveMark(cfg) != 777 || New(&netexectest.Fake{}, "vpnaw0", cfg).Table != 777 {
		t.Fatal("explicit FwMark not honored consistently")
	}
}

// TestG2SrcValidMarkSetForV4 locks that we set src_valid_mark for IPv4.
func TestG2SrcValidMarkSetForV4(t *testing.T) {
	fake := &netexectest.Fake{}
	cfg := fullTunnelConf()
	a := New(fake, "vpnaw0", cfg)
	if err := a.Up(context.Background(), cfg); err != nil {
		t.Fatalf("Up: %v", err)
	}
	mustContain(t, strings.Join(fake.Calls, "\n"), "sysctl -q net.ipv4.conf.all.src_valid_mark=1")
}

func mustContain(t *testing.T, hay, needle string) {
	t.Helper()
	if !strings.Contains(hay, needle) {
		t.Errorf("missing %q in:\n%s", needle, hay)
	}
}

// TestUplinkPinsEndpointRoute verifies U7: with an uplink set, a host route for
// the endpoint IP is installed via that interface, and removed on Down.
func TestUplinkPinsEndpointRoute(t *testing.T) {
	fake := &netexectest.Fake{}
	cfg := fullTunnelConf()
	// fullTunnelConf uses Endpoint h:51820 (hostname) — override to an IP.
	cfg.Peers[0].Endpoint = "203.0.113.50:51820"
	a := New(fake, "vpnaw0", cfg)
	a.Uplink = "enp5s0"
	if err := a.Up(context.Background(), cfg); err != nil {
		t.Fatalf("Up: %v", err)
	}
	joined := strings.Join(fake.Calls, "\n")
	mustContain(t, joined, "ip -4 route add 203.0.113.50/32 dev enp5s0")

	fake.Calls = nil
	if err := a.Down(context.Background()); err != nil {
		t.Fatalf("Down: %v", err)
	}
	mustContain(t, strings.Join(fake.Calls, "\n"), "route del 203.0.113.50/32 dev enp5s0")
}

// TestNoUplinkSkipsEndpointRoute: without an uplink, no host route is added.
func TestNoUplinkSkipsEndpointRoute(t *testing.T) {
	fake := &netexectest.Fake{}
	cfg := fullTunnelConf()
	cfg.Peers[0].Endpoint = "203.0.113.50:51820"
	a := New(fake, "vpnaw0", cfg) // no Uplink
	_ = a.Up(context.Background(), cfg)
	if strings.Contains(strings.Join(fake.Calls, "\n"), "203.0.113.50/32") {
		t.Error("no uplink should not pin an endpoint route")
	}
}

// TestUplinkHostnameEndpointSkipped: a hostname endpoint is not pinned (needs
// resolution and may change), so no route is added even with an uplink.
func TestUplinkHostnameEndpointSkipped(t *testing.T) {
	fake := &netexectest.Fake{}
	cfg := fullTunnelConf() // Endpoint h:51820 (hostname)
	a := New(fake, "vpnaw0", cfg)
	a.Uplink = "enp5s0"
	_ = a.Up(context.Background(), cfg)
	if strings.Contains(strings.Join(fake.Calls, "\n"), "route add h") {
		t.Error("hostname endpoint must not be pinned")
	}
	// BUG-N3: a skipped (hostname) pin must NOT be recorded, so teardown does
	// not try to delete a route that was never added.
	if a.appliedEndpoint != "" {
		t.Errorf("appliedEndpoint should be empty for a skipped hostname pin, got %q", a.appliedEndpoint)
	}
	before := len(fake.Calls)
	_ = a.Down(context.Background())
	for _, c := range fake.Calls[before:] {
		if strings.Contains(c, "route del") {
			t.Errorf("teardown must not delete a non-existent endpoint route: %q", c)
		}
	}
}
