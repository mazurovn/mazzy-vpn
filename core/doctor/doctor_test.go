// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package doctor

import (
	"context"
	"testing"
)

// fakeEnv scripts host queries.
type fakeEnv struct {
	present map[string]bool
	files   map[string]bool
	root    bool
}

func (f fakeEnv) LookPath(bin string) bool    { return f.present[bin] }
func (f fakeEnv) FileExists(path string) bool { return f.files[path] }
func (f fakeEnv) IsRoot() bool                { return f.root }

func fullyHealthyEnv() fakeEnv {
	return fakeEnv{
		present: map[string]bool{"ip": true, "nft": true, "resolvectl": true},
		files:   map[string]bool{"/dev/net/tun": true},
		root:    true,
	}
}

func TestHealthyEnvironmentHasNoFail(t *testing.T) {
	r := Run(context.Background(), fullyHealthyEnv())
	if !r.Healthy() {
		t.Fatalf("expected healthy, got fails: %+v", r.Checks)
	}
	if r.Fail != 0 {
		t.Errorf("fail=%d, want 0", r.Fail)
	}
}

// TestDoctorNeverRequiresLegacyVPNTools is the autonomy regression: doctor must
// NOT probe for awg/awg-quick/wg/wg-quick/jq/socat/python3. If mazzy-core were
// to reintroduce such a dependency, this test should catch it.
func TestDoctorNeverRequiresLegacyVPNTools(t *testing.T) {
	// An environment WITHOUT any legacy tool but WITH base OS tools must be
	// fully healthy (no FAIL).
	env := fakeEnv{
		present: map[string]bool{"ip": true, "nft": true, "resolvectl": true},
		files:   map[string]bool{"/dev/net/tun": true},
		root:    true,
	}
	r := Run(context.Background(), env)
	if !r.Healthy() {
		t.Fatalf("doctor must be healthy without awg/jq/socat; fails=%+v", r.Checks)
	}
	// And it must explicitly assert the embedded engine.
	found := false
	for _, c := range r.Checks {
		if c.Name == "AmneziaWG engine" && c.Level == OK {
			found = true
		}
	}
	if !found {
		t.Error("expected an explicit 'AmneziaWG engine embedded' OK check")
	}
}

func TestMissingIpAndNftAreFail(t *testing.T) {
	env := fakeEnv{
		present: map[string]bool{"resolvectl": true},
		files:   map[string]bool{"/dev/net/tun": true},
		root:    true,
	}
	r := Run(context.Background(), env)
	if r.Healthy() {
		t.Fatal("missing ip/nft must FAIL")
	}
	if r.Fail < 2 {
		t.Errorf("expected >=2 fails for missing ip and nft, got %d", r.Fail)
	}
}

func TestMissingTunIsFail(t *testing.T) {
	env := fullyHealthyEnv()
	env.files = map[string]bool{} // no /dev/net/tun
	r := Run(context.Background(), env)
	if r.Healthy() {
		t.Fatal("missing /dev/net/tun must FAIL")
	}
}

func TestNonRootIsWarnNotFail(t *testing.T) {
	env := fullyHealthyEnv()
	env.root = false
	r := Run(context.Background(), env)
	if !r.Healthy() {
		t.Fatal("non-root should WARN, not FAIL")
	}
	if r.Warn == 0 {
		t.Error("expected a privilege WARN")
	}
}

func TestNoDNSBackendWarns(t *testing.T) {
	env := fullyHealthyEnv()
	env.present = map[string]bool{"ip": true, "nft": true} // no resolvectl/resolvconf
	r := Run(context.Background(), env)
	if !r.Healthy() {
		t.Fatal("missing DNS backend should WARN, not FAIL")
	}
	if r.Warn == 0 {
		t.Error("expected a DNS backend WARN")
	}
}
