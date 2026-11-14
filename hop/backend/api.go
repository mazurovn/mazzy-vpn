// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// The Go backend owns the admin API. It does not relay bytes and it does
// not claim the egress is clean.
package main

import (
	"bufio"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	errRefused     = errors.New("refused")
	errPlaceholder = errors.New("placeholder")
)

type status struct {
	EgressClean bool   `json:"egress_clean"`
	Config      string `json:"config"`
	Protection  string `json:"protection"`
	Sessions    int    `json:"sessions"`
	Transport   string `json:"transport"`
	Tunnel      string `json:"tunnel"`
}

func askCore(socketPath, cmd string) (status, error) {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return status{}, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.WriteString(conn, cmd+"\n"); err != nil {
		return status{}, err
	}
	line, err := bufio.NewReader(io.LimitReader(conn, 4096)).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return status{}, err
	}
	var st status
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &st); err != nil {
		return status{}, err
	}
	st.EgressClean = false
	return st, nil
}

func authorized(r *http.Request, credential string) bool {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return false
	}
	got := h[len(prefix):]
	if len(got) != len(credential) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(credential)) == 1
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

func newMux(socketPath, credential string) http.Handler {
	mux := http.NewServeMux()
	handle := func(cmd string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"egress_clean": false})
				return
			}
			if !authorized(r, credential) {
				writeJSON(w, http.StatusUnauthorized, map[string]any{"egress_clean": false})
				return
			}
			st, err := askCore(socketPath, cmd)
			if err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{
					"egress_clean": false,
					"status":       "unavailable",
				})
				return
			}
			writeJSON(w, http.StatusOK, st)
		}
	}
	mux.HandleFunc("/v1/health", handle("health"))
	mux.HandleFunc("/v1/doctor", handle("doctor"))
	mux.HandleFunc("/v1/sessions", handle("sessions"))
	mux.HandleFunc("/v1/protection", handle("protection"))
	return mux
}
