// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package ui

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/mazurovn/mazzy-vpn/desktop/monitor"
	"github.com/mazurovn/mazzy-vpn/desktop/panel"
)

func (a *App) client(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(12), Bottom: unit.Dp(12)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Caption(a.theme, a.text.RootHint)
					l.Color = colorMuted
					return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, l.Layout)
				}),
				layout.Flexed(1.15, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Dp(320)
							gtx.Constraints.Max.X = gtx.Dp(360)
							return a.serverCard(gtx, snap)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							if a.palette {
								return a.actionCard(gtx, snap)
							}
							return a.dashColumn(gtx, snap)
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Flexed(0.85, func(gtx layout.Context) layout.Dimensions {
					return a.logClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return a.logCard(gtx, snap)
					})
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
				if a.query != "" {
					title += "  ·  " + a.query
				}
				l := material.H6(a.theme, title)
				l.Color = colorText
				return l.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Caption(a.theme, a.text.Filter+"  ·  ↑↓  Enter  Esc")
				l.Color = colorMuted
				return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(8)}.Layout(gtx, l.Layout)
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
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return a.button(gtx, &a.cmdToggle, a.text.Commands, colorCyan, true)
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
		label := p.Name + "    " + p.Protocol
		if port := endpointPort(p.Endpoint); port != "" {
			label += "    udp/" + port
		}
		l := material.Body1(a.theme, label)
		l.Color = colorText
		return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, l.Layout)
	})
}

func (a *App) dashColumn(gtx layout.Context, snap panel.Snapshot) layout.Dimensions {
	view := a.board()
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.lightCards(gtx, view)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.proofCards(gtx, view)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.chartCard(gtx, view)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.button(gtx, &a.checkBtn, "egress", colorCyan, a.sess != nil)
				}),
			)
		}),
	)
}

func (a *App) proofCards(gtx layout.Context, view monitor.View) layout.Dimensions {
	return layout.Flex{}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.lightCard(gtx, a.text.EgressIP, view.EgressLight, view.EgressText)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.lightCard(gtx, a.text.Clean, view.Clean, view.CleanText)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.lightCard(gtx, a.text.Service, view.Service, view.ServiceText)
		}),
	)
}
