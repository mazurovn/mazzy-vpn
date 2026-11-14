// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package monitor

import (
	"testing"

	"github.com/mazurovn/mazzy-vpn/core/pathtrace"
)

func TestUncheckedIsUnknown(t *testing.T) {
	v := FromLive(Live{})
	if v.Connection != Unknown || v.Handshake != Unknown || v.Path != Unknown {
		t.Fatalf("%+v", v)
	}
	if len(v.Points) != 0 {
		t.Fatal("invented a sample")
	}
}

func TestProtectedIsGreenWithoutHandshake(t *testing.T) {
	v := FromLive(Live{Checked: true, Link: true, Egress: true, IP: "203.0.113.9", Timed: true, ProbeMS: 40})
	if v.Connection != Green || v.Handshake != Unknown {
		t.Fatalf("%+v", v)
	}
	if len(v.Points) != 1 || !v.Points[0].HasMS || v.Points[0].MS != 40 {
		t.Fatalf("points %+v", v.Points)
	}
}

func TestFailedProbeDoesNotInventLatency(t *testing.T) {
	v := FromLive(Live{Checked: true, Link: true, Reason: "no traffic through tunnel yet", Timed: true, ProbeMS: 12})
	if v.Connection != Yellow || v.Points[0].HasMS {
		t.Fatalf("%+v", v)
	}
}

func TestTraceWorstStep(t *testing.T) {
	light, _ := FromTrace(nil)
	if light != Unknown {
		t.Fatal(light)
	}
	light, text := FromTrace([]pathtrace.Step{
		{Name: "dns", Status: pathtrace.OK, Detail: "1.2.3.4"},
		{Name: "link", Status: pathtrace.Fail, Detail: "down"},
	})
	if light != Red || text == "" {
		t.Fatalf("%s %s", light, text)
	}
}
