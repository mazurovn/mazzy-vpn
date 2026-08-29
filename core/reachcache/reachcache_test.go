package reachcache

import (
	"os"
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

func TestVerdict(t *testing.T) {
	c := NewAt(t.TempDir() + "/rc.json")
	c.RecordOK("Amsterdam")
	c.RecordFail("Zurich")
	c.RecordFail("Zurich")

	if v, _ := c.Verdict("Amsterdam", time.Hour); v != "ok" {
		t.Errorf("Amsterdam: want ok, got %s", v)
	}
	if v, streak := c.Verdict("Zurich", time.Hour); v != "fail" || streak != 2 {
		t.Errorf("Zurich: want fail/2, got %s/%d", v, streak)
	}
	if v, _ := c.Verdict("Paris", time.Hour); v != "unknown" {
		t.Errorf("Paris: want unknown, got %s", v)
	}
	// TTL 0: everything is stale → unknown.
	if v, _ := c.Verdict("Zurich", 0); v != "unknown" {
		t.Errorf("stale Zurich: want unknown, got %s", v)
	}
}

func TestSharedFileWorldReadable(t *testing.T) {
	p := t.TempDir() + "/rc.json"
	c := NewAt(p)
	c.RecordOK("Amsterdam")
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o644 {
		t.Errorf("shared cache must be 0644 (unprivileged UI reads it), got %o", fi.Mode().Perm())
	}
}
