// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/bootrecovery"
	"github.com/mazurovn/mazzy-vpn/core/connect"
	"github.com/mazurovn/mazzy-vpn/core/profile"
)

// recordingRunner captures every kernel command connect would issue.
type recordingRunner struct{ calls []string }

func (r *recordingRunner) Run(_ context.Context, bin string, args ...string) (string, error) {
	r.calls = append(r.calls, bin+" "+strings.Join(args, " "))
	return "", nil
}

func fullTunnel(t *testing.T) *profile.Config {
	t.Helper()
	c, err := profile.Parse("[Interface]\n" +
		"PrivateKey = AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=\n" +
		"Address = 10.8.0.2/32\n" +
		"Jc=4\nJmin=40\nJmax=70\nS1=50\nS2=100\nH1=1\nH2=2\nH3=3\nH4=4\n" +
		"[Peer]\n" +
		"PublicKey = AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=\n" +
		"AllowedIPs = 0.0.0.0/0, ::/0\nEndpoint = h:51820\n")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TestScenarioBootGateBlocksConnectEndToEnd wires the REAL connect package with
// the REAL boot-recovery gate: after a reboot (no ready state) a connect must
// be refused before ANY kernel command runs. This is the parity guarantee that
// the daemon never "invents" a connection at boot.
func TestScenarioBootGateBlocksConnectEndToEnd(t *testing.T) {
	rr := &recordingRunner{}
	gate := &bootrecovery.Gate{Dir: t.TempDir(), Required: true} // no state = not ready

	_, err := connect.Up(context.Background(), core.AmneziaWG, fullTunnel(t), connect.Options{
		Runner:   rr,
		BootGate: gate,
	})
	if err != connect.ErrBootRecoveryPending {
		t.Fatalf("expected boot-recovery block, got %v", err)
	}
	if len(rr.calls) != 0 {
		t.Fatalf("blocked connect must issue zero kernel commands; got %v", rr.calls)
	}
}

// TestScenarioReadyGateArmsGuardFirst confirms that once the boot gate is
// ready, connect arms the IPv6 guard as the FIRST kernel action (fail-closed
// ordering) before it fails later at the real TUN (needs root).
func TestScenarioReadyGateArmsGuardFirst(t *testing.T) {
	rr := &recordingRunner{}
	gate := &bootrecovery.Gate{Dir: t.TempDir(), Required: true}
	if err := gate.Write(bootrecovery.Ready); err != nil {
		t.Fatal(err)
	}

	_, err := connect.Up(context.Background(), core.AmneziaWG, fullTunnel(t), connect.Options{
		Runner:   rr,
		BootGate: gate,
	})
	// It must proceed past the gate (not a boot-recovery error) and reach kernel
	// actions; it then fails at the real TUN create, which is expected here.
	if err == connect.ErrBootRecoveryPending {
		t.Fatal("ready gate must not block")
	}
	if len(rr.calls) == 0 {
		t.Fatal("ready gate should let connect reach kernel actions")
	}
	if !strings.HasPrefix(rr.calls[0], "nft") {
		t.Errorf("first kernel action must be the nft IPv6 guard, got %q", rr.calls[0])
	}
}
