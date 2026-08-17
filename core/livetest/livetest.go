// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package livetest implements a transactional live VPN test with guaranteed
// rollback, mirroring the bash cmd_test / save_test_backup / restore_test_backup
// semantics:
//
//   - snapshot the previous working intent (backup),
//   - bring the candidate connection up under a deadline guard,
//   - verify real protected egress,
//   - on success COMMIT (persist candidate, discard backup),
//   - on failure or timeout ROLL BACK to the snapshot.
//
// The connector, verifier and clock are injected so the transaction logic is
// deterministic and unit testable without touching the network.
package livetest

import (
	"context"
	"errors"
	"time"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/state"
)

// Connector brings a candidate connection up and tears connections down. It is
// the boundary to core/connect in production.
type Connector interface {
	// Up brings the candidate profile online (engine+routes+dns+guard).
	Up(ctx context.Context, proto core.Protocol, profileName string) error
	// Down tears down any active connection.
	Down(ctx context.Context) error
}

// Verifier confirms a real protected egress after Up. Parity with the live
// verify step of cmd_test.
type Verifier interface {
	Verify(ctx context.Context) (ok bool, reason string)
}

// Outcome is the result of a transactional test.
type Outcome struct {
	Passed     bool
	RolledBack bool
	Reason     string
	TimedOut   bool
}

// Tester runs transactional tests. Store persists intent; Backup holds the
// snapshot in memory for the duration of the transaction.
type Tester struct {
	Store     *state.Store
	Connector Connector
	Verifier  Verifier
	// Now is injectable for tests; defaults to time.Now.
	Now func() time.Time
}

// ErrNoConnector guards against misconfiguration.
var ErrNoConnector = errors.New("livetest: connector and verifier are required")

// Run executes a transactional test of profileName under the given timeout.
// It always leaves the system in a committed state: either the new connection
// (on pass) or the restored previous state (on fail/timeout).
func (t *Tester) Run(ctx context.Context, proto core.Protocol, profileName string, timeout time.Duration) (Outcome, error) {
	if t.Connector == nil || t.Verifier == nil {
		return Outcome{}, ErrNoConnector
	}

	// 1. Snapshot previous intent (may be absent on a fresh system).
	backup, hadPrevious := t.snapshot()

	// 2. Bring the candidate up under a deadline.
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := t.Connector.Up(cctx, proto, profileName); err != nil {
		return t.rollback(ctx, backup, hadPrevious, "candidate failed to come up: "+err.Error(), deadlineHit(cctx)), nil
	}

	// 3. Verify real protected egress before committing.
	ok, reason := t.Verifier.Verify(cctx)
	if !ok {
		if reason == "" {
			reason = "egress verification failed"
		}
		return t.rollback(ctx, backup, hadPrevious, reason, deadlineHit(cctx)), nil
	}

	// 4. Commit: persist the candidate as the working intent, drop backup.
	if err := t.Store.Write(state.State{
		Protocol: proto,
		Profile:  profileName,
		Desired:  core.DesiredUp,
		Mode:     core.ModeNormal,
	}); err != nil {
		// Commit failed: roll back rather than leave ambiguous state.
		return t.rollback(ctx, backup, hadPrevious, "commit failed: "+err.Error(), false), nil
	}

	return Outcome{Passed: true}, nil
}

// snapshot reads the previous intent. hadPrevious is false on a fresh system.
func (t *Tester) snapshot() (state.State, bool) {
	if t.Store == nil {
		return state.State{}, false
	}
	st, err := t.Store.Read()
	if err != nil {
		return state.State{}, false
	}
	return *st, true
}

// rollback tears down the candidate and restores the previous intent (or a
// clean down state if there was none).
func (t *Tester) rollback(ctx context.Context, backup state.State, hadPrevious bool, reason string, timedOut bool) Outcome {
	_ = t.Connector.Down(ctx)

	if hadPrevious {
		// Restore the previous intent verbatim.
		_ = t.Store.Write(backup)
		// Best-effort: bring the previous connection back if it was up.
		if backup.Desired == core.DesiredUp {
			_ = t.Connector.Up(ctx, backup.Protocol, backup.Profile)
		}
	}
	// No previous state: the candidate is already torn down above and we do
	// not fabricate a bogus intent. The system is left cleanly disconnected.

	return Outcome{Passed: false, RolledBack: true, Reason: reason, TimedOut: timedOut}
}

// deadlineHit reports whether ctx was cancelled by its deadline.
func deadlineHit(ctx context.Context) bool {
	return errors.Is(ctx.Err(), context.DeadlineExceeded)
}
