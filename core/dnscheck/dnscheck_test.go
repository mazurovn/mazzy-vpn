// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package dnscheck

import (
	"strings"
	"testing"
)

func TestNoResolverIsCritical(t *testing.T) {
	r := Evaluate(Analysis{Interface: "", Resolvers: nil, EgressCountry: "NL"})
	if r.Healthy() {
		t.Fatal("no resolver on tunnel must be critical (leak)")
	}
	if !strings.Contains(r.Verdict, "leaking") {
		t.Errorf("verdict = %q", r.Verdict)
	}
}

func TestInCountryEncryptedIsClean(t *testing.T) {
	r := Evaluate(Analysis{
		Interface: "vpnaw0", Resolvers: []string{"1.1.1.1"},
		EgressCountry: "NL", DNSCountry: "NL", Encrypted: true,
	})
	if !r.Healthy() {
		t.Fatalf("in-country encrypted should be healthy: %+v", r.Findings)
	}
	if !strings.Contains(r.Verdict, "clean") {
		t.Errorf("verdict = %q", r.Verdict)
	}
}

func TestUnencryptedWarns(t *testing.T) {
	r := Evaluate(Analysis{
		Interface: "vpnaw0", Resolvers: []string{"1.1.1.1"},
		EgressCountry: "NL", DNSCountry: "NL", Encrypted: false,
	})
	found := false
	for _, f := range r.Findings {
		if strings.Contains(f.Title, "not encrypted") {
			found = true
		}
	}
	if !found {
		t.Error("unencrypted DNS should warn")
	}
	if !r.Healthy() {
		t.Error("unencrypted is a warning, not critical")
	}
}

func TestCountryMismatchWarns(t *testing.T) {
	r := Evaluate(Analysis{
		Interface: "vpnaw0", Resolvers: []string{"8.8.8.8"},
		EgressCountry: "NL", DNSCountry: "US", Encrypted: true,
	})
	found := false
	for _, f := range r.Findings {
		if strings.Contains(f.Title, "different country") {
			found = true
		}
	}
	if !found {
		t.Error("DNS/egress country mismatch should warn")
	}
}
