// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package ui

import (
	"testing"

	"github.com/mazurovn/mazzy-vpn/desktop/menu"
	"github.com/mazurovn/mazzy-vpn/desktop/monitor"
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

func TestVisibleStateSkipsUnknown(t *testing.T) {
	view := monitor.FromLive(monitor.Live{})
	got := visibleState(view, "Связь", "Рукопожатие", "Путь", "Внешний IP", "Чистота", "Antigravity")
	if len(got) != 0 {
		t.Fatalf("placeholders shown: %+v", got)
	}
	view.Connection = monitor.Red
	view.ConnectionText = "туннель не поднят"
	got = visibleState(view, "Связь", "Рукопожатие", "Путь", "Внешний IP", "Чистота", "Antigravity")
	if len(got) != 1 || got[0].Title != "Связь" || got[0].Light != monitor.Red {
		t.Fatalf("%+v", got)
	}
}

func TestMenuGroupsCoverEveryCommand(t *testing.T) {
	seen := map[string]int{}
	for _, g := range menuGroups {
		if g.title == "" || len(g.labels) == 0 {
			t.Fatalf("empty group %+v", g)
		}
		for _, label := range g.labels {
			seen[label]++
		}
	}
	actions := menu.Actions()
	if len(seen) != len(actions) {
		t.Fatalf("groups %d actions %d", len(seen), len(actions))
	}
	for _, action := range actions {
		if seen[action.Label] != 1 {
			t.Fatalf("%s covered %d times", action.Label, seen[action.Label])
		}
	}
	ru := Phrases()
	if ru.PanelTab != "Панель" || ru.Commands != "Команды" || ru.Journal != "Журнал" {
		t.Fatalf("%+v", ru)
	}
}
