// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/catalog"
)

// newCatalog builds the per-user profile catalog.
func newCatalog() *catalog.Catalog {
	return catalog.New(catalog.DefaultDir())
}

// collectProfileFiles walks a file or directory (recursively) and returns every
// recognized profile path. Recursion matters because real provider bundles ship
// nested folders (e.g. amnezia/, openvpn/, by-country/). Non-profile files are
// ignored so a user can point import at a messy download dir.
func collectProfileFiles(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{root}, nil
	}
	var files []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries, keep walking
		}
		if d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(d.Name())) {
		case ".conf", ".ovpn":
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// importStats tallies an import pass by protocol so the user gets a clear
// distribution summary ("the system sorted them for you").
type importStats struct {
	amnezia, wireguard, openvpn int
	imported, failed, skipped   int
}

// cmdImport imports one or more profile files or whole directories. It scans
// recursively, classifies each profile by protocol (AmneziaWG / WireGuard /
// OpenVPN) and, when the SAME server ships as both an OpenVPN and a
// WireGuard/AmneziaWG file, prefers the connectable WG/AWG variant so the
// catalog is not doubled with unusable OpenVPN twins. A per-protocol summary is
// printed at the end.
func cmdImport(_ context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mazzy-vpn import <FILE|DIR>...")
		return 2
	}
	cat := newCatalog()
	t := translator()

	// Gather every candidate file across all args first, so we can make a global
	// WG-over-OVPN preference decision per logical server (by name stem).
	var files []string
	st := importStats{}
	for _, path := range args {
		if strings.HasPrefix(path, "-") {
			continue
		}
		found, err := collectProfileFiles(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", safeDisplay(path), err)
			st.failed++
			continue
		}
		files = append(files, found...)
	}

	// Preference pass: if a server name has a WG/AWG (.conf) file, drop its
	// OpenVPN (.ovpn) twin so the user ends up with the connectable variant.
	stems := map[string]bool{} // stem -> has a .conf (WG/AWG)
	for _, f := range files {
		if strings.EqualFold(filepath.Ext(f), ".conf") {
			stems[importStem(f)] = true
		}
	}
	for _, f := range files {
		name := safeDisplay(filepath.Base(f))
		if strings.EqualFold(filepath.Ext(f), ".ovpn") && stems[importStem(f)] {
			st.skipped++
			continue // a connectable WG/AWG twin exists; skip the OpenVPN copy
		}
		e, err := cat.Import(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", name, err)
			st.failed++
			continue
		}
		st.imported++
		switch e.Protocol {
		case core.AmneziaWG:
			st.amnezia++
		case core.WireGuard:
			st.wireguard++
		case core.OpenVPN:
			st.openvpn++
		}
		cc := ""
		if e.Country != "" {
			cc = " [" + safeDisplay(e.Country) + "]"
		}
		fmt.Printf("  ✓ %s%s → %s\n", safeDisplay(e.Name), cc, safeDisplay(string(e.Protocol)))
	}

	// Distribution summary: what the system sorted the bundle into.
	fmt.Printf("\n%s\n", t.Tf("cli.catalog.imported", st.imported, st.failed))
	fmt.Printf("  sorted: AmneziaWG %d · WireGuard %d · OpenVPN %d", st.amnezia, st.wireguard, st.openvpn)
	if st.skipped > 0 {
		fmt.Printf(" · skipped %d OpenVPN twin(s) with a WireGuard variant", st.skipped)
	}
	fmt.Println()
	if st.openvpn > 0 {
		fmt.Println("  note: OpenVPN profiles are catalogued but the embedded engine cannot")
		fmt.Println("        connect them yet; import a WireGuard/AmneziaWG variant to use them.")
	}
	if st.imported == 0 {
		return 1
	}
	return 0
}

// importStem returns a normalized server key from a profile path: the lowercased
// file stem with common separators collapsed, so "Austria--Vienna-S1.ovpn" and
// "AustriaViennaS1.conf" map to the same logical server for WG-over-OVPN dedup.
func importStem(path string) string {
	base := filepath.Base(path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	stem = strings.ToLower(stem)
	replacer := strings.NewReplacer("-", "", "_", "", " ", "", ".", "")
	return replacer.Replace(stem)
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
