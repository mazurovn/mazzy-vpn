// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package core defines the shared types for mazzy-core, the autonomous VPN
// engine used by all three MAZZY_VPN products (CLI, Desktop, Android).
//
// Design: docs/AI_NATIVE_GO_VPN/ (ADR-0001..0004).
package core

// Protocol is a supported VPN protocol.
type Protocol string

const (
	AmneziaWG Protocol = "amneziawg"
	WireGuard Protocol = "wireguard"
	OpenVPN   Protocol = "openvpn"
	L2TP      Protocol = "l2tp"
)

// CanonicalProtocol normalizes a user-supplied alias to a canonical Protocol.
// Parity with the bash canonical_protocol().
func CanonicalProtocol(s string) (Protocol, bool) {
	switch s {
	case "amneziawg", "amnezia", "awg":
		return AmneziaWG, true
	case "wireguard", "wire", "wg":
		return WireGuard, true
	case "openvpn", "open", "ovpn":
		return OpenVPN, true
	case "l2tp", "ltp":
		return L2TP, true
	default:
		return "", false
	}
}

// Title returns the human-readable protocol name. Parity with protocol_title().
func (p Protocol) Title() string {
	switch p {
	case AmneziaWG:
		return "AmneziaWG"
	case WireGuard:
		return "WireGuard"
	case OpenVPN:
		return "OpenVPN"
	case L2TP:
		return "L2TP/IPsec"
	default:
		return string(p)
	}
}

// Interface returns the deterministic interface name for a protocol.
// Parity with interface_for_protocol().
func (p Protocol) Interface() string {
	switch p {
	case AmneziaWG:
		return "vpnaw0"
	case WireGuard:
		return "vpnwg0"
	case OpenVPN:
		return "vpnovpn0"
	default:
		return ""
	}
}

// ManagedInterfaces returns every VPN interface name this tool may create,
// across all supported protocols. It is the single source of truth so cleanup
// paths (recover/disconnect) and interface detection never drift from the
// per-protocol Interface() naming.
func ManagedInterfaces() []string {
	return []string{
		AmneziaWG.Interface(),
		WireGuard.Interface(),
		OpenVPN.Interface(),
	}
}

// DesiredState is the persisted connection intent. Parity with DESIRED state.
type DesiredState string

const (
	DesiredUp   DesiredState = "up"
	DesiredDown DesiredState = "down"
)

// Mode is the runtime mode of a connection.
type Mode string

const (
	ModeNormal Mode = "normal"
	ModeTest   Mode = "test"
)
