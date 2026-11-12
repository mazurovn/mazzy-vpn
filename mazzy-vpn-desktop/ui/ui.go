// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package ui draws the Mazzy VPN desktop window. The top panel is always
// visible: status, server, protocol, Connect and Disconnect.
package ui

import (
	"context"
	"image"
	"image/color"
	"net"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/mazurovn/mazzy-vpn/desktop/menu"
	"github.com/mazurovn/mazzy-vpn/desktop/panel"
	"github.com/mazurovn/mazzy-vpn/desktop/session"
)

var (
	colorBG     = color.NRGBA{R: 0x08, G: 0x0a, B: 0x16, A: 0xff}
	colorPanel  = color.NRGBA{R: 0x16, G: 0x19, B: 0x31, A: 0xff}
	colorCard   = color.NRGBA{R: 0x12, G: 0x14, B: 0x28, A: 0xff}
	colorText   = color.NRGBA{R: 0xf6, G: 0xf7, B: 0xff, A: 0xff}
	colorMuted  = color.NRGBA{R: 0x92, G: 0x99, B: 0xb9, A: 0xff}
	colorCyan   = color.NRGBA{R: 0x68, G: 0xe9, B: 0xff, A: 0xff}
	colorGreen  = color.NRGBA{R: 0x58, G: 0xeb, B: 0xb4, A: 0xff}
	colorRed    = color.NRGBA{R: 0xff, G: 0x71, B: 0x8d, A: 0xff}
	colorYellow = color.NRGBA{R: 0xff, G: 0xd4, B: 0x79, A: 0xff}
	colorLine   = color.NRGBA{R: 0x68, G: 0xe9, B: 0xff, A: 0x55}
	colorSelect = color.NRGBA{R: 0x68, G: 0x92, B: 0xff, A: 0x28}
)

// App is the window state. Clickables live across frames.
type App struct {
	model      *panel.Model
	sess       *session.Session
	text       Text
	theme      *material.Theme
	connect    widget.Clickable
	disconnect widget.Clickable
	rows       []widget.Clickable
	list       layout.List
	cmds       []menu.Action
	actBtn     []widget.Clickable
	actList    layout.List
	logList    layout.List
	pathEdit   widget.Editor
	armed      string
	query      string
	palette    bool
	cmdIndex   int
	logFocus   bool
	checkBtn   widget.Clickable
	cmdToggle  widget.Clickable
	logClick   widget.Clickable
}

// New builds the window state for model.
func New(model *panel.Model) *App {
	th := material.NewTheme()
	th.Palette.Bg = colorBG
	th.Palette.Fg = colorText
	th.Palette.ContrastBg = colorCyan
	th.Palette.ContrastFg = colorBG
	return &App{
		model:    model,
		text:     Phrases(),
		theme:    th,
		list:     layout.List{Axis: layout.Vertical},
		cmds:     menu.Actions(),
		actList:  layout.List{Axis: layout.Vertical},
		logList:  layout.List{Axis: layout.Vertical},
		pathEdit: widget.Editor{SingleLine: true, Submit: true},
	}
}

// Run draws until the window closes. sess supplies the CLI actions and the
// live traffic lights. It may be nil in tests that only draw a snapshot.
func Run(w *app.Window, model *panel.Model, sess *session.Session) error {
	a := New(model)
	a.sess = sess
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			a.frame(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (a *App) frame(gtx layout.Context) {
	snap := a.model.View()
	a.ensureRows(len(snap.Profiles))
	for i := range snap.Profiles {
		if a.rows[i].Clicked(gtx) {
			a.logFocus = false
			if snap.Selected == i {
				a.model.Connect(context.Background())
			} else {
				a.model.Select(i)
			}
			snap = a.model.View()
		}
	}
	if a.connect.Clicked(gtx) {
		a.model.Connect(context.Background())
		snap = a.model.View()
	}
	if a.disconnect.Clicked(gtx) {
		a.model.Disconnect(context.Background())
		snap = a.model.View()
	}
	a.ensureActs(len(a.cmds))
	if a.sess != nil {
		for i := range a.cmds {
			if a.actBtn[i].Clicked(gtx) {
				a.activate(snap, i)
			}
		}
	}
	if a.checkBtn.Clicked(gtx) && a.sess != nil {
		go a.sess.PollChecks(context.Background())
	}
	if a.cmdToggle.Clicked(gtx) {
		a.palette = !a.palette
		a.logFocus = false
	}
	if a.logClick.Clicked(gtx) {
		a.logFocus = true
		a.palette = false
	}
	a.keys(gtx, snap)

	paint.Fill(gtx.Ops, colorBG)
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.top(gtx, snap)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.rule(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.body(gtx, snap)
		}),
	)
}

func (a *App) top(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	gtx.Constraints.Min.Y = gtx.Dp(108)
	gtx.Constraints.Max.Y = gtx.Dp(108)
	fill(gtx, colorPanel)
	return layout.Inset{Left: unit.Dp(18), Right: unit.Dp(18), Top: unit.Dp(12), Bottom: unit.Dp(12)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.brand(gtx)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.statusCluster(gtx, snap)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.actions(gtx)
				}),
			)
		},
	)
}

func (a *App) brand(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.H6(a.theme, "Mazzy VPN")
			l.Color = colorText
			return l.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.Caption(a.theme, "AmneziaWG · WireGuard")
			l.Color = colorCyan
			return l.Layout(gtx)
		}),
	)
}

func (a *App) statusCluster(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	server, protocol, transport := display(snap)
	return layout.Inset{Left: unit.Dp(28)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.statusLine(gtx, snap.Phase)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.meta(gtx, a.text.Server, server)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				value := protocol
				if transport != "" {
					if value != "" {
						value += "  ·  "
					}
					value += transport
				}
				return a.meta(gtx, a.text.Protocol, value)
			}),
		)
	})
}

func (a *App) statusLine(gtx layout.Context, phase panel.Phase) layout.Dimensions {
	label, dot := a.phaseText(phase)
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.dot(gtx, dot)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.Body2(a.theme, a.text.Status+"   "+label)
			l.Color = colorText
			return l.Layout(gtx)
		}),
	)
}

func (a *App) phaseText(phase panel.Phase) (string, color.NRGBA) {
	switch phase {
	case panel.PhaseConnected:
		return a.text.Connected, colorGreen
	case panel.PhaseConnecting:
		return a.text.Connecting, colorYellow
	case panel.PhaseError:
		return a.text.Failed, colorRed
	default:
		return a.text.Disconnected, colorMuted
	}
}

func (a *App) dot(gtx layout.Context, c color.NRGBA) layout.Dimensions {
	size := gtx.Dp(10)
	gtx.Constraints.Min = image.Pt(size, size)
	gtx.Constraints.Max = image.Pt(size, size)
	defer clip.Ellipse{Max: image.Pt(size, size)}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: c}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return layout.Dimensions{Size: image.Pt(size, size)}
}

func (a *App) meta(gtx layout.Context, k, v string) layout.Dimensions {
	if v == "" {
		v = "—"
	}
	l := material.Body2(a.theme, k+"   "+v)
	l.Color = colorMuted
	return l.Layout(gtx)
}

func (a *App) actions(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.button(gtx, &a.connect, a.text.Connect, colorCyan, a.model.CanConnect())
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.button(gtx, &a.disconnect, a.text.Disconnect, colorRed, a.model.CanDisconnect())
		}),
	)
}

func (a *App) button(gtx layout.Context, click *widget.Clickable, label string, bg color.NRGBA, enabled bool) layout.Dimensions {
	b := material.Button(a.theme, click, label)
	b.Background = bg
	b.Color = colorBG
	b.CornerRadius = unit.Dp(8)
	b.Inset = layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(16), Right: unit.Dp(16)}
	if !enabled {
		b.Background = color.NRGBA{R: 0x2a, G: 0x2e, B: 0x48, A: 0xff}
		b.Color = colorMuted
	}
	return b.Layout(gtx)
}

func (a *App) rule(gtx layout.Context) layout.Dimensions {
	h := gtx.Dp(1)
	gtx.Constraints.Min.Y = h
	gtx.Constraints.Max.Y = h
	fill(gtx, colorLine)
	return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
}

func (a *App) body(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	return a.client(gtx, snap)
}

func display(snap panel.Snapshot) (server, protocol, transport string) {
	server, protocol, transport = snap.Server, snap.Protocol, snap.Transport
	if p, ok := snap.SelectedProfile(); ok {
		if server == "" {
			server = p.Name
		}
		if protocol == "" {
			protocol = p.Protocol
		}
		if transport == "" {
			if port := endpointPort(p.Endpoint); port != "" {
				transport = "udp/" + port
			}
		}
	}
	return server, protocol, transport
}

func endpointPort(endpoint string) string {
	_, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return ""
	}
	return port
}

func (a *App) ensureActs(n int) {
	if len(a.actBtn) >= n {
		return
	}
	next := make([]widget.Clickable, n)
	copy(next, a.actBtn)
	a.actBtn = next
}

func (a *App) ensureRows(n int) {
	if len(a.rows) >= n {
		return
	}
	next := make([]widget.Clickable, n)
	copy(next, a.rows)
	a.rows = next
}

func card(gtx layout.Context, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(16)).Layout(gtx, w)
	call := macro.Stop()
	defer clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(12)).Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, colorCard)
	call.Add(gtx.Ops)
	return dims
}

func fill(gtx layout.Context, c color.NRGBA) {
	size := gtx.Constraints.Max
	if size.X <= 0 || size.Y <= 0 {
		size = gtx.Constraints.Min
	}
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: c}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}
