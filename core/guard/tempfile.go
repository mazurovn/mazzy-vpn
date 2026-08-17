// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package guard

import (
	"os"
)

// writePrivateTemp writes content to a 0600 temp file and returns its path.
// nft applies rulesets atomically from a file; this keeps parity with the bash
// `nft -f -` while avoiding shell/stdin plumbing.
func writePrivateTemp(content string) (string, error) {
	f, err := os.CreateTemp("", "mazzy-guard-*.nft")
	if err != nil {
		return "", err
	}
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func removeFile(path string) { _ = os.Remove(path) }
