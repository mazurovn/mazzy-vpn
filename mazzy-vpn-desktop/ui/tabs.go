// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package ui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/mazurovn/mazzy-vpn/desktop/menu"
	"github.com/mazurovn/mazzy-vpn/desktop/panel"
)

func (a *App) menuCall(snap panel.Snapshot, i int) menu.Call {
	profile := ""
	if p, ok := snap.SelectedProfile(); ok {
		profile = p.Name
	}
	catalog := ""
	if a.sess != nil {
		catalog = a.sess.CatalogDir()
	}
	return a.cmds[i].Run(menu.Context{
		Profile:   profile,
		Catalog:   catalog,
		Path:      a.pathEdit.Text(),
		Confirmed: a.armed == a.cmds[i].Label,
	})
}

// menuGroups is the command menu. Labels match menu.Actions and stay Russian
// with that list. Each group is a submenu.
var menuGroups = []struct {
	title  string
	labels []string
}{
	{"Связь", []string{"Связь", "Быстро", "Выбрать зону", "Переподключить", "Отключить", "В фоне"}},
	{"Восстановление", []string{"Восстановить", "Сброс"}},
	{"Сеть", []string{"Диагностика", "Зонд", "Серверы", "Лучшая", "Адаптеры", "Сеть", "Разбор", "Путь", "DNS"}},
	{"Защита", []string{"Скрытность", "Мимик", "Поставщики"}},
	{"Каталог", []string{"Профили", "Импорт", "Список", "Аудит", "Избранное", "Снять профиль"}},
	{"Система", []string{"Языки", "Система", "Врач", "Обновление", "Состояние"}},
}

type cmdMenu struct {
	Title string
	Index []int
}

func (a *App) ensureMenu(n int) {
	for len(a.menuBtn) < n {
		a.menuBtn = append(a.menuBtn, widget.Clickable{})
	}
}

func (a *App) menus() []cmdMenu {
	by := map[string]int{}
	for i := range a.cmds {
		by[a.cmds[i].Label] = i
	}
	out := make([]cmdMenu, 0, len(menuGroups))
	for _, g := range menuGroups {
		m := cmdMenu{Title: g.title}
		for _, label := range g.labels {
			if i, ok := by[label]; ok {
				m.Index = append(m.Index, i)
			}
		}
		out = append(out, m)
	}
	return out
}

func (a *App) groupFor(cmd int) int {
	for gi, g := range a.menus() {
		for _, i := range g.Index {
			if i == cmd {
				return gi
			}
		}
	}
	return 0
}

func (a *App) placeInMenu() {
	groups := a.menus()
	if len(groups) == 0 {
		return
	}
	if a.menuAt < 0 || a.menuAt >= len(groups) {
		a.menuAt = a.groupFor(a.cmdIndex)
	}
	in := false
	for _, i := range groups[a.menuAt].Index {
		if i == a.cmdIndex {
			in = true
		}
	}
	if !in && len(groups[a.menuAt].Index) > 0 {
		a.cmdIndex = groups[a.menuAt].Index[0]
	}
}

func (a *App) chrome(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = 0
	groups := a.menus()
	menuW := make([]int, len(groups))
	for i, g := range groups {
		menuW[i] = a.menuWidth(gtx, g.Title)
	}
	total := 0
	for _, w := range menuW {
		total += w
	}
	start := gtx.Constraints.Max.X - total
	if start < 0 {
		start = 0
	}
	if a.menuOpen >= 0 && a.menuOpen < len(menuW) {
		px := start
		for i := 0; i < a.menuOpen; i++ {
			px += menuW[i]
		}
		a.popupAt = image.Pt(px, gtx.Dp(36))
	}
	dims := layout.Flex{Alignment: layout.End}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = 0
			return a.tabTitle(gtx, 0, a.text.PanelTab)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = 0
			return a.tabTitle(gtx, 1, a.text.Journal)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(8))}
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			kids := make([]layout.FlexChild, len(groups))
			for i, g := range groups {
				i, title := i, g.Title
				kids[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = 0
					gtx.Constraints.Max.X = menuW[i]
					return a.menuTitle(gtx, i, title)
				})
			}
			return layout.Flex{}.Layout(gtx, kids...)
		}),
	)
	a.barH = dims.Size.Y
	return dims
}

func (a *App) menuWidth(gtx layout.Context, title string) int {
	gtx.Constraints.Min = image.Point{}
	macro := op.Record(gtx.Ops)
	w := material.Body1(a.theme, title).Layout(gtx).Size.X + gtx.Dp(20)
	macro.Stop()
	return w
}

func (a *App) tabTitle(gtx layout.Context, i int, title string) layout.Dimensions {
	return a.tabBtn[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Right: unit.Dp(18)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := material.Body1(a.theme, title)
			l.Color = colorMuted
			if a.tab == i {
				l.Color = colorText
			}
			dims := l.Layout(gtx)
			bar := color.NRGBA{}
			if a.tab == i {
				bar = colorAccent
			}
			h := gtx.Dp(3)
			defer clip.Rect{Min: image.Pt(0, dims.Size.Y+gtx.Dp(4)), Max: image.Pt(dims.Size.X, dims.Size.Y+gtx.Dp(4)+h)}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: bar}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			dims.Size.Y += gtx.Dp(4) + h
			return dims
		})
	})
}

func (a *App) menuTitle(gtx layout.Context, i int, title string) layout.Dimensions {
	return a.menuBtn[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := material.Body1(a.theme, title)
			l.Color = colorMuted
			if a.menuOpen == i {
				l.Color = colorAccent
			}
			return l.Layout(gtx)
		})
	})
}

func (a *App) menuContentWidth(gtx layout.Context, snap panel.Snapshot, avail int) int {
	groups := a.menus()
	width := gtx.Dp(280)
	limit := avail - gtx.Dp(12)
	if a.menuOpen < 0 || a.menuOpen >= len(groups) {
		return width
	}
	for _, i := range groups[a.menuOpen].Index {
		label := a.cmds[i].Label
		w := a.measureText(gtx, unit.Sp(14), label) + gtx.Dp(32)
		if sc := itemShortcut[label]; sc != "" {
			w += gtx.Dp(16) + a.measureText(gtx, unit.Sp(12), sc)
		}
		if w > width {
			width = w
		}
	}
	floor := gtx.Dp(380)
	if limit > 0 && floor > limit {
		floor = limit
	}
	if width < floor {
		width = floor
	}
	if limit > 0 && width > limit {
		width = limit
	}
	return width
}

func (a *App) measureText(gtx layout.Context, size unit.Sp, s string) int {
	if s == "" {
		return 0
	}
	gtx.Constraints.Min = image.Point{}
	gtx.Constraints.Max.X = 1 << 20
	gtx.Constraints.Max.Y = 1 << 20
	macro := op.Record(gtx.Ops)
	w := material.Label(a.theme, size, s).Layout(gtx).Size.X
	macro.Stop()
	return w
}

func (a *App) menuHeight(gtx layout.Context, snap panel.Snapshot, width int) int {
	mg := gtx
	mg.Constraints.Min = image.Point{X: width}
	mg.Constraints.Max = image.Point{X: width, Y: 1 << 20}
	macro := op.Record(gtx.Ops)
	h := a.menuBody(mg, snap, false).Size.Y
	macro.Stop()
	groups := a.menus()
	if a.menuOpen >= 0 && a.menuOpen < len(groups) && groups[a.menuOpen].Title == "Каталог" {
		h += gtx.Dp(48)
	}
	return h
}

func (a *App) paintMenu(gtx layout.Context, snap panel.Snapshot, width int) layout.Dimensions {
	mg := gtx
	mg.Constraints.Min = image.Point{X: width}
	mg.Constraints.Max = image.Point{X: width, Y: 1 << 20}
	macro := op.Record(gtx.Ops)
	dims := a.menuBody(mg, snap, true)
	call := macro.Stop()
	r := gtx.Dp(10)
	defer clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NE: r, NW: r, SE: r, SW: r}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: colorMenu}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	call.Add(gtx.Ops)
	return dims
}

func (a *App) menuBody(gtx layout.Context, snap panel.Snapshot, live bool) layout.Dimensions {
	groups := a.menus()
	if a.menuOpen < 0 || a.menuOpen >= len(groups) {
		return layout.Dimensions{}
	}
	open := groups[a.menuOpen]
	return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		kids := make([]layout.FlexChild, 0, len(open.Index)+2)
		title := open.Title
		kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.H6(a.theme, title)
			l.Color = colorText
			return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, l.Layout)
		}))
		for _, idx := range open.Index {
			idx := idx
			kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.menuRow(gtx, snap, idx, live)
			}))
		}
		if live && open.Title == "Каталог" {
			kids = append(kids, layout.Rigid(a.pathField))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, kids...)
	})
}

func (a *App) menuRow(gtx layout.Context, snap panel.Snapshot, i int, live bool) layout.Dimensions {
	draw := func(gtx layout.Context) layout.Dimensions {
		return a.menuRowVisual(gtx, snap, i)
	}
	if !live {
		return draw(gtx)
	}
	return a.actBtn[i].Layout(gtx, draw)
}

func (a *App) menuRowVisual(gtx layout.Context, snap panel.Snapshot, i int) layout.Dimensions {
	label := a.cmds[i].Label
	call := a.menuCall(snap, i)
	reason := ""
	if call.Disabled != "" && !call.Confirm {
		reason = call.Disabled
	}
	if a.armed == label {
		reason = call.Disabled
	}
	shortcut := itemShortcut[label]
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = 0
		name := material.Body2(a.theme, label)
		name.Color = colorText
		name.MaxLines = 1
		if reason != "" && a.armed != label {
			name.Color = colorMuted
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(name.Layout),
					layout.Flexed(1, layout.Spacer{Width: unit.Dp(16)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if shortcut == "" {
							return layout.Dimensions{}
						}
						l := material.Caption(a.theme, shortcut)
						l.Color = colorMuted
						l.MaxLines = 1
						return l.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if reason == "" {
					return layout.Dimensions{}
				}
				c := material.Caption(a.theme, reason)
				c.Color = colorMuted
				c.MaxLines = 0
				return c.Layout(gtx)
			}),
		)
	})
	callOp := macro.Stop()
	if a.cmdFocus && i == a.cmdIndex {
		defer clip.Rect{Max: dims.Size}.Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, colorSelect)
	}
	callOp.Add(gtx.Ops)
	return dims
}

var itemShortcut = map[string]string{
	"Связь":          "Enter",
	"Переподключить": "Ctrl+R",
	"Отключить":      "Ctrl+D",
}
