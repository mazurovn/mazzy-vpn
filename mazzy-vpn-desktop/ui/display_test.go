// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package ui

import (
	"testing"

	"github.com/mazurovn/mazzy-vpn/desktop/panel"
)

func TestDisplayKeepsConfiguredPortWhenConnectFails(t *testing.T) {
	snap := panel.Snapshot{
		Phase:    panel.PhaseError,
		Server:   "helsinki",
		Protocol: "WireGuard",
		Profiles: []panel.Profile{{
			Name: "helsinki", Protocol: "WireGuard", Endpoint: "203.0.113.10:51820",
		}},
	}
	server, protocol, transport := display(snap)
	if server != "helsinki" || protocol != "WireGuard" || transport != "udp/51820" {
		t.Fatalf("got %q %q %q", server, protocol, transport)
	}
}

func TestDisplayKeepsSettledFallbackPort(t *testing.T) {
	snap := panel.Snapshot{
		Phase:     panel.PhaseConnected,
		Server:    "helsinki",
		Protocol:  "WireGuard",
		Transport: "udp/443",
		Profiles: []panel.Profile{{
			Name: "helsinki", Protocol: "WireGuard", Endpoint: "203.0.113.10:51820",
		}},
	}
	_, _, transport := display(snap)
	if transport != "udp/443" {
		t.Fatalf("transport %q", transport)
	}
}
