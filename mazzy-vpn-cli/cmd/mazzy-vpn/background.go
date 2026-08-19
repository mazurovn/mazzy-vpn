// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mazurovn/mazzy-vpn/core/runstatus"
)

// daemonizedEnv marks a re-executed, already-detached daemon so the self-
// daemonize step runs exactly once (parent forks the detached child, the child
// sees the marker and proceeds to the real run loop).
const daemonizedEnv = "MAZZY_DAEMONIZED"

// logPath returns the daemon activity log file. It honors MAZZY_RUN_DIR for
// tests/dev and otherwise lives beside the heartbeat under /run/mazzy-vpn, so
// the menu's log viewer and the background daemon agree on one location.
func logPath() string {
	if d := os.Getenv("MAZZY_RUN_DIR"); d != "" {
		return filepath.Join(d, "daemon.log")
	}
	return filepath.Join(runDir(), "daemon.log")
}

// maybeDaemonize implements self-detachment for background mode. When the daemon
// is asked to run in the background and is NOT yet the detached child, it forks
// a copy of itself into a new session (setsid) with output redirected to the
// daemon log, then returns detached=true so the caller can exit immediately.
//
// This runs AFTER privilege elevation (sudo/pkexec already granted root and can
// prompt on the terminal), so the detached child inherits root without ever
// needing a tty of its own. That is what lets the menu start a background VPN
// and immediately return to the dashboard, and what lets the tunnel survive
// closing the terminal window (SIGHUP goes to the old session, not the child).
//
// forkExec is indirected for tests so we can assert the child command without
// spawning anything.
var forkExec = func(cmd *exec.Cmd) (int, error) {
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	return pid, nil
}

func maybeDaemonize(background bool) (detached bool, err error) {
	if !background || os.Getenv(daemonizedEnv) == "1" {
		return false, nil
	}
	self, e := os.Executable()
	if e != nil || self == "" {
		return false, fmt.Errorf("cannot locate self executable")
	}
	lf := logPath()
	if e := os.MkdirAll(filepath.Dir(lf), 0o755); e != nil {
		return false, fmt.Errorf("prepare log dir: %w", e)
	}
	out, e := os.OpenFile(lf, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if e != nil {
		return false, fmt.Errorf("open log: %w", e)
	}
	defer out.Close()

	// Re-exec ourselves with the same daemon args (already sans elevation, since
	// we are root here) plus the marker so the child skips this branch.
	child := exec.Command(self, os.Args[1:]...)
	child.Stdin = nil
	child.Stdout = out
	child.Stderr = out
	child.Env = append(os.Environ(), daemonizedEnv+"=1")
	// New session: detach from the controlling terminal so closing the window
	// does not kill the daemon.
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	pid, e := forkExec(child)
	if e != nil {
		return false, fmt.Errorf("start background daemon: %w", e)
	}
	fmt.Printf("%s (pid %d)\n", translator().T("cli.bg.started"), pid)
	return true, nil
}

// daemonRunning reports whether a background daemon is alive by reading the
// heartbeat: fresh (updated recently) AND its PID is still a live process.
func daemonRunning() (runstatus.Snapshot, bool) {
	snap, ok := runstatus.Read()
	if !ok || !snap.Fresh(30*time.Second) {
		return runstatus.Snapshot{}, false
	}
	if snap.PID > 0 && !pidAlive(snap.PID) {
		return runstatus.Snapshot{}, false
	}
	return snap, true
}

// pidAlive reports whether a process with the given PID exists (signal 0).
//
// Cross-privilege correctness: the daemon usually runs as root while the
// menu/TUI runs unprivileged. Signal(0) to a process owned by another user
// returns EPERM — which still proves the process EXISTS. Only ESRCH (no such
// process) means it is truly gone. Treating EPERM as "dead" would make a live
// root daemon look absent to the unprivileged reader, hiding the dashboard.
func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid) // never fails on Unix
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	// EPERM: exists but owned by another uid (still alive). ESRCH: gone.
	return errors.Is(err, syscall.EPERM)
}

// signalDaemon sends SIGTERM to the daemon PID recorded in the heartbeat. It is
// only effective when the caller owns the process (i.e. root, invoked by the
// privileged `stop` subcommand). Returns false when no live daemon was found.
//
// Unprivileged callers must NOT rely on this — they route through the elevated
// `stop` subcommand (see cmdStop) so a root-owned daemon is actually signaled
// instead of failing silently with EPERM. This split is what makes "stop" work
// from the unprivileged menu.
func signalDaemon() bool {
	snap, ok := daemonRunning()
	if !ok || snap.PID <= 0 {
		return false
	}
	p, err := os.FindProcess(snap.PID)
	if err != nil {
		return false
	}
	return p.Signal(syscall.SIGTERM) == nil
}

// cmdStop terminates a running background/session daemon. It requires root so
// it can signal a root-owned daemon; the menu/TUI reach it via the elevation
// path (runPrivileged), reusing any cached sudo credential from connect.
func cmdStop(_ context.Context, _ []string) int {
	if !requireRoot("stop") {
		return 1
	}
	if signalDaemon() {
		return 0
	}
	// No live daemon: not an error for the caller's UX, but signal "nothing
	// stopped" with a distinct code so the menu can report accurately.
	return 3
}

// tailLog returns up to the last n lines of the daemon log (newest at the end),
// for the menu's log viewer. It is best-effort: a missing log yields nil.
func tailLog(n int) []string {
	data, err := os.ReadFile(logPath())
	if err != nil {
		return nil
	}
	trimmed := strings.TrimRight(string(data), "\n")
	if trimmed == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}
