// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package profile

import (
	"strings"

	"github.com/mazurovn/mazzy-vpn/core"
)

// Validate checks a parsed config for the required fields of the given
// protocol. Parity with the bash validate_profile() for amneziawg/wireguard.
//
// It returns a slice of human-readable problems; empty means valid.
func Validate(proto core.Protocol, c *Config) []string {
	var problems []string

	if c.PrivateKey == "" {
		problems = append(problems, "missing PrivateKey")
	}
	if len(c.Peers) == 0 {
		problems = append(problems, "missing [Peer]")
	}
	hasPubKey := false
	hasAllowed := false
	hasEndpoint := false
	for _, p := range c.Peers {
		if p.PublicKey != "" {
			hasPubKey = true
		}
		if len(p.AllowedIPs) > 0 {
			hasAllowed = true
		}
		if p.Endpoint != "" {
			hasEndpoint = true
		}
	}
	if !hasPubKey {
		problems = append(problems, "missing PublicKey")
	}
	if !hasAllowed {
		problems = append(problems, "missing AllowedIPs")
	}
	if !hasEndpoint {
		problems = append(problems, "missing Endpoint")
	}

	if proto == core.AmneziaWG {
		problems = append(problems, validateAmnezia(c)...)
	}
	return problems
}

// requiredAmneziaKeys are the minimum obfuscation parameters an AmneziaWG
// profile must carry (parity with the bash check for Jc/Jmin/Jmax/S1/S2/H1..H4).
// S3/S4 and i1..i5 are optional newer fields.
var requiredAmneziaKeys = []string{"jc", "jmin", "jmax", "s1", "s2", "h1", "h2", "h3", "h4"}

// validateAmnezia enforces presence of the required AmneziaWG obfuscation
// parameters.
func validateAmnezia(c *Config) []string {
	var missing []string
	for _, k := range requiredAmneziaKeys {
		if !c.Amnezia.Has(k) {
			missing = append(missing, strings.ToUpper(k))
		}
	}
	if len(missing) > 0 {
		return []string{"missing AmneziaWG parameter(s): " + strings.Join(missing, ", ")}
	}
	return nil
}
