// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"os"
	"strings"
	"testing"
)

// withStubElevator swaps the elevator lookup + invoker for the duration of a
// test and restores them afterwards.
func withStubElevator(t *testing.T, avail map[string]string, invoke func(path string, args []string) error) {
	t.Helper()
	origLookup, origInvoke, origDispatch := elevatorLookup, runInvoker, dispatchInProcess
	elevatorLookup = func(name string) (string, bool) {
		p, ok := avail[name]
		return p, ok
	}
	runInvoker = invoke
	t.Cleanup(func() {
		elevatorLookup, runInvoker, dispatchInProcess = origLookup, origInvoke, origDispatch
	})
}

func TestRunPrivileged_ElevatesWhenNotRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("must run as non-root to exercise the elevation path")
	}
	var gotPath string
	var gotArgs []string
	withStubElevator(t, map[string]string{"sudo": "/usr/bin/sudo"}, func(path string, args []string) error {
		gotPath, gotArgs = path, args
		return nil
	})

	code := runPrivileged(context.Background(), "up", "--best")
	if code != 0 {
		t.Fatalf("expected success exit 0, got %d", code)
	}
	if gotPath != "/usr/bin/sudo" {
		t.Fatalf("expected sudo elevator, got %q", gotPath)
	}
	// args = [self, "up", "--best"]
	if len(gotArgs) < 3 || gotArgs[len(gotArgs)-2] != "up" || gotArgs[len(gotArgs)-1] != "--best" {
		t.Fatalf("elevated args wrong: %v", gotArgs)
	}
}

func TestRunPrivileged_PrefersSudoOverPkexec(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("non-root only")
	}
	var gotPath string
	withStubElevator(t, map[string]string{
		"sudo":   "/usr/bin/sudo",
		"pkexec": "/usr/bin/pkexec",
	}, func(path string, args []string) error {
		gotPath = path
		return nil
	})
	_ = runPrivileged(context.Background(), "disconnect")
	if gotPath != "/usr/bin/sudo" {
		t.Fatalf("expected sudo preferred, got %q", gotPath)
	}
}

func TestRunPrivileged_FallsBackToPkexec(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("non-root only")
	}
	var gotPath string
	withStubElevator(t, map[string]string{"pkexec": "/usr/bin/pkexec"},
		func(path string, args []string) error { gotPath = path; return nil })
	_ = runPrivileged(context.Background(), "recover")
	if gotPath != "/usr/bin/pkexec" {
		t.Fatalf("expected pkexec fallback, got %q", gotPath)
	}
}

func TestRunPrivileged_NoElevatorReturnsError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("non-root only")
	}
	called := false
	withStubElevator(t, map[string]string{}, func(path string, args []string) error {
		called = true
		return nil
	})
	code := runPrivileged(context.Background(), "up", "--best")
	if code == 0 {
		t.Fatal("expected non-zero exit when no elevator is available")
	}
	if called {
		t.Fatal("invoker must not be called when no elevator exists")
	}
}

func TestRunPrivileged_ReadOnlyDispatchesInProcess(t *testing.T) {
	origDispatch := dispatchInProcess
	t.Cleanup(func() { dispatchInProcess = origDispatch })

	var dispatched []string
	dispatchInProcess = func(args []string) int {
		dispatched = args
		return 7
	}
	// "status" is not in privilegedSubcommands → always in-process.
	code := runPrivileged(context.Background(), "status", "--json")
	if code != 7 {
		t.Fatalf("expected in-process exit code 7, got %d", code)
	}
	if len(dispatched) != 2 || dispatched[0] != "status" {
		t.Fatalf("unexpected in-process args: %v", dispatched)
	}
}

func TestRunPrivileged_SurfacesChildExitCode(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("non-root only")
	}
	withStubElevator(t, map[string]string{"sudo": "/usr/bin/sudo"},
		func(path string, args []string) error {
			// Simulate a real child failure with an exit status.
			return &fakeExitErr{code: 3}
		})
	code := runPrivileged(context.Background(), "up", "berlin")
	if code != 3 {
		t.Fatalf("expected child exit code 3 surfaced, got %d", code)
	}
}

func TestPrivilegedSubcommandsMatchDispatch(t *testing.T) {
	// Every privileged subcommand must be a real dispatchable command so the
	// menu/TUI never elevate an unknown verb.
	real := map[string]bool{
		"up": true, "connect": true, "disconnect": true,
		"recover": true, "auto": true, "daemon": true, "mimic": true,
		"stop": true, "trust": true, "disarm": true,
	}
	for k := range privilegedSubcommands {
		if !real[k] {
			t.Errorf("privileged subcommand %q is not a known dispatch verb", k)
		}
	}
}

func TestBuildPrivilegedCmd_NonRootUsesElevator(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("non-root only")
	}
	withStubElevator(t, map[string]string{"sudo": "/usr/bin/sudo"}, nil)
	c, err := buildPrivilegedCmd("up", "--best")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(c.Path, "sudo") {
		t.Fatalf("expected sudo path, got %q", c.Path)
	}
	joined := strings.Join(c.Args, " ")
	if !strings.Contains(joined, "up") || !strings.Contains(joined, "--best") {
		t.Fatalf("expected up --best in args, got %v", c.Args)
	}
}

// fakeExitErr implements the ExitError shape used by exitCode via errors.As-like
// type assertion. We embed the real behaviour by matching *exec.ExitError is not
// possible directly, so we test exitCode separately below.
type fakeExitErr struct{ code int }

func (e *fakeExitErr) Error() string { return "exit status" }
func (e *fakeExitErr) ExitCode() int { return e.code }
