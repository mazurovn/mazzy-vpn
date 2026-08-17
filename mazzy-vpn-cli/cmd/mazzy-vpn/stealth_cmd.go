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
	"github.com/mazurovn/mazzy-vpn/core/stealth"
)

// systemTimezone reads the active system timezone.
func systemTimezone() string {
	if out, err := exec.Command("timedatectl", "show", "-p", "Timezone", "--value").Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	if b, err := os.ReadFile("/etc/timezone"); err == nil {
		return strings.TrimSpace(string(b))
	}
	return ""
}

// httpGetText fetches a URL and returns trimmed body text (bounded).
func httpGetText(ctx context.Context, url string, timeout time.Duration) (string, int) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		return "", 0
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 mazzy-vpn/stealth")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0
	}
	defer resp.Body.Close()
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	return strings.TrimSpace(string(buf[:n])), resp.StatusCode
}

// gatherStealthSignal probes detection vectors from the current egress.
func gatherStealthSignal(ctx context.Context) stealth.Signal {
	s := stealth.Signal{}

	// IPv4 egress + geo (ip-api gives proxy/hosting flags).
	if body, code := httpGetText(ctx, "http://ip-api.com/json/?fields=query,countryCode,city,proxy,hosting", 6*time.Second); code == 200 {
		var g struct {
			Query       string `json:"query"`
			CountryCode string `json:"countryCode"`
			City        string `json:"city"`
			Proxy       bool   `json:"proxy"`
			Hosting     bool   `json:"hosting"`
		}
		if json.Unmarshal([]byte(body), &g) == nil {
			s.EgressIPv4 = g.Query
			s.EgressCountry = g.CountryCode
			s.EgressCity = g.City
			s.IsProxyFlagged = g.Proxy
			s.IsHostingFlagged = g.Hosting
		}
	}

	// IPv6 leak check.
	if ip, code := httpGetText(ctx, "https://api6.ipify.org", 5*time.Second); code == 200 && strings.Contains(ip, ":") {
		s.IPv6Leaked = true
		s.IPv6EgressIP = ip
	}

	// Cloudflare trace: colo + loc.
	if body, code := httpGetText(ctx, "https://www.cloudflare.com/cdn-cgi/trace", 5*time.Second); code == 200 {
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(line, "colo=") {
				s.CloudflareColo = strings.TrimPrefix(line, "colo=")
			}
			if strings.HasPrefix(line, "loc=") {
				s.CloudflareLoc = strings.TrimPrefix(line, "loc=")
			}
		}
	}

	// System timezone → country.
	s.SystemTimezone = systemTimezone()
	s.ExpectedTZCC = mimicry.CountryForTimezone(s.SystemTimezone)

	return s
}

// cmdStealth analyzes how detectable the VPN is and how to look local.
func cmdStealth(ctx context.Context, args []string) int {
	fmt.Println("Analyzing detection vectors (IPv6/DNS/timezone/ASN/Cloudflare)...")
	sig := gatherStealthSignal(ctx)
	rep := stealth.Analyze(sig)

	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rep)
		if rep.Score < 60 {
			return 1
		}
		return 0
	}

	fmt.Printf("\nEgress: %s (%s, %s)\n", sig.EgressIPv4, sig.EgressCountry, sig.EgressCity)
	fmt.Printf("System timezone: %s\n\n", sig.SystemTimezone)
	for _, f := range rep.Findings {
		glyph := "●"
		switch f.Severity {
		case stealth.Critical:
			glyph = "✖"
		case stealth.Warn:
			glyph = "▲"
		}
		fmt.Printf("%s [%s] %s\n", glyph, f.Level, f.Title)
		if f.Detail != "" {
			fmt.Printf("    %s\n", f.Detail)
		}
		if f.Fix != "" {
			fmt.Printf("    fix: %s\n", f.Fix)
		}
	}
	bar := stealthBar(rep.Score)
	fmt.Printf("\nStealth score: %d/100  %s\n→ %s\n", rep.Score, bar, rep.Verdict)

	// If timezone mismatch is the issue, offer the one-liner.
	if sig.ExpectedTZCC != "" && sig.EgressCountry != "" &&
		!strings.EqualFold(sig.ExpectedTZCC, sig.EgressCountry) {
		if tz := mimicry.TimezoneFor(sig.EgressCountry); tz != "" {
			fmt.Printf("\nTo match egress: sudo mazzy-vpn mimic   (sets timezone to %s)\n", tz)
		}
	}
	if rep.Score < 60 {
		return 1
	}
	return 0
}

func stealthBar(score int) string {
	filled := score / 10
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", 10-filled) + "]"
}

// execRunner adapts os/exec to mimicry.Runner.
type execRunner struct{}

func (execRunner) Run(bin string, args ...string) (string, error) {
	out, err := exec.Command(bin, args...).CombinedOutput()
	return string(out), err
}

// cmdMimic aligns the system timezone (and shows locale) to the egress country
// so services believe you are local. --process prints env for launching apps
// without changing the system.
func cmdMimic(ctx context.Context, args []string) int {
	sig := gatherStealthSignal(ctx)
	if sig.EgressCountry == "" {
		fmt.Fprintln(os.Stderr, "cannot determine egress country (connect the VPN first)")
		return 1
	}
	mgr := &mimicry.Manager{Runner: execRunner{}, CurrentTZ: systemTimezone}
	plan, ok := mgr.PlanFor(sig.EgressCountry)
	if !ok {
		fmt.Fprintf(os.Stderr, "no timezone mapping for %s\n", sig.EgressCountry)
		return 1
	}

	// --process: print env vars to launch an app as-if-local (no system change).
	if hasFlag(args, "--process") {
		env := mgr.ProcessEnv(sig.EgressCountry)
		fmt.Printf("# launch an app as if local to %s:\n", sig.EgressCountry)
		fmt.Printf("env %s <your-app>\n", strings.Join(env, " "))
		return 0
	}

	fmt.Printf("Egress country: %s\n", sig.EgressCountry)
	fmt.Printf("Timezone: %s → %s\n", plan.FromTZ, plan.ToTZ)
	if !plan.NeedsChange {
		fmt.Println("Already aligned. Nothing to do.")
		return 0
	}
	if !hasFlag(args, "--apply") {
		fmt.Printf("\nDry run. To apply: sudo mazzy-vpn mimic --apply\n")
		fmt.Printf("Or per-process (no system change): mazzy-vpn mimic --process\n")
		return 0
	}
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "applying a system timezone needs root: sudo mazzy-vpn mimic --apply")
		return 1
	}
	if _, err := mgr.ApplySystemTZ(sig.EgressCountry); err != nil {
		fmt.Fprintln(os.Stderr, "failed to set timezone:", err)
		return 1
	}
	fmt.Printf("✔ System timezone set to %s (matches egress %s).\n", plan.ToTZ, sig.EgressCountry)
	fmt.Println("  Restart browsers/apps so they pick up the new timezone.")
	return 0
}
