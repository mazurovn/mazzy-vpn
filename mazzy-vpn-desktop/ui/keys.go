// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package ui

import (
	"context"
	"strings"
	"unicode/utf8"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"

	"github.com/mazurovn/mazzy-vpn/desktop/menu"
	"github.com/mazurovn/mazzy-vpn/desktop/panel"
)

func (a *App) keys(gtx layout.Context, snap panel.Snapshot) {
	event.Op(gtx.Ops, a)
	if a.refocus {
		gtx.Execute(key.FocusCmd{Tag: a})
		a.refocus = false
	}
	for {
		ev, ok := gtx.Event(key.Filter{Focus: a}, key.FocusFilter{Target: a})
		if !ok {
			return
		}
		switch e := ev.(type) {
		case key.Event:
			if e.State != key.Press {
				continue
			}
			a.onKey(snap, e)
		case key.EditEvent:
			if a.menuOpen >= 0 {
				continue
			}
			a.query += e.Text
			a.keepSelection(snap)
		}
	}
}

func (a *App) onKey(snap panel.Snapshot, e key.Event) {
	if e.Modifiers.Contain(key.ModCtrl) && e.Name == "R" {
		a.reconnect()
		return
	}
	if e.Modifiers.Contain(key.ModCtrl) && e.Name == "D" {
		a.model.Disconnect(context.Background())
		return
	}
	if e.Name == key.NameF5 {
		a.refreshNow()
		return
	}
	switch e.Name {
	case key.NameEscape:
		if a.menuOpen >= 0 {
			a.armed = ""
			a.menuOpen = -1
			a.cmdFocus = false
			a.refocus = true
			return
		}
		a.query = ""
		a.keepSelection(snap)
	case key.NameLeftArrow:
		if a.menuOpen >= 0 {
			a.moveCmd(snap, -1, true)
		}
	case key.NameRightArrow:
		if a.menuOpen >= 0 {
			a.moveCmd(snap, 1, true)
		}
	case key.NameUpArrow:
		a.move(snap, -1)
	case key.NameDownArrow:
		a.move(snap, 1)
	case key.NameReturn, key.NameEnter:
		if a.menuOpen >= 0 && a.cmdFocus {
			a.runCommand(snap, a.cmdIndex)
			return
		}
		a.model.Connect(context.Background())
	case key.NameDeleteBackward:
		if a.menuOpen >= 0 || a.query == "" {
			return
		}
		_, size := utf8.DecodeLastRuneInString(a.query)
		a.query = a.query[:len(a.query)-size]
		a.keepSelection(snap)
	}
}

func (a *App) reconnect() {
	if a.sess == nil {
		return
	}
	go a.sess.Do(context.Background(), menu.Call{Label: "Переподключить", Lifecycle: "reconnect"})
}

func (a *App) refreshNow() {
	a.model.Reload(context.Background())
	if a.sess != nil {
		go a.sess.PollChecks(context.Background())
	}
}

func (a *App) moveCmd(snap panel.Snapshot, delta int, horizontal bool) {
	if a.menuOpen < 0 {
		return
	}
	groups := a.menus()
	if len(groups) == 0 {
		return
	}
	if a.menuAt < 0 || a.menuAt >= len(groups) {
		a.menuAt = 0
	}
	if horizontal {
		a.menuAt += delta
		if a.menuAt < 0 {
			a.menuAt = 0
		}
		if a.menuAt >= len(groups) {
			a.menuAt = len(groups) - 1
		}
		a.menuOpen = a.menuAt
		if idxs := groups[a.menuAt].Index; len(idxs) > 0 {
			a.cmdIndex = idxs[0]
		}
		return
	}
	idxs := groups[a.menuAt].Index
	pos := 0
	for i, id := range idxs {
		if id == a.cmdIndex {
			pos = i
		}
	}
	pos += delta
	if pos < 0 {
		pos = 0
	}
	if pos >= len(idxs) {
		pos = len(idxs) - 1
	}
	if len(idxs) > 0 {
		a.cmdIndex = idxs[pos]
	}
	_ = snap
}

func (a *App) move(snap panel.Snapshot, delta int) {
	if a.menuOpen >= 0 {
		a.moveCmd(snap, delta, false)
		return
	}
	ids := matching(snap.Profiles, a.query)
	if len(ids) == 0 {
		return
	}
	at := 0
	for i, id := range ids {
		if id == snap.Selected {
			at = i
		}
	}
	at += delta
	if at < 0 {
		at = 0
	}
	if at >= len(ids) {
		at = len(ids) - 1
	}
	a.model.Select(ids[at])
	a.list.Position.First = at
}

func (a *App) keepSelection(snap panel.Snapshot) {
	ids := matching(snap.Profiles, a.query)
	if len(ids) == 0 {
		return
	}
	for _, id := range ids {
		if id == snap.Selected {
			return
		}
	}
	a.model.Select(ids[0])
}

func matching(profiles []panel.Profile, query string) []int {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]int, 0, len(profiles))
	for i, p := range profiles {
		if q == "" || strings.Contains(strings.ToLower(p.Name), q) || strings.Contains(strings.ToLower(p.Protocol), q) {
			out = append(out, i)
		}
	}
	return out
}
