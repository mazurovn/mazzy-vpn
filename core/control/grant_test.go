// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package control

import (
	"testing"
	"time"
)

func TestIssueAndVerify(t *testing.T) {
	issuer, _ := NewIdentity()
	gb := NewGrantBook()
	g, err := gb.Issue(issuer, Grant{From: "a", To: "b", Scopes: []Scope{ScopeConnect}})
	if err != nil {
		t.Fatal(err)
	}
	if g.IssuedBy != issuer.ID || len(g.Signature) == 0 {
		t.Errorf("grant not signed properly: %+v", g)
	}
	if !gb.Verify(g, issuer) {
		t.Error("issuer must verify its own grant")
	}
	// Tampering breaks verification.
	g.Scopes = []Scope{ScopeControl}
	if gb.Verify(g, issuer) {
		t.Error("tampered grant must not verify")
	}
}

func TestAllowsDenyByDefault(t *testing.T) {
	gb := NewGrantBook()
	now := time.Now()
	if gb.Allows("a", "b", ScopeConnect, now) {
		t.Fatal("no grant → deny")
	}
	issuer, _ := NewIdentity()
	gb.Issue(issuer, Grant{From: "a", To: "b", Scopes: []Scope{ScopeConnect}})
	if !gb.Allows("a", "b", ScopeConnect, now) {
		t.Error("granted scope should be allowed")
	}
	// Different scope not granted.
	if gb.Allows("a", "b", ScopeControl, now) {
		t.Error("ungranted scope must be denied")
	}
}

func TestExpiry(t *testing.T) {
	gb := NewGrantBook()
	issuer, _ := NewIdentity()
	past := time.Now().Add(-time.Hour).Unix()
	gb.Issue(issuer, Grant{From: "a", To: "b", Scopes: []Scope{ScopeConnect}, ExpiresAt: past})
	if gb.Allows("a", "b", ScopeConnect, time.Now()) {
		t.Error("expired grant must not allow")
	}
}

func TestRouteScope(t *testing.T) {
	gb := NewGrantBook()
	issuer, _ := NewIdentity()
	gb.Issue(issuer, Grant{From: "agent", To: "gw", Scopes: []Scope{RouteScope("openai")}})
	if !gb.Allows("agent", "gw", RouteScope("openai"), time.Now()) {
		t.Error("route:openai scope should be allowed")
	}
	if gb.Allows("agent", "gw", RouteScope("anthropic"), time.Now()) {
		t.Error("different provider route must be denied")
	}
}

func TestRevoke(t *testing.T) {
	gb := NewGrantBook()
	issuer, _ := NewIdentity()
	gb.Issue(issuer, Grant{From: "a", To: "b", Scopes: []Scope{ScopeConnect}})
	gb.Revoke("a", "b")
	if gb.Allows("a", "b", ScopeConnect, time.Now()) {
		t.Error("revoked grant must not allow")
	}
}

func TestRevokeAllToCascade(t *testing.T) {
	gb := NewGrantBook()
	issuer, _ := NewIdentity()
	gb.Issue(issuer, Grant{From: "a", To: "victim", Scopes: []Scope{ScopeConnect}})
	gb.Issue(issuer, Grant{From: "b", To: "victim", Scopes: []Scope{ScopeConnect}})
	gb.Issue(issuer, Grant{From: "a", To: "other", Scopes: []Scope{ScopeConnect}})
	n := gb.RevokeAllTo("victim")
	if n != 2 {
		t.Errorf("expected 2 edges revoked, got %d", n)
	}
	if gb.Allows("a", "victim", ScopeConnect, time.Now()) || gb.Allows("b", "victim", ScopeConnect, time.Now()) {
		t.Error("cascade revoke must drop all grants to the participant")
	}
	if !gb.Allows("a", "other", ScopeConnect, time.Now()) {
		t.Error("unrelated grants must survive cascade revoke")
	}
}

func TestPruneExpired(t *testing.T) {
	gb := NewGrantBook()
	issuer, _ := NewIdentity()
	gb.Issue(issuer, Grant{From: "a", To: "b", Scopes: []Scope{ScopeConnect}, ExpiresAt: time.Now().Add(-time.Hour).Unix()})
	gb.Issue(issuer, Grant{From: "a", To: "c", Scopes: []Scope{ScopeConnect}}) // no expiry
	dropped := gb.Prune(time.Now())
	if dropped != 1 {
		t.Errorf("expected 1 pruned, got %d", dropped)
	}
	if !gb.Allows("a", "c", ScopeConnect, time.Now()) {
		t.Error("non-expiring grant must survive prune")
	}
}
