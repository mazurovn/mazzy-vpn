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
			app.Size(unit.Dp(980), unit.Dp(680)),
			app.MinSize(unit.Dp(760), unit.Dp(520)),
		)
		model.OnChange = func() { w.Invalidate() }
		model.Reload(context.Background())
		err := ui.Run(w, model)
		// The userspace tunnel lives in this process. Closing the window
		// drops only the connection this window started.
		sess.Close(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
