// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package control

import (
	"sort"
	"sync"
	"time"
)

// Channel is a resolved, authorized peer-to-peer link between two participants
// (L4-3). It is the bridge from the control plane (policy: who may talk to whom,
// for what) to the data plane (where the peer actually is). A Channel only
// exists when ALL THREE gates pass:
//
//  1. both participants are registered in the Plane,
//  2. the initiator trusts the target (TrustStore, at least Paired),
//  3. a non-expired Grant conveys the requested scope (GrantBook).
//
// This keeps the AI-native promise: an agent gets a usable egress channel only
// through explicit, auditable, revocable authorization — never implicitly.
type Channel struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Endpoint string  `json:"endpoint"` // data-plane address of the target
	Scopes   []Scope `json:"scopes"`   // scopes actually granted on this link
	OpenedAt int64   `json:"opened_at"`
}

// ChannelError explains why a channel could not be opened.
type ChannelError string

func (e ChannelError) Error() string { return string(e) }

const (
	// ErrNotRegistered: a participant is not in the Plane.
	ErrNotRegistered = ChannelError("control: participant not registered")
	// ErrNotTrusted: the initiator does not trust the target.
	ErrNotTrusted = ChannelError("control: target not trusted (pair first)")
	// ErrNotGranted: no valid grant conveys the requested scope.
	ErrNotGranted = ChannelError("control: no grant for the requested scope")
	// ErrNoEndpoint: the target has no data-plane endpoint to reach.
	ErrNoEndpoint = ChannelError("control: target has no endpoint")
)

// Broker opens Channels by composing a Plane (registry), a TrustStore (trust)
// and a GrantBook (authorization). It is deny-by-default and safe for
// concurrent use.
type Broker struct {
	plane  *Plane
	trust  *TrustStore
	grants *GrantBook

	mu   sync.RWMutex
	open map[string]Channel // key "from->to"
}

// NewBroker wires the three control-plane stores into a channel broker.
func NewBroker(plane *Plane, trust *TrustStore, grants *GrantBook) *Broker {
	return &Broker{plane: plane, trust: trust, grants: grants, open: map[string]Channel{}}
}

// Open resolves and authorizes a channel from->to for the given scope at time
// now. It enforces all three gates and returns a Channel with the target's
// data-plane endpoint, or a ChannelError explaining the first failed gate.
func (b *Broker) Open(from, to string, scope Scope, now time.Time) (Channel, error) {
	// Gate 1: both participants registered.
	if _, ok := b.plane.Get(from); !ok {
		return Channel{}, ErrNotRegistered
	}
	target, ok := b.plane.Get(to)
	if !ok {
		return Channel{}, ErrNotRegistered
	}
	// Gate 2: initiator trusts the target.
	if !b.trust.IsTrusted(to) {
		return Channel{}, ErrNotTrusted
	}
	// Gate 3: a non-expired grant conveys the scope.
	if !b.grants.Allows(from, to, scope, now) {
		return Channel{}, ErrNotGranted
	}
	if target.Endpoint == "" {
		return Channel{}, ErrNoEndpoint
	}
	scopes := b.grants.scopesOn(from, to, now)
	sort.Slice(scopes, func(i, j int) bool { return scopes[i] < scopes[j] })
	ch := Channel{
		From:     from,
		To:       to,
		Endpoint: target.Endpoint,
		Scopes:   scopes,
		OpenedAt: now.Unix(),
	}
	b.mu.Lock()
	b.open[from+"->"+to] = ch
	b.mu.Unlock()
	return ch, nil
}

// Close tears down an open channel (idempotent).
func (b *Broker) Close(from, to string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.open, from+"->"+to)
}

// Reachable lists the endpoints an initiator can currently open for a scope,
// honoring all three gates. Used by an agent to discover its authorized egress.
func (b *Broker) Reachable(from string, scope Scope, now time.Time) []Channel {
	var out []Channel
	for _, to := range b.plane.Participants() {
		if to == from {
			continue
		}
		if ch, err := b.Open(from, to, scope, now); err == nil {
			out = append(out, ch)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].To < out[j].To })
	return out
}
