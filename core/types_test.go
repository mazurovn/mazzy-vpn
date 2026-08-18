// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package core

import "testing"

func TestManagedInterfacesCoversAllProtocols(t *testing.T) {
	got := ManagedInterfaces()
	// Must include each protocol's deterministic interface name, and match
	// Interface() exactly (single source of truth).
	want := map[string]bool{
		AmneziaWG.Interface(): false,
		WireGuard.Interface(): false,
		OpenVPN.Interface():   false,
	}
	if len(got) != len(want) {
		t.Fatalf("ManagedInterfaces() = %v, want %d entries", got, len(want))
	}
	for _, n := range got {
		if _, ok := want[n]; !ok {
			t.Errorf("unexpected interface %q", n)
		}
		want[n] = true
	}
	for n, seen := range want {
		if !seen {
			t.Errorf("missing interface %q", n)
		}
	}
}
