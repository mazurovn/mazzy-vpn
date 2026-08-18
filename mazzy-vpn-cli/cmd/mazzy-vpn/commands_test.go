// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"strings"
	"testing"
)

func TestSafeDisplayStripsControl(t *testing.T) {
	in := "zone\x1b[31mRED\x1b[0m\x07\nNL\t中文"
	got := safeDisplay(in)
	for _, r := range got {
		if (r < 0x20 && r != ' ') || (r >= 0x7f && r <= 0x9f) {
			t.Errorf("control char survived: %q in %q", r, got)
		}
	}
	if !strings.Contains(got, "中文") {
		t.Error("printable Unicode must be preserved")
	}
	if strings.ContainsAny(got, "\x1b\x07\n") {
		t.Errorf("escape/bell/newline survived: %q", got)
	}
}
