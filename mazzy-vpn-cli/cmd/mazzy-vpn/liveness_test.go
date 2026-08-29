// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"testing"

	"github.com/mazurovn/mazzy-vpn/core/measure"
	"github.com/mazurovn/mazzy-vpn/core/reachcache"
)

func TestListWindow(t *testing.T) {
	cases := []struct {
		name                   string
		cursor, total, visible int
		wantStart, wantEnd     int
	}{
		{"fits entirely", 0, 5, 10, 0, 5},
		{"zero visible shows all", 3, 5, 0, 0, 5},
		{"cursor at top", 0, 40, 10, 0, 10},
		{"cursor mid keeps centered", 20, 40, 10, 15, 25},
		{"cursor at bottom clamps", 39, 40, 10, 30, 40},
	}
	for _, c := range cases {
		s, e := listWindow(c.cursor, c.total, c.visible)
		if s != c.wantStart || e != c.wantEnd {
			t.Errorf("%s: listWindow(%d,%d,%d) = [%d,%d), want [%d,%d)",
				c.name, c.cursor, c.total, c.visible, s, e, c.wantStart, c.wantEnd)
		}
		if c.cursor < s || c.cursor >= e {
			t.Errorf("%s: cursor %d not inside window [%d,%d)", c.name, c.cursor, s, e)
		}
	}
}

func TestReorderByEgressHistory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAZZY_STATE_DIR", dir)
	c := reachcache.New()
	c.RecordFail("FastButDead")
	c.RecordOK("SlowButWorks")

	in := []measure.Result{
		{Name: "FastButDead", ICMPAlive: true, LatencyMS: 10},
		{Name: "Unknown", ICMPAlive: true, LatencyMS: 50},
		{Name: "SlowButWorks", ICMPAlive: true, LatencyMS: 200},
	}
	out := reorderByEgressHistory(in)
	if out[0].Name != "SlowButWorks" {
		t.Errorf("proven-working zone must rank first, got %v", out[0].Name)
	}
	if out[len(out)-1].Name != "FastButDead" {
		t.Errorf("proven-dead zone must rank last, got %v", out[len(out)-1].Name)
	}
	if len(out) != len(in) {
		t.Errorf("membership must not change: %d != %d", len(out), len(in))
	}
}

func TestLivenessLabel(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAZZY_STATE_DIR", dir)
	c := reachcache.New()
	c.RecordOK("Works")
	c.RecordFail("Dead")
	c.RecordFail("Dead")

	if got := livenessLabel("Works"); got != "✔ routes" {
		t.Errorf("Works: got %q", got)
	}
	if got := livenessLabel("Dead"); got != "✖ no-route ×2" {
		t.Errorf("Dead: got %q", got)
	}
	if got := livenessLabel("Never"); got != "· untested" {
		t.Errorf("Never: got %q", got)
	}
}

func TestRecordProbeVerdict(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAZZY_STATE_DIR", dir)

	recordProbeVerdict(ProbeResult{Name: "A", Verdict: "WORKS"})
	recordProbeVerdict(ProbeResult{Name: "B", Verdict: "SERVER_NOT_ROUTING"})
	recordProbeVerdict(ProbeResult{Name: "C", Verdict: "DEAD"})
	recordProbeVerdict(ProbeResult{Name: "D", Verdict: "BAD_CONFIG"}) // says nothing about the server

	c := reachcache.New()
	if v, _ := c.Verdict("A", egressHistoryTTL); v != "ok" {
		t.Errorf("WORKS must record ok, got %s", v)
	}
	for _, z := range []string{"B", "C"} {
		if v, _ := c.Verdict(z, egressHistoryTTL); v != "fail" {
			t.Errorf("%s must record fail, got %s", z, v)
		}
	}
	if v, _ := c.Verdict("D", egressHistoryTTL); v != "unknown" {
		t.Errorf("BAD_CONFIG must record nothing, got %s", v)
	}
}

func TestAllNonFlagValuesAware(t *testing.T) {
	got := allNonFlagValuesAware([]string{"ZoneA", "--uplink", "eth0", "ZoneB", "--deep", "ZoneC"})
	want := []string{"ZoneA", "ZoneB", "ZoneC"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	if out := allNonFlagValuesAware([]string{"--all", "--deep"}); len(out) != 0 {
		t.Errorf("flags only must yield no names, got %v", out)
	}
}

func TestGradeQuality(t *testing.T) {
	cases := []struct {
		q    ProbeQuality
		want string
	}{
		{ProbeQuality{PingsSent: 10, PingsLost: 0, JitterMS: 5, EgressChecks: 3, EgressOK: 3}, "excellent"},
		{ProbeQuality{PingsSent: 10, PingsLost: 1, JitterMS: 30, EgressChecks: 3, EgressOK: 3}, "good"},
		{ProbeQuality{PingsSent: 10, PingsLost: 0, JitterMS: 5, EgressChecks: 3, EgressOK: 2}, "unstable"},
		{ProbeQuality{PingsSent: 10, PingsLost: 5, JitterMS: 5, EgressChecks: 3, EgressOK: 3}, "unstable"},
		{ProbeQuality{PingsSent: 10, PingsLost: 2, JitterMS: 80, EgressChecks: 3, EgressOK: 3}, "poor"},
	}
	for i, c := range cases {
		if got := gradeQuality(c.q); got != c.want {
			t.Errorf("case %d: got %s, want %s", i, got, c.want)
		}
	}
}
