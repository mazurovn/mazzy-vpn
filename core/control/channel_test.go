// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package control

import (
	"testing"
	"time"
)

func newBrokerFixture(t *testing.T) (*Broker, *Identity, string, string) {
	t.Helper()
	plane := NewPlane()
	trust := NewTrustStore()
	grants := NewGrantBook()
	issuer, _ := NewIdentity()
	agent, _ := NewIdentity()
	gw, _ := NewIdentity()

	if err := plane.Register(Participant{ID: agent.ID, Kind: Agent}); err != nil {
		t.Fatal(err)
	}
	if err := plane.Register(Participant{ID: gw.ID, Kind: Peer, Endpoint: "10.9.0.1:51820"}); err != nil {
		t.Fatal(err)
	}
	_ = trust.Pair(gw.ID, gw.PublicKey)
	_, _ = grants.Issue(issuer, Grant{From: agent.ID, To: gw.ID, Scopes: []Scope{ScopeConnect, RouteScope("openai")}})
	return NewBroker(plane, trust, grants), issuer, agent.ID, gw.ID
}

func TestBrokerOpensAuthorizedChannel(t *testing.T) {
	b, _, agent, gw := newBrokerFixture(t)
	ch, err := b.Open(agent, gw, ScopeConnect, time.Now())
	if err != nil {
		t.Fatalf("authorized channel must open: %v", err)
	}
	if ch.Endpoint != "10.9.0.1:51820" {
		t.Errorf("endpoint = %q", ch.Endpoint)
	}
	if len(ch.Scopes) == 0 {
		t.Error("channel should carry its granted scopes")
	}
}

func TestBrokerDenyUnregistered(t *testing.T) {
	b, _, _, gw := newBrokerFixture(t)
	if _, err := b.Open("ghost", gw, ScopeConnect, time.Now()); err != ErrNotRegistered {
		t.Errorf("unregistered initiator must be denied, got %v", err)
	}
}

func TestBrokerDenyUntrusted(t *testing.T) {
	plane := NewPlane()
	trust := NewTrustStore()
	grants := NewGrantBook()
	issuer, _ := NewIdentity()
	agent, _ := NewIdentity()
	gw, _ := NewIdentity()
	plane.Register(Participant{ID: agent.ID, Kind: Agent})
	plane.Register(Participant{ID: gw.ID, Kind: Peer, Endpoint: "e:1"})
	// No pairing → untrusted, even though a grant exists.
	grants.Issue(issuer, Grant{From: agent.ID, To: gw.ID, Scopes: []Scope{ScopeConnect}})
	b := NewBroker(plane, trust, grants)
	if _, err := b.Open(agent.ID, gw.ID, ScopeConnect, time.Now()); err != ErrNotTrusted {
		t.Errorf("untrusted target must be denied, got %v", err)
	}
}

func TestBrokerDenyUngrantedScope(t *testing.T) {
	b, _, agent, gw := newBrokerFixture(t)
	if _, err := b.Open(agent, gw, ScopeControl, time.Now()); err != ErrNotGranted {
		t.Errorf("ungranted scope must be denied, got %v", err)
	}
}

func TestBrokerDenyNoEndpoint(t *testing.T) {
	plane := NewPlane()
	trust := NewTrustStore()
	grants := NewGrantBook()
	issuer, _ := NewIdentity()
	agent, _ := NewIdentity()
	gw, _ := NewIdentity()
	plane.Register(Participant{ID: agent.ID, Kind: Agent})
	plane.Register(Participant{ID: gw.ID, Kind: Peer}) // no endpoint
	trust.Pair(gw.ID, gw.PublicKey)
	grants.Issue(issuer, Grant{From: agent.ID, To: gw.ID, Scopes: []Scope{ScopeConnect}})
	b := NewBroker(plane, trust, grants)
	if _, err := b.Open(agent.ID, gw.ID, ScopeConnect, time.Now()); err != ErrNoEndpoint {
		t.Errorf("missing endpoint must be denied, got %v", err)
	}
}

func TestBrokerReachableHonorsGates(t *testing.T) {
	b, _, agent, gw := newBrokerFixture(t)
	chans := b.Reachable(agent, ScopeConnect, time.Now())
	if len(chans) != 1 || chans[0].To != gw {
		t.Fatalf("Reachable should list exactly the one authorized gateway, got %+v", chans)
	}
	// A scope with no grant yields nothing.
	if got := b.Reachable(agent, ScopeControl, time.Now()); len(got) != 0 {
		t.Errorf("no channels should be reachable for an ungranted scope, got %d", len(got))
	}
}

func TestBrokerCloseIdempotent(t *testing.T) {
	b, _, agent, gw := newBrokerFixture(t)
	b.Open(agent, gw, ScopeConnect, time.Now())
	b.Close(agent, gw)
	b.Close(agent, gw) // second close must not panic
}
