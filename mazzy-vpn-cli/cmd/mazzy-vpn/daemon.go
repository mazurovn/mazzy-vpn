// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/connect"
	"github.com/mazurovn/mazzy-vpn/core/engine/wireguard"
	"github.com/mazurovn/mazzy-vpn/core/livecheck"
	"github.com/mazurovn/mazzy-vpn/core/lock"
	"github.com/mazurovn/mazzy-vpn/core/measure"
	"github.com/mazurovn/mazzy-vpn/core/notify"
	"github.com/mazurovn/mazzy-vpn/core/state"
)

// cmdDaemon runs the VPN persistently with self-healing: it connects to a zone
// (a name, or --best to auto-pick a live one), verifies egress, auto-reconnects
// on drops, and — if a zone keeps failing — fails over to the next best LIVE
// zone. This is the entry point for the systemd service.
//
// Usage: sudo mazzy-vpn daemon <ZONE_NAME|--best>
func cmdDaemon(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sudo mazzy-vpn daemon <ZONE_NAME|--best>")
		return 2
	}
	if !requireRoot("daemon") {
		return 1
	}

	explicit := ""
	autoBest := false
	for _, a := range args {
		switch {
		case a == "--best" || a == "--auto":
			autoBest = true
		case a != "" && a[0] != '-':
			explicit = a
		}
	}

	mu, err := lock.Acquire(lockDir())
	if err != nil {
		fmt.Fprintln(os.Stderr, "another mazzy-vpn operation is in progress")
		return 1
	}
	defer mu.Unlock()

	st := newStore()
	lc := livecheck.New()
	nfy := notify.New()
	d := &daemonState{st: st, lc: lc, nfy: nfy}

	// Initial zone selection.
	zone := explicit
	if zone == "" || autoBest {
		if z, ok := d.pickBestLive(ctx); ok {
			zone = z
		} else if zone == "" {
			fmt.Fprintln(os.Stderr, "no live zone found; check your connection (mazzy-vpn netdiag)")
			return 1
		}
	}

	conn, _ := d.connectZone(ctx, zone)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	fails := 0
	const reconnectLimit = 2 // reconnect same zone this many times
	const failoverLimit = 4  // then fail over to another live zone

	for {
		select {
		case <-sig:
			d.logf("stopping...")
			if conn != nil {
				_ = conn.Down(ctx)
			}
			nfy.Disconnected(zone)
			_ = st.SetDesired(core.DesiredDown)
			return 0
		case <-ticker.C:
			// Honor an intent written by the (unprivileged) TUI (ADR-0006 D2).
			if di, ok := readDesired(); ok {
				if di.Desired == "down" && conn != nil {
					d.logf("TUI requested disconnect")
					_ = conn.Down(ctx)
					conn = nil
					nfy.Disconnected(zone)
					continue
				}
				if di.Desired == "up" && di.Zone != "" && di.Zone != "--best" && di.Zone != zone {
					d.logf("TUI requested zone switch: %s → %s", zone, di.Zone)
					if conn != nil {
						_ = conn.Down(ctx)
						conn = nil
					}
					zone = di.Zone
					fails = 0
				}
			}
			if conn == nil {
				time.Sleep(backoff(fails))
				conn, _ = d.connectZone(ctx, zone)
				if conn == nil {
					fails++
				} else {
					fails = 0
				}
				// If we keep failing this zone, fail over.
				if conn == nil && fails >= failoverLimit {
					if nz, ok := d.pickBestLive(ctx); ok && nz != zone {
						d.logf("⟳ failing over: %s → %s", zone, nz)
						zone = nz
						fails = 0
					}
				}
				continue
			}
			s := lc.Check(ctx, conn.Interface)
			if s.Protected() {
				fails = 0
				continue
			}
			fails++
			if fails < reconnectLimit {
				d.logf("egress check failed (%d)", fails)
				continue
			}
			d.logf("⟳ egress lost; reconnecting %s...", zone)
			nfy.Reconnecting(zone, s.Reason)
			_ = conn.Down(ctx)
			time.Sleep(backoff(fails))

			// After several failures, fail over to another live zone.
			if fails >= failoverLimit {
				if nz, ok := d.pickBestLive(ctx); ok && nz != zone {
					d.logf("⟳ zone %s unhealthy; failing over to %s", zone, nz)
					zone = nz
					fails = 0
				}
			}
			conn, _ = d.connectZone(ctx, zone)
		}
	}
}

// daemonState holds the shared daemon dependencies.
type daemonState struct {
	st  *state.Store
	lc  *livecheck.Checker
	nfy *notify.Notifier
}

func (d *daemonState) logf(format string, a ...any) {
	fmt.Printf("%s "+format+"\n", append([]any{time.Now().Format("15:04:05")}, a...)...)
}

// connectZone loads and connects a managed zone by name, verifying egress.
func (d *daemonState) connectZone(ctx context.Context, name string) (*connect.Conn, bool) {
	cat := newCatalog()
	entry, err := cat.Get(name)
	if err != nil {
		d.logf("profile %q not found", name)
		return nil, false
	}
	proto, cfg, err := loadProfile(entry.File)
	if err != nil {
		d.logf("load %s: %v", name, err)
		return nil, false
	}
	d.logf("connecting %s (%s)...", name, proto.Title())
	c, err := connect.Up(ctx, proto, cfg, connect.Options{LogLevel: wireguard.LogError})
	if err != nil {
		d.logf("connect failed: %v", err)
		d.nfy.Failed(name, err.Error())
		return nil, false
	}
	snap := d.lc.WaitProtected(ctx, c.Interface, 20*time.Second)
	if snap.Protected() {
		d.logf("✔ protected. egress=%s", snap.EgressIP)
		d.nfy.Connected(name, snap.EgressIP)
		_ = d.st.Write(state.State{Protocol: proto, Profile: name, Desired: core.DesiredUp, Mode: core.ModeNormal})
		return c, true
	}
	d.logf("⚠ egress not confirmed: %s", snap.Reason)
	d.nfy.Failed(name, snap.Reason)
	return c, false
}

// pickBestLive ranks all AmneziaWG zones through the physical uplink and returns
// the best ICMP-alive one.
func (d *daemonState) pickBestLive(ctx context.Context) (string, bool) {
	cat := newCatalog()
	targets, err := targetsFromCatalog(cat)
	if err != nil || len(targets) == 0 {
		return "", false
	}
	ranked := newMeasurer().RankBest(ctx, targets)
	best, ok := measure.BestAlive(ranked)
	if !ok {
		return "", false
	}
	return best.Name, true
}

// backoff returns an increasing but bounded delay for reconnect attempts.
func backoff(fails int) time.Duration {
	switch {
	case fails <= 0:
		return 0
	case fails == 1:
		return 2 * time.Second
	case fails == 2:
		return 5 * time.Second
	case fails <= 4:
		return 15 * time.Second
	default:
		return 30 * time.Second
	}
}
