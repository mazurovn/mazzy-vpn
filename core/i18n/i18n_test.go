// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package i18n

import "testing"

func TestNormalizeLocales(t *testing.T) {
	cases := map[string]Lang{
		"ru": RU, "RU": RU, "russian": RU,
		"de_DE.UTF-8": DE, "de-DE": DE, "deutsch": DE,
		"en_US.UTF-8": EN, "english": EN,
		"zh_CN": ZH, "中文": ZH,
		"ja_JP.UTF-8": JA, "ko_KR": KO,
	}
	for in, want := range cases {
		got, ok := Normalize(in)
		if !ok || got != want {
			t.Errorf("Normalize(%q) = %q,%v; want %q", in, got, ok, want)
		}
	}
}

func TestNormalizeUnknown(t *testing.T) {
	for _, in := range []string{"", "xx", "klingon", "  "} {
		if _, ok := Normalize(in); ok {
			t.Errorf("Normalize(%q) should fail", in)
		}
	}
}

func TestResolvePrecedence(t *testing.T) {
	// Override wins.
	if got := Resolve("de", "ru", "en"); got != DE {
		t.Errorf("override should win, got %q", got)
	}
	// Falls through candidates.
	if got := Resolve("bogus", "nonsense", "ja_JP"); got != JA {
		t.Errorf("candidate fallthrough failed, got %q", got)
	}
	// Ultimate fallback.
	if got := Resolve("", "also-bad"); got != DefaultLang {
		t.Errorf("expected default, got %q", got)
	}
}

func TestTranslatorFallback(t *testing.T) {
	tr := NewTranslator(RU)
	if tr.T("menu.connect") != "Подключить" {
		t.Errorf("ru translation wrong: %q", tr.T("menu.connect"))
	}
	// Unknown key returns the key itself.
	if tr.T("no.such.key") != "no.such.key" {
		t.Errorf("unknown key should echo itself")
	}
	// Invalid language falls back to default.
	tr2 := NewTranslator(Lang("xx"))
	if tr2.Lang != DefaultLang {
		t.Errorf("invalid lang should fall back to default")
	}
}

// TestCatalogCompleteness is the anti-"ущербность" gate: EVERY catalog key
// must have a non-empty translation for EVERY supported language. A missing
// entry would silently fall back and degrade UX.
func TestCatalogCompleteness(t *testing.T) {
	for key, entries := range catalog {
		for _, lang := range Supported {
			s, ok := entries[lang]
			if !ok || s == "" {
				t.Errorf("catalog key %q missing translation for %q", key, lang)
			}
		}
	}
}

func TestNativeNames(t *testing.T) {
	for _, l := range Supported {
		if l.NativeName() == string(l) {
			t.Errorf("language %q has no native name", l)
		}
	}
}

func TestSupportedCount(t *testing.T) {
	if len(Supported) != 6 {
		t.Fatalf("expected 6 languages, got %d", len(Supported))
	}
}

// TestVerifyKeysLocalized locks catalog/verify sync (audit I3): every verdict
// message key emitted by core/verify must have a full localized entry, so
// agents and UIs never see a raw key.
func TestVerifyKeysLocalized(t *testing.T) {
	for _, key := range verifyMessageKeys {
		if !HasKey(key) {
			t.Errorf("verify message key %q missing from catalog", key)
			continue
		}
		for _, lang := range Supported {
			if s := (&Translator{Lang: lang}).T(key); s == key {
				t.Errorf("verify key %q not localized for %q", key, lang)
			}
		}
	}
}
