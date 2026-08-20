// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package netdiag analyzes the host network situation before a VPN connection
// and recommends concrete fixes. It detects: no physical uplink, no default
// route, a conflicting VPN already active (e.g. AdGuard on Wi‑Fi), and offers
// an adapter recommendation.
package netdiag

import (
	"github.com/mazurovn/mazzy-vpn/core/netadapter"
)

// Severity of a finding.
type Severity int

const (
	Info Severity = iota
	Warn
	Fail
)

func (s Severity) String() string {
	switch s {
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Fail:
		return "FAIL"
	default:
		return "?"
	}
}

// Finding is one diagnostic observation with an optional fix suggestion.
type Finding struct {
	Severity Severity `json:"-"`
	Level    string   `json:"level"`
	Title    string   `json:"title"`
	Detail   string   `json:"detail,omitempty"`
	Fix      string   `json:"fix,omitempty"`
}

// Report is the network analysis result.
type Report struct {
	Findings          []Finding `json:"findings"`
	RecommendedUplink string    `json:"recommended_uplink,omitempty"`
	ConflictingVPN    string    `json:"conflicting_vpn,omitempty"`
	OK                int       `json:"ok"`
	Warn              int       `json:"warn"`
	Fail              int       `json:"fail"`
}

// Healthy reports whether there are no FAIL findings.
func (r *Report) Healthy() bool { return r.Fail == 0 }

// Analyze inspects the adapters and produces findings + fixes.
func Analyze(adapters []netadapter.Adapter) *Report {
	r := &Report{}
	add := func(sev Severity, title, detail, fix string) {
		switch sev {
		case Info:
			r.OK++
		case Warn:
			r.Warn++
		case Fail:
			r.Fail++
		}
		r.Findings = append(r.Findings, Finding{
			Severity: sev, Level: sev.String(),
			Title: title, Detail: detail, Fix: fix,
		})
	}

	uplinks := netadapter.PhysicalUplinks(adapters)
	if len(uplinks) == 0 {
		add(Fail, "No physical uplink up",
			"No wired/wireless interface with a carrier is up.",
			"Plug in a cable or enable Wi‑Fi, then re-run.")
	} else {
		wired, wifi := 0, 0
		for _, u := range uplinks {
			if u.Wireless {
				wifi++
			} else {
				wired++
			}
		}
		add(Info, "Physical uplink available",
			formatUplinks(uplinks), "")
		if wired > 0 && wifi > 0 {
			add(Info, "Both wired and Wi‑Fi are up",
				"You can choose which uplink to run the VPN over.",
				"Prefer the wired uplink for the VPN test (see recommendation).")
		}
	}

	// Detect a conflicting VPN interface already up (AmneziaWG/WireGuard/other).
	for _, a := range adapters {
		if a.Virtual && a.Up && isVPNInterface(a.Name) {
			r.ConflictingVPN = a.Name
			add(Warn, "A VPN interface is already active",
				"Interface "+a.Name+" is up; another VPN (e.g. AdGuard/WireGuard) may capture traffic.",
				"Disconnect the other VPN before testing, or bind the test to a specific uplink.")
			break
		}
	}

	// Adapter recommendation.
	if rec, reason, ok := netadapter.Recommend(adapters); ok {
		r.RecommendedUplink = rec.Name
		add(Info, "Recommended uplink: "+rec.Name, reason, "")
	}

	return r
}

// isVPNInterface reports whether a name looks like a VPN tunnel.
func isVPNInterface(name string) bool {
	for _, p := range []string{"vpn", "tun", "wg", "awg", "tailscale", "zt", "ppp"} {
		if len(name) >= len(p) && name[:len(p)] == p {
			return true
		}
	}
	return false
}

// formatUplinks renders a compact list of uplink names with their kind.
func formatUplinks(uplinks []netadapter.Adapter) string {
	out := ""
	for i, u := range uplinks {
		if i > 0 {
			out += ", "
		}
		out += u.Name + " (" + u.Kind() + ")"
	}
	return out
}
