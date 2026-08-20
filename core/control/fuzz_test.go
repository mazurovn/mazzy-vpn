// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package control

import (
	"crypto/ed25519"
	"testing"
)

// FuzzVerifyID ensures identity verification never panics on arbitrary IDs and
// key bytes, and never authenticates a wrong-sized key.
func FuzzVerifyID(f *testing.F) {
	f.Add("LZRZ7UD7N3IHDRHG", []byte("not-a-key"))
	f.Add("", []byte{})
	f.Fuzz(func(t *testing.T, id string, key []byte) {
		got := VerifyID(id, ed25519.PublicKey(key))
		// A key that isn't exactly the ed25519 size can never authenticate.
		if got && len(key) != ed25519.PublicKeySize {
			t.Fatalf("undersized/oversized key authenticated ID %q", id)
		}
	})
}

// FuzzPair ensures pairing never panics and never trusts a mismatched key.
func FuzzPair(f *testing.F) {
	f.Add("ABCDEFGHIJKLMNOP", []byte("x"))
	f.Fuzz(func(t *testing.T, id string, key []byte) {
		ts := NewTrustStore()
		_ = ts.Pair(id, ed25519.PublicKey(key))
		// After a bad pair attempt the store must not trust an unverifiable id.
		if ts.IsTrusted(id) && !VerifyID(id, ed25519.PublicKey(key)) {
			t.Fatalf("trusted an id whose key does not derive it: %q", id)
		}
	})
}
