// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package catalog manages a user's imported VPN profiles ("zones"): importing
// them into a managed store, listing, selecting, favoriting and removing them.
// It gives the CLI a real client workflow instead of passing raw file paths.
package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/profile"
)

// Entry is one managed profile.
type Entry struct {
	Name     string        `json:"name"`     // stable display name (file stem)
	File     string        `json:"file"`     // absolute path in the managed store
	Protocol core.Protocol `json:"protocol"` // amneziawg/wireguard/openvpn
	Favorite bool          `json:"favorite"`
	Country  string        `json:"country,omitempty"` // inferred 2-letter code
}

// Catalog is a file-backed registry of managed profiles.
type Catalog struct {
	Dir string // e.g. ~/.config/mazzy-vpn/profiles
}

var (
	// ErrNotFound is returned when a named entry does not exist.
	ErrNotFound = errors.New("catalog: profile not found")
	// ErrExists is returned when importing a duplicate name.
	ErrExists = errors.New("catalog: profile already exists")
)

// New returns a Catalog rooted at dir.
func New(dir string) *Catalog { return &Catalog{Dir: dir} }

// DefaultDir returns the catalog directory, honoring MAZZY_CONFIG_DIR.
//
// Root-under-sudo reads the INVOKING user's catalog: the daemon/connect run
// elevated (HOME=/root under sudo), but the profiles were imported by the human
// into ~/.config. Without this the root daemon read /root/.config (a different,
// usually empty or stale catalog) — so zones the user imported were "profile
// not found" when the daemon tried to connect/switch to them, and the daemon
// and the interactive CLI silently used two divergent catalogs. Parity with
// the desired.json (P0-A) and settings SUDO_USER fixes. An explicit
// MAZZY_CONFIG_DIR (e.g. the systemd unit) still wins.
func DefaultDir() string {
	if d := os.Getenv("MAZZY_CONFIG_DIR"); d != "" {
		return d
	}
	if p, ok := sudoUserCatalogDir(); ok {
		return p
	}
	if h, err := os.UserConfigDir(); err == nil {
		return filepath.Join(h, "mazzy-vpn", "profiles")
	}
	return filepath.Join(os.TempDir(), "mazzy-vpn", "profiles")
}

// sudoUserCatalogDir returns the invoking user's catalog dir when running as
// root under sudo. Uses the real user database (never a blind /home/<name>) and
// only when the dir already exists, so a normal root install is unaffected.
func sudoUserCatalogDir() (string, bool) {
	if os.Geteuid() != 0 {
		return "", false
	}
	su := os.Getenv("SUDO_USER")
	if su == "" || su == "root" || !sudoUserNameRe.MatchString(su) {
		return "", false
	}
	u, err := user.Lookup(su)
	if err != nil || u.HomeDir == "" {
		return "", false
	}
	p := filepath.Join(u.HomeDir, ".config", "mazzy-vpn", "profiles")
	// SECURITY: Lstat (not Stat) so a symlink at `p` is NOT followed — otherwise
	// a least-privilege sudoer (allowed to run only mazzy-vpn as root) could
	// plant ~/.config/mazzy-vpn/profiles → /etc and make root chmod/write there.
	// Require a real directory owned by the invoking user.
	fi, err := os.Lstat(p)
	if err != nil || !fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 {
		return "", false
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		if uid, err := strconv.ParseUint(u.Uid, 10, 32); err != nil || st.Uid != uint32(uid) {
			return "", false // not owned by the invoking user: untrusted
		}
	}
	return p, true
}

// sudoUserNameRe accepts conventional Unix user names only (no path traversal
// via a crafted SUDO_USER).
var sudoUserNameRe = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}\$?$`)

func (c *Catalog) metaFile() string { return filepath.Join(c.Dir, ".catalog.json") }

// fileWithinDir reports whether p resolves to a location inside c.Dir. It is
// the single guard against a path-injection: a hand-edited manifest, a crafted
// profile name, or a stored File value must never let a catalog operation
// read/delete a file outside the managed store (defense in depth — names are
// already sanitized on import, but Get/Remove also trust the stored File).
func (c *Catalog) fileWithinDir(p string) bool {
	dir, err := filepath.Abs(c.Dir)
	if err != nil {
		return false
	}
	fp, err := filepath.Abs(p)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(dir, fp)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// validLookupName rejects a name that could traverse the filesystem when it is
// (directly or via a stored File) turned into a path. Managed names never
// contain separators or "..", so this is a cheap correctness gate.
func validLookupName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	return !strings.ContainsAny(name, "/\\") && !strings.Contains(name, "..")
}

// ensureDir makes the managed directory with private permissions.
func (c *Catalog) ensureDir() error {
	if err := os.MkdirAll(c.Dir, 0o700); err != nil {
		return err
	}
	return os.Chmod(c.Dir, 0o700)
}

// safeName sanitizes a filename stem into a stable catalog name.
func safeName(path string) string {
	base := filepath.Base(path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	// Keep alnum, dash, underscore; collapse the rest to '-'.
	var b strings.Builder
	for _, r := range stem {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		name = "profile"
	}
	return name
}

// detectProtocol infers a protocol from extension + content.
func detectProtocol(path string, data []byte) core.Protocol {
	if strings.EqualFold(filepath.Ext(path), ".ovpn") {
		return core.OpenVPN
	}
	if cfg, err := profile.Parse(string(data)); err == nil && cfg.HasAmneziaFields {
		return core.AmneziaWG
	}
	return core.WireGuard
}

// Import copies a profile file into the managed store and records metadata. It
// validates WireGuard/AmneziaWG profiles before accepting them.
func (c *Catalog) Import(path string) (*Entry, error) {
	if err := c.ensureDir(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile: %w", err)
	}
	proto := detectProtocol(path, data)
	if proto == core.AmneziaWG || proto == core.WireGuard {
		cfg, err := profile.Parse(string(data))
		if err != nil {
			return nil, fmt.Errorf("parse profile: %w", err)
		}
		if problems := profile.Validate(proto, cfg); len(problems) != 0 {
			return nil, fmt.Errorf("invalid profile: %s", strings.Join(problems, "; "))
		}
	}

	// Reject an exact-content duplicate already in the store (different name,
	// same profile) so the catalog stays clean.
	fp := fingerprint(data)
	existing, _ := c.load()
	for i := range existing {
		if d, err := os.ReadFile(existing[i].File); err == nil && fingerprint(d) == fp {
			return nil, fmt.Errorf("%w: identical to %q", ErrExists, existing[i].Name)
		}
	}

	name := safeName(path)
	ext := ".conf"
	if proto == core.OpenVPN {
		ext = ".ovpn"
	}
	dest := filepath.Join(c.Dir, name+ext)
	if _, err := os.Stat(dest); err == nil {
		return nil, ErrExists
	}
	if err := os.WriteFile(dest, data, 0o600); err != nil {
		return nil, err
	}

	entry := Entry{Name: name, File: dest, Protocol: proto, Country: inferCountry(name)}
	entries := existing
	entries = append(entries, entry)
	if err := c.save(entries); err != nil {
		return nil, err
	}
	return &entry, nil
}

// List returns all managed entries, favorites first then alphabetical. It
// always returns a non-nil slice so JSON callers see [] rather than null.
func (c *Catalog) List() ([]Entry, error) {
	entries, err := c.load()
	if err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []Entry{}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Favorite != entries[j].Favorite {
			return entries[i].Favorite
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

// Get returns one entry by name.
func (c *Catalog) Get(name string) (*Entry, error) {
	if !validLookupName(name) {
		return nil, ErrNotFound
	}
	entries, err := c.load()
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].Name == name {
			if !c.fileWithinDir(entries[i].File) {
				return nil, fmt.Errorf("catalog: profile %q resolves outside the managed store", name)
			}
			return &entries[i], nil
		}
	}
	return nil, ErrNotFound
}

// Remove deletes a managed profile and its file.
func (c *Catalog) Remove(name string) error {
	entries, err := c.load()
	if err != nil {
		return err
	}
	out := entries[:0]
	found := false
	for _, e := range entries {
		if e.Name == name {
			// Only ever delete inside the managed store — never a path a crafted
			// manifest could point at (e.g. an absolute /etc/... File value).
			if c.fileWithinDir(e.File) {
				_ = os.Remove(e.File)
			}
			found = true
			continue
		}
		out = append(out, e)
	}
	if !found {
		return ErrNotFound
	}
	return c.save(out)
}

// SetFavorite toggles the favorite flag on an entry.
func (c *Catalog) SetFavorite(name string, fav bool) error {
	entries, err := c.load()
	if err != nil {
		return err
	}
	found := false
	for i := range entries {
		if entries[i].Name == name {
			entries[i].Favorite = fav
			found = true
		}
	}
	if !found {
		return ErrNotFound
	}
	return c.save(entries)
}

// Count returns the number of managed profiles.
func (c *Catalog) Count() int {
	entries, _ := c.load()
	return len(entries)
}

// load reads the metadata file (empty slice when absent).
func (c *Catalog) load() ([]Entry, error) {
	data, err := os.ReadFile(c.metaFile())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("corrupt catalog metadata: %w", err)
	}
	return entries, nil
}

// save atomically writes the metadata file.
func (c *Catalog) save(entries []Entry) error {
	if err := c.ensureDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.metaFile() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, c.metaFile())
}

// fingerprint returns a short content hash (for dedupe/debugging).
func fingerprint(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}
