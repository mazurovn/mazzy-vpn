// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package mlog

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l, _ := New(Options{Console: &buf, MinLevel: WARN})
	l.Info("test.info", "should be hidden")
	l.Warn("test.warn", "should show")
	out := buf.String()
	if strings.Contains(out, "should be hidden") {
		t.Error("INFO below WARN threshold should be filtered")
	}
	if !strings.Contains(out, "should show") {
		t.Error("WARN should be shown")
	}
}

func TestFieldsRendered(t *testing.T) {
	var buf bytes.Buffer
	l, _ := New(Options{Console: &buf, MinLevel: DEBUG})
	l.Info("connect.up", "connected", F("zone", "NL", "rtt", 68))
	out := buf.String()
	if !strings.Contains(out, "zone=NL") || !strings.Contains(out, "rtt=68") {
		t.Errorf("fields not rendered: %q", out)
	}
}

func TestJSONSink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.jsonl")
	l, err := New(Options{Console: &bytes.Buffer{}, MinLevel: INFO, JSONPath: path})
	if err != nil {
		t.Fatal(err)
	}
	l.Info("health.ok", "protected", F("egress", "203.0.113.9"))
	l.Error("connect.fail", "no egress")
	l.Close()

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 json lines, got %d", len(lines))
	}
	var e Entry
	if err := json.Unmarshal([]byte(lines[0]), &e); err != nil {
		t.Fatal(err)
	}
	if e.Event != "health.ok" || e.Fields["egress"] != "203.0.113.9" {
		t.Errorf("json entry wrong: %+v", e)
	}
}

func TestFHelper(t *testing.T) {
	m := F("a", 1, "b", "x")
	if m["a"] != 1 || m["b"] != "x" {
		t.Errorf("F() wrong: %+v", m)
	}
}
