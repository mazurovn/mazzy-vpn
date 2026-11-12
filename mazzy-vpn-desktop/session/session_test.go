// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mazurovn/mazzy-vpn/desktop/elevate"
	"github.com/mazurovn/mazzy-vpn/desktop/menu"
	"github.com/mazurovn/mazzy-vpn/desktop/monitor"
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

func TestConnectElevatesTheCLI(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "helsinki.conf")
	if err := os.WriteFile(src, []byte(sampleConf), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(Config{CatalogDir: filepath.Join(dir, "catalog"), StateDir: filepath.Join(dir, "state"), RunDir: filepath.Join(dir, "run")})
	if _, err := s.cat.Import(src); err != nil {
		t.Fatal(err)
	}
	var got []string
	s.launch = func(_ context.Context, args []string) (*elevate.Proc, error) {
		got = append([]string(nil), args...)
		return nil, fmt.Errorf("stopped before exec")
	}
	_, err := s.Connect(context.Background(), "helsinki")
	if err == nil || !strings.Contains(err.Error(), "stopped before exec") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "start Mazzy VPN Desktop") {
		t.Fatalf("window was told to become root: %v", err)
	}
	if len(got) != 2 || got[0] != "up" || got[1] != "helsinki" {
		t.Fatalf("args = %q", got)
	}
}

func TestLoadDoesNotInventMeasurements(t *testing.T) {
	dir := t.TempDir()
	s := New(Config{CatalogDir: dir, StateDir: dir, RunDir: dir})
	if _, err := s.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	board := s.Board()
	if board.Handshake != monitor.Unknown || board.Path != monitor.Unknown {
		t.Fatalf("lights = %s %s", board.Handshake, board.Path)
	}
	if len(board.Points) != 0 {
		t.Fatalf("points = %+v", board.Points)
	}
}

func TestDisabledActionDoesNotLaunch(t *testing.T) {
	dir := t.TempDir()
	s := New(Config{CatalogDir: dir, StateDir: dir, RunDir: dir})
	s.launch = func(context.Context, []string) (*elevate.Proc, error) {
		t.Fatal("disabled action launched a command")
		return nil, nil
	}
	s.Do(context.Background(), menu.Call{Label: "Зонд", Disabled: "команды probe нет"})
	if !strings.Contains(s.Log(), "команды probe нет") {
		t.Fatalf("log = %q", s.Log())
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
