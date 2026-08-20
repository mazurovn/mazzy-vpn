// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package netadapter enumerates and diagnoses host network interfaces so the
// CLI can let the user pick which physical uplink to run the VPN over (e.g. a
// wired cable vs Wi‑Fi where another VPN like AdGuard is active) and analyze
// what is wrong when connectivity fails.
package netadapter

import (
	"net"
	"sort"
	"strings"
)

// Adapter describes one network interface.
type Adapter struct {
	Name       string   `json:"name"`
	Up         bool     `json:"up"`
	Loopback   bool     `json:"loopback"`
	Virtual    bool     `json:"virtual"` // tun/vpn/docker/bridge etc.
	Wireless   bool     `json:"wireless"`
	HasCarrier bool     `json:"has_carrier"` // link/cable present
	MAC        string   `json:"mac,omitempty"`
	IPv4       []string `json:"ipv4,omitempty"`
	IPv6       []string `json:"ipv6,omitempty"`
	MTU        int      `json:"mtu,omitempty"`
}

// IsPhysical reports whether this is a usable physical uplink (not loopback,
// not a virtual/vpn interface).
func (a Adapter) IsPhysical() bool {
	return !a.Loopback && !a.Virtual
}

// HasRoutableIPv4 reports whether the adapter has a globally routable IPv4
// address (not link-local 169.254.x, which means "cable up but no DHCP lease").
func (a Adapter) HasRoutableIPv4() bool {
	for _, cidr := range a.IPv4 {
		if ip, _, err := net.ParseCIDR(cidr); err == nil {
			if ip.To4() != nil && !ip.IsLinkLocalUnicast() && !ip.IsLoopback() {
				return true
			}
		}
	}
	return false
}

// Kind returns a short human label for the adapter type.
func (a Adapter) Kind() string {
	switch {
	case a.Loopback:
		return "loopback"
	case a.Virtual:
		return "virtual"
	case a.Wireless:
		return "wifi"
	default:
		return "wired"
	}
}

// virtualPrefixes are name prefixes that indicate a non-physical interface.
var virtualPrefixes = []string{
	"lo", "vpn", "tun", "tap", "wg", "awg", "docker", "br-", "veth",
	"virbr", "vmnet", "vboxnet", "zt", "tailscale", "utun", "ppp",
}

func looksVirtual(name string) bool {
	lower := strings.ToLower(name)
	for _, p := range virtualPrefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

func looksWireless(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, "wl") || strings.HasPrefix(lower, "wlan") ||
		strings.HasPrefix(lower, "wlp") || strings.Contains(lower, "wifi")
}

// Lister enumerates interfaces. It is an interface for testability.
type Lister interface {
	Interfaces() ([]net.Interface, error)
	Addrs(iface net.Interface) ([]net.Addr, error)
}

// realLister uses the net package.
type realLister struct{}

func (realLister) Interfaces() ([]net.Interface, error) { return net.Interfaces() }
func (realLister) Addrs(iface net.Interface) ([]net.Addr, error) {
	return iface.Addrs()
}

// List returns all adapters, physical uplinks sorted first.
func List() ([]Adapter, error) {
	return listWith(realLister{})
}

func listWith(l Lister) ([]Adapter, error) {
	ifaces, err := l.Interfaces()
	if err != nil {
		return nil, err
	}
	var out []Adapter
	for _, ifi := range ifaces {
		a := Adapter{
			Name:     ifi.Name,
			Up:       ifi.Flags&net.FlagUp != 0,
			Loopback: ifi.Flags&net.FlagLoopback != 0,
			Virtual:  looksVirtual(ifi.Name),
			Wireless: looksWireless(ifi.Name),
			MTU:      ifi.MTU,
		}
		if ifi.HardwareAddr != nil {
			a.MAC = ifi.HardwareAddr.String()
		}
		// Carrier ~ running flag + has a non-link-local address.
		a.HasCarrier = ifi.Flags&net.FlagRunning != 0
		addrs, _ := l.Addrs(ifi)
		for _, ad := range addrs {
			ipnet, ok := ad.(*net.IPNet)
			if !ok {
				continue
			}
			if v4 := ipnet.IP.To4(); v4 != nil {
				a.IPv4 = append(a.IPv4, ipnet.String())
			} else {
				a.IPv6 = append(a.IPv6, ipnet.String())
			}
		}
		out = append(out, a)
	}
	sort.SliceStable(out, func(i, j int) bool {
		// Physical uplinks with carrier first.
		pi, pj := out[i].IsPhysical(), out[j].IsPhysical()
		if pi != pj {
			return pi
		}
		if out[i].HasCarrier != out[j].HasCarrier {
			return out[i].HasCarrier
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// PhysicalUplinks returns only usable physical interfaces that are up.
func PhysicalUplinks(adapters []Adapter) []Adapter {
	var out []Adapter
	for _, a := range adapters {
		if a.IsPhysical() && a.Up {
			out = append(out, a)
		}
	}
	return out
}

// Recommend returns the best uplink to use: a wired interface with carrier and
// an IPv4 address is preferred over Wi‑Fi. Returns the adapter and a reason.
func Recommend(adapters []Adapter) (Adapter, string, bool) {
	// Prefer interfaces with a routable IPv4; fall back to any IPv4 (link-local)
	// only if nothing better exists.
	var wiredRoutable, wifiRoutable, wiredAny, wifiAny *Adapter
	for i := range adapters {
		a := &adapters[i]
		if !a.IsPhysical() || !a.Up || len(a.IPv4) == 0 {
			continue
		}
		routable := a.HasRoutableIPv4()
		if a.Wireless {
			if routable && wifiRoutable == nil {
				wifiRoutable = a
			}
			if wifiAny == nil {
				wifiAny = a
			}
		} else {
			if routable && wiredRoutable == nil {
				wiredRoutable = a
			}
			if wiredAny == nil {
				wiredAny = a
			}
		}
	}
	switch {
	case wiredRoutable != nil:
		return *wiredRoutable, "wired uplink with a routable IPv4 (preferred; avoids Wi‑Fi VPNs)", true
	case wifiRoutable != nil:
		return *wifiRoutable, "wireless uplink with a routable IPv4", true
	case wiredAny != nil:
		return *wiredAny, "wired uplink, but only a link-local IPv4 (no DHCP lease?)", true
	case wifiAny != nil:
		return *wifiAny, "wireless uplink, but only a link-local IPv4", true
	default:
		return Adapter{}, "no physical uplink with an IPv4 address is up", false
	}
}
