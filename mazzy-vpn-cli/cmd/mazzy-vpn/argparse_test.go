// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import "testing"

// TestFirstNonFlag locks in the argument parsing that fixed the `favorite --off
// NAME` and `remove --off NAME` bugs where a flag was mistaken for the name.
func TestFirstNonFlag(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"Berlin"}, "Berlin"},
		{[]string{"--off", "Berlin"}, "Berlin"},
		{[]string{"Berlin", "--off"}, "Berlin"},
		{[]string{"--off"}, ""},
		{nil, ""},
		{[]string{"", "--x", "Zone"}, "Zone"},
	}
	for _, tc := range cases {
		if got := firstNonFlag(tc.args); got != tc.want {
			t.Errorf("firstNonFlag(%v) = %q, want %q", tc.args, got, tc.want)
		}
	}
}

// TestFlagValue covers the value-taking flag parser, including the end-of-args
// edge case (no panic, empty result).
func TestFlagValue(t *testing.T) {
	cases := []struct {
		args []string
		name string
		want string
	}{
		{[]string{"--uplink", "eth0"}, "--uplink", "eth0"},
		{[]string{"Berlin", "--uplink", "wlan0"}, "--uplink", "wlan0"},
		{[]string{"--uplink"}, "--uplink", ""}, // no value at end → "" (no panic)
		{[]string{"--other", "x"}, "--uplink", ""},
	}
	for _, tc := range cases {
		if got := flagValue(tc.args, tc.name); got != tc.want {
			t.Errorf("flagValue(%v, %q) = %q, want %q", tc.args, tc.name, got, tc.want)
		}
	}
}

// TestSafeDisplayStripsInjection is the guard for the terminal-injection audit:
// control characters, ANSI escapes and newlines must never survive into output.
func TestSafeDisplayStripsInjection(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Berlin", "Berlin"},
		{"a\x1b[31mred", "a[31mred"},         // ESC dropped, rest kept
		{"line1\nline2", "line1 line2"},      // newline → space
		{"tab\there", "tab here"},            // tab → space
		{"bel\x07end", "belend"},             // BEL dropped
		{"cr\rlf", "cr lf"},                  // CR → space
		{"unicode-Zürich", "unicode-Zürich"}, // printable Unicode kept
	}
	for _, tc := range cases {
		if got := safeDisplay(tc.in); got != tc.want {
			t.Errorf("safeDisplay(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
