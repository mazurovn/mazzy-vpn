// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

//go:build unix

package main

import (
	"io/fs"
	"os"
	"syscall"
	"testing"
	"time"
)

type hopInfo struct {
	size int64
	mode os.FileMode
	uid  uint32
}

func (h hopInfo) Name() string       { return "virtual-hop" }
func (h hopInfo) Size() int64        { return h.size }
func (h hopInfo) Mode() os.FileMode  { return h.mode }
func (h hopInfo) ModTime() time.Time { return time.Time{} }
func (h hopInfo) IsDir() bool        { return h.mode.IsDir() }
func (h hopInfo) Sys() any           { return &syscall.Stat_t{Uid: h.uid} }

func TestRootOnlyFileCountsWithoutReturningBody(t *testing.T) {
	root := hopInfo{size: 7, mode: 0o600, uid: 0}
	if !hopOn("", root, nil, []byte("present")) {
		t.Fatal("private root file did not count")
	}
	if hopOn("", root, nil, []byte(" \n")) {
		t.Fatal("blank file counted")
	}
	if !hopOn("", root, fs.ErrPermission, nil) {
		t.Fatal("unreadable private file did not count")
	}
	loose := hopInfo{size: 7, mode: 0o644, uid: 0}
	if hopOn("", loose, nil, []byte("present")) {
		t.Fatal("world-readable file counted")
	}
	user := hopInfo{size: 7, mode: 0o600, uid: 1000}
	if hopOn("", user, nil, []byte("present")) {
		t.Fatal("non-root file counted")
	}
	link := hopInfo{size: 7, mode: os.ModeSymlink | 0o600, uid: 0}
	if hopOn("", link, nil, []byte("present")) {
		t.Fatal("symlink counted")
	}
}
