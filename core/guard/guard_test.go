// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package guard

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mazurovn/mazzy-vpn/core/netexec/netexectest"
)

// captureFake records nft -f <file> calls and reads back the ruleset file so we
// can assert its content (parity with bash nft rulesets).
type captureFake struct {
	netexectest.Fake
	lastRuleset string
}

func (c *captureFake) Run(ctx context.Context, bin string, args ...string) (string, error) {
	if bin == "nft" && len(args) == 2 && args[0] == "-f" {
		if data, err := os.ReadFile(args[1]); err == nil {
			c.lastRuleset = string(data)
		}
	}
	return c.Fake.Run(ctx, bin, args...)
}

func TestIPv6GuardRuleset(t *testing.T) {
	cf := &captureFake{}
	g := New(cf)
	if err := g.InstallIPv6Guard(context.Background(), "vpnaw0"); err != nil {
		t.Fatalf("install: %v", err)
	}
	rs := cf.lastRuleset
	for _, want := range []string{
		"table inet " + IPv6GuardTable,
		"type filter hook output priority -140; policy drop;",
		"meta nfproto ipv4 accept",
		`oifname "lo" accept`,
		`oifname "vpnaw0" accept`,
	} {
		if !strings.Contains(rs, want) {
			t.Errorf("ruleset missing %q:\n%s", want, rs)
		}
	}
	// deletes table first for idempotency
	if !strings.Contains(strings.Join(cf.Calls, "\n"), "nft delete table inet "+IPv6GuardTable) {
		t.Errorf("expected pre-delete of table for idempotency; calls=%v", cf.Calls)
	}
}

func TestFailClosedRuleset(t *testing.T) {
	cf := &captureFake{}
	g := New(cf)
	if err := g.InstallFailClosed(context.Background()); err != nil {
		t.Fatalf("install: %v", err)
	}
	rs := cf.lastRuleset
	for _, want := range []string{
		"table inet " + TransitionGuardTable,
		"policy accept;",
		`oifname "lo" accept`,
		"reject with icmpx type admin-prohibited",
	} {
		if !strings.Contains(rs, want) {
			t.Errorf("fail-closed ruleset missing %q:\n%s", want, rs)
		}
	}
}

func TestInvalidInterfaceRejected(t *testing.T) {
	g := New(&netexectest.Fake{})
	if err := g.InstallIPv6Guard(context.Background(), "bad name!"); err == nil {
		t.Fatal("expected rejection of invalid interface name")
	}
}

func TestConnmarkRuleset(t *testing.T) {
	cf := &captureFake{}
	g := New(cf)
	if err := g.InstallConnmark(context.Background(), 51820); err != nil {
		t.Fatalf("install: %v", err)
	}
	rs := cf.lastRuleset
	for _, want := range []string{
		"table inet " + ConnmarkTable,
		"type filter hook prerouting priority -150",
		"meta l4proto udp meta mark set ct mark",
		"type filter hook postrouting priority -150",
		"meta l4proto udp mark 51820 ct mark set mark",
	} {
		if !strings.Contains(rs, want) {
			t.Errorf("connmark ruleset missing %q:\n%s", want, rs)
		}
	}
}
