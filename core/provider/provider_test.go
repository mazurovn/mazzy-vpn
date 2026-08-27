// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package provider

import "testing"

func openai() *Provider {
	return &Provider{
		ID: "openai", DisplayName: "OpenAI",
		SupportedCountries: []string{"US", "DE", "GB"},
		ReasonCodePrefix:   "service.openai",
	}
}

func TestSupportsCaseInsensitive(t *testing.T) {
	p := openai()
	if !p.Supports("us") || !p.Supports("DE") {
		t.Error("supported countries should match case-insensitively")
	}
	if p.Supports("RU") {
		t.Error("RU is not supported")
	}
}

func TestSupportsUnrestricted(t *testing.T) {
	p := &Provider{ID: "x"} // no SupportedCountries = unrestricted
	if !p.Supports("ANY") {
		t.Error("empty supported list means unrestricted")
	}
}

// TestRegionReadyWhenConsistent is the AI-native happy path: egress country is
// supported AND matches the timezone -> ready (provider won't flag the VPN).
func TestRegionReadyWhenConsistent(t *testing.T) {
	r := CheckRegion(openai(), RegionInput{EgressCountry: "DE", TimezoneCountry: "DE"})
	if r.Verdict != Ready {
		t.Fatalf("expected ready, got %+v", r)
	}
	if !r.Supported || !r.Consistent || len(r.Mismatches) != 0 {
		t.Errorf("ready must be fully consistent: %+v", r)
	}
}

// TestRegionBlockedWhenUnsupported: egress in an unsupported country -> not-ready
// (this is the block the agent must avoid).
func TestRegionBlockedWhenUnsupported(t *testing.T) {
	r := CheckRegion(openai(), RegionInput{EgressCountry: "RU", TimezoneCountry: "RU"})
	if r.Verdict != NotReady {
		t.Fatalf("unsupported egress must be not-ready, got %+v", r)
	}
	if !hasMismatch(r, "region.provider.country-unsupported") {
		t.Errorf("expected unsupported mismatch: %+v", r.Mismatches)
	}
}

// TestRegionTimezoneMismatchIsBlocker: supported egress but timezone from a
// different country -> not-ready (classic VPN-detection trigger).
func TestRegionTimezoneMismatchIsBlocker(t *testing.T) {
	r := CheckRegion(openai(), RegionInput{EgressCountry: "DE", TimezoneCountry: "US"})
	if r.Verdict != NotReady {
		t.Fatalf("egress/timezone mismatch must be not-ready, got %+v", r)
	}
	if !hasMismatch(r, "region.country.egress-timezone-mismatch") {
		t.Errorf("expected egress-timezone mismatch: %+v", r.Mismatches)
	}
}

func TestRegionUnknownWithoutEgress(t *testing.T) {
	r := CheckRegion(openai(), RegionInput{TimezoneCountry: "DE"})
	if r.Verdict != Unknown {
		t.Fatalf("no egress country -> unknown, got %+v", r)
	}
}

func TestRegionTargetConstraints(t *testing.T) {
	// Want US, but egress is DE -> target mismatch even though DE is supported.
	r := CheckRegion(openai(), RegionInput{EgressCountry: "DE", TimezoneCountry: "DE", TargetCountry: "US"})
	if r.Verdict != NotReady {
		t.Fatalf("target mismatch must be not-ready, got %+v", r)
	}
	if !hasMismatch(r, "region.target.egress-mismatch") {
		t.Errorf("expected target egress mismatch: %+v", r.Mismatches)
	}
	if r.HintCountry != "US" {
		t.Errorf("hint should be target US, got %q", r.HintCountry)
	}
}

func TestRegistryGetAndIDs(t *testing.T) {
	reg := &Registry{Providers: []*Provider{openai(), {ID: "google"}}}
	if reg.Get("openai") == nil || reg.Get("missing") != nil {
		t.Error("Get should find openai and miss unknown")
	}
	ids := reg.IDs()
	if len(ids) != 2 || ids[0] != "openai" || ids[1] != "google" {
		t.Errorf("IDs order wrong: %v", ids)
	}
}

func hasMismatch(r RegionResult, code string) bool {
	for _, m := range r.Mismatches {
		if m == code {
			return true
		}
	}
	return false
}
