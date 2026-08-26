// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
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

func TestKillSwitchRuleset(t *testing.T) {
	cf := &captureFake{}
	g := New(cf)
	if err := g.InstallKillSwitch(context.Background(), 51820, []string{"vpnaw0"}); err != nil {
		t.Fatalf("install: %v", err)
	}
	rs := cf.lastRuleset
	for _, want := range []string{
		"table inet " + TransitionGuardTable,
		"type filter hook output priority -150; policy accept;",
		`oifname "lo" accept`,
		"meta mark 51820 accept",
		"reject with icmpx type admin-prohibited",
	} {
		if !strings.Contains(rs, want) {
			t.Errorf("kill-switch ruleset missing %q:\n%s", want, rs)
		}
	}
	// Idempotency: pre-delete the table before applying.
	if !strings.Contains(strings.Join(cf.Calls, "\n"), "nft delete table inet "+TransitionGuardTable) {
		t.Errorf("expected pre-delete for idempotency; calls=%v", cf.Calls)
	}
}

// TestKillSwitchRejectsMaliciousIfaceNames: names that could inject nft rules
// must be dropped before ruleset interpolation.
func TestKillSwitchRejectsMaliciousIfaceNames(t *testing.T) {
	cf := &captureFake{}
	g := New(cf)
	bad := []string{`vpn"0`, "a b", "x\nreject", ""}
	if err := g.InstallKillSwitch(context.Background(), 51820, append(bad, "vpnaw0")); err != nil {
		t.Fatalf("install: %v", err)
	}
	rs := cf.lastRuleset
	if !strings.Contains(rs, `oifname { "vpnaw0" } accept`) {
		t.Errorf("valid iface must survive filtering:\n%s", rs)
	}
	for _, b := range bad {
		if b != "" && strings.Contains(rs, b) {
			t.Errorf("malicious iface %q leaked into the ruleset:\n%s", b, rs)
		}
	}
}
