// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package runstatus

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestWriteReadRoundTripAndClose(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAZZY_RUN_DIR", dir)

	w := NewWriter("Berlin", "mazzy0", "AmneziaWG", false)
	w.SetState(StateProtected, "mazzy0", "1.2.3.4")
	w.Tick(42, true)
	w.Tick(0, false)
	w.Error("egress lost")

	snap, ok := Read()
	if !ok {
		t.Fatal("expected a readable heartbeat")
	}
	if snap.State != StateProtected || snap.Zone != "Berlin" || snap.Egress != "1.2.3.4" {
		t.Errorf("unexpected snapshot: %+v", snap)
	}
	if snap.Checks != 2 || snap.Fails != 1 {
		t.Errorf("checks/fails = %d/%d, want 2/1", snap.Checks, snap.Fails)
	}
	if len(snap.Samples) != 2 || len(snap.Errors) != 1 {
		t.Errorf("samples/errors = %d/%d, want 2/1", len(snap.Samples), len(snap.Errors))
	}
	if !snap.Fresh(time.Minute) {
		t.Error("just-written snapshot should be fresh")
	}

	// Close removes the file so a stale heartbeat never lingers.
	w.Close()
	if _, err := os.Stat(Path()); !os.IsNotExist(err) {
		t.Errorf("Close should remove the heartbeat file, stat err=%v", err)
	}
	if _, ok := Read(); ok {
		t.Error("Read should fail after Close")
	}
}

func TestRingBuffersAreBounded(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MAZZY_RUN_DIR", dir)
	w := NewWriter("z", "i", "p", true)
	for i := 0; i < maxSamples+30; i++ {
		w.Tick(i+1, true)
	}
	for i := 0; i < maxErrors+10; i++ {
		w.Error("e")
	}
	snap, _ := Read()
	if len(snap.Samples) != maxSamples {
		t.Errorf("samples bounded to %d, got %d", maxSamples, len(snap.Samples))
	}
	if len(snap.Errors) != maxErrors {
		t.Errorf("errors bounded to %d, got %d", maxErrors, len(snap.Errors))
	}
	if !snap.Background {
		t.Error("background flag should persist")
	}
}

func TestFreshFalseWhenStale(t *testing.T) {
	s := Snapshot{UpdatedAt: time.Now().Add(-time.Hour).Unix()}
	if s.Fresh(time.Minute) {
		t.Error("hour-old snapshot must not be fresh within a minute")
	}
	if (Snapshot{}).Fresh(time.Minute) {
		t.Error("zero snapshot must not be fresh")
	}
}

func TestErrorRateAndRecent(t *testing.T) {
	now := time.Now().Unix()
	s := Snapshot{Errors: []ErrEvent{
		{TS: now - 30, Reason: "a"},
		{TS: now - 20, Reason: "b"},
		{TS: now - 5000, Reason: "old"},
	}}
	// 2 errors in the last minute → 2.0/min.
	if got := s.ErrorRatePerMin(time.Minute); got != 2.0 {
		t.Errorf("error rate = %v, want 2.0", got)
	}
	recent := s.RecentErrors(2)
	if len(recent) != 2 || recent[0].Reason != "old" {
		// newest first: last appended is "old"
		t.Errorf("recent errors = %+v", recent)
	}
}

func TestSparklineRendersAndMarksDrops(t *testing.T) {
	if got := Sparkline(nil, 5); got != "·····" {
		t.Errorf("empty series should be all gaps, got %q", got)
	}
	spark := Sparkline([]int{10, 0, 100}, 3)
	if len([]rune(spark)) != 3 {
		t.Errorf("sparkline width = %d, want 3", len([]rune(spark)))
	}
	if !strings.Contains(spark, "·") {
		t.Errorf("drop (0) should render a gap marker, got %q", spark)
	}
}

func TestLatencyStats(t *testing.T) {
	min, avg, max := LatencyStats([]int{10, 0, 30, 20})
	if min != 10 || max != 30 || avg != 20 {
		t.Errorf("stats = %d/%d/%d, want 10/20/30", min, avg, max)
	}
	if min, avg, max := LatencyStats([]int{0, 0}); min+avg+max != 0 {
		t.Errorf("all-zero series should yield 0/0/0")
	}
}

func TestDerivedDashboardMetrics(t *testing.T) {
	s := Snapshot{
		Checks: 4, Fails: 1, UpdatedAt: time.Now().Add(-2 * time.Second).Unix(),
		Samples: []Sample{
			{LatencyMS: 10, OK: true},
			{LatencyMS: 20, OK: true},
			{LatencyMS: 0, OK: false},
			{LatencyMS: 40, OK: true},
		},
	}
	if got := s.LossPercent(); got != 25 {
		t.Errorf("loss = %.1f, want 25", got)
	}
	if got := s.LatencyPercentile(50); got != 20 {
		t.Errorf("p50 = %d, want 20", got)
	}
	if got := s.LatencyPercentile(95); got != 40 {
		t.Errorf("p95 = %d, want 40", got)
	}
	if got := s.JitterMS(); got != 15 { // |20-10| + |40-20| / 2
		t.Errorf("jitter = %d, want 15", got)
	}
	if age := s.HeartbeatAge(); age < time.Second || age > 5*time.Second {
		t.Errorf("heartbeat age = %v, want about 2s", age)
	}
}

// TestTouchRefreshesHeartbeat: the daemon's pulse goroutine must be able to
// keep the file fresh without changing any payload fields.
func TestTouchRefreshesHeartbeat(t *testing.T) {
	t.Setenv("MAZZY_RUN_DIR", t.TempDir())
	w := NewWriter("Berlin", "vpnaw0", "AmneziaWG", true)
	w.Touch()
	s, ok := Read()
	if !ok || !s.Fresh(5*time.Second) {
		t.Fatalf("touched heartbeat must be fresh, got %+v ok=%v", s, ok)
	}
	if s.Zone != "Berlin" {
		t.Errorf("Touch must not alter payload, zone = %q", s.Zone)
	}
}

// TestTouchAfterCloseDoesNotResurrect: a racing pulse tick right after a clean
// shutdown must not recreate the heartbeat file (readers would see a ghost
// daemon forever).
func TestTouchAfterCloseDoesNotResurrect(t *testing.T) {
	t.Setenv("MAZZY_RUN_DIR", t.TempDir())
	w := NewWriter("Berlin", "vpnaw0", "AmneziaWG", true)
	w.Close()
	w.Touch()
	if _, ok := Read(); ok {
		t.Fatal("Touch after Close must not resurrect the heartbeat file")
	}
}

// TestSetProtocolRecordsProtocol: the dashboard previously always saw "".
func TestSetProtocolRecordsProtocol(t *testing.T) {
	t.Setenv("MAZZY_RUN_DIR", t.TempDir())
	w := NewWriter("Berlin", "", "", false)
	w.SetProtocol("AmneziaWG")
	if s, ok := Read(); !ok || s.Protocol != "AmneziaWG" {
		t.Fatalf("protocol not recorded: %+v ok=%v", s, ok)
	}
}

// TestEffectiveMSPrefersPing: the graph/stats must use the honest ICMP RTT
// when measured and fall back to the HTTPS probe duration otherwise.
func TestEffectiveMSPrefersPing(t *testing.T) {
	if (Sample{LatencyMS: 250, PingMS: 40}).EffectiveMS() != 40 {
		t.Error("ping must win over probe duration")
	}
	if (Sample{LatencyMS: 250}).EffectiveMS() != 250 {
		t.Error("probe duration is the fallback")
	}
	s := Snapshot{Samples: []Sample{{LatencyMS: 250, PingMS: 40, OK: true}, {LatencyMS: 260, OK: true}}}
	series := s.LatencySeries()
	if series[0] != 40 || series[1] != 260 {
		t.Errorf("series = %v, want [40 260]", series)
	}
	if p := s.LatencyPercentile(100); p != 260 {
		t.Errorf("p100 = %d, want 260", p)
	}
}
