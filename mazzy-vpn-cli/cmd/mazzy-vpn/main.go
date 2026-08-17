// SPDX-License-Identifier: AGPL-3.0-or-later
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
	"os"
)

// version is the CLI source line; the published tag governs releases.
const version = "2.0.0-dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	ctx := context.Background()
	// No args: launch the interactive menu (ergonomic default).
	if len(args) == 0 {
		return cmdMenu(ctx, nil)
	}
	cmd, rest := args[0], args[1:]

	switch cmd {
	case "menu":
		return cmdMenu(ctx, rest)
	case "doctor":
		return cmdDoctor(ctx, rest)
	case "list":
		return cmdList(ctx, rest)
	case "validate":
		return cmdValidate(ctx, rest)
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
	case "up":
		return cmdUp(ctx, rest)
	case "auto":
		return cmdAuto(ctx, rest)
	case "daemon":
		return cmdDaemon(ctx, rest)
	case "connect":
		return cmdConnect(ctx, rest)
	case "disconnect", "down":
		return cmdDisconnect(ctx, rest)
	case "recover", "clean", "panic":
		return cmdRecover(ctx, rest)
	case "providers", "ai":
		return cmdProviders(ctx, rest)
	case "status":
		return cmdStatus(ctx, rest)
	case "version", "--version", "-v":
		fmt.Println("mazzy-vpn", version)
		return 0
	case "help", "--help", "-h":
		printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		return 2
	}
}

func printUsage() {
	fmt.Fprint(os.Stderr, `mazzy-vpn — autonomous AI-ready VPN client (Go)

Usage:
  mazzy-vpn                           interactive menu (no args)

 Profiles (managed catalog):
  mazzy-vpn import <FILE|DIR>...      import profiles into the catalog
  mazzy-vpn profiles [--json]         list managed profiles
  mazzy-vpn favorite NAME [--off]     mark/unmark a favorite zone
  mazzy-vpn remove NAME               remove a managed profile

 Connect:
  sudo mazzy-vpn up [NAME|--best]     connect by name, or auto-pick best zone
  sudo mazzy-vpn auto                 rank zones + connect to the best (failover)
  sudo mazzy-vpn connect FILE         connect using a raw file path
  mazzy-vpn status [--json]           show the connection intent

  While connected, a live dashboard shows PROTECTED/LINK-UP status, sends
  desktop notifications (connected/reconnecting/disconnected), and
  auto-reconnects if the egress drops. Add --no-reconnect to disable.

 Permanent (background) operation:
  sudo mazzy-vpn daemon NAME          run persistently with auto-reconnect
  sudo systemctl enable --now mazzy-vpn@NAME   start at boot (systemd)

 Network:
  mazzy-vpn test [--json]             probe all servers (latency/reachability)
  mazzy-vpn best [--json]             print the best zone to connect to
  mazzy-vpn adapters [--json]         list network interfaces + recommendation
  mazzy-vpn netdiag [--json]          analyze the network + suggest fixes

 Recovery:
  sudo mazzy-vpn disconnect           bring the tunnel down gracefully
  sudo mazzy-vpn recover              force-clean ALL tunnels/guards (panic button)
  sudo mazzy-vpn recover --reset-catalog   also wipe imported profiles

 Diagnostics:
  mazzy-vpn doctor [--json]           host diagnostics (no awg/jq required)
  mazzy-vpn providers [--type llm|agent|search] [--json]   check AI providers
  mazzy-vpn list DIR [--json]         validate profiles in a directory
  mazzy-vpn validate FILE             validate a single profile
  mazzy-vpn version | help

Profiles: AmneziaWG (.conf), WireGuard (.conf), OpenVPN (.ovpn).
connect/up run in the foreground and hold the tunnel until Ctrl+C.
`)
}
