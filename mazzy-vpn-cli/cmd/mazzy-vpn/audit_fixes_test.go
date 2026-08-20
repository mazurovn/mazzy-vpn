// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// TestDesiredPathIsSharedRuntimeDir is the regression guard for audit P0-A: the
// intent file must live in the SHARED runtime dir (runDir), NOT under
// os.UserConfigDir() ($HOME/.config), which differs between the unprivileged
// writer (HOME=/home/user) and the root daemon reader (HOME=/root under sudo).
//
// Crucially this test pins ONLY runDir via MAZZY_RUN_DIR — the same variable the
// daemon's heartbeat already honors — and asserts the writer/reader agree even
// when HOME changes underneath them (simulating the sudo env_reset split that
// the old code silently broke).
func TestDesiredPathIsSharedRuntimeDir(t *testing.T) {
	runtime := t.TempDir()
	t.Setenv("MAZZY_RUN_DIR", runtime)

	// The path must be inside runDir and must not depend on HOME/XDG at all.
	want := filepath.Join(runtime, "desired.json")
	if got := desiredPath(); got != want {
		t.Fatalf("desiredPath() = %q, want shared runtime path %q", got, want)
	}

	// Writer runs as the user (HOME=/home/user)...
	t.Setenv("HOME", "/home/user")
	t.Setenv("XDG_CONFIG_HOME", "")
	if err := writeDesired("Berlin", "down"); err != nil {
		t.Fatalf("writeDesired: %v", err)
	}
	writerPath := desiredPath()

	// ...reader runs as root (HOME=/root, as under `sudo`); it MUST read the same
	// file. With the old UserConfigDir() logic these two would diverge.
	t.Setenv("HOME", "/root")
	readerPath := desiredPath()
	if writerPath != readerPath {
		t.Fatalf("writer path %q and reader path %q diverge across HOME change (P0-A regression)", writerPath, readerPath)
	}
	di, ok := readDesired()
	if !ok || di.Desired != "down" || di.Zone != "Berlin" {
		t.Fatalf("root reader must see the user's intent, got %+v ok=%v", di, ok)
	}
}

// TestDesiredFileIsRootReadable verifies the intent file is world-readable so a
// root daemon can consume an intent written by the unprivileged UI (P0-A).
func TestDesiredFileIsRootReadable(t *testing.T) {
	t.Setenv("MAZZY_RUN_DIR", t.TempDir())
	if err := writeDesired("Oslo", "up"); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(desiredPath())
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o044 == 0 {
		t.Errorf("intent file mode %o is not group/other-readable; a root daemon may not read the UI's intent", fi.Mode().Perm())
	}
}

// TestIntentStalenessEnforced is the regression guard for audit P0-B: a stale
// "down" intent must not be honored (it would wedge a freshly-started daemon).
func TestIntentStalenessEnforced(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		di   desiredIntent
		want bool
	}{
		{"fresh", desiredIntent{Desired: "up", TS: now.Unix()}, true},
		{"recent-within-window", desiredIntent{Desired: "down", TS: now.Add(-1 * time.Minute).Unix()}, true},
		{"stale-beyond-window", desiredIntent{Desired: "down", TS: now.Add(-10 * time.Minute).Unix()}, false},
		{"zero-ts", desiredIntent{Desired: "down", TS: 0}, false},
		{"future-ts-clock-skew", desiredIntent{Desired: "up", TS: now.Add(5 * time.Minute).Unix()}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := intentFresh(tc.di, now); got != tc.want {
				t.Errorf("intentFresh(%+v) = %v, want %v", tc.di, got, tc.want)
			}
		})
	}
}

// TestReadDesiredDropsStaleFile ensures the on-disk stale intent is ignored end
// to end (not just the pure predicate), so a leftover down-intent from a prior
// session cannot pause a new daemon (P0-B).
func TestReadDesiredDropsStaleFile(t *testing.T) {
	t.Setenv("MAZZY_RUN_DIR", t.TempDir())
	// Write a stale intent directly (TS well past the window).
	if err := writeDesired("Berlin", "down"); err != nil {
		t.Fatal(err)
	}
	// Backdate it by rewriting with an old timestamp.
	old := desiredIntent{Zone: "Berlin", Desired: "down", TS: time.Now().Add(-time.Hour).Unix()}
	writeRawDesired(t, old)
	if _, ok := readDesired(); ok {
		t.Error("a stale down-intent must be ignored by readDesired (P0-B)")
	}
}

// TestProfileDisplayNameNormalization is the regression guard for audit P1-C:
// the foreground connect and the daemon must present the SAME label for a zone.
func TestProfileDisplayNameNormalization(t *testing.T) {
	cases := map[string]string{
		"/etc/mazzy/Berlin.conf":    "Berlin",
		"Berlin.conf":               "Berlin",
		"Berlin":                    "Berlin",
		"/x/y/Oslo-01.ovpn":         "Oslo-01",
		"weird.name.with.dots.conf": "weird.name.with.dots",
	}
	for in, want := range cases {
		if got := profileDisplayName(in); got != want {
			t.Errorf("profileDisplayName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestDisplayVersionStripsLeadingV is the regression guard for audit P1-F: a
// tag-stamped build (v2.2.0) and a dev build (2.2.0) must print identically.
func TestDisplayVersionStripsLeadingV(t *testing.T) {
	orig := version
	t.Cleanup(func() { version = orig })
	cases := map[string]string{
		"2.2.0":         "2.2.0",
		"v2.2.0":        "2.2.0",
		"v2.3.0-rc1":    "2.3.0-rc1",
		"2.2.0-vpn.dev": "2.2.0-vpn.dev",
	}
	for in, want := range cases {
		version = in
		if got := displayVersion(); got != want {
			t.Errorf("displayVersion() with %q = %q, want %q", in, got, want)
		}
	}
}

// TestFirstNonFlagValueAware is the regression guard for audit P2-7: the shared
// positional parser must skip value-flag tokens (e.g. `--uplink eth0`) so the
// value is never mistaken for the zone name, for every value-flag.
func TestFirstNonFlagValueAware(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"Berlin"}, "Berlin"},
		{[]string{"--uplink", "eth0", "Berlin"}, "Berlin"},
		{[]string{"Berlin", "--uplink", "eth0"}, "Berlin"},
		{[]string{"--uplink", "eth0"}, ""}, // no name, just a value-flag
		{[]string{"--best"}, ""},
		{[]string{"--proto", "wireguard", "Oslo"}, "Oslo"},
		{[]string{"--type", "llm", "Paris"}, "Paris"},
		{[]string{"--clean", "--uplink", "wlan0", "Tokyo"}, "Tokyo"},
		{nil, ""},
	}
	for _, tc := range cases {
		if got := firstNonFlagValueAware(tc.args); got != tc.want {
			t.Errorf("firstNonFlagValueAware(%v) = %q, want %q", tc.args, got, tc.want)
		}
	}
}

// writeRawDesired writes an exact intent struct (bypassing the fresh TS stamp)
// so tests can inject a backdated intent.
func writeRawDesired(t *testing.T, di desiredIntent) {
	t.Helper()
	p := desiredPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte("{\"zone\":\"" + di.Zone + "\",\"desired\":\"" + di.Desired + "\",\"ts\":" + strconv.FormatInt(di.TS, 10) + "}")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
