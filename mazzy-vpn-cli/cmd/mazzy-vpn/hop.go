// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"errors"
	"io/fs"
	"os"
	"strings"
)

// virtualHopEnv and virtualHopPath are presence checks. The value is an
// upstream URL and is never printed, logged, or written into a profile.
const (
	virtualHopEnv  = "MAZZY_VIRTUAL_HOP"
	virtualHopPath = "/etc/mazzy-vpn/virtual-hop"
)

// virtualHopConfigured reports whether the operator named an upstream.
// A symlink is ignored. The file counts only when root owns it and the
// mode is private (0600). An unreadable private file still counts: the
// user session can see that it exists without learning the URL.
func virtualHopConfigured() bool {
	env := os.Getenv(virtualHopEnv)
	fi, err := os.Lstat(virtualHopPath)
	if err != nil {
		fi = nil
	}
	var body []byte
	var readErr error
	// Read only a private root-owned file. A looser file is not a hop,
	// and its contents stay on disk.
	if fi != nil && fi.Mode().IsRegular() && fi.Size() > 0 && fi.Mode().Perm()&0o077 == 0 && uidRoot(fi) {
		body, readErr = os.ReadFile(virtualHopPath)
	}
	return hopOn(env, fi, readErr, body)
}

// hopOn is the pure decision. env is the session value. fi is the root
// file, or nil when it is absent. body is returned only to be discarded
// after a non-empty check.
func hopOn(env string, fi os.FileInfo, readErr error, body []byte) bool {
	if strings.TrimSpace(env) != "" {
		return true
	}
	if fi == nil || !fi.Mode().IsRegular() || fi.Size() == 0 {
		return false
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return false
	}
	if !uidRoot(fi) {
		return false
	}
	if readErr != nil {
		return errors.Is(readErr, fs.ErrPermission)
	}
	return strings.TrimSpace(string(body)) != ""
}
