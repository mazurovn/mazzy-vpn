// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package dns applies VPN DNS for an interface, mirroring wg-quick set_dns /
// unset_dns. Kernel/service access is via base tools (ADR-0005): resolvectl
// when available, otherwise resolvconf.
package dns

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/mazurovn/mazzy-vpn/core/netexec"
)

// Manager applies and reverts DNS for one interface.
type Manager struct {
	Runner    netexec.Runner
	Interface string
	applied   bool
	backend   string
}

// New creates a DNS Manager.
func New(r netexec.Runner, iface string) *Manager {
	return &Manager{Runner: r, Interface: iface}
}

// Up sets the given nameservers for the interface. Empty list is a no-op.
func (m *Manager) Up(ctx context.Context, servers []string) error {
	if len(servers) == 0 {
		return nil
	}
	for _, s := range servers {
		if _, err := netip.ParseAddr(strings.TrimSpace(s)); err != nil {
			return fmt.Errorf("invalid DNS server %q: %w", s, err)
		}
	}
	switch {
	case netexec.Available("resolvectl"):
		m.backend = "resolvectl"
		args := append([]string{"dns", m.Interface}, servers...)
		if _, err := m.Runner.Run(ctx, "resolvectl", args...); err != nil {
			return err
		}
	case netexec.Available("resolvconf"):
		m.backend = "resolvconf"
		// resolvconf reads "nameserver <ip>" lines from stdin (C1-4b2).
		ir, ok := m.Runner.(netexec.InputRunner)
		if !ok {
			return fmt.Errorf("resolvconf backend requires an input-capable runner")
		}
		var b strings.Builder
		for _, s := range servers {
			fmt.Fprintf(&b, "nameserver %s\n", strings.TrimSpace(s))
		}
		if _, err := ir.RunInput(ctx, b.String(), "resolvconf", "-a", m.Interface, "-m", "0", "-x"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("no DNS backend (resolvectl/resolvconf) available")
	}
	m.applied = true
	return nil
}

// Down reverts DNS for the interface.
func (m *Manager) Down(ctx context.Context) error {
	if !m.applied {
		return nil
	}
	switch m.backend {
	case "resolvectl":
		_, err := m.Runner.Run(ctx, "resolvectl", "revert", m.Interface)
		return err
	case "resolvconf":
		_, err := m.Runner.Run(ctx, "resolvconf", "-d", m.Interface, "-f")
		return err
	}
	return nil
}
