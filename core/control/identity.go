// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package control

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
)

// Identity is a self-authenticating participant identity: the ID is derived
// from the Ed25519 public key, so no one can claim another's ID without the
// matching private key (SDD L4-0). Registration is proven by signing a
// challenge with the private key.
type Identity struct {
	ID        string             // derived from PublicKey
	PublicKey ed25519.PublicKey  `json:"-"`
	priv      ed25519.PrivateKey // held only by the owner
}

// b32 is lowercase, unpadded base32 for compact IDs.
var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// DeriveID computes the self-authenticating ID from a public key:
// lowercase base32 of the first 10 bytes of sha256(pubkey) (16 chars).
func DeriveID(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return b32.EncodeToString(sum[:10])
}

// NewIdentity generates a fresh keypair and its self-authenticating ID.
func NewIdentity() (*Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Identity{ID: DeriveID(pub), PublicKey: pub, priv: priv}, nil
}

// IdentityFromKey builds an Identity for verification from a public key only
// (no signing capability).
func IdentityFromKey(pub ed25519.PublicKey) *Identity {
	return &Identity{ID: DeriveID(pub), PublicKey: pub}
}

// Sign signs a message with the identity's private key. Fails if this identity
// has no private key (verification-only).
func (i *Identity) Sign(msg []byte) ([]byte, error) {
	if i.priv == nil {
		return nil, errNoPrivateKey
	}
	return ed25519.Sign(i.priv, msg), nil
}

// Verify checks a signature against the identity's public key.
func (i *Identity) Verify(msg, sig []byte) bool {
	if len(i.PublicKey) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(i.PublicKey, msg, sig)
}

// VerifyID reports whether the public key actually derives to id — the core of
// self-authentication: a claimed ID is only valid if it matches its key.
func VerifyID(id string, pub ed25519.PublicKey) bool {
	return len(pub) == ed25519.PublicKeySize && DeriveID(pub) == id
}

var errNoPrivateKey = errors.New("control: identity has no private key (verification only)")
