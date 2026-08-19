// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package measure probes VPN endpoints for reachability and latency so the CLI
// can validate that profiles actually work and rank the best zones to connect
// to — without bringing the tunnel up.
package measure

import (
	"context"
	"net"
	"sort"
	"time"
)

// Result is the reachability measurement for one endpoint.
type Result struct {
	Name      string `json:"name"`
	Endpoint  string `json:"endpoint"`
	Reachable bool   `json:"reachable"`
	LatencyMS int64  `json:"latency_ms"`
	// ICMPAlive is true when the server host answered an ICMP ping. This is the
	// strongest liveness signal: a WireGuard server that does not answer ICMP
	// AND gives no handshake is almost certainly dead. UDP-socket setup alone
	// (no ICMP) is a weak signal and must not outrank a real ICMP reply.
	ICMPAlive bool          `json:"icmp_alive"`
	Err       string        `json:"error,omitempty"`
	latency   time.Duration // internal, for sorting
}

// Dialer abstracts dialing so tests can inject a fake (no real network). The
// network is "tcp" or "udp".
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Resolver abstracts DNS resolution so tests can avoid the network.
type Resolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}

// Measurer probes endpoints. Timeout bounds each probe (default 3s).
type Measurer struct {
	Dialer   Dialer
	Resolver Resolver
	Timeout  time.Duration
	// Pinger, when set, provides a true ICMP RTT for the endpoint host, which
	// is far more meaningful than a UDP socket setup time for WireGuard.
	Pinger *Pinger
}

// New returns a Measurer using the real network, including an ICMP pinger for
// true round-trip latency.
func New() *Measurer {
	return &Measurer{
		Dialer:   &net.Dialer{},
		Resolver: net.DefaultResolver,
		Timeout:  3 * time.Second,
		Pinger:   NewPinger(),
	}
}

// NewViaUplink is like New but binds ICMP pings to a physical uplink interface,
// bypassing any active VPN tunnel (e.g. AdGuard) so server liveness is measured
// truthfully. Pass an empty string to fall back to the default route.
func NewViaUplink(iface string) *Measurer {
	m := New()
	if iface != "" {
		m.Pinger.Interface = iface
	}
	return m
}

func (m *Measurer) timeout() time.Duration {
	if m.Timeout > 0 {
		return m.Timeout
	}
	return 3 * time.Second
}

// Probe measures reachability + latency to a WireGuard/AmneziaWG UDP endpoint.
//
// WireGuard servers speak UDP and never reply to an unauthenticated packet, so
// a TCP connect would always fail. Instead we:
//  1. resolve the endpoint host (proves the server DNS/name is valid), and
//  2. "connect" a UDP socket and send a probe datagram, measuring the time to
//     establish the socket and write. A successful DNS + UDP write is treated
//     as reachable; the latency is the round time of resolve+dial+write.
//
// This yields a stable ranking signal without bringing the tunnel up, and does
// not require the server to answer (which it won't, by design).
func (m *Measurer) Probe(ctx context.Context, name, endpoint string) Result {
	res := Result{Name: name, Endpoint: endpoint}
	if endpoint == "" {
		res.Err = "no endpoint"
		return res
	}
	to := m.timeout()
	cctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()

	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		res.Err = "bad endpoint: " + err.Error()
		res.latency = to + time.Second
		return res
	}

	start := time.Now()
	// 1. DNS resolution (skip if already an IP literal).
	if net.ParseIP(host) == nil && m.Resolver != nil {
		if _, err := m.Resolver.LookupHost(cctx, host); err != nil {
			res.Err = "dns: " + err.Error()
			res.LatencyMS = time.Since(start).Milliseconds()
			res.latency = to + time.Second
			return res
		}
	}
	// 2. UDP socket connect + probe write.
	conn, err := m.Dialer.DialContext(cctx, "udp", endpoint)
	if err != nil {
		res.Err = err.Error()
		res.LatencyMS = time.Since(start).Milliseconds()
		res.latency = to + time.Second
		return res
	}
	_, wErr := conn.Write([]byte{0})
	_ = conn.Close()
	elapsed := time.Since(start)
	if wErr != nil {
		res.Err = wErr.Error()
		res.LatencyMS = elapsed.Milliseconds()
		res.latency = to + time.Second
		return res
	}
	res.Reachable = true

	// Prefer a true ICMP RTT to the server host; fall back to socket-setup time.
	// An ICMP reply also proves the host is actually alive (ICMPAlive).
	if m.Pinger != nil {
		if rtt, ok := m.Pinger.Ping(cctx, host); ok {
			res.ICMPAlive = true
			res.latency = time.Duration(rtt * float64(time.Millisecond))
			res.LatencyMS = int64(rtt + 0.5)
			return res
		}
	}
	res.latency = elapsed
	res.LatencyMS = elapsed.Milliseconds()
	return res
}

// Target is an endpoint to measure.
type Target struct {
	Name     string
	Endpoint string
}

// RankBest probes all targets concurrently and returns them sorted best-first
// (reachable + lowest latency). Concurrency is bounded to avoid a probe storm.
func (m *Measurer) RankBest(ctx context.Context, targets []Target) []Result {
	return m.RankBestProgress(ctx, targets, nil)
}

// RankBestProgress is RankBest with an optional progress callback invoked once
// per completed probe (done, total). It lets a UI render a live "probing k/N"
// indicator so a large catalog never looks like a hang. The callback runs on a
// worker goroutine and must be cheap/thread-safe; nil disables it.
//
// Concurrency scales with the catalog size (bounded) so 50 profiles no longer
// serialize into ~40s of dead air — the tiny ICMP pool was the single biggest
// cause of the "test/rank hangs" report. It stays modest to avoid false
// timeouts through a busy VPN tunnel, but honoring ctx cancellation plus the
// per-probe deadline guarantees bounded work.
func (m *Measurer) RankBestProgress(ctx context.Context, targets []Target, progress func(done, total int)) []Result {
	maxConcurrent := 8
	if len(targets) > 24 {
		maxConcurrent = 12
	}
	if len(targets) > 0 && len(targets) < maxConcurrent {
		maxConcurrent = len(targets)
	}
	sem := make(chan struct{}, maxConcurrent)
	results := make([]Result, len(targets))
	done := make(chan int, len(targets))

	for i, t := range targets {
		select {
		case <-ctx.Done():
			// Context cancelled/timed out: stop launching new probes and mark the
			// remainder cancelled so the caller still gets a complete slice.
			results[i] = Result{Name: t.Name, Endpoint: t.Endpoint, Err: "cancelled", latency: m.timeout() + time.Second}
			done <- i
			continue
		case sem <- struct{}{}:
		}
		go func(idx int, tg Target) {
			defer func() { <-sem; done <- idx }()
			results[idx] = m.Probe(ctx, tg.Name, tg.Endpoint)
		}(i, t)
	}
	completed := 0
	for range targets {
		<-done
		completed++
		if progress != nil {
			progress(completed, len(targets))
		}
	}

	// Ranking priority:
	//   1. ICMP-alive servers first (proven reachable at the IP layer),
	//   2. then any UDP-reachable server,
	//   3. then by lowest latency.
	// This fixes dead servers (100% ICMP loss) ranking as "0 ms" ahead of a
	// real, live server.
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].ICMPAlive != results[j].ICMPAlive {
			return results[i].ICMPAlive
		}
		if results[i].Reachable != results[j].Reachable {
			return results[i].Reachable
		}
		return results[i].latency < results[j].latency
	})
	return results
}

// BestAlive returns the first ICMP-alive result, or the first reachable one if
// none answered ICMP, or false if nothing is usable.
func BestAlive(results []Result) (Result, bool) {
	for _, r := range results {
		if r.ICMPAlive {
			return r, true
		}
	}
	for _, r := range results {
		if r.Reachable {
			return r, true
		}
	}
	return Result{}, false
}
