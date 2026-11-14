// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package elevate

import (
	"strings"
	"testing"
)

func look(names ...string) Paths {
	set := map[string]bool{}
	for _, n := range names {
		set[n] = true
	}
	return func(name string) string {
		if set[name] {
			return "/usr/bin/" + name
		}
		return ""
	}
}

func TestReadOnlyStaysDirect(t *testing.T) {
	p, err := Command(1000, true, look("mazzy-vpn", "pkexec", "sudo"), []string{"status"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Via != "direct" || p.Argv[0] != "/usr/bin/mazzy-vpn" || p.Argv[1] != "status" {
		t.Fatalf("%+v", p)
	}
}

func TestGraphicalPrivilegedUsesPkexec(t *testing.T) {
	p, err := Command(1000, true, look("mazzy-vpn", "pkexec", "sudo"), []string{"up", "helsinki"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Via != "pkexec" || p.Argv[0] != "/usr/bin/pkexec" || p.Argv[2] != "up" {
		t.Fatalf("%+v", p)
	}
}

func TestPlainPrivilegedUsesSudo(t *testing.T) {
	p, err := Command(1000, false, look("mazzy-vpn", "sudo"), []string{"disconnect"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Via != "sudo" || p.Argv[1] != "--" || p.Argv[3] != "disconnect" {
		t.Fatalf("%+v", p)
	}
}

func TestRootSkipsElevation(t *testing.T) {
	p, err := Command(0, true, look("mazzy-vpn", "pkexec"), []string{"recover"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Via != "direct" {
		t.Fatalf("%+v", p)
	}
}

func TestForwardEnvSurvivesSudoReset(t *testing.T) {
	p, err := Command(1000, false, look("mazzy-vpn", "sudo"), []string{"up", "berlin"})
	if err != nil {
		t.Fatal(err)
	}
	p = ForwardEnv(p, []string{"MAZZY_CONFIG_DIR=/tmp/catalog"}, look("env"))
	if len(p.Argv) < 6 || p.Argv[2] != "/usr/bin/env" || p.Argv[3] != "MAZZY_CONFIG_DIR=/tmp/catalog" || p.Argv[4] != "/usr/bin/mazzy-vpn" {
		t.Fatalf("%q", p.Argv)
	}
}

func TestMissingCLI(t *testing.T) {
	_, err := Command(1000, true, look("pkexec"), []string{"up", "helsinki"})
	if err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("err = %v", err)
	}
}
