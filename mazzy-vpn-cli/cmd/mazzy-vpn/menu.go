// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

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
		case "1": // quick connect
			menuQuickConnect(ctx, set)
		case "2": // choose zone
			menuChooseZone(ctx, in, cat)
		case "3": // reconnect with diagnostics
			menuReconnectDiagnostics(ctx)
		case "4": // disconnect
			runPrivileged(ctx, "disconnect")
		case "5": // recover / clean
			menuRecover(ctx, in)
		case "6": // test servers
			cmdTest(ctx, nil)
		case "7": // best zone
			cmdBest(ctx, nil)
		case "8": // adapters
			cmdAdapters(ctx, nil)
		case "9": // network diagnostics
			cmdNetdiag(ctx, nil)
		case "10": // AI providers
			menuProviders(ctx, in)
		case "11": // import
			menuImport(ctx, in)
		case "12": // profiles
			cmdProfiles(ctx, nil)
		case "13": // settings
			menuSettings(ctx, in, setStore)
		case "14": // doctor
			cmdDoctor(ctx, nil)
		case "15": // update from GitHub
			cmdUpdate(ctx, nil)
		case "16": // diagnose
			cmdDiagnose(ctx, nil)
		case "17": // trace
			menuTrace(ctx, in)
		case "18": // stealth
			cmdStealth(ctx, nil)
		case "19": // dns privacy
			cmdDNSCheck(ctx, nil)
		case "0", "q", "quit", "exit":
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

// drawHeader samples the live connection status and prints a compact banner.
func drawHeader(ctx context.Context) {
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
			fmt.Printf("│  Status: ✔ PROTECTED  egress %-17s │\n", snap.EgressIP)
		default:
			fmt.Printf("│  Status: ⚠ LINK UP (%s)%*s│\n", iface, 24-len(iface), "")
		}
	}
	// Show the physical uplink.
	if adapters, err := netadapter.List(); err == nil {
		if rec, _, ok := netadapter.Recommend(adapters); ok {
			fmt.Printf("│  Uplink: %-37s│\n", rec.Name+" ("+rec.Kind()+")")
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
	fmt.Println("  Diagnostics")
	fmt.Println("   6. 📶 Test servers (live ping)")
	fmt.Println("   7. 🏆 Best zone")
	fmt.Println("   8. 🔌 Network adapters")
	fmt.Println("   9. 🩺 Analyze network (fixes)")
	fmt.Println("  10. 🤖 Check AI providers")
	fmt.Println("  16. 🔍 Diagnose problems (what's wrong)")
	fmt.Println("  17. 🧭 Trace packet path")
	fmt.Println("  18. 🕵️  Stealth check (anti-detection)")
	fmt.Println("  19. 🔒 DNS privacy check")
	fmt.Println("  Profiles & settings")
	fmt.Println("  11. 📥 Import config / folder")
	fmt.Println("  12. 📋 List profiles")
	fmt.Println("  13. ⚙️  Settings")
	fmt.Println("  14. 🔧 Doctor")
	fmt.Println("  15. ⬆️  Update from GitHub")
	fmt.Println("   0. Quit")
}

// menuQuickConnect connects to the preferred zone or the best live one.
func menuQuickConnect(ctx context.Context, set settings.Settings) {
	if set.PreferredZone != "" {
		fmt.Println(translator().Tf("cli.menu.quick_connect", safeDisplay(set.PreferredZone)))
		runPrivileged(ctx, "up", set.PreferredZone)
		return
	}
	runPrivileged(ctx, "up", "--best")
}

// menuReconnectDiagnostics runs diagnostics then reconnects to the best zone.
func menuReconnectDiagnostics(ctx context.Context) {
	fmt.Println(translator().T("cli.netdiag.running"))
	cmdNetdiag(ctx, nil)
	fmt.Println("\nRe-testing servers and reconnecting to the best live zone...")
	runPrivileged(ctx, "disconnect")
	runPrivileged(ctx, "up", "--best")
}

// menuRecover confirms then force-cleans all tunnels.
func menuRecover(ctx context.Context, in *bufio.Reader) {
	fmt.Print(translator().T("cli.menu.confirm_recover"))
	line, _ := in.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(line)) != "y" {
		fmt.Println(translator().T("cli.menu.cancelled"))
		return
	}
	runPrivileged(ctx, "recover")
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
		fmt.Printf("  %2d.%s %-24s %-10s %-4s %s\n", i+1, star, e.Name, e.Protocol, e.Country, ping)
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
	runPrivileged(ctx, "up", entries[n-1].Name)
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
		fmt.Printf("  7. Preferred zone          : %s\n", orDash(set.PreferredZone))
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

// runPrivileged runs a mazzy-vpn subcommand, re-invoking via sudo when the
// action needs root and we are not root. This lets the menu trigger connect/
// disconnect without the user leaving the TUI.
func runPrivileged(_ context.Context, subcmd string, sargs ...string) {
	self, _ := os.Executable()
	args := append([]string{subcmd}, sargs...)
	needsRoot := subcmd == "up" || subcmd == "connect" || subcmd == "disconnect" ||
		subcmd == "recover" || subcmd == "auto" || subcmd == "daemon"
	if needsRoot && os.Geteuid() != 0 {
		fmt.Printf("(needs root) run: sudo %s %s\n", self, strings.Join(args, " "))
		return
	}
	// In-process dispatch keeps it simple and avoids re-exec.
	_ = run(args)
}
