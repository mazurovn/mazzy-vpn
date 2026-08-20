// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package netdiag

import (
	"testing"

	"github.com/mazurovn/mazzy-vpn/core/netadapter"
)

func TestAnalyzeNoUplinkFails(t *testing.T) {
	r := Analyze([]netadapter.Adapter{
		{Name: "lo", Loopback: true, Up: true},
	})
	if r.Healthy() {
		t.Fatal("no physical uplink must FAIL")
	}
}

func TestAnalyzeRecommendsWiredAndFlagsWifiVPN(t *testing.T) {
	adapters := []netadapter.Adapter{
		{Name: "enp3s0", Up: true, IPv4: []string{"192.168.1.10/24"}},
		{Name: "wlp2s0", Up: true, Wireless: true, IPv4: []string{"192.168.1.20/24"}},
		{Name: "vpnaw0", Up: true, Virtual: true},
	}
	r := Analyze(adapters)
	if r.RecommendedUplink != "enp3s0" {
		t.Errorf("should recommend wired enp3s0, got %q", r.RecommendedUplink)
	}
	if r.ConflictingVPN != "vpnaw0" {
		t.Errorf("should detect conflicting VPN vpnaw0, got %q", r.ConflictingVPN)
	}
	if !r.Healthy() {
		t.Error("uplink present should not FAIL")
	}
}

func TestAnalyzeBothUplinksInfo(t *testing.T) {
	adapters := []netadapter.Adapter{
		{Name: "enp3s0", Up: true, IPv4: []string{"192.168.1.10/24"}},
		{Name: "wlp2s0", Up: true, Wireless: true, IPv4: []string{"192.168.1.20/24"}},
	}
	r := Analyze(adapters)
	found := false
	for _, f := range r.Findings {
		if f.Title == "Both wired and Wi‑Fi are up" {
			found = true
		}
	}
	if !found {
		t.Error("should note both uplinks are available")
	}
}
