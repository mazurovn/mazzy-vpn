// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package provider implements the AI-native Plane 2 logic: deciding whether an
// agent/harness can reliably reach a given LLM/provider from the current VPN
// egress, and whether the egress region is *consistent* (egress country ==
// timezone country == provider-supported country) so the provider does not
// flag the session as a suspicious/blocked VPN.
//
// This is the core of "агенты → провайдеры/LLM, надёжно обходя блокировки":
// a provider that geo-blocks or challenges mismatched regions will break long
// agent sessions, so mazzy-core checks region consistency BEFORE the agent
// relies on the tunnel.
package provider

import "strings"

// Provider describes an LLM/AI service endpoint and where it is usable.
type Provider struct {
	ID                 string   `json:"id"`
	DisplayName        string   `json:"display_name"`
	ProbeEndpoints     []string `json:"probe_endpoints"`
	SupportedCountries []string `json:"supported_countries"`
	ReasonCodePrefix   string   `json:"reason_code_prefix"`
	ProbeStrategy      string   `json:"probe_strategy"`
}

// Supports reports whether the provider is usable from country cc (case
// insensitive). An empty SupportedCountries list means "unrestricted".
func (p *Provider) Supports(cc string) bool {
	if len(p.SupportedCountries) == 0 {
		return true
	}
	cc = strings.ToUpper(strings.TrimSpace(cc))
	for _, c := range p.SupportedCountries {
		if strings.ToUpper(c) == cc {
			return true
		}
	}
	return false
}

// Registry is the catalog of known providers.
type Registry struct {
	SchemaVersion int         `json:"schema_version"`
	Providers     []*Provider `json:"providers"`
}

// Get returns a provider by id, or nil.
func (r *Registry) Get(id string) *Provider {
	for _, p := range r.Providers {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// IDs returns all provider ids in registry order.
func (r *Registry) IDs() []string {
	out := make([]string, len(r.Providers))
	for i, p := range r.Providers {
		out[i] = p.ID
	}
	return out
}

// Verdict is the region-readiness outcome for reaching a provider.
type Verdict string

const (
	// Ready: egress country is supported and consistent with the timezone.
	Ready Verdict = "ready"
	// NotReady: at least one blocking mismatch (unsupported/inconsistent).
	NotReady Verdict = "not-ready"
	// Unknown: not enough signal (e.g. egress country undetermined).
	Unknown Verdict = "unknown"
)

// RegionInput is the observed environment used for a region check.
type RegionInput struct {
	EgressCountry   string // observed egress country code (may be empty)
	TimezoneCountry string // country derived from system timezone (may be empty)
	TargetCountry   string // optional desired country
}

// RegionResult is the structured result of a region check.
type RegionResult struct {
	ProviderID  string   `json:"provider_id"`
	Verdict     Verdict  `json:"verdict"`
	EgressCC    string   `json:"egress_country,omitempty"`
	HintCountry string   `json:"hint_country,omitempty"`
	Supported   bool     `json:"supported_by_provider"`
	Consistent  bool     `json:"country_consistent"`
	Mismatches  []string `json:"mismatches"`
}

// CheckRegion evaluates whether an agent can reliably use p from the observed
// environment. It mirrors the bash region_check_json verdict logic: the egress
// must be provider-supported AND consistent with the timezone; a target
// country, when given, must also match.
func CheckRegion(p *Provider, in RegionInput) RegionResult {
	res := RegionResult{ProviderID: p.ID, Verdict: NotReady}

	egress := strings.ToUpper(strings.TrimSpace(in.EgressCountry))
	tz := strings.ToUpper(strings.TrimSpace(in.TimezoneCountry))
	target := strings.ToUpper(strings.TrimSpace(in.TargetCountry))
	res.EgressCC = egress

	if egress == "" {
		// Without an egress country we cannot conclude readiness.
		res.Verdict = Unknown
		res.Mismatches = append(res.Mismatches, "region.egress.country-unknown")
		if target != "" {
			res.HintCountry = target
		}
		return res
	}

	// Provider support for the egress country.
	if p.Supports(egress) {
		res.Supported = true
	} else {
		res.Mismatches = append(res.Mismatches, "region.provider.country-unsupported")
	}

	// Egress vs timezone consistency (a VPN with a mismatched timezone is a
	// classic block/challenge trigger).
	if tz != "" {
		if egress == tz {
			if res.Supported {
				res.Consistent = true
			}
		} else {
			res.Mismatches = append(res.Mismatches, "region.country.egress-timezone-mismatch")
		}
	} else {
		res.Mismatches = append(res.Mismatches, "region.timezone.country-unknown")
	}

	// Optional target country constraints.
	if target != "" {
		if !p.Supports(target) {
			res.Mismatches = append(res.Mismatches, "region.target.country-unsupported")
		}
		if egress != target {
			res.Mismatches = append(res.Mismatches, "region.target.egress-mismatch")
		}
		if tz != "" && tz != target {
			res.Mismatches = append(res.Mismatches, "region.target.timezone-mismatch")
		}
		res.HintCountry = target
	} else {
		res.HintCountry = egress
	}

	if res.Supported && res.Consistent && len(res.Mismatches) == 0 {
		res.Verdict = Ready
	}
	return res
}
