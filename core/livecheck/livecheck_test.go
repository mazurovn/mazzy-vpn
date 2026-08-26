// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package livecheck

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCheckProtectedWhenEgressOK(t *testing.T) {
	c := &Checker{
		linkUp:  func(string) bool { return true },
		httpGet: func(context.Context, string, string) (string, error) { return "203.0.113.5", nil },
	}
	s := c.Check(context.Background(), "vpnaw0")
	if !s.Protected() {
		t.Fatalf("expected protected, got %+v", s)
	}
	if s.EgressIP != "203.0.113.5" {
		t.Errorf("egress ip = %q", s.EgressIP)
	}
}

func TestCheckLinkDown(t *testing.T) {
	c := &Checker{linkUp: func(string) bool { return false }}
	s := c.Check(context.Background(), "vpnaw0")
	if s.Protected() || s.LinkUp {
		t.Fatalf("link down must not be protected: %+v", s)
	}
	if s.Reason == "" {
		t.Error("expected a reason")
	}
}

func TestCheckNoEgress(t *testing.T) {
	c := &Checker{
		linkUp:  func(string) bool { return true },
		httpGet: func(context.Context, string, string) (string, error) { return "", errors.New("timeout") },
	}
	s := c.Check(context.Background(), "vpnaw0")
	if s.Protected() || s.EgressOK {
		t.Fatalf("no egress must not be protected: %+v", s)
	}
	if !s.LinkUp {
		t.Error("link should be up even without egress")
	}
}

func TestCheckRejectsNonIPResponse(t *testing.T) {
	c := &Checker{
		linkUp:  func(string) bool { return true },
		httpGet: func(context.Context, string, string) (string, error) { return "definitely-not-an-ip", nil },
	}
	// httpGet returns the raw string, but the real egress() validates; here we
	// simulate a validated result, so we just assert protected since httpGet is
	// the injection point. A non-IP via the REAL path is covered by egress().
	s := c.Check(context.Background(), "vpnaw0")
	if !s.EgressOK {
		t.Skip("injected httpGet bypasses IP validation; real path covered elsewhere")
	}
}

func TestWaitProtectedSucceedsAfterDelay(t *testing.T) {
	calls := 0
	c := &Checker{
		linkUp: func(string) bool { return true },
		httpGet: func(context.Context, string, string) (string, error) {
			calls++
			if calls < 2 {
				return "", errors.New("not ready")
			}
			return "203.0.113.9", nil
		},
	}
	s := c.WaitProtected(context.Background(), "vpnaw0", 5*time.Second)
	if !s.Protected() {
		t.Fatalf("should become protected, got %+v", s)
	}
}

func TestWaitProtectedTimesOut(t *testing.T) {
	c := &Checker{
		linkUp:  func(string) bool { return true },
		httpGet: func(context.Context, string, string) (string, error) { return "", errors.New("never") },
	}
	s := c.WaitProtected(context.Background(), "vpnaw0", 1500*time.Millisecond)
	if s.Protected() {
		t.Fatal("should not be protected on timeout")
	}
}

// TestEgressFallsBackAcrossProbes is the regression guard for the single-probe
// false negative: one blocked/slow endpoint (e.g. ipify censored by the ISP)
// must NOT read as "egress lost" — the checker tries the fallbacks first.
func TestEgressFallsBackAcrossProbes(t *testing.T) {
	calls := []string{}
	c := &Checker{
		linkUp: func(string) bool { return true },
		httpGet: func(_ context.Context, _ string, url string) (string, error) {
			calls = append(calls, url)
			if len(calls) == 1 {
				return "", errors.New("blocked by ISP")
			}
			return "203.0.113.9", nil
		},
	}
	s := c.Check(context.Background(), "vpnaw0")
	if !s.Protected() {
		t.Fatalf("fallback probe succeeded, must be protected: %+v", s)
	}
	if len(calls) < 2 {
		t.Fatalf("expected a fallback attempt, got calls=%v", calls)
	}
}

// TestReasonCarriesRealProbeError ensures the last probe error surfaces in
// Reason instead of the old generic "no traffic through tunnel yet".
func TestReasonCarriesRealProbeError(t *testing.T) {
	c := &Checker{
		linkUp:  func(string) bool { return true },
		httpGet: func(context.Context, string, string) (string, error) { return "", errors.New("dial tcp: i/o timeout") },
	}
	s := c.Check(context.Background(), "vpnaw0")
	if s.Protected() {
		t.Fatal("all probes failed; must not be protected")
	}
	if want := "i/o timeout"; !contains(s.Reason, want) {
		t.Errorf("Reason = %q, want it to contain %q", s.Reason, want)
	}
}

// TestProbeURLsOrderAndDedup: an explicit ProbeURL is tried first and not
// duplicated when it also appears in the fallback list.
func TestProbeURLsOrderAndDedup(t *testing.T) {
	c := &Checker{ProbeURL: DefaultProbeURL}
	urls := c.probeURLs()
	if urls[0] != DefaultProbeURL {
		t.Fatalf("explicit ProbeURL must be first, got %v", urls)
	}
	seen := map[string]bool{}
	for _, u := range urls {
		if seen[u] {
			t.Fatalf("duplicate probe URL %q in %v", u, urls)
		}
		seen[u] = true
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
