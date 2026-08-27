// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package integration

import (
	"testing"
	"time"

	"github.com/mazurovn/mazzy-vpn/core/control"
)

// TestControlPlaneEndToEnd exercises the full L4-0 contract: two participants
// generate self-authenticating identities, pair (establishing trust), and an
// issuer signs a scoped, time-bounded grant that authorizes one to reach the
// other for a specific provider — while everything else stays deny-by-default.
func TestControlPlaneEndToEnd(t *testing.T) {
	// 1. Identities (self-authenticating).
	harness, err := control.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	agent, err := control.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if !control.VerifyID(agent.ID, agent.PublicKey) {
		t.Fatal("agent identity must self-authenticate")
	}

	// 2. Trust: the harness pairs with the agent's key (out-of-band).
	trust := control.NewTrustStore()
	if err := trust.Pair(agent.ID, agent.PublicKey); err != nil {
		t.Fatal(err)
	}
	if !trust.IsTrusted(agent.ID) {
		t.Fatal("agent should be trusted after pairing")
	}
	// Anti-impersonation: a forged pairing (agent ID, harness key) fails.
	if err := trust.Pair(agent.ID, harness.PublicKey); err == nil {
		t.Fatal("forged pairing must be rejected")
	}

	// 3. Authorization: the harness issues a signed grant letting the agent
	//    reach a gateway for the openai provider, expiring in an hour.
	gb := control.NewGrantBook()
	g, err := gb.Issue(harness, control.Grant{
		From:      agent.ID,
		To:        "gateway",
		Scopes:    []control.Scope{control.RouteScope("openai"), control.ScopeConnect},
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !gb.Verify(g, harness) {
		t.Fatal("grant must verify against the issuer")
	}

	now := time.Now()
	// Allowed: exactly what was granted.
	if !gb.Allows(agent.ID, "gateway", control.RouteScope("openai"), now) {
		t.Error("granted openai route should be allowed")
	}
	if !gb.Allows(agent.ID, "gateway", control.ScopeConnect, now) {
		t.Error("granted connect should be allowed")
	}
	// Denied: an ungranted scope and an ungranted provider.
	if gb.Allows(agent.ID, "gateway", control.ScopeControl, now) {
		t.Error("ungranted control scope must be denied")
	}
	if gb.Allows(agent.ID, "gateway", control.RouteScope("anthropic"), now) {
		t.Error("ungranted provider must be denied")
	}

	// 4. Revocation on unpair: cascade-revoke everything to the agent, then the
	//    agent's own grants remain but trust is gone.
	trust.Unpair(agent.ID)
	if trust.IsTrusted(agent.ID) {
		t.Error("unpair must drop trust")
	}
	gb.RevokeAllTo("gateway")
	if gb.Allows(agent.ID, "gateway", control.ScopeConnect, now) {
		t.Error("cascade revoke must drop the grant")
	}
}
