// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mazurovn/mazzy-vpn/core/mimicry"
)

// antigravityProbeURL is the credential-free HEAD target from the provider
// registry. 401, 403 and 404 mean the auth boundary was reached.
const antigravityProbeURL = "https://daily-cloudcode-pa.googleapis.com/"

// classifyService applies the Antigravity/Google auth-boundary rules.
// A transport error is unreachable. Any other status is indeterminate.
// No number is returned; the caller records latency only after a response.
func classifyService(status int, transportErr bool) (state, reason string) {
	if transportErr {
		return "unreachable", "no HTTP response"
	}
	switch status {
	case 401, 403, 404:
		return "eligible", "auth boundary reached"
	case 429:
		return "rate-limited", "HTTP 429"
	}
	if status >= 500 && status <= 599 {
		return "unavailable", fmt.Sprintf("HTTP %d", status)
	}
	return "indeterminate", fmt.Sprintf("unrecognized HTTP %d", status)
}

type gap struct {
	Check  string `json:"check"`
	Reason string `json:"reason"`
}

type serviceCheck struct {
	Service   string `json:"service"`
	State     string `json:"state"`
	HTTP      int    `json:"http_status,omitempty"`
	Reason    string `json:"reason,omitempty"`
	LatencyMS *int64 `json:"latency_ms,omitempty"`
}

// egressDoc is what `mazzy-vpn egress --json` prints. Empty fields were not
// observed. Unknown explains each check that did not run.
type egressDoc struct {
	IPv4            string       `json:"ipv4,omitempty"`
	Country         string       `json:"country,omitempty"`
	City            string       `json:"city,omitempty"`
	Proxy           *bool        `json:"proxy,omitempty"`
	Hosting         *bool        `json:"hosting,omitempty"`
	IPv6            string       `json:"ipv6,omitempty"`
	Timezone        string       `json:"timezone,omitempty"`
	TimezoneCountry string       `json:"timezone_country,omitempty"`
	DNSBound        *bool        `json:"dns_bound,omitempty"`
	DNSCountry      string       `json:"dns_country,omitempty"`
	VirtualHop      bool         `json:"virtual_hop,omitempty"`
	Antigravity     serviceCheck `json:"antigravity"`
	Unknown         []gap        `json:"unknown,omitempty"`
}

func cmdEgress(ctx context.Context, args []string) int {
	doc := collectEgress(ctx)
	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(doc)
		return 0
	}
	if doc.IPv4 != "" {
		fmt.Printf("IPv4: %s", safeDisplay(doc.IPv4))
		if doc.Country != "" {
			fmt.Printf(" (%s", safeDisplay(doc.Country))
			if doc.City != "" {
				fmt.Printf(", %s", safeDisplay(doc.City))
			}
			fmt.Print(")")
		}
		fmt.Println()
	}
	for _, g := range doc.Unknown {
		// safeDisplay drops CR/LF so a timezone or probe string cannot start a second log line.
		fmt.Printf("unknown  %s  %s\n", safeDisplay(g.Check), safeDisplay(g.Reason))
	}
	fmt.Printf("Antigravity: %s", safeDisplay(doc.Antigravity.State))
	if doc.Antigravity.Reason != "" {
		fmt.Printf(" (%s)", safeDisplay(doc.Antigravity.Reason))
	}
	fmt.Println()
	if doc.VirtualHop {
		fmt.Println("virtual hop: configured")
	} else {
		fmt.Println("virtual hop: unset")
	}
	return 0
}

func collectEgress(ctx context.Context) egressDoc {
	doc := egressDoc{VirtualHop: virtualHopConfigured()}
	sig := gatherStealthSignal(ctx)
	if sig.EgressIPv4 == "" {
		doc.Unknown = append(doc.Unknown, gap{Check: "ipv4", Reason: "ip-api did not return an address"})
	} else {
		doc.IPv4 = sig.EgressIPv4
		doc.Country = sig.EgressCountry
		doc.City = sig.EgressCity
		proxy, hosting := sig.IsProxyFlagged, sig.IsHostingFlagged
		doc.Proxy = &proxy
		doc.Hosting = &hosting
	}

	if ip, ok := probeIPv6(ctx); !ok {
		doc.Unknown = append(doc.Unknown, gap{Check: "ipv6", Reason: "api6.ipify.org did not answer"})
	} else if ip != "" {
		doc.IPv6 = ip
	}

	tz := safeDisplay(systemTimezone())
	if tz == "" {
		doc.Unknown = append(doc.Unknown, gap{Check: "timezone", Reason: "system timezone was not read"})
	} else {
		doc.Timezone = tz
		doc.TimezoneCountry = mimicry.CountryForTimezone(tz)
		if doc.TimezoneCountry == "" {
			doc.Unknown = append(doc.Unknown, gap{Check: "timezone", Reason: "no country for " + tz})
		}
	}

	iface := detectLiveInterface()
	if iface == "" {
		doc.Unknown = append(doc.Unknown, gap{Check: "dns", Reason: "no tunnel interface"})
	} else if bound, ok := dnsBound(iface); !ok {
		doc.Unknown = append(doc.Unknown, gap{Check: "dns", Reason: "resolvectl did not answer"})
	} else {
		doc.DNSBound = &bound
	}
	if cc := dnsCountryProbe(ctx); cc != "" {
		doc.DNSCountry = cc
	} else {
		doc.Unknown = append(doc.Unknown, gap{Check: "dns-country", Reason: "resolver country was not observed"})
	}

	doc.Antigravity = probeAntigravity(ctx)
	if doc.Antigravity.State == "unreachable" || doc.Antigravity.State == "indeterminate" {
		doc.Unknown = append(doc.Unknown, gap{Check: "antigravity", Reason: doc.Antigravity.Reason})
	}
	return doc
}

func probeIPv6(ctx context.Context) (string, bool) {
	ip, code := httpGetText(ctx, "https://api6.ipify.org", 5*time.Second)
	if code != 200 {
		return "", false
	}
	ip = strings.TrimSpace(ip)
	if !strings.Contains(ip, ":") {
		return "", false
	}
	return ip, true
}

func dnsBound(iface string) (bool, bool) {
	out, err := exec.Command("resolvectl", "status", iface).Output()
	if err != nil {
		return false, false
	}
	resolvers, _ := resolverInfoForIface(iface)
	if len(resolvers) == 0 && !strings.Contains(string(out), "DNS") {
		return false, true
	}
	return len(resolvers) > 0, true
}

func probeAntigravity(ctx context.Context) serviceCheck {
	cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	client := &http.Client{
		Timeout:   8 * time.Second,
		Transport: &http.Transport{Proxy: nil},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(cctx, http.MethodHead, antigravityProbeURL, nil)
	if err != nil {
		state, reason := classifyService(0, true)
		return serviceCheck{Service: "antigravity", State: state, Reason: reason}
	}
	req.Header.Set("User-Agent", "mazzy-vpn/egress")
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		state, reason := classifyService(0, true)
		return serviceCheck{Service: "antigravity", State: state, Reason: reason}
	}
	defer resp.Body.Close()
	ms := time.Since(start).Milliseconds()
	state, reason := classifyService(resp.StatusCode, false)
	return serviceCheck{
		Service: "antigravity", State: state, HTTP: resp.StatusCode,
		Reason: reason, LatencyMS: &ms,
	}
}
