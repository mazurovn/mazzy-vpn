// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package diagnose performs root-cause analysis of VPN problems. Unlike doctor
// (which lists independent checks), diagnose correlates signals — network,
// uplink, tunnel, DNS, egress, conflicting VPNs — into a ranked list of likely
// problems, each with a plain explanation and a concrete fix. It answers the
// user's real question: "what exactly is wrong and how do I fix it?"
package diagnose

import "github.com/mazurovn/mazzy-vpn/core/transport"

// Signal is an observed fact fed into the analyzer. All fields are optional;
// unknown values are left at their zero value and treated as "not observed".
type Signal struct {
	HasUplink        bool // a physical uplink with a routable IPv4 is up
	UplinkName       string
	InternetOK       bool   // plain internet works (DNS + HTTP off-tunnel)
	TunnelIface      string // "" if no Mazzy tunnel
	TunnelLinkUp     bool
	EgressOK         bool // traffic actually flows through the tunnel
	EgressIP         string
	DNSOK            bool   // DNS resolves
	ConflictVPN      string // e.g. "tun0" (AdGuard) if another VPN is active
	ServerAlive      bool   // the selected server answered ICMP
	ServerName       string
	AnyServerAlive   bool // at least one managed server is alive
	ProfilesImported int
	// TCPReachesHost is true when TCP/443 to the VPN host connected or was
	// refused (the SYN was delivered). A timeout leaves it false.
	TCPReachesHost bool
	// TransportClass is transport.ClassCorporateUDPBlocked when the caller
	// already correlated a silent UDP path with a reachable TCP/443.
	TransportClass string
}

// Severity of a problem.
type Severity int

const (
	Info Severity = iota
	Warn
	Critical
)

func (s Severity) String() string {
	switch s {
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Critical:
		return "CRITICAL"
	default:
		return "?"
	}
}

// Problem is one diagnosed issue with a fix.
type Problem struct {
	Severity Severity `json:"-"`
	Level    string   `json:"level"`
	Title    string   `json:"title"`
	Cause    string   `json:"cause"`
	Fix      string   `json:"fix"`
}

// Report is the ranked analysis.
type Report struct {
	Problems []Problem `json:"problems"`
	Summary  string    `json:"summary"`
}

// Healthy reports whether nothing critical was found.
func (r *Report) Healthy() bool {
	for _, p := range r.Problems {
		if p.Severity == Critical {
			return false
		}
	}
	return true
}

// Analyze correlates signals into ranked problems. The ordering is by root
// cause: no internet → no uplink → conflicting VPN → no servers → tunnel down →
// no egress → DNS. Earlier causes explain later symptoms, so we stop reporting
// downstream symptoms once an upstream root cause is found.
func Analyze(s Signal) *Report {
	r := &Report{}
	add := func(sev Severity, title, cause, fix string) {
		r.Problems = append(r.Problems, Problem{Severity: sev, Level: sev.String(), Title: title, Cause: cause, Fix: fix})
	}

	// --- Root cause 1: no physical uplink ---
	if !s.HasUplink {
		add(Critical, "No usable internet uplink",
			"No wired/Wi‑Fi interface with a routable IPv4 address is up. Nothing can connect.",
			"Plug in a cable or connect Wi‑Fi, then: mazzy-vpn adapters")
		r.Summary = "No uplink — connect to a network first."
		return r
	}

	// --- Root cause 2: no plain internet even off-tunnel ---
	if !s.InternetOK && s.TunnelIface == "" {
		add(Critical, "No internet on the uplink",
			"The uplink "+or(s.UplinkName, "interface")+" is up but cannot reach the internet (DNS/HTTP fails off-tunnel).",
			"Check the router/Wi‑Fi login (captive portal?). Verify with: mazzy-vpn netdiag")
		r.Summary = "Uplink up but no internet — check the network/router."
		return r
	}

	// --- Root cause 3: no managed servers / none alive ---
	if s.ProfilesImported == 0 {
		add(Critical, "No VPN profiles imported",
			"There are no managed profiles to connect to.",
			"Import your configs: mazzy-vpn import <folder>")
		r.Summary = "Import VPN profiles first."
		return r
	}
	// A live tunnel is better evidence than ICMP. Servers that drop ping must
	// not hide a connection that is already carrying traffic.
	if !s.AnyServerAlive && !(s.TunnelLinkUp && s.EgressOK) {
		if s.TCPReachesHost || s.TransportClass == string(transport.ClassCorporateUDPBlocked) {
			add(Critical, "UDP is filtered; TCP/443 still reaches the VPN host",
				"The VPN host answered TCP/443 (connected or refused), so it is reachable, but no imported server answered ICMP. Corporate firewalls often allow TCP/443 and drop other UDP. AdGuard VPN CLI uses HTTP/2 over TLS on TCP/443 (adguardvpn-cli config set-protocol http2) or QUIC on UDP/443 (set-protocol quic); auto switches when UDP is blocked. This engine speaks AmneziaWG/WireGuard over UDP only. On connect it also tries UDP 443, 51820, 123, 53 and 1194, which work only if the server listens there. HTTP/2 is not implemented.",
				"If you control the server, also listen on UDP/443 (or DNAT it to the WireGuard port) and reconnect. mazzy-vpn up tries those UDP ports automatically.")
			r.Summary = "TCP/443 reaches the server; UDP VPN does not. AdGuard CLI's HTTP/2 fallback is not available here."
			return r
		}
		add(Critical, "No live VPN server",
			"None of the imported servers answered ICMP through the uplink — they may be down, or all blocked.",
			"Re-test: mazzy-vpn test. If all are dead, your provider's servers may be offline or your ISP blocks them.")
		r.Summary = "No reachable server — provider or ISP issue."
		return r
	}

	// --- Symptom: tunnel up but no egress (the reported problem) ---
	if s.TunnelIface != "" {
		if !s.TunnelLinkUp {
			add(Critical, "Tunnel interface missing",
				"State says connected but "+s.TunnelIface+" is not present.",
				"Recover to a clean state: sudo mazzy-vpn recover, then reconnect.")
		} else if !s.EgressOK {
			cause := "The interface " + s.TunnelIface + " is up, but no traffic flows through it."
			fix := "Try another live zone: mazzy-vpn test then sudo mazzy-vpn up <zone>."
			if s.TCPReachesHost || s.TransportClass == string(transport.ClassCorporateUDPBlocked) {
				cause = "The interface " + s.TunnelIface + " is up, but the UDP handshake carried no traffic. TCP/443 to the VPN host is not blocked. AdGuard VPN CLI survives that by falling back from QUIC (UDP/443) to HTTP/2 over TLS (TCP/443). This client tried the configured UDP port and UDP 443, 51820, 123, 53 and 1194."
				fix = "Point the server at one of those UDP ports. This build cannot open an HTTP/2 tunnel."
			} else if !s.ServerAlive && s.ServerName != "" {
				cause = "Server " + s.ServerName + " did not answer ICMP; it may be down or dropping the handshake."
				fix = "Switch to a live server: sudo mazzy-vpn up --best (auto-picks an alive one)."
			}
			add(Critical, "Tunnel up but no egress", cause, fix)
		} else {
			// Tunnel works.
			if s.ConflictVPN != "" {
				add(Warn, "Another VPN is also active",
					"Interface "+s.ConflictVPN+" (e.g. AdGuard) is up alongside the Mazzy tunnel; routing may be ambiguous.",
					"If Mazzy egress is correct you can ignore this, or disconnect the other VPN.")
			}
			add(Info, "Connection healthy",
				"Tunnel "+s.TunnelIface+" is up and egress "+or(s.EgressIP, "confirmed")+" flows through it.", "")
			r.Summary = "Protected: traffic flows through " + s.TunnelIface + "."
			return r
		}
	}

	// --- Downstream: DNS ---
	if !s.DNSOK {
		add(Warn, "DNS not resolving",
			"Name resolution is failing; sites may not load even if the tunnel is up.",
			"Check the profile's DNS; reconnect. netdiag for details.")
	}

	// --- Advisory: conflicting VPN when not connected ---
	if s.TunnelIface == "" && s.ConflictVPN != "" {
		add(Info, "Another VPN is active",
			"Interface "+s.ConflictVPN+" is up (e.g. AdGuard). Mazzy VPN is not connected.",
			"To use Mazzy VPN: sudo mazzy-vpn up --best (it will take over).")
	}

	if r.Summary == "" {
		if r.Healthy() {
			r.Summary = "No critical problems found."
		} else {
			r.Summary = "Problems found — see fixes above."
		}
	}
	return r
}

func or(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
