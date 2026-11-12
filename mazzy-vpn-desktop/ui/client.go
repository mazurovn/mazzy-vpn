// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package ui

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/mazurovn/mazzy-vpn/desktop/monitor"
	"github.com/mazurovn/mazzy-vpn/desktop/panel"
)

func (a *App) client(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(14), Right: unit.Dp(14), Top: unit.Dp(8), Bottom: unit.Dp(10)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.tabBar(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(280)
							gtx.Constraints.Max.X = gtx.Dp(300)
							return a.serverCard(gtx, snap)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return a.workspace(gtx, snap)
						}),
					)
				}),
			)
		},
	)
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
						l := material.Label(a.theme, unit.Sp(14), p.Name)
						l.Color = colorText
						l.MaxLines = 1
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

func (a *App) workspace(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	switch a.tab {
	case 1:
		return a.menuPane(gtx, snap)
	case 2:
		return a.logCard(gtx, snap)
	default:
		// Rigid so the status card stays as tall as its content. The
		// parent flex already fills the pane, and an empty stretch
		// would paint a hollow card.
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return a.hero(gtx, snap)
			}),
		)
	}
}

func (a *App) hero(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	server, protocol, transport := display(snap)
	if server == "" {
		server = a.text.Empty
	}
	sub := protocol
	if transport != "" {
		if sub != "" {
			sub += "   ·   "
		}
		sub += transport
	}
	view := a.board()
	status, dot := a.phaseText(snap.Phase)
	statusText := status
	if view.Connection != monitor.Unknown && view.ConnectionText != "" {
		dot = lightColor(view.Connection)
		statusText = view.ConnectionText
	}
	extra := extraLights(visibleState(view, a.text.Link, a.text.Handshake, a.text.Route, a.text.EgressIP, a.text.Clean, a.text.Service), a.text.Link)
	return card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.H5(a.theme, server)
				l.Color = colorText
				return l.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if sub == "" {
					return layout.Dimensions{}
				}
				l := material.Body1(a.theme, sub)
				l.Color = colorAccent
				return layout.Inset{Top: unit.Dp(2)}.Layout(gtx, l.Layout)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return a.dot(gtx, dot)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Body1(a.theme, statusText)
							l.Color = colorText
							return l.Layout(gtx)
						}),
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Body2(a.theme, fmt.Sprintf("%d %s. %s", len(snap.Profiles), a.text.InCatalog, a.stepLine(snap)))
				l.Color = colorMuted
				return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, l.Layout)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if len(extra) == 0 && !hasMeasurement(view.Points) {
					return layout.Dimensions{}
				}
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					kids := make([]layout.FlexChild, 0, len(extra)+1)
					for _, c := range extra {
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
			}),
		)
	})
}

func (a *App) stepLine(snap panel.Snapshot) string {
	if snap.Phase == panel.PhaseError && snap.Err != "" {
		return snap.Err
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
