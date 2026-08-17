// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package lock provides the single-flight mutation lock used to serialize
// state-changing VPN operations, mirroring the bash acquire_action_lock /
// acquire_recovery_lock (flock on a lock file).
//
// Only one mutation may run at a time across the whole host. A non-blocking
// acquire fails fast ("another operation in progress"); a recovery acquire
// waits up to a bounded deadline.
package lock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

// ErrBusy is returned when the lock is held by another process (non-blocking).
var ErrBusy = errors.New("another mazzy-vpn mutation is already in progress")

// Mutation is a held single-flight lock. Release with Unlock.
type Mutation struct {
	f *os.File
}

// path returns the mutation lock file path under dir.
func lockPath(dir string) string { return filepath.Join(dir, "mutation.lock") }

// Acquire takes the mutation lock without blocking. It returns ErrBusy if the
// lock is already held. Parity with acquire_action_lock (flock -n).
func Acquire(dir string) (*Mutation, error) {
	f, err := openLockFile(dir)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, ErrBusy
		}
		return nil, fmt.Errorf("flock: %w", err)
	}
	return &Mutation{f: f}, nil
}

// AcquireRecovery takes the mutation lock, waiting up to wait for it. Parity
// with acquire_recovery_lock (flock -w). A zero or negative wait behaves like
// Acquire.
func AcquireRecovery(dir string, wait time.Duration) (*Mutation, error) {
	if wait <= 0 {
		return Acquire(dir)
	}
	f, err := openLockFile(dir)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(wait)
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return &Mutation{f: f}, nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) {
			f.Close()
			return nil, fmt.Errorf("flock: %w", err)
		}
		if time.Now().After(deadline) {
			f.Close()
			return nil, ErrBusy
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// Unlock releases the lock and closes the underlying file.
func (m *Mutation) Unlock() error {
	if m == nil || m.f == nil {
		return nil
	}
	// Closing the fd releases the flock; do it explicitly for clarity.
	ferr := unix.Flock(int(m.f.Fd()), unix.LOCK_UN)
	cerr := m.f.Close()
	m.f = nil
	if ferr != nil {
		return ferr
	}
	return cerr
}

func openLockFile(dir string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, err
	}
	return os.OpenFile(lockPath(dir), os.O_CREATE|os.O_RDWR, 0o600)
}
