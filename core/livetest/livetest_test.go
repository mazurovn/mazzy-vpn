// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package livetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/state"
)

// fakeConn records up/down calls and can be scripted to fail.
type fakeConn struct {
	ups     []string // "proto/profile"
	downs   int
	upErr   error // error for the FIRST up (candidate); later ups succeed
	upCalls int
}

func (f *fakeConn) Up(_ context.Context, proto core.Protocol, profile string) error {
	f.upCalls++
	f.ups = append(f.ups, string(proto)+"/"+profile)
	if f.upCalls == 1 {
		return f.upErr
	}
	return nil
}
func (f *fakeConn) Down(context.Context) error { f.downs++; return nil }

type fakeVerifier struct {
	ok     bool
	reason string
}

func (v fakeVerifier) Verify(context.Context) (bool, string) { return v.ok, v.reason }

func newTester(t *testing.T, conn Connector, ver Verifier) (*Tester, *state.Store) {
	t.Helper()
	st := &state.Store{Dir: t.TempDir()}
	return &Tester{Store: st, Connector: conn, Verifier: ver}, st
}

func writePrevious(t *testing.T, st *state.Store) {
	t.Helper()
	if err := st.Write(state.State{
		Protocol: core.WireGuard, Profile: "prev.conf",
		Desired: core.DesiredUp, Mode: core.ModeNormal,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPassCommitsCandidate(t *testing.T) {
	conn := &fakeConn{}
	ter, st := newTester(t, conn, fakeVerifier{ok: true})
	writePrevious(t, st)

	out, err := ter.Run(context.Background(), core.AmneziaWG, "cand.conf", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Passed || out.RolledBack {
		t.Fatalf("expected pass, got %+v", out)
	}
	// Committed state must be the candidate.
	got, _ := st.Read()
	if got.Protocol != core.AmneziaWG || got.Profile != "cand.conf" || got.Desired != core.DesiredUp {
		t.Errorf("committed state wrong: %+v", got)
	}
	if conn.downs != 0 {
		t.Errorf("pass path must not tear down; downs=%d", conn.downs)
	}
}

func TestVerifyFailureRollsBackToPrevious(t *testing.T) {
	conn := &fakeConn{}
	ter, st := newTester(t, conn, fakeVerifier{ok: false, reason: "no egress"})
	writePrevious(t, st)

	out, _ := ter.Run(context.Background(), core.AmneziaWG, "cand.conf", 5*time.Second)
	if out.Passed || !out.RolledBack || out.Reason != "no egress" {
		t.Fatalf("expected rollback with reason, got %+v", out)
	}
	// Candidate must have been torn down.
	if conn.downs != 1 {
		t.Errorf("expected one teardown, got %d", conn.downs)
	}
	// State restored to previous.
	got, _ := st.Read()
	if got.Protocol != core.WireGuard || got.Profile != "prev.conf" {
		t.Errorf("state not restored to previous: %+v", got)
	}
	// Previous was up, so it must have been brought back (up #2).
	if conn.upCalls != 2 {
		t.Errorf("expected candidate up + previous restore up = 2, got %d", conn.upCalls)
	}
}

func TestCandidateUpFailureRollsBack(t *testing.T) {
	conn := &fakeConn{upErr: errors.New("engine up failed")}
	ter, st := newTester(t, conn, fakeVerifier{ok: true})
	writePrevious(t, st)

	out, _ := ter.Run(context.Background(), core.AmneziaWG, "cand.conf", 5*time.Second)
	if out.Passed || !out.RolledBack {
		t.Fatalf("expected rollback on candidate up failure, got %+v", out)
	}
	got, _ := st.Read()
	if got.Profile != "prev.conf" {
		t.Errorf("state not restored: %+v", got)
	}
}

func TestNoPreviousLeavesCleanDown(t *testing.T) {
	conn := &fakeConn{}
	ter, st := newTester(t, conn, fakeVerifier{ok: false, reason: "bad"})
	// No writePrevious: fresh system.

	out, _ := ter.Run(context.Background(), core.AmneziaWG, "cand.conf", 5*time.Second)
	if !out.RolledBack {
		t.Fatalf("expected rollback, got %+v", out)
	}
	// Candidate torn down; no bogus state fabricated.
	if conn.downs != 1 {
		t.Errorf("expected teardown, got %d", conn.downs)
	}
	if _, err := st.Read(); err == nil {
		t.Error("no previous state existed; rollback must not fabricate one")
	}
}

// slowConn blocks in Up until the context deadline fires, simulating a
// candidate that never becomes ready within the timeout.
type slowConn struct {
	downs    int
	upCalls  int
	restored bool
}

func (s *slowConn) Up(ctx context.Context, _ core.Protocol, profile string) error {
	s.upCalls++
	if s.upCalls == 1 {
		<-ctx.Done() // candidate: block until deadline
		return ctx.Err()
	}
	s.restored = true // restore up must NOT be pre-cancelled
	return nil
}
func (s *slowConn) Down(context.Context) error { s.downs++; return nil }

// TestTimeoutRollsBackWithLiveParentContext locks audit T3: when the candidate
// deadline fires, rollback must run on the PARENT context (not the expired
// child), so teardown and previous-restore actually execute.
func TestTimeoutRollsBackWithLiveParentContext(t *testing.T) {
	conn := &slowConn{}
	ter, st := newTester(t, conn, fakeVerifier{ok: true})
	writePrevious(t, st)

	out, _ := ter.Run(context.Background(), core.AmneziaWG, "cand.conf", 100*time.Millisecond)
	if !out.RolledBack || !out.TimedOut {
		t.Fatalf("expected timed-out rollback, got %+v", out)
	}
	if conn.downs != 1 {
		t.Errorf("teardown must run on live parent ctx; downs=%d", conn.downs)
	}
	if !conn.restored {
		t.Error("previous connection must be restored on a live context after timeout")
	}
	got, _ := st.Read()
	if got.Profile != "prev.conf" {
		t.Errorf("state not restored after timeout: %+v", got)
	}
}

func TestMissingConnectorErrors(t *testing.T) {
	ter := &Tester{Store: &state.Store{Dir: t.TempDir()}}
	if _, err := ter.Run(context.Background(), core.AmneziaWG, "x", time.Second); err != ErrNoConnector {
		t.Fatalf("expected ErrNoConnector, got %v", err)
	}
}
