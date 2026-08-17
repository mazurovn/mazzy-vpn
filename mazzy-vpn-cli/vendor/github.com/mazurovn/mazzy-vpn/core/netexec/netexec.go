// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package netexec centralizes execution of base OS network tools (ip, nft,
// resolvectl) for mazzy-core. Per ADR-0005 these base iproute2/nftables tools
// are part of the OS, not installable VPN dependencies.
//
// Invariant: NEVER use a shell. Always exec.Command(bin, args...) with a
// validated argument vector, so profile-derived values can never inject.
package netexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Runner executes network commands. It is an interface so tests can inject a
// fake and so no real kernel changes happen during unit tests.
type Runner interface {
	Run(ctx context.Context, bin string, args ...string) (stdout string, err error)
}

// ExecRunner is the production Runner using os/exec with no shell.
type ExecRunner struct {
	// Timeout bounds each command. Zero means 15s.
	Timeout time.Duration
}

// Run executes bin with args and no shell. stderr is folded into the error.
func (r ExecRunner) Run(ctx context.Context, bin string, args ...string) (string, error) {
	if err := validateArgv(bin, args); err != nil {
		return "", err
	}
	to := r.Timeout
	if to == 0 {
		to = 15 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()

	cmd := exec.CommandContext(cctx, bin, args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	if cctx.Err() == context.DeadlineExceeded {
		return out.String(), fmt.Errorf("%s timed out after %s", bin, to)
	}
	if err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return out.String(), fmt.Errorf("%s %s: %s", bin, strings.Join(args, " "), msg)
	}
	return out.String(), nil
}

// validateArgv rejects empty binaries and any argument containing a NUL or
// newline (defense in depth; exec already avoids shell parsing).
func validateArgv(bin string, args []string) error {
	if strings.TrimSpace(bin) == "" {
		return errors.New("empty command")
	}
	for _, a := range args {
		if strings.ContainsAny(a, "\x00\n") {
			return fmt.Errorf("illegal control character in argument %q", a)
		}
	}
	return nil
}

// Available reports whether bin is resolvable in PATH.
func Available(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
