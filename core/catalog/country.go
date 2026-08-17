// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package catalog

import "strings"

// countryNames maps lowercase country name fragments to ISO 3166-1 alpha-2
// codes. Profile names commonly embed the country (e.g. "AustriaGrazS6",
// "Netherlands, Amsterdam"), so we infer a zone for selection/filtering.
var countryNames = map[string]string{
	"austria": "AT", "belgium": "BE", "bulgaria": "BG", "canada": "CA",
	"chile": "CL", "croatia": "HR", "czech": "CZ", "denmark": "DK",
	"estonia": "EE", "finland": "FI", "france": "FR", "germany": "DE",
	"greece": "GR", "hungary": "HU", "iceland": "IS", "ireland": "IE",
	"italy": "IT", "japan": "JP", "latvia": "LV", "lithuania": "LT",
	"luxembourg": "LU", "netherlands": "NL", "norway": "NO", "poland": "PL",
	"portugal": "PT", "romania": "RO", "serbia": "RS", "singapore": "SG",
	"slovakia": "SK", "slovenia": "SI", "spain": "ES", "sweden": "SE",
	"switzerland": "CH", "turkey": "TR", "ukraine": "UA", "usa": "US",
	"unitedstates": "US", "unitedkingdom": "GB", "britain": "GB", "england": "GB",
	"australia": "AU", "brazil": "BR", "mexico": "MX", "india": "IN",
	"southkorea": "KR", "korea": "KR", "hongkong": "HK", "taiwan": "TW",
	"moldova": "MD", "cyprus": "CY", "malta": "MT",
}

// inferCountry guesses a 2-letter country code from a profile name.
func inferCountry(name string) string {
	lower := strings.ToLower(name)
	// Longest fragment match wins (e.g. "unitedstates" before "states").
	best, bestLen := "", 0
	for frag, code := range countryNames {
		if strings.Contains(lower, frag) && len(frag) > bestLen {
			best, bestLen = code, len(frag)
		}
	}
	return best
}
