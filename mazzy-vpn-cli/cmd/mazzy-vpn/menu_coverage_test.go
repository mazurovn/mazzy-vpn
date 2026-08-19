// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"os"
	"strings"
	"testing"
)

// TestMenuCoversUserFacingCommands enforces audit P2: every user-facing command
// verb that a human would reach from the interactive menu must actually be
// wired into the line menu. This prevents "missing menu functions" regressions.
//
// It reads menu.go and checks that each expected action appears either as a
// direct cmd* call or via runPrivileged.
func TestMenuCoversUserFacingCommands(t *testing.T) {
	src, err := os.ReadFile("menu.go")
	if err != nil {
		t.Fatalf("read menu.go: %v", err)
	}
	menu := string(src)

	// action -> a token that must be present in menu.go when it is wired.
	required := map[string]string{
		"quick connect": "menuQuickConnect",
		"choose zone":   "menuChooseZone",
		"disconnect":    `runPrivileged(ctx, "disconnect"`,
		"recover":       `runPrivileged(ctx, "recover"`,
		"test":          "cmdTest",
		"best":          "cmdBest",
		"adapters":      "cmdAdapters",
		"netdiag":       "cmdNetdiag",
		"providers":     "menuProviders",
		"import":        "menuImport",
		"profiles":      "cmdProfiles",
		"settings":      "menuSettings",
		"doctor":        "cmdDoctor",
		"update":        "cmdUpdate",
		"diagnose":      "cmdDiagnose",
		"trace":         "menuTrace",
		"stealth":       "cmdStealth",
		"dns-check":     "cmdDNSCheck",
		// P2 additions — previously missing from the menu:
		"favorite": "menuFavorite",
		"remove":   "menuRemove",
		"mimic":    `runPrivileged(ctx, "mimic"`,
		"language": "menuLanguage",
		// Non-blocking dashboard + background/log additions (the old blocking
		// foreground "daemon" menu action is replaced by the detached background
		// daemon so the user lands back in the menu):
		"background": "menuBackground",
		"view log":   "menuViewLog",
		"stop bg":    "menuStopBackground",
	}

	for action, token := range required {
		if !strings.Contains(menu, token) {
			t.Errorf("menu is missing the %q action (expected token %q in menu.go)", action, token)
		}
	}
}
