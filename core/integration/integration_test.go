// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/health"
	"github.com/mazurovn/mazzy-vpn/core/livetest"
	"github.com/mazurovn/mazzy-vpn/core/probe"
	"github.com/mazurovn/mazzy-vpn/core/state"
	"github.com/mazurovn/mazzy-vpn/core/verify"
)

// --- Compile-time interface satisfaction (fails the build on drift) ---

// probe.NetProbe must satisfy health.Probe so the real network prober drives
// the real health monitor without an adapter.
var _ health.Probe = (*probe.NetProbe)(nil)

// --- Scenario 1: connect -> health degrades -> auto-reconnect (parity) ---

// scriptedProbe drives the health monitor through a scripted sequence.
type scriptedProbe struct {
	link  []bool
	inet  []bool
	grace []bool
	i     int
}

func (s *scriptedProbe) at(sl []bool) bool {
	if s.i < len(sl) {
		return sl[s.i]
	}
	return sl[len(sl)-1]
}
func (s *scriptedProbe) LinkPresent(context.Context) bool { return s.at(s.link) }
func (s *scriptedProbe) InternetOK(context.Context) (bool, string) {
	if s.at(s.inet) {
		return true, ""
	}
	return false, "no egress"
}
func (s *scriptedProbe) InStartupGrace(context.Context) bool { return s.at(s.grace) }

type recorder struct{ recovered int }

func (r *recorder) Recover(context.Context, string) error { r.recovered++; return nil }

func TestScenarioHealthDegradesThenRecovers(t *testing.T) {
	// Sequence: healthy, healthy, then two consecutive failures -> recover once.
	sp := &scriptedProbe{
		link:  []bool{true, true, true, true},
		inet:  []bool{true, true, false, false},
		grace: []bool{false},
	}
	rec := &recorder{}
	m := health.New(health.Config{FailureLimit: 2}, sp, rec)

	results := []bool{}
	for tick := 0; tick < 4; tick++ {
		r := m.Check(context.Background())
		results = append(results, r.Healthy)
		sp.i++
	}
	// Ticks: healthy, healthy, fail1, fail2(recover).
	if results[0] != true || results[1] != true {
		t.Fatalf("first two ticks should be healthy: %v", results)
	}
	if rec.recovered != 1 {
		t.Fatalf("expected exactly one recovery after two consecutive failures, got %d", rec.recovered)
	}
}

// --- Scenario 2: live test rolls back and restores previous state ---

// fakeConnector is a livetest.Connector wired to a real state.Store so we test
// the full transactional path end to end.
type fakeConnector struct {
	upErr error
	ups   int
	downs int
}

func (f *fakeConnector) Up(_ context.Context, _ core.Protocol, _ string) error {
	f.ups++
	if f.ups == 1 {
		return f.upErr
	}
	return nil
}
func (f *fakeConnector) Down(context.Context) error { f.downs++; return nil }

type failVerifier struct{}

func (failVerifier) Verify(context.Context) (bool, string) { return false, "egress not confirmed" }

func TestScenarioLiveTestRollsBackToPreviousState(t *testing.T) {
	st := &state.Store{Dir: t.TempDir()}
	// Previous working intent.
	if err := st.Write(state.State{
		Protocol: core.WireGuard, Profile: "prev.conf",
		Desired: core.DesiredUp, Mode: core.ModeNormal,
	}); err != nil {
		t.Fatal(err)
	}

	conn := &fakeConnector{}
	ter := &livetest.Tester{Store: st, Connector: conn, Verifier: failVerifier{}}
	out, err := ter.Run(context.Background(), core.AmneziaWG, "cand.conf", 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if out.Passed || !out.RolledBack {
		t.Fatalf("expected rollback, got %+v", out)
	}
	// State must be restored to the previous profile, not the candidate.
	got, _ := st.Read()
	if got.Profile != "prev.conf" || got.Protocol != core.WireGuard {
		t.Fatalf("state not restored to previous: %+v", got)
	}
	if conn.downs != 1 {
		t.Errorf("candidate must be torn down exactly once, got %d", conn.downs)
	}
}

// --- Scenario 3: verify verdict feeds a UI-facing decision ---

type fixedObserver struct{ o verify.Observation }

func (f fixedObserver) Observe(context.Context) (verify.Observation, error) { return f.o, nil }

func TestScenarioVerifyProducesActionableVerdict(t *testing.T) {
	// A full-tunnel with an IPv6 leak must yield a warning verdict that a UI or
	// agent can act on.
	obs := fixedObserver{o: verify.Observation{
		TunnelActive: true,
		BoundV4:      "203.0.113.9",
		DefaultV4:    "203.0.113.9",
		DefaultV6:    "2001:db8::1",
		BoundV6:      "",
	}}
	r, err := verify.Run(context.Background(), obs)
	if err != nil {
		t.Fatal(err)
	}
	if r.Verdict != verify.Warning || !r.IPv6Leak {
		t.Fatalf("expected ipv6-leak warning verdict, got %+v", r)
	}
}
