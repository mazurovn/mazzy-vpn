// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mazurovn/mazzy-vpn/core"
)

const sampleAWG = `[Interface]
PrivateKey = AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=
Address = 10.8.0.2/32
Jc = 4
Jmin = 40
Jmax = 70
S1 = 50
S2 = 100
H1 = 1
H2 = 2
H3 = 3
H4 = 4
[Peer]
PublicKey = AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=
AllowedIPs = 0.0.0.0/0
Endpoint = vpn.example.com:51820
`

func writeTmp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestImportAndList(t *testing.T) {
	c := New(t.TempDir())
	src := writeTmp(t, "AustriaGrazS6.conf", sampleAWG)
	e, err := c.Import(src)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if e.Protocol != core.AmneziaWG {
		t.Errorf("protocol = %s, want amneziawg", e.Protocol)
	}
	if e.Country != "AT" {
		t.Errorf("country = %q, want AT (inferred from Austria)", e.Country)
	}
	list, _ := c.List()
	if len(list) != 1 || list[0].Name != "AustriaGrazS6" {
		t.Fatalf("list = %+v", list)
	}
}

func TestImportRejectsInvalid(t *testing.T) {
	c := New(t.TempDir())
	src := writeTmp(t, "bad.conf", "[Interface]\nAddress = 10.0.0.1/32\n")
	if _, err := c.Import(src); err == nil {
		t.Fatal("expected rejection of invalid AmneziaWG profile")
	}
}

func TestImportDedupByContent(t *testing.T) {
	c := New(t.TempDir())
	a := writeTmp(t, "Germany.conf", sampleAWG)
	b := writeTmp(t, "GermanyCopy.conf", sampleAWG)
	if _, err := c.Import(a); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Import(b); err == nil {
		t.Fatal("expected content-duplicate rejection")
	}
}

func TestFavoriteSortsFirst(t *testing.T) {
	c := New(t.TempDir())
	c.Import(writeTmp(t, "Zzz.conf", sampleAWG))
	// second import with different content
	other := sampleAWG[:len(sampleAWG)-1] + "\n#unique\n"
	c.Import(writeTmp(t, "Aaa.conf", other))
	if err := c.SetFavorite("Zzz", true); err != nil {
		t.Fatal(err)
	}
	list, _ := c.List()
	if list[0].Name != "Zzz" {
		t.Errorf("favorite should sort first, got %s", list[0].Name)
	}
}

func TestRemove(t *testing.T) {
	c := New(t.TempDir())
	c.Import(writeTmp(t, "France.conf", sampleAWG))
	if err := c.Remove("France"); err != nil {
		t.Fatal(err)
	}
	if c.Count() != 0 {
		t.Error("catalog should be empty after remove")
	}
	if err := c.Remove("France"); err != ErrNotFound {
		t.Error("removing missing entry should be ErrNotFound")
	}
}

func TestInferCountry(t *testing.T) {
	cases := map[string]string{
		"AustriaGrazS6": "AT", "NetherlandsAmsterdam": "NL",
		"usa-newyork": "US", "UnitedKingdomLondon": "GB", "random": "",
	}
	for name, want := range cases {
		if got := inferCountry(name); got != want {
			t.Errorf("inferCountry(%q) = %q, want %q", name, got, want)
		}
	}
}

// TestGetRejectsTraversalNames guards the path-injection hardening: a lookup
// name with separators or ".." must never resolve, and a manifest whose stored
// File escapes the managed dir must be refused.
func TestGetRejectsTraversalNames(t *testing.T) {
	c := &Catalog{Dir: t.TempDir()}
	for _, bad := range []string{"../etc/shadow", "a/b", "..", "", `x\y`} {
		if _, err := c.Get(bad); err == nil {
			t.Errorf("Get(%q) must fail", bad)
		}
	}
}

// TestValidLookupName unit-checks the name gate.
func TestValidLookupName(t *testing.T) {
	for _, ok := range []string{"Berlin", "USA-NewYork_S1", "zone123"} {
		if !validLookupName(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", ".", "..", "a/b", "a\\b", "../x", "x/../y"} {
		if validLookupName(bad) {
			t.Errorf("%q must be rejected", bad)
		}
	}
}

// TestFileWithinDir checks the containment predicate.
func TestFileWithinDir(t *testing.T) {
	dir := t.TempDir()
	c := &Catalog{Dir: dir}
	if !c.fileWithinDir(filepath.Join(dir, "Berlin.conf")) {
		t.Error("in-dir file must be within")
	}
	if c.fileWithinDir("/etc/shadow") {
		t.Error("/etc/shadow must not be within")
	}
	if c.fileWithinDir(filepath.Join(dir, "..", "escape.conf")) {
		t.Error("parent-escape must not be within")
	}
}
