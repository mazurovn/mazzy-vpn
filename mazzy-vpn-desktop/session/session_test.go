// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package session

import (
	"context"
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

func TestConnectDoesNotElevate(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "helsinki.conf")
	if err := os.WriteFile(src, []byte(sampleConf), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(Config{CatalogDir: filepath.Join(dir, "catalog"), StateDir: filepath.Join(dir, "state"), RunDir: dir})
	if _, err := s.cat.Import(src); err != nil {
		t.Fatal(err)
	}
	s.launch = func(context.Context, []string) (*elevate.Proc, error) {
		t.Fatal("connect launched a process")
		return nil, nil
	}
	snap, err := s.Connect(context.Background(), "helsinki")
	if err != nil {
		t.Fatal(err)
	}
	if snap.Phase != "disconnected" {
		t.Fatalf("phase %s", snap.Phase)
	}
	if !strings.Contains(snap.Notice, "сокет") {
		t.Fatalf("notice %q", snap.Notice)
	}
	if strings.Contains(s.Log(), "Not authorized") || strings.Contains(s.Log(), "pkexec") {
		t.Fatalf("log %q", s.Log())
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
