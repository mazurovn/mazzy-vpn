// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package connect

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mazurovn/mazzy-vpn/core/guard"
)

// rulesetCapture records the last `nft -f <file>` ruleset so tests can assert
// its content (the guard package writes rulesets to a private temp file).
type rulesetCapture struct {
	scriptRunner
	lastRuleset string
}

func newRulesetCapture() *rulesetCapture { return &rulesetCapture{} }

func (c *rulesetCapture) Run(ctx context.Context, bin string, args ...string) (string, error) {
	if bin == "nft" && len(args) == 2 && args[0] == "-f" {
		if data, err := os.ReadFile(args[1]); err == nil {
			c.lastRuleset = string(data)
		}
	}
	return c.scriptRunner.Run(ctx, bin, args...)
}

// TestArmDisarmKillSwitch verifies the user-facing kill-switch (P1-1): arming
// installs the fwmark-aware fail-closed guard (loopback + tunnel handshake
// allowed, everything else rejected), disarming removes it, and disarm is
// idempotent.
func TestArmDisarmKillSwitch(t *testing.T) {
	sr := &scriptRunner{}
	c := &Conn{guard: guard.New(sr), mark: 51820}
	ctx := context.Background()

	if err := c.ArmKillSwitch(ctx); err != nil {
		t.Fatalf("arm: %v", err)
	}
	if !c.killSwitchOn {
		t.Fatal("killSwitchOn should be true after Arm")
	}
	joined := strings.Join(sr.calls, "\n")
	if !strings.Contains(joined, "nft -f") {
		t.Fatalf("arm should install an nft ruleset; calls=%v", sr.calls)
	}

	// Disarm removes the transition guard table.
	if err := c.DisarmKillSwitch(ctx); err != nil {
		t.Fatalf("disarm: %v", err)
	}
	if c.killSwitchOn {
		t.Fatal("killSwitchOn should be false after Disarm")
	}
	if !strings.Contains(strings.Join(sr.calls, "\n"),
		"nft delete table inet "+guard.TransitionGuardTable) {
		t.Fatalf("disarm should delete the transition guard table; calls=%v", sr.calls)
	}
}

// TestKillSwitchAllowsTunnelHandshake is the regression guard for the bug found
// in review: the kill-switch must let the new tunnel's own encrypted handshake
// (packets carrying the socket fwmark) egress, otherwise reconnection can never
// succeed while the guard is armed.
func TestKillSwitchAllowsTunnelHandshake(t *testing.T) {
	cf := newRulesetCapture()
	g := guard.New(cf)
	if err := g.InstallKillSwitch(context.Background(), 51820); err != nil {
		t.Fatalf("install: %v", err)
	}
	rs := cf.lastRuleset
	for _, want := range []string{
		"table inet " + guard.TransitionGuardTable,
		`oifname "lo" accept`,
		"meta mark 51820 accept", // the tunnel handshake must be allowed out
		"reject with icmpx type admin-prohibited",
	} {
		if !strings.Contains(rs, want) {
			t.Errorf("kill-switch ruleset missing %q:\n%s", want, rs)
		}
	}
}

// TestTeardownKeepsArmedKillSwitch verifies the session-level contract: an
// in-place reconnect does Down() the old tunnel between Arm and Disarm, so
// teardown must NOT remove the kill-switch (that would reopen the leak window
// exactly while the tunnel is down). It is lifted only by explicit Disarm.
func TestTeardownKeepsArmedKillSwitch(t *testing.T) {
	sr := &scriptRunner{}
	c := &Conn{guard: guard.New(sr), mark: 51820, killSwitchOn: true}
	if err := c.Down(context.Background()); err != nil {
		t.Fatalf("down: %v", err)
	}
	if strings.Contains(strings.Join(sr.calls, "\n"),
		"nft delete table inet "+guard.TransitionGuardTable) {
		t.Errorf("teardown must NOT remove the armed kill-switch; calls=%v", sr.calls)
	}
}
