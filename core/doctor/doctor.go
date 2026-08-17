// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package doctor provides structured, portable diagnostics for the autonomous
// mazzy-core stack. Unlike the legacy bash doctor, it does NOT check for awg,
// awg-quick, wg, wg-quick, jq, socat or python3: mazzy-core embeds the engine
// and replaces those tools (ADR-0003). It only checks base OS facilities that
// the native routes/dns/guard code legitimately uses (ADR-0005): ip, nft, and
// a DNS backend, plus /dev/net/tun and privilege.
//
// Results are structured so the CLI, Desktop and API can all render them; no
// credentials or absolute host paths are disclosed.
package doctor

import (
	"context"
	"os"

	"github.com/mazurovn/mazzy-vpn/core/netexec"
)

// Level is the severity of a diagnostic result.
type Level int

const (
	OK Level = iota
	WARN
	FAIL
)

func (l Level) String() string {
	switch l {
	case OK:
		return "OK"
	case WARN:
		return "WARN"
	case FAIL:
		return "FAIL"
	default:
		return "?"
	}
}

// Check is one diagnostic result.
type Check struct {
	Name   string `json:"name"`
	Level  Level  `json:"-"`
	Status string `json:"status"` // "OK"/"WARN"/"FAIL"
	Detail string `json:"detail,omitempty"`
}

// Report is the full diagnostic outcome.
type Report struct {
	Checks []Check `json:"checks"`
	OK     int     `json:"ok"`
	Warn   int     `json:"warn"`
	Fail   int     `json:"fail"`
}

// Healthy reports whether no FAIL-level checks are present.
func (r *Report) Healthy() bool { return r.Fail == 0 }

// Environment abstracts host queries so diagnostics are unit-testable.
type Environment interface {
	// LookPath reports whether a base tool is resolvable.
	LookPath(bin string) bool
	// FileExists reports whether a path exists (e.g. /dev/net/tun).
	FileExists(path string) bool
	// IsRoot reports whether we have privilege to manage interfaces.
	IsRoot() bool
}

// osEnv is the production Environment.
type osEnv struct{}

func (osEnv) LookPath(bin string) bool    { return netexec.Available(bin) }
func (osEnv) FileExists(path string) bool { _, err := os.Stat(path); return err == nil }
func (osEnv) IsRoot() bool                { return os.Geteuid() == 0 }

// Run executes all diagnostics. Pass nil env to use the real host.
func Run(_ context.Context, env Environment) *Report {
	if env == nil {
		env = osEnv{}
	}
	r := &Report{}

	// Base OS tools the native data path uses (ADR-0005). These are part of a
	// networking-capable OS, not installable VPN dependencies.
	add := func(c Check) {
		switch c.Level {
		case OK:
			r.OK++
		case WARN:
			r.Warn++
		case FAIL:
			r.Fail++
		}
		c.Status = c.Level.String()
		r.Checks = append(r.Checks, c)
	}

	// ip (iproute2): required for addressing and policy routing.
	if env.LookPath("ip") {
		add(Check{Name: "iproute2 (ip)", Level: OK, Detail: "present"})
	} else {
		add(Check{Name: "iproute2 (ip)", Level: FAIL, Detail: "required for routing"})
	}

	// nft (nftables): required for the kill-switch / leak guard.
	if env.LookPath("nft") {
		add(Check{Name: "nftables (nft)", Level: OK, Detail: "present"})
	} else {
		add(Check{Name: "nftables (nft)", Level: FAIL, Detail: "required for leak guard"})
	}

	// DNS backend: resolvectl preferred; resolvconf acceptable.
	switch {
	case env.LookPath("resolvectl"):
		add(Check{Name: "DNS backend", Level: OK, Detail: "resolvectl"})
	case env.LookPath("resolvconf"):
		add(Check{Name: "DNS backend", Level: WARN, Detail: "resolvconf (stdin backend pending)"})
	default:
		add(Check{Name: "DNS backend", Level: WARN, Detail: "no resolvectl/resolvconf; VPN DNS may not apply"})
	}

	// /dev/net/tun: required to create the userspace tunnel.
	if env.FileExists("/dev/net/tun") {
		add(Check{Name: "TUN device (/dev/net/tun)", Level: OK, Detail: "present"})
	} else {
		add(Check{Name: "TUN device (/dev/net/tun)", Level: FAIL, Detail: "missing; cannot create tunnel"})
	}

	// Privilege: managing interfaces needs root/CAP_NET_ADMIN.
	if env.IsRoot() {
		add(Check{Name: "privilege", Level: OK, Detail: "running as root"})
	} else {
		add(Check{Name: "privilege", Level: WARN, Detail: "not root; connect/disconnect need privilege"})
	}

	// Embedded engine: mazzy-core embeds amneziawg-go, so there is nothing to
	// install. This is an explicit positive to make the autonomy visible and to
	// contrast with the legacy awg/awg-quick checks that are intentionally gone.
	add(Check{Name: "AmneziaWG engine", Level: OK, Detail: "embedded (no awg/awg-quick required)"})

	return r
}
