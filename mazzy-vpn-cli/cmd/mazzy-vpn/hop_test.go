// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHopEnvIsPresenceOnly(t *testing.T) {
	if !hopOn("present", nil, nil, nil) {
		t.Fatal("session value did not count")
	}
	if hopOn("  \n", nil, nil, nil) {
		t.Fatal("blank session value counted")
	}
	if hopOn("", nil, nil, []byte("present")) {
		t.Fatal("body counted without a file")
	}
}

func TestEgressJSONKeepsHopValueOut(t *testing.T) {
	doc := egressDoc{
		VirtualHop:  true,
		Antigravity: serviceCheck{Service: "antigravity", State: "eligible", Reason: "auth boundary reached"},
	}
	raw := mustJSON(t, doc)
	if !strings.Contains(raw, `"virtual_hop":true`) {
		t.Fatalf("missing flag: %s", raw)
	}
	if strings.Contains(raw, "://") || strings.Contains(raw, virtualHopEnv) || strings.Contains(raw, virtualHopPath) {
		t.Fatalf("hop value leaked: %s", raw)
	}
	doc.VirtualHop = false
	raw = mustJSON(t, doc)
	if strings.Contains(raw, "virtual_hop") {
		t.Fatalf("unset hop encoded: %s", raw)
	}
}

func mustJSON(t *testing.T, doc egressDoc) string {
	t.Helper()
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
