// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadShowsImportedProfile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "helsinki.conf")
	if err := os.WriteFile(src, []byte(sampleConf), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(Config{CatalogDir: filepath.Join(dir, "catalog"), StateDir: filepath.Join(dir, "state"), RunDir: filepath.Join(dir, "run")})
	if _, err := s.cat.Import(src); err != nil {
		t.Fatal(err)
	}
	snap, err := s.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Profiles) != 1 || snap.Profiles[0].Name != "helsinki" {
		t.Fatalf("profiles = %+v", snap.Profiles)
	}
	if snap.Profiles[0].Protocol != "WireGuard" {
		t.Fatalf("protocol = %s", snap.Profiles[0].Protocol)
	}
	if snap.Profiles[0].Endpoint != "203.0.113.10:51820" {
		t.Fatalf("endpoint = %s", snap.Profiles[0].Endpoint)
	}
	if snap.Phase != "disconnected" {
		t.Fatalf("phase = %s", snap.Phase)
	}
}

func TestConnectRequiresRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root")
	}
	dir := t.TempDir()
	s := New(Config{CatalogDir: dir, StateDir: dir, RunDir: dir})
	_, err := s.Connect(context.Background(), "helsinki")
	if err == nil || !strings.Contains(err.Error(), "root") {
		t.Fatalf("err = %v", err)
	}
}

const sampleConf = `[Interface]
PrivateKey = AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=
Address = 10.8.0.2/32

[Peer]
PublicKey = AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=
AllowedIPs = 0.0.0.0/0
Endpoint = 203.0.113.10:51820
`
