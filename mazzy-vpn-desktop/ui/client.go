// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package ui

import (
	"image"
	"strings"
	"unicode/utf8"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/mazurovn/mazzy-vpn/desktop/monitor"
	"github.com/mazurovn/mazzy-vpn/desktop/panel"
)

func (a *App) client(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	return a.page(gtx, snap)
}

func (a *App) page(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(14), Right: unit.Dp(14), Top: unit.Dp(8), Bottom: unit.Dp(10)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.chrome(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					if a.tab == 1 {
						return a.journal(gtx, snap)
					}
					return layout.Flex{}.Layout(gtx,
						layout.Flexed(2, func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(420)
							return a.serverCard(gtx, snap)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							w := summaryWidth(gtx)
							gtx.Constraints.Min.X = w
							gtx.Constraints.Max.X = w
							gtx.Constraints.Min.Y = 0
							return a.dashboard(gtx, snap)
						}),
					)
				}),
			)
			if a.menuOpen >= 0 {
				a.overlay(gtx, snap, dims.Size)
			}
			return dims
		},
	)
}

// overlay draws the open submenu as a solid layer above the page.
// op.Defer paints it after the server list, so rows cannot show through.
func (a *App) overlay(gtx layout.Context, snap panel.Snapshot, size image.Point) {
	if a.barH < size.Y {
		off := op.Offset(image.Pt(0, a.barH)).Push(gtx.Ops)
		cover := image.Point{X: size.X, Y: size.Y - a.barH}
		st := clip.Rect{Max: cover}.Push(gtx.Ops)
		_ = a.dismiss.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: cover}
		})
		st.Pop()
		off.Pop()
	}
	// The menu hangs over the profile list and stops before the summary,
	// so the sentence stays whole while a menu is open.
	room := size.X - summaryWidth(gtx) - gtx.Dp(16)
	if room < gtx.Dp(280) {
		room = gtx.Dp(280)
	}
	width := a.menuContentWidth(gtx, snap, room)
	height := a.menuHeight(gtx, snap, width)
	right := size.X - summaryWidth(gtx) - gtx.Dp(12)
	x := a.popupAt.X
	if x+width > right {
		x = right - width
	}
	if x < 0 {
		x = 0
	}
	y := a.popupAt.Y
	if y+height > size.Y {
		y = size.Y - height
	}
	if y < 0 {
		y = 0
	}
	macro := op.Record(gtx.Ops)
	off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
	a.paintMenu(gtx, snap, width)
	off.Pop()
	op.Defer(gtx.Ops, macro.Stop())
}

func (a *App) serverCard(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	ids := matching(snap.Profiles, a.query)
	return card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				title := a.text.Profiles
				q := a.query
				qColor := colorText
				if q == "" {
					q = a.text.Search
					qColor = colorMuted
				} else {
					title += "  ·  " + q
					q = a.text.Filter + "  ·  ↑↓  Enter"
				}
				l := material.H6(a.theme, title)
				l.Color = colorText
				hint := material.Caption(a.theme, q)
				hint.Color = qColor
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(l.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(6)}.Layout(gtx, hint.Layout)
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(ids) == 0 {
					l := material.Body2(a.theme, a.text.Empty)
					l.Color = colorMuted
					return l.Layout(gtx)
				}
				return a.list.Layout(gtx, len(ids), func(gtx layout.Context, i int) layout.Dimensions {
					return a.serverRow(gtx, snap, ids[i])
				})
			}),
		)
	})
}

func (a *App) serverRow(gtx layout.Context, snap panel.Snapshot, i int) layout.Dimensions {
	p := snap.Profiles[i]
	return a.rows[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if i == snap.Selected {
			fill(gtx, colorSelect)
		}
		return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				port := ""
				if n := endpointPort(p.Endpoint); n != "" {
					port = "udp/" + n
				}
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(a.theme, unit.Sp(14), endEllipsis(p.Name, gtx.Constraints.Max.X, gtx.Sp(14)))
						l.Color = colorText
						l.MaxLines = 1
						l.Truncator = "…"
						return l.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(72)
						l := material.Label(a.theme, unit.Sp(13), port)
						l.Color = colorAccent
						l.MaxLines = 1
						return l.Layout(gtx)
					}),
				)
			},
		)
	})
}

type chip struct {
	Title string
	Text  string
	Light monitor.Light
}

// visibleState keeps a light only when it carries a real result.
// Unknown placeholders are not drawn.
func visibleState(view monitor.View, link, handshake, route, egress, clean, service string) []chip {
	var out []chip
	add := func(title, text string, light monitor.Light) {
		if light == monitor.Unknown || light == "" {
			return
		}
		out = append(out, chip{Title: title, Text: text, Light: light})
	}
	add(link, view.ConnectionText, view.Connection)
	add(handshake, view.HandshakeText, view.Handshake)
	add(route, view.PathText, view.Path)
	add(egress, view.EgressText, view.EgressLight)
	add(clean, view.CleanText, view.Clean)
	add(service, view.ServiceText, view.Service)
	return out
}

func summaryWidth(gtx layout.Context) int {
	return gtx.Dp(560)
}

func (a *App) dashboard(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	view := a.board()
	strip := a.compact(view)
	points := measuredPoints(view.Points)
	gtx.Constraints.Min.Y = 0
	if len(strip) == 0 && len(points) == 0 {
		return a.sentence(gtx, a.dashLine(snap))
	}
	var kids []layout.FlexChild
	if len(strip) > 0 {
		kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.strip(gtx, strip)
		}))
	}
	if len(points) > 0 {
		kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.bars(gtx, points)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, kids...)
}

// sentence wraps on spaces and keeps a path on one line.
func (a *App) sentence(gtx layout.Context, s string) layout.Dimensions {
	width := gtx.Constraints.Max.X
	if width <= 0 {
		width = summaryWidth(gtx)
	}
	lines := wrapOnSpaces(s, func(part string) int {
		return a.measureText(gtx, unit.Sp(16), part)
	}, width)
	kids := make([]layout.FlexChild, len(lines))
	for i, line := range lines {
		line := line
		kids[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.Body1(a.theme, line)
			l.Color = colorText
			l.MaxLines = 1
			l.WrapPolicy = text.WrapWords
			return l.Layout(gtx)
		})
	}
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx, kids...)
	if dims.Size.X < width {
		dims.Size.X = width
	}
	return dims
}

func wrapOnSpaces(s string, widthOf func(string) int, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{s}
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		next := cur + " " + w
		if widthOf(next) <= width {
			cur = next
			continue
		}
		lines = append(lines, cur)
		cur = w
	}
	lines = append(lines, cur)
	return lines
}

func (a *App) dashLine(snap panel.Snapshot) string {
	if snap.Phase == panel.PhaseError && snap.Err != "" {
		return snap.Err
	}
	if snap.Notice != "" && snap.Phase != panel.PhaseConnected {
		return snap.Notice
	}
	if snap.Phase == panel.PhaseConnected {
		return a.text.DashUp
	}
	return a.text.DashDown
}

func (a *App) strip(gtx layout.Context, chips []chip) layout.Dimensions {
	kids := make([]layout.FlexChild, 0, len(chips))
	for _, c := range chips {
		c := c
		kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return a.chip(gtx, c)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, kids...)
}

func (a *App) stepLine(snap panel.Snapshot) string {
	if snap.Phase == panel.PhaseError && snap.Err != "" {
		return snap.Err
	}
	if snap.Notice != "" && snap.Phase != panel.PhaseConnected {
		return snap.Notice
	}
	switch snap.Phase {
	case panel.PhaseConnecting:
		return a.text.Busy
	case panel.PhaseConnected:
		return a.text.UpNote
	case panel.PhaseError:
		return a.text.Fail
	default:
		return a.text.Ready
	}
}

func (a *App) compact(view monitor.View) []chip {
	var out []chip
	if view.Connection == monitor.Green || (view.Connection == monitor.Yellow && view.ConnectionText != "запрошено включение, интерфейса нет") {
		out = append(out, chip{Title: a.text.Link, Text: view.ConnectionText, Light: view.Connection})
	}
	if view.EgressLight != monitor.Unknown && view.EgressLight != "" {
		out = append(out, chip{Title: a.text.Exit, Text: view.EgressText, Light: view.EgressLight})
	}
	if view.Egress != "" {
		out = append(out, chip{Title: a.text.EgressIP, Text: view.Egress, Light: monitor.Green})
	}
	if view.Clean != monitor.Unknown && view.Clean != "" {
		out = append(out, chip{Title: a.text.Clean, Text: view.CleanText, Light: view.Clean})
	}
	return out
}

func (a *App) journal(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	if len(a.logLines()) == 0 {
		l := material.Body1(a.theme, a.text.LogHint)
		l.Color = colorMuted
		return l.Layout(gtx)
	}
	return a.logCard(gtx, snap)
}

func endEllipsis(s string, width, em int) string {
	if s == "" || width <= 0 || em <= 0 {
		return s
	}
	max := width * 10 / (em * 6)
	if max < 4 {
		max = 4
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max-1]) + "…"
}

func extraLights(chips []chip, linkTitle string) []chip {
	out := chips[:0:0]
	for _, c := range chips {
		if c.Title == linkTitle {
			continue
		}
		out = append(out, c)
	}
	return out
}

func (a *App) pathField(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		ed := material.Editor(a.theme, &a.pathEdit, a.text.ImportHint)
		ed.Color = colorText
		ed.HintColor = colorMuted
		ed.TextSize = unit.Sp(13)
		return ed.Layout(gtx)
	})
}

func (a *App) stateStrip(gtx layout.Context) layout.Dimensions {
	view := a.board()
	chips := visibleState(view, a.text.Link, a.text.Handshake, a.text.Route, a.text.EgressIP, a.text.Clean, a.text.Service)
	if len(chips) == 0 && !hasMeasurement(view.Points) {
		return layout.Dimensions{}
	}
	return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return card(gtx, func(gtx layout.Context) layout.Dimensions {
			kids := make([]layout.FlexChild, 0, len(chips)+1)
			for _, c := range chips {
				c := c
				kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.chip(gtx, c)
				}))
			}
			if hasMeasurement(view.Points) {
				kids = append(kids, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.bars(gtx, view.Points)
				}))
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx, kids...)
		})
	})
}

func (a *App) chip(gtx layout.Context, c chip) layout.Dimensions {
	text := c.Text
	if text == "" {
		text = a.text.Unknown
	}
	return layout.Inset{Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.dot(gtx, lightColor(c.Light))
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(a.theme, unit.Sp(13), c.Title+"  "+text)
				l.Color = colorText
				l.MaxLines = 1
				return l.Layout(gtx)
			}),
		)
	})
}
