// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package health implements the mazzy-core connection health monitor and
// auto-reconnect decision logic, mirroring the bash health_check /
// health_failure / health_recover semantics:
//
//   - a consecutive-failure counter (default limit 2),
//   - a startup grace window (default 60s) during which interface/data-plane
//     churn does not count as failure,
//   - recovery is triggered exactly once when the counter reaches the limit,
//   - after the limit, further failures back off to the caller's own retry
//     (no repeated destructive restarts).
//
// The actual probes (link present, handshake, internet reachable) and the
// recover action are injected, so this package is deterministic and unit
// testable without touching the network.
package health

import (
	"context"
	"time"
)

// Probe reports the observed state of the connection. All methods must be
// non-destructive and bounded.
type Probe interface {
	// LinkPresent reports whether the VPN interface exists.
	LinkPresent(ctx context.Context) bool
	// InternetOK reports whether the internet is reachable THROUGH the tunnel.
	// reason carries a human-readable cause when not ok.
	InternetOK(ctx context.Context) (ok bool, reason string)
	// InStartupGrace reports whether the connection is still within its
	// startup grace window (interface/data-plane may still be settling).
	InStartupGrace(ctx context.Context) bool
}

// Recoverer performs the recovery action (e.g. reconnect). It returns nil on a
// confirmed protected egress, or an error otherwise.
type Recoverer interface {
	Recover(ctx context.Context, reason string) error
}

// Config tunes the monitor. Zero values fall back to bash-parity defaults.
type Config struct {
	FailureLimit int // default 2
}

func (c Config) limit() int {
	if c.FailureLimit >= 1 {
		return c.FailureLimit
	}
	return 2
}

// Monitor tracks consecutive failures and decides when to recover. It is not
// safe for concurrent use; drive it from a single loop.
type Monitor struct {
	cfg   Config
	probe Probe
	rec   Recoverer

	failures int
	// recoveredAtLimit marks that we already fired recovery for the current
	// failure streak, so we don't restart destructively on every tick.
	recoveredAtLimit bool
}

// New builds a Monitor.
func New(cfg Config, probe Probe, rec Recoverer) *Monitor {
	return &Monitor{cfg: cfg, probe: probe, rec: rec}
}

// Result describes the outcome of a single Check tick.
type Result struct {
	Healthy      bool
	Failures     int
	Limit        int
	Recovered    bool   // recovery was attempted this tick
	RecoverError error  // non-nil if recovery attempted and failed
	Reason       string // failure/deferral reason
	Deferred     bool   // failure not counted (startup grace)
}

// Check performs one health tick. Parity with health_check:
//  1. if link missing → failure (unless startup grace defers it);
//  2. else if internet not ok → failure (unless startup grace defers it);
//  3. else → healthy, reset counter.
//
// When the failure counter reaches the limit, recovery fires once.
func (m *Monitor) Check(ctx context.Context) Result {
	limit := m.cfg.limit()

	if !m.probe.LinkPresent(ctx) {
		if m.probe.InStartupGrace(ctx) {
			return Result{Failures: m.failures, Limit: limit, Deferred: true,
				Reason: "interface still being created (startup grace)"}
		}
		return m.fail(ctx, "VPN interface missing", limit)
	}

	if ok, reason := m.probe.InternetOK(ctx); !ok {
		if m.probe.InStartupGrace(ctx) {
			return Result{Failures: m.failures, Limit: limit, Deferred: true,
				Reason: "data plane not ready (startup grace)"}
		}
		if reason == "" {
			reason = "no internet access through VPN"
		}
		return m.fail(ctx, reason, limit)
	}

	// Healthy: reset the streak.
	m.reset()
	return Result{Healthy: true, Failures: 0, Limit: limit}
}

// fail increments the failure counter and fires recovery exactly at the limit.
func (m *Monitor) fail(ctx context.Context, reason string, limit int) Result {
	if m.failures < limit+1 {
		m.failures++
	}
	res := Result{Failures: m.failures, Limit: limit, Reason: reason}

	if m.failures == limit && !m.recoveredAtLimit {
		m.recoveredAtLimit = true
		res.Recovered = true
		res.RecoverError = m.rec.Recover(ctx, reason)
	}
	return res
}

func (m *Monitor) reset() {
	m.failures = 0
	m.recoveredAtLimit = false
}

// Failures returns the current consecutive-failure count (for observability).
func (m *Monitor) Failures() int { return m.failures }

// RunUntil drives Check on an interval until ctx is cancelled, invoking onTick
// for each result. Intended for the daemon loop; tests call Check directly.
func (m *Monitor) RunUntil(ctx context.Context, interval time.Duration, onTick func(Result)) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r := m.Check(ctx)
			if onTick != nil {
				onTick(r)
			}
		}
	}
}
