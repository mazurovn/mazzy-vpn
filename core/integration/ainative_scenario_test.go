// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package integration

import (
	"testing"

	"github.com/mazurovn/mazzy-vpn/core/control"
	"github.com/mazurovn/mazzy-vpn/core/provider"
)

// TestScenarioAINativeTwoPlanes exercises both AI-native planes end to end:
//
//	Plane 1 (control): an external harness is authorized to reach an agent.
//	Plane 2 (provider): that agent's egress is region-checked against an LLM so
//	                    the provider does not block/flag the session.
//
// This is the "AI-native VPN" core: connect agents from outside (Plane 1) and
// connect agents to LLMs reliably avoiding blocks (Plane 2).
func TestScenarioAINativeTwoPlanes(t *testing.T) {
	// --- Plane 1: control plane ---
	plane := control.NewPlane()
	if err := plane.Register(control.Participant{ID: "harness-ci", Kind: control.Harness}); err != nil {
		t.Fatal(err)
	}
	if err := plane.Register(control.Participant{ID: "agent-coder", Kind: control.Agent}); err != nil {
		t.Fatal(err)
	}
	// Deny-by-default: the harness cannot reach the agent yet.
	if plane.CanReach("harness-ci", "agent-coder") {
		t.Fatal("must be denied before explicit Allow")
	}
	if err := plane.Allow("harness-ci", "agent-coder"); err != nil {
		t.Fatal(err)
	}
	if !plane.CanReach("harness-ci", "agent-coder") {
		t.Fatal("harness should reach the agent after Allow")
	}

	// --- Plane 2: provider region check for the agent's LLM egress ---
	openai := &provider.Provider{
		ID: "openai", DisplayName: "OpenAI",
		SupportedCountries: []string{"US", "DE", "GB"},
	}

	// Good egress: DE supported and consistent with timezone -> ready.
	ready := provider.CheckRegion(openai, provider.RegionInput{
		EgressCountry: "DE", TimezoneCountry: "DE",
	})
	if ready.Verdict != provider.Ready {
		t.Fatalf("consistent DE egress should be ready for the agent, got %+v", ready)
	}

	// Blocked egress: RU unsupported -> not-ready, the agent must pick another
	// exit before relying on the tunnel.
	blocked := provider.CheckRegion(openai, provider.RegionInput{
		EgressCountry: "RU", TimezoneCountry: "RU",
	})
	if blocked.Verdict != provider.NotReady {
		t.Fatalf("RU egress should be not-ready (blocked), got %+v", blocked)
	}
}
