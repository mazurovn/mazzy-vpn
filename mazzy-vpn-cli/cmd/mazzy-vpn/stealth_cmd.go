// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mazurovn/mazzy-vpn/core/mimicry"
	"github.com/mazurovn/mazzy-vpn/core/settings"
	"github.com/mazurovn/mazzy-vpn/core/stealth"
	"github.com/mazurovn/mazzy-vpn/core/zonescore"
)

// stealthScoreOf computes the stealth score for an already-gathered signal.
func stealthScoreOf(sig stealth.Signal) int {
	return stealth.Analyze(sig).Score
}

// recordZoneScore stores the current egress stealth quality for a zone so the
// cleanest-server selector (up --clean) can learn over time.
func recordZoneScore(ctx context.Context, zone string) {
	if zone == "" {
		return
	}
	sig := gatherStealthSignal(ctx)
	if sig.EgressCountry == "" {
		return
	}
	rep := stealth.Analyze(sig)
	_ = zonescore.New().Record(zonescore.Score{
		Zone:         zone,
		StealthScore: rep.Score,
		IsDatacenter: sig.IsHostingFlagged || sig.IsProxyFlagged,
		EgressCC:     sig.EgressCountry,
	})
}

// maybeAutoMimic aligns the system timezone to the egress country when the
// AutoMimic setting is on and we are root. It is a no-op otherwise, and only
// reports when it actually changes something.
func maybeAutoMimic(ctx context.Context) {
	set := settings.NewStore().Load()
	if !set.AutoMimic || os.Geteuid() != 0 {
		return
	}
	sig := gatherStealthSignal(ctx)
	if sig.EgressCountry == "" {
		return
	}
	mgr := &mimicry.Manager{Runner: execRunner{}, CurrentTZ: systemTimezone}
	plan, ok := mgr.PlanFor(sig.EgressCountry)
	if !ok || !plan.NeedsChange {
		return
	}
	if _, err := mgr.ApplySystemTZ(sig.EgressCountry); err == nil {
		fmt.Printf("  timezone  : %s (auto-aligned to %s)\n", safeDisplay(plan.ToTZ), safeDisplay(sig.EgressCountry))
	}
}

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
	// Read the full (bounded) body: a single Read() can return partial data for
	// chunked/TLS responses, which previously dropped Cloudflare trace fields.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	return strings.TrimSpace(string(body)), resp.StatusCode
}

// gatherStealthSignal probes detection vectors from the current egress. The
// three network probes run concurrently so the whole check is bounded by the
// slowest one (~6s) instead of their sum (~16s).
func gatherStealthSignal(ctx context.Context) stealth.Signal {
	s := stealth.Signal{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(3)

	// IPv4 egress + geo (ip-api gives proxy/hosting flags).
	go func() {
		defer wg.Done()
		body, code := httpGetText(ctx, "http://ip-api.com/json/?fields=query,countryCode,city,proxy,hosting", 6*time.Second)
		if code != 200 {
			return
		}
		var g struct {
			Query       string `json:"query"`
			CountryCode string `json:"countryCode"`
			City        string `json:"city"`
			Proxy       bool   `json:"proxy"`
			Hosting     bool   `json:"hosting"`
		}
		if json.Unmarshal([]byte(body), &g) == nil {
			mu.Lock()
			s.EgressIPv4 = g.Query
			s.EgressCountry = g.CountryCode
			s.EgressCity = g.City
			s.IsProxyFlagged = g.Proxy
			s.IsHostingFlagged = g.Hosting
			mu.Unlock()
		}
	}()

	// IPv6 leak check.
	go func() {
		defer wg.Done()
		if ip, code := httpGetText(ctx, "https://api6.ipify.org", 5*time.Second); code == 200 && strings.Contains(ip, ":") {
			mu.Lock()
			s.IPv6Leaked = true
			s.IPv6EgressIP = ip
			mu.Unlock()
		}
	}()

	// Cloudflare trace: colo + loc.
	go func() {
		defer wg.Done()
		if body, code := httpGetText(ctx, "https://www.cloudflare.com/cdn-cgi/trace", 5*time.Second); code == 200 {
			mu.Lock()
			for _, line := range strings.Split(body, "\n") {
				if strings.HasPrefix(line, "colo=") {
					s.CloudflareColo = strings.TrimPrefix(line, "colo=")
				}
				if strings.HasPrefix(line, "loc=") {
					s.CloudflareLoc = strings.TrimPrefix(line, "loc=")
				}
			}
			mu.Unlock()
		}
	}()

	wg.Wait()

	// System timezone → country (local, no network).
	s.SystemTimezone = systemTimezone()
	s.ExpectedTZCC = mimicry.CountryForTimezone(s.SystemTimezone)

	return s
}

// cmdStealth analyzes how detectable the VPN is and how to look local.
func cmdStealth(ctx context.Context, args []string) int {
	fmt.Println(translator().T("cli.stealth.analyzing"))
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

	// Egress IP/country/city come from an untrusted remote JSON API; sanitize
	// before printing so a crafted response cannot inject terminal escapes.
	fmt.Printf("\nEgress: %s (%s, %s)\n", safeDisplay(sig.EgressIPv4), safeDisplay(sig.EgressCountry), safeDisplay(sig.EgressCity))
	fmt.Printf("System timezone: %s\n\n", safeDisplay(sig.SystemTimezone))
	for _, f := range rep.Findings {
		glyph := "●"
		switch f.Severity {
		case stealth.Critical:
			glyph = "✖"
		case stealth.Warn:
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
		fmt.Fprintf(os.Stderr, "no timezone mapping for %s\n", safeDisplay(sig.EgressCountry))
		return 1
	}

	// --process: print env vars to launch an app as-if-local (no system change).
	if hasFlag(args, "--process") {
		env := mgr.ProcessEnv(sig.EgressCountry)
		fmt.Printf("# launch an app as if local to %s:\n", safeDisplay(sig.EgressCountry))
		fmt.Printf("env %s <your-app>\n", strings.Join(env, " "))
		fmt.Println("\n# for a browser, also disable WebRTC IP leak, e.g. Chromium:")
		fmt.Printf("env %s chromium --force-webrtc-ip-handling-policy=disable_non_proxied_udp\n",
			strings.Join(env, " "))
		return 0
	}

	fmt.Printf("Egress country: %s\n", safeDisplay(sig.EgressCountry))
	fmt.Printf("Timezone: %s → %s\n", plan.FromTZ, plan.ToTZ)
	if !plan.NeedsChange {
		fmt.Println(translator().T("cli.stealth.aligned"))
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
	fmt.Printf("✔ System timezone set to %s (matches egress %s).\n", safeDisplay(plan.ToTZ), safeDisplay(sig.EgressCountry))
	fmt.Println("  Restart browsers/apps so they pick up the new timezone.")
	return 0
}
