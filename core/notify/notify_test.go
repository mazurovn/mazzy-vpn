// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package notify

import (
	"context"
	"testing"
)

func TestDisabledIsNoOp(t *testing.T) {
	called := false
	n := &Notifier{Enabled: false, send: func(context.Context, string, string, string, Level) error {
		called = true
		return nil
	}}
	n.Connected("Berlin", "1.2.3.4")
	if called {
		t.Fatal("disabled notifier must not send")
	}
}

func TestEventsDispatch(t *testing.T) {
	var events []string
	n := &Notifier{
		AppName: "Mazzy VPN", Enabled: true,
		send: func(_ context.Context, app, title, body string, lvl Level) error {
			events = append(events, title+"|"+string(lvl))
			return nil
		},
	}
	n.Connected("NL", "9.9.9.9")
	n.Reconnecting("NL", "no egress")
	n.Reconnected("NL", "9.9.9.9")
	n.Disconnected("NL")
	n.Failed("NL", "server down")

	want := []string{
		"Mazzy VPN — Connected|normal",
		"Mazzy VPN — Reconnecting|critical",
		"Mazzy VPN — Reconnected|normal",
		"Mazzy VPN — Disconnected|low",
		"Mazzy VPN — Failed|critical",
	}
	if len(events) != len(want) {
		t.Fatalf("got %d events, want %d: %v", len(events), len(want), events)
	}
	for i, w := range want {
		if events[i] != w {
			t.Errorf("event %d = %q, want %q", i, events[i], w)
		}
	}
}

func TestNilNotifierSafe(t *testing.T) {
	var n *Notifier
	// Must not panic.
	n.Connected("X", "Y")
}
