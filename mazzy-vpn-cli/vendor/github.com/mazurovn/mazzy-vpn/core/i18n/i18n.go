// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package i18n provides message localization for mazzy-core UIs (CLI, TUI,
// Desktop). It supports the same six languages as the legacy bash CLI
// (ru/en/de/zh/ja/ko) and resolves a language from an explicit override, a
// locale string, or a fallback.
//
// Messages are keyed by stable identifiers so all products share one catalog
// and the machine-first API can return message_key values that map here.
package i18n

import "strings"

// Lang is a supported UI language code.
type Lang string

const (
	RU Lang = "ru"
	EN Lang = "en"
	DE Lang = "de"
	ZH Lang = "zh"
	JA Lang = "ja"
	KO Lang = "ko"
)

// DefaultLang is used when no language can be resolved. Unlike the legacy bash
// CLI (which defaulted to Russian), mazzy-core defaults to English as an
// international-first product choice; the user's locale still takes precedence
// via Resolve, so this only affects hosts with no detectable locale.
const DefaultLang = EN

// Supported lists all languages in a stable order (for menus).
var Supported = []Lang{RU, EN, DE, ZH, JA, KO}

// nativeNames maps a language to its endonym.
var nativeNames = map[Lang]string{
	RU: "Русский", EN: "English", DE: "Deutsch",
	ZH: "中文", JA: "日本語", KO: "한국어",
}

// NativeName returns the language's own name, or the code if unknown.
func (l Lang) NativeName() string {
	if n, ok := nativeNames[l]; ok {
		return n
	}
	return string(l)
}

// Valid reports whether l is a supported language.
func (l Lang) Valid() bool {
	_, ok := nativeNames[l]
	return ok
}

// aliases maps normalized locale prefixes/aliases to a Lang.
var aliases = map[string]Lang{
	"ru": RU, "rus": RU, "russian": RU, "русский": RU,
	"en": EN, "eng": EN, "english": EN,
	"de": DE, "deu": DE, "ger": DE, "german": DE, "germany": DE, "deutsch": DE,
	"zh": ZH, "zho": ZH, "chi": ZH, "chinese": ZH, "china": ZH, "中文": ZH,
	"ja": JA, "jpn": JA, "japanese": JA, "japan": JA, "日本語": JA,
	"ko": KO, "kor": KO, "korean": KO, "korea": KO, "한국어": KO,
}

// Normalize resolves a raw language/locale string (e.g. "de_DE.UTF-8") to a
// supported Lang. It returns ok=false when the input maps to nothing.
func Normalize(raw string) (Lang, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	// Strip encoding and region: "de_DE.UTF-8" -> "de".
	for _, sep := range []string{".", "_", "-"} {
		if i := strings.Index(s, sep); i >= 0 {
			s = s[:i]
		}
	}
	if s == "" {
		return "", false
	}
	if l, ok := aliases[s]; ok {
		return l, true
	}
	return "", false
}

// Resolve picks a language from an explicit override first, then a list of
// candidate locale strings, falling back to DefaultLang.
func Resolve(override string, candidates ...string) Lang {
	if l, ok := Normalize(override); ok {
		return l
	}
	for _, c := range candidates {
		if l, ok := Normalize(c); ok {
			return l
		}
	}
	return DefaultLang
}
