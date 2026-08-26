// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package settings persists user preferences for the Mazzy VPN CLI/TUI:
// auto-connect, auto-diagnostics, notifications, preferred zone, and more.
// It is a small JSON file so both the CLI and a future GUI share one config.
package settings

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
)

// sudoUserNameRe accepts conventional Unix user names only, so a crafted
// SUDO_USER (e.g. "../etc") can never traverse outside the user's home.
var sudoUserNameRe = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}\$?$`)

// sudoUserSettingsPath resolves the INVOKING user's settings file when running
// as root under sudo. Hardened: the name is validated against a strict
// pattern, the home directory comes from the real user database (never a
// blind /home/<name> join), and the file must be owned by that user and not
// group/world-writable — a symlink-planted or attacker-writable settings.json
// must not be able to steer root's kill-switch policy.
func sudoUserSettingsPath() (string, bool) {
	su := os.Getenv("SUDO_USER")
	if su == "" || su == "root" || !sudoUserNameRe.MatchString(su) {
		return "", false
	}
	u, err := user.Lookup(su)
	if err != nil || u.HomeDir == "" {
		return "", false
	}
	p := filepath.Join(u.HomeDir, ".config", "mazzy-vpn", "settings.json")
	fi, err := os.Stat(p)
	if err != nil || !fi.Mode().IsRegular() {
		return "", false
	}
	if fi.Mode().Perm()&0o022 != 0 {
		return "", false // group/world-writable: untrusted
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		if uid, err := strconv.ParseUint(u.Uid, 10, 32); err != nil || st.Uid != uint32(uid) {
			return "", false // not owned by the invoking user
		}
	}
	return p, true
}

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
//
// Root-under-sudo reads the INVOKING user's settings: the daemon/connect run
// elevated, but the toggles (kill-switch, notifications, …) belong to the
// human who configured them in the menu. Without this the root daemon read
// /root/.config (absent) and silently fell back to defaults — the incident
// where a user with kill_switch:false got a fail-closed kill-switch anyway
// because root's defaults said true.
func DefaultPath() string {
	if d := os.Getenv("MAZZY_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "settings.json")
	}
	if os.Geteuid() == 0 {
		if p, ok := sudoUserSettingsPath(); ok {
			return p
		}
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
