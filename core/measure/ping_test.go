// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package measure

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPingParsesAverageRTT(t *testing.T) {
	p := &Pinger{Count: 1, Timeout: time.Second, runner: func(_ context.Context, _ ...string) (string, error) {
		return "PING 1.1.1.1 ...\n64 bytes from 1.1.1.1: time=12.3 ms\nrtt min/avg/max/mdev = 12.300/12.345/12.400/0.050 ms\n", nil
	}}
	ms, ok := p.Ping(context.Background(), "1.1.1.1")
	if !ok || ms < 12.3 || ms > 12.4 {
		t.Fatalf("expected ~12.34ms, got %.3f ok=%v", ms, ok)
	}
}

func TestPingFallsBackToTimeField(t *testing.T) {
	p := &Pinger{runner: func(_ context.Context, _ ...string) (string, error) {
		return "64 bytes from host: time=45.6 ms\n", nil
	}}
	ms, ok := p.Ping(context.Background(), "host")
	if !ok || ms < 45 || ms > 46 {
		t.Fatalf("expected ~45.6ms from time= field, got %.3f ok=%v", ms, ok)
	}
}

func TestPingUnreachable(t *testing.T) {
	p := &Pinger{runner: func(_ context.Context, _ ...string) (string, error) {
		return "100% packet loss\n", errors.New("exit 1")
	}}
	if _, ok := p.Ping(context.Background(), "10.255.255.1"); ok {
		t.Fatal("100% loss must be unreachable")
	}
}

func TestPingEmptyHost(t *testing.T) {
	p := NewPinger()
	if _, ok := p.Ping(context.Background(), "  "); ok {
		t.Fatal("empty host must not be reachable")
	}
}

func TestPingPartialParseOnError(t *testing.T) {
	// ping can exit non-zero but still report a reply for some packets.
	p := &Pinger{runner: func(_ context.Context, _ ...string) (string, error) {
		return "rtt min/avg/max/mdev = 88.0/90.0/92.0/1.0 ms\n", errors.New("exit 1")
	}}
	ms, ok := p.Ping(context.Background(), "host")
	if !ok || ms < 89 || ms > 91 {
		t.Fatalf("expected partial parse ~90ms, got %.3f ok=%v", ms, ok)
	}
}

func TestPingInterfaceFlag(t *testing.T) {
	var gotArgs []string
	p := &Pinger{Interface: "wlp3s0", runner: func(_ context.Context, args ...string) (string, error) {
		gotArgs = args
		return "rtt min/avg/max/mdev = 10/10/10/0 ms\n", nil
	}}
	p.Ping(context.Background(), "1.2.3.4")
	found := false
	for i, a := range gotArgs {
		if a == "-I" && i+1 < len(gotArgs) && gotArgs[i+1] == "wlp3s0" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected -I wlp3s0 in args, got %v", gotArgs)
	}
}
