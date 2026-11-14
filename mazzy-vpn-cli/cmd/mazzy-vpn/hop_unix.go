// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

//go:build unix

package main

import (
	"os"
	"syscall"
)

func uidRoot(fi os.FileInfo) bool {
	if fi == nil || fi.Sys() == nil {
		return false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || st == nil {
		return false
	}
	return st.Uid == 0
}
