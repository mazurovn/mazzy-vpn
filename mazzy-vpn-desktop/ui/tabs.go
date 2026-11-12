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

	"github.com/mazurovn/mazzy-vpn/desktop/panel"
)

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

func (a *App) tabBar(gtx layout.Context) layout.Dimensions {
	titles := []string{a.text.PanelTab, a.text.Commands, a.text.Journal}
	kids := make([]layout.FlexChild, 0, len(titles))
	for i, title := range titles {
		i, title := i, title
		kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.tabBtn[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Right: unit.Dp(22)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Body1(a.theme, title)
					l.Color = colorMuted
					if a.tab == i {
						l.Color = colorText
					}
					dims := l.Layout(gtx)
					h := gtx.Dp(3)
					bar := color.NRGBA{}
					if a.tab == i {
						bar = colorAccent
					}
					defer clip.Rect{Min: image.Pt(0, dims.Size.Y+gtx.Dp(4)), Max: image.Pt(dims.Size.X, dims.Size.Y+gtx.Dp(4)+h)}.Push(gtx.Ops).Pop()
					paint.ColorOp{Color: bar}.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)
					dims.Size.Y += gtx.Dp(4) + h
					return dims
				})
			})
		}))
	}
	return layout.Flex{Alignment: layout.End}.Layout(gtx, kids...)
}

func (a *App) menuPane(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	groups := a.menus()
	if len(groups) == 0 {
		return layout.Dimensions{}
	}
	if a.menuAt < 0 || a.menuAt >= len(groups) {
		a.menuAt = 0
	}
	open := groups[a.menuAt]
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(196)
			gtx.Constraints.Max.X = gtx.Dp(210)
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			return card(gtx, func(gtx layout.Context) layout.Dimensions {
				kids := make([]layout.FlexChild, 0, len(groups))
				for i, g := range groups {
					i, g := i, g
					kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.menuBtn[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							macro := op.Record(gtx.Ops)
							dims := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(10), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								l := material.Body1(a.theme, g.Title)
								l.Color = colorMuted
								if i == a.menuAt {
									l.Color = colorAccent
								}
								return l.Layout(gtx)
							})
							call := macro.Stop()
							if i == a.menuAt {
								defer clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(8)).Push(gtx.Ops).Pop()
								paint.Fill(gtx.Ops, colorSelect)
							}
							call.Add(gtx.Ops)
							return dims
						})
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, kids...)
			})
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			return card(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.H6(a.theme, open.Title)
						l.Color = colorText
						return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, l.Layout)
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Push(gtx.Ops).Pop()
						return a.actList.Layout(gtx, len(open.Index), func(gtx layout.Context, i int) layout.Dimensions {
							return a.actionRow(gtx, snap, open.Index[i])
						})
					}),
					layout.Rigid(a.pathField),
				)
			})
		}),
	)
}
