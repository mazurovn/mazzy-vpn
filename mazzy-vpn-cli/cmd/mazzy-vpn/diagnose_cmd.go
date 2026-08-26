// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/diagnose"
	"github.com/mazurovn/mazzy-vpn/core/guard"
	"github.com/mazurovn/mazzy-vpn/core/netexec"
	"github.com/mazurovn/mazzy-vpn/core/routes"
	"github.com/mazurovn/mazzy-vpn/core/settings"
	"github.com/mazurovn/mazzy-vpn/core/livecheck"
	"github.com/mazurovn/mazzy-vpn/core/measure"
	"github.com/mazurovn/mazzy-vpn/core/netadapter"
	"github.com/mazurovn/mazzy-vpn/core/pathtrace"
	"github.com/mazurovn/mazzy-vpn/core/profile"
)

// gatherSignal collects real observations for root-cause analysis.
func gatherSignal(ctx context.Context) diagnose.Signal {
	s := diagnose.Signal{}

	// Daemon heartbeat: paused / reconnect-looping / wedged states explain the
	// user-visible "VPN does nothing" better than any network probe, and were
	// previously invisible to diagnose.
	if snap, ok := daemonRunning(); ok {
		s.DaemonAlive = true
		s.DaemonState = string(snap.State)
		s.DaemonHeartbeatAge = int64(snap.HeartbeatAge().Seconds())
		s.DaemonReconnects = snap.Reconnects
		if recent := snap.RecentErrors(1); len(recent) > 0 {
			s.DaemonLastError = recent[0].Reason
		}
	}

	// Uplink.
	if adapters, err := netadapter.List(); err == nil {
		if rec, _, ok := netadapter.Recommend(adapters); ok && rec.HasRoutableIPv4() {
			s.HasUplink = true
			s.UplinkName = rec.Name
		}
		managed := map[string]bool{}
		for _, m := range core.ManagedInterfaces() {
			managed[m] = true
		}
		for _, a := range adapters {
			// Any foreign VPN-looking virtual interface conflicts — the old check
			// only knew tun0/tun1 and was blind to wg0, tailscale0, proton0, ...
			if a.Virtual && a.Up && !managed[a.Name] && looksLikeForeignVPN(a.Name) {
				s.ConflictVPN = a.Name
			}
		}
	}

	// Plain internet (off-tunnel) via a quick HTTP probe.
	s.InternetOK = plainInternetOK(ctx)
	s.DNSOK = dnsOK(ctx)

	// Firewall/routing residue: our own leftover state that can seal the host.
	gatherGuardResidue(ctx, &s)

	// Tunnel + egress.
	s.TunnelIface = detectLiveInterface()
	if s.TunnelIface != "" {
		snap := livecheck.New().Check(ctx, s.TunnelIface)
		s.TunnelLinkUp = snap.LinkUp
		s.EgressOK = snap.EgressOK
		s.EgressIP = snap.EgressIP
	}

	// Servers: how many imported, any alive.
	cat := newCatalog()
	s.ProfilesImported = cat.Count()
	if targets, err := targetsFromCatalog(cat); err == nil && len(targets) > 0 {
		ranked := newMeasurer().RankBest(ctx, targets)
		if best, ok := measure.BestAlive(ranked); ok {
			s.AnyServerAlive = best.ICMPAlive
		}
		// If connected, check whether the current server is alive.
		if cur, err := newStore().Read(); err == nil {
			for _, r := range ranked {
				if r.Name == cur.Profile {
					s.ServerName = r.Name
					s.ServerAlive = r.ICMPAlive
				}
			}
		}
	}
	return s
}

// gatherGuardResidue inspects OUR nftables tables (root only), policy-routing
// rules and resolv.conf for leftover state that blocks the internet without a
// tunnel — the "kill-switch sealed the host" incident class.
func gatherGuardResidue(ctx context.Context, s *diagnose.Signal) {
	s.KillSwitchByCfg = settings.NewStore().Load().KillSwitch
	r := netexec.ExecRunner{}

	// nft needs root; without it we honestly report "not checked".
	if os.Geteuid() == 0 {
		s.GuardChecked = true
		if out, err := r.Run(ctx, "nft", "list", "tables"); err == nil {
			for _, tbl := range []string{guard.IPv6GuardTable, guard.TransitionGuardTable, guard.ConnmarkTable} {
				if strings.Contains(out, tbl) {
					s.GuardTables = append(s.GuardTables, tbl)
					if tbl == guard.TransitionGuardTable {
						s.KillSwitchOn = true
					}
				}
			}
		}
	}

	// Policy rules are visible without root.
	mark := strconv.Itoa(routes.DefaultMark)
	count := 0
	for _, fam := range []string{"-4", "-6"} {
		if out, err := r.Run(ctx, "ip", fam, "rule", "show"); err == nil {
			for _, line := range strings.Split(out, "\n") {
				if strings.Contains(line, "fwmark") && strings.Contains(line, mark) ||
					strings.Contains(line, "suppress_prefixlength 0") {
					count++
				}
			}
		}
	}
	s.PolicyRules = count

	// DNS pointing at the tunnel resolver while no tunnel exists = dead lookups.
	if detectLiveInterface() == "" {
		if data, err := os.ReadFile("/etc/resolv.conf"); err == nil {
			for _, iface := range core.ManagedInterfaces() {
				if strings.Contains(string(data), iface) {
					s.StaleTunnelDNS = true
				}
			}
		}
	}
}

// looksLikeForeignVPN reports whether an interface name matches a known VPN
// naming scheme (another client's tunnel that can steal the default route).
func looksLikeForeignVPN(name string) bool {
	for _, p := range []string{"tun", "wg", "tailscale", "proton", "nordlynx", "ipsec", "ppp", "outline"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func plainInternetOK(ctx context.Context) bool {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, "https://api.ipify.org", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	// Drain then close so the connection can be reused and nothing leaks.
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode == 200
}

func dnsOK(ctx context.Context) bool {
	cctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupHost(cctx, "one.one.one.one")
	return err == nil && len(ips) > 0
}

// cmdDiagnose runs the smart root-cause analyzer and prints ranked problems +
// fixes. This is the "what's wrong and how to fix it" command.
func cmdDiagnose(ctx context.Context, args []string) int {
	fmt.Println(translator().T("cli.diagnose.analyzing"))
	sig := gatherSignal(ctx)
	rep := diagnose.Analyze(sig)

	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rep)
		if !rep.Healthy() {
			return 1
		}
		return 0
	}

	fmt.Println()
	for _, p := range rep.Problems {
		glyph := "•"
		switch p.Severity {
		case diagnose.Critical:
			glyph = "✖"
		case diagnose.Warn:
			glyph = "▲"
		case diagnose.Info:
			glyph = "●"
		}
		fmt.Printf("%s [%s] %s\n", glyph, p.Level, safeDisplay(p.Title))
		if p.Cause != "" {
			fmt.Printf("    why: %s\n", safeDisplay(p.Cause))
		}
		if p.Fix != "" {
			fmt.Printf("    fix: %s\n", safeDisplay(p.Fix))
		}
	}
	fmt.Printf("\n→ %s\n", safeDisplay(rep.Summary))
	if !rep.Healthy() {
		return 1
	}
	return 0
}

// cmdTrace shows the packet path for the current or a named zone's server.
func cmdTrace(ctx context.Context, args []string) int {
	// Determine the endpoint: current connection, or a named zone.
	var endpoint string
	iface := detectLiveInterface()
	cat := newCatalog()

	name := ""
	for _, a := range args {
		if a != "" && a[0] != '-' {
			name = a
		}
	}
	if name == "" {
		if cur, err := newStore().Read(); err == nil {
			name = cur.Profile
		}
	}
	if name != "" {
		if e, err := cat.Get(name); err == nil {
			if data, err := os.ReadFile(e.File); err == nil {
				if cfg, err := profile.Parse(string(data)); err == nil {
					endpoint = cfg.Endpoint()
				}
			}
		}
	}
	if endpoint == "" {
		fmt.Fprintln(os.Stderr, "no endpoint to trace; specify a zone: mazzy-vpn trace <zone>")
		return 2
	}

	tr := &pathtrace.Tracer{
		Resolver: net.DefaultResolver,
		Pinger:   measure.NewPinger(),
		Link:     func(i string) bool { _, err := net.InterfaceByName(i); return err == nil },
		Egress: func(c context.Context, i string) (string, error) {
			snap := livecheck.New().Check(c, i)
			if snap.EgressOK {
				return snap.EgressIP, nil
			}
			return "", fmt.Errorf("no egress")
		},
	}
	result := tr.Run(ctx, endpoint, iface)

	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
		return 0
	}

	fmt.Printf("Packet path to %s:\n\n", result.Endpoint)
	for _, st := range result.Steps {
		glyph := "✔"
		switch st.Status {
		case pathtrace.Fail:
			glyph = "✖"
		case pathtrace.Warn:
			glyph = "▲"
		}
		fmt.Printf("  %s %-22s %s (%d ms)\n", glyph, safeDisplay(st.Name), safeDisplay(st.Detail), st.Duration)
	}
	if result.Healthy() {
		fmt.Println("\n→ Packet path is clear.")
	} else {
		fmt.Println("\n→ Packet path broken — see the ✖ step above.")
		return 1
	}
	return 0
}
