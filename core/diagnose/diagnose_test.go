// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package diagnose

import (
	"strings"
	"testing"
)

func TestNoUplinkIsRootCause(t *testing.T) {
	r := Analyze(Signal{HasUplink: false})
	if r.Healthy() {
		t.Fatal("no uplink must be critical")
	}
	if !strings.Contains(r.Problems[0].Title, "uplink") {
		t.Errorf("expected uplink problem first, got %q", r.Problems[0].Title)
	}
	// Should stop at root cause (only 1 problem).
	if len(r.Problems) != 1 {
		t.Errorf("root cause should short-circuit, got %d problems", len(r.Problems))
	}
}

func TestTunnelUpNoEgressWithDeadServer(t *testing.T) {
	r := Analyze(Signal{
		HasUplink: true, InternetOK: true, ProfilesImported: 10, AnyServerAlive: true,
		TunnelIface: "vpnaw0", TunnelLinkUp: true, EgressOK: false,
		ServerName: "FinlandHelsinkiS4", ServerAlive: false,
	})
	if r.Healthy() {
		t.Fatal("no egress must be critical")
	}
	p := r.Problems[0]
	if !strings.Contains(p.Title, "no egress") {
		t.Errorf("expected no-egress problem, got %q", p.Title)
	}
	// The fix should point to switching to a live server (dead server root).
	if !strings.Contains(p.Cause, "FinlandHelsinkiS4") {
		t.Errorf("cause should name the dead server, got %q", p.Cause)
	}
	if !strings.Contains(p.Fix, "--best") {
		t.Errorf("fix should suggest --best, got %q", p.Fix)
	}
}

func TestHealthyConnection(t *testing.T) {
	r := Analyze(Signal{
		HasUplink: true, InternetOK: true, ProfilesImported: 10, AnyServerAlive: true,
		TunnelIface: "vpnaw0", TunnelLinkUp: true, EgressOK: true, EgressIP: "203.0.113.9",
		DNSOK: true, ServerAlive: true,
	})
	if !r.Healthy() {
		t.Fatalf("healthy connection should be healthy, got %+v", r.Problems)
	}
	if !strings.Contains(r.Summary, "Protected") {
		t.Errorf("summary should say protected, got %q", r.Summary)
	}
}

func TestNoServersImported(t *testing.T) {
	r := Analyze(Signal{HasUplink: true, InternetOK: true, ProfilesImported: 0})
	if r.Healthy() || !strings.Contains(r.Problems[0].Title, "No VPN profiles") {
		t.Errorf("expected no-profiles critical, got %+v", r.Problems)
	}
}

func TestNoLiveServer(t *testing.T) {
	r := Analyze(Signal{HasUplink: true, InternetOK: true, ProfilesImported: 10, AnyServerAlive: false})
	if r.Healthy() || !strings.Contains(r.Problems[0].Title, "No live VPN server") {
		t.Errorf("expected no-live-server critical, got %+v", r.Problems)
	}
}

func TestConflictVPNWhenConnected(t *testing.T) {
	r := Analyze(Signal{
		HasUplink: true, InternetOK: true, ProfilesImported: 10, AnyServerAlive: true,
		TunnelIface: "vpnaw0", TunnelLinkUp: true, EgressOK: true, EgressIP: "203.0.113.9",
		DNSOK: true, ServerAlive: true, ConflictVPN: "tun0",
	})
	found := false
	for _, p := range r.Problems {
		if strings.Contains(p.Title, "Another VPN") {
			found = true
		}
	}
	if !found {
		t.Error("should warn about conflicting VPN even when healthy")
	}
}
