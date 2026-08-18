// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mazurovn/mazzy-vpn/core/i18n"
	"github.com/mazurovn/mazzy-vpn/core/settings"
)

// resolveLang picks the UI language WITHOUT hardcoding: saved setting first,
// then MAZZY_LANG / LC_ALL / LC_MESSAGES / LANG environment, else the i18n
// default (English). Returns the resolved language.
func resolveLang() i18n.Lang {
	saved := settings.NewStore().Load().Language
	return i18n.Resolve(saved,
		os.Getenv("MAZZY_LANG"),
		os.Getenv("LC_ALL"),
		os.Getenv("LC_MESSAGES"),
		os.Getenv("LANG"),
	)
}

// translator returns a Translator for the currently resolved language.
func translator() *i18n.Translator { return i18n.NewTranslator(resolveLang()) }

// cmdLanguage shows or sets the UI language.
//
//	mazzy-vpn language           interactive selection menu
//	mazzy-vpn language <code>    set directly (en/ru/de/zh/ja/ko)
//	mazzy-vpn language --list    print supported languages
func cmdLanguage(_ context.Context, args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "--list", "list":
			return languageList()
		default:
			return languageSet(args[0])
		}
	}
	return languageMenu()
}

// languageList prints supported languages and marks the current one.
func languageList() int {
	cur := resolveLang()
	for _, l := range i18n.Supported {
		mark := "  "
		if l == cur {
			mark = "* "
		}
		fmt.Printf("%s%-3s %s\n", mark, l, l.NativeName())
	}
	return 0
}

// languageSet persists an explicit language code.
func languageSet(code string) int {
	lang, ok := i18n.Normalize(code)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown language %q; try: mazzy-vpn language --list\n", code)
		return 1
	}
	st := settings.NewStore()
	cur := st.Load()
	cur.Language = string(lang)
	if err := st.Save(cur); err != nil {
		fmt.Fprintln(os.Stderr, "could not save language:", err)
		return 1
	}
	t := i18n.NewTranslator(lang)
	fmt.Printf("%s → %s (%s)\n", t.T("menu.language"), lang.NativeName(), lang)
	return 0
}

// languageMenu presents an interactive numbered selection of languages.
func languageMenu() int {
	cur := resolveLang()
	t := i18n.NewTranslator(cur)
	fmt.Printf("%s:\n", t.T("menu.language"))
	for i, l := range i18n.Supported {
		mark := " "
		if l == cur {
			mark = "*"
		}
		fmt.Printf("  %d) %s %-8s (%s)\n", i+1, mark, l.NativeName(), l)
	}
	fmt.Print(t.T("prompt.choose"))
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	choice := strings.TrimSpace(line)
	if choice == "" {
		return 0 // no change
	}
	n, err := strconv.Atoi(choice)
	if err != nil || n < 1 || n > len(i18n.Supported) {
		fmt.Fprintln(os.Stderr, "invalid selection")
		return 1
	}
	return languageSet(string(i18n.Supported[n-1]))
}
