package reachcache

import (
	"testing"
	"time"
)

func TestReorderPrefersWorking(t *testing.T) {
	c := NewAt(t.TempDir() + "/rc.json")
	c.RecordOK("Amsterdam")
	c.RecordFail("Zurich")
	c.RecordFail("Zurich")
	// input in ICMP order: Zurich fastest, Amsterdam second, Paris unknown
	got := c.Reorder([]string{"Zurich", "Amsterdam", "Paris"}, time.Hour)
	if got[0] != "Amsterdam" {
		t.Errorf("working zone must be first, got %v", got)
	}
	if got[len(got)-1] != "Zurich" {
		t.Errorf("failed zone must be last, got %v", got)
	}
}
func TestReorderStaleIgnored(t *testing.T) {
	c := NewAt(t.TempDir() + "/rc.json")
	c.RecordFail("Zurich")
	// with 0 ttl every record is stale → order preserved
	got := c.Reorder([]string{"Zurich", "Amsterdam"}, 0)
	if got[0] != "Zurich" {
		t.Errorf("stale record must not reorder, got %v", got)
	}
}
