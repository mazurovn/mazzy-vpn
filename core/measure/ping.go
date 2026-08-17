// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package measure

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Pinger measures ICMP round-trip time to a host. Implemented via the base OS
// `ping` tool (part of iputils; already required by the runtime), so it needs
// no raw-socket privileges and no extra Go dependencies.
type Pinger struct {
	// Count is the number of echo requests (default 1).
	Count int
	// Timeout bounds the whole ping (default 3s).
	Timeout time.Duration
	// Interface, when set, binds ping to a physical uplink (ping -I <iface>).
	// This is essential when another VPN (e.g. AdGuard) is active: pinging via
	// the default route would go through that tunnel, which often drops ICMP
	// and produces false "dead" results. Measuring via the real uplink gives
	// the true server liveness.
	Interface string
	// runner executes ping; overridable in tests.
	runner func(ctx context.Context, args ...string) (string, error)
}

// NewPinger returns a Pinger using the system ping command.
func NewPinger() *Pinger {
	return &Pinger{Count: 1, Timeout: 3 * time.Second}
}

func (p *Pinger) count() int {
	if p.Count > 0 {
		return p.Count
	}
	return 1
}

func (p *Pinger) timeout() time.Duration {
	if p.Timeout > 0 {
		return p.Timeout
	}
	return 3 * time.Second
}

func (p *Pinger) run(ctx context.Context, args ...string) (string, error) {
	if p.runner != nil {
		return p.runner(ctx, args...)
	}
	cctx, cancel := context.WithTimeout(ctx, p.timeout())
	defer cancel()
	out, err := exec.CommandContext(cctx, "ping", args...).CombinedOutput()
	return string(out), err
}

// rttRe matches the summary line "rtt min/avg/max/mdev = 89.670/91.088/..."
// and the per-reply "time=91.0 ms".
var (
	rttAvgRe  = regexp.MustCompile(`=\s*[\d.]+/([\d.]+)/`)
	rttTimeRe = regexp.MustCompile(`time[=<]([\d.]+)`)
)

// Ping returns the average RTT in milliseconds to host, or ok=false when the
// host is unreachable. host must be a bare hostname/IP (no port).
func (p *Pinger) Ping(ctx context.Context, host string) (ms float64, ok bool) {
	if strings.TrimSpace(host) == "" {
		return 0, false
	}
	// -c count, -w deadline(s), -n numeric output.
	deadline := int(p.timeout().Seconds())
	if deadline < 1 {
		deadline = 1
	}
	args := []string{
		"-c", strconv.Itoa(p.count()),
		"-w", strconv.Itoa(deadline),
		"-n",
	}
	if p.Interface != "" {
		args = append(args, "-I", p.Interface)
	}
	args = append(args, host)
	out, err := p.run(ctx, args...)
	if err != nil {
		// ping exits non-zero on 100% loss; still try to parse a partial reply.
		if m := rttAvgRe.FindStringSubmatch(out); m != nil {
			if v, e := strconv.ParseFloat(m[1], 64); e == nil {
				return v, true
			}
		}
		return 0, false
	}
	if m := rttAvgRe.FindStringSubmatch(out); m != nil {
		if v, e := strconv.ParseFloat(m[1], 64); e == nil {
			return v, true
		}
	}
	if m := rttTimeRe.FindStringSubmatch(out); m != nil {
		if v, e := strconv.ParseFloat(m[1], 64); e == nil {
			return v, true
		}
	}
	return 0, false
}
