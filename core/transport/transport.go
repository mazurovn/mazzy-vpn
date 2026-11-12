// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package transport picks a UDP path the embedded AmneziaWG/WireGuard engine
// can actually dial, and classifies networks where that path is filtered.
//
// It does not invent an HTTP/2, QUIC, or TLS tunnel. AdGuard VPN CLI speaks
// those (set-protocol http2|quic|auto). This engine's bind is UDP only, so the
// expansion is other UDP ports on the same peer — and only a completed
// WireGuard handshake counts as success.
package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// FallbackUDPPorts are tried, in order, when the configured UDP port does not
// complete a handshake. 443 is the UDP port corporate firewalls usually leave
// open for HTTP/3 and the port AdGuard's QUIC mode uses. 51820 and 1194 are
// the well-known WireGuard and OpenVPN UDP ports. 123 and 53 are NTP and DNS,
// which many filters allow. The server must listen on a port for it to work.
var FallbackUDPPorts = []int{443, 51820, 123, 53, 1194}

// DefaultHandshakeWait is how long one UDP port is given to complete a
// handshake before the next candidate is tried.
const DefaultHandshakeWait = 2 * time.Second

// Candidate is one UDP endpoint the engine can retarget a peer to.
type Candidate struct {
	Endpoint string
	Port     int
	Reason   string // "configured" or "fallback"
}

// Lookup resolves a hostname to IP strings. Nil means net.DefaultResolver.
type Lookup func(ctx context.Context, host string) ([]string, error)

// ResolveEndpoint returns host:port with a literal IP. Hostnames are resolved
// once so amneziawg-go's ParseEndpoint (numeric addresses only) accepts them.
func ResolveEndpoint(ctx context.Context, endpoint string, lookup Lookup) (string, error) {
	host, port, err := split(endpoint)
	if err != nil {
		return "", err
	}
	if ip := net.ParseIP(host); ip != nil {
		return net.JoinHostPort(ip.String(), strconv.Itoa(port)), nil
	}
	if lookup == nil {
		lookup = func(ctx context.Context, host string) ([]string, error) {
			cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			return net.DefaultResolver.LookupHost(cctx, host)
		}
	}
	ips, err := lookup(ctx, host)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", host, err)
	}
	ip := preferIPv4(ips)
	if ip == "" {
		return "", fmt.Errorf("resolve %s: no addresses", host)
	}
	return net.JoinHostPort(ip, strconv.Itoa(port)), nil
}

// Candidates lists the configured UDP endpoint first, then FallbackUDPPorts
// on the same host, skipping duplicates. resolved must be an IP:port.
func Candidates(resolved string) ([]Candidate, error) {
	host, port, err := split(resolved)
	if err != nil {
		return nil, err
	}
	if net.ParseIP(host) == nil {
		return nil, fmt.Errorf("endpoint %q: host is not an IP", resolved)
	}
	out := []Candidate{{
		Endpoint: net.JoinHostPort(host, strconv.Itoa(port)),
		Port:     port,
		Reason:   "configured",
	}}
	seen := map[int]bool{port: true}
	for _, p := range FallbackUDPPorts {
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, Candidate{
			Endpoint: net.JoinHostPort(host, strconv.Itoa(p)),
			Port:     p,
			Reason:   "fallback",
		})
	}
	return out, nil
}

// Link is the slice of the running engine Seek needs. SetEndpoint retargets
// the peer and elicits a handshake; LastHandshake is zero when none has
// completed.
type Link interface {
	SetEndpoint(endpoint string) error
	LastHandshake() time.Time
}

// Result is the UDP path Seek settled on.
type Result struct {
	Configured string
	Endpoint   string
	Port       int
	Transport  string
	Tried      []string
	Handshake  bool
}

// Label renders the transport the UI and CLI show next to the protocol name,
// for example "udp/443".
func (r Result) Label() string {
	if r.Transport == "" {
		return ""
	}
	if r.Port <= 0 {
		return r.Transport
	}
	return fmt.Sprintf("%s/%d", r.Transport, r.Port)
}

// FallbackNote explains a handshake that landed on a different port than the
// profile configured. DNS resolution alone does not count.
func (r Result) FallbackNote() string {
	if !r.Handshake || r.Port <= 0 || samePort(r.Configured, r.Endpoint) {
		return ""
	}
	return fmt.Sprintf("handshake on %s; configured %s did not answer", r.Endpoint, r.Configured)
}

// FailureNote explains a missed handshake, including why AdGuard VPN CLI can
// still be up on the same network. Empty when a handshake completed or when
// no port was tried.
func (r Result) FailureNote() string {
	if r.Handshake || len(r.Tried) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"no AmneziaWG/WireGuard handshake after UDP %s. Corporate networks often drop UDP and allow TCP/443. AdGuard VPN CLI then uses HTTP/2 over TLS (adguardvpn-cli config set-protocol http2) instead of QUIC (set-protocol quic, UDP/443); auto switches. This engine cannot speak HTTP/2. A fallback UDP port completes only when the server listens on it.",
		strings.Join(r.Tried, ", "),
	)
}

// Seek tries the configured UDP port, then the fallback ports, until a
// handshake appears or the context ends. perPort <= 0 uses DefaultHandshakeWait.
func Seek(ctx context.Context, link Link, configured, resolved string, perPort time.Duration) Result {
	if perPort <= 0 {
		perPort = DefaultHandshakeWait
	}
	res := Result{Configured: configured, Endpoint: resolved, Transport: "udp"}
	cands, err := Candidates(resolved)
	if err != nil || len(cands) == 0 {
		return res
	}
	res.Port = cands[0].Port
	res.Endpoint = cands[0].Endpoint
	for _, c := range cands {
		if ctx.Err() != nil {
			break
		}
		res.Tried = append(res.Tried, c.Endpoint)
		before := link.LastHandshake()
		if err := link.SetEndpoint(c.Endpoint); err != nil {
			continue
		}
		if waitHandshake(ctx, link, before, perPort) {
			res.Endpoint = c.Endpoint
			res.Port = c.Port
			res.Handshake = true
			return res
		}
	}
	return res
}

func waitHandshake(ctx context.Context, link Link, before time.Time, perPort time.Duration) bool {
	deadline := time.NewTimer(perPort)
	defer deadline.Stop()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		if handshakeAdvanced(before, link.LastHandshake()) {
			return true
		}
		select {
		case <-ctx.Done():
			return handshakeAdvanced(before, link.LastHandshake())
		case <-deadline.C:
			return handshakeAdvanced(before, link.LastHandshake())
		case <-tick.C:
		}
	}
}

func handshakeAdvanced(before, now time.Time) bool {
	if now.IsZero() {
		return false
	}
	if before.IsZero() {
		return true
	}
	return now.After(before)
}

// DialStatus is the outcome of one TCP probe.
type DialStatus string

const (
	DialConnected DialStatus = "connected"
	DialRefused   DialStatus = "refused"
	DialTimeout   DialStatus = "timeout"
	DialError     DialStatus = "error"
)

// Class is a coarse explanation of a UDP VPN failure.
type Class string

const (
	ClassUnknown             Class = ""
	ClassCorporateUDPBlocked Class = "corporate-udp-blocked"
)

// ProbeTCP dials host:port over TCP. A completed handshake or a refusal both
// mean the network delivered the SYN; a timeout does not.
func ProbeTCP(ctx context.Context, host string, port int, timeout time.Duration) DialStatus {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	d := net.Dialer{}
	conn, err := d.DialContext(cctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err == nil {
		_ = conn.Close()
		return DialConnected
	}
	if errors.Is(err, context.DeadlineExceeded) || isTimeout(err) {
		return DialTimeout
	}
	var op *net.OpError
	if errors.As(err, &op) && op.Err != nil {
		msg := op.Err.Error()
		if strings.Contains(msg, "refused") || strings.Contains(msg, "reset") {
			return DialRefused
		}
	}
	if strings.Contains(err.Error(), "refused") || strings.Contains(err.Error(), "reset") {
		return DialRefused
	}
	if cctx.Err() != nil {
		return DialTimeout
	}
	return DialError
}

// TCPReaches reports whether ProbeTCP proved the host is reachable over TCP.
func TCPReaches(status DialStatus) bool {
	return status == DialConnected || status == DialRefused
}

// Classify returns ClassCorporateUDPBlocked when the VPN host is reachable
// over TCP but the UDP VPN path produced no handshake or ICMP answer.
// udpSilent must be observed by the caller; a TCP probe alone is not a VPN test.
func Classify(tcp DialStatus, udpSilent bool) Class {
	if udpSilent && TCPReaches(tcp) {
		return ClassCorporateUDPBlocked
	}
	return ClassUnknown
}

// Host returns the host portion of an endpoint, or "" when it cannot be split.
func Host(endpoint string) string {
	host, _, err := split(endpoint)
	if err != nil {
		return ""
	}
	return host
}

func split(endpoint string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(strings.TrimSpace(endpoint))
	if err != nil {
		return "", 0, fmt.Errorf("endpoint %q: %w", endpoint, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("endpoint %q: bad port", endpoint)
	}
	if host == "" {
		return "", 0, fmt.Errorf("endpoint %q: empty host", endpoint)
	}
	return host, port, nil
}

func preferIPv4(ips []string) string {
	for _, s := range ips {
		ip := net.ParseIP(s)
		if ip != nil && ip.To4() != nil {
			return ip.String()
		}
	}
	for _, s := range ips {
		if ip := net.ParseIP(s); ip != nil {
			return ip.String()
		}
	}
	return ""
}

func samePort(a, b string) bool {
	_, pa, ea := net.SplitHostPort(a)
	_, pb, eb := net.SplitHostPort(b)
	return ea == nil && eb == nil && pa == pb
}

func isTimeout(err error) bool {
	var n net.Error
	return errors.As(err, &n) && n.Timeout()
}
