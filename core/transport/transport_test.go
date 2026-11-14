// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package transport

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestResolveEndpointPrefersIPv4(t *testing.T) {
	got, err := ResolveEndpoint(context.Background(), "vpn.example:51820", func(context.Context, string) ([]string, error) {
		return []string{"2001:db8::1", "203.0.113.10"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "203.0.113.10:51820" {
		t.Fatalf("got %s", got)
	}
}

func TestResolveEndpointKeepsLiteral(t *testing.T) {
	got, err := ResolveEndpoint(context.Background(), "[2001:db8::5]:443", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "[2001:db8::5]:443" {
		t.Fatalf("got %s", got)
	}
}

func TestCandidatesOrderAndSkipConfigured(t *testing.T) {
	cands, err := Candidates("203.0.113.10:443")
	if err != nil {
		t.Fatal(err)
	}
	if cands[0].Reason != "configured" || cands[0].Port != 443 {
		t.Fatalf("first candidate = %+v", cands[0])
	}
	var ports []int
	for _, c := range cands {
		ports = append(ports, c.Port)
	}
	if count(ports, 443) != 1 {
		t.Fatalf("443 count = %d in %v", count(ports, 443), ports)
	}
	if ports[1] != 51820 || ports[2] != 123 || ports[3] != 53 || ports[4] != 1194 {
		t.Fatalf("fallback order = %v", ports)
	}
}

func TestSeekUsesFallbackPort(t *testing.T) {
	link := &fakeLink{openPort: 443}
	res := Seek(context.Background(), link, "vpn.example:39743", "203.0.113.10:39743", 30*time.Millisecond)
	if !res.Handshake || res.Port != 443 {
		t.Fatalf("result = %+v", res)
	}
	if res.Label() != "udp/443" {
		t.Fatalf("label = %s", res.Label())
	}
	if res.FallbackNote() == "" {
		t.Fatal("expected a fallback note")
	}
	if res.FailureNote() != "" {
		t.Fatal("successful seek should not have a failure note")
	}
}

func TestSeekConfiguredPortWins(t *testing.T) {
	link := &fakeLink{openPort: 51820}
	res := Seek(context.Background(), link, "203.0.113.10:51820", "203.0.113.10:51820", 30*time.Millisecond)
	if !res.Handshake || res.Port != 51820 {
		t.Fatalf("result = %+v", res)
	}
	if res.FallbackNote() != "" {
		t.Fatalf("same port should not be described as a fallback: %s", res.FallbackNote())
	}
	if len(res.Tried) != 1 {
		t.Fatalf("should stop after the configured port, tried %v", res.Tried)
	}
}

func TestSeekNoneAnswer(t *testing.T) {
	link := &fakeLink{openPort: 0}
	res := Seek(context.Background(), link, "203.0.113.10:39743", "203.0.113.10:39743", 15*time.Millisecond)
	if res.Handshake {
		t.Fatal("expected no handshake")
	}
	note := res.FailureNote()
	for _, want := range []string{"http2", "quic", "39743", "443"} {
		if !strings.Contains(note, want) {
			t.Errorf("failure note missing %q: %s", want, note)
		}
	}
}

func TestClassifyCorporate(t *testing.T) {
	if Classify(DialConnected, true) != ClassCorporateUDPBlocked {
		t.Fatal("connected TCP + silent UDP should classify as corporate filtering")
	}
	if Classify(DialRefused, true) != ClassCorporateUDPBlocked {
		t.Fatal("TCP refusal still proves the SYN was delivered")
	}
	if Classify(DialTimeout, true) != ClassUnknown {
		t.Fatal("timeout is not evidence the host is up")
	}
	if Classify(DialConnected, false) != ClassUnknown {
		t.Fatal("a live UDP path is not a corporate block")
	}
}

func TestProbeTCPLocal(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err == nil {
			_ = c.Close()
		}
	}()
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	if got := ProbeTCP(context.Background(), "127.0.0.1", port, time.Second); got != DialConnected {
		t.Fatalf("open port: %s", got)
	}
	if got := ProbeTCP(context.Background(), "127.0.0.1", 1, 300*time.Millisecond); got != DialRefused && got != DialTimeout {
		t.Fatalf("closed port: %s", got)
	}
}

type fakeLink struct {
	openPort int
	endpoint string
	hs       time.Time
}

func (f *fakeLink) SetEndpoint(endpoint string) error {
	f.endpoint = endpoint
	_, portStr, err := net.SplitHostPort(endpoint)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}
	if f.openPort != 0 && port == f.openPort {
		f.hs = time.Unix(1_700_000_000, 0)
	}
	return nil
}

func (f *fakeLink) LastHandshake() time.Time { return f.hs }

func count(ports []int, want int) int {
	n := 0
	for _, p := range ports {
		if p == want {
			n++
		}
	}
	return n
}
