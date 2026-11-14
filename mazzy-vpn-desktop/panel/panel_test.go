// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package panel

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTopPanelConnectAndDisconnect(t *testing.T) {
	ctrl := &fakeCtrl{snap: Snapshot{
		Phase: PhaseDisconnected,
		Profiles: []Profile{
			{Name: "helsinki", Protocol: "AmneziaWG", Endpoint: "203.0.113.10:51820"},
			{Name: "berlin", Protocol: "WireGuard", Endpoint: "203.0.113.11:443"},
		},
	}}
	m := New(ctrl)
	done := make(chan struct{}, 8)
	m.OnChange = func() { done <- struct{}{} }

	m.Reload(context.Background())
	view := waitFunc(t, m, done, func(s Snapshot) bool {
		return len(s.Profiles) == 2 && s.Phase == PhaseDisconnected
	})
	if view.SelectedProfileName() != "helsinki" {
		t.Fatalf("default server = %q", view.SelectedProfileName())
	}

	m.Select(1)
	wait(t, done)
	if got := m.View().SelectedProfileName(); got != "berlin" {
		t.Fatalf("selected server = %q", got)
	}
	if !m.CanConnect() || m.CanDisconnect() {
		t.Fatalf("disconnected panel connect=%v disconnect=%v", m.CanConnect(), m.CanDisconnect())
	}

	ctrl.connected = Snapshot{
		Phase: PhaseConnected, Server: "berlin", Protocol: "WireGuard", Transport: "udp/443",
		Profiles: ctrl.snap.Profiles,
	}
	m.Connect(context.Background())
	view = waitPhase(t, m, done, PhaseConnected)
	if view.Phase != PhaseConnected || view.Server != "berlin" || view.Transport != "udp/443" {
		t.Fatalf("connected view = %+v", view)
	}
	if view.Protocol != "WireGuard" {
		t.Fatalf("protocol = %s", view.Protocol)
	}
	if !m.CanDisconnect() {
		t.Fatal("connected panel should disconnect")
	}

	m.Disconnect(context.Background())
	view = waitPhase(t, m, done, PhaseDisconnected)
	if view.Phase != PhaseDisconnected {
		t.Fatalf("after disconnect: %+v", view)
	}
}

func TestConnectErrorSurfacesInStatus(t *testing.T) {
	ctrl := &fakeCtrl{
		snap: Snapshot{
			Phase:    PhaseDisconnected,
			Profiles: []Profile{{Name: "helsinki", Protocol: "AmneziaWG"}},
		},
		err: errRoot,
	}
	m := New(ctrl)
	done := make(chan struct{}, 8)
	m.OnChange = func() { done <- struct{}{} }
	m.Reload(context.Background())
	waitFunc(t, m, done, func(s Snapshot) bool { return len(s.Profiles) == 1 })
	m.Connect(context.Background())
	view := waitPhase(t, m, done, PhaseError)
	if view.Phase != PhaseError || view.Err == "" {
		t.Fatalf("view = %+v", view)
	}
}

func (s Snapshot) SelectedProfileName() string {
	p, ok := s.SelectedProfile()
	if !ok {
		return ""
	}
	return p.Name
}

type fakeCtrl struct {
	mu        sync.Mutex
	snap      Snapshot
	connected Snapshot
	err       error
}

func (f *fakeCtrl) Load(context.Context) (Snapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.snap, nil
}

func (f *fakeCtrl) Connect(context.Context, string) (Snapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return Snapshot{}, f.err
	}
	return f.connected, nil
}

func (f *fakeCtrl) Disconnect(context.Context) (Snapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.snap.Phase = PhaseDisconnected
	return f.snap, nil
}

type rootErr string

func (e rootErr) Error() string { return string(e) }

const errRoot rootErr = "connect requires root (CAP_NET_ADMIN)"

func wait(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the panel to update")
	}
}

func waitPhase(t *testing.T, m *Model, ch <-chan struct{}, want Phase) Snapshot {
	t.Helper()
	return waitFunc(t, m, ch, func(s Snapshot) bool { return s.Phase == want })
}

func waitFunc(t *testing.T, m *Model, ch <-chan struct{}, ok func(Snapshot) bool) Snapshot {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		view := m.View()
		if ok(view) {
			return view
		}
		select {
		case <-ch:
		case <-deadline:
			t.Fatalf("panel did not reach the expected state: %+v", view)
		}
	}
}
