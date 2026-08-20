// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package control

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"
)

// Scope is a permission a Grant conveys (SDD L4-0 §4). Instead of a plain
// "reach", a grant carries specific scopes.
type Scope string

const (
	// ScopeConnect: establish a protected channel.
	ScopeConnect Scope = "connect"
	// ScopeControl: manage (start/stop) the target participant.
	ScopeControl Scope = "control"
	// ScopeRoutePrefix is prepended to a provider id, e.g. "route:openai".
	ScopeRoutePrefix = "route:"
)

// RouteScope builds a routing scope for a provider.
func RouteScope(provider string) Scope { return Scope(ScopeRoutePrefix + provider) }

// Grant authorizes From to reach To for specific scopes, optionally expiring,
// signed by the issuer. It replaces the plain Route with auditable, scoped,
// time-bounded, signed authorization.
type Grant struct {
	From      string  `json:"from"`
	To        string  `json:"to"`
	Scopes    []Scope `json:"scopes"`
	ExpiresAt int64   `json:"expires_at"` // unix seconds; 0 = never
	IssuedBy  string  `json:"issued_by"`
	IssuedAt  int64   `json:"issued_at"`
	Signature []byte  `json:"signature,omitempty"`
}

// signingBytes is the canonical byte form signed by the issuer (excludes the
// signature itself).
func (g Grant) signingBytes() []byte {
	c := struct {
		From      string  `json:"from"`
		To        string  `json:"to"`
		Scopes    []Scope `json:"scopes"`
		ExpiresAt int64   `json:"expires_at"`
		IssuedBy  string  `json:"issued_by"`
		IssuedAt  int64   `json:"issued_at"`
	}{g.From, g.To, sortedScopes(g.Scopes), g.ExpiresAt, g.IssuedBy, g.IssuedAt}
	b, _ := json.Marshal(c)
	return b
}

func sortedScopes(s []Scope) []Scope {
	out := append([]Scope(nil), s...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Expired reports whether the grant has expired at time now.
func (g Grant) Expired(now time.Time) bool {
	return g.ExpiresAt != 0 && now.Unix() >= g.ExpiresAt
}

// HasScope reports whether the grant conveys a scope.
func (g Grant) HasScope(s Scope) bool {
	for _, x := range g.Scopes {
		if x == s {
			return true
		}
	}
	return false
}

// GrantBook stores signed grants and answers scoped authorization queries. It
// is deny-by-default: no valid, unexpired grant → no access.
type GrantBook struct {
	mu     sync.RWMutex
	grants map[string][]Grant // key "from->to"
}

// NewGrantBook creates an empty grant book.
func NewGrantBook() *GrantBook {
	return &GrantBook{grants: map[string][]Grant{}}
}

func key(from, to string) string { return from + "->" + to }

// Issue signs and stores a grant. The issuer identity must have a private key;
// IssuedBy is set to the issuer ID.
func (b *GrantBook) Issue(issuer *Identity, g Grant) (Grant, error) {
	if issuer == nil {
		return Grant{}, errNoIssuer
	}
	g.IssuedBy = issuer.ID
	if g.IssuedAt == 0 {
		g.IssuedAt = time.Now().Unix()
	}
	sig, err := issuer.Sign(g.signingBytes())
	if err != nil {
		return Grant{}, err
	}
	g.Signature = sig
	b.mu.Lock()
	defer b.mu.Unlock()
	k := key(g.From, g.To)
	b.grants[k] = append(b.grants[k], g)
	return g, nil
}

// Verify checks a grant's signature against the issuer's public key.
func (b *GrantBook) Verify(g Grant, issuerKey verifier) bool {
	return issuerKey.Verify(g.signingBytes(), g.Signature)
}

// verifier is anything that can verify a signature (Identity or TrustStore key).
type verifier interface {
	Verify(msg, sig []byte) bool
}

// Allows reports whether from is authorized to reach to with scope, given the
// current time. Deny-by-default: needs at least one non-expired grant with the
// scope. Signature validity is the caller's responsibility at issue time; this
// answers the runtime scope/expiry question.
func (b *GrantBook) Allows(from, to string, scope Scope, now time.Time) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, g := range b.grants[key(from, to)] {
		if !g.Expired(now) && g.HasScope(scope) {
			return true
		}
	}
	return false
}

// scopesOn returns the distinct, non-expired scopes granted on the from->to
// edge at time now (used to enumerate a channel's live scopes).
func (b *GrantBook) scopesOn(from, to string, now time.Time) []Scope {
	b.mu.RLock()
	defer b.mu.RUnlock()
	seen := map[Scope]bool{}
	var out []Scope
	for _, g := range b.grants[key(from, to)] {
		if g.Expired(now) {
			continue
		}
		for _, s := range g.Scopes {
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	return out
}

// Revoke removes all grants from->to.
func (b *GrantBook) Revoke(from, to string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.grants, key(from, to))
}

// RevokeAllTo cascade-revokes every grant that targets a participant (used on
// unpair). Returns the number of edges revoked.
func (b *GrantBook) RevokeAllTo(to string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := 0
	for k := range b.grants {
		if strings.HasSuffix(k, "->"+to) {
			delete(b.grants, k)
			n++
		}
	}
	return n
}

// Prune removes expired grants and returns how many were dropped.
func (b *GrantBook) Prune(now time.Time) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	dropped := 0
	for k, list := range b.grants {
		kept := list[:0]
		for _, g := range list {
			if g.Expired(now) {
				dropped++
				continue
			}
			kept = append(kept, g)
		}
		if len(kept) == 0 {
			delete(b.grants, k)
		} else {
			b.grants[k] = kept
		}
	}
	return dropped
}

var (
	errNoIssuer = ident("control: grant needs an issuer identity")
)
