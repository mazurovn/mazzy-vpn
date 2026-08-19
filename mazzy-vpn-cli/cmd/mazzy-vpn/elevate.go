// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// privilegedSubcommands are the mazzy-vpn actions that mutate the network and
// therefore need CAP_NET_ADMIN (root). Kept in one place so the menu, the TUI
// and any future GUI agree on exactly which actions must be elevated.
var privilegedSubcommands = map[string]bool{
	"up":         true,
	"connect":    true,
	"disconnect": true,
	"recover":    true,
	"auto":       true,
	"daemon":     true,
	"mimic":      true,
	"stop":       true,
}

// elevatorLookup is indirected for tests. It resolves a privilege helper on
// PATH (sudo, then pkexec) and reports its absolute path.
var elevatorLookup = func(name string) (string, bool) {
	p, err := exec.LookPath(name)
	if err != nil {
		return "", false
	}
	return p, true
}

// findElevator returns the first available privilege helper and its name.
// sudo is preferred (works in a terminal); pkexec is the graphical fallback.
func findElevator() (path, name string, ok bool) {
	for _, cand := range []string{"sudo", "pkexec"} {
		if p, found := elevatorLookup(cand); found {
			return p, cand, true
		}
	}
	return "", "", false
}

// runInvoker is indirected for tests so we can assert the exact command that
// would run without actually spawning a privileged process.
var runInvoker = func(path string, args []string) error {
	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// dispatchInProcess is indirected for tests; nil means "call run()". A var
// (not an initializer that references run) avoids a package init cycle.
var dispatchInProcess func(args []string) int

// runPrivileged executes a mazzy-vpn subcommand. When the action needs root and
// we are not root, it re-executes this same binary through sudo/pkexec so the
// menu/TUI can trigger connect/disconnect without the user leaving the UI. The
// child's exit status is surfaced (no silently-swallowed failures). When we are
// already root, it dispatches in-process.
//
// Returns the process exit code (0 = success).
func runPrivileged(_ context.Context, subcmd string, sargs ...string) int {
	args := append([]string{subcmd}, sargs...)

	if privilegedSubcommands[subcmd] && os.Geteuid() != 0 {
		self, err := os.Executable()
		if err != nil || self == "" {
			fmt.Fprintln(os.Stderr, translator().T("cli.elevate.no_self"))
			return 1
		}
		elevator, name, ok := findElevator()
		if !ok {
			fmt.Printf("%s\n", translator().Tf("cli.elevate.needs_root",
				safeDisplay(self+" "+joinArgs(args))))
			return 1
		}
		fmt.Println(translator().Tf("cli.elevate.via", name))
		full := append([]string{self}, args...)
		if err := runInvoker(elevator, full); err != nil {
			if code, ok := exitCode(err); ok {
				return code
			}
			fmt.Fprintln(os.Stderr, translator().Tf("cli.elevate.failed", safeDisplay(err.Error())))
			return 1
		}
		return 0
	}

	// Already privileged (or a read-only action): dispatch in-process.
	if dispatchInProcess != nil {
		return dispatchInProcess(args)
	}
	return run(args)
}

// buildPrivilegedCmd returns an *exec.Cmd that runs `mazzy-vpn <subcmd> ...`
// with root, elevating via sudo/pkexec when we are not already root. It returns
// (nil, error) when self-resolution or elevator discovery fails. This is used
// by the bubbletea TUI with tea.ExecProcess, which suspends the alt-screen so
// sudo/pkexec can prompt on the real terminal, then resumes the UI.
func buildPrivilegedCmd(subcmd string, sargs ...string) (*exec.Cmd, error) {
	args := append([]string{subcmd}, sargs...)
	self, err := os.Executable()
	if err != nil || self == "" {
		return nil, fmt.Errorf("cannot locate self executable")
	}
	if privilegedSubcommands[subcmd] && os.Geteuid() != 0 {
		elevator, _, ok := findElevator()
		if !ok {
			return nil, fmt.Errorf("no sudo/pkexec available")
		}
		return exec.Command(elevator, append([]string{self}, args...)...), nil
	}
	return exec.Command(self, args...), nil
}

// exitCoder is satisfied by *exec.ExitError (and by test doubles). Duck-typing
// keeps exitCode testable without spawning a real process.
type exitCoder interface{ ExitCode() int }

// exitCode extracts a child process exit code from an exit error.
func exitCode(err error) (int, bool) {
	if ee, ok := err.(exitCoder); ok {
		return ee.ExitCode(), true
	}
	return 0, false
}

// joinArgs joins args with single spaces for display in a copy-paste hint.
func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}
