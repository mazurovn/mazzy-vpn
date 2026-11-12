// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package api

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMissingSocketStaysUnavailable(t *testing.T) {
	c := Client{Path: filepath.Join(t.TempDir(), "api-v1.sock")}
	out := c.Disconnect(context.Background())
	if out.Kind != KindUnavailable {
		t.Fatalf("kind %v line %q", out.Kind, out.Line)
	}
	if !strings.Contains(out.Line, "сокет") {
		t.Fatalf("line %q", out.Line)
	}
	if strings.Contains(out.Line, "Not authorized") {
		t.Fatal(out.Line)
	}
}

func TestPermissionDeniedIsTheGroup(t *testing.T) {
	c := Client{Dial: func(context.Context, string, string) (net.Conn, error) {
		return nil, &net.OpError{Op: "dial", Err: errDenied{}}
	}}
	out := c.Connect(context.Background(), "profile-0123456789abcdef0123456789abcdef")
	if out.Kind != KindUnavailable || !strings.Contains(out.Line, "mazzy-vpn") {
		t.Fatalf("%+v", out)
	}
}

type errDenied struct{}

func (errDenied) Error() string   { return "permission denied" }
func (errDenied) Timeout() bool   { return false }
func (errDenied) Temporary() bool { return false }

func TestEngineFailureIsNotAPrivilegeError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api-v1.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go serveOnce(ln, "failed")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out := Client{Path: path}.Connect(ctx, ProfileID("wireguard", "helsinki.conf"))
	if out.Kind != KindFailed {
		t.Fatalf("%+v", out)
	}
	if strings.Contains(out.Line, "Not authorized") || strings.Contains(out.Line, "pkexec") {
		t.Fatal(out.Line)
	}
}

func TestAcceptedConnect(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api-v1.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go serveOnce(ln, "succeeded")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out := Client{Path: path}.Reconnect(ctx)
	if out.Kind != KindAccepted {
		t.Fatalf("%+v", out)
	}
}

func TestProfileIDKeepsAStablePrefix(t *testing.T) {
	id := ProfileID("wireguard", "helsinki.conf")
	if !strings.HasPrefix(id, "profile-") || len(id) != len("profile-")+32 {
		t.Fatal(id)
	}
	if id != ProfileID("wireguard", "helsinki.conf") {
		t.Fatal("unstable")
	}
}

func TestScrubDropsPolkit(t *testing.T) {
	got := Scrub("Error executing command as another user: Not authorized\nThis incident has been reported.")
	if strings.Contains(got, "Not authorized") || strings.Contains(got, "incident") {
		t.Fatal(got)
	}
}

func serveOnce(ln net.Listener, state string) {
	conn, err := ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	buf := make([]byte, 8192)
	n, _ := conn.Read(buf)
	var req struct {
		RequestID string `json:"request_id"`
		ActionID  string `json:"action_id"`
	}
	_ = json.Unmarshal(buf[:n], &req)
	resp := map[string]any{
		"api_version": "1.0",
		"request_id":  req.RequestID,
		"status":      "ok",
		"result": map[string]any{
			"action_id":     req.ActionID,
			"state":         state,
			"state_changed": state == "succeeded",
			"rollback":      map[string]any{"required": false, "state": "not-needed"},
		},
	}
	raw, _ := json.Marshal(resp)
	_, _ = io.WriteString(conn, string(raw)+"\n")
}
