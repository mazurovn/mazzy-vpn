// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package control

import "testing"

func TestPairRequiresMatchingKey(t *testing.T) {
	ts := NewTrustStore()
	a, _ := NewIdentity()
	b, _ := NewIdentity()
	// Pairing a's ID with b's key must fail (anti-impersonation).
	if err := ts.Pair(a.ID, b.PublicKey); err == nil {
		t.Fatal("pairing with a mismatched key must fail")
	}
	// Correct pairing works.
	if err := ts.Pair(a.ID, a.PublicKey); err != nil {
		t.Fatal(err)
	}
	if ts.Level(a.ID) != Paired {
		t.Errorf("level = %s, want paired", ts.Level(a.ID))
	}
}

func TestTrustNonTransitive(t *testing.T) {
	// A's store trusting B says nothing about C. Two independent stores.
	aStore := NewTrustStore()
	bStore := NewTrustStore()
	b, _ := NewIdentity()
	c, _ := NewIdentity()
	aStore.Pair(b.ID, b.PublicKey) // A trusts B
	bStore.Pair(c.ID, c.PublicKey) // B trusts C
	// A does not trust C.
	if aStore.IsTrusted(c.ID) {
		t.Error("trust must not be transitive")
	}
}

func TestUnpairRemovesTrust(t *testing.T) {
	ts := NewTrustStore()
	a, _ := NewIdentity()
	ts.Pair(a.ID, a.PublicKey)
	ts.Unpair(a.ID)
	if ts.IsTrusted(a.ID) || ts.Level(a.ID) != Untrusted {
		t.Error("unpair must remove trust")
	}
}

func TestOwnedNotDowngradedByPair(t *testing.T) {
	ts := NewTrustStore()
	a, _ := NewIdentity()
	ts.SetOwned(a.ID, a.PublicKey)
	ts.Pair(a.ID, a.PublicKey) // re-pair should not downgrade
	if ts.Level(a.ID) != Owned {
		t.Errorf("owned must not be downgraded to paired, got %s", ts.Level(a.ID))
	}
}

func TestVerifyFromTrusted(t *testing.T) {
	ts := NewTrustStore()
	a, _ := NewIdentity()
	ts.Pair(a.ID, a.PublicKey)
	sig, _ := a.Sign([]byte("challenge"))
	if !ts.VerifyFrom(a.ID, []byte("challenge"), sig) {
		t.Error("valid signature from a trusted participant must verify")
	}
	if ts.VerifyFrom(a.ID, []byte("other"), sig) {
		t.Error("signature must not verify a different message")
	}
	// Unknown participant.
	b, _ := NewIdentity()
	if ts.VerifyFrom(b.ID, []byte("x"), sig) {
		t.Error("unknown participant must not verify")
	}
}

func TestLevelOrdering(t *testing.T) {
	if !(Untrusted < Paired && Paired < Owned) {
		t.Error("trust levels must be ordered untrusted<paired<owned")
	}
}
