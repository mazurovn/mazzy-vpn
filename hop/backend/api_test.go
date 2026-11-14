// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package main

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func flatProgram(adminBind, cred, status string) string {
	return strings.Join([]string{
		"ok",
		"transport nested-https",
		"bind 127.0.0.1:1",
		"cert /etc/mazzy-vpn/cert",
		"key /etc/mazzy-vpn/key",
		"idle 30",
		"max 4",
		"doctor 60",
		"claim-clean no",
		"require-auth yes",
		"credential hop-credential-ok",
		"allow 127.0.0.1:9",
		"mcp-socket /run/mazzy-vpn/mcp.sock",
		"admin-bind " + adminBind,
		"admin-credential " + cred,
		"status-socket " + status,
		"a2a-bind 127.0.0.1:3",
		"admin-url http://127.0.0.1:2",
		"artifact mazzy-core mcp",
		"artifact mazzy-backend admin",
		"artifact mazzy-front a2a",
		"logic relay copy",
		"logic protect mojo",
		"logic doctor nim",
		"",
	}, "\n")
}

func TestExampleIsPlaceholder(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "example.toeac"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if !strings.Contains(text, "REPLACE_") {
		t.Fatal("example lost its placeholders")
	}
	if strings.Contains(text, "BEGIN ") {
		t.Fatal("example contains key material")
	}
}

func TestParseFlat(t *testing.T) {
	got, err := parseFlat(flatProgram("127.0.0.1:2", "admin-credential-ok", "/run/mazzy-vpn/status.sock"))
	if err != nil {
		t.Fatal(err)
	}
	if got.adminBind != "127.0.0.1:2" || got.statusSocket != "/run/mazzy-vpn/status.sock" {
		t.Fatalf("parsed %#v", got)
	}
	if _, err := parseFlat(strings.ReplaceAll(flatProgram("127.0.0.1:2", "admin-credential-ok", "/run/mazzy-vpn/status.sock"), "claim-clean no", "claim-clean yes")); err == nil {
		t.Fatal("claim-clean yes was accepted")
	}
	if _, err := parseFlat(flatProgram("REPLACE_ADMIN_BIND", "admin-credential-ok", "/run/mazzy-vpn/status.sock")); err != errPlaceholder {
		t.Fatalf("placeholder err %v", err)
	}
	wrong := strings.Replace(flatProgram("127.0.0.1:2", "admin-credential-ok", "/run/mazzy-vpn/status.sock"), "artifact mazzy-core mcp", "artifact other mcp", 1)
	if _, err := parseFlat(wrong); err == nil {
		t.Fatal("foreign owner was accepted")
	}
}

func TestAdminForcesDirtyEgress(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "status.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_, _ = bufio.NewReader(c).ReadString('\n')
			_, _ = io.WriteString(c, `{"egress_clean":true,"config":"trusted","protection":"on","sessions":2,"transport":"nested-https","tunnel":"independent"}`+"\n")
			_ = c.Close()
		}
	}()

	h := newMux(sock, "admin-credential-ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/doctor", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anon %d", rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/doctor", nil)
	req.Header.Set("Authorization", "Bearer admin-credential-ok")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("doctor %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"egress_clean":true`) || !strings.Contains(rec.Body.String(), `"egress_clean":false`) {
		t.Fatalf("egress claim %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "hop-credential") || strings.Contains(rec.Body.String(), "admin-credential-ok") {
		t.Fatal("credential leaked")
	}

	post := httptest.NewRequest(http.MethodPost, "/v1/health", nil)
	post.Header.Set("Authorization", "Bearer admin-credential-ok")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, post)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("post %d", rec.Code)
	}
}

func TestAdminUnavailable(t *testing.T) {
	h := newMux(filepath.Join(t.TempDir(), "missing.sock"), "admin-credential-ok")
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.Header.Set("Authorization", "Bearer admin-credential-ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"egress_clean":false`) {
		t.Fatalf("body %s", rec.Body.String())
	}
}
