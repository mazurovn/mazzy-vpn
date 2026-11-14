// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package checks reads `mazzy-vpn egress --json` into traffic lights.
// A missing field stays unknown. No latency is stored unless the CLI
// reported one.
package checks

import (
	"encoding/json"

	"github.com/mazurovn/mazzy-vpn/desktop/monitor"
)

// Doc is the CLI egress document.
type Doc struct {
	IPv4            string  `json:"ipv4,omitempty"`
	Country         string  `json:"country,omitempty"`
	City            string  `json:"city,omitempty"`
	Proxy           *bool   `json:"proxy,omitempty"`
	Hosting         *bool   `json:"hosting,omitempty"`
	IPv6            string  `json:"ipv6,omitempty"`
	Timezone        string  `json:"timezone,omitempty"`
	TimezoneCountry string  `json:"timezone_country,omitempty"`
	DNSBound        *bool   `json:"dns_bound,omitempty"`
	DNSCountry      string  `json:"dns_country,omitempty"`
	VirtualHop      bool    `json:"virtual_hop,omitempty"`
	Antigravity     Service `json:"antigravity"`
	Unknown         []Gap   `json:"unknown,omitempty"`
}

// Service is one provider probe.
type Service struct {
	Service   string `json:"service"`
	State     string `json:"state"`
	HTTP      int    `json:"http_status,omitempty"`
	Reason    string `json:"reason,omitempty"`
	LatencyMS *int64 `json:"latency_ms,omitempty"`
}

// Gap is a check that did not produce a value.
type Gap struct {
	Check  string `json:"check"`
	Reason string `json:"reason"`
}

// Parse decodes a CLI document.
func Parse(raw string) (Doc, error) {
	var doc Doc
	err := json.Unmarshal([]byte(raw), &doc)
	return doc, err
}

// Apply copies a document onto a view. Handshake and path are left as they
// were. A probe latency is appended only when the CLI sent one.
func Apply(v monitor.View, doc Doc) monitor.View {
	if doc.IPv4 == "" {
		v.EgressLight = monitor.Unknown
		v.EgressText = reason(doc, "ipv4", "address was not observed")
	} else {
		v.Egress = doc.IPv4
		v.EgressLight = monitor.Green
		v.EgressText = doc.IPv4
		if doc.Country != "" {
			v.EgressText += " " + doc.Country
		}
	}

	v.Clean = monitor.Green
	v.CleanText = "not flagged"
	switch {
	case doc.IPv4 == "" || doc.Proxy == nil || doc.Hosting == nil:
		v.Clean = monitor.Unknown
		v.CleanText = reason(doc, "ipv4", "cleanliness was not observed")
	case *doc.Proxy || *doc.Hosting:
		v.Clean = monitor.Yellow
		v.CleanText = "flagged proxy or hosting"
	case doc.DNSBound != nil && !*doc.DNSBound:
		v.Clean = monitor.Red
		v.CleanText = "DNS is not bound to the tunnel"
	case doc.IPv6 != "":
		v.Clean = monitor.Yellow
		v.CleanText = "IPv6 " + doc.IPv6 + " was observed"
	case doc.TimezoneCountry != "" && doc.Country != "" && !eq(doc.TimezoneCountry, doc.Country):
		v.Clean = monitor.Yellow
		v.CleanText = "timezone " + doc.TimezoneCountry + " != " + doc.Country
	}
	if gap := first(doc, "ipv6", "dns", "timezone", "dns-country"); v.Clean == monitor.Green && gap != "" {
		v.Clean = monitor.Unknown
		v.CleanText = gap
	}

	switch doc.Antigravity.State {
	case "eligible":
		// HEAD 401/403/404 is reachability. A hosting or proxy tunnel
		// egress is the address Google scores unless a virtual hop is set.
		if egressFlagged(doc) && !doc.VirtualHop {
			v.Service = monitor.Yellow
			v.ServiceText = "virtual hop unset on a hosting or proxy egress"
		} else {
			v.Service = monitor.Green
			v.ServiceText = "Antigravity auth boundary"
		}
	case "rate-limited":
		v.Service = monitor.Yellow
		v.ServiceText = "Antigravity rate-limited"
	case "unavailable":
		v.Service = monitor.Red
		v.ServiceText = "Antigravity unavailable"
	case "unreachable":
		v.Service = monitor.Red
		v.ServiceText = "Antigravity " + doc.Antigravity.Reason
	default:
		v.Service = monitor.Unknown
		v.ServiceText = "Antigravity " + doc.Antigravity.Reason
		if doc.Antigravity.Reason == "" {
			v.ServiceText = "Antigravity was not checked"
		}
	}
	if doc.Antigravity.LatencyMS != nil && *doc.Antigravity.LatencyMS >= 0 && doc.Antigravity.HTTP != 0 {
		v.Points = append(v.Points, monitor.Point{MS: *doc.Antigravity.LatencyMS, HasMS: true})
	}
	return v
}

func egressFlagged(doc Doc) bool {
	if doc.Proxy != nil && *doc.Proxy {
		return true
	}
	return doc.Hosting != nil && *doc.Hosting
}

func reason(doc Doc, check, fallback string) string {
	for _, g := range doc.Unknown {
		if g.Check == check && g.Reason != "" {
			return g.Reason
		}
	}
	return fallback
}

func first(doc Doc, checks ...string) string {
	for _, c := range checks {
		if r := reason(doc, c, ""); r != "" {
			return c + ": " + r
		}
	}
	return ""
}

func eq(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'a' && ca <= 'z' {
			ca -= 32
		}
		if cb >= 'a' && cb <= 'z' {
			cb -= 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
