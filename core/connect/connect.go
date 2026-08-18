// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package connect orchestrates a full AmneziaWG/WireGuard connection by
// composing the autonomous mazzy-core pieces in a fail-closed order:
//
//	IPv6 guard  ->  engine (crypto+TUN)  ->  addresses+routes  ->  DNS
//
// This is the Go replacement for the bash service_run + run_quick_service path
// for wireguard/amneziawg. It shells out to no *-quick tools.
//
// Ordering rationale (parity with service_run): the IPv6 leak guard is armed
// BEFORE the interface exists, so there is never a window where IPv6 can leak
// around a half-configured tunnel. On any failure the connection unwinds every
// applied step in reverse.
package connect

import (
	"context"
	"errors"
	"fmt"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/bootrecovery"
	"github.com/mazurovn/mazzy-vpn/core/dns"
	"github.com/mazurovn/mazzy-vpn/core/engine/wireguard"
	"github.com/mazurovn/mazzy-vpn/core/guard"
	"github.com/mazurovn/mazzy-vpn/core/netexec"
	"github.com/mazurovn/mazzy-vpn/core/profile"
	"github.com/mazurovn/mazzy-vpn/core/routes"
)

// ErrBootRecoveryPending is returned by Up when the boot-recovery gate has not
// yet confirmed a safe protected egress. Network mutations are blocked until
// then (fail-closed after reboot).
var ErrBootRecoveryPending = errors.New("connect: blocked; boot recovery has not confirmed a safe state")

// Conn is a live AmneziaWG/WireGuard connection with all layers applied.
type Conn struct {
	Protocol  core.Protocol
	Interface string

	engine *wireguard.Engine
	routes *routes.Applier
	dns    *dns.Manager
	guard  *guard.Guard
	runner netexec.Runner

	mark uint32 // WireGuard socket fwmark; lets the kill-switch pass the tunnel

	ipv6GuardOn  bool
	routesOn     bool
	dnsOn        bool
	connmarkOn   bool
	killSwitchOn bool
}

// Options tune a connection.
type Options struct {
	Runner   netexec.Runner // defaults to a real ExecRunner
	LogLevel wireguard.LogLevel
	// BootGate, when non-nil, is consulted before any network change. If it
	// disallows mutations, Up fails with ErrBootRecoveryPending and performs no
	// kernel action. Nil disables the check (e.g. unit tests, dev mode).
	BootGate *bootrecovery.Gate
	// Uplink, when set, pins the encrypted-traffic egress to this physical
	// interface (U7). Empty uses the default route.
	Uplink string
}

// Up brings a wireguard/amneziawg connection fully online from a parsed config.
// It is fail-closed: if any step fails, all applied steps are reverted and the
// error is returned.
func Up(ctx context.Context, proto core.Protocol, cfg *profile.Config, opts Options) (*Conn, error) {
	if proto != core.AmneziaWG && proto != core.WireGuard {
		return nil, fmt.Errorf("connect: unsupported protocol %q", proto)
	}
	// Fail-closed boot-recovery gate: refuse to touch the network until a safe
	// state is confirmed. Checked BEFORE arming any guard or interface.
	if opts.BootGate != nil && !opts.BootGate.MutationAllowed() {
		return nil, ErrBootRecoveryPending
	}
	runner := opts.Runner
	if runner == nil {
		runner = netexec.ExecRunner{}
	}

	c := &Conn{
		Protocol: proto,
		guard:    guard.New(runner),
		runner:   runner,
	}

	// The effective fwmark that BOTH the engine socket and routing table use
	// (audit G1). Computed once, shared, so encrypted packets bypass the
	// tunnel correctly.
	mark := routes.EffectiveMark(cfg)
	c.mark = mark

	// 1. Arm IPv6 leak guard before the interface exists. We name the target
	//    interface deterministically; the engine creates that same name.
	iface := proto.Interface()
	if err := c.guard.InstallIPv6Guard(ctx, iface); err != nil {
		return nil, fmt.Errorf("arm IPv6 guard: %w", err)
	}
	c.ipv6GuardOn = true

	// 2. Bring up the crypto engine + TUN.
	eng, err := wireguard.Up(proto, cfg, mark, opts.LogLevel)
	if err != nil {
		c.unwind(ctx)
		return nil, fmt.Errorf("engine up: %w", err)
	}
	c.engine = eng
	c.Interface = eng.Interface

	// 2b. If the kernel assigned a different real interface name than the
	//     deterministic one we pre-armed the guard with (audit C4), re-affirm
	//     the IPv6 guard against the REAL name. Pre-arming with the intended
	//     name still gave leak protection during creation; this corrects the
	//     allowed interface so IPv6 through the real tunnel is not over-blocked.
	//     Fail-safe either way: a name mismatch drops IPv6, never leaks it.
	if eng.Interface != iface {
		if err := c.guard.InstallIPv6Guard(ctx, eng.Interface); err != nil {
			c.unwind(ctx)
			return nil, fmt.Errorf("re-affirm IPv6 guard for %s: %w", eng.Interface, err)
		}
	}

	// 3. Addresses + policy routing on the real interface name.
	c.routes = routes.New(runner, eng.Interface, cfg)
	c.routes.Uplink = opts.Uplink
	if err := c.routes.Up(ctx, cfg); err != nil {
		c.unwind(ctx)
		return nil, fmt.Errorf("routes up: %w", err)
	}
	c.routesOn = true

	// 3b. CONNMARK save/restore so marked reply packets keep the mark (wg-quick
	//     parity, C1-4a2). Best-effort: connectivity works without it in most
	//     setups, so a failure here only drops the connmark helper, not the
	//     whole connection.
	if err := c.guard.InstallConnmark(ctx, mark); err == nil {
		c.connmarkOn = true
	}

	// 4. DNS (optional; empty DNS is a no-op).
	c.dns = dns.New(runner, eng.Interface)
	if err := c.dns.Up(ctx, cfg.DNS); err != nil {
		c.unwind(ctx)
		return nil, fmt.Errorf("dns up: %w", err)
	}
	c.dnsOn = true

	return c, nil
}

// teardown reverts every applied layer in reverse order (best-effort) and
// returns the first error encountered. It is the single source of truth for
// both a normal Down and the unwind of a failed Up (audit N1), so the two can
// never drift out of sync.
func (c *Conn) teardown(ctx context.Context) error {
	var firstErr error
	note := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if c.dnsOn {
		note(c.dns.Down(ctx))
		c.dnsOn = false
	}
	if c.connmarkOn {
		note(c.guard.RemoveConnmark(ctx))
		c.connmarkOn = false
	}
	if c.routesOn {
		note(c.routes.Down(ctx))
		c.routesOn = false
	}
	if c.engine != nil {
		note(c.engine.Down())
		c.engine = nil
	}
	if c.ipv6GuardOn {
		note(c.guard.RemoveIPv6Guard(ctx))
		c.ipv6GuardOn = false
	}
	// NOTE: the fail-closed kill-switch is deliberately NOT removed here. It is a
	// session-level guard that must survive the intermediate Down()/Up() of an
	// in-place reconnect (otherwise a leak window reopens exactly when the tunnel
	// is down). The owner lifts it explicitly via DisarmKillSwitch once egress is
	// re-confirmed, and `recover`/`disconnect` clear the table unconditionally.
	return firstErr
}

// Down tears the connection down in reverse order. Best-effort: it attempts
// every layer and returns the first error.
func (c *Conn) Down(ctx context.Context) error {
	return c.teardown(ctx)
}

// ArmKillSwitch installs the fwmark-aware fail-closed guard so that, while the
// tunnel is being re-established after an egress drop, no plaintext can leave
// via the plain uplink (no leak window) — yet the new tunnel's own encrypted
// handshake (carrying the socket fwmark) is still allowed out so reconnection
// can succeed. It is idempotent. This is what the user-facing "Kill-switch
// (fail-closed)" setting controls.
func (c *Conn) ArmKillSwitch(ctx context.Context) error {
	if err := c.guard.InstallKillSwitch(ctx, c.mark); err != nil {
		return err
	}
	c.killSwitchOn = true
	return nil
}

// DisarmKillSwitch removes the fail-closed transition guard once egress is
// confirmed again. Best-effort and idempotent. Because the kill-switch is a
// session-level nftables table (not tied to one Conn), this may be called on a
// different Conn than the one that armed it; it clears the table regardless.
func (c *Conn) DisarmKillSwitch(ctx context.Context) error {
	err := c.guard.RemoveFailClosed(ctx)
	c.killSwitchOn = false
	return err
}

// unwind reverts partially-applied state after a failed Up. It shares the same
// reverse-order chain as Down and deliberately ignores the error (already in an
// error path).
func (c *Conn) unwind(ctx context.Context) {
	_ = c.teardown(ctx)
}
