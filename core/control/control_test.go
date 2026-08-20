// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package control

import (
	"sync"
	"testing"
)

func TestRegisterAndGet(t *testing.T) {
	p := NewPlane()
	if err := p.Register(Participant{ID: "agent-1", Kind: Agent}); err != nil {
		t.Fatal(err)
	}
	got, ok := p.Get("agent-1")
	if !ok || got.Kind != Agent {
		t.Fatalf("get failed: %+v %v", got, ok)
	}
}

func TestRegisterRejectsInvalid(t *testing.T) {
	p := NewPlane()
	if err := p.Register(Participant{ID: "", Kind: Agent}); err != ErrInvalid {
		t.Error("empty id must be invalid")
	}
	if err := p.Register(Participant{ID: "x", Kind: Kind("bogus")}); err != ErrInvalid {
		t.Error("bad kind must be invalid")
	}
}

func TestRegisterRejectsDuplicate(t *testing.T) {
	p := NewPlane()
	p.Register(Participant{ID: "a", Kind: Agent})
	if err := p.Register(Participant{ID: "a", Kind: Harness}); err != ErrExists {
		t.Error("duplicate id must be rejected")
	}
}

// TestDenyByDefault is the security core of Plane 1: without an explicit Allow,
// no participant may reach another.
func TestDenyByDefault(t *testing.T) {
	p := NewPlane()
	p.Register(Participant{ID: "harness", Kind: Harness})
	p.Register(Participant{ID: "agent", Kind: Agent})
	if p.CanReach("harness", "agent") {
		t.Fatal("must be deny-by-default before Allow")
	}
	if err := p.Allow("harness", "agent"); err != nil {
		t.Fatal(err)
	}
	if !p.CanReach("harness", "agent") {
		t.Fatal("Allow should grant reach")
	}
	// Reverse direction is still denied (routes are directional).
	if p.CanReach("agent", "harness") {
		t.Fatal("routes must be directional")
	}
}

func TestAllowRequiresKnownParticipants(t *testing.T) {
	p := NewPlane()
	p.Register(Participant{ID: "a", Kind: Agent})
	if err := p.Allow("a", "ghost"); err != ErrUnknown {
		t.Error("allow to unknown target must fail")
	}
	if err := p.Allow("ghost", "a"); err != ErrUnknown {
		t.Error("allow from unknown source must fail")
	}
}

func TestRevokeAndReachable(t *testing.T) {
	p := NewPlane()
	for _, id := range []string{"h", "a1", "a2"} {
		p.Register(Participant{ID: id, Kind: Agent})
	}
	p.Allow("h", "a1")
	p.Allow("h", "a2")
	if got := p.Reachable("h"); len(got) != 2 || got[0] != "a1" || got[1] != "a2" {
		t.Fatalf("reachable = %v, want sorted [a1 a2]", got)
	}
	p.Revoke("h", "a1")
	if p.CanReach("h", "a1") {
		t.Error("revoke should remove reach")
	}
	if got := p.Reachable("h"); len(got) != 1 || got[0] != "a2" {
		t.Errorf("reachable after revoke = %v", got)
	}
}

func TestDeregisterRemovesRoutes(t *testing.T) {
	p := NewPlane()
	p.Register(Participant{ID: "h", Kind: Harness})
	p.Register(Participant{ID: "a", Kind: Agent})
	p.Allow("h", "a")
	p.Deregister("a")
	if _, ok := p.Get("a"); ok {
		t.Error("deregistered participant must be gone")
	}
	if p.CanReach("h", "a") {
		t.Error("routes to a deregistered participant must be removed")
	}
}

func TestParticipantsSorted(t *testing.T) {
	p := NewPlane()
	for _, id := range []string{"c", "a", "b"} {
		p.Register(Participant{ID: id, Kind: Agent})
	}
	got := p.Participants()
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Errorf("participants not sorted: %v", got)
	}
}

// TestConcurrentAccess exercises the RWMutex under the race detector.
func TestConcurrentAccess(t *testing.T) {
	p := NewPlane()
	p.Register(Participant{ID: "h", Kind: Harness})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "a" + string(rune('0'+n%10))
			_ = p.Register(Participant{ID: id, Kind: Agent})
			_ = p.Allow("h", id)
			_ = p.CanReach("h", id)
			_ = p.Reachable("h")
		}(i)
	}
	wg.Wait()
}
