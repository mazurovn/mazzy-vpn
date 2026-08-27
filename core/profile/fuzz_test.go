// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package profile

import (
	"testing"

	"github.com/mazurovn/mazzy-vpn/core"
)

// FuzzParse ensures the .conf parser never panics on arbitrary input, and that
// any successfully parsed config renders to UAPI without panicking either.
func FuzzParse(f *testing.F) {
	f.Add("[Interface]\nPrivateKey = AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=\n[Peer]\nPublicKey = AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=\nAllowedIPs = 0.0.0.0/0\nEndpoint = h:1\n")
	f.Add("[Interface]\nJc=4\n")
	f.Add("garbage\n\x00\x00[[[")
	f.Add("[Interface]\nFwMark = 0xffffffff\n")
	f.Add("[Interface]\nMTU = 999999999999999999999\n")
	f.Fuzz(func(t *testing.T, in string) {
		c, err := Parse(in)
		if err != nil {
			return // rejecting bad input is fine; must not panic
		}
		// A parsed config must render to UAPI without panicking.
		_, _ = ToUAPI(core.AmneziaWG, c)
		_, _ = ToUAPI(core.WireGuard, c)
		_ = Validate(core.AmneziaWG, c)
		_ = c.EndpointHost()
	})
}
