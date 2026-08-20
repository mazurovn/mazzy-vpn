// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package bootrecovery

import (
	"os"
	"path/filepath"
	"testing"
)

func requiredGate(t *testing.T) *Gate {
	t.Helper()
	return &Gate{Dir: t.TempDir(), Required: true}
}

// TestMutationBlockedUntilReady is the core fail-closed invariant: after a
// reboot, no state (or any non-ready state) blocks network mutations.
func TestMutationBlockedUntilReady(t *testing.T) {
	g := requiredGate(t)

	// No state file at all → blocked (fresh boot).
	if g.MutationAllowed() {
		t.Fatal("mutation must be blocked when no boot-recovery state exists")
	}

	for _, s := range []State{Running, TestRecovery, AwaitingEgress, AwaitingClean, RecoveryOnly} {
		if err := g.Write(s); err != nil {
			t.Fatalf("write %s: %v", s, err)
		}
		if g.MutationAllowed() {
			t.Errorf("mutation must be blocked in state %q", s)
		}
	}

	if err := g.Write(Ready); err != nil {
		t.Fatal(err)
	}
	if !g.MutationAllowed() {
		t.Fatal("mutation must be allowed once ready")
	}
}

// TestNotRequiredAllowsEverything covers unprivileged/dev mode.
func TestNotRequiredAllowsEverything(t *testing.T) {
	g := &Gate{Dir: t.TempDir(), Required: false}
	if !g.MutationAllowed() || !g.ServiceGate() {
		t.Fatal("non-required gate must permit everything")
	}
}

func TestServiceGateStates(t *testing.T) {
	g := requiredGate(t)
	allow := map[State]bool{Ready: true, AwaitingEgress: true, AwaitingClean: true}
	for _, s := range []State{Running, TestRecovery, AwaitingEgress, AwaitingClean, Ready, RecoveryOnly} {
		g.Write(s)
		if got := g.ServiceGate(); got != allow[s] {
			t.Errorf("ServiceGate(%q) = %v, want %v", s, got, allow[s])
		}
	}
	// Unknown (no file) → blocked.
	g2 := requiredGate(t)
	if g2.ServiceGate() {
		t.Error("ServiceGate must block on unknown state")
	}
}

func TestServiceExitCodes(t *testing.T) {
	g := requiredGate(t)
	cases := map[State]int{
		Running: 75, TestRecovery: 75,
		AwaitingEgress: 0, AwaitingClean: 0,
		RecoveryOnly: 77,
	}
	for s, want := range cases {
		g.Write(s)
		if got := g.ServiceExitCode(); got != want {
			t.Errorf("ExitCode(%q) = %d, want %d", s, got, want)
		}
	}
	// Unknown → 77 (manual).
	g2 := requiredGate(t)
	if g2.ServiceExitCode() != 77 {
		t.Errorf("unknown exit code = %d, want 77", g2.ServiceExitCode())
	}
}

func TestWriteRejectsInvalidState(t *testing.T) {
	g := requiredGate(t)
	if err := g.Write(State("bogus")); err == nil {
		t.Fatal("expected rejection of invalid state")
	}
}

// TestSymlinkStateRefused locks the `! -L` protection: a symlinked state file
// must be treated as Unknown (blocked), not followed.
func TestSymlinkStateRefused(t *testing.T) {
	g := requiredGate(t)
	// Point the state path at a file that says "ready" via a symlink.
	target := filepath.Join(t.TempDir(), "evil")
	os.WriteFile(target, []byte("ready\n"), 0o600)
	if err := os.Symlink(target, g.file()); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if g.Read() != Unknown {
		t.Fatal("symlinked state file must be refused (Unknown)")
	}
	if g.MutationAllowed() {
		t.Fatal("symlinked 'ready' must NOT allow mutations")
	}
}

func TestFilePermissions(t *testing.T) {
	g := requiredGate(t)
	g.Write(Ready)
	fi, _ := os.Stat(g.file())
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("state file perm = %o, want 600", fi.Mode().Perm())
	}
}
