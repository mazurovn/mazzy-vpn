// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/mazurovn/mazzy-vpn/core/runstatus"
)

// TestDaemonRunningReadsHeartbeat verifies the menu's liveness check honors both
// freshness and PID liveness, so a stale or dead-PID heartbeat is not shown as
// an active background daemon.
func TestDaemonRunningReadsHeartbeat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAZZY_RUN_DIR", dir)

	// No file yet → not running.
	if _, ok := daemonRunning(); ok {
		t.Fatal("no heartbeat should mean no daemon")
	}

	// Fresh heartbeat with OUR pid (alive) → running.
	w := runstatus.NewWriter("Berlin", "mazzy0", "AmneziaWG", true)
	w.SetState(runstatus.StateProtected, "mazzy0", "9.9.9.9")
	if snap, ok := daemonRunning(); !ok {
		t.Fatal("fresh heartbeat with a live pid should be running")
	} else if snap.Zone != "Berlin" || !snap.Background {
		t.Errorf("unexpected snapshot: %+v", snap)
	}

	// Dead PID → not running (impossible-high pid).
	w2 := runstatus.NewWriter("Zed", "mazzy0", "wg", true)
	_ = w2
	// Overwrite the file with a dead pid by writing directly.
	deadForcePID(t, dir, 2147480000)
	if _, ok := daemonRunning(); ok {
		t.Error("heartbeat with a dead pid must not be reported as running")
	}
}

// deadForcePID rewrites the heartbeat with a specific (dead) pid.
func deadForcePID(t *testing.T, dir string, pid int) {
	t.Helper()
	p := filepath.Join(dir, "status.json")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	s := string(data)
	// crude: replace the pid line
	s = replacePID(s, pid)
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
}

func replacePID(json string, pid int) string {
	start := strings.Index(json, `"pid":`)
	if start < 0 {
		return json
	}
	end := strings.IndexByte(json[start:], ',')
	if end < 0 {
		return json
	}
	return json[:start] + `"pid": ` + itoa(pid) + json[start+end:]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// TestForwardToActiveDaemon ensures a second connect request does not start a
// competing daemon: it is delivered through the owner's shared intent file.
func TestForwardToActiveDaemon(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAZZY_RUN_DIR", dir)

	w := runstatus.NewWriter("Berlin", "vpnaw0", "AmneziaWG", false)
	defer w.Close()
	w.SetState(runstatus.StateProtected, "vpnaw0", "9.9.9.9")

	snap, running, err := forwardToActiveDaemon("Amsterdam")
	if err != nil || !running {
		t.Fatalf("forwardToActiveDaemon = (%+v, %v, %v), want live owner", snap, running, err)
	}
	if snap.Zone != "Berlin" {
		t.Fatalf("owner zone = %q, want Berlin", snap.Zone)
	}
	di, ok := readDesired()
	if !ok || di.Desired != "up" || di.Zone != "Amsterdam" {
		t.Fatalf("forwarded intent = %+v ok=%v, want Amsterdam/up", di, ok)
	}
}

// TestTailLog returns the trailing lines of the daemon log and tolerates a
// missing file.
func TestTailLog(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAZZY_RUN_DIR", dir)

	if got := tailLog(10); got != nil {
		t.Errorf("missing log should yield nil, got %v", got)
	}

	body := "l1\nl2\nl3\nl4\nl5\n"
	if err := os.WriteFile(filepath.Join(dir, "daemon.log"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got := tailLog(3)
	if len(got) != 3 || got[0] != "l3" || got[2] != "l5" {
		t.Errorf("tail(3) = %v, want [l3 l4 l5]", got)
	}
}

// TestMaybeDaemonizeForksDetachedChild asserts that background mode builds a
// detached, setsid child carrying the daemonized marker, and that the marker
// short-circuits a second fork (the child proceeds to the real loop).
func TestMaybeDaemonizeForksDetachedChild(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAZZY_RUN_DIR", dir)

	var captured *exec.Cmd
	orig := forkExec
	forkExec = func(cmd *exec.Cmd) (int, error) {
		captured = cmd
		return 4242, nil
	}
	t.Cleanup(func() { forkExec = orig })

	// background=false → no fork, not detached.
	if detached, err := maybeDaemonize(false); err != nil || detached {
		t.Fatalf("background=false must not detach (detached=%v err=%v)", detached, err)
	}
	if captured != nil {
		t.Fatal("no child should be forked when background is off")
	}

	// background=true → forks a detached child.
	detached, err := maybeDaemonize(true)
	if err != nil {
		t.Fatalf("maybeDaemonize error: %v", err)
	}
	if !detached {
		t.Fatal("background=true must detach the parent")
	}
	if captured == nil {
		t.Fatal("expected a forked child")
	}
	if captured.SysProcAttr == nil || !captured.SysProcAttr.Setsid {
		t.Error("detached child must start a new session (Setsid)")
	}
	if !envHas(captured.Env, daemonizedEnv+"=1") {
		t.Error("child must carry the daemonized marker so it skips re-forking")
	}

	// With the marker set (as the child would see), we must NOT fork again.
	t.Setenv(daemonizedEnv, "1")
	captured = nil
	if detached, err := maybeDaemonize(true); err != nil || detached {
		t.Fatalf("marked child must run in-process (detached=%v err=%v)", detached, err)
	}
	if captured != nil {
		t.Error("marked child must not fork a grandchild")
	}
}

func envHas(env []string, kv string) bool {
	for _, e := range env {
		if e == kv {
			return true
		}
	}
	return false
}

// TestDrawLiveDashboardFallsBackWithoutHeartbeat ensures the menu header falls
// back cleanly (returns false) when no fresh heartbeat exists.
func TestDrawLiveDashboardFallsBackWithoutHeartbeat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAZZY_RUN_DIR", dir)
	if drawLiveDashboard() {
		t.Error("no heartbeat should mean no live dashboard (fallback)")
	}
}

// TestSignalDaemonNoDaemon reports false when nothing is running.
func TestSignalDaemonNoDaemon(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAZZY_RUN_DIR", dir)
	if signalDaemon() {
		t.Error("signalDaemon with no daemon should return false")
	}
}

// TestPidAliveTreatsEPERMAsAlive is the regression guard for the cross-privilege
// bug: an unprivileged menu probing a root-owned daemon PID gets EPERM from
// Signal(0), which STILL proves the process exists. Treating it as dead would
// hide the dashboard. pid 1 (init) is always root-owned and alive.
func TestPidAliveTreatsEPERMAsAlive(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("non-root only: EPERM path requires an unprivileged caller")
	}
	if !pidAlive(1) {
		t.Error("pid 1 (root init) must be reported alive even when we cannot signal it (EPERM)")
	}
}

// TestPidAliveRejectsDeadAndInvalid ensures a clearly-dead / invalid pid is not
// reported alive.
func TestPidAliveRejectsDeadAndInvalid(t *testing.T) {
	if pidAlive(0) || pidAlive(-1) {
		t.Error("non-positive pids must not be alive")
	}
	if pidAlive(2147480000) {
		t.Error("an impossibly-high pid must not be alive (ESRCH)")
	}
}

// TestCmdStopRequiresRoot ensures the stop subcommand refuses to run unprivileged
// (the menu/TUI reach it through elevation instead).
func TestCmdStopRequiresRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("non-root only")
	}
	if code := cmdStop(context.Background(), nil); code != 1 {
		t.Errorf("cmdStop unprivileged should return 1 (requires root), got %d", code)
	}
}

// TestShortDur formats durations compactly.
func TestShortDur(t *testing.T) {
	cases := map[time.Duration]string{
		30 * time.Second: "30s",
		5 * time.Minute:  "5m",
		90 * time.Minute: "1h30m",
	}
	for d, want := range cases {
		if got := shortDur(d); got != want {
			t.Errorf("shortDur(%v) = %q, want %q", d, got, want)
		}
	}
}

// TestTrunc shortens long strings with an ellipsis.
func TestTrunc(t *testing.T) {
	if got := trunc("hello", 10); got != "hello" {
		t.Errorf("short string unchanged, got %q", got)
	}
	if got := trunc("abcdefgh", 4); got != "abc…" {
		t.Errorf("trunc = %q, want abc…", got)
	}
}

// TestTruncDisplayWidth is the regression guard for audit P2-5: truncation must
// budget by DISPLAY COLUMNS so double-width CJK/emoji never overflow a box.
func TestTruncDisplayWidth(t *testing.T) {
	// Each CJK glyph is 2 columns wide. "東京サーバー" = 6 runes / 12 columns.
	if w := runewidth.StringWidth(trunc("東京サーバー", 6)); w > 6 {
		t.Errorf("CJK trunc exceeded 6 columns: width=%d", w)
	}
	// A pure-ASCII string still truncates by rune==column.
	if got := trunc("abcdefgh", 4); got != "abc…" {
		t.Errorf("ascii trunc = %q, want abc…", got)
	}
	// n<=0 yields empty; n==1 yields just the ellipsis.
	if got := trunc("anything", 0); got != "" {
		t.Errorf("trunc n=0 = %q, want empty", got)
	}
	if got := trunc("anything", 1); got != "…" {
		t.Errorf("trunc n=1 = %q, want …", got)
	}
}

// TestDaemonRunningSurvivesStaleHeartbeat is the regression guard for the
// "busy daemon looks dead" deadlock: existence is PID-based, so a daemon whose
// loop spent >30s in a connect/failover phase (stale UpdatedAt) must STILL be
// reported as running — otherwise `stop` says "nothing to stop" and a second
// `daemon` request crashes into the mutation lock held by the live daemon.
func TestDaemonRunningSurvivesStaleHeartbeat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAZZY_RUN_DIR", dir)

	w := runstatus.NewWriter("Berlin", "vpnaw0", "AmneziaWG", true)
	w.SetState(runstatus.StateReconnect, "vpnaw0", "")
	// Backdate UpdatedAt far beyond any freshness window; PID stays ours (alive).
	backdateHeartbeat(t, dir, time.Now().Add(-10*time.Minute).Unix())

	if snap, ok := daemonRunning(); !ok {
		t.Fatal("stale heartbeat with a live PID must still be a running daemon")
	} else if snap.State != runstatus.StateReconnect {
		t.Errorf("snapshot state = %q, want reconnecting", snap.State)
	}
}

// backdateHeartbeat rewrites updated_at in the raw heartbeat file.
func backdateHeartbeat(t *testing.T, dir string, ts int64) {
	t.Helper()
	p := filepath.Join(dir, "status.json")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	s := string(data)
	start := strings.Index(s, `"updated_at":`)
	if start < 0 {
		t.Fatal("no updated_at field")
	}
	end := strings.IndexByte(s[start:], ',')
	if end < 0 {
		t.Fatal("unterminated updated_at")
	}
	s = s[:start] + `"updated_at": ` + strconv.FormatInt(ts, 10) + s[start+end:]
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
}
