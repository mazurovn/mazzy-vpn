// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// desiredIntent is the TUI→daemon control file (ADR-0006 D2). The unprivileged
// TUI writes it; the privileged daemon polls and applies it.
type desiredIntent struct {
	Zone    string `json:"zone"`
	Desired string `json:"desired"` // "up" | "down"
	TS      int64  `json:"ts"`
}

// desiredPath returns the per-user intent file path.
func desiredPath() string {
	if d := os.Getenv("MAZZY_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "desired.json")
	}
	if h, err := os.UserConfigDir(); err == nil {
		return filepath.Join(h, "mazzy-vpn", "desired.json")
	}
	return filepath.Join(os.TempDir(), "mazzy-vpn", "desired.json")
}

// writeDesired atomically records the desired connection intent.
func writeDesired(zone, desired string) error {
	p := desiredPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(desiredIntent{Zone: zone, Desired: desired, TS: time.Now().Unix()}, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// readDesired reads the intent file (used by the daemon).
func readDesired() (desiredIntent, bool) {
	data, err := os.ReadFile(desiredPath())
	if err != nil {
		return desiredIntent{}, false
	}
	var di desiredIntent
	if json.Unmarshal(data, &di) != nil {
		return desiredIntent{}, false
	}
	return di, true
}
