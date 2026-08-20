// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package health

import (
	"context"
	"errors"
	"testing"
)

// fakeProbe is a scriptable probe.
type fakeProbe struct {
	link    bool
	inet    bool
	inetWhy string
	grace   bool
}

func (f *fakeProbe) LinkPresent(context.Context) bool { return f.link }
func (f *fakeProbe) InternetOK(context.Context) (bool, string) {
	return f.inet, f.inetWhy
}
func (f *fakeProbe) InStartupGrace(context.Context) bool { return f.grace }

// fakeRec records recovery calls.
type fakeRec struct {
	calls int
	err   error
	last  string
}

func (r *fakeRec) Recover(_ context.Context, reason string) error {
	r.calls++
	r.last = reason
	return r.err
}

func healthy() *fakeProbe { return &fakeProbe{link: true, inet: true} }

func TestHealthyResetsCounter(t *testing.T) {
	p := healthy()
	rec := &fakeRec{}
	m := New(Config{}, p, rec)
	// Two failures then a recovery success, then healthy resets.
	p.link = false
	m.Check(context.Background())      // fail 1
	r := m.Check(context.Background()) // fail 2 -> recover
	if !r.Recovered {
		t.Fatal("expected recovery at limit 2")
	}
	p.link, p.inet = true, true
	r = m.Check(context.Background())
	if !r.Healthy || m.Failures() != 0 {
		t.Fatalf("healthy tick should reset; got %+v failures=%d", r, m.Failures())
	}
}

func TestRecoversExactlyOnceAtLimit(t *testing.T) {
	p := &fakeProbe{link: false}
	rec := &fakeRec{}
	m := New(Config{FailureLimit: 2}, p, rec)

	m.Check(context.Background())       // 1
	r2 := m.Check(context.Background()) // 2 -> recover
	r3 := m.Check(context.Background()) // 3 -> NO second destructive recover

	if rec.calls != 1 {
		t.Fatalf("expected exactly 1 recovery, got %d", rec.calls)
	}
	if !r2.Recovered || r3.Recovered {
		t.Fatalf("recover should fire only at limit: r2=%v r3=%v", r2.Recovered, r3.Recovered)
	}
}

func TestStartupGraceDefersFailure(t *testing.T) {
	p := &fakeProbe{link: false, grace: true}
	rec := &fakeRec{}
	m := New(Config{}, p, rec)

	r := m.Check(context.Background())
	if !r.Deferred || r.Failures != 0 {
		t.Fatalf("grace should defer without counting: %+v", r)
	}
	if rec.calls != 0 {
		t.Fatal("no recovery during startup grace")
	}
}

func TestInternetFailureCounts(t *testing.T) {
	p := &fakeProbe{link: true, inet: false, inetWhy: "default egress bypasses VPN"}
	rec := &fakeRec{}
	m := New(Config{FailureLimit: 2}, p, rec)
	m.Check(context.Background())
	r := m.Check(context.Background())
	if !r.Recovered || rec.last != "default egress bypasses VPN" {
		t.Fatalf("expected recovery with reason; got %+v last=%q", r, rec.last)
	}
}

func TestRecoverErrorSurfaced(t *testing.T) {
	p := &fakeProbe{link: false}
	rec := &fakeRec{err: errors.New("reconnect failed")}
	m := New(Config{FailureLimit: 1}, p, rec)
	r := m.Check(context.Background())
	if !r.Recovered || r.RecoverError == nil {
		t.Fatalf("expected surfaced recover error; got %+v", r)
	}
}

func TestDefaultLimitIsTwo(t *testing.T) {
	m := New(Config{}, healthy(), &fakeRec{})
	if m.cfg.limit() != 2 {
		t.Fatalf("default limit = %d, want 2", m.cfg.limit())
	}
}

// TestSingleFailureDoesNotRecover locks the core audit lesson: one transient
// failure must NEVER trigger a destructive reconnect (default limit 2).
func TestSingleFailureDoesNotRecover(t *testing.T) {
	p := &fakeProbe{link: true, inet: true}
	rec := &fakeRec{}
	m := New(Config{}, p, rec)
	p.inet = false // one bad tick
	r := m.Check(context.Background())
	if r.Recovered || rec.calls != 0 {
		t.Fatalf("single failure must not recover; got %+v calls=%d", r, rec.calls)
	}
	// Recovery on the next healthy tick is cleared.
	p.inet = true
	if r := m.Check(context.Background()); !r.Healthy || m.Failures() != 0 {
		t.Fatalf("recovery from transient blip; got %+v", r)
	}
}

// TestGraceBetweenFailuresDoesNotResetStreak locks H2: a startup-grace tick
// between two real failures neither counts nor resets — the two failures are
// still consecutive (parity: bash returns early on grace without health_reset).
func TestGraceBetweenFailuresDoesNotResetStreak(t *testing.T) {
	p := &fakeProbe{link: false}
	rec := &fakeRec{}
	m := New(Config{FailureLimit: 2}, p, rec)

	m.Check(context.Background()) // fail 1
	p.grace = true
	if r := m.Check(context.Background()); !r.Deferred || m.Failures() != 1 {
		t.Fatalf("grace tick should defer and preserve streak; got %+v failures=%d", r, m.Failures())
	}
	p.grace = false
	r := m.Check(context.Background()) // fail 2 -> recover
	if !r.Recovered || rec.calls != 1 {
		t.Fatalf("two real failures across a grace tick should recover once; got %+v calls=%d", r, rec.calls)
	}
}
