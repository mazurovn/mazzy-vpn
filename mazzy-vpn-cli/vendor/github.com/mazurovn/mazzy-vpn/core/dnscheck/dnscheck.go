// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package dnscheck analyzes DNS privacy for an active VPN: whether DNS resolves
// through the tunnel (from the egress country rather than the real one),
// whether it is encrypted (DoT/DoH), and whether it leaks to a system resolver.
// It answers "is my DNS clean, or does it reveal my real location?"
package dnscheck

import "strings"

// Analysis is the DNS privacy result.
type Analysis struct {
	EgressCountry string   `json:"egress_country"` // country of the VPN egress
	DNSCountry    string   `json:"dns_country"`    // country the resolver appears in
	Encrypted     bool     `json:"encrypted"`      // DoT/DoH in use
	Interface     string   `json:"interface"`      // tunnel interface DNS is bound to
	Resolvers     []string `json:"resolvers"`      // active resolvers for the tunnel
}

// Severity of a DNS finding.
type Severity int

const (
	Info Severity = iota
	Warn
	Critical
)

func (s Severity) String() string {
	switch s {
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Critical:
		return "CRITICAL"
	default:
		return "?"
	}
}

// Finding is one DNS privacy issue with a fix.
type Finding struct {
	Severity Severity `json:"-"`
	Level    string   `json:"level"`
	Title    string   `json:"title"`
	Detail   string   `json:"detail"`
	Fix      string   `json:"fix"`
}

// Report is the DNS privacy report.
type Report struct {
	Findings []Finding `json:"findings"`
	Verdict  string    `json:"verdict"`
}

// Healthy reports whether nothing critical was found.
func (r *Report) Healthy() bool {
	for _, f := range r.Findings {
		if f.Severity == Critical {
			return false
		}
	}
	return true
}

// Evaluate turns an Analysis into ranked findings + verdict.
func Evaluate(a Analysis) *Report {
	r := &Report{}
	add := func(sev Severity, title, detail, fix string) {
		r.Findings = append(r.Findings, Finding{Severity: sev, Level: sev.String(), Title: title, Detail: detail, Fix: fix})
	}

	// No DNS bound to the tunnel → likely using the system resolver (leak).
	if a.Interface == "" || len(a.Resolvers) == 0 {
		add(Critical, "DNS not bound to the tunnel",
			"No resolver is set on the VPN interface; queries may use the system/ISP resolver and reveal your real location.",
			"Reconnect so the profile's DNS is applied, or set: resolvectl dns <iface> <dns>.")
		r.Verdict = "leaking — DNS not through the tunnel"
		return r
	}

	// DNS country mismatch with egress → resolver in a different country.
	if a.DNSCountry != "" && a.EgressCountry != "" &&
		!strings.EqualFold(a.DNSCountry, a.EgressCountry) {
		add(Warn, "DNS resolves from a different country",
			"DNS appears to resolve from "+a.DNSCountry+" but egress is "+a.EgressCountry+"; a mismatch is a VPN tell.",
			"Use the in-country DNS pushed by the profile, not a global resolver.")
	}

	// Not encrypted → the VPN server (and its ISP) can see plain DNS.
	if !a.Encrypted {
		add(Warn, "DNS is not encrypted (no DoT/DoH)",
			"DNS travels through the tunnel but is plaintext to the resolver; the VPN server's network can observe queries.",
			"Enable DoT if supported: resolvectl (systemd) with a #hostname resolver, or use a DoH resolver.")
	} else {
		add(Info, "DNS is encrypted", "DNS uses DoT/DoH — queries are hidden even from the VPN server's network.", "")
	}

	// Everything good.
	if a.DNSCountry != "" && strings.EqualFold(a.DNSCountry, a.EgressCountry) {
		add(Info, "DNS resolves from the egress country",
			"DNS ("+a.DNSCountry+") matches egress ("+a.EgressCountry+") — consistent location.", "")
	}

	switch {
	case r.Healthy() && a.Encrypted:
		r.Verdict = "clean — DNS is in-country and encrypted"
	case r.Healthy():
		r.Verdict = "ok — DNS through the tunnel; consider DoT for full privacy"
	default:
		r.Verdict = "issues — see fixes above"
	}
	return r
}
