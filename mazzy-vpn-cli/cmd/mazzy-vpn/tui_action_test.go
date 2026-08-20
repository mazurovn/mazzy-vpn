// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIConnect_StartsNonBlockingSessionDaemon(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("non-root only: exercises the elevation build path")
	}
	// Point the intent file at a temp dir so we can inspect it. The intent file
	// now lives in the SHARED runtime dir (runDir) so the unprivileged writer and
	// the root daemon reader agree on one path (audit P0-A); tests pin it here.
	dir := t.TempDir()
	t.Setenv("MAZZY_RUN_DIR", dir)

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

	// --best must start a detached session daemon instead of foreground `up`,
	// which used to suspend Bubble Tea until the VPN disconnected.
	cmd := requestConnectCmd("--best")
	_ = cmd() // runs the (stubbed) exec builder
	if _, err := os.Stat(desiredPath()); err == nil {
		t.Error("connect --best must not write a desired.json '--best' sentinel")
	}
	if captured == nil {
		t.Fatal("expected an exec.Cmd to be built for connect --best")
	}
	joined := strings.Join(captured.Args, " ")
	if !strings.Contains(joined, "daemon") || !strings.Contains(joined, "--best") || !strings.Contains(joined, "--session") {
		t.Errorf("expected 'daemon --best --session' action, got %v", captured.Args)
	}
	if strings.Contains(joined, " up ") {
		t.Errorf("TUI must not use blocking foreground up: %v", captured.Args)
	}
	if !strings.HasSuffix(captured.Path, "sudo") {
		t.Errorf("expected elevation via sudo, got %q", captured.Path)
	}

	// A concrete zone starts the same non-blocking session mode.
	captured = nil
	cmd = requestConnectCmd("Berlin")
	_ = cmd()
	if captured == nil {
		t.Fatal("expected a command for concrete zone")
	}
	joined = strings.Join(captured.Args, " ")
	if !strings.Contains(joined, "daemon") || !strings.Contains(joined, "Berlin") || !strings.Contains(joined, "--session") {
		t.Errorf("expected 'daemon Berlin --session' action, got %v", captured.Args)
	}
}
