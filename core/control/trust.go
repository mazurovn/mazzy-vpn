// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package control

import (
	"crypto/ed25519"
	"sync"
)

// TrustLevel is how much a participant is trusted (SDD L4-0 §3). Trust is NOT
// transitive: A trusting B and B trusting C does not make A trust C.
type TrustLevel int

const (
	// Untrusted: merely known; gets nothing (deny-by-default).
	Untrusted TrustLevel = iota
	// Paired: mutual key exchange completed (out-of-band); may be an Allow target.
	Paired
	// Owned: same owner (shared root/device); trusted for control operations.
	Owned
)

func (t TrustLevel) String() string {
	switch t {
	case Paired:
		return "paired"
	case Owned:
		return "owned"
	default:
		return "untrusted"
	}
}

// TrustStore records the public keys and trust levels of known participants. It
// is the local view of "who I trust" — trust is per-node and non-transitive.
type TrustStore struct {
	mu    sync.RWMutex
	trust map[string]trustEntry // participant ID -> entry
}

type trustEntry struct {
	pub   ed25519.PublicKey
	level TrustLevel
}

// NewTrustStore creates an empty trust store.
func NewTrustStore() *TrustStore {
	return &TrustStore{trust: map[string]trustEntry{}}
}

// Pair records a participant's public key at the Paired level, but only if the
// key self-authenticates the claimed ID (anti-impersonation). This is the
// out-of-band pairing step (the key arrives via QR/code/file).
func (s *TrustStore) Pair(id string, pub ed25519.PublicKey) error {
	if !VerifyID(id, pub) {
		return errIDKeyMismatch
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Do not downgrade an owned entry to paired.
	if e, ok := s.trust[id]; ok && e.level == Owned {
		s.trust[id] = trustEntry{pub: pub, level: Owned}
		return nil
	}
	s.trust[id] = trustEntry{pub: pub, level: Paired}
	return nil
}

// SetOwned marks an already-known participant as owned (same owner/device).
func (s *TrustStore) SetOwned(id string, pub ed25519.PublicKey) error {
	if !VerifyID(id, pub) {
		return errIDKeyMismatch
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trust[id] = trustEntry{pub: pub, level: Owned}
	return nil
}

// Unpair removes all trust for a participant (revokes trust; callers should
// cascade-revoke grants).
func (s *TrustStore) Unpair(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.trust, id)
}

// Level returns the trust level for a participant (Untrusted if unknown).
func (s *TrustStore) Level(id string) TrustLevel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if e, ok := s.trust[id]; ok {
		return e.level
	}
	return Untrusted
}

// Key returns the stored public key for a participant.
func (s *TrustStore) Key(id string) (ed25519.PublicKey, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.trust[id]
	return e.pub, ok
}

// IsTrusted reports whether a participant is at least Paired.
func (s *TrustStore) IsTrusted(id string) bool {
	return s.Level(id) >= Paired
}

// VerifyFrom checks a signature claimed to be from a trusted participant: the
// participant must be trusted AND the signature must verify against its stored
// key. This is the challenge-response used to confirm possession during pairing
// and for signed grants.
func (s *TrustStore) VerifyFrom(id string, msg, sig []byte) bool {
	pub, ok := s.Key(id)
	if !ok {
		return false
	}
	return ed25519.Verify(pub, msg, sig)
}

var errIDKeyMismatch = ident("control: public key does not derive the claimed ID")

type ident string

func (e ident) Error() string { return string(e) }
