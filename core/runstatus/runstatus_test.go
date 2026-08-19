// SPDX-License-Identifier: AGPL-3.0-or-later
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
