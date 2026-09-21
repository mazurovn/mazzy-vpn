package main

import "time"

// linkQuality accumulates the two signals that distinguish a *degraded*
// tunnel (crypto layer alive, data path dropping packets) from a healthy one
// with an occasional probe blip. Observed incident 2026-09-21: a tunnel whose
// handshake was always "fresh" but whose data path dropped ~50% of probes sat
// for 18 minutes at softFails=1/6 because every successful probe reset the
// counter to zero, so the daemon never failed over.
//
//   - Rolling probe window: too many failures inside the last probeWindow
//     outcomes count as loss even when they are not consecutive.
//   - Handshake churn: WireGuard re-handshakes every ~30s only when its data
//     packets go unanswered (healthy peers rekey every 2–3 min). Repeated young
//     handshakes therefore mean "crypto works, routing does not".
type linkQuality struct {
	hist      []bool // rolling probe outcomes, true = egress confirmed
	lastHSAge time.Duration
	churn     int
}

const (
	probeWindow    = 12               // rolling window of probe outcomes (~2–4 min of ticks)
	probeLossLimit = 5                // ≥ this many failures in the window = degraded tunnel
	hsChurnLimit   = 4                // this many back-to-back young re-handshakes = dead data path
	hsChurnAge     = 90 * time.Second // a handshake younger than this that gets replaced is "early"
)

// noteProbe records one probe outcome and returns the failure count inside
// the window.
func (q *linkQuality) noteProbe(ok bool) int {
	q.hist = append(q.hist, ok)
	if len(q.hist) > probeWindow {
		q.hist = q.hist[len(q.hist)-probeWindow:]
	}
	return q.lost()
}

func (q *linkQuality) lost() int {
	n := 0
	for _, v := range q.hist {
		if !v {
			n++
		}
	}
	return n
}

// noteHandshake feeds the churn counter from the current handshake age: an
// age that went *down* while the previous one was still young means the peer
// re-keyed early. An age at or beyond hsChurnAge proves a stable session and
// clears the counter.
func (q *linkQuality) noteHandshake(age time.Duration, ok bool) {
	if !ok {
		return
	}
	switch {
	case q.lastHSAge > 0 && age < q.lastHSAge && q.lastHSAge < hsChurnAge:
		q.churn++
	case age >= hsChurnAge:
		q.churn = 0
	}
	q.lastHSAge = age
}

// degraded reports whether either signal crossed its limit.
func (q *linkQuality) degraded() bool {
	return q.lost() >= probeLossLimit || q.churn >= hsChurnLimit
}

// reset clears everything; used when the tunnel is torn down so the next
// zone is judged only by its own evidence.
func (q *linkQuality) reset() { *q = linkQuality{} }
