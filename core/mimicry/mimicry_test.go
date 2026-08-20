// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package mimicry

import (
	"strings"
	"testing"
)

type fakeRunner struct {
	calls []string
	err   error
}

func (f *fakeRunner) Run(bin string, args ...string) (string, error) {
	f.calls = append(f.calls, bin+" "+strings.Join(args, " "))
	return "", f.err
}

func TestTimezoneFor(t *testing.T) {
	cases := map[string]string{"NL": "Europe/Amsterdam", "US": "America/New_York", "de": "Europe/Berlin", "XX": ""}
	for cc, want := range cases {
		if got := TimezoneFor(cc); got != want {
			t.Errorf("TimezoneFor(%q) = %q, want %q", cc, got, want)
		}
	}
}

func TestPlanDetectsMismatch(t *testing.T) {
	m := &Manager{CurrentTZ: func() string { return "Europe/Moscow" }}
	plan, ok := m.PlanFor("NL")
	if !ok {
		t.Fatal("NL should be known")
	}
	if !plan.NeedsChange || plan.FromTZ != "Europe/Moscow" || plan.ToTZ != "Europe/Amsterdam" {
		t.Errorf("bad plan: %+v", plan)
	}
}

func TestPlanNoChangeWhenAligned(t *testing.T) {
	m := &Manager{CurrentTZ: func() string { return "Europe/Amsterdam" }}
	plan, _ := m.PlanFor("NL")
	if plan.NeedsChange {
		t.Error("aligned timezone should need no change")
	}
}

func TestApplySystemTZ(t *testing.T) {
	fr := &fakeRunner{}
	m := &Manager{Runner: fr, CurrentTZ: func() string { return "Europe/Moscow" }}
	if _, err := m.ApplySystemTZ("NL"); err != nil {
		t.Fatal(err)
	}
	if len(fr.calls) != 1 || !strings.Contains(fr.calls[0], "set-timezone Europe/Amsterdam") {
		t.Errorf("expected timedatectl set-timezone, got %v", fr.calls)
	}
}

func TestApplyUnknownCountry(t *testing.T) {
	m := &Manager{Runner: &fakeRunner{}, CurrentTZ: func() string { return "UTC" }}
	if _, err := m.ApplySystemTZ("XX"); err == nil {
		t.Error("unknown country should error")
	}
}

func TestProcessEnv(t *testing.T) {
	m := &Manager{}
	env := m.ProcessEnv("NL")
	joined := strings.Join(env, " ")
	if !strings.Contains(joined, "TZ=Europe/Amsterdam") {
		t.Errorf("process env missing TZ: %v", env)
	}
	if !strings.Contains(joined, "LANG=nl_NL.UTF-8") {
		t.Errorf("process env missing LANG: %v", env)
	}
}

func TestCountryForTimezone(t *testing.T) {
	if CountryForTimezone("Europe/Amsterdam") != "NL" {
		t.Error("reverse map NL failed")
	}
	if CountryForTimezone("Nowhere/Nope") != "" {
		t.Error("unknown tz should be empty")
	}
}
