// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"testing"

	"github.com/mazurovn/mazzy-vpn/core/settings"
)

// These tests lock in that the previously-dead settings toggles now actually
// drive behaviour (audit P1). They assert the resolution logic directly so a
// regression that silently ignores a setting fails the gate.

func TestResolveNotifications(t *testing.T) {
	if !resolveNotifications(settings.Settings{Notifications: true}) {
		t.Error("Notifications=true must enable notifications")
	}
	if resolveNotifications(settings.Settings{Notifications: false}) {
		t.Error("Notifications=false must disable notifications (P1-2)")
	}
}

func TestResolveAutoReconnect(t *testing.T) {
	cases := []struct {
		name string
		set  settings.Settings
		args []string
		want bool
	}{
		{"setting on, no flag", settings.Settings{AutoReconnect: true}, nil, true},
		{"setting off, no flag", settings.Settings{AutoReconnect: false}, nil, false},
		{"setting on, --no-reconnect overrides", settings.Settings{AutoReconnect: true}, []string{"--no-reconnect"}, false},
		{"setting off, flag redundant", settings.Settings{AutoReconnect: false}, []string{"--no-reconnect"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveAutoReconnect(tc.set, tc.args); got != tc.want {
				t.Errorf("resolveAutoReconnect = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestKillSwitchDefaultOn guards the security-relevant default: the fail-closed
// kill-switch must ship enabled.
func TestKillSwitchDefaultOn(t *testing.T) {
	if !settings.Default().KillSwitch {
		t.Fatal("KillSwitch must default to on (fail-closed)")
	}
}
