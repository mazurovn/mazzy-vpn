// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mazurovn/mazzy-vpn/core/catalog"
)

// newCatalog builds the per-user profile catalog.
func newCatalog() *catalog.Catalog {
	return catalog.New(catalog.DefaultDir())
}

// cmdImport imports one or more profile files or a whole directory.
func cmdImport(_ context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mazzy-vpn import <FILE|DIR>...")
		return 2
	}
	cat := newCatalog()
	imported, failed := 0, 0
	for _, path := range args {
		if strings.HasPrefix(path, "-") {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", path, err)
			failed++
			continue
		}
		var files []string
		if info.IsDir() {
			entries, _ := os.ReadDir(path)
			for _, e := range entries {
				ext := strings.ToLower(filepath.Ext(e.Name()))
				if ext == ".conf" || ext == ".ovpn" {
					files = append(files, filepath.Join(path, e.Name()))
				}
			}
		} else {
			files = []string{path}
		}
		for _, f := range files {
			e, err := cat.Import(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "skip %s: %v\n", safeDisplay(filepath.Base(f)), err)
				failed++
				continue
			}
			cc := ""
			if e.Country != "" {
				cc = " [" + safeDisplay(e.Country) + "]"
			}
			fmt.Printf("imported %s%s (%s)\n", safeDisplay(e.Name), cc, safeDisplay(string(e.Protocol)))
			imported++
		}
	}
	fmt.Println(translator().Tf("cli.catalog.imported", imported, failed))
	if imported == 0 {
		return 1
	}
	return 0
}

// cmdProfiles lists managed profiles. With --ping it also probes and shows the
// live latency and reachability for each server.
func cmdProfiles(ctx context.Context, args []string) int {
	cat := newCatalog()
	entries, err := cat.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, "catalog error:", err)
		return 1
	}
	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(entries)
		return 0
	}
	if len(entries) == 0 {
		fmt.Println(translator().T("cli.catalog.none"))
		return 0
	}

	withPing := hasFlag(args, "--ping")
	var pings map[string]string
	if withPing {
		fmt.Println(translator().T("cli.catalog.measuring"))
		pings = measureCatalogPings(ctx, cat)
	}

	if withPing {
		fmt.Printf("%-3s %-24s %-10s %-4s %-8s %s\n", "", "NAME", "PROTOCOL", "CC", "PING", "FAV")
	} else {
		fmt.Printf("%-3s %-24s %-10s %-4s %s\n", "", "NAME", "PROTOCOL", "CC", "FAV")
	}
	for i, e := range entries {
		star := ""
		if e.Favorite {
			star = "★"
		}
		if withPing {
			ping := pings[e.Name]
			if ping == "" {
				ping = "n/a"
			}
			fmt.Printf("%-3d %-24s %-10s %-4s %-8s %s\n", i+1, safeDisplay(e.Name), safeDisplay(string(e.Protocol)), safeDisplay(e.Country), safeDisplay(ping), star)
		} else {
			fmt.Printf("%-3d %-24s %-10s %-4s %s\n", i+1, safeDisplay(e.Name), safeDisplay(string(e.Protocol)), safeDisplay(e.Country), star)
		}
	}
	return 0
}

// cmdFavorite toggles a favorite.
func cmdFavorite(_ context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mazzy-vpn favorite NAME [--off]")
		return 2
	}
	cat := newCatalog()
	fav := !hasFlag(args, "--off")
	// Accept `favorite NAME [--off]` and `favorite --off NAME`: the name is the
	// first non-flag argument, not blindly args[0] (which could be "--off").
	name := firstNonFlag(args)
	if name == "" {
		fmt.Fprintln(os.Stderr, "usage: mazzy-vpn favorite NAME [--off]")
		return 2
	}
	if err := cat.SetFavorite(name, fav); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	state := "added to"
	if !fav {
		state = "removed from"
	}
	fmt.Printf("%s %s favorites\n", safeDisplay(name), state)
	return 0
}

// cmdRemove deletes a managed profile.
func cmdRemove(_ context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mazzy-vpn remove NAME")
		return 2
	}
	cat := newCatalog()
	name := firstNonFlag(args)
	if name == "" {
		fmt.Fprintln(os.Stderr, "usage: mazzy-vpn remove NAME")
		return 2
	}
	if err := cat.Remove(name); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(translator().Tf("cli.catalog.removed", safeDisplay(name)))
	return 0
}
