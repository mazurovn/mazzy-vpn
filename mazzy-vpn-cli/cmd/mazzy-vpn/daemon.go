// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
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

	// A daemon owns the mutation lock for its entire lifetime. A second request
	// must therefore become a zone/resume request for that owner rather than a
	// detached child which quietly fails to acquire the lock. A --best request is
	// forwarded as the literal sentinel: the OWNER re-ranks with its live zone
	// cooldown map — pre-resolving here (no cooldown state) could bounce the
	// daemon right back into a zone it just quarantined.
	requestedZone := explicit
	if autoBest {
		requestedZone = "--best"
	}
	if snap, running, err := forwardToActiveDaemon(requestedZone); err != nil {
		fmt.Fprintf(os.Stderr, "request active daemon: %v\n", err)
		return 1
	} else if running && os.Getenv("NOTIFY_SOCKET") != "" {
		// Under a Type=notify systemd unit, forward-and-exit would end the main
		// process without READY=1 and systemd would mark the service failed.
		// Another daemon owning the VPN while the unit starts is a real conflict:
		// report it honestly and let systemd's restart policy retry.
		fmt.Fprintf(os.Stderr, "another mazzy-vpn daemon (pid %d, zone %s) already owns the VPN; stop it before using the systemd unit\n",
			snap.PID, safeDisplay(snap.Zone))
		return 1
	} else if running {
		switch {
		case requestedZone == "" || requestedZone == snap.Zone:
			fmt.Printf("active daemon pid %d resumed\n", snap.PID)
		case requestedZone == "--best":
			fmt.Printf("active daemon pid %d re-ranking for the best zone\n", snap.PID)
		default:
			fmt.Printf("active daemon pid %d switching %s → %s\n", snap.PID, safeDisplay(snap.Zone), safeDisplay(requestedZone))
		}
		return 0
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

	// Initial zone selection (the "--best" sentinel is resolved HERE, never used
	// as a profile name).
	zone := explicit
	if zone == "" || autoBest {
		if z, ok := d.pickBestLive(ctx, nil); ok {
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

	// Independent heartbeat pulse: keep the status file fresh on a fixed cadence
	// even while the main loop is inside a long connect/failover phase. Without
	// this, any phase longer than the readers' freshness window made the daemon
	// look dead: the dashboard vanished, `stop` reported "nothing to stop", and a
	// new `daemon` request fell through to the mutation lock and failed.
	hbDone := make(chan struct{})
	go func() {
		hb := time.NewTicker(5 * time.Second)
		defer hb.Stop()
		for {
			select {
			case <-hbDone:
				return
			case <-hb.C:
				rw.Touch()
				// Feed the systemd watchdog from the same pulse: if even this
				// goroutine dies, systemd (WatchdogSec) restarts the daemon.
				sdNotify("WATCHDOG=1")
			}
		}
	}()
	defer close(hbDone)
	sdNotify("READY=1")

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
	softFails := 0           // egress probes failing while the handshake is fresh
	paused := false          // true after a Disconnect intent; blocks auto-reconnect
	reconnecting := false    // an egress loss triggered a teardown; next success is a "reconnect"
	const reconnectLimit = 2 // reconnect same zone this many times
	const softFailLimit = 6  // ~1 min of probe failures with a FRESH handshake before reconnecting
	const failoverLimit = 4  // then fail over to another live zone

	// killSwitchArmed tracks whether the fail-closed guard is currently held, so
	// the stop path can always lift it even when there is no live conn to do so.
	//
	// Deliberate scope: the guard is armed only when an ESTABLISHED session loses
	// egress (protecting the reconnect gap), and stays armed across retries and
	// failovers until egress is re-confirmed. It is NOT armed while the daemon
	// has never had a working tunnel this session — sealing the host because the
	// first connect attempt failed would cut all connectivity with no leak to
	// prevent, and would strand the user with no VPN and no internet.
	killSwitchArmed := false
	// nextAttempt gates reconnect attempts. It replaces the previous in-loop
	// time.Sleep(backoff(...)) — sleeping inside the tick handler blocked signal
	// handling, intent processing and (before the pulse goroutine) the heartbeat
	// for up to a minute per iteration.
	var nextAttempt time.Time
	// cooldown remembers zones that repeatedly failed EGRESS recently. A server
	// can answer ICMP yet not route traffic (provider throttling WireGuard); the
	// old failover happily re-picked exactly that zone forever.
	cooldown := map[string]time.Time{}

	stop := func() int {
		d.logf("stopping...")
		sdNotify("STOPPING=1")
		// Teardown must run even when ctx is already cancelled (SIGTERM path).
		sctx := context.WithoutCancel(ctx)
		if conn != nil {
			if set.KillSwitch {
				_ = conn.DisarmKillSwitch(sctx)
			}
			_ = conn.Down(sctx)
		} else if killSwitchArmed {
			// No live conn but the guard is held: clear the table directly so
			// the host is not left fail-closed after the daemon exits.
			_ = guard.New(netexec.ExecRunner{}).RemoveFailClosed(sctx)
		}
		rw.SetState(runstatus.StateDown, "", "")
		rw.Close()
		nfy.Disconnected(zone)
		_ = st.SetDesired(core.DesiredDown)
		return 0
	}

	for {
		select {
		case <-sig:
			return stop()
		case <-ctx.Done():
			return stop()
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
			rw.SetStealth(score, stSig.EgressCountry, stSig.EgressCity)
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
						nfy.Disconnected(zone)
					}
					// Publish PAUSED (not "down"): the daemon is alive and holding
					// the tunnel down on purpose. Readers previously could not tell
					// a paused daemon from a dead one.
					rw.SetState(runstatus.StatePaused, "", "")
					paused = true
					continue
				}
				if di.Desired == "up" {
					if paused {
						d.logf("reconnect requested; resuming")
						paused = false
						fails = 0
						softFails = 0
						reconnecting = false
						nextAttempt = time.Time{}
					}
					// A "--best" request re-ranks HERE, inside the daemon, so its
					// live cooldown map is honored. Pre-resolving in the requesting
					// process (which has no cooldown state) could bounce the daemon
					// right back into the zone it just fled.
					switchTo := ""
					if di.Zone == "--best" {
						if nz, okz := d.pickBestLive(ctx, cooldown); okz && nz != zone {
							switchTo = nz
						}
					} else if di.Zone != "" && di.Zone != zone {
						switchTo = di.Zone
					}
					if switchTo != "" {
						d.logf("zone switch requested: %s → %s", zone, switchTo)
						if conn != nil {
							_ = conn.Down(ctx)
							conn = nil
						}
						zone = switchTo
						rw.SetZone(zone)
						fails = 0
						softFails = 0
						reconnecting = false
						nextAttempt = time.Time{}
					}
				}
			}
			// While paused (Disconnect intent) do nothing until intent resumes.
			if paused {
				continue
			}
			if conn == nil {
				// Backoff without sleeping: skip ticks until the retry moment.
				if time.Now().Before(nextAttempt) {
					continue
				}
				var ok bool
				conn, ok = d.connectZone(ctx, zone)
				if ok {
					// Egress CONFIRMED — only now is it safe to lift the fail-closed
					// guard (the old code disarmed on a non-nil conn even when the
					// egress was never verified: a plaintext leak window).
					if killSwitchArmed && set.KillSwitch {
						_ = conn.DisarmKillSwitch(ctx)
						killSwitchArmed = false
					}
					if reconnecting {
						rw.Reconnected()
						reconnecting = false
					}
					fails = 0
					softFails = 0
					delete(cooldown, zone)
					continue
				}
				// Not confirmed. A non-nil conn (link up, egress unverified) stays
				// under watch: the health ticks below will confirm or fail it.
				fails++
				nextAttempt = time.Now().Add(backoff(fails))
				if fails >= failoverLimit {
					if nz, changed := d.failoverZone(ctx, zone, cooldown); changed {
						zone = nz
						rw.SetZone(zone)
						fails = 0
						nextAttempt = time.Time{}
					}
				}
				continue
			}
			tickStart := time.Now()
			s := lc.Check(ctx, conn.Interface)
			if s.Protected() {
				if killSwitchArmed && set.KillSwitch {
					_ = conn.DisarmKillSwitch(ctx)
					killSwitchArmed = false
				}
				if reconnecting {
					rw.Reconnected()
					reconnecting = false
				}
				fails = 0
				softFails = 0
				delete(cooldown, zone)
				rw.SetState(runstatus.StateProtected, conn.Interface, s.EgressIP)
				// Publish link facts the unprivileged dashboard cannot read itself.
				hsAge := int64(0)
				if age, hsOK := conn.HandshakeAge(); hsOK {
					hsAge = int64(age.Seconds())
				}
				rx, tx, _ := conn.Transfer()
				rw.SetLinkHealth(hsAge, rx, tx)
				rw.TickPing(int(time.Since(tickStart).Milliseconds()), d.serverPingMS(ctx), true)
				continue
			}
			rw.Tick(0, false)
			// Handshake-aware distinction: if the WireGuard handshake is FRESH the
			// server is alive and answering crypto — the probe endpoints themselves
			// are likely blocked/degraded. Tolerate that far longer before tearing
			// down a working tunnel (the old behavior reconnect-stormed a healthy
			// tunnel whenever the single probe URL was blocked by the ISP).
			if age, hsOK := conn.HandshakeAge(); hsOK && age < 3*time.Minute {
				softFails++
				d.logf("egress probe failed but handshake fresh (%ds ago): %s (%d/%d)",
					int(age.Seconds()), s.Reason, softFails, softFailLimit)
				rw.Error("probe degraded (handshake fresh): " + s.Reason)
				if softFails < softFailLimit {
					continue
				}
				d.logf("all egress probes failing for %d ticks despite fresh handshake; treating as real loss", softFails)
				// Escalate straight to the reconnect path — the soft window already
				// consumed the patience the reconnectLimit gate would re-impose.
				fails = reconnectLimit - 1
			}
			fails++
			if fails < reconnectLimit {
				d.logf("egress check failed (%d): %s", fails, s.Reason)
				rw.Error(s.Reason)
				continue
			}
			d.logf("⟳ egress lost; reconnecting %s...", zone)
			rw.SetState(runstatus.StateReconnect, conn.Interface, "")
			rw.Error("egress lost: " + s.Reason)
			nfy.Reconnecting(zone, s.Reason)
			// Arm the fail-closed kill-switch before tearing the tunnel down so no
			// plaintext leaks during the reconnect gap (parity with the foreground
			// connect path). It is lifted only after egress is re-confirmed.
			if set.KillSwitch {
				if err := conn.ArmKillSwitch(ctx); err != nil {
					d.logf("warning: could not arm kill-switch: %v", err)
				} else {
					killSwitchArmed = true
				}
			}
			_ = conn.Down(ctx)
			conn = nil
			reconnecting = true
			softFails = 0
			nextAttempt = time.Now().Add(backoff(fails))
			// After several failures, fail over to another live zone (excluding
			// zones that recently failed egress, this one included).
			if fails >= failoverLimit {
				if nz, changed := d.failoverZone(ctx, zone, cooldown); changed {
					zone = nz
					rw.SetZone(zone)
					fails = 0
					nextAttempt = time.Time{}
				}
			}
		}
	}
}

// forwardToActiveDaemon turns a second connect request into an intent for the
// current lock owner. It deliberately does not create another daemon or take
// the mutation lock. The snapshot is returned for an honest caller message.
func forwardToActiveDaemon(zone string) (runstatus.Snapshot, bool, error) {
	snap, running := daemonRunning()
	if !running {
		return runstatus.Snapshot{}, false, nil
	}
	if err := writeDesired(zone, "up"); err != nil {
		return runstatus.Snapshot{}, true, err
	}
	return snap, true, nil
}

// daemonState holds the shared daemon dependencies.
type daemonState struct {
	st  *state.Store
	lc  *livecheck.Checker
	nfy *notify.Notifier
	rw  *runstatus.Writer
	// endpointHost is the current zone's server host (set by connectZone), and
	// png pings it via the PHYSICAL uplink each healthy tick — the honest link
	// metric for the dashboard graph (the HTTPS probe duration bundles
	// TCP+TLS+HTTP and overstates latency several-fold).
	endpointHost string
	png          *measure.Pinger
	pingBusy     atomic.Bool  // one in-flight ping at a time
	lastPingMS   atomic.Int32 // most recent completed RTT (0 = none yet)
}

// serverPingMS returns the most recent completed ICMP RTT to the current
// server and kicks off the next measurement in the background. The ping (up
// to 2s) deliberately never runs inline in the tick handler: blocking there
// would delay SIGTERM/intent handling on every healthy tick.
func (d *daemonState) serverPingMS(ctx context.Context) int {
	if d.endpointHost == "" {
		return 0
	}
	if d.png == nil {
		d.png = measure.NewPinger()
		d.png.Timeout = 2 * time.Second
		d.png.Interface = settingsUplink()
	}
	if d.pingBusy.CompareAndSwap(false, true) {
		host := d.endpointHost
		go func() {
			defer d.pingBusy.Store(false)
			if ms, ok := d.png.Ping(ctx, host); ok {
				d.lastPingMS.Store(int32(ms + 0.5))
			} else {
				d.lastPingMS.Store(0)
			}
		}()
	}
	return int(d.lastPingMS.Load())
}

func (d *daemonState) logf(format string, a ...any) {
	// Sanitize any string args (zone/egress names may be user-controlled) so a
	// crafted value cannot inject terminal escapes into the daemon log line.
	for i, v := range a {
		if s, ok := v.(string); ok {
			a[i] = safeDisplay(s)
		}
	}
	// Full date in the stamp: a long-lived daemon's log previously showed only
	// clock time, so "which day did this happen" was unanswerable.
	fmt.Printf("%s "+format+"\n", append([]any{time.Now().Format("2006-01-02 15:04:05")}, a...)...)
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
	// Remember the server host for the per-tick ICMP metric (graph source).
	if host, _, herr := net.SplitHostPort(cfg.Endpoint()); herr == nil {
		d.endpointHost = host
	} else {
		d.endpointHost = cfg.Endpoint()
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
		d.rw.SetProtocol(proto.Title())
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
// the best ICMP-alive one. avoid maps zone→cooldown-expiry for zones that
// recently failed EGRESS (ICMP-alive but not routing); they are skipped unless
// that would leave no candidates at all — a degraded pick still beats none.
func (d *daemonState) pickBestLive(ctx context.Context, avoid map[string]time.Time) (string, bool) {
	cat := newCatalog()
	targets, err := targetsFromCatalog(cat)
	if err != nil || len(targets) == 0 {
		return "", false
	}
	if len(avoid) > 0 {
		now := time.Now()
		kept := targets[:0:0]
		for _, t := range targets {
			if until, bad := avoid[t.Name]; bad && now.Before(until) {
				continue
			}
			kept = append(kept, t)
		}
		if len(kept) > 0 {
			targets = kept
		}
	}
	ranked := newMeasurer().RankBest(ctx, targets)
	best, ok := measure.BestAlive(ranked)
	if !ok {
		return "", false
	}
	return best.Name, true
}

// zoneCooldown is how long a zone that repeatedly failed EGRESS is quarantined
// from failover selection.
const zoneCooldown = 10 * time.Minute

// failoverZone quarantines the failing zone and picks a replacement. It ranks
// live zones first; when ranking finds nothing (all servers down — or
// measurement itself impaired, e.g. while the kill-switch is armed), it falls
// back to BLINDLY trying the next catalog zone rather than hammering the same
// dead one forever (the "Belgium loop": 7 minutes of silent non-failover).
// Every decision is logged so the operator can see WHY.
func (d *daemonState) failoverZone(ctx context.Context, zone string, cooldown map[string]time.Time) (string, bool) {
	cooldown[zone] = time.Now().Add(zoneCooldown)
	d.logf("zone %s quarantined for %s after repeated egress failures", zone, zoneCooldown)
	if nz, ok := d.pickBestLive(ctx, cooldown); ok && nz != zone {
		d.logf("⟳ failing over: %s → %s (ranked best live)", zone, nz)
		return nz, true
	}
	if nz, ok := d.nextCatalogZone(zone, cooldown); ok {
		d.logf("⟳ failover: ranking found no live zone (all down or measurement blocked); blind fallback %s → %s", zone, nz)
		return nz, true
	}
	d.logf("failover: no alternative zone available; retrying %s", zone)
	return zone, false
}

// nextCatalogZone returns the first catalog zone that is neither the current
// one nor quarantined — the last-resort candidate when ranking is impossible.
func (d *daemonState) nextCatalogZone(zone string, cooldown map[string]time.Time) (string, bool) {
	targets, err := targetsFromCatalog(newCatalog())
	if err != nil {
		return "", false
	}
	now := time.Now()
	for _, t := range targets {
		if t.Name == zone {
			continue
		}
		if until, bad := cooldown[t.Name]; bad && now.Before(until) {
			continue
		}
		return t.Name, true
	}
	return "", false
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
