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
			if a.cmdFocus {
				continue
			}
			a.query += e.Text
			a.keepSelection(snap)
		}
	}
}

func (a *App) onKey(snap panel.Snapshot, e key.Event) {
	switch e.Name {
	case key.NameEscape:
		if a.cmdFocus {
			a.cmdFocus = false
			a.refocus = true
			return
		}
		a.query = ""
		a.keepSelection(snap)
	case key.NameLeftArrow:
		a.moveCmd(snap, -1, true)
	case key.NameRightArrow:
		a.moveCmd(snap, 1, true)
	case key.NameUpArrow:
		a.move(snap, -1)
	case key.NameDownArrow:
		a.move(snap, 1)
	case key.NameReturn, key.NameEnter:
		if a.cmdFocus {
			a.activate(snap, a.cmdIndex)
			return
		}
		a.model.Connect(context.Background())
	case key.NameDeleteBackward:
		if a.cmdFocus || a.query == "" {
			return
		}
		_, size := utf8.DecodeLastRuneInString(a.query)
		a.query = a.query[:len(a.query)-size]
		a.keepSelection(snap)
	}
}

func (a *App) moveCmd(snap panel.Snapshot, delta int, horizontal bool) {
	if !a.cmdFocus {
		if delta > 0 {
			a.cmdFocus = true
		}
		return
	}
	if len(a.cmds) == 0 {
		return
	}
	cols := a.cmdCols
	if cols < 1 {
		cols = 1
	}
	if horizontal && delta < 0 && a.cmdIndex%cols == 0 {
		a.cmdFocus = false
		return
	}
	step := delta
	if !horizontal {
		step = delta * cols
	}
	next := a.cmdIndex + step
	if next < 0 {
		next = 0
	}
	if next >= len(a.cmds) {
		next = len(a.cmds) - 1
	}
	a.cmdIndex = next
	_ = snap
}

func (a *App) move(snap panel.Snapshot, delta int) {
	if a.cmdFocus {
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
