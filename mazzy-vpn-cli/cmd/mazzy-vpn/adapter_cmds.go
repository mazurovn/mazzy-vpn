// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mazurovn/mazzy-vpn/core/netadapter"
	"github.com/mazurovn/mazzy-vpn/core/netdiag"
)

// cmdAdapters lists host network interfaces and marks the recommended uplink.
func cmdAdapters(_ context.Context, args []string) int {
	adapters, err := netadapter.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot list adapters:", err)
		return 1
	}
	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(adapters)
		return 0
	}
	rec, _, hasRec := netadapter.Recommend(adapters)
	fmt.Printf("%-14s %-8s %-6s %-6s %s\n", "NAME", "KIND", "UP", "REC", "IPv4")
	for _, a := range adapters {
		up := "no"
		if a.Up {
			up = "yes"
		}
		star := ""
		if hasRec && a.Name == rec.Name {
			star = "★"
		}
		ip := "-"
		if len(a.IPv4) > 0 {
			ip = a.IPv4[0]
		}
		fmt.Printf("%-14s %-8s %-6s %-6s %s\n", a.Name, a.Kind(), up, star, ip)
	}
	if hasRec {
		fmt.Printf("\nRecommended uplink: %s\n", rec.Name)
	}
	return 0
}

// cmdNetdiag analyzes the network situation and prints findings + fixes.
func cmdNetdiag(_ context.Context, args []string) int {
	adapters, err := netadapter.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot list adapters:", err)
		return 1
	}
	rep := netdiag.Analyze(adapters)
	if hasFlag(args, "--json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rep)
		if !rep.Healthy() {
			return 1
		}
		return 0
	}
	for _, f := range rep.Findings {
		fmt.Printf("[%-4s] %s\n", f.Level, f.Title)
		if f.Detail != "" {
			fmt.Printf("        %s\n", f.Detail)
		}
		if f.Fix != "" {
			fmt.Printf("        fix: %s\n", f.Fix)
		}
	}
	fmt.Printf("\nSummary: OK=%d WARN=%d FAIL=%d\n", rep.OK, rep.Warn, rep.Fail)
	if rep.RecommendedUplink != "" {
		fmt.Printf("Recommended uplink: %s\n", rep.RecommendedUplink)
	}
	if !rep.Healthy() {
		return 1
	}
	return 0
}
