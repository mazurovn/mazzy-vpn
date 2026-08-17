// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package mimicry makes the local environment look like it belongs to the VPN
// egress country, defeating detection systems (Google/Gemini/Antigravity,
// Cloudflare) that compare the browser/system timezone and locale against the
// egress IP. The strongest tell is a timezone/egress mismatch; mimicry aligns
// the system timezone (and offers a per-process TZ for launching apps) to the
// egress country.
package mimicry

import (
	"strings"
)

// countryTZ maps an ISO 3166-1 alpha-2 country code to a representative IANA
// timezone (the capital/primary business zone). Used to align the local
// timezone with the VPN egress country.
var countryTZ = map[string]string{
	"AT": "Europe/Vienna", "BE": "Europe/Brussels", "BG": "Europe/Sofia",
	"CH": "Europe/Zurich", "CZ": "Europe/Prague", "DE": "Europe/Berlin",
	"DK": "Europe/Copenhagen", "EE": "Europe/Tallinn", "ES": "Europe/Madrid",
	"FI": "Europe/Helsinki", "FR": "Europe/Paris", "GB": "Europe/London",
	"GR": "Europe/Athens", "HR": "Europe/Zagreb", "HU": "Europe/Budapest",
	"IE": "Europe/Dublin", "IS": "Atlantic/Reykjavik", "IT": "Europe/Rome",
	"LT": "Europe/Vilnius", "LU": "Europe/Luxembourg", "LV": "Europe/Riga",
	"MD": "Europe/Chisinau", "NL": "Europe/Amsterdam", "NO": "Europe/Oslo",
	"PL": "Europe/Warsaw", "PT": "Europe/Lisbon", "RO": "Europe/Bucharest",
	"RS": "Europe/Belgrade", "SE": "Europe/Stockholm", "SI": "Europe/Ljubljana",
	"SK": "Europe/Bratislava", "TR": "Europe/Istanbul", "UA": "Europe/Kyiv",
	"RU": "Europe/Moscow",
	// Americas
	"US": "America/New_York", "CA": "America/Toronto", "BR": "America/Sao_Paulo",
	"MX": "America/Mexico_City", "CL": "America/Santiago", "AR": "America/Argentina/Buenos_Aires",
	// APAC / other
	"JP": "Asia/Tokyo", "KR": "Asia/Seoul", "SG": "Asia/Singapore",
	"HK": "Asia/Hong_Kong", "TW": "Asia/Taipei", "IN": "Asia/Kolkata",
	"AU": "Australia/Sydney", "IL": "Asia/Jerusalem", "KZ": "Asia/Almaty",
	"AE": "Asia/Dubai",
}

// countryLocale maps a country to a representative POSIX locale (for LANG/LC_*),
// used when launching apps that expose locale to fingerprinting.
var countryLocale = map[string]string{
	"NL": "nl_NL.UTF-8", "DE": "de_DE.UTF-8", "FR": "fr_FR.UTF-8",
	"GB": "en_GB.UTF-8", "US": "en_US.UTF-8", "ES": "es_ES.UTF-8",
	"IT": "it_IT.UTF-8", "SE": "sv_SE.UTF-8", "NO": "nb_NO.UTF-8",
	"FI": "fi_FI.UTF-8", "DK": "da_DK.UTF-8", "PL": "pl_PL.UTF-8",
	"AT": "de_AT.UTF-8", "BE": "nl_BE.UTF-8", "CH": "de_CH.UTF-8",
	"JP": "ja_JP.UTF-8", "KR": "ko_KR.UTF-8", "RU": "ru_RU.UTF-8",
}

// TimezoneFor returns the representative timezone for a country code, or "" if
// unknown.
func TimezoneFor(country string) string {
	return countryTZ[strings.ToUpper(strings.TrimSpace(country))]
}

// LocaleFor returns the representative locale for a country, or "" if unknown.
func LocaleFor(country string) string {
	return countryLocale[strings.ToUpper(strings.TrimSpace(country))]
}

// CountryForTimezone reverse-maps a timezone to a country code (best-effort),
// used to detect the current mismatch.
func CountryForTimezone(tz string) string {
	tz = strings.TrimSpace(tz)
	for cc, z := range countryTZ {
		if z == tz {
			return cc
		}
	}
	return ""
}

// Runner executes privileged system commands (e.g. timedatectl). Injected for
// testability.
type Runner interface {
	Run(bin string, args ...string) (string, error)
}

// Manager applies and reverts timezone mimicry.
type Manager struct {
	Runner Runner
	// CurrentTZ reads the active system timezone; injected for tests.
	CurrentTZ func() string
}

// Plan describes what mimicry would change (dry-run friendly).
type Plan struct {
	Country     string `json:"country"`
	FromTZ      string `json:"from_tz"`
	ToTZ        string `json:"to_tz"`
	Locale      string `json:"locale,omitempty"`
	NeedsChange bool   `json:"needs_change"`
}

// PlanFor computes the mimicry plan to match the egress country.
func (m *Manager) PlanFor(country string) (Plan, bool) {
	tz := TimezoneFor(country)
	if tz == "" {
		return Plan{}, false
	}
	cur := ""
	if m.CurrentTZ != nil {
		cur = m.CurrentTZ()
	}
	return Plan{
		Country:     strings.ToUpper(country),
		FromTZ:      cur,
		ToTZ:        tz,
		Locale:      LocaleFor(country),
		NeedsChange: cur != tz,
	}, true
}

// ApplySystemTZ sets the system timezone to match the country (needs root /
// timedatectl). Returns the plan applied.
func (m *Manager) ApplySystemTZ(country string) (Plan, error) {
	plan, ok := m.PlanFor(country)
	if !ok {
		return Plan{}, errUnknownCountry
	}
	if !plan.NeedsChange {
		return plan, nil
	}
	if m.Runner == nil {
		return plan, errNoRunner
	}
	_, err := m.Runner.Run("timedatectl", "set-timezone", plan.ToTZ)
	return plan, err
}

// ProcessEnv returns environment variables that make a launched process (e.g. a
// browser) present the egress country's timezone and locale WITHOUT changing
// the whole system. Safer than a system-wide change.
func (m *Manager) ProcessEnv(country string) []string {
	var env []string
	if tz := TimezoneFor(country); tz != "" {
		env = append(env, "TZ="+tz)
	}
	if loc := LocaleFor(country); loc != "" {
		env = append(env, "LANG="+loc, "LC_ALL="+loc)
	}
	return env
}

type mimErr string

func (e mimErr) Error() string { return string(e) }

const (
	errUnknownCountry mimErr = "mimicry: no timezone mapping for country"
	errNoRunner       mimErr = "mimicry: no runner to apply system timezone"
)
