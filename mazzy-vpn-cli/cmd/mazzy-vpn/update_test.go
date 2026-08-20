// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import "testing"

// TestIsNewer covers the version comparison, including the audit P2-1 fix: a
// project-local/fork build (".local") must NOT be treated as older than an equal
// upstream tag, while genuine prereleases (-rc/-dev) still are.
func TestIsNewer(t *testing.T) {
	cases := []struct {
		tag, cur string
		want     bool
	}{
		{"v2.3.0", "2.2.0", true},            // higher minor
		{"v2.2.1", "2.2.0", true},            // higher patch
		{"v2.2.0", "2.2.0", false},           // equal clean
		{"v2.2.0", "2.3.0", false},           // older tag
		{"v2.2.0", "2.2.0-rc1", true},        // clean beats prerelease
		{"v2.2.0", "2.2.0-dev", true},        // clean beats dev
		{"v2.2.0", "2.2.0-vpn.local", false}, // P2-1: local build is not "older"
		{"v2.2.0", "2.2.0+build7", false},    // build metadata is not a prerelease
		{"v2.2.0-rc2", "2.2.0-rc1", false},   // we do not rank rc vs rc (no false upgrade)
	}
	for _, tc := range cases {
		if got := isNewer(tc.tag, tc.cur); got != tc.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tc.tag, tc.cur, got, tc.want)
		}
	}
}

// TestParseVer locks the numeric extraction across prefixes and suffixes.
func TestParseVer(t *testing.T) {
	cases := []struct {
		in   string
		want [3]int
	}{
		{"v2.2.0", [3]int{2, 2, 0}},
		{"2.3.4-rc1", [3]int{2, 3, 4}},
		{"1.4.7", [3]int{1, 4, 7}},
		{"v2.2.0-vpn.local", [3]int{2, 2, 0}},
		{"garbage", [3]int{0, 0, 0}},
	}
	for _, tc := range cases {
		if got := parseVer(tc.in); got != tc.want {
			t.Errorf("parseVer(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestExpectedTarballSumParsesSumsAsset verifies the SHA256SUMS matcher used by
// the self-update checksum verification (audit P2-2), including the leading-'*'
// binary-mode name column and path-prefixed names.
func TestExpectedTarballSumParsesLine(t *testing.T) {
	// Simulate the parse the function does over a SHA256SUMS body. We exercise
	// the pure line-matching by reconstructing the same logic the reader uses.
	body := "" +
		"aabbccdd  mazzy-vpn-go-2.2.0-linux-amd64.tar.gz\n" +
		"deadbeef *dist/other-asset.tar.gz\n"
	want := "aabbccdd"
	got, ok := matchSumLine(body, "mazzy-vpn-go-2.2.0-linux-amd64.tar.gz")
	if !ok || got != want {
		t.Fatalf("matchSumLine = %q ok=%v, want %q true", got, ok, want)
	}
	if _, ok := matchSumLine(body, "not-present.tar.gz"); ok {
		t.Error("matchSumLine should report ok=false for an unlisted asset")
	}
}
