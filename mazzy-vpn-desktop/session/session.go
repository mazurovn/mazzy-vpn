// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package session is the desktop's engine binding. It links mazzy-core
// directly (no CLI process) and owns the userspace tunnel for the life of
// the window.
package session

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/catalog"
	"github.com/mazurovn/mazzy-vpn/core/connect"
	"github.com/mazurovn/mazzy-vpn/core/engine/wireguard"
	"github.com/mazurovn/mazzy-vpn/core/lock"
	"github.com/mazurovn/mazzy-vpn/core/netexec"
	"github.com/mazurovn/mazzy-vpn/core/profile"
	"github.com/mazurovn/mazzy-vpn/core/state"
	"github.com/mazurovn/mazzy-vpn/desktop/panel"
)

// Config points the session at the same directories the CLI uses.
type Config struct {
	CatalogDir string
	StateDir   string
	RunDir     string
}

// DefaultConfig honors MAZZY_CONFIG_DIR, MAZZY_STATE_DIR and MAZZY_RUN_DIR.
func DefaultConfig() Config {
	stateDir := "/var/lib/mazzy-vpn"
	if d := os.Getenv("MAZZY_STATE_DIR"); d != "" {
		stateDir = d
	}
	runDir := "/run/mazzy-vpn"
	if d := os.Getenv("MAZZY_RUN_DIR"); d != "" {
		runDir = d
	}
	return Config{CatalogDir: catalog.DefaultDir(), StateDir: stateDir, RunDir: runDir}
}

// Session implements panel.Control.
type Session struct {
	cat   *catalog.Catalog
	store *state.Store
	run   string

	mu     sync.Mutex
	conn   *connect.Conn
	server string
	proto  core.Protocol
	path   string
	detail string
	phase  panel.Phase
}

// New binds a session to cfg.
func New(cfg Config) *Session {
	return &Session{
		cat:   catalog.New(cfg.CatalogDir),
		store: &state.Store{Dir: cfg.StateDir},
		run:   cfg.RunDir,
		phase: panel.PhaseDisconnected,
	}
}

// Load lists profiles and reports a tunnel this process owns, or one left up
// by another Mazzy process.
func (s *Session) Load(context.Context) (panel.Snapshot, error) {
	return s.snapshot(), nil
}

// Connect brings up name. The returned snapshot's transport is the UDP port
// the handshake settled on, including a corporate-port fallback.
func (s *Session) Connect(ctx context.Context, name string) (panel.Snapshot, error) {
	if os.Geteuid() != 0 {
		return s.snapshot(), fmt.Errorf("connect requires root (CAP_NET_ADMIN); start Mazzy VPN Desktop with sudo")
	}
	entry, err := s.cat.Get(name)
	if err != nil {
		return s.snapshot(), err
	}
	if entry.Protocol != core.AmneziaWG && entry.Protocol != core.WireGuard {
		return s.snapshot(), fmt.Errorf("%s is not carried by the embedded engine (UDP AmneziaWG/WireGuard only)", entry.Protocol.Title())
	}
	proto, cfg, err := loadProfile(entry.File)
	if err != nil {
		return s.snapshot(), err
	}
	mu, err := lock.Acquire(s.run)
	if err != nil {
		return s.snapshot(), err
	}
	defer mu.Unlock()

	conn, err := connect.Up(ctx, proto, cfg, connect.Options{LogLevel: wireguard.LogError})
	if err != nil {
		return s.snapshot(), err
	}
	_ = s.store.Write(state.State{
		Protocol: proto, Profile: name, Desired: core.DesiredUp, Mode: core.ModeNormal,
	})

	detail := conn.Path.FallbackNote()
	phase := panel.PhaseConnected
	if !conn.Path.Handshake {
		phase = panel.PhaseError
		detail = conn.Path.FailureNote()
	}
	s.mu.Lock()
	if s.conn != nil {
		_ = s.conn.Down(ctx)
	}
	s.conn = conn
	s.server = name
	s.proto = proto
	s.path = conn.Path.Label()
	s.detail = detail
	s.phase = phase
	s.mu.Unlock()
	return s.snapshot(), nil
}

// Close drops a tunnel this process started. Closing the window must not
// delete a tunnel owned by the CLI.
func (s *Session) Close(ctx context.Context) {
	s.mu.Lock()
	conn := s.conn
	s.conn = nil
	s.phase = panel.PhaseDisconnected
	s.path = ""
	s.detail = ""
	s.mu.Unlock()
	if conn == nil {
		return
	}
	_ = conn.Down(ctx)
	if st, err := s.store.Read(); err == nil {
		st.Desired = core.DesiredDown
		_ = s.store.Write(*st)
	}
}

// Disconnect tears down the tunnel this process owns. A tunnel left by another
// process is removed by interface name, same as the CLI.
func (s *Session) Disconnect(ctx context.Context) (panel.Snapshot, error) {
	if os.Geteuid() != 0 {
		return s.snapshot(), fmt.Errorf("disconnect requires root (CAP_NET_ADMIN); start Mazzy VPN Desktop with sudo")
	}
	mu, err := lock.Acquire(s.run)
	if err != nil {
		return s.snapshot(), err
	}
	defer mu.Unlock()

	s.mu.Lock()
	conn := s.conn
	s.conn = nil
	s.mu.Unlock()

	var first error
	if conn != nil {
		first = conn.Down(ctx)
	} else {
		r := netexec.ExecRunner{}
		for _, iface := range core.ManagedInterfaces() {
			if _, statErr := net.InterfaceByName(iface); statErr != nil {
				continue
			}
			if _, delErr := r.Run(ctx, "ip", "link", "del", iface); delErr != nil && first == nil {
				first = delErr
			}
		}
	}
	if st, err := s.store.Read(); err == nil {
		st.Desired = core.DesiredDown
		_ = s.store.Write(*st)
	}
	s.mu.Lock()
	s.phase = panel.PhaseDisconnected
	s.path = ""
	s.detail = ""
	s.server = ""
	s.mu.Unlock()
	return s.snapshot(), first
}

func (s *Session) snapshot() panel.Snapshot {
	profiles, _ := s.profiles()
	s.mu.Lock()
	defer s.mu.Unlock()
	phase := s.phase
	server, proto, path, detail := s.server, s.proto.Title(), s.path, s.detail
	if s.conn == nil && phase != panel.PhaseError {
		if iface := liveInterface(); iface != "" {
			phase = panel.PhaseConnected
			if server == "" {
				if st, err := s.store.Read(); err == nil {
					server = st.Profile
					proto = st.Protocol.Title()
				}
			}
			if detail == "" {
				detail = "tunnel " + iface + " is up"
			}
		} else if phase == "" {
			phase = panel.PhaseDisconnected
		}
	}
	if proto == "" && phase == panel.PhaseDisconnected {
		proto = ""
	}
	return panel.Snapshot{
		Phase:     phase,
		Server:    server,
		Protocol:  proto,
		Transport: path,
		Detail:    detail,
		Profiles:  profiles,
	}
}

func (s *Session) profiles() ([]panel.Profile, error) {
	entries, err := s.cat.List()
	if err != nil {
		return nil, err
	}
	out := make([]panel.Profile, 0, len(entries))
	for _, e := range entries {
		item := panel.Profile{Name: e.Name, Protocol: e.Protocol.Title()}
		if data, err := os.ReadFile(e.File); err == nil {
			if cfg, err := profile.Parse(string(data)); err == nil {
				item.Endpoint = cfg.Endpoint()
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func loadProfile(path string) (core.Protocol, *profile.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	cfg, err := profile.Parse(string(data))
	if err != nil {
		return "", nil, err
	}
	proto := core.WireGuard
	if cfg.HasAmneziaFields {
		proto = core.AmneziaWG
	}
	if problems := profile.Validate(proto, cfg); len(problems) != 0 {
		return "", nil, fmt.Errorf("invalid profile: %s", problems[0])
	}
	return proto, cfg, nil
}

func liveInterface() string {
	for _, iface := range core.ManagedInterfaces() {
		if iface == "" {
			continue
		}
		if _, err := net.InterfaceByName(iface); err == nil {
			return iface
		}
	}
	return ""
}
