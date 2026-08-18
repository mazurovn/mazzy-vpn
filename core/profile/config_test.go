// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package profile

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/mazurovn/mazzy-vpn/core"
)

// A valid 32-byte key in base64 for tests.
func testKey(fill byte) string {
	b := make([]byte, 32)
	for i := range b {
		b[i] = fill
	}
	return base64.StdEncoding.EncodeToString(b)
}

func amneziaConf() string {
	return "[Interface]\n" +
		"PrivateKey = " + testKey(1) + "\n" +
		"Address = 10.8.0.2/32\n" +
		"DNS = 1.1.1.1\n" +
		"MTU = 1420\n" +
		"Jc = 4\nJmin = 40\nJmax = 70\nS1 = 50\nS2 = 100\n" +
		"H1 = 1\nH2 = 2\nH3 = 3\nH4 = 4\n" +
		"[Peer]\n" +
		"PublicKey = " + testKey(2) + "\n" +
		"PresharedKey = " + testKey(3) + "\n" +
		"AllowedIPs = 0.0.0.0/0, ::/0\n" +
		"Endpoint = vpn.example.com:51820\n" +
		"PersistentKeepalive = 25\n"
}

func TestParseAmneziaFull(t *testing.T) {
	c, err := Parse(amneziaConf())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(c.Addresses) != 1 || c.Addresses[0] != "10.8.0.2/32" {
		t.Errorf("addresses = %v", c.Addresses)
	}
	if c.MTU != 1420 {
		t.Errorf("mtu = %d", c.MTU)
	}
	if v, ok := c.Amnezia.Get("jc"); !c.HasAmneziaFields || !ok || v != "4" {
		t.Errorf("amnezia Jc not parsed: %q ok=%v", v, ok)
	}
	if len(c.Peers) != 1 {
		t.Fatalf("peers = %d", len(c.Peers))
	}
	p := c.Peers[0]
	if len(p.AllowedIPs) != 2 {
		t.Errorf("allowedips = %v", p.AllowedIPs)
	}
	if p.Endpoint != "vpn.example.com:51820" {
		t.Errorf("endpoint = %q", p.Endpoint)
	}
	if p.PersistentKeepalive != 25 {
		t.Errorf("keepalive = %d", p.PersistentKeepalive)
	}
}

func TestParseForwardCompatFields(t *testing.T) {
	// AmneziaWG 1.5 fields must be accepted (audit N2) and forwarded to UAPI,
	// not rejected as unknown keys.
	conf := "[Interface]\nPrivateKey = " + testKey(1) + "\nAddress = 10.0.0.2/32\n" +
		"Jc = 4\nJmin = 40\nJmax = 70\nS1 = 50\nS2 = 100\nH1 = 1\nH2 = 2\nH3 = 3\nH4 = 4\n" +
		"header_protection_key = " + testKey(9) + "\ncontent_padding_addition = 10-20\n" +
		"[Peer]\nPublicKey = " + testKey(2) + "\nEndpoint = 1.2.3.4:51820\nAllowedIPs = 0.0.0.0/0\n"
	c, err := Parse(conf)
	if err != nil {
		t.Fatalf("parse must accept forward-compat fields: %v", err)
	}
	if v, ok := c.Amnezia.Get("content_padding_addition"); !ok || v != "10-20" {
		t.Errorf("content_padding_addition not parsed: %q ok=%v", v, ok)
	}
	uapi, err := ToUAPI(core.AmneziaWG, c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(uapi, "header_protection_key=") {
		t.Error("header_protection_key must be forwarded to UAPI")
	}
	if !strings.Contains(uapi, "content_padding_addition=10-20") {
		t.Error("content_padding_addition must be forwarded to UAPI")
	}
}

func TestValidateAmneziaOK(t *testing.T) {
	c, _ := Parse(amneziaConf())
	if problems := Validate(core.AmneziaWG, c); len(problems) != 0 {
		t.Errorf("expected valid, got %v", problems)
	}
}

func TestValidateAmneziaMissingObfuscation(t *testing.T) {
	// A wireguard-only config must fail AmneziaWG validation.
	conf := "[Interface]\nPrivateKey = " + testKey(1) + "\nAddress = 10.0.0.2/32\n" +
		"[Peer]\nPublicKey = " + testKey(2) + "\nAllowedIPs = 0.0.0.0/0\n" +
		"Endpoint = h:1\n"
	c, err := Parse(conf)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	problems := Validate(core.AmneziaWG, c)
	if len(problems) == 0 {
		t.Fatal("expected AmneziaWG validation to fail without obfuscation params")
	}
	if !strings.Contains(strings.Join(problems, " "), "AmneziaWG parameter") {
		t.Errorf("expected obfuscation complaint, got %v", problems)
	}
	// The same config is valid as plain WireGuard.
	if p := Validate(core.WireGuard, c); len(p) != 0 {
		t.Errorf("wireguard should be valid, got %v", p)
	}
}

// TestParseRealWorldAmneziaFields locks the fix for real AmneziaWG configs:
// S3/S4 ints, i1..i5 obfuscation chains, and H1..H4 as integer RANGES must all
// parse and forward verbatim to UAPI (amneziawg-go is the authoritative parser).
func TestParseRealWorldAmneziaFields(t *testing.T) {
	conf := "[Interface]\n" +
		"PrivateKey = " + testKey(1) + "\nAddress = 10.1.2.3/32\n" +
		"Jc = 3\nJmin = 50\nJmax = 100\nS1 = 37\nS2 = 56\nS3 = 31\nS4 = 7\n" +
		"H1 = 127765534-127831069\nH2 = 811679222-811744757\n" +
		"H3 = 2069862060-2069927595\nH4 = 822734577-822800112\n" +
		"i1 = <b 0xdeadbeef>\n" +
		"[Peer]\nPublicKey = " + testKey(2) + "\nAllowedIPs = 0.0.0.0/0\nEndpoint = h:51820\n"
	c, err := Parse(conf)
	if err != nil {
		t.Fatalf("parse real-world amnezia: %v", err)
	}
	if problems := Validate(core.AmneziaWG, c); len(problems) != 0 {
		t.Fatalf("validate: %v", problems)
	}
	// H ranges and i1 chain preserved verbatim.
	if v, _ := c.Amnezia.Get("h1"); v != "127765534-127831069" {
		t.Errorf("H1 range not preserved: %q", v)
	}
	if v, _ := c.Amnezia.Get("i1"); v != "<b 0xdeadbeef>" {
		t.Errorf("i1 chain not preserved: %q", v)
	}
	uapi, err := ToUAPI(core.AmneziaWG, c)
	if err != nil {
		t.Fatalf("uapi: %v", err)
	}
	for _, want := range []string{"s3=31", "s4=7", "h1=127765534-127831069", "i1=<b 0xdeadbeef>"} {
		if !strings.Contains(uapi, want) {
			t.Errorf("UAPI missing %q", want)
		}
	}
}

func TestParseRejectsExecutableDirective(t *testing.T) {
	conf := "[Interface]\nPrivateKey = x\nPostUp = rm -rf /\n"
	if _, err := Parse(conf); err == nil {
		t.Fatal("expected rejection of PostUp executable directive")
	}
}

func TestToUAPIConvertsBase64ToHex(t *testing.T) {
	c, _ := Parse(amneziaConf())
	uapi, err := ToUAPI(core.AmneziaWG, c)
	if err != nil {
		t.Fatalf("uapi: %v", err)
	}
	// key of 32 bytes 0x01 -> hex is 64 chars of "01".
	wantPriv := "private_key=" + strings.Repeat("01", 32)
	if !strings.Contains(uapi, wantPriv) {
		t.Errorf("private_key not hex-encoded in UAPI:\n%s", uapi)
	}
	for _, want := range []string{"jc=4", "jmin=40", "jmax=70", "s1=50", "s2=100",
		"h1=1", "h2=2", "h3=3", "h4=4", "replace_peers=true",
		"endpoint=vpn.example.com:51820", "persistent_keepalive_interval=25",
		"allowed_ip=0.0.0.0/0", "allowed_ip=::/0"} {
		if !strings.Contains(uapi, want) {
			t.Errorf("UAPI missing %q\n%s", want, uapi)
		}
	}
}

func TestToUAPIRejectsBadKey(t *testing.T) {
	c, _ := Parse(amneziaConf())
	c.PrivateKey = "not-base64!!!"
	if _, err := ToUAPI(core.AmneziaWG, c); err == nil {
		t.Fatal("expected bad key rejection")
	}
}

func TestFwMarkHexAndDecimal(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want uint32
	}{{"0xca6c", 0xca6c}, {"51820", 51820}} {
		c, err := Parse("[Interface]\nPrivateKey = " + testKey(1) + "\nFwMark = " + tc.in + "\n")
		if err != nil {
			t.Fatalf("parse %q: %v", tc.in, err)
		}
		if c.FwMark != tc.want {
			t.Errorf("fwmark %q = %d, want %d", tc.in, c.FwMark, tc.want)
		}
	}
}
