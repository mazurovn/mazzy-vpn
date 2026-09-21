package main

import (
	"testing"
	"time"
)

// TestLinkQualityAlternatingProbes reproduces the 2026-09-21 incident shape:
// failure/success alternate forever. The old consecutive counter never fired;
// the window must.
func TestLinkQualityAlternatingProbes(t *testing.T) {
	var q linkQuality
	fired := -1
	for i := 0; i < 24; i++ {
		q.noteProbe(i%2 == 1) // F,S,F,S,...
		if q.degraded() {
			fired = i
			break
		}
	}
	if fired < 0 {
		t.Fatal("alternating F/S probes never marked the tunnel degraded")
	}
	// 5th failure is at index 8 (F at 0,2,4,6,8).
	if fired != 8 {
		t.Errorf("degraded fired at probe %d, want 8", fired)
	}
}

func TestLinkQualityIsolatedBlipsStayHealthy(t *testing.T) {
	var q linkQuality
	// 4 failures spread over a full window must not trip it.
	pattern := []bool{true, true, false, true, true, false, true, true, false, true, true, false}
	for _, ok := range pattern {
		q.noteProbe(ok)
	}
	if q.degraded() {
		t.Fatalf("4/12 failures marked degraded: lost=%d", q.lost())
	}
	// Window slides: after 12 more successes the old failures are gone.
	for i := 0; i < 12; i++ {
		q.noteProbe(true)
	}
	if q.lost() != 0 {
		t.Errorf("window did not slide, lost=%d", q.lost())
	}
	if len(q.hist) != probeWindow {
		t.Errorf("window length %d, want %d", len(q.hist), probeWindow)
	}
}

func TestLinkQualityHandshakeChurn(t *testing.T) {
	var q linkQuality
	// Peer re-keys every ~30s: age seen at ticks 28s, 0s, 30s, 1s, 29s, 2s...
	ages := []time.Duration{28 * time.Second, 1 * time.Second, 30 * time.Second, 2 * time.Second,
		29 * time.Second, 0, 28 * time.Second, 1 * time.Second}
	fired := -1
	for i, a := range ages {
		q.noteHandshake(a, true)
		if q.degraded() {
			fired = i
			break
		}
	}
	if fired < 0 {
		t.Fatalf("30s re-handshakes never marked degraded, churn=%d", q.churn)
	}
	// Drops: 28→1 (1), 30→2 (2), 29→0 (3), 28→1 (4) — fires at index 7.
	if fired != 7 {
		t.Errorf("churn fired at tick %d, want 7", fired)
	}
}

func TestLinkQualityNormalRekeyResetsChurn(t *testing.T) {
	var q linkQuality
	q.noteHandshake(20*time.Second, true)
	q.noteHandshake(1*time.Second, true) // one early re-key
	q.noteHandshake(10*time.Second, true)
	if q.churn != 1 {
		t.Fatalf("churn=%d, want 1", q.churn)
	}
	// Session then stays up: age grows past hsChurnAge.
	q.noteHandshake(95*time.Second, true)
	if q.churn != 0 {
		t.Errorf("mature handshake did not reset churn, got %d", q.churn)
	}
	// A normal 2-minute rekey (age drops from 130s) is not churn: the previous
	// handshake was older than hsChurnAge.
	q.noteHandshake(130*time.Second, true)
	q.noteHandshake(3*time.Second, true)
	if q.churn != 0 {
		t.Errorf("normal rekey counted as churn, got %d", q.churn)
	}
	// Unknown handshake is ignored.
	q.noteHandshake(0, false)
	if q.lastHSAge != 3*time.Second {
		t.Errorf("hsOK=false mutated state: lastHSAge=%v", q.lastHSAge)
	}
}

func TestLinkQualityReset(t *testing.T) {
	var q linkQuality
	for i := 0; i < 6; i++ {
		q.noteProbe(false)
	}
	q.noteHandshake(20*time.Second, true)
	q.noteHandshake(1*time.Second, true)
	q.reset()
	if q.degraded() || len(q.hist) != 0 || q.churn != 0 || q.lastHSAge != 0 {
		t.Errorf("reset left state behind: %+v", q)
	}
}
