// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLinkPresentFalseForMissingInterface(t *testing.T) {
	p := &NetProbe{Interface: "definitely-not-an-iface-xyz"}
	if p.LinkPresent(context.Background()) {
		t.Fatal("expected missing interface to be absent")
	}
	p2 := &NetProbe{Interface: ""}
	if p2.LinkPresent(context.Background()) {
		t.Fatal("empty interface must be absent")
	}
}

func TestLinkPresentTrueForLoopback(t *testing.T) {
	// "lo" exists on any Linux host.
	p := &NetProbe{Interface: "lo"}
	if !p.LinkPresent(context.Background()) {
		t.Fatal("loopback should be present")
	}
}

func TestStartupGraceClock(t *testing.T) {
	base := time.Unix(1000, 0)
	p := &NetProbe{
		StartupDeadline: base.Add(60 * time.Second),
		Now:             func() time.Time { return base.Add(10 * time.Second) },
	}
	if !p.InStartupGrace(context.Background()) {
		t.Fatal("should be within grace at t+10s of 60s window")
	}
	p.Now = func() time.Time { return base.Add(90 * time.Second) }
	if p.InStartupGrace(context.Background()) {
		t.Fatal("should be past grace at t+90s")
	}
}

func TestInternetOKParsesValidIP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("203.0.113.7\n"))
	}))
	defer srv.Close()
	// No RequireDefaultEgress: a single successful bound probe is enough. We
	// probe without binding (default egress) by leaving Interface empty so the
	// httptest loopback server is reachable.
	p := &NetProbe{ProbeURL: srv.URL, Timeout: 2 * time.Second}
	ok, reason := p.InternetOK(context.Background())
	if !ok {
		t.Fatalf("expected ok, got reason=%q", reason)
	}
}

func TestInternetOKRejectsNonIP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("not-an-ip"))
	}))
	defer srv.Close()
	p := &NetProbe{ProbeURL: srv.URL, Timeout: 2 * time.Second}
	ok, reason := p.InternetOK(context.Background())
	if ok || reason == "" {
		t.Fatalf("expected failure on non-IP body; ok=%v reason=%q", ok, reason)
	}
}

func TestInternetOKFailsWhenUnreachable(t *testing.T) {
	p := &NetProbe{ProbeURL: "http://127.0.0.1:1/", Timeout: 500 * time.Millisecond}
	ok, reason := p.InternetOK(context.Background())
	if ok || reason == "" {
		t.Fatalf("expected failure for unreachable probe; ok=%v", ok)
	}
}
