// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Command mazzy-vpn-desktop is the Linux control window. It links mazzy-core
// in-process (ADR-0002) and draws with gioui (ADR-0004), so it does not need
// the CLI binary or a system WebKit.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/unit"

	"github.com/mazurovn/mazzy-vpn/desktop/panel"
	"github.com/mazurovn/mazzy-vpn/desktop/session"
	"github.com/mazurovn/mazzy-vpn/desktop/ui"
)

func main() {
	sess := session.New(session.DefaultConfig())
	model := panel.New(sess)
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Mazzy VPN"),
			app.Size(unit.Dp(1180), unit.Dp(760)),
			app.MinSize(unit.Dp(960), unit.Dp(640)),
		)
		model.OnChange = func() { w.Invalidate() }
		sess.SetFire(func() { model.Reload(context.Background()) })
		model.Reload(context.Background())
		go func() {
			tick := time.NewTicker(4 * time.Second)
			defer tick.Stop()
			for range tick.C {
				model.Reload(context.Background())
			}
		}()
		err := ui.Run(w, model, sess)
		// Closing the window stops only a CLI process this window started.
		sess.Close(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
