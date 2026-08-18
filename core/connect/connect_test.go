// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package connect

import (
	"context"
	"strings"
	"testing"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/bootrecovery"
	"github.com/mazurovn/mazzy-vpn/core/dns"
	"github.com/mazurovn/mazzy-vpn/core/guard"
	"github.com/mazurovn/mazzy-vpn/core/profile"
	"github.com/mazurovn/mazzy-vpn/core/routes"
)

// scriptRunner fails the Nth matching command so we can test fail-closed
// unwinding without a real kernel. It records the call order.
type scriptRunner struct {
	calls  []string
	failOn string // substring; first matching call returns an error
}

func (s *scriptRunner) Run(_ context.Context, bin string, args ...string) (string, error) {
	call := bin + " " + strings.Join(args, " ")
	s.calls = append(s.calls, call)
	if s.failOn != "" && strings.Contains(call, s.failOn) {
		return "", &testErr{s.failOn}
	}
	return "", nil
}

type testErr struct{ m string }

func (e *testErr) Error() string { return "forced failure: " + e.m }

func fullTunnelCfg(t *testing.T) *profile.Config {
	t.Helper()
	c, err := profile.Parse("[Interface]\n" +
		"PrivateKey = AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=\n" +
		"Address = 10.8.0.2/32\n" +
		"Jc=4\nJmin=40\nJmax=70\nS1=50\nS2=100\nH1=1\nH2=2\nH3=3\nH4=4\n" +
		"[Peer]\n" +
		"PublicKey = AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=\n" +
		"AllowedIPs = 0.0.0.0/0, ::/0\n" +
		"Endpoint = h:51820\n")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TestUpArmsIPv6GuardBeforeAnythingElse verifies the fail-closed ordering: the
// IPv6 guard install must be the very first kernel action. We force the guard
// install to "succeed" but fail immediately after (at engine) is not possible
// without root, so instead we check ordering by failing the guard step and
// asserting nothing else ran.
func TestUpArmsIPv6GuardFirst(t *testing.T) {
	sr := &scriptRunner{failOn: "InstallIPv6Guard-not-a-real-cmd"}
	// Make the guard's nft apply fail by failing the nft command.
	sr.failOn = "nft -f"
	cfg := fullTunnelCfg(t)
	_, err := Up(context.Background(), core.AmneziaWG, cfg, Options{Runner: sr})
	if err == nil {
		t.Fatal("expected failure when IPv6 guard cannot be armed")
	}
	if !strings.Contains(err.Error(), "IPv6 guard") {
		t.Fatalf("expected IPv6 guard error, got %v", err)
	}
	// The very first kernel call must be the guard delete/apply, never an
	// interface/route command.
	if len(sr.calls) == 0 {
		t.Fatal("no calls recorded")
	}
	first := sr.calls[0]
	if !strings.HasPrefix(first, "nft") {
		t.Errorf("first kernel action = %q, want an nft guard command", first)
	}
	for _, c := range sr.calls {
		if strings.Contains(c, "ip -4 route") || strings.Contains(c, "ip link") {
			t.Errorf("routing ran despite guard failure: %q", c)
		}
	}
}

// TestUnsupportedProtocol guards the protocol check.
func TestUnsupportedProtocol(t *testing.T) {
	if _, err := Up(context.Background(), core.OpenVPN, &profile.Config{}, Options{}); err == nil {
		t.Fatal("expected unsupported protocol error")
	}
}

// TestBootRecoveryGateBlocksBeforeAnyKernelAction locks that a non-ready boot
// gate blocks the connection and performs NO kernel action at all.
func TestBootRecoveryGateBlocksBeforeAnyKernelAction(t *testing.T) {
	sr := &scriptRunner{}
	// A required gate with no state file = not ready = blocked.
	gate := &bootrecovery.Gate{Dir: t.TempDir(), Required: true}
	cfg := fullTunnelCfg(t)
	_, err := Up(context.Background(), core.AmneziaWG, cfg, Options{Runner: sr, BootGate: gate})
	if err != ErrBootRecoveryPending {
		t.Fatalf("expected ErrBootRecoveryPending, got %v", err)
	}
	if len(sr.calls) != 0 {
		t.Fatalf("blocked connect must issue zero kernel commands; got %v", sr.calls)
	}
}

// TestBootRecoveryReadyAllowsProceeding confirms a ready gate does not block
// (it then fails later at the real TUN, which needs root — that's fine).
func TestBootRecoveryReadyAllowsProceeding(t *testing.T) {
	sr := &scriptRunner{}
	gate := &bootrecovery.Gate{Dir: t.TempDir(), Required: true}
	if err := gate.Write(bootrecovery.Ready); err != nil {
		t.Fatal(err)
	}
	cfg := fullTunnelCfg(t)
	_, err := Up(context.Background(), core.AmneziaWG, cfg, Options{Runner: sr, BootGate: gate})
	// Must NOT be the boot-recovery error; it proceeds past the gate.
	if err == ErrBootRecoveryPending {
		t.Fatal("ready gate must not block")
	}
	// The guard (first kernel action) should have been attempted.
	if len(sr.calls) == 0 {
		t.Fatal("ready gate should let the connection reach kernel actions")
	}
}

// TestTeardownIdempotentAndUnified verifies audit N1: Down and unwind share one
// reverse-order chain, each layer is reverted once, and a second call is a
// safe no-op (flags cleared).
func TestTeardownIdempotentAndUnified(t *testing.T) {
	sr := &scriptRunner{}
	cfg := fullTunnelCfg(t)
	c := &Conn{
		guard:  guard.New(sr),
		routes: routes.New(sr, "vpnaw0", cfg),
		dns:    dns.New(sr, "vpnaw0"),
		// engine intentionally nil (no real TUN in a unit test).
		ipv6GuardOn: true,
		routesOn:    true,
		dnsOn:       true,
		connmarkOn:  true,
	}
	if err := c.Down(context.Background()); err != nil {
		t.Fatalf("first Down should succeed best-effort: %v", err)
	}
	// All flags must be cleared after teardown.
	if c.ipv6GuardOn || c.routesOn || c.dnsOn || c.connmarkOn {
		t.Errorf("flags not cleared: guard=%v routes=%v dns=%v connmark=%v",
			c.ipv6GuardOn, c.routesOn, c.dnsOn, c.connmarkOn)
	}
	calls := len(sr.calls)
	// A second teardown (or unwind) must be a no-op: no new kernel calls.
	c.unwind(context.Background())
	if len(sr.calls) != calls {
		t.Errorf("second teardown made %d extra calls; must be a no-op", len(sr.calls)-calls)
	}
}
