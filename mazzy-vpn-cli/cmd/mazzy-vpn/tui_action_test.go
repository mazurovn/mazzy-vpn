// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIConnect_ElevatesAndNoBestSentinelIntent(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("non-root only: exercises the elevation build path")
	}
	// Point the intent file at a temp dir so we can inspect it.
	dir := t.TempDir()
	t.Setenv("MAZZY_CONFIG_HOME", dir)

	// Stub elevator so buildPrivilegedCmd resolves sudo deterministically.
	origLookup := elevatorLookup
	elevatorLookup = func(name string) (string, bool) {
		if name == "sudo" {
			return "/usr/bin/sudo", true
		}
		return "", false
	}
	t.Cleanup(func() { elevatorLookup = origLookup })

	var captured *exec.Cmd
	origExec := execProcess
	execProcess = func(c *exec.Cmd, fn tea.ExecCallback) tea.Cmd {
		captured = c
		return func() tea.Msg { return fn(nil) }
	}
	t.Cleanup(func() { execProcess = origExec })

	// --best must NOT write the sentinel intent the daemon cannot resolve (P0-4).
	cmd := requestConnectCmd("--best")
	_ = cmd() // runs the (stubbed) exec builder
	if _, err := os.Stat(dir + "/desired.json"); err == nil {
		t.Error("connect --best must not write a desired.json '--best' sentinel")
	}
	if captured == nil {
		t.Fatal("expected an exec.Cmd to be built for connect --best")
	}
	joined := strings.Join(captured.Args, " ")
	if !strings.Contains(joined, "up") || !strings.Contains(joined, "--best") {
		t.Errorf("expected 'up --best' action, got %v", captured.Args)
	}
	if !strings.HasSuffix(captured.Path, "sudo") {
		t.Errorf("expected elevation via sudo, got %q", captured.Path)
	}

	// A concrete zone DOES record intent (for a running daemon) AND elevates.
	captured = nil
	cmd = requestConnectCmd("Berlin")
	_ = cmd()
	if data, err := os.ReadFile(dir + "/desired.json"); err != nil {
		t.Errorf("concrete zone should record desired intent: %v", err)
	} else if !strings.Contains(string(data), "Berlin") {
		t.Errorf("intent should reference the zone, got %s", data)
	}
	if captured == nil || !strings.Contains(strings.Join(captured.Args, " "), "Berlin") {
		t.Errorf("expected 'up Berlin' action, got %v", captured)
	}
}
