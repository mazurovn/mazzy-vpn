// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package lock

import (
	"testing"
	"time"
)

func TestAcquireAndRelease(t *testing.T) {
	dir := t.TempDir()
	m, err := Acquire(dir)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if err := m.Unlock(); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	// Re-acquire after release must succeed.
	m2, err := Acquire(dir)
	if err != nil {
		t.Fatalf("re-acquire: %v", err)
	}
	m2.Unlock()
}

func TestSingleFlightBusy(t *testing.T) {
	dir := t.TempDir()
	m, err := Acquire(dir)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer m.Unlock()

	// A second acquire from a different fd on the same file must report busy.
	// NB: same process re-lock via a distinct fd — flock is per open file
	// description, so a fresh open contends correctly.
	_, err = Acquire(dir)
	if err != ErrBusy {
		t.Fatalf("expected ErrBusy, got %v", err)
	}
}

func TestRecoveryWaitTimesOut(t *testing.T) {
	dir := t.TempDir()
	m, _ := Acquire(dir)
	defer m.Unlock()

	start := time.Now()
	_, err := AcquireRecovery(dir, 300*time.Millisecond)
	if err != ErrBusy {
		t.Fatalf("expected ErrBusy after wait, got %v", err)
	}
	if elapsed := time.Since(start); elapsed < 250*time.Millisecond {
		t.Errorf("recovery returned too early: %v", elapsed)
	}
}

func TestRecoveryAcquiresAfterRelease(t *testing.T) {
	dir := t.TempDir()
	m, _ := Acquire(dir)
	go func() {
		time.Sleep(150 * time.Millisecond)
		m.Unlock()
	}()
	got, err := AcquireRecovery(dir, 2*time.Second)
	if err != nil {
		t.Fatalf("recovery should acquire after release: %v", err)
	}
	got.Unlock()
}
