// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package panel is the desktop top-bar state: connect, disconnect, status,
// server and protocol. It is pure Go so it can be tested without a display.
package panel

import (
	"context"
	"sync"
)

// Phase is the connection status shown in the top panel.
type Phase string

const (
	PhaseDisconnected Phase = "disconnected"
	PhaseConnecting   Phase = "connecting"
	PhaseConnected    Phase = "connected"
	PhaseError        Phase = "error"
)

// Profile is one catalog entry the panel can select.
type Profile struct {
	Name     string
	Protocol string
	Endpoint string
}

// Snapshot is the top-panel view plus the profile list under it.
type Snapshot struct {
	Phase     Phase
	Server    string
	Protocol  string
	Transport string
	Detail    string
	Err       string
	Profiles  []Profile
	Selected  int
}

// SelectedProfile returns the highlighted profile.
func (s Snapshot) SelectedProfile() (Profile, bool) {
	if s.Selected < 0 || s.Selected >= len(s.Profiles) {
		return Profile{}, false
	}
	return s.Profiles[s.Selected], true
}

// Control talks to the VPN engine. Implementations must be safe to call from
// one goroutine at a time; the Model already serializes Connect and Disconnect.
type Control interface {
	Load(ctx context.Context) (Snapshot, error)
	Connect(ctx context.Context, name string) (Snapshot, error)
	Disconnect(ctx context.Context) (Snapshot, error)
}

// Model owns panel state for the window.
type Model struct {
	mu       sync.Mutex
	ctrl     Control
	snap     Snapshot
	busy     bool
	OnChange func()
}

// New returns a disconnected panel bound to ctrl.
func New(ctrl Control) *Model {
	return &Model{ctrl: ctrl}
}

// View returns a copy of the current snapshot.
func (m *Model) View() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return clone(m.snap)
}

// CanConnect reports whether the Connect control should act.
func (m *Model) CanConnect() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.snap.SelectedProfile()
	return ok && !m.busy && m.snap.Phase != PhaseConnecting
}

// CanDisconnect reports whether the Disconnect control should act.
func (m *Model) CanDisconnect() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.busy {
		return false
	}
	return m.snap.Phase == PhaseConnected || m.snap.Phase == PhaseError
}

// Select highlights a profile. Out-of-range values are ignored.
func (m *Model) Select(i int) {
	m.mu.Lock()
	if i < 0 || i >= len(m.snap.Profiles) || m.snap.Selected == i {
		m.mu.Unlock()
		return
	}
	m.snap.Selected = i
	m.mu.Unlock()
	m.notify()
}

// Reload refreshes profiles and status without changing the VPN.
func (m *Model) Reload(ctx context.Context) {
	m.async(ctx, func(ctx context.Context) (Snapshot, error) {
		return m.ctrl.Load(ctx)
	}, false)
}

// Connect brings up the selected profile.
func (m *Model) Connect(ctx context.Context) {
	m.mu.Lock()
	p, ok := m.snap.SelectedProfile()
	if !ok || m.busy || m.snap.Phase == PhaseConnecting {
		m.mu.Unlock()
		return
	}
	name := p.Name
	m.busy = true
	m.snap.Phase = PhaseConnecting
	m.snap.Err = ""
	m.snap.Server = name
	m.snap.Protocol = p.Protocol
	m.mu.Unlock()
	m.notify()
	m.async(ctx, func(ctx context.Context) (Snapshot, error) {
		return m.ctrl.Connect(ctx, name)
	}, true)
}

// Disconnect tears the tunnel down.
func (m *Model) Disconnect(ctx context.Context) {
	m.mu.Lock()
	if m.busy || (m.snap.Phase != PhaseConnected && m.snap.Phase != PhaseError) {
		m.mu.Unlock()
		return
	}
	m.busy = true
	m.mu.Unlock()
	m.notify()
	m.async(ctx, func(ctx context.Context) (Snapshot, error) {
		return m.ctrl.Disconnect(ctx)
	}, true)
}

// async runs op. alreadyBusy is set when the caller has already marked the
// model busy (Connect and Disconnect).
func (m *Model) async(ctx context.Context, op func(context.Context) (Snapshot, error), alreadyBusy bool) {
	if !alreadyBusy {
		m.mu.Lock()
		if m.busy {
			m.mu.Unlock()
			return
		}
		m.busy = true
		m.mu.Unlock()
		m.notify()
	}
	go func() {
		snap, err := op(ctx)
		m.mu.Lock()
		prev, _ := m.snap.SelectedProfile()
		m.busy = false
		if err != nil {
			m.snap.Phase = PhaseError
			m.snap.Err = err.Error()
			m.snap.Detail = err.Error()
		} else {
			m.apply(snap, prev.Name)
		}
		m.mu.Unlock()
		m.notify()
	}()
}

func (m *Model) apply(snap Snapshot, keepName string) {
	snap.Selected = 0
	if keepName != "" {
		for i, p := range snap.Profiles {
			if p.Name == keepName {
				snap.Selected = i
				break
			}
		}
	}
	m.snap = snap
}

func (m *Model) notify() {
	if m.OnChange != nil {
		m.OnChange()
	}
}

func clone(s Snapshot) Snapshot {
	s.Profiles = append([]Profile(nil), s.Profiles...)
	return s
}
