// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package state persists the connection intent (which profile, up/down, mode)
// with the same durability guarantees as the bash write_state: atomic
// temp+rename, 0700 dir / 0600 file, fsync of file and parent directory.
package state

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mazurovn/mazzy-vpn/core"
)

// State is the persisted connection intent. Parity with the bash STATE_FILE
// key=value format (PROTOCOL/PROFILE/DESIRED/MODE/TEST_TOKEN/TEST_DEADLINE).
type State struct {
	Protocol     core.Protocol
	Profile      string // basename only, matching bash
	Desired      core.DesiredState
	Mode         core.Mode
	TestToken    string
	TestDeadline int64 // unix seconds; 0 if unset
}

// Store reads and writes State atomically under Dir.
type Store struct {
	Dir string // e.g. /var/lib/mazzy-vpn
}

func (s *Store) file() string { return filepath.Join(s.Dir, "active") }

// Write atomically persists st. It mirrors write_state's durability: 0700 dir,
// 0600 file, temp+rename, fsync of file and directory.
func (s *Store) Write(st State) error {
	if st.Protocol == "" || st.Profile == "" {
		return fmt.Errorf("state: protocol and profile are required")
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(s.Dir, 0o700); err != nil {
		return err
	}
	mode := st.Mode
	if mode == "" {
		mode = core.ModeNormal
	}

	var b strings.Builder
	fmt.Fprintf(&b, "PROTOCOL=%s\n", st.Protocol)
	fmt.Fprintf(&b, "PROFILE=%s\n", filepath.Base(st.Profile))
	fmt.Fprintf(&b, "DESIRED=%s\n", st.Desired)
	fmt.Fprintf(&b, "MODE=%s\n", mode)
	if mode == core.ModeTest {
		fmt.Fprintf(&b, "TEST_TOKEN=%s\n", st.TestToken)
		fmt.Fprintf(&b, "TEST_DEADLINE=%d\n", st.TestDeadline)
	}

	return atomicWrite(s.file(), []byte(b.String()))
}

// SetDesired updates only the DESIRED field, preserving the rest.
func (s *Store) SetDesired(d core.DesiredState) error {
	st, err := s.Read()
	if err != nil {
		return err
	}
	st.Desired = d
	return s.Write(*st)
}

// Read loads the persisted State. Returns an error if the file is absent.
func (s *Store) Read() (*State, error) {
	f, err := os.Open(s.file())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	kv := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		i := strings.IndexByte(line, '=')
		if i < 0 {
			continue
		}
		kv[line[:i]] = line[i+1:]
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	proto, ok := core.CanonicalProtocol(kv["PROTOCOL"])
	if !ok {
		return nil, fmt.Errorf("state: unknown or missing PROTOCOL %q", kv["PROTOCOL"])
	}
	if kv["PROFILE"] == "" {
		return nil, fmt.Errorf("state: missing PROFILE")
	}
	st := &State{
		Protocol: proto,
		Profile:  kv["PROFILE"],
		Desired:  core.DesiredState(kv["DESIRED"]),
		Mode:     core.Mode(kv["MODE"]),
	}
	if st.Mode == "" {
		st.Mode = core.ModeNormal
	}
	if st.Mode == core.ModeTest {
		st.TestToken = kv["TEST_TOKEN"]
		if d, err := strconv.ParseInt(kv["TEST_DEADLINE"], 10, 64); err == nil {
			st.TestDeadline = d
		}
	}
	return st, nil
}

// atomicWrite writes data to path via temp+rename with fsync of the file and
// its parent directory. Parity with write_state's sync -f.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".active.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return fsyncDir(dir)
}

func fsyncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
