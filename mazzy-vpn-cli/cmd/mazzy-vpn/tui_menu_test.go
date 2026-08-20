// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mazurovn/mazzy-vpn/core/runstatus"
)

func runeKey(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func TestDefaultTUIReachesProfilesDiagnosticsLanguageAndHelp(t *testing.T) {
	t.Setenv("MAZZY_CONFIG_DIR", t.TempDir())
	m := newTUIModel()

	model, _ := m.handleKey(runeKey('p'))
	m = model.(tuiModel)
	if m.scr != scrProfiles {
		t.Fatalf("p screen=%v, want profiles", m.scr)
	}
	model, _ = m.keyProfiles(runeKey('i'))
	m = model.(tuiModel)
	if m.scr != scrImport {
		t.Fatalf("i screen=%v, want import", m.scr)
	}
	model, _ = m.keyImport(tea.KeyMsg{Type: tea.KeyEsc})
	m = model.(tuiModel)
	if m.scr != scrProfiles {
		t.Fatalf("esc screen=%v, want profiles", m.scr)
	}

	m.scr = scrMain
	model, _ = m.handleKey(runeKey('x'))
	m = model.(tuiModel)
	if m.scr != scrDiagnostics {
		t.Fatalf("x screen=%v, want diagnostics", m.scr)
	}

	m.scr = scrMain
	model, _ = m.handleKey(runeKey('s'))
	m = model.(tuiModel)
	model, _ = m.keySettings(runeKey('7'))
	m = model.(tuiModel)
	if m.scr != scrLanguage {
		t.Fatalf("settings 7 screen=%v, want language", m.scr)
	}

	m.scr = scrMain
	model, _ = m.handleKey(runeKey('?'))
	m = model.(tuiModel)
	if m.scr != scrHelp {
		t.Fatalf("? screen=%v, want help", m.scr)
	}
}

func TestTUIRecoverRequiresConfirmation(t *testing.T) {
	m := newTUIModel()
	model, cmd := m.handleKey(runeKey('r'))
	m = model.(tuiModel)
	if cmd != nil {
		t.Fatal("recover shortcut must not execute before confirmation")
	}
	if m.scr != scrRemoveConfirm || m.pendingDelete != "__RECOVER__" {
		t.Fatalf("recover did not open confirmation: %+v", m)
	}
}

func TestStatusPaneShowsDerivedMetrics(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAZZY_RUN_DIR", dir)
	w := runstatus.NewWriter("Amsterdam-01", "vpn0", "AmneziaWG", true)
	w.SetState(runstatus.StateProtected, "vpn0", "203.0.113.42")
	w.Tick(20, true)
	w.Tick(40, true)
	w.Tick(0, false)
	w.Error("temporary egress failure")
	defer w.Close()

	pane := newTUIModel().statusPane()
	for _, want := range []string{"Status & errors", "33.3% loss", "p50", "p95", "jitter", "temporary egress failure"} {
		if !strings.Contains(pane, want) {
			t.Errorf("status pane missing %q:\n%s", want, pane)
		}
	}
}
