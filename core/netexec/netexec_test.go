// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package netexec

import (
	"context"
	"testing"
)

func TestValidateArgvRejectsControlChars(t *testing.T) {
	r := ExecRunner{}
	if _, err := r.Run(context.Background(), "ip", "link\nadd"); err == nil {
		t.Fatal("expected rejection of newline in argument")
	}
	if _, err := r.Run(context.Background(), "ip", "a\x00b"); err == nil {
		t.Fatal("expected rejection of NUL in argument")
	}
}

func TestValidateArgvRejectsEmptyBinary(t *testing.T) {
	r := ExecRunner{}
	if _, err := r.Run(context.Background(), "  "); err == nil {
		t.Fatal("expected rejection of empty binary")
	}
}

// TestRunExecutesRealCommand exercises the real ExecRunner against harmless
// coreutils so the exec path (not just validation) is covered.
func TestRunExecutesRealCommand(t *testing.T) {
	if !Available("printf") && !Available("echo") {
		t.Skip("no echo/printf available")
	}
	r := ExecRunner{}
	out, err := r.Run(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatalf("echo failed: %v", err)
	}
	if out != "hello\n" {
		t.Errorf("echo output = %q, want \"hello\\n\"", out)
	}
}

// TestRunSurfacesNonZeroExit confirms a failing command becomes an error with
// stderr folded in.
func TestRunSurfacesNonZeroExit(t *testing.T) {
	if !Available("false") {
		t.Skip("no false available")
	}
	r := ExecRunner{}
	if _, err := r.Run(context.Background(), "false"); err == nil {
		t.Fatal("expected error from `false`")
	}
}

func TestAvailable(t *testing.T) {
	if !Available("sh") {
		t.Error("sh should be available on a POSIX host")
	}
	if Available("definitely-not-a-real-binary-xyz") {
		t.Error("nonexistent binary must not be reported available")
	}
}

func TestRunInputPipesStdin(t *testing.T) {
	if !Available("cat") {
		t.Skip("cat unavailable")
	}
	r := ExecRunner{}
	out, err := r.RunInput(context.Background(), "hello-stdin", "cat")
	if err != nil {
		t.Fatalf("cat: %v", err)
	}
	if out != "hello-stdin" {
		t.Errorf("stdin not piped: got %q", out)
	}
}
