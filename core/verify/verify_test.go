// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package verify

import "testing"

func boolPtr(b bool) *bool { return &b }

func fullyGood() Observation {
	return Observation{
		TunnelActive:  true,
		BoundV4:       "203.0.113.9",
		DefaultV4:     "203.0.113.9",
		ExpectedCC:    "DE",
		ObservedCC:    "de",
		DNSThroughVPN: boolPtr(true),
	}
}

func TestVerifiedWhenAllGood(t *testing.T) {
	r := Evaluate(fullyGood())
	if r.Verdict != Verified {
		t.Fatalf("verdict = %s, want verified; findings=%v", r.Verdict, r.Findings)
	}
	if !r.RouteMatch || r.CountryOK != "match" {
		t.Errorf("route/country not confirmed: %+v", r)
	}
	if len(r.Findings) != 0 {
		t.Errorf("no findings expected, got %v", r.Findings)
	}
}

func TestInactiveTunnelIsFailed(t *testing.T) {
	r := Evaluate(Observation{TunnelActive: false})
	if r.Verdict != Failed || r.MessageKey != "verify.failed.inactive" {
		t.Fatalf("expected failed/inactive, got %+v", r)
	}
}

func TestNoEgressIsFailed(t *testing.T) {
	o := fullyGood()
	o.BoundV4 = ""
	r := Evaluate(o)
	if r.Verdict != Failed || r.MessageKey != "verify.failed.no-egress" {
		t.Fatalf("expected failed/no-egress, got %+v", r)
	}
}

func TestRouteMismatchIsWarning(t *testing.T) {
	o := fullyGood()
	o.DefaultV4 = "198.51.100.1" // differs from bound → not through tunnel
	r := Evaluate(o)
	if r.Verdict != Warning || r.RouteMatch {
		t.Fatalf("expected warning route-mismatch, got %+v", r)
	}
	if !hasFinding(r, "verify.ipv4.route-mismatch") {
		t.Errorf("missing route-mismatch finding: %v", r.Findings)
	}
}

func TestIPv6LeakIsWarning(t *testing.T) {
	o := fullyGood()
	o.DefaultV6 = "2001:db8::1" // default reaches v6, tunnel does not
	o.BoundV6 = ""
	r := Evaluate(o)
	if r.Verdict != Warning || !r.IPv6Leak {
		t.Fatalf("expected ipv6 leak warning, got %+v", r)
	}
}

func TestCountryMismatchIsWarning(t *testing.T) {
	o := fullyGood()
	o.ObservedCC = "US"
	r := Evaluate(o)
	if r.Verdict != Warning || r.CountryOK != "mismatch" {
		t.Fatalf("expected country mismatch warning, got %+v", r)
	}
}

func TestUnknownCountryDoesNotWarn(t *testing.T) {
	o := fullyGood()
	o.ObservedCC = "" // geo unavailable
	r := Evaluate(o)
	if r.Verdict != Verified || r.CountryOK != "unknown" {
		t.Fatalf("unknown country must not warn, got %+v", r)
	}
}

// TestFailedBeatsWarning locks the precedence: a no-egress failure combined
// with an IPv6 leak stays FAILED, never downgraded to warning.
func TestFailedBeatsWarning(t *testing.T) {
	o := fullyGood()
	o.BoundV4 = ""              // failure
	o.DefaultV6 = "2001:db8::1" // would-be warning
	o.BoundV6 = ""
	r := Evaluate(o)
	if r.Verdict != Failed {
		t.Fatalf("failed must not be downgraded by a warning; got %s", r.Verdict)
	}
	// The leak is still reported as a finding.
	if !r.IPv6Leak {
		t.Error("ipv6 leak should still be reported alongside failure")
	}
}

func TestDNSLeakIsWarning(t *testing.T) {
	o := fullyGood()
	o.DNSThroughVPN = boolPtr(false)
	r := Evaluate(o)
	if r.Verdict != Warning || !hasFinding(r, "verify.dns.not-through-vpn") {
		t.Fatalf("expected dns leak warning, got %+v", r)
	}
}

func hasFinding(r Result, f string) bool {
	for _, x := range r.Findings {
		if x == f {
			return true
		}
	}
	return false
}
