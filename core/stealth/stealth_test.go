// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package stealth

import (
	"strings"
	"testing"
)

func cleanSignal() Signal {
	return Signal{
		EgressIPv4: "203.0.113.9", EgressCountry: "NL", EgressCity: "Amsterdam",
		IPv6Leaked: false, DNSCountry: "NL",
		SystemTimezone: "Europe/Amsterdam", ExpectedTZCC: "NL",
		IsProxyFlagged: false, IsHostingFlagged: false,
		CloudflareLoc: "NL",
	}
}

func TestCleanScoresHigh(t *testing.T) {
	r := Analyze(cleanSignal())
	if r.Score < 85 {
		t.Fatalf("clean signal should score high, got %d (%v)", r.Score, r.Findings)
	}
	if !strings.Contains(r.Verdict, "clean") {
		t.Errorf("verdict = %q", r.Verdict)
	}
}

func TestTimezoneMismatchIsCritical(t *testing.T) {
	s := cleanSignal()
	s.SystemTimezone = "Europe/Moscow"
	s.ExpectedTZCC = "RU"
	r := Analyze(s)
	found := false
	for _, f := range r.Findings {
		if f.Vector == "timezone.mismatch" && f.Severity == Critical {
			found = true
		}
	}
	if !found {
		t.Fatal("timezone/egress mismatch must be critical")
	}
	if r.Score >= 85 {
		t.Errorf("mismatch should lower score, got %d", r.Score)
	}
}

func TestIPv6LeakIsCritical(t *testing.T) {
	s := cleanSignal()
	s.IPv6Leaked = true
	s.IPv6EgressIP = "2001:db8::1"
	r := Analyze(s)
	if r.Score >= 70 {
		t.Errorf("ipv6 leak should heavily penalize, got %d", r.Score)
	}
	if r.Findings[0].Vector != "ipv6.leak" {
		t.Errorf("ipv6 leak should be first finding, got %s", r.Findings[0].Vector)
	}
}

func TestDatacenterFlagged(t *testing.T) {
	s := cleanSignal()
	s.IsProxyFlagged = true
	s.IsHostingFlagged = true
	r := Analyze(s)
	found := false
	for _, f := range r.Findings {
		if f.Vector == "asn.datacenter" {
			found = true
		}
	}
	if !found {
		t.Error("datacenter/proxy flag should be reported")
	}
}

func TestDNSMismatch(t *testing.T) {
	s := cleanSignal()
	s.DNSCountry = "US"
	r := Analyze(s)
	found := false
	for _, f := range r.Findings {
		if f.Vector == "dns.mismatch" {
			found = true
		}
	}
	if !found {
		t.Error("DNS/egress country mismatch should be reported")
	}
}

func TestExposedVerdict(t *testing.T) {
	// Everything wrong.
	s := Signal{
		EgressCountry: "NL", IPv6Leaked: true, IPv6EgressIP: "2001:db8::1",
		SystemTimezone: "Europe/Moscow", ExpectedTZCC: "RU",
		DNSCountry: "US", IsProxyFlagged: true, IsHostingFlagged: true,
	}
	r := Analyze(s)
	if r.Score > 35 {
		t.Errorf("fully exposed should score low, got %d", r.Score)
	}
	if !strings.Contains(r.Verdict, "exposed") && !strings.Contains(r.Verdict, "risky") {
		t.Errorf("verdict should be exposed/risky, got %q", r.Verdict)
	}
}
