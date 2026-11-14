// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package wireguard

import (
	"strings"
	"testing"
	"time"

	"github.com/mazurovn/mazzy-vpn/core/profile"
)

func TestParseHandshake(t *testing.T) {
	const blob = "" +
		"public_key=aa\n" +
		"last_handshake_time_sec=0\n" +
		"last_handshake_time_nsec=0\n" +
		"public_key=bb\n" +
		"last_handshake_time_sec=1700000000\n" +
		"last_handshake_time_nsec=5\n" +
		"public_key=cc\n" +
		"last_handshake_time_sec=9\n"
	if got := ParseHandshake(blob, "aa"); !got.IsZero() {
		t.Fatalf("zero timestamp should be no handshake, got %v", got)
	}
	got := ParseHandshake(blob, "bb")
	want := time.Unix(1700000000, 5)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if got := ParseHandshake(blob, "missing"); !got.IsZero() {
		t.Fatalf("missing peer: %v", got)
	}
}

func TestRetargetUAPIForcesKeepalive(t *testing.T) {
	u := profile.RetargetUAPI(strings.Repeat("ab", 32), "203.0.113.10:443", 0)
	for _, want := range []string{
		"update_only=true",
		"endpoint=203.0.113.10:443",
		"persistent_keepalive_interval=0",
		"persistent_keepalive_interval=25",
	} {
		if !strings.Contains(u, want) {
			t.Errorf("missing %q in %s", want, u)
		}
	}
	if strings.Index(u, "persistent_keepalive_interval=0") > strings.Index(u, "persistent_keepalive_interval=25") {
		t.Fatal("keepalive must pass through zero before the non-zero value")
	}
}
