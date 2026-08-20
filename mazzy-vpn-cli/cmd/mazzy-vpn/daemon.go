// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
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
	"github.com/mazurovn/mazzy-vpn/core/guard"
	"github.com/mazurovn/mazzy-vpn/core/livecheck"
	"github.com/mazurovn/mazzy-vpn/core/lock"
	"github.com/mazurovn/mazzy-vpn/core/measure"
	"github.com/mazurovn/mazzy-vpn/core/netexec"
	"github.com/mazurovn/mazzy-vpn/core/notify"
	"github.com/mazurovn/mazzy-vpn/core/runstatus"
	"github.com/mazurovn/mazzy-vpn/core/settings"
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
	background := false // persistent: survives menu quit / terminal close
	session := false    // detached but tied to the menu session (stopped on quit)
	for _, a := range args {
		switch {
		case a == "--best" || a == "--auto":
			autoBest = true
		case a == "--background" || a == "--detach":
			background = true
		case a == "--session":
			session = true
		case a != "" && a[0] != '-':
			explicit = a
		}
	}

	// Detached modes: fork a copy into a new session and return, so the caller
	// (menu/TUI) lands back in the dashboard while the VPN keeps running. The
	// detached child re-enters here with the marker set and runs the loop below.
	// --background persists (heartbeat Background=true); --session is stopped when
	// the user quits the menu.
	if detached, derr := maybeDaemonize(background || session); derr != nil {
		fmt.Fprintln(os.Stderr, "background:", derr)
		return 1
	} else if detached {
		return 0
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
	// Honor the same user settings as the foreground connect path (audit: the
	// daemon previously ignored Notifications, so the toggle did nothing in the
	// recommended production mode).
	set := settings.NewStore().Load()
	if !set.Notifications {
		nfy.Enabled = false
	}
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

	// Heartbeat: the (root) daemon publishes a world-readable status file so the
	// unprivileged menu/TUI can render a live dashboard without holding the
	// terminal. It is removed on clean shutdown (see the stop path below).
	// A --session daemon publishes the same heartbeat as --background; the only
	// difference is lifecycle (the menu stops sessions on quit, backgrounds
	// persist). Persist the persistence bit so the reader can distinguish them.
	rw := runstatus.NewWriter(zone, "", "", background)
	d.rw = rw
	defer rw.Close()

	// Clear any stale down-intent left by a previous session's Disconnect so this
	// freshly-started daemon does not immediately pause itself on its first tick.
	_ = st.SetDesired(core.DesiredUp)
	_ = writeDesired(zone, "up")

	conn, _ := d.connectZone(ctx, zone)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	// Periodic stealth monitor (Task 2): re-checks detection score every few
	// minutes and notifies if it drops, so a silently-degraded exit is caught.
	stealthTicker := time.NewTicker(5 * time.Minute)
	defer stealthTicker.Stop()
	lastStealth := -1

	fails := 0
	paused := false          // true after a Disconnect intent; blocks auto-reconnect
	const reconnectLimit = 2 // reconnect same zone this many times
	const failoverLimit = 4  // then fail over to another live zone

	// killSwitchArmed tracks whether the fail-closed guard is currently held, so
	// the stop path can always lift it even when there is no live conn to do so.
	killSwitchArmed := false

	for {
		select {
		case <-sig:
			d.logf("stopping...")
			if conn != nil {
				if set.KillSwitch {
					_ = conn.DisarmKillSwitch(ctx)
				}
				_ = conn.Down(ctx)
			} else if killSwitchArmed {
				// No live conn but the guard is held: clear the table directly so
				// the host is not left fail-closed after the daemon exits.
				_ = guard.New(netexec.ExecRunner{}).RemoveFailClosed(ctx)
			}
			rw.SetState(runstatus.StateDown, "", "")
			rw.Close()
			nfy.Disconnected(zone)
			_ = st.SetDesired(core.DesiredDown)
			return 0
		case <-stealthTicker.C:
			if conn == nil {
				continue
			}
			// Note: do not shadow the OS signal channel `sig` above.
			stSig := gatherStealthSignal(ctx)
			if stSig.EgressCountry == "" {
				continue
			}
			score := stealthScoreOf(stSig)
			d.logf("stealth score: %d (egress %s)", score, stSig.EgressCountry)
			if lastStealth >= 0 && score < lastStealth-15 {
				msg := fmt.Sprintf("stealth dropped %d→%d (more detectable)", lastStealth, score)
				d.nfy.Failed(zone, msg)
				rw.Error(msg)
			}
			lastStealth = score
		case <-ticker.C:
			// Honor an intent written by the (unprivileged) TUI/menu (ADR-0006 D2).
			if di, ok := readDesired(); ok {
				if di.Desired == "down" {
					// Paused: tear down if up, and (critically) do NOT auto-reconnect
					// until the intent flips back to "up". Without this guard the loop
					// below would immediately revive the tunnel, making the user's
					// Disconnect ineffective while a daemon runs.
					if conn != nil {
						d.logf("disconnect requested; pausing auto-reconnect")
						_ = conn.Down(ctx)
						conn = nil
						rw.SetState(runstatus.StateDown, "", "")
						nfy.Disconnected(zone)
					}
					paused = true
					continue
				}
				if di.Desired == "up" {
					if paused {
						d.logf("reconnect requested; resuming")
						paused = false
						fails = 0
					}
					if di.Zone != "" && di.Zone != "--best" && di.Zone != zone {
						d.logf("zone switch requested: %s → %s", zone, di.Zone)
						if conn != nil {
							_ = conn.Down(ctx)
							conn = nil
						}
						zone = di.Zone
						rw.SetZone(zone)
						fails = 0
					}
				}
			}
			// While paused (Disconnect intent) do nothing until intent resumes.
			if paused {
				continue
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
			tickStart := time.Now()
			s := lc.Check(ctx, conn.Interface)
			if s.Protected() {
				fails = 0
				rw.SetState(runstatus.StateProtected, conn.Interface, s.EgressIP)
				rw.Tick(int(time.Since(tickStart).Milliseconds()), true)
				continue
			}
			rw.Tick(0, false)
			fails++
			if fails < reconnectLimit {
				d.logf("egress check failed (%d)", fails)
				rw.Error(s.Reason)
				continue
			}
			d.logf("⟳ egress lost; reconnecting %s...", zone)
			rw.SetState(runstatus.StateReconnect, conn.Interface, "")
			rw.Error("egress lost: " + s.Reason)
			nfy.Reconnecting(zone, s.Reason)
			// Arm the fail-closed kill-switch before tearing the tunnel down so no
			// plaintext leaks during the reconnect gap (parity with the foreground
			// connect path). connectZone lifts it once egress is re-confirmed.
			if set.KillSwitch {
				if err := conn.ArmKillSwitch(ctx); err != nil {
					d.logf("warning: could not arm kill-switch: %v", err)
				} else {
					killSwitchArmed = true
				}
			}
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
			// Egress re-confirmed inside connectZone; lift the kill-switch.
			if set.KillSwitch && conn != nil {
				_ = conn.DisarmKillSwitch(ctx)
				killSwitchArmed = false
			}
			if conn != nil {
				rw.Reconnected()
			}
		}
	}
}

// daemonState holds the shared daemon dependencies.
type daemonState struct {
	st  *state.Store
	lc  *livecheck.Checker
	nfy *notify.Notifier
	rw  *runstatus.Writer
}

func (d *daemonState) logf(format string, a ...any) {
	// Sanitize any string args (zone/egress names may be user-controlled) so a
	// crafted value cannot inject terminal escapes into the daemon log line.
	for i, v := range a {
		if s, ok := v.(string); ok {
			a[i] = safeDisplay(s)
		}
	}
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
	c, err := connect.Up(ctx, proto, cfg, connect.Options{LogLevel: wireguard.LogError, Uplink: settingsUplink()})
	if err != nil {
		d.logf("connect failed: %v", err)
		d.nfy.Failed(name, err.Error())
		return nil, false
	}
	if d.rw != nil {
		d.rw.SetZone(name)
		d.rw.SetState(runstatus.StateConnecting, c.Interface, "")
	}
	snap := d.lc.WaitProtected(ctx, c.Interface, 20*time.Second)
	if snap.Protected() {
		d.logf("✔ protected. egress=%s", snap.EgressIP)
		d.nfy.Connected(name, snap.EgressIP)
		if d.rw != nil {
			d.rw.SetState(runstatus.StateProtected, c.Interface, snap.EgressIP)
		}
		_ = d.st.Write(state.State{Protocol: proto, Profile: name, Desired: core.DesiredUp, Mode: core.ModeNormal})
		return c, true
	}
	d.logf("⚠ egress not confirmed: %s", snap.Reason)
	d.nfy.Failed(name, snap.Reason)
	if d.rw != nil {
		d.rw.SetState(runstatus.StateLinkUp, c.Interface, "")
		d.rw.Error("egress not confirmed: " + snap.Reason)
	}
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
