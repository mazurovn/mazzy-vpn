// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package ui

import (
	"context"
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/mazurovn/mazzy-vpn/desktop/menu"
	"github.com/mazurovn/mazzy-vpn/desktop/monitor"
	"github.com/mazurovn/mazzy-vpn/desktop/panel"
)

func (a *App) board() monitor.View {
	if a.sess == nil {
		return monitor.FromLive(monitor.Live{})
	}
	return a.sess.Board()
}

func (a *App) activate(snap panel.Snapshot, i int) {
	if a.sess == nil || i < 0 || i >= len(a.cmds) {
		return
	}
	label := a.cmds[i].Label
	profile := ""
	if p, ok := snap.SelectedProfile(); ok {
		profile = p.Name
	}
	call := a.cmds[i].Run(menu.Context{
		Profile:   profile,
		Catalog:   a.sess.CatalogDir(),
		Path:      a.pathEdit.Text(),
		Confirmed: a.armed == label,
	})
	if call.Disabled != "" {
		if call.Confirm {
			a.armed = label
		} else {
			a.armed = ""
		}
		a.sess.Note(call.Label + " — " + call.Disabled)
		return
	}
	a.armed = ""
	go a.sess.Do(context.Background(), call)
}

// runCommand starts a menu action and opens the journal when there is
// something to read. A confirm stays on the command tab for the second press.
func (a *App) runCommand(snap panel.Snapshot, i int) {
	a.activate(snap, i)
	if a.armed != "" {
		a.tab = 1
		a.cmdFocus = true
		return
	}
	a.tab = 2
	a.cmdFocus = false
}

func (a *App) actionCard(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	return card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.H6(a.theme, a.text.Commands)
				l.Color = colorText
				return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, l.Layout)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				defer clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}.Push(gtx.Ops).Pop()
				return a.actList.Layout(gtx, len(a.cmds), func(gtx layout.Context, i int) layout.Dimensions {
					return a.actionRow(gtx, snap, i)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					// Paint the field so the scrolling list cannot show through it.
					macro := op.Record(gtx.Ops)
					ed := material.Editor(a.theme, &a.pathEdit, a.text.PathHint)
					ed.Color = colorText
					ed.HintColor = colorMuted
					ed.TextSize = unit.Sp(13)
					dims := layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(2), Left: unit.Dp(4)}.Layout(gtx, ed.Layout)
					call := macro.Stop()
					sz := image.Pt(gtx.Constraints.Max.X, dims.Size.Y)
					defer clip.Rect{Max: sz}.Push(gtx.Ops).Pop()
					paint.ColorOp{Color: colorPanel}.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)
					call.Add(gtx.Ops)
					return layout.Dimensions{Size: sz}
				})
			}),
		)
	})
}

func (a *App) actionRow(gtx layout.Context, snap panel.Snapshot, i int) layout.Dimensions {
	label := a.cmds[i].Label
	profile := ""
	if p, ok := snap.SelectedProfile(); ok {
		profile = p.Name
	}
	catalog := ""
	if a.sess != nil {
		catalog = a.sess.CatalogDir()
	}
	call := a.cmds[i].Run(menu.Context{
		Profile: profile,
		Catalog: catalog,
		Path:    a.pathEdit.Text(),
	})
	reason := ""
	if call.Disabled != "" && !call.Confirm {
		reason = call.Disabled
	}
	if a.armed == label {
		reason = call.Disabled
		if reason == "" {
			reason = "…"
		}
	}
	btn := &a.actBtn[i]
	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if a.armed == label || (a.cmdFocus && i == a.cmdIndex) {
			fill(gtx, colorSelect)
		}
		return layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				name := material.Body2(a.theme, label)
				name.Color = colorText
				if reason != "" && !call.Confirm && a.armed != label {
					name.Color = colorMuted
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(name.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if reason == "" {
							return layout.Dimensions{}
						}
						c := material.Caption(a.theme, reason)
						c.Color = colorMuted
						return c.Layout(gtx)
					}),
				)
			},
		)
	})
}

func (a *App) dashboard(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	view := a.board()
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.lightCards(gtx, view)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.chartCard(gtx, view)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.logCard(gtx, snap)
		}),
	)
}

func (a *App) lightCards(gtx layout.Context, view monitor.View) layout.Dimensions {
	return layout.Flex{}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.lightCard(gtx, a.text.Link, view.Connection, view.ConnectionText)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.lightCard(gtx, a.text.Handshake, view.Handshake, view.HandshakeText)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.lightCard(gtx, a.text.Route, view.Path, view.PathText)
		}),
	)
}

func (a *App) lightCard(gtx layout.Context, title string, light monitor.Light, text string) layout.Dimensions {
	if text == "" {
		text = a.text.Unknown
	}
	return card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.dot(gtx, lightColor(light))
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Body2(a.theme, title)
						l.Color = colorText
						return l.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Caption(a.theme, text)
				l.Color = colorMuted
				return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, l.Layout)
			}),
		)
	})
}

func lightColor(light monitor.Light) color.NRGBA {
	switch light {
	case monitor.Green:
		return colorGreen
	case monitor.Yellow:
		return colorYellow
	case monitor.Red:
		return colorRed
	default:
		return colorMuted
	}
}

func (a *App) chartCard(gtx layout.Context, view monitor.View) layout.Dimensions {
	return card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Body2(a.theme, a.text.Chart)
				l.Color = colorText
				return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, l.Layout)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.bars(gtx, view.Points)
			}),
		)
	})
}

func (a *App) bars(gtx layout.Context, points []monitor.Point) layout.Dimensions {
	height := gtx.Dp(36)
	gtx.Constraints.Min.Y = height
	gtx.Constraints.Max.Y = height
	if !hasMeasurement(points) {
		l := material.Caption(a.theme, a.text.NoData)
		l.Color = colorMuted
		return l.Layout(gtx)
	}
	max := int64(1)
	for _, p := range points {
		if p.HasMS && p.MS > max {
			max = p.MS
		}
	}
	kids := make([]layout.FlexChild, 0, len(points))
	for _, p := range points {
		p := p
		kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.bar(gtx, p, max, height)
		}))
	}
	return layout.Flex{Alignment: layout.End}.Layout(gtx, kids...)
}

func hasMeasurement(points []monitor.Point) bool {
	return len(points) > 0
}

func (a *App) bar(gtx layout.Context, p monitor.Point, max int64, height int) layout.Dimensions {
	h := gtx.Dp(4)
	c := colorRed
	if p.HasMS {
		c = colorAccent
		scaled := int(int64(height) * p.MS / max)
		if scaled < h {
			scaled = h
		}
		h = scaled
	}
	w := gtx.Dp(8)
	gtx.Constraints.Min = image.Pt(w+gtx.Dp(3), height)
	gtx.Constraints.Max = gtx.Constraints.Min
	defer clip.Rect{Min: image.Pt(0, height-h), Max: image.Pt(w, height)}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: c}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: image.Pt(w+gtx.Dp(3), height)}
}

func (a *App) logCard(gtx layout.Context, _ panel.Snapshot) layout.Dimensions {
	lines := a.logLines()
	if n := len(lines); n > a.logSeen {
		if a.logList.Position.First >= a.logSeen-8 {
			a.logList.Position.First = n - 1
		}
		a.logSeen = n
	}
	return card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Body2(a.theme, a.text.Journal)
				l.Color = colorText
				return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, l.Layout)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(lines) == 0 {
					l := material.Body2(a.theme, a.text.LogHint)
					l.Color = colorMuted
					return l.Layout(gtx)
				}
				return a.logList.Layout(gtx, len(lines), func(gtx layout.Context, i int) layout.Dimensions {
					l := material.Label(a.theme, unit.Sp(13), lines[i])
					l.Color = colorMuted
					l.MaxLines = 1
					return layout.Inset{Bottom: unit.Dp(2)}.Layout(gtx, l.Layout)
				})
			}),
		)
	})
}

func (a *App) logLines() []string {
	if a.sess == nil {
		return nil
	}
	text := a.sess.Log()
	if text == "" {
		return nil
	}
	lines := splitLines(text)
	if len(lines) > 80 {
		lines = lines[len(lines)-80:]
	}
	return lines
}

func splitLines(text string) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			out = append(out, text[start:i])
			start = i + 1
		}
	}
	if start < len(text) {
		out = append(out, text[start:])
	}
	return out
}
