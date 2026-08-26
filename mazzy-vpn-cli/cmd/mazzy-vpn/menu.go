// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/livecheck"
	"github.com/mazurovn/mazzy-vpn/core/netadapter"
	"github.com/mazurovn/mazzy-vpn/core/settings"
)

// cmdMenu runs the interactive TUI: it shows a live status header, a grouped
// action menu, and does not block while displaying status (status is sampled
// fresh each time the menu is drawn).
func cmdMenu(ctx context.Context, _ []string) int {
	in := bufio.NewReader(os.Stdin)
	cat := newCatalog()
	setStore := settings.NewStore()
	t := translator() // localized prompts (resolves language, no hardcode)

	// P1-4: auto-connect on start when enabled and nothing is up yet.
	if s := setStore.Load(); s.AutoConnect && detectLiveInterface() == "" && cat.Count() > 0 {
		fmt.Println(t.T("cli.menu.autoconnect"))
		menuQuickConnect(ctx, s)
	}

	for {
		set := setStore.Load()
		drawHeader(ctx)
		drawMenu(cat.Count(), set)

		fmt.Print("\n" + t.T("prompt.choose"))
		line, err := in.ReadString('\n')
		if err != nil {
			return 0
		}
		choice := strings.TrimSpace(line)

		switch choice {
		// -- Connect --
		case "1": // quick connect
			menuQuickConnect(ctx, set)
		case "2": // choose zone
			menuChooseZone(ctx, in, cat)
		case "3": // reconnect with diagnostics
			menuReconnectDiagnostics(ctx)
		case "4": // disconnect
			menuDisconnect(ctx)
		case "5": // recover / clean
			menuRecover(ctx, in)
		case "6": // run in background (detached daemon)
			menuBackground(ctx, set)
		case "l", "L", "log": // view the activity log
			menuViewLog(ctx, in)
		case "k", "K", "stop": // stop the background daemon
			menuStopBackground(ctx)
		// -- Diagnostics --
		case "7": // test servers
			cmdTest(ctx, nil)
		case "8": // best zone
			cmdBest(ctx, nil)
		case "9": // adapters
			cmdAdapters(ctx, nil)
		case "10": // network diagnostics
			cmdNetdiag(ctx, nil)
		case "11": // diagnose
			cmdDiagnose(ctx, nil)
		case "12": // trace
			menuTrace(ctx, in)
		case "13": // stealth
			cmdStealth(ctx, nil)
		case "14": // dns privacy
			cmdDNSCheck(ctx, nil)
		case "15": // timezone mimic (align to egress)
			reportResult("mimic", runPrivileged(ctx, "mimic", "--apply"))
		case "16": // AI providers
			menuProviders(ctx, in)
		// -- Profiles & settings --
		case "17": // import
			menuImport(ctx, in)
		case "18": // profiles
			cmdProfiles(ctx, nil)
		case "v", "V", "verify": // audit every config's health
			cmdVerify(ctx, nil)
		case "19": // favorite/unfavorite a zone
			menuFavorite(ctx, in, cat)
		case "20": // remove a profile
			menuRemove(ctx, in, cat)
		case "21": // settings
			menuSettings(ctx, in, setStore)
		case "22": // language
			menuLanguage(ctx, in)
		// -- System --
		case "23": // doctor
			cmdDoctor(ctx, nil)
		case "24": // update from GitHub
			cmdUpdate(ctx, nil)
		case "0", "q", "quit", "exit":
			// Stop a menu-scoped (--session) daemon on exit, but leave a persistent
			// (--background) daemon running so it survives closing the terminal.
			// Route through the elevated `stop` so a root-owned daemon is actually
			// terminated (unprivileged SIGTERM would EPERM).
			if snap, ok := daemonRunning(); ok && !snap.Background {
				if runPrivileged(ctx, "stop") == 0 {
					fmt.Println(translator().T("cli.bg.stopped"))
				}
			}
			fmt.Println("Bye.")
			return 0
		case "":
			// redraw
		default:
			fmt.Println(translator().T("cli.menu.unknown_choice"))
		}
		fmt.Println()
	}
}

// drawHeader renders the menu banner. When a background daemon is publishing a
// heartbeat it shows the rich live dashboard (status, egress, latency graph,
// recent errors, error-rate); otherwise it samples a one-shot live status so
// the user still sees whether a foreground/manual tunnel is up.
func drawHeader(ctx context.Context) {
	// Prefer the rich dashboard fed by a background daemon's heartbeat.
	if drawLiveDashboard() {
		return
	}
	iface := detectLiveInterface()
	fmt.Println("┌───────────────────────────────────────────────┐")
	fmt.Println("│  Mazzy VPN                                      │")
	fmt.Println("├───────────────────────────────────────────────┤")
	if iface == "" {
		fmt.Println("│  Status: ✖ DISCONNECTED (plain uplink)          │")
	} else {
		snap := livecheck.New().Check(ctx, iface)
		switch {
		case snap.Protected():
			fmt.Printf("│  Status: ✔ PROTECTED  egress %-17s │\n", safeDisplay(snap.EgressIP))
		default:
			fmt.Printf("│  Status: ⚠ LINK UP (%s)%*s│\n", iface, 24-len(iface), "")
		}
	}
	// Show the physical uplink.
	if adapters, err := netadapter.List(); err == nil {
		if rec, _, ok := netadapter.Recommend(adapters); ok {
			fmt.Printf("│  Uplink: %-37s│\n", safeDisplay(rec.Name)+" ("+rec.Kind()+")")
		}
	}
	fmt.Println("└───────────────────────────────────────────────┘")
}

// drawMenu prints the grouped action menu with current settings hints.
func drawMenu(profileCount int, set settings.Settings) {
	onoff := func(b bool) string {
		if b {
			return "on"
		}
		return "off"
	}
	fmt.Printf("\nProfiles: %d   |   auto-connect:%s  auto-reconnect:%s  notify:%s\n",
		profileCount, onoff(set.AutoConnect), onoff(set.AutoReconnect), onoff(set.Notifications))
	fmt.Println()
	fmt.Println("  Connect")
	fmt.Println("   1. ⚡ Quick connect (best/preferred zone)")
	fmt.Println("   2. 🌍 Choose a zone (with ping)")
	fmt.Println("   3. 🔄 Reconnect with diagnostics")
	fmt.Println("   4. ⏹  Disconnect")
	fmt.Println("   5. 🧹 Recover / clean (panic → plain Wi‑Fi)")
	fmt.Println("   6. 🛰  Run in background (survives closing terminal)")
	fmt.Println("   l. 📜 View activity log     k. ⏹  Stop background daemon")
	fmt.Println("  Diagnostics")
	fmt.Println("   7. 📶 Test servers (live ping)")
	fmt.Println("   8. 🏆 Best zone")
	fmt.Println("   9. 🔌 Network adapters")
	fmt.Println("  10. 🩺 Analyze network (fixes)")
	fmt.Println("  11. 🔍 Diagnose problems (what's wrong)")
	fmt.Println("  12. 🧭 Trace packet path")
	fmt.Println("  13. 🕵️  Stealth check (anti-detection)")
	fmt.Println("  14. 🔒 DNS privacy check")
	fmt.Println("  15. 🕰️  Align timezone to egress (mimic)")
	fmt.Println("  16. 🤖 Check AI providers")
	fmt.Println("  Profiles & settings")
	fmt.Println("  17. 📥 Import config / folder")
	fmt.Println("  18. 📋 List profiles          v. 🩺 Verify configs (health audit)")
	fmt.Println("  19. ★ Favorite / unfavorite a zone")
	fmt.Println("  20. 🗑  Remove a profile")
	fmt.Println("  21. ⚙️  Settings")
	fmt.Println("  22. 🌐 Language")
	fmt.Println("  System")
	fmt.Println("  23. 🔧 Doctor")
	fmt.Println("  24. ⬆️  Update from GitHub")
	fmt.Println("   0. Quit")
}

// menuQuickConnect connects to the preferred zone or the best live one. It runs
// the self-healing daemon DETACHED in --session mode: the connection comes up
// with auto-reconnect/failover, the user lands right back in the menu (no
// blocking log / Ctrl+C), and the live dashboard header tracks it. The session
// daemon is stopped automatically when the user quits the menu.
func menuQuickConnect(ctx context.Context, set settings.Settings) {
	zone := set.PreferredZone
	if zone == "" {
		zone = "--best"
	} else {
		fmt.Println(translator().Tf("cli.menu.quick_connect", safeDisplay(zone)))
	}
	// Resume, zone switch and cold start ALL go through the elevated daemon
	// entry point. The unprivileged menu cannot write the intent file itself —
	// /run/mazzy-vpn is created root-owned 0755 by the daemon, so the previous
	// direct writeDesired here failed silently (EACCES was discarded) and the
	// "resumed" message lied while the daemon never saw the request. The
	// elevated cmdDaemon forwards the intent as root to a live daemon (same
	// zone → resume, --best → the OWNER re-ranks with its cooldown map) or
	// starts a fresh session daemon when none is running.
	reportResult("connect", runPrivileged(ctx, "daemon", zone, "--session"))
}

// menuReconnectDiagnostics runs diagnostics then reconnects to the best zone.
func menuReconnectDiagnostics(ctx context.Context) {
	fmt.Println(translator().T("cli.netdiag.running"))
	cmdNetdiag(ctx, nil)
	fmt.Println("\nRe-testing servers and reconnecting to the best live zone...")
	runPrivileged(ctx, "disconnect")
	reportResult("connect", runPrivileged(ctx, "up", "--best"))
}

// menuRecover confirms then force-cleans all tunnels.
func menuRecover(ctx context.Context, in *bufio.Reader) {
	fmt.Print(translator().T("cli.menu.confirm_recover"))
	line, _ := in.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(line)) != "y" {
		fmt.Println(translator().T("cli.menu.cancelled"))
		return
	}
	reportResult("recover", runPrivileged(ctx, "recover"))
}

// menuProviders lets the user filter AI provider checks by type.
func menuProviders(ctx context.Context, in *bufio.Reader) {
	fmt.Print(translator().T("cli.menu.prompt.filter"))
	line, _ := in.ReadString('\n')
	t := strings.TrimSpace(line)
	if t == "" {
		cmdProviders(ctx, nil)
	} else {
		cmdProviders(ctx, []string{"--type", t})
	}
}

// menuChooseZone lists profiles with protocol + live ping and connects.
func menuChooseZone(ctx context.Context, in *bufio.Reader, cat interface {
	Count() int
}) {
	c := newCatalog()
	entries, _ := c.List()
	if len(entries) == 0 {
		fmt.Println(translator().T("cli.menu.no_profiles_opt"))
		return
	}
	fmt.Println(translator().T("cli.catalog.measuring"))
	pings := measureCatalogPings(ctx, c)

	fmt.Printf("  %2s  %-24s %-10s %-4s %s\n", "#", "NAME", "PROTOCOL", "CC", "PING")
	for i, e := range entries {
		star := " "
		if e.Favorite {
			star = "★"
		}
		ping := pings[e.Name]
		if ping == "" {
			ping = "—"
		}
		fmt.Printf("  %2d.%s %-24s %-10s %-4s %s\n", i+1, star, safeDisplay(e.Name), safeDisplay(string(e.Protocol)), safeDisplay(e.Country), safeDisplay(ping))
	}
	fmt.Print(translator().T("cli.menu.prompt.zone_num"))
	line, _ := in.ReadString('\n')
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || n < 1 || n > len(entries) {
		if n != 0 {
			fmt.Println(translator().T("cli.menu.invalid_selection"))
		}
		return
	}
	selected := entries[n-1]
	// P4: OpenVPN zones cannot be brought up by the embedded engine. Tell the
	// user explicitly instead of failing deep inside connect.
	if selected.Protocol == core.OpenVPN {
		fmt.Println(translator().Tf("cli.menu.ovpn_unsupported", safeDisplay(selected.Name)))
		return
	}
	// Detached --session daemon: connect + auto-reconnect, then back to the menu.
	reportResult("connect", runPrivileged(ctx, "daemon", selected.Name, "--session"))
}

// menuFavorite lists zones and toggles the favorite flag on a chosen one.
func menuFavorite(ctx context.Context, in *bufio.Reader, cat interface{ Count() int }) {
	c := newCatalog()
	entries, _ := c.List()
	if len(entries) == 0 {
		fmt.Println(translator().T("cli.menu.no_profiles_opt"))
		return
	}
	for i, e := range entries {
		star := " "
		if e.Favorite {
			star = "★"
		}
		fmt.Printf("  %2d.%s %s\n", i+1, star, safeDisplay(e.Name))
	}
	fmt.Print(translator().T("cli.menu.prompt.zone_num"))
	line, _ := in.ReadString('\n')
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || n < 1 || n > len(entries) {
		return
	}
	sel := entries[n-1]
	args := []string{sel.Name}
	if sel.Favorite {
		args = append(args, "--off")
	}
	cmdFavorite(ctx, args)
}

// menuRemove lists zones and removes the chosen managed profile after confirm.
func menuRemove(ctx context.Context, in *bufio.Reader, cat interface{ Count() int }) {
	c := newCatalog()
	entries, _ := c.List()
	if len(entries) == 0 {
		fmt.Println(translator().T("cli.menu.no_profiles_opt"))
		return
	}
	for i, e := range entries {
		fmt.Printf("  %2d. %s\n", i+1, safeDisplay(e.Name))
	}
	fmt.Print(translator().T("cli.menu.prompt.zone_num"))
	line, _ := in.ReadString('\n')
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || n < 1 || n > len(entries) {
		return
	}
	sel := entries[n-1]
	fmt.Print(translator().Tf("cli.menu.confirm_remove", safeDisplay(sel.Name)))
	confirm, _ := in.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
		fmt.Println(translator().T("cli.menu.cancelled"))
		return
	}
	cmdRemove(ctx, []string{sel.Name})
}

// menuLanguage prompts for a UI language code (or --list) and applies it.
func menuLanguage(ctx context.Context, in *bufio.Reader) {
	cmdLanguage(ctx, []string{"--list"})
	fmt.Print(translator().T("cli.menu.prompt.language"))
	line, _ := in.ReadString('\n')
	code := strings.TrimSpace(line)
	if code == "" {
		return
	}
	cmdLanguage(ctx, []string{code})
}

// menuDisconnect brings the tunnel down. When a self-healing daemon owns the
// connection, a bare `disconnect` would be futile (the daemon reconnects on its
// next tick), so we first record a durable down-intent that pauses the daemon's
// auto-reconnect, then tear the tunnel down. A subsequent Quick connect clears
// the intent and resumes.
func menuDisconnect(ctx context.Context) {
	// No direct intent write here: the unprivileged menu cannot write into the
	// root-owned runtime dir. The elevated `disconnect` records the down-intent
	// itself (recordDownIntent) BEFORE touching the interface, which is what
	// actually pauses the daemon's auto-reconnect.
	reportResult("disconnect", runPrivileged(ctx, "disconnect"))
}

// menuBackground starts a DETACHED background daemon that survives closing the
// terminal, then returns to the menu. Elevation (sudo/pkexec) prompts on the
// terminal now; the daemon self-detaches into a new session afterward. The
// menu's live dashboard header then reflects its heartbeat.
func menuBackground(ctx context.Context, set settings.Settings) {
	t := translator()
	if snap, ok := daemonRunning(); ok {
		fmt.Println(t.Tf("cli.bg.running", snap.PID, safeDisplay(snap.Zone)))
		return
	}
	zone := set.PreferredZone
	if zone == "" {
		zone = "--best"
	}
	fmt.Println(t.Tf("cli.bg.starting", safeDisplay(zone)))
	// runPrivileged elevates and runs `daemon <zone> --background`; the elevated
	// process forks the detached child and returns 0 immediately.
	reportResult("background", runPrivileged(ctx, "daemon", zone, "--background"))
}

// menuStopBackground stops a running background daemon via the elevated `stop`
// subcommand, so a root-owned daemon is actually terminated (an unprivileged
// SIGTERM would fail with EPERM).
func menuStopBackground(ctx context.Context) {
	t := translator()
	if _, ok := daemonRunning(); !ok {
		fmt.Println(t.T("cli.bg.none"))
		return
	}
	if runPrivileged(ctx, "stop") == 0 {
		fmt.Println(t.T("cli.bg.stopped"))
		return
	}
	fmt.Println(t.T("cli.bg.none"))
}

// menuViewLog shows the tail of the daemon activity log, then waits for Enter
// so the user can page back to the menu (the "log on a key, back on a key" UX).
func menuViewLog(_ context.Context, in *bufio.Reader) {
	t := translator()
	const n = 40
	lines := tailLog(n)
	fmt.Println(t.Tf("cli.log.title", n))
	if len(lines) == 0 {
		fmt.Println("  " + t.T("cli.log.empty"))
	} else {
		for _, ln := range lines {
			fmt.Println("  " + safeDisplay(ln))
		}
	}
	fmt.Println("\n" + t.T("cli.log.hint"))
	_, _ = in.ReadString('\n')
}

// menuTrace prompts for a zone and traces its packet path.
func menuTrace(ctx context.Context, in *bufio.Reader) {
	fmt.Print(translator().T("cli.menu.prompt.zone_name"))
	line, _ := in.ReadString('\n')
	z := strings.TrimSpace(line)
	if z == "" {
		cmdTrace(ctx, nil)
	} else {
		cmdTrace(ctx, []string{z})
	}
}

// menuImport prompts for a path and imports it.
func menuImport(ctx context.Context, in *bufio.Reader) {
	fmt.Print(translator().T("cli.menu.prompt.path"))
	line, _ := in.ReadString('\n')
	path := strings.TrimSpace(line)
	if path == "" {
		return
	}
	cmdImport(ctx, []string{path})
}

// menuSettings lets the user toggle preferences.
func menuSettings(ctx context.Context, in *bufio.Reader, store *settings.Store) {
	for {
		set := store.Load()
		onoff := func(b bool) string {
			if b {
				return "✔ on"
			}
			return "✖ off"
		}
		fmt.Println("\n── Settings ──")
		fmt.Printf("  1. Auto-connect on start   : %s\n", onoff(set.AutoConnect))
		fmt.Printf("  2. Auto-diagnostics        : %s\n", onoff(set.AutoDiagnostics))
		fmt.Printf("  3. Notifications           : %s\n", onoff(set.Notifications))
		fmt.Printf("  4. Auto-reconnect          : %s\n", onoff(set.AutoReconnect))
		fmt.Printf("  5. Kill-switch (fail-closed): %s\n", onoff(set.KillSwitch))
		fmt.Printf("  6. Auto-mimic timezone     : %s\n", onoff(set.AutoMimic))
		fmt.Printf("  7. Preferred zone          : %s\n", orDash(safeDisplay(set.PreferredZone)))
		fmt.Println("  0. Back")
		fmt.Print(translator().T("cli.menu.prompt.toggle"))
		line, _ := in.ReadString('\n')
		switch strings.TrimSpace(line) {
		case "1":
			set.AutoConnect = !set.AutoConnect
		case "2":
			set.AutoDiagnostics = !set.AutoDiagnostics
		case "3":
			set.Notifications = !set.Notifications
		case "4":
			set.AutoReconnect = !set.AutoReconnect
		case "5":
			set.KillSwitch = !set.KillSwitch
		case "6":
			set.AutoMimic = !set.AutoMimic
		case "7":
			fmt.Print(translator().T("cli.menu.prompt.pref_zone"))
			z, _ := in.ReadString('\n')
			set.PreferredZone = strings.TrimSpace(z)
		case "0", "":
			return
		default:
			fmt.Println(translator().T("cli.menu.unknown_choice"))
			continue
		}
		if err := store.Save(set); err != nil {
			fmt.Fprintln(os.Stderr, "could not save settings:", err)
		} else {
			fmt.Println(translator().T("cli.menu.saved"))
		}
	}
}

func orDash(s string) string {
	if s == "" {
		return "best (auto)"
	}
	return s
}

// reportResult prints a localized outcome line for a privileged action so the
// user always gets explicit feedback (success or failure) instead of a silent
// redraw. code == 0 means success.
func reportResult(action string, code int) {
	t := translator()
	if code == 0 {
		fmt.Println(t.Tf("cli.menu.action_ok", action))
		return
	}
	fmt.Fprintln(os.Stderr, t.Tf("cli.menu.action_failed", action, code))
}
