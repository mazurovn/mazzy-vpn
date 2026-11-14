// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package elevate builds the argv for one CLI command.
//
// The line menu's privileged path is `sudo mazzy-vpn …` (up, connect,
// disconnect, recover, auto, daemon). The window itself is not root. In a
// graphical session sudo has no terminal to ask for a password, so the same
// command is handed to pkexec, which is how the existing Linux desktop runs
// those CLI operations.
package elevate

import (
	"fmt"
	"strings"
)

// Privileged reports whether the CLI menu treats sub as a root command.
func Privileged(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "up", "connect", "disconnect", "recover", "auto", "daemon":
		return true
	case "mimic", "update":
		for _, a := range args[1:] {
			if a == "--apply" {
				return true
			}
		}
	}
	return false
}

// Plan is the process argv for one CLI invocation.
type Plan struct {
	Argv []string
	Via  string // direct, sudo, or pkexec
}

// Paths resolves a command name to an absolute path. Empty means absent.
type Paths func(name string) string

// Command chooses how to run args. cli and the helpers must already be paths.
func Command(euid int, graphical bool, look Paths, args []string) (Plan, error) {
	if len(args) == 0 {
		return Plan{}, fmt.Errorf("empty command")
	}
	if look == nil {
		look = func(string) string { return "" }
	}
	cli := look("mazzy-vpn")
	if cli == "" {
		return Plan{}, fmt.Errorf("mazzy-vpn CLI is not installed")
	}
	if !Privileged(args) || euid == 0 {
		return Plan{Argv: append([]string{cli}, args...), Via: "direct"}, nil
	}
	shown := "sudo " + cli + " " + strings.Join(args, " ")
	if graphical {
		if pk := look("pkexec"); pk != "" {
			return Plan{Argv: append([]string{pk, cli}, args...), Via: "pkexec"}, nil
		}
	}
	if sudo := look("sudo"); sudo != "" {
		argv := append([]string{sudo, "--", cli}, args...)
		return Plan{Argv: argv, Via: "sudo"}, nil
	}
	return Plan{}, fmt.Errorf("needs root; run: %s", shown)
}

// ForwardEnv inserts env assignments so a privileged command still sees the
// desktop's catalog. sudo and pkexec clear the caller's environment, which
// makes `mazzy-vpn up NAME` look at root's empty catalog.
func ForwardEnv(plan Plan, kv []string, look Paths) Plan {
	if plan.Via == "direct" || len(kv) == 0 || len(plan.Argv) == 0 {
		return plan
	}
	envBin := ""
	if look != nil {
		envBin = look("env")
	}
	if envBin == "" {
		return plan
	}
	switch plan.Via {
	case "sudo":
		if len(plan.Argv) < 3 || plan.Argv[1] != "--" {
			return plan
		}
		argv := []string{plan.Argv[0], "--", envBin}
		argv = append(argv, kv...)
		argv = append(argv, plan.Argv[2:]...)
		plan.Argv = argv
	case "pkexec":
		argv := []string{plan.Argv[0], envBin}
		argv = append(argv, kv...)
		argv = append(argv, plan.Argv[1:]...)
		plan.Argv = argv
	}
	return plan
}
