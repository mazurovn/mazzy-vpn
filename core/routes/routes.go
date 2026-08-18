// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package routes applies interface addressing and the wg-quick policy-routing
// model (fwmark table + suppress_prefixlength) for a full-tunnel (0.0.0.0/0,
// ::/0) AmneziaWG/WireGuard connection.
//
// This is the code amneziawg-go does NOT provide (audit R1). Behavior mirrors
// wg-quick's add_default() / add_route(). Kernel access is via base `ip`
// (ADR-0005). The kill-switch (nft) lives in core/guard.
package routes

import (
	"context"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"github.com/mazurovn/mazzy-vpn/core/netexec"
	"github.com/mazurovn/mazzy-vpn/core/profile"
)

// Applier applies and reverts the network configuration for one interface.
type Applier struct {
	Runner    netexec.Runner
	Interface string
	// Table is the routing table / fwmark used for policy routing. wg-quick
	// defaults to 51820 when no explicit fwmark; we pin an explicit value for
	// determinism unless the config sets FwMark.
	Table uint32
	// Uplink, when set, pins the encrypted-traffic egress to a specific physical
	// interface (e.g. a wired uplink vs Wi‑Fi). A host route for the server
	// endpoint is installed via this uplink so WireGuard's own marked packets
	// leave through it (U7). Empty = use the default route.
	Uplink string

	appliedAddrs    []netip.Prefix
	appliedDefault  map[bool]bool // family(isV6)->added
	appliedEndpoint string        // endpoint host route we installed (for teardown)
}

// DefaultMark is wg-quick's default fwmark/table value used when the config
// does not pin one. The SAME number is used as the socket fwmark and the
// routing table (G1): the `not fwmark <m> table <m>` rule only works when the
// engine's own encrypted packets carry that mark, otherwise they loop back
// into the tunnel and connectivity breaks.
const DefaultMark = 51820

// EffectiveMark returns the single mark to apply to BOTH the wireguard socket
// (UAPI fwmark=) and the routing table. Engine and routes MUST agree.
func EffectiveMark(cfg *profile.Config) uint32 {
	if cfg.FwMark != 0 {
		return cfg.FwMark
	}
	return DefaultMark
}

// New builds an Applier from a parsed config.
func New(r netexec.Runner, iface string, cfg *profile.Config) *Applier {
	return &Applier{
		Runner:         r,
		Interface:      iface,
		Table:          EffectiveMark(cfg),
		appliedDefault: map[bool]bool{},
	}
}

// Up assigns addresses and, for any /0 AllowedIPs, installs policy routing.
func (a *Applier) Up(ctx context.Context, cfg *profile.Config) error {
	if err := a.setLinkUp(ctx); err != nil {
		return err
	}
	for _, addr := range cfg.Addresses {
		p, err := netip.ParsePrefix(addr)
		if err != nil {
			return fmt.Errorf("bad interface address %q: %w", addr, err)
		}
		if err := a.addAddress(ctx, p); err != nil {
			return err
		}
		a.appliedAddrs = append(a.appliedAddrs, p)
	}

	// U7: pin the encrypted-traffic egress to a chosen uplink by routing the
	// server endpoint host through it. Must be done BEFORE the default route so
	// the WireGuard socket can reach the server via that uplink.
	if a.Uplink != "" {
		if host := cfg.EndpointHost(); host != "" {
			installed, err := a.addEndpointRoute(ctx, host)
			if err != nil {
				return fmt.Errorf("pin endpoint via uplink %s: %w", a.Uplink, err)
			}
			// Only record the pin when a route was actually installed (a literal
			// IP). Hostnames are skipped, so teardown must not try to remove one.
			if installed {
				a.appliedEndpoint = host
			}
		}
	}

	// Determine which families need a default route.
	needV4, needV6 := allowedDefaults(cfg)
	if needV4 {
		if err := a.addDefault(ctx, false); err != nil {
			return err
		}
	}
	if needV6 {
		if err := a.addDefault(ctx, true); err != nil {
			return err
		}
	}
	return nil
}

// addEndpointRoute installs a host route for the server endpoint via the pinned
// uplink, so the WireGuard socket's encrypted packets egress that interface. It
// reports whether a route was actually installed (false for a hostname, which
// is skipped because it needs resolution and may change).
func (a *Applier) addEndpointRoute(ctx context.Context, host string) (bool, error) {
	ip, err := netip.ParseAddr(host)
	if err != nil {
		// Endpoint is a hostname; skip pinning (needs resolution + may change).
		return false, nil
	}
	fam, hostRoute := "-4", ip.String()+"/32"
	if ip.Is6() {
		fam, hostRoute = "-6", ip.String()+"/128"
	}
	if _, err := a.Runner.Run(ctx, "ip", fam, "route", "add", hostRoute, "dev", a.Uplink); err != nil {
		return false, err
	}
	return true, nil
}

// delEndpointRoute removes the pinned endpoint host route on teardown.
func (a *Applier) delEndpointRoute(ctx context.Context) error {
	if a.appliedEndpoint == "" {
		return nil
	}
	ip, err := netip.ParseAddr(a.appliedEndpoint)
	if err != nil {
		return nil
	}
	fam, hostRoute := "-4", ip.String()+"/32"
	if ip.Is6() {
		fam, hostRoute = "-6", ip.String()+"/128"
	}
	_, err = a.Runner.Run(ctx, "ip", fam, "route", "del", hostRoute, "dev", a.Uplink)
	a.appliedEndpoint = ""
	return err
}

// Down reverts everything Up applied. It is best-effort and tolerant.
func (a *Applier) Down(ctx context.Context) error {
	var firstErr error
	note := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for isV6, added := range a.appliedDefault {
		if !added {
			continue
		}
		note(a.delDefault(ctx, isV6))
	}
	note(a.delEndpointRoute(ctx))
	// Addresses are removed automatically when the interface is destroyed by
	// the engine; explicit flush is a safety net.
	return firstErr
}

func (a *Applier) setLinkUp(ctx context.Context) error {
	_, err := a.Runner.Run(ctx, "ip", "link", "set", "up", "dev", a.Interface)
	return err
}

func (a *Applier) addAddress(ctx context.Context, p netip.Prefix) error {
	_, err := a.Runner.Run(ctx, "ip", a.famFlag(p.Addr().Is6()),
		"address", "add", p.String(), "dev", a.Interface)
	return err
}

func (a *Applier) famFlag(isV6 bool) string {
	if isV6 {
		return "-6"
	}
	return "-4"
}

// addDefault mirrors wg-quick add_default(): fwmark rule + suppress rule +
// default route into the dedicated table.
func (a *Applier) addDefault(ctx context.Context, isV6 bool) error {
	fam := a.famFlag(isV6)
	tbl := strconv.FormatUint(uint64(a.Table), 10)
	def := "0.0.0.0/0"
	if isV6 {
		def = "::/0"
	}
	// ip -N rule add not fwmark <t> table <t>
	if _, err := a.Runner.Run(ctx, "ip", fam, "rule", "add", "not", "fwmark", tbl, "table", tbl); err != nil {
		return err
	}
	// ip -N rule add table main suppress_prefixlength 0
	if _, err := a.Runner.Run(ctx, "ip", fam, "rule", "add", "table", "main", "suppress_prefixlength", "0"); err != nil {
		return err
	}
	// ip -N route add <def> dev <iface> table <t>
	if _, err := a.Runner.Run(ctx, "ip", fam, "route", "add", def, "dev", a.Interface, "table", tbl); err != nil {
		return err
	}
	// G2: wg-quick sets src_valid_mark for v4 so marked reply packets are not
	// dropped by reverse-path filtering.
	if !isV6 {
		if _, err := a.Runner.Run(ctx, "sysctl", "-q", "net.ipv4.conf.all.src_valid_mark=1"); err != nil {
			return err
		}
	}
	a.appliedDefault[isV6] = true
	return nil
}

func (a *Applier) delDefault(ctx context.Context, isV6 bool) error {
	fam := a.famFlag(isV6)
	tbl := strconv.FormatUint(uint64(a.Table), 10)
	var firstErr error
	try := func(args ...string) {
		if _, err := a.Runner.Run(ctx, "ip", args...); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	try(fam, "rule", "del", "not", "fwmark", tbl, "table", tbl)
	try(fam, "rule", "del", "table", "main", "suppress_prefixlength", "0")
	return firstErr
}

// allowedDefaults reports whether any peer AllowedIPs contains a /0 for each
// family (i.e. full-tunnel).
func allowedDefaults(cfg *profile.Config) (v4, v6 bool) {
	for _, p := range cfg.Peers {
		for _, aip := range p.AllowedIPs {
			pfx, err := netip.ParsePrefix(strings.TrimSpace(aip))
			if err != nil {
				continue
			}
			if pfx.Bits() == 0 {
				if pfx.Addr().Is6() {
					v6 = true
				} else {
					v4 = true
				}
			}
		}
	}
	return
}
