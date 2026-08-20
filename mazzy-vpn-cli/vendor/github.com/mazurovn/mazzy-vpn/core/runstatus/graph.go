// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package runstatus

import "strings"

// sparkBars are the eight Unicode block levels used for the latency graph.
var sparkBars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Sparkline renders a latency series as a compact Unicode bar chart of the last
// width samples. Missing egress (value <= 0) renders as a low gap marker so the
// user can see drops at a glance. Lower bars = lower latency (better).
func Sparkline(series []int, width int) string {
	if width <= 0 {
		width = 40
	}
	if len(series) == 0 {
		return strings.Repeat("·", width)
	}
	// Take the trailing window.
	if len(series) > width {
		series = series[len(series)-width:]
	}
	// Find min/max over the positive samples to scale the bars.
	min, max := 0, 0
	first := true
	for _, v := range series {
		if v <= 0 {
			continue
		}
		if first {
			min, max = v, v
			first = false
			continue
		}
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	var b strings.Builder
	for _, v := range series {
		if v <= 0 {
			b.WriteRune('·') // drop / no egress
			continue
		}
		if max == min {
			b.WriteRune(sparkBars[0])
			continue
		}
		// Scale into [0,7].
		idx := (v - min) * (len(sparkBars) - 1) / (max - min)
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkBars) {
			idx = len(sparkBars) - 1
		}
		b.WriteRune(sparkBars[idx])
	}
	return b.String()
}

// LatencyStats returns min/avg/max over the positive samples (0,0,0 if none).
func LatencyStats(series []int) (min, avg, max int) {
	sum, n := 0, 0
	for _, v := range series {
		if v <= 0 {
			continue
		}
		if n == 0 || v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
		n++
	}
	if n == 0 {
		return 0, 0, 0
	}
	return min, sum / n, max
}
