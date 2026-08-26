// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.
//
// Command mazzy-vpn is the autonomous Go CLI for Mazzy VPN. It links the
// mazzy-core engine directly (no external awg/awg-quick/wg/jq) and provides a
// human TUI-style output plus a machine-first --json mode for AI agents.
//
// This is the Phase 3 transition binary (backlog P3-1). Privileged operations
// (connect/disconnect) require root; read-only commands (doctor/verify/list)
// do not.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

// version is the CLI version. It is a var (not const) so release builds can
// stamp the exact tag with the linker, keeping ONE source of truth (audit
// P1-F): `go build -ldflags "-X main.version=$(git describe --tags)"`. The
// baseline default matches the current git tag for un-stamped/dev builds.
var version = "2.4.2"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	// A signal-aware root context: SIGINT/SIGTERM cancel every in-flight network
	// operation (connect, egress probes, zone ranking) so commands can unwind
	// cleanly. Previously ctx was context.Background(): an interrupt during
	// connect.Up killed the process mid-mutation with no teardown, leaving
	// interfaces, nft guards and — worst — an armed fail-closed kill-switch
	// behind ("no internet until recover"). Teardown paths that must still run
	// after cancellation use context.WithoutCancel(ctx).
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	// No args: launch the full-screen TUI when attached to a terminal, else the
	// line-based menu (scripts/pipes). ADR-0006.
	if len(args) == 0 {
		if isTTY() {
			return runTUI()
		}
		return cmdMenu(ctx, nil)
	}
	cmd, rest := args[0], args[1:]

	// Resolve all help forms before dispatch so `COMMAND -h` can never trigger
	// network, filesystem, elevation, or catalog side effects.
	if cmd == "help" {
		if len(rest) == 0 {
			printUsageTo(os.Stdout)
			return 0
		}
		if hasFlag(rest, "-h") || hasFlag(rest, "--help") {
			return printCommandHelp("help")
		}
		return printCommandHelp(rest[0])
	}
	if hasFlag(rest, "-h") || hasFlag(rest, "--help") {
		return printCommandHelp(cmd)
	}

	switch cmd {
	case "tui":
		return runTUI()
	case "menu", "--plain":
		return cmdMenu(ctx, rest)
	case "doctor":
		return cmdDoctor(ctx, rest)
	case "list":
		return cmdList(ctx, rest)
	case "validate":
		return cmdValidate(ctx, rest)
	case "verify", "audit":
		return cmdVerify(ctx, rest)
	case "import":
		return cmdImport(ctx, rest)
	case "profiles":
		return cmdProfiles(ctx, rest)
	case "favorite", "fav":
		return cmdFavorite(ctx, rest)
	case "remove", "rm":
		return cmdRemove(ctx, rest)
	case "test":
		return cmdTest(ctx, rest)
	case "best":
		return cmdBest(ctx, rest)
	case "adapters", "interfaces":
		return cmdAdapters(ctx, rest)
	case "netdiag", "analyze":
		return cmdNetdiag(ctx, rest)
	case "diagnose", "why":
		return cmdDiagnose(ctx, rest)
	case "trace":
		return cmdTrace(ctx, rest)
	case "stealth", "leaks":
		return cmdStealth(ctx, rest)
	case "mimic", "timezone":
		return cmdMimic(ctx, rest)
	case "dns-check", "dns":
		return cmdDNSCheck(ctx, rest)
	case "control", "identity":
		return cmdControl(ctx, rest)
	case "language", "lang":
		return cmdLanguage(ctx, rest)
	case "up":
		return cmdUp(ctx, rest)
	case "auto":
		return cmdAuto(ctx, rest)
	case "daemon":
		return cmdDaemon(ctx, rest)
	case "stop":
		return cmdStop(ctx, rest)
	case "connect":
		return cmdConnect(ctx, rest)
	case "disconnect", "down":
		return cmdDisconnect(ctx, rest)
	case "recover", "clean", "panic":
		return cmdRecover(ctx, rest)
	case "trust":
		return cmdTrust(ctx, rest)
	case "providers", "ai":
		return cmdProviders(ctx, rest)
	case "update", "self-update":
		return cmdUpdate(ctx, rest)
	case "status":
		return cmdStatus(ctx, rest)
	case "version", "--version", "-v":
		fmt.Println("mazzy-vpn", displayVersion())
		return 0
	case "--help", "-h":
		// Root help is a success path and writes to stdout.
		printUsageTo(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		return 2
	}
}

// displayVersion returns the version without a leftover leading "v" so a
// tag-stamped build (v2.2.0) and a dev build (2.2.0) print identically.
func displayVersion() string {
	if len(version) > 1 && (version[0] == 'v' || version[0] == 'V') {
		return version[1:]
	}
	return version
}

// isTTY reports whether stdin and stdout are both terminals (so the full-screen
// TUI is appropriate). Piped/redirected use falls back to the line menu.
func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		return false
	}
	fi2, err := os.Stdin.Stat()
	if err != nil || (fi2.Mode()&os.ModeCharDevice) == 0 {
		return false
	}
	return true
}

// printUsage writes the usage text to stderr (used by the error/unknown-command
// paths). printUsageTo lets the success `help` path target stdout.
func printUsage() { printUsageTo(os.Stderr) }

func printUsageTo(w io.Writer) {
	t := translator()
	fmt.Fprintf(w, `mazzy-vpn — %s

Usage:
  mazzy-vpn                           full-screen TUI dashboard (or menu if piped)
  mazzy-vpn tui | menu | --plain      force TUI / line menu

 %s
  mazzy-vpn import <FILE|DIR>...      import profiles into the catalog
  mazzy-vpn profiles [--json]         list managed profiles
  mazzy-vpn favorite NAME [--off]     mark/unmark a favorite zone
  mazzy-vpn remove NAME               remove a managed profile

 %s
  sudo mazzy-vpn up [NAME|--best]     connect by name, or auto-pick best zone
  sudo mazzy-vpn auto                 rank zones + connect to the best (failover)
  sudo mazzy-vpn connect FILE [--uplink IF]  connect via a raw file path / pinned uplink
  mazzy-vpn status [--json]           show the connection intent

  While connected, a live dashboard shows PROTECTED/LINK-UP status, sends
  desktop notifications (connected/reconnecting/disconnected), and
  auto-reconnects if the egress drops. Add --no-reconnect to disable.

 %s
  sudo mazzy-vpn daemon NAME              run persistently with auto-reconnect
  sudo mazzy-vpn daemon NAME --background  detach; survives closing the terminal
  sudo mazzy-vpn trust [--revoke]         passwordless daemon control (sudoers drop-in)
  sudo systemctl enable --now mazzy-vpn@NAME   start at boot (systemd)

  The interactive menu now keeps you IN the menu after connecting: a live
  dashboard header shows status, egress, a latency graph, recent errors and
  the error rate. Press 'l' to view the activity log, 'k' to stop a background
  daemon. Notifications toggle in Settings.

 %s
  mazzy-vpn test [--json]             probe all servers (latency/reachability)
  mazzy-vpn best [--json]             print the best zone to connect to
  mazzy-vpn adapters [--json]         list network interfaces + recommendation
  mazzy-vpn netdiag [--json]          analyze the network + suggest fixes
  mazzy-vpn diagnose [--json]         root-cause analysis: what's wrong + fixes
  mazzy-vpn trace [ZONE] [--json]     packet path: DNS→server→tunnel→egress
  mazzy-vpn stealth [--json]          detection risks (IPv6/DNS/timezone/ASN)
  sudo mazzy-vpn mimic [--apply]      align timezone to egress (look local)
  mazzy-vpn dns-check [--json]        DNS privacy: in-country + encrypted?
  mazzy-vpn control id|pair|list      control-plane identity & pairing
  mazzy-vpn language [code|--list]    choose UI language (en/ru/de/zh/ja/ko)

 %s
  sudo mazzy-vpn disconnect           bring the tunnel down gracefully
  sudo mazzy-vpn recover              force-clean ALL tunnels/guards (panic button)
  sudo mazzy-vpn recover --reset-catalog   also wipe imported profiles

 %s
  mazzy-vpn doctor [--json]           host diagnostics (no awg/jq required)
  mazzy-vpn providers [--type llm|agent|search] [--json]   check AI providers
  mazzy-vpn list DIR [--json]         validate profiles in a directory
  mazzy-vpn validate FILE             validate a single profile
  mazzy-vpn verify [--no-dns] [--json]  audit ALL managed configs' health
  mazzy-vpn update [--apply]          check/install a newer release from GitHub
  mazzy-vpn version | help

Profiles: AmneziaWG (.conf), WireGuard (.conf), OpenVPN (.ovpn).
%s
`,
		t.T("cli.usage.tagline"),
		t.T("cli.usage.sec.profiles"),
		t.T("cli.usage.sec.connect"),
		t.T("cli.usage.sec.background"),
		t.T("cli.usage.sec.network"),
		t.T("cli.usage.sec.recovery"),
		t.T("cli.usage.sec.diagnostics"),
		t.T("cli.usage.footer"))
}
