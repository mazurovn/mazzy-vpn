// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mazurovn/mazzy-vpn/core/api"
	"github.com/mazurovn/mazzy-vpn/core/doctor"
	"github.com/mazurovn/mazzy-vpn/core/i18n"
	"github.com/mazurovn/mazzy-vpn/core/lock"
	"github.com/mazurovn/mazzy-vpn/core/tui"
)

// TestScenarioCLIComposition wires the leaf packages the CLI binary will use
// (api + doctor + lock + tui + i18n) so their composition is exercised before
// the Phase 3 binary exists. It proves none of them are dead-in-practice.
func TestScenarioCLIComposition(t *testing.T) {
	ctx := context.Background()

	// 1. A mutation must hold the single-flight lock first.
	dir := t.TempDir()
	held, err := lock.Acquire(dir)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	// Contending acquire fails fast while held (parity with acquire_action_lock).
	if _, err := lock.Acquire(dir); err != lock.ErrBusy {
		t.Fatalf("expected ErrBusy while lock held, got %v", err)
	}

	// 2. doctor.Run feeds a status.get handler as a JSON result.
	router := api.NewRouter(func() string { return "req-int" })
	router.Handle("doctor.run", func(ctx context.Context, _ *api.Request) (any, error) {
		return doctor.Run(ctx, fakeHealthyEnv{}), nil
	})

	env := router.Dispatch(ctx, []byte(`{"api_version":"1.0","operation":"doctor.run"}`))
	var got map[string]any
	if err := json.Unmarshal(env.Marshal(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["status"] != "ok" {
		t.Fatalf("doctor.run via api should be ok, got %v", got)
	}
	result := got["result"].(map[string]any)
	if result["fail"].(float64) != 0 {
		t.Errorf("healthy doctor env should have zero fails: %v", result)
	}

	// 3. Release the lock and confirm re-acquire works (no leak).
	if err := held.Unlock(); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	reacq, err := lock.Acquire(dir)
	if err != nil {
		t.Fatalf("re-acquire after unlock: %v", err)
	}
	reacq.Unlock()

	// 4. The TUI menu localizes and maps a selection to an action.
	menu := tui.NewMenu(i18n.RU)
	if len(menu.Lines()) != menu.Len() || menu.Len() == 0 {
		t.Fatal("menu produced no lines")
	}
	if act, ok := menu.Select(1); !ok || act != tui.ActConnect {
		t.Errorf("menu Select(1) = %q,%v; want connect", act, ok)
	}
}

// fakeHealthyEnv is a doctor.Environment with everything present.
type fakeHealthyEnv struct{}

func (fakeHealthyEnv) LookPath(bin string) bool {
	switch bin {
	case "ip", "nft", "resolvectl":
		return true
	}
	return false
}
func (fakeHealthyEnv) FileExists(path string) bool { return path == "/dev/net/tun" }
func (fakeHealthyEnv) IsRoot() bool                { return true }

// Compile-time: doctor.Environment is satisfied by our fake.
var _ doctor.Environment = fakeHealthyEnv{}
