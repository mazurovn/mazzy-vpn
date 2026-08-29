// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/catalog"
	"github.com/mazurovn/mazzy-vpn/core/connect"
	"github.com/mazurovn/mazzy-vpn/core/livecheck"
	"github.com/mazurovn/mazzy-vpn/core/lock"
	"github.com/mazurovn/mazzy-vpn/core/measure"
	"github.com/mazurovn/mazzy-vpn/core/profile"
	"github.com/mazurovn/mazzy-vpn/core/reachcache"
)

type catalogT = catalog.Catalog

const coreOpenVPN = core.OpenVPN

// isFullTunnel reports whether any peer routes a default (0.0.0.0/0 or ::/0).
func isFullTunnel(cfg *profile.Config) bool {
	for _, p := range cfg.Peers {
		for _, a := range p.AllowedIPs {
			if a == "0.0.0.0/0" || a == "::/0" {
				return true
			}
		}
	}
	return false
}

// pathMTUOK reports whether a large (1400-byte payload) DF ping reaches the
// endpoint host via the uplink — i.e. the encrypted path can carry full-size
// packets, ruling MTU in or out as a cause of "handshake works, data doesn't".
func pathMTUOK(ctx context.Context, uplink, host string) bool {
	args := []string{"-c", "1", "-W", "3", "-M", "do", "-s", "1400"}
	if uplink != "" {
		args = append(args, "-I", uplink)
	}
	args = append(args, host)
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return exec.CommandContext(cctx, "ping", args...).Run() == nil
}

// ProbeResult is the deep per-zone verdict.
type ProbeResult struct {
	Name string `json:"name"`
	// Config layer.
	ConfigValid bool     `json:"config_valid"`
	Protocol    string   `json:"protocol"`
	Endpoint    string   `json:"endpoint"`
	FullTunnel  bool     `json:"full_tunnel"`
	AmneziaV2   bool     `json:"amnezia_v2"` // has S3/S4/i1 signature-packet obfuscation
	ConfigNotes []string `json:"config_notes,omitempty"`
	// Endpoint layer (via physical uplink, no tunnel).
	ICMPAlive bool  `json:"icmp_alive"`
	RTTMs     int64 `json:"rtt_ms"`
	PathMTUOK bool  `json:"path_mtu_ok"` // large DF packet reaches the endpoint
	// Connection layer (real tunnel up).
	Handshake    bool   `json:"handshake"`
	EgressOK     bool   `json:"egress_ok"`
	EgressIP     string `json:"egress_ip,omitempty"`
	TxBytes      int64  `json:"tx_bytes"`
	RxBytes      int64  `json:"rx_bytes"`
	Verdict      string `json:"verdict"`
	VerdictNote  string `json:"verdict_note,omitempty"`
	DurationMsec int64  `json:"duration_ms"`
}

// cmdProbe is the HARD, deep connectivity test: for each selected zone it
// checks the config, probes the endpoint through the physical uplink, then
// ACTUALLY brings the tunnel up and measures real egress plus the WireGuard
// tx/rx byte counters — so it can tell "works", "server accepts the tunnel but
// does not route internet egress" (tx≫rx, no egress), "dead" (no handshake)
// and "bad config" apart. This is far stronger than `test` (ICMP only).
//
// It needs exclusive tunnel access. When a daemon owns the tunnel it is
// stopped automatically (announced), and after the sweep the VPN is brought
// BACK automatically: the daemon restarts on the original zone if it proved
// usable, else on the best proven-working zone — the machine is never left
// offline because a diagnostic ran.
//
// Every verdict is persisted to the shared reachcache, so ranking/failover
// immediately prefer the zones that actually routed and sink the dead ones.
//
// Usage: sudo mazzy-vpn probe [NAME|--all] [--json]
func cmdProbe(ctx context.Context, args []string) int {
	if !requireRoot("probe") {
		return 1
	}
	jsonOut := hasFlag(args, "--json")
	restoreZone := "" // non-empty: a daemon was stopped and must be brought back
	if snap, running := daemonRunning(); running {
		restoreZone = snap.Zone
		if !jsonOut {
			fmt.Printf("stopping the daemon (zone %s) for exclusive tunnel access — it will be restarted after the probe\n",
				safeDisplay(snap.Zone))
		}
		if rc := cmdStop(ctx, nil); rc != 0 && rc != 3 {
			fmt.Fprintln(os.Stderr, "could not stop the running daemon; aborting probe")
			return 1
		}
	}
	mu, err := lock.Acquire(lockDir())
	if err != nil {
		fmt.Fprintln(os.Stderr, "another mazzy-vpn operation is in progress")
		return 1
	}
	// The lock is released EXPLICITLY before the post-probe daemon restart —
	// the restarted daemon must acquire it itself and would fail while this
	// process still held it. locked guards against a double unlock.
	locked := true
	unlock := func() {
		if locked {
			locked = false
			mu.Unlock()
		}
	}
	defer unlock()

	names := probeTargets(newCatalog(), args)
	if len(names) == 0 {
		fmt.Fprintln(os.Stderr, "no zones to probe (import profiles, or name one)")
		return 2
	}
	if !jsonOut {
		fmt.Printf("Deep-probing %d zone(s) — each is connected for real and measured (tx/rx). This takes a while.\n\n", len(names))
	}

	results := make([]ProbeResult, 0, len(names))
	uplink := settingsUplink()
	for _, name := range names {
		res := probeOne(ctx, name, uplink)
		results = append(results, res)
		recordProbeVerdict(res)
		if !jsonOut {
			printProbeLine(res)
		}
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
	} else {
		printProbeSummary(results)
	}
	usable := false
	for _, r := range results {
		if r.Verdict == "WORKS" {
			usable = true
			break
		}
	}
	if restoreZone != "" {
		unlock() // the restarted daemon takes the lock itself
		restartDaemonAfterProbe(ctx, restoreZone, results, jsonOut)
	}
	if usable {
		return 0 // at least one usable zone
	}
	return 1
}

// recordProbeVerdict persists one deep-test outcome into the shared egress
// history, so selection stops trusting ping alone: WORKS ranks the zone first,
// every dead/non-routing verdict sinks it. BAD_CONFIG says nothing about the
// server, so it records nothing.
func recordProbeVerdict(r ProbeResult) {
	switch r.Verdict {
	case "WORKS":
		reachcache.New().RecordOK(r.Name)
	case "DEAD", "SERVER_NOT_ROUTING", "NO_EGRESS":
		reachcache.New().RecordFail(r.Name)
	}
}

// restartDaemonAfterProbe brings the VPN back after a probe that had to stop a
// running daemon (the caller must have released the mutation lock first — the
// detached daemon acquires it itself). Zone choice: the original zone if the
// sweep proved it WORKS, else --best, which now ranks with the fresh probe
// verdicts and so lands on a proven-working server.
func restartDaemonAfterProbe(ctx context.Context, original string, results []ProbeResult, jsonOut bool) {
	zone := "--best"
	for _, r := range results {
		if r.Name == original && r.Verdict == "WORKS" {
			zone = original
			break
		}
	}
	if !jsonOut {
		fmt.Printf("\nrestarting the VPN daemon (%s)...\n", safeDisplay(zone))
	}
	if rc := cmdDaemon(ctx, []string{zone, "--background"}); rc != 0 {
		fmt.Fprintf(os.Stderr, "could not restart the daemon automatically; reconnect with: sudo mazzy-vpn up %s\n", safeDisplay(zone))
	}
}

// probeTargets resolves the zone list: an explicit NAME, or every managed
// profile (default / --all). OpenVPN entries are skipped (the embedded engine
// cannot bring them up).
func probeTargets(cat *catalogT, args []string) []string {
	if name := firstNonFlagValueAware(args); name != "" {
		return []string{name}
	}
	entries, err := cat.List()
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.Protocol == coreOpenVPN {
			continue
		}
		out = append(out, e.Name)
	}
	return out
}

// probeOne runs the full config→endpoint→connection ladder for one zone.
func probeOne(ctx context.Context, name, uplink string) ProbeResult {
	start := time.Now()
	res := ProbeResult{Name: name, Verdict: "DEAD"}

	entry, err := newCatalog().Get(name)
	if err != nil {
		res.Verdict = "BAD_CONFIG"
		res.VerdictNote = "not in catalog"
		return res
	}
	proto, cfg, err := loadProfile(entry.File)
	if err != nil {
		res.Verdict = "BAD_CONFIG"
		res.VerdictNote = err.Error()
		res.DurationMsec = time.Since(start).Milliseconds()
		return res
	}
	res.ConfigValid = true
	res.Protocol = proto.Title()
	res.Endpoint = cfg.Endpoint()
	res.FullTunnel = isFullTunnel(cfg)
	res.AmneziaV2 = cfg.HasAmneziaFields
	if !res.FullTunnel {
		res.ConfigNotes = append(res.ConfigNotes, "not a full-tunnel (0.0.0.0/0) config")
	}
	if len(cfg.DNS) == 0 {
		res.ConfigNotes = append(res.ConfigNotes, "no DNS set")
	}

	// Endpoint layer: reachability + path MTU via the physical uplink.
	host, _, _ := net.SplitHostPort(cfg.Endpoint())
	if host != "" {
		png := measure.NewPinger()
		png.Interface = uplink
		png.Timeout = 3 * time.Second
		if rtt, ok := png.Ping(ctx, host); ok {
			res.ICMPAlive = true
			res.RTTMs = int64(rtt + 0.5)
		}
		res.PathMTUOK = pathMTUOK(ctx, uplink, host)
	}

	// Connection layer: bring the tunnel up for real and measure egress + bytes.
	conn, err := connect.Up(ctx, proto, cfg, connectOpts(uplink))
	if err != nil {
		res.VerdictNote = "connect: " + err.Error()
		res.DurationMsec = time.Since(start).Milliseconds()
		return res
	}
	defer func() { _ = conn.Down(context.WithoutCancel(ctx)) }()

	snap := livecheck.New().WaitProtected(ctx, conn.Interface, 20*time.Second)
	if age, ok := conn.HandshakeAge(); ok && age < 3*time.Minute {
		res.Handshake = true
	}
	res.RxBytes, res.TxBytes, _ = conn.Transfer()
	res.EgressOK = snap.Protected()
	res.EgressIP = snap.EgressIP

	switch {
	case res.EgressOK:
		res.Verdict = "WORKS"
	case res.Handshake && res.TxBytes > res.RxBytes*4 && res.TxBytes > 2000:
		// We sent real data into the tunnel but the server returned almost
		// nothing — definite one-way traffic: it accepts the handshake but does
		// not forward internet egress.
		res.Verdict = "SERVER_NOT_ROUTING"
		res.VerdictNote = fmt.Sprintf("tunnel up, one-way traffic: tx=%dB rx=%dB (server accepts the tunnel but forwards nothing)", res.TxBytes, res.RxBytes)
	case res.Handshake:
		// Handshake up but egress not confirmed in the window, and traffic is not
		// clearly one-way — could be a slow/flaky server or blocked probes.
		res.Verdict = "NO_EGRESS"
		res.VerdictNote = fmt.Sprintf("handshake ok but egress unconfirmed (tx=%dB rx=%dB): %s", res.TxBytes, res.RxBytes, snap.Reason)
	default:
		res.Verdict = "DEAD"
		res.VerdictNote = "no WireGuard handshake (server down or blocked)"
	}
	res.DurationMsec = time.Since(start).Milliseconds()
	return res
}

// printProbeLine renders one zone's verdict for humans.
func printProbeLine(r ProbeResult) {
	glyph, label := "✖", r.Verdict
	switch r.Verdict {
	case "WORKS":
		glyph = "✔"
	case "SERVER_NOT_ROUTING":
		glyph, label = "▲", "SERVER NOT ROUTING"
	case "NO_EGRESS":
		glyph, label = "▲", "NO EGRESS"
	case "BAD_CONFIG":
		glyph, label = "✖", "BAD CONFIG"
	}
	line := fmt.Sprintf("%s %-26s %-18s", glyph, safeDisplay(trunc(r.Name, 26)), label)
	if r.EgressOK {
		line += fmt.Sprintf(" egress %s (%d ms)", safeDisplay(r.EgressIP), r.RTTMs)
	} else if r.ICMPAlive {
		line += fmt.Sprintf(" ping %d ms · tx %s rx %s", r.RTTMs, fmtBytes(r.TxBytes), fmtBytes(r.RxBytes))
	}
	fmt.Println(line)
	if r.VerdictNote != "" && r.Verdict != "WORKS" {
		fmt.Println("    " + safeDisplay(r.VerdictNote))
	}
	for _, n := range r.ConfigNotes {
		fmt.Println("    config: " + safeDisplay(n))
	}
}

// printProbeSummary prints the aggregate counts and the usable-zone list.
func printProbeSummary(rs []ProbeResult) {
	works, notrouting, dead, bad := 0, 0, 0, 0
	usable := []string{}
	for _, r := range rs {
		switch r.Verdict {
		case "WORKS":
			works++
			usable = append(usable, r.Name)
		case "SERVER_NOT_ROUTING", "NO_EGRESS":
			notrouting++
		case "BAD_CONFIG":
			bad++
		default:
			dead++
		}
	}
	fmt.Printf("\n→ %d work · %d no-egress · %d dead · %d bad-config (of %d)\n",
		works, notrouting, dead, bad, len(rs))
	if len(usable) > 0 {
		fmt.Println("  usable zones: " + safeDisplay(joinStrs(usable)))
	}
	if notrouting > 0 {
		fmt.Println("  server-not-routing zones accept the tunnel but give no internet — likely provider/account issue, not mazzy.")
	}
}

func joinStrs(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out
}
