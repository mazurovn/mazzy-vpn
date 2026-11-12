// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import "testing"

func TestClassifyAntigravity(t *testing.T) {
	cases := []struct {
		status int
		net    bool
		state  string
	}{
		{0, true, "unreachable"},
		{401, false, "eligible"},
		{403, false, "eligible"},
		{404, false, "eligible"},
		{429, false, "rate-limited"},
		{503, false, "unavailable"},
		{200, false, "indeterminate"},
	}
	for _, c := range cases {
		state, reason := classifyService(c.status, c.net)
		if state != c.state || reason == "" {
			t.Fatalf("status %d net %v -> %s %q", c.status, c.net, state, reason)
		}
	}
}

func TestEgressDocOmitsMissingIP(t *testing.T) {
	doc := egressDoc{}
	doc.Unknown = append(doc.Unknown, gap{Check: "ipv4", Reason: "ip-api did not return an address"})
	if doc.IPv4 != "" || doc.Proxy != nil || doc.Hosting != nil {
		t.Fatalf("invented address flags: %+v", doc)
	}
	if doc.Unknown[0].Reason == "" {
		t.Fatal("missing reason")
	}
}
