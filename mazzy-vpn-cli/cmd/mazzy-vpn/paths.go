// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"os"

	"github.com/mazurovn/mazzy-vpn/core/state"
)

// stateDir returns the persistent state directory. It honors MAZZY_STATE_DIR
// for tests/dev and falls back to the system location. Parity with the bash
// VPNCTL_STATE_DIR (defaulting to /var/lib/mazzy-vpn).
func stateDir() string {
	if d := os.Getenv("MAZZY_STATE_DIR"); d != "" {
		return d
	}
	return "/var/lib/mazzy-vpn"
}

// runDir returns the runtime (lock) directory. Honors MAZZY_RUN_DIR.
func runDir() string {
	if d := os.Getenv("MAZZY_RUN_DIR"); d != "" {
		return d
	}
	return "/run/mazzy-vpn"
}

// newStore builds the state.Store for the CLI.
func newStore() *state.Store {
	return &state.Store{Dir: stateDir()}
}

// lockDir is where the single-flight mutation lock lives.
func lockDir() string {
	return runDir()
}
