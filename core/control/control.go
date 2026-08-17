// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package control implements the AI-native Plane 1: the control plane that
// connects agents, harnesses, users, apps and peer devices, and routes
// "who may reach whom". Where core/provider (Plane 2) decides whether an agent
// can reach an LLM through the tunnel, this plane decides which external
// participants may reach a given agent/harness and over which route.
//
// It is transport-agnostic policy: identity + authorization + route selection.
// The actual secure channel is provided by the data plane (engine/routes).
package control

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

// Kind classifies a control-plane participant.
type Kind string

const (
	Agent   Kind = "agent"
	Harness Kind = "harness"
	User    Kind = "user"
	App     Kind = "app"
	Peer    Kind = "peer" // another computer/device
)

// ValidKinds is the set of accepted participant kinds.
var ValidKinds = map[Kind]bool{Agent: true, Harness: true, User: true, App: true, Peer: true}

// Participant is an entity in the control plane.
type Participant struct {
	ID   string `json:"id"`
	Kind Kind   `json:"kind"`
	// Endpoint is where this participant is reachable in the data plane
	// (e.g. a tunnel address); opaque to the policy layer.
	Endpoint string `json:"endpoint,omitempty"`
	// Labels are free-form tags used by routing rules (e.g. "team=research").
	Labels map[string]string `json:"labels,omitempty"`
}

// Route authorizes traffic from a source participant to a target participant.
type Route struct {
	From string `json:"from"` // participant ID
	To   string `json:"to"`   // participant ID
}

var (
	// ErrExists is returned when registering a duplicate participant ID.
	ErrExists = errors.New("control: participant already registered")
	// ErrUnknown is returned when a referenced participant does not exist.
	ErrUnknown = errors.New("control: unknown participant")
	// ErrInvalid is returned for malformed input.
	ErrInvalid = errors.New("control: invalid participant")
)

// Plane is the in-memory control plane. It is safe for concurrent use.
type Plane struct {
	mu           sync.RWMutex
	participants map[string]Participant
	// routes[from][to] = allowed
	routes map[string]map[string]bool
}

// NewPlane creates an empty control plane.
func NewPlane() *Plane {
	return &Plane{
		participants: map[string]Participant{},
		routes:       map[string]map[string]bool{},
	}
}

// Register adds a participant. IDs must be unique and non-empty; kind must be
// valid.
func (p *Plane) Register(part Participant) error {
	id := strings.TrimSpace(part.ID)
	if id == "" || !ValidKinds[part.Kind] {
		return ErrInvalid
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.participants[id]; ok {
		return ErrExists
	}
	part.ID = id
	p.participants[id] = part
	return nil
}

// Deregister removes a participant and any routes referencing it.
func (p *Plane) Deregister(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.participants, id)
	delete(p.routes, id)
	for from := range p.routes {
		delete(p.routes[from], id)
	}
}

// Get returns a participant and whether it exists.
func (p *Plane) Get(id string) (Participant, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	part, ok := p.participants[id]
	return part, ok
}

// Allow authorizes traffic from -> to. Both participants must exist.
func (p *Plane) Allow(from, to string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.participants[from]; !ok {
		return ErrUnknown
	}
	if _, ok := p.participants[to]; !ok {
		return ErrUnknown
	}
	if p.routes[from] == nil {
		p.routes[from] = map[string]bool{}
	}
	p.routes[from][to] = true
	return nil
}

// Revoke removes an authorization.
func (p *Plane) Revoke(from, to string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if m := p.routes[from]; m != nil {
		delete(m, to)
	}
}

// CanReach reports whether from is authorized to reach to. It is deny-by-default:
// only explicit Allow grants access.
func (p *Plane) CanReach(from, to string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	m := p.routes[from]
	return m != nil && m[to]
}

// Reachable returns the sorted IDs a participant may reach (deny-by-default).
func (p *Plane) Reachable(from string) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	m := p.routes[from]
	out := make([]string, 0, len(m))
	for to, ok := range m {
		if ok {
			out = append(out, to)
		}
	}
	sort.Strings(out)
	return out
}

// Participants returns all participant IDs, sorted.
func (p *Plane) Participants() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]string, 0, len(p.participants))
	for id := range p.participants {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
