// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package netexectest provides a fake Runner for unit tests so that no real
// kernel network changes happen while asserting the exact argv mazzy-core
// would issue (parity checks).
package netexectest

import "context"

// Fake records every command and returns canned output/errors.
type Fake struct {
	Calls  []string
	Err    error
	Output string
}

// Run records the command as a single space-joined string and returns the
// canned response.
func (f *Fake) Run(_ context.Context, bin string, args ...string) (string, error) {
	call := bin
	for _, a := range args {
		call += " " + a
	}
	f.Calls = append(f.Calls, call)
	return f.Output, f.Err
}
