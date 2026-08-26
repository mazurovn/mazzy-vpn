// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package guard implements the mazzy-core kill-switch and IPv6 leak guard via
// nftables (base `nft`, ADR-0005). It preserves parity with the bash
// ipv6_guard_install and transition_guard fail-closed logic.
package guard

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mazurovn/mazzy-vpn/core/netexec"
)

const (
	// IPv6GuardTable drops all IPv6 egress except via the VPN interface and lo,
	// while allowing IPv4 (which the policy routing / fail-closed layer covers).
	IPv6GuardTable = "mazzy_vpn_ipv6_guard"
	// TransitionGuardTable is the fail-closed kill-switch used while switching
	// or recovering: everything except lo is administratively prohibited.
	TransitionGuardTable = "mazzy_vpn_transition_guard"
	// ConnmarkTable carries the CONNMARK save/restore rules (wg-quick parity,
	// C1-4a2): it saves the socket fwmark onto the connection in postrouting and
	// restores it in prerouting, so reply packets keep the mark and route out
	// the correct table instead of looping into the tunnel.
	ConnmarkTable = "mazzy_vpn_connmark"
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

// InstallKillSwitch installs a fwmark-aware fail-closed kill-switch armed
// while a tunnel is being re-established after an egress drop. Allowed out:
//
//   - loopback;
//   - packets INTO a managed tunnel interface (they leave the host encrypted,
//     so this is not a plaintext leak) — without this the daemon's own egress
//     verification through the re-established tunnel was rejected, the
//     kill-switch could never be disarmed, and the host stayed sealed until a
//     manual disconnect/recover (the "Belgium loop" incident);
//   - packets carrying the WireGuard socket fwmark (encrypted handshake);
//   - ICMP echo-requests, so zone-liveness ranking (failover!) keeps working
//     while sealed. This reveals only which servers we ping — an acceptable
//     trade for a transition guard whose job is enabling recovery.
//
// Everything else is rejected: no plaintext application traffic can leave via
// the plain uplink during the gap.
func (g *Guard) InstallKillSwitch(ctx context.Context, mark uint32, tunnelIfaces []string) error {
	tunAccept := ""
	if len(tunnelIfaces) > 0 {
		quoted := make([]string, 0, len(tunnelIfaces))
		for _, n := range tunnelIfaces {
			// Strict whitelist before interpolation into the ruleset: a name
			// containing a quote/space/newline could otherwise inject nft rules.
			// Kernel interface names are ≤15 chars from a safe alphabet anyway.
			if ifaceRe.MatchString(n) {
				quoted = append(quoted, `"`+n+`"`)
			}
		}
		if len(quoted) > 0 {
			tunAccept = "        oifname { " + strings.Join(quoted, ", ") + " } accept\n"
		}
	}
	ruleset := fmt.Sprintf(`table inet %[1]s {
    chain output {
        type filter hook output priority -150; policy accept;
        oifname "lo" accept
%[3]s        meta mark %[2]d accept
        meta l4proto icmp icmp type echo-request accept
        meta l4proto ipv6-icmp icmpv6 type echo-request accept
        reject with icmpx type admin-prohibited
    }
    chain forward {
        type filter hook forward priority -150; policy accept;
        reject with icmpx type admin-prohibited
    }
}
`, TransitionGuardTable, mark, tunAccept)
	g.deleteTable(ctx, TransitionGuardTable)
	return g.applyRuleset(ctx, ruleset)
}

// InstallConnmark installs the CONNMARK save/restore rules for the given fwmark
// so marked UDP reply packets keep their mark (wg-quick parity). It covers both
// IPv4 and IPv6 via an inet table.
func (g *Guard) InstallConnmark(ctx context.Context, mark uint32) error {
	ruleset := fmt.Sprintf(`table inet %[1]s {
    chain premangle {
        type filter hook prerouting priority -150; policy accept;
        meta l4proto udp meta mark set ct mark
    }
    chain postmangle {
        type filter hook postrouting priority -150; policy accept;
        meta l4proto udp mark %[2]d ct mark set mark
    }
}
`, ConnmarkTable, mark)
	g.deleteTable(ctx, ConnmarkTable)
	return g.applyRuleset(ctx, ruleset)
}

// RemoveConnmark deletes the CONNMARK table.
func (g *Guard) RemoveConnmark(ctx context.Context) error {
	return g.deleteTableErr(ctx, ConnmarkTable)
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
