// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package notify sends desktop notifications about VPN state changes, the same
// way AdGuard VPN does (connected / reconnecting / disconnected), via the base
// notify-send tool when available. It degrades gracefully to a no-op when no
// notifier is present, so headless/CI use is safe.
package notify

import (
	"context"
	"os/exec"
	"time"
)

// Level maps to notify-send urgency.
type Level string

const (
	Low      Level = "low"
	Normal   Level = "normal"
	Critical Level = "critical"
)

// Notifier sends desktop notifications.
type Notifier struct {
	// AppName appears as the notification source.
	AppName string
	// Enabled turns notifications on/off (respects MAZZY_NO_NOTIFY).
	Enabled bool
	// send is overridable in tests; nil uses notify-send.
	send func(ctx context.Context, app, title, body string, level Level) error
}

// New returns a Notifier that uses notify-send if present.
func New() *Notifier {
	return &Notifier{AppName: "Mazzy VPN", Enabled: available()}
}

// available reports whether a desktop notifier is usable.
func available() bool {
	_, err := exec.LookPath("notify-send")
	return err == nil
}

func (n *Notifier) dispatch(title, body string, level Level) {
	if n == nil || !n.Enabled {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if n.send != nil {
		_ = n.send(ctx, n.AppName, title, body, level)
		return
	}
	_ = exec.CommandContext(ctx, "notify-send",
		"--app-name", n.AppName,
		"--urgency", string(level),
		title, body,
	).Run()
}

// Connected notifies that the tunnel is up and protected.
func (n *Notifier) Connected(zone, egressIP string) {
	n.dispatch("Mazzy VPN — Connected", "Zone: "+zone+"\nEgress: "+egressIP, Normal)
}

// Reconnecting notifies that a drop was detected and recovery is underway.
func (n *Notifier) Reconnecting(zone, reason string) {
	n.dispatch("Mazzy VPN — Reconnecting", "Zone: "+zone+"\n"+reason, Critical)
}

// Reconnected notifies that recovery succeeded.
func (n *Notifier) Reconnected(zone, egressIP string) {
	n.dispatch("Mazzy VPN — Reconnected", "Zone: "+zone+"\nEgress: "+egressIP, Normal)
}

// Disconnected notifies that the tunnel is down.
func (n *Notifier) Disconnected(zone string) {
	n.dispatch("Mazzy VPN — Disconnected", "Zone: "+zone, Low)
}

// Failed notifies that a connection or recovery attempt failed.
func (n *Notifier) Failed(zone, reason string) {
	n.dispatch("Mazzy VPN — Failed", "Zone: "+zone+"\n"+reason, Critical)
}
