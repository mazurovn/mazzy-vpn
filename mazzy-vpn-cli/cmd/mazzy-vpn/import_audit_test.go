// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mazurovn/mazzy-vpn/core"
)

// TestImportStemNormalization locks the WG-over-OVPN dedup key: the same logical
// server must map to one stem regardless of separators/case/extension.
func TestImportStemNormalization(t *testing.T) {
	cases := map[string]string{
		"Austria--Vienna-S1.ovpn": "austriaviennas1",
		"AustriaViennaS1.conf":    "austriaviennas1",
		"/x/y/Foo_Bar.conf":       "foobar",
		"foo.bar.baz.ovpn":        "foobarbaz",
	}
	for in, want := range cases {
		if got := importStem(in); got != want {
			t.Errorf("importStem(%q) = %q, want %q", in, got, want)
		}
	}
	// The paired .ovpn and .conf must collide (that is what enables dedup).
	if importStem("Austria--Vienna-S1.ovpn") != importStem("AustriaViennaS1.conf") {
		t.Error("ovpn and conf twins must share a stem for dedup")
	}
}

// TestCollectProfileFilesRecurses verifies the importer walks nested provider
// bundles and only picks up .conf/.ovpn files.
func TestCollectProfileFilesRecurses(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.conf"), "x")
	mustWrite(t, filepath.Join(root, "sub", "b.ovpn"), "x")
	mustWrite(t, filepath.Join(root, "sub", "deep", "c.conf"), "x")
	mustWrite(t, filepath.Join(root, "readme.txt"), "ignore me")

	files, err := collectProfileFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 profile files (recursive), got %d: %v", len(files), files)
	}
	for _, f := range files {
		ext := filepath.Ext(f)
		if ext != ".conf" && ext != ".ovpn" {
			t.Errorf("non-profile file picked up: %s", f)
		}
	}
}

// TestAuditProfileOpenVPNIsWarnNotConnectable verifies OpenVPN entries are
// reported as catalogued-but-not-connectable (a WARN, not a hard FAIL).
func TestAuditProfileOpenVPNIsWarnNotConnectable(t *testing.T) {
	a := auditProfile(context.Background(), catalogEntry{
		Name: "Chile--Santiago", Protocol: core.OpenVPN, Country: "CL",
	}, false)
	if a.Connectable {
		t.Error("OpenVPN must not be reported connectable")
	}
	if a.healthy() {
		t.Error("OpenVPN must not be reported healthy (not connectable yet)")
	}
	_, glyph := a.verdict()
	if glyph != "▲" {
		t.Errorf("OpenVPN verdict glyph = %q, want ▲ (warn)", glyph)
	}
}

// TestAuditProfileBrokenConfIsFail verifies an unparseable/invalid WG profile is
// a hard FAIL that makes `verify` exit non-zero.
func TestAuditProfileBrokenConfIsFail(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "broken.conf")
	mustWrite(t, bad, "this is not a wireguard profile")
	a := auditProfile(context.Background(), catalogEntry{
		Name: "broken", File: bad, Protocol: core.WireGuard,
	}, false)
	if a.Valid {
		t.Error("a garbage .conf must not validate")
	}
	if exitForAudits([]profileAudit{a}) != 1 {
		t.Error("a broken profile must make verify exit non-zero")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
