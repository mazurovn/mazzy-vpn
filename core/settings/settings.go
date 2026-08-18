// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package settings persists user preferences for the Mazzy VPN CLI/TUI:
// auto-connect, auto-diagnostics, notifications, preferred zone, and more.
// It is a small JSON file so both the CLI and a future GUI share one config.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings holds user preferences. Zero value is a sensible default.
type Settings struct {
	// AutoConnect connects to the preferred/best zone on start.
	AutoConnect bool `json:"auto_connect"`
	// AutoDiagnostics runs network diagnostics before connecting.
	AutoDiagnostics bool `json:"auto_diagnostics"`
	// Notifications enables desktop notifications.
	Notifications bool `json:"notifications"`
	// AutoReconnect keeps the tunnel up by reconnecting on drops.
	AutoReconnect bool `json:"auto_reconnect"`
	// PreferredZone is the default zone for quick-connect ("" = best).
	PreferredZone string `json:"preferred_zone"`
	// PreferredUplink pins measurements/connect to an interface ("" = auto).
	PreferredUplink string `json:"preferred_uplink"`
	// KillSwitch keeps the fail-closed guard on if the tunnel drops.
	KillSwitch bool `json:"kill_switch"`
	// AutoMimic aligns the system timezone to the egress country on connect, so
	// services (Google/Gemini/Antigravity) see a consistent local timezone.
	AutoMimic bool `json:"auto_mimic"`
	// Language is the UI language code (en/ru/de/zh/ja/ko). Empty means resolve
	// from the OS locale, defaulting to English. Never hardcoded at call sites.
	Language string `json:"language,omitempty"`
}

// Default returns settings with recommended defaults enabled.
func Default() Settings {
	return Settings{
		AutoConnect:     false,
		AutoDiagnostics: true,
		Notifications:   true,
		AutoReconnect:   true,
		KillSwitch:      true,
	}
}

// Store loads and saves settings from a JSON file.
type Store struct {
	Path string
}

// DefaultPath returns the per-user settings path (honors MAZZY_CONFIG_HOME).
func DefaultPath() string {
	if d := os.Getenv("MAZZY_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "settings.json")
	}
	if h, err := os.UserConfigDir(); err == nil {
		return filepath.Join(h, "mazzy-vpn", "settings.json")
	}
	return filepath.Join(os.TempDir(), "mazzy-vpn", "settings.json")
}

// NewStore returns a Store at the default path.
func NewStore() *Store { return &Store{Path: DefaultPath()} }

// Load reads settings, returning defaults when the file is absent.
func (s *Store) Load() Settings {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return Default()
	}
	var out Settings
	if err := json.Unmarshal(data, &out); err != nil {
		return Default()
	}
	return out
}

// Save writes settings atomically (temp + rename, 0600).
func (s *Store) Save(set Settings) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}
