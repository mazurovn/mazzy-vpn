// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mazurovn/mazzy-vpn/core"
)

func TestWriteReadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Dir: dir}
	in := State{
		Protocol: core.AmneziaWG,
		Profile:  "/some/path/berlin.conf", // must be stored as basename
		Desired:  core.DesiredUp,
		Mode:     core.ModeNormal,
	}
	if err := s.Write(in); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := s.Read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Profile != "berlin.conf" {
		t.Errorf("profile = %q, want basename berlin.conf", got.Profile)
	}
	if got.Protocol != core.AmneziaWG || got.Desired != core.DesiredUp {
		t.Errorf("got %+v", got)
	}
}

func TestFilePermissions(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Dir: dir}
	_ = s.Write(State{Protocol: core.WireGuard, Profile: "x.conf", Desired: core.DesiredUp})
	fi, err := os.Stat(filepath.Join(dir, "active"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("state file perm = %o, want 600", fi.Mode().Perm())
	}
	di, _ := os.Stat(dir)
	if di.Mode().Perm() != 0o700 {
		t.Errorf("dir perm = %o, want 700", di.Mode().Perm())
	}
}

func TestTestModePersistsTokenAndDeadline(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Dir: dir}
	in := State{
		Protocol: core.OpenVPN, Profile: "t.conf", Desired: core.DesiredUp,
		Mode: core.ModeTest, TestToken: "tok123", TestDeadline: 1700000000,
	}
	if err := s.Write(in); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Read()
	if got.Mode != core.ModeTest || got.TestToken != "tok123" || got.TestDeadline != 1700000000 {
		t.Errorf("test fields not persisted: %+v", got)
	}
}

func TestSetDesiredPreservesRest(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Dir: dir}
	_ = s.Write(State{Protocol: core.AmneziaWG, Profile: "p.conf", Desired: core.DesiredUp})
	if err := s.SetDesired(core.DesiredDown); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Read()
	if got.Desired != core.DesiredDown {
		t.Errorf("desired = %q, want down", got.Desired)
	}
	if got.Protocol != core.AmneziaWG || got.Profile != "p.conf" {
		t.Errorf("SetDesired clobbered other fields: %+v", got)
	}
}

func TestReadRejectsUnknownProtocol(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "active"), []byte("PROTOCOL=bogus\nPROFILE=x\n"), 0o600)
	s := &Store{Dir: dir}
	if _, err := s.Read(); err == nil {
		t.Fatal("expected error on unknown protocol")
	}
}
