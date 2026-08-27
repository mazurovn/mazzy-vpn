// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"time"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/profile"
)

// profileAudit is the per-profile health verdict for `mazzy-vpn verify`.
type profileAudit struct {
	Name        string   `json:"name"`
	Protocol    string   `json:"protocol"`
	Country     string   `json:"country,omitempty"`
	Endpoint    string   `json:"endpoint,omitempty"`
	Parses      bool     `json:"parses"`
	Valid       bool     `json:"valid"`
	Connectable bool     `json:"connectable"`  // engine can bring it up (WG/AWG)
	EndpointDNS bool     `json:"endpoint_dns"` // endpoint host resolves
	Problems    []string `json:"problems,omitempty"`
	Advice      string   `json:"advice,omitempty"`
}

// auditProfile inspects one catalog entry: parse, validate, connectability, and
// (optionally) whether its endpoint host resolves in DNS. It never connects.
func auditProfile(ctx context.Context, e catalogEntry, resolveDNS bool) profileAudit {
	a := profileAudit{Name: e.Name, Protocol: string(e.Protocol), Country: e.Country}

	// OpenVPN: catalogued but not connectable by the embedded engine yet.
	if e.Protocol == core.OpenVPN {
		a.Parses = true // we do not deeply parse OVPN here
		a.Advice = "OpenVPN is not connectable by the embedded engine yet; import a WireGuard/AmneziaWG variant"
		return a
	}

	data, err := os.ReadFile(e.File)
	if err != nil {
		a.Problems = []string{"cannot read file: " + err.Error()}
		return a
	}
	cfg, err := profile.Parse(string(data))
	if err != nil {
		a.Problems = []string{"parse error: " + err.Error()}
		return a
	}
	a.Parses = true
	proto := core.WireGuard
	if cfg.HasAmneziaFields {
		proto = core.AmneziaWG
	}
	a.Protocol = string(proto)
	a.Endpoint = cfg.Endpoint()

	problems := profile.Validate(proto, cfg)
	a.Valid = len(problems) == 0
	a.Problems = problems
	a.Connectable = a.Valid // WG/AWG valid profiles are connectable by the engine

	// Optional DNS resolvability of the endpoint host — a cheap, high-signal
	// check that catches dead/renamed servers without a full probe.
	if resolveDNS && a.Endpoint != "" {
		host := cfg.EndpointHost()
		if host != "" {
			if net.ParseIP(host) != nil {
				a.EndpointDNS = true // IP literal always "resolves"
			} else {
				cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
				ips, derr := net.DefaultResolver.LookupHost(cctx, host)
				cancel()
				a.EndpointDNS = derr == nil && len(ips) > 0
				if !a.EndpointDNS {
					a.Problems = append(a.Problems, "endpoint host does not resolve: "+host)
				}
			}
		}
	}
	if a.Advice == "" && !a.Valid {
		a.Advice = "fix the problems above or re-export the profile from your provider"
	}
	return a
}

// catalogEntry is the minimal shape auditProfile needs (decouples from the
// catalog package's concrete Entry for testability).
type catalogEntry struct {
	Name     string
	File     string
	Protocol core.Protocol
	Country  string
}

// cmdVerify audits every managed profile and reports health: parses, valid,
// connectable, and (with default DNS on) whether the endpoint resolves. This is
// the "diagnostics for configs" surface. Use --json for machine output and
// --no-dns to skip the network resolution pass (pure offline validation).
func cmdVerify(ctx context.Context, args []string) int {
	cat := newCatalog()
	entries, err := cat.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, "catalog error:", err)
		return 1
	}
	if len(entries) == 0 {
		fmt.Println(translator().T("cli.catalog.none"))
		return 1
	}
	resolveDNS := !hasFlag(args, "--no-dns")

	audits := make([]profileAudit, len(entries))
	for i, e := range entries {
		audits[i] = auditProfile(ctx, catalogEntry{
			Name: e.Name, File: e.File, Protocol: e.Protocol, Country: e.Country,
		}, resolveDNS)
	}
	sort.SliceStable(audits, func(i, j int) bool {
		// Broken first (so problems surface), then by name.
		hi, hj := audits[i].healthy(), audits[j].healthy()
		if hi != hj {
			return !hi
		}
		return audits[i].Name < audits[j].Name
	})

	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(audits)
		return exitForAudits(audits)
	}

	fmt.Printf("%-24s %-10s %-4s %-8s %s\n", "NAME", "PROTOCOL", "CC", "ENDPOINT", "VERDICT")
	okCount, warnCount, failCount := 0, 0, 0
	for _, a := range audits {
		verdict, glyph := a.verdict()
		switch glyph {
		case "✔":
			okCount++
		case "▲":
			warnCount++
		default:
			failCount++
		}
		dns := "-"
		if a.Endpoint != "" {
			if a.EndpointDNS {
				dns = "dns✔"
			} else if resolveDNS {
				dns = "dns✖"
			} else {
				dns = "?"
			}
		}
		fmt.Printf("%s %-22s %-10s %-4s %-8s %s\n",
			glyph, safeDisplay(a.Name), safeDisplay(a.Protocol), safeDisplay(a.Country), dns, verdict)
		for _, p := range a.Problems {
			fmt.Printf("    - %s\n", safeDisplay(p))
		}
		if a.Advice != "" && !a.healthy() {
			fmt.Printf("    → %s\n", safeDisplay(a.Advice))
		}
	}
	fmt.Printf("\nSummary: OK=%d WARN=%d FAIL=%d (of %d profiles)\n", okCount, warnCount, failCount, len(audits))
	return exitForAudits(audits)
}

// healthy reports whether a profile is fully usable (connectable + valid + DNS).
func (a profileAudit) healthy() bool {
	if a.Protocol == string(core.OpenVPN) {
		return false // catalogued but not connectable
	}
	return a.Valid && a.Connectable && (a.Endpoint == "" || a.EndpointDNS)
}

// verdict returns a human label + glyph for a profile audit.
func (a profileAudit) verdict() (label, glyph string) {
	switch {
	case a.Protocol == string(core.OpenVPN):
		return "catalogued (OpenVPN, not connectable yet)", "▲"
	case !a.Parses:
		return "broken (cannot parse)", "✖"
	case !a.Valid:
		return "invalid profile", "✖"
	case a.Endpoint != "" && !a.EndpointDNS:
		return "endpoint does not resolve", "▲"
	default:
		return "healthy", "✔"
	}
}

// exitForAudits returns 1 if any profile is broken/invalid (FAIL), else 0. A
// WARN (OpenVPN/no-DNS) does not fail the command.
func exitForAudits(audits []profileAudit) int {
	for _, a := range audits {
		if _, glyph := a.verdict(); glyph == "✖" {
			return 1
		}
	}
	return 0
}
