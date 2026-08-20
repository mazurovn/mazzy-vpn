// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func() int) (string, int) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	code := fn()
	_ = w.Close()
	os.Stdout = old
	data, _ := io.ReadAll(r)
	_ = r.Close()
	return string(data), code
}

func TestEveryCommandSupportsSideEffectFreeHelpFlags(t *testing.T) {
	for _, name := range registeredCommandNames() {
		for _, flag := range []string{"-h", "--help"} {
			t.Run(name+flag, func(t *testing.T) {
				out, code := captureStdout(t, func() int { return run([]string{name, flag}) })
				if code != 0 {
					t.Fatalf("run(%s %s) = %d, want 0", name, flag, code)
				}
				if !strings.Contains(out, "Usage:") || !strings.Contains(out, flag[:2]) {
					t.Errorf("help output missing usage/options: %q", out)
				}
			})
		}
	}
}

func TestHelpCommandTargetsSubcommand(t *testing.T) {
	out, code := captureStdout(t, func() int { return run([]string{"help", "import"}) })
	if code != 0 || !strings.Contains(out, "mazzy-vpn import <FILE|DIR>...") {
		t.Fatalf("help import code=%d output=%q", code, out)
	}
}
