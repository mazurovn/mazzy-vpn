// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func load(configPath, trustBin, toeacBin string) (flat, error) {
	if out, err := exec.Command(trustBin, "trust-config", configPath).CombinedOutput(); err != nil {
		_ = out
		return flat{}, errRefused
	}
	cmd := exec.Command(toeacBin, configPath)
	out, err := cmd.Output()
	if err != nil {
		return flat{}, errRefused
	}
	return parseFlat(string(out))
}

func main() {
	configPath := flag.String("config", "/etc/mazzy-vpn/hop.toeac", "toeac program")
	trustBin := flag.String("trust", "mazzy-core", "core binary used for the file trust check")
	toeacBin := flag.String("toeac", "mazzy-toeac", "toeac language binary")
	flag.Parse()
	cfg, err := load(*configPath, *trustBin, *toeacBin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "backend: refused")
		os.Exit(78)
	}
	srv := &http.Server{
		Handler:           newMux(cfg.statusSocket, cfg.adminCredential),
		ReadHeaderTimeout: 5 * time.Second,
	}
	ln, err := net.Listen("tcp", cfg.adminBind)
	if err != nil {
		fmt.Fprintln(os.Stderr, "backend: refused")
		os.Exit(1)
	}
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "backend: refused")
		os.Exit(1)
	}
}
