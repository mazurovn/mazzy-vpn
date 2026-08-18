// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mazurovn/mazzy-vpn/core/dnscheck"
)

// resolverInfoForIface reads active resolvers + DoT status for an interface via
// resolvectl. Returns resolvers and whether encryption (DoT) is on.
func resolverInfoForIface(iface string) ([]string, bool) {
	if iface == "" {
		return nil, false
	}
	out, err := exec.Command("resolvectl", "status", iface).Output()
	if err != nil {
		return nil, false
	}
	var resolvers []string
	encrypted := false
	for _, line := range strings.Split(string(out), "\n") {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "Current DNS Server:") || strings.HasPrefix(l, "DNS Servers:") {
			fields := strings.Fields(strings.SplitN(l, ":", 2)[1])
			resolvers = append(resolvers, fields...)
		}
		// resolvectl reports DoT as "DNSOverTLS=opportunistic", "DNSOverTLS=yes",
		// or "+DNSOverTLS" in the Protocols line. A leading "-DNSOverTLS" means off.
		if strings.Contains(l, "DNSOverTLS=opportunistic") ||
			strings.Contains(l, "DNSOverTLS=yes") ||
			strings.Contains(l, "+DNSOverTLS") {
			encrypted = true
		}
	}
	return resolvers, encrypted
}

// dnsCountry probes which country the DNS path resolves from (via a trace that
// reflects the resolver's egress). We reuse the Cloudflare trace loc as a proxy
// for the DNS/egress location.
func dnsCountryProbe(ctx context.Context) string {
	body, code := httpGetText(ctx, "https://1.1.1.1/cdn-cgi/trace", 5*time.Second)
	if code != 200 {
		return ""
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "loc=") {
			return strings.TrimPrefix(line, "loc=")
		}
	}
	return ""
}

// dash returns s or "—" when empty (generic, unlike zone-specific orDash).
func dash(s string) string {
	if s == "" {
		return "—"
	}
	// Values come from resolvectl / remote probes; sanitize before display.
	return safeDisplay(s)
}

// enableDoT turns on DNS-over-TLS for the tunnel interface via resolvectl,
// encrypting DNS queries all the way to the resolver (opportunistic mode keeps
// resolution working if the resolver lacks DoT).
func enableDoT(iface string) error {
	if iface == "" {
		return fmt.Errorf("no active tunnel interface")
	}
	// "opportunistic" avoids breaking resolution when the server has no DoT.
	out, err := exec.Command("resolvectl", "dnsovertls", iface, "opportunistic").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// cmdDNSCheck analyzes DNS privacy for the active connection. With --dot it
// enables DNS-over-TLS on the tunnel interface (needs root).
func cmdDNSCheck(ctx context.Context, args []string) int {
	iface := detectLiveInterface()

	if hasFlag(args, "--dot") {
		if os.Geteuid() != 0 {
			fmt.Fprintln(os.Stderr, translator().T("cli.dnscheck.dot_needs_root"))
			return 1
		}
		if err := enableDoT(iface); err != nil {
			fmt.Fprintln(os.Stderr, "failed to enable DoT:", err)
			return 1
		}
		fmt.Printf("✔ DNS-over-TLS (opportunistic) enabled on %s.\n", iface)
		fmt.Println("  DNS queries are now encrypted to the resolver. Re-check below:")
		fmt.Println()
	}
	resolvers, encrypted := resolverInfoForIface(iface)

	// Egress country from the stealth signal (reuses ip-api).
	sig := gatherStealthSignal(ctx)

	a := dnscheck.Analysis{
		Interface:     iface,
		Resolvers:     resolvers,
		Encrypted:     encrypted,
		EgressCountry: sig.EgressCountry,
		DNSCountry:    dnsCountryProbe(ctx),
	}
	rep := dnscheck.Evaluate(a)

	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(struct {
			Analysis dnscheck.Analysis `json:"analysis"`
			Report   *dnscheck.Report  `json:"report"`
		}{a, rep})
		if !rep.Healthy() {
			return 1
		}
		return 0
	}

	safeResolvers := make([]string, len(a.Resolvers))
	for i, r := range a.Resolvers {
		safeResolvers[i] = safeDisplay(r)
	}
	fmt.Printf("DNS on %s: resolvers=%v encrypted=%v\n", dash(iface), safeResolvers, a.Encrypted)
	fmt.Printf("Egress: %s   DNS location: %s\n\n", dash(a.EgressCountry), dash(a.DNSCountry))
	for _, f := range rep.Findings {
		glyph := "●"
		switch f.Severity {
		case dnscheck.Critical:
			glyph = "✖"
		case dnscheck.Warn:
			glyph = "▲"
		}
		fmt.Printf("%s [%s] %s\n", glyph, f.Level, safeDisplay(f.Title))
		if f.Detail != "" {
			fmt.Printf("    %s\n", safeDisplay(f.Detail))
		}
		if f.Fix != "" {
			fmt.Printf("    fix: %s\n", safeDisplay(f.Fix))
		}
	}
	fmt.Printf("\n→ %s\n", rep.Verdict)
	if !rep.Healthy() {
		return 1
	}
	return 0
}
