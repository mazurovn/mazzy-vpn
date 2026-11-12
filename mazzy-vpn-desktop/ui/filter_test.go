// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package ui

import (
	"testing"

	"github.com/mazurovn/mazzy-vpn/desktop/panel"
)

func TestMatchingFiltersByName(t *testing.T) {
	ps := []panel.Profile{
		{Name: "AustriaGraz", Protocol: "AmneziaWG"},
		{Name: "Berlin", Protocol: "WireGuard"},
		{Name: "Vienna", Protocol: "AmneziaWG"},
	}
	got := matching(ps, "vi")
	if len(got) != 1 || ps[got[0]].Name != "Vienna" {
		t.Fatalf("%v", got)
	}
	if len(matching(ps, "")) != 3 {
		t.Fatal("empty query")
	}
}
