// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
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
	// Under systemd (Type=notify), self-daemonizing would make the READY/WATCHDOG
	// notifications come from a pid that is not MAINPID (NotifyAccess=main drops
	// them) and the unit would hang until TimeoutStartSec. systemd already
	// provides the detachment --background exists for, so run in the foreground.
	if os.Getenv("NOTIFY_SOCKET") != "" {
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
	rotateLog(lf)
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

// maxLogBytes bounds daemon.log growth. The log was previously O_APPEND forever
// with no rotation — a 24/7 daemon grew it without limit.
const maxLogBytes = 5 << 20 // 5 MiB

// rotateLog moves an oversized daemon.log to daemon.log.1 (replacing any
// previous rotation) so a fresh session starts with a bounded file while the
// previous history stays inspectable.
func rotateLog(path string) {
	fi, err := os.Stat(path)
	if err != nil || fi.Size() < maxLogBytes {
		return
	}
	_ = os.Rename(path, path+".1")
}

// daemonRunning reports whether a background daemon EXISTS: the heartbeat file
// is readable and its PID is a live mazzy-vpn process.
//
// Existence is deliberately PID-based, NOT freshness-based. The old rule
// (Fresh(30s) && pidAlive) declared a busy daemon dead whenever its loop spent
// longer than 30s in a connect/failover phase — then `stop` reported "nothing
// to stop", the dashboard vanished mid-reconnect, and a new `daemon` request
// skipped intent-forwarding, fell through to the mutation lock (held by the
// very-much-alive daemon) and failed with "another operation is in progress".
// Freshness is a HEALTH signal for the dashboard (HeartbeatAge), not an
// existence test. PID-reuse after a crash is guarded by checking the process
// actually is mazzy-vpn via /proc.
func daemonRunning() (runstatus.Snapshot, bool) {
	snap, ok := runstatus.Read()
	if !ok {
		return runstatus.Snapshot{}, false
	}
	if snap.PID <= 0 || !pidAlive(snap.PID) {
		return runstatus.Snapshot{}, false
	}
	if !procLooksLikeMazzy(snap.PID) {
		return runstatus.Snapshot{}, false
	}
	return snap, true
}

// procLooksLikeMazzy guards against PID reuse: a crashed daemon leaves its
// heartbeat behind, and the recorded PID may later belong to an unrelated
// process. /proc/<pid>/cmdline is world-readable even for root processes; when
// it cannot be read at all we err on the side of "it is the daemon" (the old
// EPERM-tolerant behavior).
func procLooksLikeMazzy(pid int) bool {
	// Strongest signal: the executable behind the PID really is a mazzy-vpn
	// binary. This defeats the "argv contains mazzy-vpn" spoof a crafted
	// process could otherwise use to get itself signalled (gate finding 2).
	if exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil && exe != "" {
		base := exe
		if i := strings.LastIndexByte(base, '/'); i >= 0 {
			base = base[i+1:]
		}
		// "(deleted)" suffix appears after an in-place upgrade — still ours.
		base = strings.TrimSuffix(base, " (deleted)")
		// Prefix match accepts the real binary ("mazzy-vpn") and the test binary
		// ("mazzy-vpn.test") while still requiring the actual EXECUTABLE FILE to
		// be a mazzy-vpn — a far higher bar than an argv substring, and combined
		// with the root-owned /run/mazzy-vpn dir the PID cannot be spoofed.
		return strings.HasPrefix(base, "mazzy-vpn")
	}
	// Fall back to cmdline only when /proc/<pid>/exe is unreadable (EPERM under
	// hidepid, or non-Linux): a weaker but still useful check.
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return true // cannot verify at all: assume alive (old EPERM-tolerant behavior)
	}
	return strings.Contains(strings.ReplaceAll(string(data), "\x00", " "), "mazzy-vpn")
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

// signalDaemonPID sends SIGTERM to the daemon PID recorded in the heartbeat.
// It is only effective when the caller owns the process (i.e. root, invoked by
// the privileged `stop` subcommand). The PID is returned so the caller can wait
// for actual termination instead of falsely reporting a successful stop.
func signalDaemonPID() (int, bool) {
	snap, ok := daemonRunning()
	if !ok || snap.PID <= 0 {
		return 0, false
	}
	p, err := os.FindProcess(snap.PID)
	if err != nil || p.Signal(syscall.SIGTERM) != nil {
		return 0, false
	}
	return snap.PID, true
}

// signalDaemon is retained as the small boolean helper used by dashboard tests.
func signalDaemon() bool {
	_, ok := signalDaemonPID()
	return ok
}

// waitDaemonExit waits for a signalled daemon to disappear. A bounded wait is
// necessary because a daemon can be inside an egress probe when SIGTERM arrives.
func waitDaemonExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for pidAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	return !pidAlive(pid)
}

// cmdStop terminates a running background/session daemon. It requires root so
// it can signal a root-owned daemon; the menu/TUI reach it via the elevation
// path (runPrivileged), reusing any cached sudo credential from connect.
func cmdStop(_ context.Context, _ []string) int {
	if !requireRoot("stop") {
		return 1
	}
	// First prevent a still-running daemon from treating interface teardown as a
	// fault and reconnecting. This is also safe when no daemon is present.
	if err := recordDownIntent(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	pid, ok := signalDaemonPID()
	if !ok {
		// No live daemon: not an error for the caller's UX, but signal "nothing
		// stopped" with a distinct code so the menu can report accurately.
		return 3
	}
	if !waitDaemonExit(pid, 35*time.Second) {
		// Escalate to SIGKILL rather than leaving a wedged daemon (and any armed
		// kill-switch) alive — parity with disarm/heal, which never give up.
		fmt.Fprintf(os.Stderr, "daemon pid %d ignored SIGTERM after 35s — sending SIGKILL\n", pid)
		if p, err := os.FindProcess(pid); err == nil {
			_ = p.Signal(syscall.SIGKILL)
		}
		if !waitDaemonExit(pid, 5*time.Second) {
			fmt.Fprintf(os.Stderr, "daemon pid %d survived SIGKILL; run: sudo mazzy-vpn disarm\n", pid)
			return 1
		}
	}
	return 0
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
