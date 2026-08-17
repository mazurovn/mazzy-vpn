// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package stealth checks how "clean" the VPN egress looks to detection systems
// (Google/Gemini/Antigravity, Cloudflare, generic geo-IP). It surfaces the
// signals that reveal a VPN — IPv6/DNS leaks, timezone vs egress-country
// mismatch, datacenter/proxy ASN flags — and gives concrete fixes so services
// believe the session originates from the intended location.
package stealth

import "strings"

// Signal is the observed environment for a stealth analysis.
type Signal struct {
	EgressIPv4       string // public IPv4 through the tunnel
	EgressCountry    string // 2-letter country of the egress IP
	EgressCity       string
	IPv6Leaked       bool // an IPv6 address is reachable (leak)
	IPv6EgressIP     string
	DNSCountry       string // country of the resolver seen by services ("" if unknown)
	SystemTimezone   string // e.g. "Europe/Moscow"
	ExpectedTZCC     string // country implied by the system timezone ("" if unknown)
	IsProxyFlagged   bool   // geo-IP marks the egress as proxy
	IsHostingFlagged bool   // geo-IP marks the egress as hosting/datacenter
	CloudflareColo   string // Cloudflare edge PoP (e.g. "LHR")
	CloudflareLoc    string // Cloudflare-reported country
}

// Severity of a stealth finding.
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

// Finding is one detection risk with an explanation and fix.
type Finding struct {
	Severity Severity `json:"-"`
	Level    string   `json:"level"`
	Vector   string   `json:"vector"` // stable key, e.g. "timezone.mismatch"
	Title    string   `json:"title"`
	Detail   string   `json:"detail"`
	Fix      string   `json:"fix"`
}

// Report is the stealth analysis.
type Report struct {
	Findings []Finding `json:"findings"`
	// Score 0..100: how convincingly the session looks like a real local user.
	Score   int    `json:"score"`
	Verdict string `json:"verdict"`
}

// Analyze evaluates the signals into detection risks + a stealth score.
func Analyze(s Signal) *Report {
	r := &Report{Score: 100}
	add := func(sev Severity, vector, title, detail, fix string, penalty int) {
		r.Findings = append(r.Findings, Finding{
			Severity: sev, Level: sev.String(), Vector: vector,
			Title: title, Detail: detail, Fix: fix,
		})
		r.Score -= penalty
	}

	// --- IPv6 leak: reveals the real ISP/location outside the tunnel ---
	if s.IPv6Leaked {
		add(Critical, "ipv6.leak", "IPv6 leak",
			"An IPv6 address ("+or(s.IPv6EgressIP, "reachable")+") is exposed outside the tunnel; services see your real IPv6 location.",
			"Enable the IPv6 kill-switch (default on). Verify: no IPv6 egress after connect.", 40)
	}

	// --- Timezone vs egress-country mismatch: the strongest browser signal ---
	if s.ExpectedTZCC != "" && s.EgressCountry != "" &&
		!strings.EqualFold(s.ExpectedTZCC, s.EgressCountry) {
		add(Critical, "timezone.mismatch", "Timezone does not match egress country",
			"System timezone ("+s.SystemTimezone+" ≈ "+s.ExpectedTZCC+") disagrees with the egress country ("+s.EgressCountry+"). Browsers expose Intl timezone; Gemini/Google/Antigravity flag this as a VPN.",
			"Set the system (or browser profile) timezone to match the egress country, e.g. a NL exit → Europe/Amsterdam.", 35)
	}

	// --- DNS country mismatch: resolver in a different country than egress ---
	if s.DNSCountry != "" && s.EgressCountry != "" &&
		!strings.EqualFold(s.DNSCountry, s.EgressCountry) {
		add(Warn, "dns.mismatch", "DNS resolver in a different country",
			"Your DNS resolver appears to be in "+s.DNSCountry+" but egress is "+s.EgressCountry+". A DNS/egress mismatch is a known VPN tell.",
			"Use the DNS pushed by the VPN profile (in-country), not a global resolver.", 15)
	}

	// --- Datacenter / proxy ASN: many services block hosting IP ranges ---
	if s.IsProxyFlagged || s.IsHostingFlagged {
		add(Warn, "asn.datacenter", "Egress IP is a datacenter/proxy range",
			"The egress IP is flagged as hosting/proxy (typical for VPN servers). Strict services (Google/Cloudflare) may challenge or block it.",
			"Prefer residential/ISP-grade exit servers if available; rotate to a less-flagged zone (mazzy-vpn test).", 15)
	}

	// --- Cloudflare edge country mismatch (minor) ---
	if s.CloudflareLoc != "" && s.EgressCountry != "" &&
		!strings.EqualFold(s.CloudflareLoc, s.EgressCountry) {
		add(Info, "cloudflare.colo", "Cloudflare sees a different country",
			"Cloudflare reports loc="+s.CloudflareLoc+" (edge "+s.CloudflareColo+") vs egress "+s.EgressCountry+". Usually benign (nearest PoP), but worth noting.",
			"No action needed unless a service geo-blocks; try a zone closer to a matching PoP.", 5)
	}

	if r.Score < 0 {
		r.Score = 0
	}
	switch {
	case r.Score >= 85:
		r.Verdict = "clean — looks like a real local user"
	case r.Score >= 60:
		r.Verdict = "acceptable — minor tells, fix warnings for strict services"
	case r.Score >= 35:
		r.Verdict = "risky — likely detectable as VPN"
	default:
		r.Verdict = "exposed — will be flagged as VPN"
	}
	return r
}

func or(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
