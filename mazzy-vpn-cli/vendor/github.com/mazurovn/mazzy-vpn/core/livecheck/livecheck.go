// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package livecheck verifies that a VPN connection actually carries traffic
// after it is brought up, and provides a compact live status snapshot for the
// CLI dashboard. It answers the user's real question: "am I actually
// connected and protected, or did it just create an interface?"
package livecheck

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"
)

// Snapshot is a point-in-time view of the connection health.
type Snapshot struct {
	Interface    string `json:"interface"`
	LinkUp       bool   `json:"link_up"`
	EgressIP     string `json:"egress_ip,omitempty"`
	EgressOK     bool   `json:"egress_ok"`
	HandshakeAge int64  `json:"handshake_age_s,omitempty"`
	Reason       string `json:"reason,omitempty"`
	At           string `json:"at"`
}

// Protected reports whether the tunnel is up AND carrying traffic.
func (s Snapshot) Protected() bool { return s.LinkUp && s.EgressOK }

// Checker performs live checks. Fields are injectable for tests.
type Checker struct {
	// ProbeURL returns the caller's public IP as text via the tunnel.
	ProbeURL string
	// Timeout bounds each network probe (default 5s).
	Timeout time.Duration
	// httpGet is overridable in tests; nil uses a bound HTTP client.
	httpGet func(ctx context.Context, iface, url string) (string, error)
	// linkUp is overridable in tests; nil uses net.InterfaceByName.
	linkUp func(iface string) bool
}

// DefaultProbeURL is a plain-text IP echo endpoint.
const DefaultProbeURL = "https://api.ipify.org"

// New returns a Checker with sensible defaults.
func New() *Checker {
	return &Checker{ProbeURL: DefaultProbeURL, Timeout: 5 * time.Second}
}

func (c *Checker) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 5 * time.Second
}

// Check produces a Snapshot for the given interface. It verifies the link
// exists and that an HTTP request bound to the tunnel returns a public IP
// (proof of real egress).
func (c *Checker) Check(ctx context.Context, iface string) Snapshot {
	s := Snapshot{Interface: iface, At: time.Now().Format(time.RFC3339)}

	if !c.linkPresent(iface) {
		s.Reason = "interface not present"
		return s
	}
	s.LinkUp = true

	ip, err := c.egress(ctx, iface)
	if err != nil || ip == "" {
		s.Reason = "no traffic through tunnel yet"
		return s
	}
	s.EgressIP = ip
	s.EgressOK = true
	return s
}

// WaitProtected polls Check until the connection is protected or the deadline
// passes. It returns the final snapshot. Useful right after connect to confirm
// the tunnel works before telling the user it is up.
func (c *Checker) WaitProtected(ctx context.Context, iface string, timeout time.Duration) Snapshot {
	deadline := time.Now().Add(timeout)
	var last Snapshot
	for {
		last = c.Check(ctx, iface)
		if last.Protected() {
			return last
		}
		if time.Now().After(deadline) {
			return last
		}
		select {
		case <-ctx.Done():
			return last
		case <-time.After(time.Second):
		}
	}
}

func (c *Checker) linkPresent(iface string) bool {
	if c.linkUp != nil {
		return c.linkUp(iface)
	}
	if iface == "" {
		return false
	}
	_, err := net.InterfaceByName(iface)
	return err == nil
}

// egress fetches the public IP through the given interface.
func (c *Checker) egress(ctx context.Context, iface string) (string, error) {
	if c.httpGet != nil {
		return c.httpGet(ctx, iface, c.ProbeURL)
	}
	dialer := &net.Dialer{Timeout: c.timeout()}
	if ips, err := ifaceIPv4(iface); err == nil && len(ips) > 0 {
		dialer.LocalAddr = &net.TCPAddr{IP: ips[0]}
	}
	tr := &http.Transport{
		DialContext:         dialer.DialContext,
		DisableKeepAlives:   true,
		TLSHandshakeTimeout: c.timeout(),
	}
	client := &http.Client{Timeout: c.timeout(), Transport: tr}
	cctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, c.ProbeURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	buf := make([]byte, 64)
	n, _ := resp.Body.Read(buf)
	ip := strings.TrimSpace(string(buf[:n]))
	if net.ParseIP(ip) == nil {
		return "", errBadIP
	}
	return ip, nil
}

func ifaceIPv4(name string) ([]net.IP, error) {
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

type lcErr string

func (e lcErr) Error() string { return string(e) }

const errBadIP lcErr = "probe returned a non-IP response"
