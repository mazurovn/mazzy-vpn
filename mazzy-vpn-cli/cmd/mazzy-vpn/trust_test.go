// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSudoersContentShape pins the generated rule: one line per user, absolute
// binary, every trusted subcommand in bare AND wildcard form (sudo's `cmd *`
// does not match zero args), and nothing outside the trusted set.
func TestSudoersContentShape(t *testing.T) {
	c := sudoersContent("mazurov", "/usr/local/bin/mazzy-vpn")
	if !strings.Contains(c, "mazurov ALL=(root) NOPASSWD: ") {
		t.Fatalf("missing NOPASSWD spec:\n%s", c)
	}
	for _, sc := range trustedSubcommands {
		if !strings.Contains(c, "/usr/local/bin/mazzy-vpn "+sc+",") && !strings.Contains(c, "/usr/local/bin/mazzy-vpn "+sc+"\n") {
			t.Errorf("bare form for %q missing", sc)
		}
		if !strings.Contains(c, "/usr/local/bin/mazzy-vpn "+sc+" *") {
			t.Errorf("wildcard form for %q missing", sc)
		}
	}
	// `connect` must NEVER appear: it takes a raw path and would become a
	// root file-read oracle under NOPASSWD (security gate finding #3).
	for _, forbidden := range []string{" import", " remove", " update", " connect"} {
		if strings.Contains(c, "/usr/local/bin/mazzy-vpn"+forbidden) {
			t.Errorf("untrusted subcommand%q leaked into the rule", forbidden)
		}
	}
	if !strings.HasSuffix(c, "\n") {
		t.Error("sudoers file must end with a newline")
	}
}

// TestValidSudoersName rejects anything that could inject sudoers syntax via a
// crafted SUDO_USER.
func TestValidSudoersName(t *testing.T) {
	for _, good := range []string{"mazurov", "a", "user_1", "svc-account", "worker$"} {
		if !validSudoersName(good) {
			t.Errorf("%q should be accepted", good)
		}
	}
	for _, bad := range []string{"", "ALL", "User", "a b", "x,y", "u\nv", "root ALL=(ALL)", strings.Repeat("a", 40)} {
		if validSudoersName(bad) {
			t.Errorf("%q must be rejected", bad)
		}
	}
}

// TestBinarySafeForNopasswdRejectsWritable is the escalation guard: a NOPASSWD
// rule may never point at a binary the user can replace.
func TestBinarySafeForNopasswdRejectsWritable(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "mazzy-vpn")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// World-writable binary must be rejected regardless of ownership.
	if err := os.Chmod(bin, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := binarySafeForNopasswd(bin); err == nil {
		t.Error("world-writable binary must be rejected")
	}
	// 0755 but owned by the (non-root) test user must also be rejected.
	if os.Geteuid() != 0 {
		if err := os.Chmod(bin, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := binarySafeForNopasswd(bin); err == nil {
			t.Error("non-root-owned binary must be rejected")
		}
	}
}
