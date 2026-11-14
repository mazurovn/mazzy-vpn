// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"bufio"
	"strings"
)

// flat is the toeac program after the language has rendered it.
// Values are kept in memory and are not logged.
type flat struct {
	adminBind       string
	adminCredential string
	statusSocket    string
}

func parseFlat(text string) (flat, error) {
	var out flat
	sawOK := false
	sawCore := false
	sawBackend := false
	sawFront := false
	sawRelay := false
	sawProtect := false
	sawDoctor := false
	seen := map[string]bool{}
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		key, val, _ := strings.Cut(line, " ")
		if key != "ok" && key != "allow" && key != "artifact" && key != "logic" && seen[key] {
			return flat{}, errRefused
		}
		if key != "ok" && key != "artifact" && key != "logic" {
			seen[key] = true
		}
		switch key {
		case "ok":
			sawOK = true
		case "transport":
			if val != "nested-https" {
				return flat{}, errRefused
			}
		case "claim-clean":
			if val != "no" {
				return flat{}, errRefused
			}
		case "require-auth":
			if val != "yes" {
				return flat{}, errRefused
			}
		case "admin-bind":
			out.adminBind = val
		case "admin-credential":
			out.adminCredential = val
		case "status-socket":
			out.statusSocket = val
		case "bind", "cert", "key", "idle", "max", "doctor", "credential", "allow",
			"mcp-socket", "a2a-bind", "admin-url":
		case "artifact":
			switch val {
			case "mazzy-core mcp":
				if sawCore {
					return flat{}, errRefused
				}
				sawCore = true
			case "mazzy-backend admin":
				if sawBackend {
					return flat{}, errRefused
				}
				sawBackend = true
			case "mazzy-front a2a":
				if sawFront {
					return flat{}, errRefused
				}
				sawFront = true
			default:
				return flat{}, errRefused
			}
		case "logic":
			switch val {
			case "relay copy":
				if sawRelay {
					return flat{}, errRefused
				}
				sawRelay = true
			case "protect mojo":
				if sawProtect {
					return flat{}, errRefused
				}
				sawProtect = true
			case "doctor nim":
				if sawDoctor {
					return flat{}, errRefused
				}
				sawDoctor = true
			default:
				return flat{}, errRefused
			}
		default:
			return flat{}, errRefused
		}
	}
	if !sawOK || !sawCore || !sawBackend || !sawFront || !sawRelay || !sawProtect || !sawDoctor ||
		out.adminBind == "" || len(out.adminCredential) < 16 || out.statusSocket == "" {
		return flat{}, errRefused
	}
	for _, part := range []string{out.adminBind, out.adminCredential, out.statusSocket} {
		if strings.Contains(part, "REPLACE_") {
			return flat{}, errPlaceholder
		}
	}
	return out, nil
}
