// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultHasRecommended(t *testing.T) {
	d := Default()
	if !d.AutoDiagnostics || !d.Notifications || !d.AutoReconnect || !d.KillSwitch {
		t.Errorf("recommended defaults not set: %+v", d)
	}
	if d.AutoConnect {
		t.Error("auto-connect should default off")
	}
}

func TestLoadMissingReturnsDefault(t *testing.T) {
	s := &Store{Path: filepath.Join(t.TempDir(), "nope.json")}
	got := s.Load()
	if got != Default() {
		t.Errorf("missing file should load defaults, got %+v", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	s := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	in := Settings{
		AutoConnect: true, AutoDiagnostics: false, Notifications: true,
		AutoReconnect: true, PreferredZone: "NetherlandsAmsterdamH4",
		PreferredUplink: "wlp3s0", KillSwitch: true,
	}
	if err := s.Save(in); err != nil {
		t.Fatal(err)
	}
	got := s.Load()
	if got != in {
		t.Errorf("round trip mismatch:\n got %+v\nwant %+v", got, in)
	}
}

func TestLoadCorruptReturnsDefault(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.json")
	s := &Store{Path: p}
	_ = os.WriteFile(p, []byte("{not json"), 0o600)
	if s.Load() != Default() {
		t.Error("corrupt file should fall back to defaults")
	}
}
