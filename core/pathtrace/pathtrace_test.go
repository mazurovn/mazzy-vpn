// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package pathtrace

import (
	"context"
	"errors"
	"testing"
)

type fakeResolver struct{ fail bool }

func (r fakeResolver) LookupHost(_ context.Context, _ string) ([]string, error) {
	if r.fail {
		return nil, errors.New("no such host")
	}
	return []string{"192.0.2.1"}, nil
}

type fakePinger struct{ ok bool }

func (p fakePinger) Ping(_ context.Context, _ string) (float64, bool) {
	return 42.0, p.ok
}

func TestTraceAllOK(t *testing.T) {
	tr := &Tracer{
		Resolver: fakeResolver{}, Pinger: fakePinger{ok: true},
		Link:   func(string) bool { return true },
		Egress: func(context.Context, string) (string, error) { return "203.0.113.9", nil },
	}
	res := tr.Run(context.Background(), "vpn.example.com:51820", "vpnaw0")
	if !res.Healthy() {
		t.Fatalf("expected healthy, got %+v", res.Steps)
	}
	if len(res.Steps) != 4 {
		t.Errorf("expected 4 steps, got %d", len(res.Steps))
	}
}

func TestTraceDNSFail(t *testing.T) {
	tr := &Tracer{Resolver: fakeResolver{fail: true}, Pinger: fakePinger{ok: true}}
	res := tr.Run(context.Background(), "bad.example:51820", "")
	if res.Healthy() {
		t.Fatal("DNS failure must make trace unhealthy")
	}
	if res.Steps[0].Status != Fail {
		t.Errorf("first step should fail, got %s", res.Steps[0].Status)
	}
}

func TestTraceEgressFail(t *testing.T) {
	tr := &Tracer{
		Resolver: fakeResolver{}, Pinger: fakePinger{ok: true},
		Link:   func(string) bool { return true },
		Egress: func(context.Context, string) (string, error) { return "", errors.New("timeout") },
	}
	res := tr.Run(context.Background(), "1.2.3.4:51820", "vpnaw0")
	if res.Healthy() {
		t.Fatal("egress failure must make trace unhealthy")
	}
}

func TestTraceNoTunnelYet(t *testing.T) {
	tr := &Tracer{Resolver: fakeResolver{}, Pinger: fakePinger{ok: true}}
	res := tr.Run(context.Background(), "1.2.3.4:51820", "")
	// DNS+ping ok, link/egress "warn" (not up) — not a fail.
	if !res.Healthy() {
		t.Errorf("no-tunnel should warn, not fail: %+v", res.Steps)
	}
}

func TestLiteralIPSkipsDNS(t *testing.T) {
	tr := &Tracer{Pinger: fakePinger{ok: true}}
	res := tr.Run(context.Background(), "203.0.113.5:51820", "")
	if res.Steps[0].Status != OK {
		t.Errorf("literal IP should resolve OK, got %s", res.Steps[0].Status)
	}
}
