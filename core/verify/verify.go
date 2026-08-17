// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package verify performs a real egress verification of an active VPN
// connection, mirroring the bash verify_connection_json: it checks the tunnel
// is active, that IPv4 egress goes through the tunnel (route match), that there
// is no IPv6 leak, and that the observed country matches the profile's expected
// country.
//
// The network observations are gathered by an injected Observer so the verdict
// logic is deterministic and unit testable. The verdict/findings model matches
// the bash contract: verdict in {verified, warning, failed}.
package verify

import "context"

// Verdict is the overall verification outcome.
type Verdict string

const (
	Verified Verdict = "verified"
	Warning  Verdict = "warning"
	Failed   Verdict = "failed"
)

// Observation is the raw network state gathered for verification. Empty string
// means "not observed / unavailable".
type Observation struct {
	TunnelActive  bool
	BoundV4       string // public IPv4 seen when bound to the tunnel interface
	DefaultV4     string // public IPv4 seen via the system default route
	BoundV6       string // public IPv6 via the tunnel (empty if none)
	DefaultV6     string // public IPv6 via default route (empty if none)
	ObservedCC    string // observed country code from geo lookup (may be empty)
	ExpectedCC    string // expected country code from the profile (may be empty)
	DNSThroughVPN *bool  // nil = unknown; true/false when determinable
}

// Observer gathers the Observation. In production it wraps core/probe + geo.
type Observer interface {
	Observe(ctx context.Context) (Observation, error)
}

// Result is the structured verification result.
type Result struct {
	Verdict    Verdict  `json:"verdict"`
	MessageKey string   `json:"message_key"`
	Findings   []string `json:"findings"`
	// Echoed observations for the caller/UI.
	Active     bool   `json:"active"`
	RouteMatch bool   `json:"route_match"`
	IPv6Leak   bool   `json:"ipv6_leak"`
	CountryOK  string `json:"country_match"` // "match"/"mismatch"/"unknown"
	EgressIPv4 string `json:"egress_ipv4,omitempty"`
}

// Run evaluates an Observation into a Result. This is pure decision logic; the
// severity precedence matches bash (failed > warning > verified).
func Run(ctx context.Context, obs Observer) (Result, error) {
	o, err := obs.Observe(ctx)
	if err != nil {
		return Result{}, err
	}
	return Evaluate(o), nil
}

// Evaluate is the pure verdict function over an Observation.
func Evaluate(o Observation) Result {
	r := Result{
		Verdict:    Verified,
		MessageKey: "verify.verified",
		CountryOK:  "unknown",
	}

	// Tunnel must be active; otherwise nothing else matters.
	if !o.TunnelActive {
		return Result{
			Verdict:    Failed,
			MessageKey: "verify.failed.inactive",
			Findings:   []string{"verify.tunnel.inactive"},
		}
	}
	r.Active = true

	// IPv4 egress must exist through the tunnel.
	if o.BoundV4 == "" {
		r.Verdict = Failed
		r.MessageKey = "verify.failed.no-egress"
		r.Findings = append(r.Findings, "verify.ipv4.bound-unavailable")
		// A failed egress is terminal for the verdict but we still report leaks.
	} else {
		r.EgressIPv4 = o.BoundV4
		if o.DefaultV4 != "" && o.BoundV4 == o.DefaultV4 {
			r.RouteMatch = true
		} else {
			r.Findings = append(r.Findings, "verify.ipv4.route-mismatch")
			r.escalate(Warning, "verify.warning.route-mismatch")
		}
	}

	// IPv6 leak: default route reaches IPv6 that the tunnel does not (or differs).
	if o.DefaultV6 != "" && (o.BoundV6 == "" || o.BoundV6 != o.DefaultV6) {
		r.IPv6Leak = true
		r.Findings = append(r.Findings, "verify.ipv6.potential-leak")
		r.escalate(Warning, "verify.warning.ipv6-leak")
	}

	// Country match, only when both are known.
	switch {
	case o.ExpectedCC == "" || o.ObservedCC == "":
		r.CountryOK = "unknown"
	case eqFold(o.ExpectedCC, o.ObservedCC):
		r.CountryOK = "match"
	default:
		r.CountryOK = "mismatch"
		r.Findings = append(r.Findings, "verify.geo.country-mismatch")
		r.escalate(Warning, "verify.warning.country-mismatch")
	}

	// DNS through VPN, when determinable.
	if o.DNSThroughVPN != nil && !*o.DNSThroughVPN {
		r.Findings = append(r.Findings, "verify.dns.not-through-vpn")
		r.escalate(Warning, "verify.warning.dns-leak")
	}

	return r
}

// escalate raises the verdict severity but never downgrades a failure.
func (r *Result) escalate(to Verdict, key string) {
	if r.Verdict == Failed {
		return // failed stays failed
	}
	if to == Warning && r.Verdict == Verified {
		r.Verdict = Warning
		r.MessageKey = key
	}
}

func eqFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
