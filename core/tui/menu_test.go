// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package tui

import (
	"strings"
	"testing"

	"github.com/mazurovn/mazzy-vpn/core/i18n"
)

func TestMenuLinesLocalized(t *testing.T) {
	m := NewMenu(i18n.RU)
	lines := m.Lines()
	if len(lines) != m.Len() {
		t.Fatalf("lines=%d, items=%d", len(lines), m.Len())
	}
	if !strings.HasPrefix(lines[0], "1. ") {
		t.Errorf("first line not numbered: %q", lines[0])
	}
	// Russian label for connect.
	if !strings.Contains(lines[0], "Подключить") {
		t.Errorf("expected localized ru label, got %q", lines[0])
	}
}

func TestMenuSelectMapsToAction(t *testing.T) {
	m := NewMenu(i18n.EN)
	act, ok := m.Select(1)
	if !ok || act != ActConnect {
		t.Errorf("Select(1) = %q,%v; want connect", act, ok)
	}
	// Last item is quit.
	last, ok := m.Select(m.Len())
	if !ok || last != ActQuit {
		t.Errorf("last item should be quit, got %q", last)
	}
}

func TestMenuSelectOutOfRange(t *testing.T) {
	m := NewMenu(i18n.EN)
	for _, bad := range []int{0, -1, 999} {
		if _, ok := m.Select(bad); ok {
			t.Errorf("Select(%d) should be out of range", bad)
		}
	}
}

func TestMenuLanguageSwitch(t *testing.T) {
	m := NewMenu(i18n.EN)
	enFirst := m.Lines()[0]
	m.SetLanguage(i18n.DE)
	deFirst := m.Lines()[0]
	if enFirst == deFirst {
		t.Errorf("language switch did not change labels: %q == %q", enFirst, deFirst)
	}
	if !strings.Contains(deFirst, "Verbinden") {
		t.Errorf("expected German label after switch, got %q", deFirst)
	}
}

func TestMenuTitleAndPrompt(t *testing.T) {
	m := NewMenu(i18n.EN)
	if m.Title() == "" || m.Prompt() == "" {
		t.Error("title/prompt must be non-empty")
	}
}

// TestItoa checks the small integer formatter used for menu numbering.
func TestItoa(t *testing.T) {
	cases := map[int]string{0: "0", 1: "1", 7: "7", 10: "10", 123: "123"}
	for in, want := range cases {
		if got := itoa(in); got != want {
			t.Errorf("itoa(%d) = %q, want %q", in, got, want)
		}
	}
}
