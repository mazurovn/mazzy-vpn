// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package control

import (
	"crypto/ed25519"
	"testing"
)

func TestNewIdentitySelfAuthenticating(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if id.ID == "" || len(id.ID) != 16 {
		t.Errorf("ID should be 16 chars, got %q", id.ID)
	}
	// The ID must derive from the public key.
	if !VerifyID(id.ID, id.PublicKey) {
		t.Error("ID must self-authenticate against its public key")
	}
}

func TestSignVerify(t *testing.T) {
	id, _ := NewIdentity()
	msg := []byte("challenge-123")
	sig, err := id.Sign(msg)
	if err != nil {
		t.Fatal(err)
	}
	if !id.Verify(msg, sig) {
		t.Error("valid signature must verify")
	}
	if id.Verify([]byte("tampered"), sig) {
		t.Error("signature must not verify a different message")
	}
}

func TestVerificationOnlyIdentityCannotSign(t *testing.T) {
	full, _ := NewIdentity()
	verOnly := IdentityFromKey(full.PublicKey)
	if verOnly.ID != full.ID {
		t.Error("verification identity should share the ID")
	}
	if _, err := verOnly.Sign([]byte("x")); err == nil {
		t.Error("verification-only identity must not be able to sign")
	}
	// But it can verify signatures from the full identity.
	sig, _ := full.Sign([]byte("m"))
	if !verOnly.Verify([]byte("m"), sig) {
		t.Error("verification identity should verify the owner's signature")
	}
}

func TestVerifyIDRejectsWrongKey(t *testing.T) {
	a, _ := NewIdentity()
	b, _ := NewIdentity()
	// b's key must not authenticate a's ID (anti-impersonation).
	if VerifyID(a.ID, b.PublicKey) {
		t.Error("a different key must not authenticate another's ID")
	}
}

func TestVerifyIDRejectsBadKeySize(t *testing.T) {
	if VerifyID("whatever", ed25519.PublicKey([]byte{1, 2, 3})) {
		t.Error("undersized key must not authenticate")
	}
}

func TestDeriveIDStable(t *testing.T) {
	id, _ := NewIdentity()
	if DeriveID(id.PublicKey) != id.ID {
		t.Error("DeriveID must be stable for the same key")
	}
}
