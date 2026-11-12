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

func (a *App) popup(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	groups := a.menus()
	if a.menuOpen < 0 || a.menuOpen >= len(groups) {
		return layout.Dimensions{}
	}
	open := groups[a.menuOpen]
	gtx.Constraints.Min.Y = 0
	return card(gtx, func(gtx layout.Context) layout.Dimensions {
		kids := make([]layout.FlexChild, 0, len(open.Index)+2)
		kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.H6(a.theme, open.Title)
			l.Color = colorText
			return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, l.Layout)
		}))
		for _, idx := range open.Index {
			idx := idx
			kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.popupItem(gtx, snap, idx)
			}))
		}
		if open.Title == "Каталог" {
			kids = append(kids, layout.Rigid(a.pathField))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, kids...)
	})
}

func (a *App) popupItem(gtx layout.Context, snap panel.Snapshot, i int) layout.Dimensions {
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
	return a.actBtn[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			name := material.Body2(a.theme, label)
			name.Color = colorText
			if reason != "" && a.armed != label {
				name.Color = colorMuted
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, name.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if shortcut == "" {
								return layout.Dimensions{}
							}
							l := material.Caption(a.theme, shortcut)
							l.Color = colorMuted
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
		call := macro.Stop()
		if a.cmdFocus && i == a.cmdIndex {
			defer clip.Rect{Max: dims.Size}.Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, colorSelect)
		}
		call.Add(gtx.Ops)
		return dims
	})
}

var itemShortcut = map[string]string{
	"Связь":          "Enter",
	"Переподключить": "Ctrl+R",
	"Отключить":      "Ctrl+D",
}
