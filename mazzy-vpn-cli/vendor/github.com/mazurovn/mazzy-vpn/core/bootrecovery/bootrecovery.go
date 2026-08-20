// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package bootrecovery implements the boot-recovery gate: after a reboot the
// system must independently confirm a safe protected egress BEFORE any network
// mutation is allowed. Until then, status/profiles are readable but connect/
// disconnect/reconnect are blocked. This prevents the daemon from "inventing" a
// new connection merely because a unit started at boot.
//
// Parity with the bash boot_recovery_state_* / *_gate functions.
package bootrecovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// State is the boot-recovery lifecycle state.
type State string

const (
	Running        State = "running"
	TestRecovery   State = "test-recovery"
	AwaitingEgress State = "awaiting-egress"
	AwaitingClean  State = "awaiting-cleanup"
	Ready          State = "ready"
	RecoveryOnly   State = "recovery-only"
	// Unknown is used when no valid state file is present.
	Unknown State = ""
)

var validStates = map[State]bool{
	Running: true, TestRecovery: true, AwaitingEgress: true,
	AwaitingClean: true, Ready: true, RecoveryOnly: true,
}

// Gate manages the boot-recovery state under Dir. When Required is false (e.g.
// unprivileged/dev mode) the gate is a no-op that permits everything.
type Gate struct {
	Dir      string
	Required bool
}

func (g *Gate) file() string { return filepath.Join(g.Dir, "boot-recovery-state") }

// Write atomically persists a valid state (temp+rename+fsync, 0700/0600).
func (g *Gate) Write(s State) error {
	if !validStates[s] {
		return fmt.Errorf("bootrecovery: invalid state %q", s)
	}
	if err := os.MkdirAll(g.Dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(g.Dir, 0o700); err != nil {
		return err
	}
	return atomicWrite(g.file(), []byte(string(s)+"\n"))
}

// Read returns the current state, or Unknown if absent/invalid. It refuses to
// follow a symlink (parity with the `! -L` guard) to avoid state redirection.
func (g *Gate) Read() State {
	fi, err := os.Lstat(g.file())
	if err != nil || fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		return Unknown
	}
	data, err := os.ReadFile(g.file())
	if err != nil {
		return Unknown
	}
	s := State(strings.TrimSpace(string(data)))
	if !validStates[s] {
		return Unknown
	}
	return s
}

// MutationAllowed reports whether a network mutation may proceed. Parity with
// boot_recovery_mutation_gate: only the "ready" state permits mutations. When
// the gate is not Required, everything is allowed.
func (g *Gate) MutationAllowed() bool {
	if !g.Required {
		return true
	}
	return g.Read() == Ready
}

// ServiceGate reports whether the connection executor may run at boot. Parity
// with boot_recovery_service_gate: ready or the two awaiting-* states.
func (g *Gate) ServiceGate() bool {
	if !g.Required {
		return true
	}
	switch g.Read() {
	case Ready, AwaitingEgress, AwaitingClean:
		return true
	default:
		return false
	}
}

// ServiceExitCode maps the current state to the executor exit code, mirroring
// boot_recovery_service_exit_code: 75 = retry later, 0 = ok/no-op, 77 = manual.
func (g *Gate) ServiceExitCode() int {
	switch g.Read() {
	case Running, TestRecovery:
		return 75
	case AwaitingEgress, AwaitingClean:
		return 0
	case RecoveryOnly:
		return 77
	default:
		return 77
	}
}

// atomicWrite writes data via temp+rename with fsync of file and directory.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".boot-recovery.*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	fail := func(e error) error { tmp.Close(); os.Remove(name); return e }
	if err := tmp.Chmod(0o600); err != nil {
		return fail(err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fail(err)
	}
	if err := tmp.Sync(); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
