// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package guard implements the mazzy-core kill-switch and IPv6 leak guard via
// nftables (base `nft`, ADR-0005). It preserves parity with the bash
// ipv6_guard_install and transition_guard fail-closed logic.
package guard

import (
	"context"
	"fmt"
	"regexp"

	"github.com/mazurovn/mazzy-vpn/core/netexec"
)

const (
	// IPv6GuardTable drops all IPv6 egress except via the VPN interface and lo,
	// while allowing IPv4 (which the policy routing / fail-closed layer covers).
	IPv6GuardTable = "mazzy_vpn_ipv6_guard"
	// TransitionGuardTable is the fail-closed kill-switch used while switching
	// or recovering: everything except lo is administratively prohibited.
	TransitionGuardTable = "mazzy_vpn_transition_guard"
)

var ifaceRe = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,64}$`)

// Guard installs/removes nftables guards.
type Guard struct {
	Runner netexec.Runner
}

// New creates a Guard.
func New(r netexec.Runner) *Guard { return &Guard{Runner: r} }

// applyRuleset feeds an nft ruleset via `nft -f -` (stdin). Runner does not
// support stdin, so we pass the ruleset with `nft -f /dev/stdin` is also not
// possible; instead we use `nft -f -` requires stdin. We therefore build the
// ruleset through discrete `nft` commands is verbose; to keep parity and
// atomicity we write to a private temp file and `nft -f <file>`.
func (g *Guard) applyRuleset(ctx context.Context, ruleset string) error {
	f, err := writePrivateTemp(ruleset)
	if err != nil {
		return err
	}
	defer removeFile(f)
	_, err = g.Runner.Run(ctx, "nft", "-f", f)
	return err
}

// InstallIPv6Guard blocks IPv6 leaks: only lo and the VPN interface may emit
// IPv6; IPv4 is accepted (handled by routing + transition guard).
func (g *Guard) InstallIPv6Guard(ctx context.Context, iface string) error {
	if !ifaceRe.MatchString(iface) {
		return fmt.Errorf("invalid interface name %q", iface)
	}
	ruleset := fmt.Sprintf(`table inet %[1]s {
    chain output {
        type filter hook output priority -140; policy drop;
        meta nfproto ipv4 accept
        oifname "lo" accept
        oifname "%[2]s" accept
    }
    chain forward {
        type filter hook forward priority -140; policy drop;
        meta nfproto ipv4 accept
        oifname "%[2]s" accept
    }
}
`, IPv6GuardTable, iface)
	g.deleteTable(ctx, IPv6GuardTable)
	return g.applyRuleset(ctx, ruleset)
}

// InstallFailClosed installs the transition kill-switch: reject everything
// except loopback. Used while switching/recovering so there is no leak window.
func (g *Guard) InstallFailClosed(ctx context.Context) error {
	ruleset := fmt.Sprintf(`table inet %[1]s {
    chain output {
        type filter hook output priority -150; policy accept;
        oifname "lo" accept
        reject with icmpx type admin-prohibited
    }
    chain forward {
        type filter hook forward priority -150; policy accept;
        reject with icmpx type admin-prohibited
    }
}
`, TransitionGuardTable)
	g.deleteTable(ctx, TransitionGuardTable)
	return g.applyRuleset(ctx, ruleset)
}

// RemoveIPv6Guard deletes the IPv6 guard table.
func (g *Guard) RemoveIPv6Guard(ctx context.Context) error {
	return g.deleteTableErr(ctx, IPv6GuardTable)
}

// RemoveFailClosed deletes the transition kill-switch table.
func (g *Guard) RemoveFailClosed(ctx context.Context) error {
	return g.deleteTableErr(ctx, TransitionGuardTable)
}

// deleteTable best-effort removes a table (ignores "not found").
func (g *Guard) deleteTable(ctx context.Context, table string) {
	_, _ = g.Runner.Run(ctx, "nft", "delete", "table", "inet", table)
}

func (g *Guard) deleteTableErr(ctx context.Context, table string) error {
	_, err := g.Runner.Run(ctx, "nft", "delete", "table", "inet", table)
	return err
}
