// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package measure

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

// nopConn is a writable/closable no-op net.Conn for tests (net.Pipe would
// block on Write with no reader).
type nopConn struct{ net.Conn }

func (nopConn) Write(b []byte) (int, error) { return len(b), nil }
func (nopConn) Close() error                { return nil }

// fakeDialer returns scripted latency/error per address.
type fakeDialer struct {
	latency map[string]time.Duration
	fail    map[string]bool
}

func (f *fakeDialer) DialContext(_ context.Context, _, address string) (net.Conn, error) {
	if f.fail[address] {
		return nil, errors.New("connection refused")
	}
	if d, ok := f.latency[address]; ok {
		time.Sleep(d)
	}
	return nopConn{}, nil
}

// fakeResolver resolves everything to a fixed IP unless marked failing.
type fakeResolver struct{ fail map[string]bool }

func (r *fakeResolver) LookupHost(_ context.Context, host string) ([]string, error) {
	if r.fail[host] {
		return nil, errors.New("no such host")
	}
	return []string{"192.0.2.1"}, nil
}

func TestProbeReachable(t *testing.T) {
	m := &Measurer{Dialer: &fakeDialer{}, Resolver: &fakeResolver{}, Timeout: time.Second}
	r := m.Probe(context.Background(), "berlin", "1.2.3.4:51820")
	if !r.Reachable {
		t.Fatalf("expected reachable, got %+v", r)
	}
}

func TestProbeUnreachable(t *testing.T) {
	fd := &fakeDialer{fail: map[string]bool{"1.2.3.4:51820": true}}
	m := &Measurer{Dialer: fd, Resolver: &fakeResolver{}, Timeout: time.Second}
	r := m.Probe(context.Background(), "down", "1.2.3.4:51820")
	if r.Reachable || r.Err == "" {
		t.Fatalf("expected unreachable with error, got %+v", r)
	}
}

func TestProbeDNSFailure(t *testing.T) {
	m := &Measurer{Dialer: &fakeDialer{}, Resolver: &fakeResolver{fail: map[string]bool{"bad.example": true}}, Timeout: time.Second}
	r := m.Probe(context.Background(), "dns", "bad.example:51820")
	if r.Reachable {
		t.Fatalf("dns failure must be unreachable, got %+v", r)
	}
}

func TestProbeEmptyEndpoint(t *testing.T) {
	m := New()
	r := m.Probe(context.Background(), "x", "")
	if r.Reachable || r.Err != "no endpoint" {
		t.Fatalf("empty endpoint should fail cleanly, got %+v", r)
	}
}

func TestRankBestOrdersByReachabilityThenLatency(t *testing.T) {
	fd := &fakeDialer{
		latency: map[string]time.Duration{
			"192.0.2.10:51820": 5 * time.Millisecond,
			"192.0.2.20:51820": 40 * time.Millisecond,
		},
		fail: map[string]bool{"192.0.2.30:51820": true},
	}
	m := &Measurer{Dialer: fd, Resolver: &fakeResolver{}, Timeout: time.Second}
	targets := []Target{
		{"slow", "192.0.2.20:51820"},
		{"down", "192.0.2.30:51820"},
		{"fast", "192.0.2.10:51820"},
	}
	ranked := m.RankBest(context.Background(), targets)
	if ranked[0].Name != "fast" {
		t.Errorf("fastest should rank first, got %s", ranked[0].Name)
	}
	if ranked[len(ranked)-1].Name != "down" {
		t.Errorf("unreachable should rank last, got %s", ranked[len(ranked)-1].Name)
	}
}
