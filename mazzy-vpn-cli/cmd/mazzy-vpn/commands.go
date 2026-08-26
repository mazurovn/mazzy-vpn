// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/doctor"
	"github.com/mazurovn/mazzy-vpn/core/livecheck"
	"github.com/mazurovn/mazzy-vpn/core/profile"
	"github.com/mazurovn/mazzy-vpn/core/runstatus"
)

// safeDisplay sanitizes a user-controlled string (profile name, path, zone,
// uplink) before printing it to the terminal, stripping ASCII control and
// C1 characters so a crafted name cannot inject ANSI escape sequences, move the
// cursor, or forge new log lines. Printable Unicode is preserved.
func safeDisplay(s string) string {
	// Explicitly drop the line-break characters first (recognized by static
	// analysis as a log-injection barrier), then strip the remaining control
	// and C1 range.
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.Map(func(r rune) rune {
		if r == '\t' {
			return ' '
		}
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return -1 // drop control / C1
		}
		return r
	}, s)
}

// hasFlag reports whether args contain a flag.
func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// firstNonFlag returns the first bare (non-"-" prefixed) argument, or "". It
// lets subcommands accept flags in any position, e.g. `favorite --off NAME`.
func firstNonFlag(args []string) string {
	for _, a := range args {
		if a != "" && a[0] != '-' {
			return a
		}
	}
	return ""
}

// flagValue returns the value following --name, or "".
func flagValue(args []string, name string) string {
	for i, a := range args {
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// valueFlags are the flags that CONSUME the following token as their value, so a
// positional parser must skip that token instead of mistaking it for a name.
// Keeping this in one place means any new value-flag is honored everywhere and
// cannot regress the "value read as zone name" bug (audit P2-7).
var valueFlags = map[string]bool{
	"--uplink": true,
	"--proto":  true,
	"--type":   true,
}

// firstNonFlagValueAware returns the first bare (non-flag) argument, skipping the
// value token that belongs to a value-taking flag (e.g. `--uplink eth0`). This
// is the single positional parser the subcommands share.
func firstNonFlagValueAware(args []string) string {
	skipNext := false
	for _, a := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if valueFlags[a] {
			skipNext = true
			continue
		}
		if a != "" && a[0] != '-' {
			return a
		}
	}
	return ""
}

// cmdDoctor runs host diagnostics. With --heal it becomes the active rescuer:
// diagnose, then climb the rescue ladder until the connection is PROTECTED.
func cmdDoctor(ctx context.Context, args []string) int {
	if hasFlag(args, "--heal") {
		return healConnection(ctx)
	}
	rep := doctor.Run(ctx, nil)
	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rep)
		if !rep.Healthy() {
			return 1
		}
		return 0
	}
	for _, c := range rep.Checks {
		fmt.Printf("[%-4s] %s: %s\n", c.Status, safeDisplay(c.Name), safeDisplay(c.Detail))
	}
	// Live connection line: plain doctor previously said nothing about the actual
	// VPN state, so it looked "broken" during a real connection problem. Show the
	// daemon/egress state so doctor reflects reality.
	fmt.Println(liveConnectionLine(ctx))
	// Catalog health line: a quick, offline count so the user sees whether their
	// profiles are usable without running the full `verify` audit.
	if line := catalogHealthLine(); line != "" {
		fmt.Println(line)
	}
	fmt.Printf("\nSummary: OK=%d WARN=%d FAIL=%d\n", rep.OK, rep.Warn, rep.Fail)
	fmt.Println("  Tips: `sudo mazzy-vpn doctor --heal` fixes the connection · `sudo mazzy-vpn probe --all` deep-tests every zone")
	if !rep.Healthy() {
		return 1
	}
	return 0
}

// liveConnectionLine reports the ACTUAL connection state for doctor: the live
// daemon's state when present, else a one-shot egress check on any live tunnel,
// else "no VPN". This is what makes plain doctor honest about the connection.
func liveConnectionLine(ctx context.Context) string {
	if snap, ok := daemonRunning(); ok {
		switch snap.State {
		case runstatus.StateProtected:
			return "[OK  ] connection: PROTECTED via " + safeDisplay(snap.Zone) + " (egress " + safeDisplay(snap.Egress) + ")"
		case runstatus.StateReconnect, runstatus.StateConnecting:
			note := ""
			if r := snap.RecentErrors(1); len(r) > 0 {
				note = " — " + safeDisplay(r[0].Reason)
			}
			return "[WARN] connection: daemon " + string(snap.State) + " on " + safeDisplay(snap.Zone) + note
		case runstatus.StatePaused:
			return "[WARN] connection: daemon PAUSED (run: sudo mazzy-vpn daemon <zone> to resume)"
		default:
			return "[WARN] connection: daemon state " + string(snap.State)
		}
	}
	if iface := detectLiveInterface(); iface != "" {
		if livecheck.New().Check(ctx, iface).Protected() {
			return "[OK  ] connection: tunnel " + safeDisplay(iface) + " is routing (no daemon)"
		}
		return "[WARN] connection: tunnel " + safeDisplay(iface) + " up but NOT routing (try: sudo mazzy-vpn doctor --heal)"
	}
	return "[INFO] connection: no VPN active (plain uplink)"
}

// catalogHealthLine returns a one-line offline summary of catalog health for the
// doctor output: how many profiles are connectable (WG/AWG) vs OpenVPN-only.
// Empty when the catalog is empty. It never touches the network.
func catalogHealthLine() string {
	entries, err := newCatalog().List()
	if err != nil || len(entries) == 0 {
		return "[WARN] catalog: no profiles imported (run: mazzy-vpn import <dir>)"
	}
	connectable, ovpn := 0, 0
	for _, e := range entries {
		a := auditProfile(context.Background(), catalogEntry{
			Name: e.Name, File: e.File, Protocol: e.Protocol, Country: e.Country,
		}, false)
		switch {
		case a.Connectable:
			connectable++
		default:
			if e.Protocol == core.OpenVPN {
				ovpn++
			}
		}
	}
	status := "OK"
	if connectable == 0 {
		status = "WARN"
	}
	return fmt.Sprintf("[%-4s] catalog: %d profiles (%d connectable, %d OpenVPN-only) — run `mazzy-vpn verify` for details",
		status, len(entries), connectable, ovpn)
}

// profileInfo is the machine-first summary of one profile file.
type profileInfo struct {
	File     string   `json:"file"`
	Protocol string   `json:"protocol"`
	Valid    bool     `json:"valid"`
	Problems []string `json:"problems,omitempty"`
}

// detectProtocol infers a protocol from a filename extension.
func detectProtocol(path string) core.Protocol {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ovpn":
		return core.OpenVPN
	case ".conf":
		// Distinguish AmneziaWG vs WireGuard by content later; default to
		// AmneziaWG detection with WireGuard fallback in inspectProfile.
		return core.AmneziaWG
	default:
		return ""
	}
}

// inspectProfile parses and validates one profile file (WireGuard/AmneziaWG).
// OpenVPN validation is minimal here (structural checks land with the OpenVPN
// engine in a later step).
func inspectProfile(path string) profileInfo {
	info := profileInfo{File: filepath.Base(path)}
	proto := detectProtocol(path)
	info.Protocol = string(proto)

	if proto == core.OpenVPN || proto == "" {
		// Defer full OpenVPN parsing; report as recognized but unvalidated.
		info.Valid = proto == core.OpenVPN
		if proto == "" {
			info.Problems = []string{"unrecognized profile extension"}
		}
		return info
	}

	data, err := os.ReadFile(path)
	if err != nil {
		info.Problems = []string{"cannot read file"}
		return info
	}
	cfg, err := profile.Parse(string(data))
	if err != nil {
		info.Problems = []string{err.Error()}
		return info
	}
	// Try AmneziaWG first; if obfuscation params are absent, treat as WireGuard.
	if !cfg.HasAmneziaFields {
		proto = core.WireGuard
		info.Protocol = string(proto)
	}
	problems := profile.Validate(proto, cfg)
	info.Valid = len(problems) == 0
	info.Problems = problems
	return info
}

// cmdList lists and validates all profiles in a directory.
func cmdList(_ context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mazzy-vpn list DIR [--json]")
		return 2
	}
	dir := args[0]
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot read directory: %v\n", err)
		return 1
	}
	var infos []profileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".conf" && ext != ".ovpn" {
			continue
		}
		infos = append(infos, inspectProfile(filepath.Join(dir, e.Name())))
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].File < infos[j].File })

	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(infos)
		return 0
	}
	if len(infos) == 0 {
		fmt.Println("No profiles found in", dir)
		return 0
	}
	for _, in := range infos {
		mark := "OK"
		if !in.Valid {
			mark = "INVALID"
		}
		fmt.Printf("[%-7s] %-10s %s\n", mark, safeDisplay(in.Protocol), safeDisplay(in.File))
		for _, p := range in.Problems {
			fmt.Printf("            - %s\n", safeDisplay(p))
		}
	}
	return 0
}

// cmdValidate validates a single profile file.
func cmdValidate(_ context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mazzy-vpn validate FILE [--proto amneziawg|wireguard] [--json]")
		return 2
	}
	path := args[0]
	info := inspectProfile(path)
	if forced := flagValue(args, "--proto"); forced != "" {
		if p, ok := core.CanonicalProtocol(forced); ok {
			info.Protocol = string(p)
			if data, err := os.ReadFile(path); err == nil {
				if cfg, err := profile.Parse(string(data)); err == nil {
					problems := profile.Validate(p, cfg)
					info.Valid = len(problems) == 0
					info.Problems = problems
				}
			}
		}
	}
	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(info)
	} else {
		fmt.Printf("File:     %s\nProtocol: %s\nValid:    %v\n", safeDisplay(info.File), safeDisplay(info.Protocol), info.Valid)
		for _, p := range info.Problems {
			fmt.Println("  -", safeDisplay(p))
		}
	}
	if !info.Valid {
		return 1
	}
	return 0
}
