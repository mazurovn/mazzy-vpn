// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package livecheck verifies that a VPN connection actually carries traffic
// after it is brought up, and provides a compact live status snapshot for the
// CLI dashboard. It answers the user's real question: "am I actually
// connected and protected, or did it just create an interface?"
package livecheck

import (
	"context"
	"io"
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
	// ProbeURL returns the caller's public IP as text via the tunnel. When set it
	// is tried FIRST; the ProbeURLs fallbacks follow. Kept for compatibility.
	ProbeURL string
	// ProbeURLs are independent plain-text IP echo endpoints tried in order until
	// one answers. A single provider being blocked/slow (common under censorship)
	// must not read as "egress lost" — that false negative previously sent the
	// daemon into an endless reconnect storm against a healthy tunnel.
	ProbeURLs []string
	// Timeout bounds each network probe (default 5s).
	Timeout time.Duration
	// httpGet is overridable in tests; nil uses a bound HTTP client.
	httpGet func(ctx context.Context, iface, url string) (string, error)
	// linkUp is overridable in tests; nil uses net.InterfaceByName.
	linkUp func(iface string) bool
}

// DefaultProbeURL is a plain-text IP echo endpoint.
const DefaultProbeURL = "https://api.ipify.org"

// DefaultProbeURLs are the fallback egress probes, ordered by preference. They
// are operated by unrelated parties so a single block/outage cannot blind the
// health check. Each must return the caller's IP as plain text.
var DefaultProbeURLs = []string{
	DefaultProbeURL,
	"https://checkip.amazonaws.com",
	"https://icanhazip.com",
}

// New returns a Checker with sensible defaults.
func New() *Checker {
	return &Checker{ProbeURLs: DefaultProbeURLs, Timeout: 5 * time.Second}
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
		// Surface the REAL probe failure (DNS error, timeout, reset) instead of a
		// generic phrase: the daemon log and dashboard show this string, and "no
		// traffic yet" hid every actionable detail from the user.
		s.Reason = "no traffic through tunnel yet"
		if err != nil {
			s.Reason = "egress probe failed: " + err.Error()
		}
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

// probeURLs returns the ordered probe list: explicit ProbeURL first (test/dev
// override), then the configured or default fallbacks, deduplicated.
func (c *Checker) probeURLs() []string {
	urls := c.ProbeURLs
	if len(urls) == 0 {
		urls = DefaultProbeURLs
	}
	if c.ProbeURL == "" {
		return urls
	}
	out := []string{c.ProbeURL}
	for _, u := range urls {
		if u != c.ProbeURL {
			out = append(out, u)
		}
	}
	return out
}

// egress fetches the public IP through the given interface, trying each probe
// endpoint in order until one returns a valid IP. Only when ALL endpoints fail
// is the egress considered lost; the last error is returned for the Reason.
//
// The WHOLE chain shares one bounded budget (2× the per-probe timeout): a
// fast-fail first endpoint (RST/DNS error) leaves time for fallbacks, while a
// hanging first endpoint cannot stretch a health tick to len(urls)×timeout and
// wreck the caller's cadence.
func (c *Checker) egress(ctx context.Context, iface string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*c.timeout())
	defer cancel()
	// Collect EVERY endpoint's failure, not just the last one: "all three
	// probes failed, each like this" is a far stronger diagnostic than a single
	// error that hides whether the others were even tried.
	var errs []string
	for _, u := range c.probeURLs() {
		ip, err := c.egressVia(ctx, iface, u)
		if err == nil && ip != "" {
			return ip, nil
		}
		if err == nil {
			err = errBadIP
		}
		errs = append(errs, probeHost(u)+": "+compactNetErr(err.Error()))
		if ctx.Err() != nil {
			break
		}
	}
	if len(errs) == 0 {
		return "", errBadIP
	}
	return "", lcErr(strings.Join(errs, "; "))
}

// probeHost reduces a probe URL to its host for compact error summaries.
func probeHost(u string) string {
	s := strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[:i]
	}
	return s
}

// compactNetErr strips the noisy Go net/http prefixes ('Get "https://…": ')
// that would otherwise repeat the host we already print.
func compactNetErr(msg string) string {
	if i := strings.LastIndex(msg, `": `); i >= 0 && strings.HasPrefix(msg, "Get \"") {
		return msg[i+3:]
	}
	return msg
}

// egressVia performs one probe against a single endpoint.
func (c *Checker) egressVia(ctx context.Context, iface, probeURL string) (string, error) {
	if c.httpGet != nil {
		return c.httpGet(ctx, iface, probeURL)
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
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, probeURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// Read the full bounded body: a single Read() can truncate an IP that spans
	// two TLS/chunked segments (e.g. "203.0.113." + "9").
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 128))
	ip := strings.TrimSpace(string(body))
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
