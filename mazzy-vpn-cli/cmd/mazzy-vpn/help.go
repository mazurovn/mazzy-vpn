// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// commandHelp is the side-effect-free help contract for one command. Keeping
// this data centralized lets `help CMD`, `CMD -h`, and `CMD --help` agree and
// gives parity tests a canonical command inventory.
type commandHelp struct {
	Usage   string
	Summary string
	Aliases []string
	Flags   []string
}

var commandHelpRegistry = map[string]commandHelp{
	"help":       {"mazzy-vpn help [COMMAND]", "Show root or command-specific help.", nil, nil},
	"tui":        {"mazzy-vpn tui", "Open the full-screen dashboard.", nil, nil},
	"menu":       {"mazzy-vpn menu", "Open the line-based fallback menu.", []string{"--plain"}, nil},
	"doctor":     {"mazzy-vpn doctor [--json] | sudo mazzy-vpn doctor --heal", "Check host health; --heal actively rescues the connection.", nil, []string{"--json", "--heal"}},
	"list":       {"mazzy-vpn list DIR [--json]", "Validate profiles in a directory.", nil, []string{"--json"}},
	"validate":   {"mazzy-vpn validate FILE [--proto PROTOCOL] [--json]", "Validate one profile.", nil, []string{"--proto", "--json"}},
	"verify":     {"mazzy-vpn verify [--no-dns] [--json]", "Audit all managed profiles.", []string{"audit"}, []string{"--no-dns", "--json"}},
	"import":     {"mazzy-vpn import <FILE|DIR>...", "Recursively import and classify profiles.", nil, nil},
	"profiles":   {"mazzy-vpn profiles [--ping] [--json]", "List managed profiles.", nil, []string{"--ping", "--json"}},
	"favorite":   {"mazzy-vpn favorite NAME [--off]", "Add or remove a favorite zone.", []string{"fav"}, []string{"--off"}},
	"remove":     {"mazzy-vpn remove NAME", "Remove a managed profile.", []string{"rm"}, nil},
	"test":       {"mazzy-vpn test [--json]", "Probe and rank all servers.", nil, []string{"--json"}},
	"probe":      {"sudo mazzy-vpn probe [NAME...|--all] [--deep] [--json]", "HARD deep test: real connect + egress + tx/rx per zone; --deep adds link quality/stability.", []string{"deeptest"}, []string{"--all", "--deep", "--json"}},
	"best":       {"mazzy-vpn best [--json]", "Print the best reachable zone.", nil, []string{"--json"}},
	"adapters":   {"mazzy-vpn adapters [--json]", "List network interfaces and recommendation.", []string{"interfaces"}, []string{"--json"}},
	"netdiag":    {"mazzy-vpn netdiag [--json]", "Analyze the current network.", []string{"analyze"}, []string{"--json"}},
	"diagnose":   {"mazzy-vpn diagnose [--json]", "Explain likely connection failures.", []string{"why"}, []string{"--json"}},
	"trace":      {"mazzy-vpn trace [ZONE] [--json]", "Trace DNS, endpoint, tunnel and egress.", nil, []string{"--json"}},
	"stealth":    {"mazzy-vpn stealth [--json]", "Check detection and leak risks.", []string{"leaks"}, []string{"--json"}},
	"mimic":      {"sudo mazzy-vpn mimic [--apply]", "Align host timezone with VPN egress.", []string{"timezone"}, []string{"--apply"}},
	"dns-check":  {"mazzy-vpn dns-check [--dot] [--json]", "Check DNS privacy.", []string{"dns"}, []string{"--dot", "--json"}},
	"control":    {"mazzy-vpn control id|pair|list", "Manage control-plane identity and pairing.", []string{"identity"}, nil},
	"language":   {"mazzy-vpn language [CODE|--list]", "Show or change the UI language.", []string{"lang"}, []string{"--list"}},
	"up":         {"sudo mazzy-vpn up [NAME|--best|--clean] [--uplink IF]", "Connect using a managed profile.", nil, []string{"--best", "--clean", "--uplink", "--no-diagnostics", "--no-reconnect"}},
	"auto":       {"sudo mazzy-vpn auto", "Rank zones and start failover mode.", nil, nil},
	"reconnect":  {"sudo mazzy-vpn reconnect [NAME]", "Drop the tunnel NOW and reconnect to the best proven-working zone.", nil, nil},
	"daemon":     {"sudo mazzy-vpn daemon <NAME|--best> [--background|--session]", "Run self-healing connection mode.", nil, []string{"--best", "--background", "--session"}},
	"stop":       {"sudo mazzy-vpn stop", "Stop a running Mazzy VPN daemon.", nil, nil},
	"connect":    {"sudo mazzy-vpn connect FILE [--uplink IF] [--no-reconnect]", "Connect using a raw profile path.", nil, []string{"--uplink", "--no-reconnect"}},
	"disconnect": {"sudo mazzy-vpn disconnect", "Bring down the active tunnel.", []string{"down"}, nil},
	"recover":    {"sudo mazzy-vpn recover [--reset-catalog] [--purge-legacy]", "Force-clean tunnels, guards and legacy leftovers.", []string{"clean", "panic"}, []string{"--reset-catalog", "--purge-legacy"}},
	"trust":      {"sudo mazzy-vpn trust [--revoke] [--user NAME]", "Allow passwordless daemon control via a sudoers drop-in.", nil, []string{"--revoke", "--user"}},
	"disarm":     {"sudo mazzy-vpn disarm [--purge-legacy]", "HARD reset: kill daemon, drop all rules/kill-switch, restore DNS, verify internet.", []string{"hard-reset"}, []string{"--purge-legacy"}},
	"providers":  {"mazzy-vpn providers [--type TYPE] [--json]", "Check configured AI providers.", []string{"ai"}, []string{"--type", "--json"}},
	"update":     {"mazzy-vpn update [--apply]", "Check or install a GitHub release.", []string{"self-update"}, []string{"--apply"}},
	"status":     {"mazzy-vpn status [--json]", "Show live connection state.", nil, []string{"--json"}},
	"version":    {"mazzy-vpn version", "Print the CLI version.", []string{"-v", "--version"}, nil},
}

func canonicalHelpCommand(name string) string {
	if _, ok := commandHelpRegistry[name]; ok {
		return name
	}
	for canonical, spec := range commandHelpRegistry {
		for _, alias := range spec.Aliases {
			if name == alias {
				return canonical
			}
		}
	}
	return ""
}

// printCommandHelp writes command help to stdout and returns a shell exit code.
// Unknown commands are an error on stderr and never invoke a handler.
func printCommandHelp(name string) int {
	canonical := canonicalHelpCommand(name)
	if canonical == "" {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", safeDisplay(name))
		return 2
	}
	writeCommandHelp(os.Stdout, canonical, commandHelpRegistry[canonical])
	return 0
}

func writeCommandHelp(w io.Writer, name string, spec commandHelp) {
	fmt.Fprintf(w, "%s — %s\n\nUsage:\n  %s\n", name, spec.Summary, spec.Usage)
	if len(spec.Aliases) > 0 {
		fmt.Fprintf(w, "\nAliases: %s\n", strings.Join(spec.Aliases, ", "))
	}
	if len(spec.Flags) > 0 {
		fmt.Fprintln(w, "\nOptions:")
		for _, flag := range spec.Flags {
			fmt.Fprintf(w, "  %s\n", flag)
		}
	}
	fmt.Fprintln(w, "  -h, --help")
}

func registeredCommandNames() []string {
	names := make([]string, 0, len(commandHelpRegistry))
	for name := range commandHelpRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
