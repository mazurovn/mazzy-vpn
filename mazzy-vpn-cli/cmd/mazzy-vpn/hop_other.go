// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

//go:build !unix

package main

import "os"

func uidRoot(os.FileInfo) bool { return false }
