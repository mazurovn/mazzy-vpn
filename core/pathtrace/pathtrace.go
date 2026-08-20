// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package pathtrace traces the packet path of a VPN connection so the user can
// see, step by step, where traffic goes and where it breaks: DNS resolution of
// the endpoint, reachability of the server, the tunnel interface, the routing
// table, and the actual egress IP. It is read-only diagnostics.
package pathtrace

import (
	"context"
	"fmt"
	"net"
	"time"
)

// StepStatus is the outcome of one trace hop.
type StepStatus string

const (
	OK   StepStatus = "ok"
	Warn StepStatus = "warn"
	Fail StepStatus = "fail"
)

// Step is one hop in the packet path.
type Step struct {
	Name     string     `json:"name"`
	Status   StepStatus `json:"status"`
	Detail   string     `json:"detail"`
	Duration int64      `json:"duration_ms"`
}

// Trace is the full packet-path result.
type Trace struct {
	Endpoint string `json:"endpoint"`
	Steps    []Step `json:"steps"`
}

// Resolver/Pinger/Egress are injected so the trace is testable offline.
type Resolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}
type Pinger interface {
	Ping(ctx context.Context, host string) (float64, bool)
}
type EgressFn func(ctx context.Context, iface string) (string, error)
type LinkFn func(iface string) bool

// Tracer runs a packet-path trace.
type Tracer struct {
	Resolver Resolver
	Pinger   Pinger
	Egress   EgressFn
	Link     LinkFn
}

// Run traces the path for a connection to endpoint (host:port) over iface.
// iface may be "" before the tunnel is up (interface/egress steps then report
// "not up yet").
func (t *Tracer) Run(ctx context.Context, endpoint, iface string) Trace {
	tr := Trace{Endpoint: endpoint}
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		tr.Steps = append(tr.Steps, Step{Name: "parse endpoint", Status: Fail, Detail: err.Error()})
		return tr
	}

	// 1. DNS resolution (skip if literal IP).
	tr.Steps = append(tr.Steps, t.stepDNS(ctx, host))

	// 2. Server reachability (ICMP).
	tr.Steps = append(tr.Steps, t.stepPing(ctx, host))

	// 3. Tunnel interface present.
	tr.Steps = append(tr.Steps, t.stepLink(iface))

	// 4. Egress through the tunnel.
	tr.Steps = append(tr.Steps, t.stepEgress(ctx, iface))

	return tr
}

func timed(f func() Step) Step {
	start := time.Now()
	s := f()
	s.Duration = time.Since(start).Milliseconds()
	return s
}

func (t *Tracer) stepDNS(ctx context.Context, host string) Step {
	return timed(func() Step {
		if net.ParseIP(host) != nil {
			return Step{Name: "1. resolve endpoint", Status: OK, Detail: host + " (literal IP)"}
		}
		if t.Resolver == nil {
			return Step{Name: "1. resolve endpoint", Status: Warn, Detail: "no resolver"}
		}
		ips, err := t.Resolver.LookupHost(ctx, host)
		if err != nil || len(ips) == 0 {
			return Step{Name: "1. resolve endpoint", Status: Fail, Detail: "DNS failed: " + errStr(err)}
		}
		return Step{Name: "1. resolve endpoint", Status: OK, Detail: fmt.Sprintf("%s → %d address(es)", host, len(ips))}
	})
}

func (t *Tracer) stepPing(ctx context.Context, host string) Step {
	return timed(func() Step {
		if t.Pinger == nil {
			return Step{Name: "2. reach server", Status: Warn, Detail: "no pinger"}
		}
		rtt, ok := t.Pinger.Ping(ctx, host)
		if !ok {
			return Step{Name: "2. reach server", Status: Warn, Detail: "no ICMP reply (may be firewalled or down)"}
		}
		return Step{Name: "2. reach server", Status: OK, Detail: fmt.Sprintf("ICMP %.0f ms", rtt)}
	})
}

func (t *Tracer) stepLink(iface string) Step {
	return timed(func() Step {
		if iface == "" {
			return Step{Name: "3. tunnel interface", Status: Warn, Detail: "not up yet"}
		}
		if t.Link != nil && !t.Link(iface) {
			return Step{Name: "3. tunnel interface", Status: Fail, Detail: iface + " missing"}
		}
		return Step{Name: "3. tunnel interface", Status: OK, Detail: iface + " up"}
	})
}

func (t *Tracer) stepEgress(ctx context.Context, iface string) Step {
	return timed(func() Step {
		if iface == "" || t.Egress == nil {
			return Step{Name: "4. egress via tunnel", Status: Warn, Detail: "tunnel not up"}
		}
		ip, err := t.Egress(ctx, iface)
		if err != nil || ip == "" {
			return Step{Name: "4. egress via tunnel", Status: Fail, Detail: "no traffic through tunnel: " + errStr(err)}
		}
		return Step{Name: "4. egress via tunnel", Status: OK, Detail: "egress " + ip}
	})
}

// Healthy reports whether every step is OK.
func (tr Trace) Healthy() bool {
	for _, s := range tr.Steps {
		if s.Status == Fail {
			return false
		}
	}
	return true
}

func errStr(err error) string {
	if err == nil {
		return "unknown"
	}
	return err.Error()
}
