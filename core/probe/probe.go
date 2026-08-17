// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package probe provides the concrete, network-facing health probes used by
// core/health in production. Splitting them out keeps core/health pure and
// deterministic (it depends only on the Probe interface).
//
// Parity with bash health_internet_ok / connection_local_ok: it verifies the
// interface exists, that egress through the tunnel returns a public IP, and —
// for full-tunnel profiles — that the system default egress equals the tunnel
// egress (no split leak).
package probe

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// NetProbe implements core/health.Probe against the real network.
type NetProbe struct {
	Interface string
	// ProbeURL returns the caller's public IP as plain text (e.g. an
	// ip-echo endpoint). Bound-interface request must egress via the tunnel.
	ProbeURL string
	// RequireDefaultEgress is true for full-tunnel (0.0.0.0/0) profiles: the
	// system default egress IP must equal the tunnel egress IP.
	RequireDefaultEgress bool
	// Timeout bounds each probe. Zero means 4s (bash HEALTH_PROBE_TIMEOUT).
	Timeout time.Duration
	// StartupDeadline is the monotonic time until which startup grace applies.
	StartupDeadline time.Time
	// Now is injectable for tests; defaults to time.Now.
	Now func() time.Time
}

func (p *NetProbe) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

func (p *NetProbe) timeout() time.Duration {
	if p.Timeout > 0 {
		return p.Timeout
	}
	return 4 * time.Second
}

// LinkPresent reports whether the VPN interface exists.
func (p *NetProbe) LinkPresent(context.Context) bool {
	if p.Interface == "" {
		return false
	}
	_, err := net.InterfaceByName(p.Interface)
	return err == nil
}

// InStartupGrace reports whether we are still before the startup deadline.
func (p *NetProbe) InStartupGrace(context.Context) bool {
	return p.now().Before(p.StartupDeadline)
}

// InternetOK verifies egress through the tunnel, and (for full-tunnel) that the
// default egress matches. Parity with health_internet_ok.
func (p *NetProbe) InternetOK(ctx context.Context) (bool, string) {
	bound, err := p.publicIP(ctx, p.Interface)
	if err != nil || bound == "" {
		return false, "no internet access through VPN"
	}
	if p.RequireDefaultEgress {
		def, err := p.publicIP(ctx, "")
		if err == nil && def != "" && def != bound {
			return false, "system default egress bypasses VPN"
		}
	}
	return true, ""
}

// publicIP fetches the public IP, optionally binding to iface. An empty iface
// uses the default route.
func (p *NetProbe) publicIP(ctx context.Context, iface string) (string, error) {
	dialer := &net.Dialer{Timeout: p.timeout()}
	if iface != "" {
		if ips, err := interfaceIPv4(iface); err == nil && len(ips) > 0 {
			dialer.LocalAddr = &net.TCPAddr{IP: ips[0]}
		}
	}
	transport := &http.Transport{
		DialContext:         dialer.DialContext,
		DisableKeepAlives:   true,
		TLSHandshakeTimeout: p.timeout(),
	}
	client := &http.Client{Timeout: p.timeout(), Transport: transport}

	cctx, cancel := context.WithTimeout(ctx, p.timeout())
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, p.ProbeURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// Full bounded read: a single Read() can truncate an IP across segments.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 128))
	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return "", errBadIP
	}
	return ip, nil
}

// interfaceIPv4 returns the IPv4 addresses bound to iface.
func interfaceIPv4(name string) ([]net.IP, error) {
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	addrs, err := ifi.Addrs()
	if err != nil {
		return nil, err
	}
	var out []net.IP
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok {
			if v4 := ipnet.IP.To4(); v4 != nil {
				out = append(out, v4)
			}
		}
	}
	return out, nil
}

type probeError string

func (e probeError) Error() string { return string(e) }

const errBadIP probeError = "probe returned a non-IP response"
