// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package netadapter

import (
	"net"
	"testing"
)

// fakeLister returns scripted interfaces.
type fakeLister struct {
	ifaces []net.Interface
	addrs  map[string][]net.Addr
}

func (f *fakeLister) Interfaces() ([]net.Interface, error) { return f.ifaces, nil }
func (f *fakeLister) Addrs(iface net.Interface) ([]net.Addr, error) {
	return f.addrs[iface.Name], nil
}

func mustCIDR(s string) *net.IPNet {
	_, n, _ := net.ParseCIDR(s)
	return n
}

func TestListClassifiesInterfaces(t *testing.T) {
	fl := &fakeLister{
		ifaces: []net.Interface{
			{Name: "lo", Flags: net.FlagUp | net.FlagLoopback | net.FlagRunning},
			{Name: "enp3s0", Flags: net.FlagUp | net.FlagRunning, MTU: 1500},
			{Name: "wlp2s0", Flags: net.FlagUp | net.FlagRunning, MTU: 1500},
			{Name: "vpnaw0", Flags: net.FlagUp | net.FlagRunning},
			{Name: "docker0", Flags: net.FlagUp},
		},
		addrs: map[string][]net.Addr{
			"enp3s0": {mustCIDR("192.168.1.10/24")},
			"wlp2s0": {mustCIDR("192.168.1.20/24")},
		},
	}
	adapters, err := listWith(fl)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Adapter{}
	for _, a := range adapters {
		byName[a.Name] = a
	}
	if !byName["lo"].Loopback {
		t.Error("lo should be loopback")
	}
	if byName["enp3s0"].Kind() != "wired" {
		t.Errorf("enp3s0 kind = %s, want wired", byName["enp3s0"].Kind())
	}
	if byName["wlp2s0"].Kind() != "wifi" {
		t.Errorf("wlp2s0 kind = %s, want wifi", byName["wlp2s0"].Kind())
	}
	if !byName["vpnaw0"].Virtual || !byName["docker0"].Virtual {
		t.Error("vpnaw0 and docker0 should be virtual")
	}
	// Physical uplinks first in sort order.
	if !adapters[0].IsPhysical() {
		t.Errorf("first adapter should be physical, got %s", adapters[0].Name)
	}
}

func TestRecommendPrefersWired(t *testing.T) {
	adapters := []Adapter{
		{Name: "wlp2s0", Up: true, Wireless: true, IPv4: []string{"192.168.1.20/24"}},
		{Name: "enp3s0", Up: true, IPv4: []string{"192.168.1.10/24"}},
	}
	rec, reason, ok := Recommend(adapters)
	if !ok || rec.Name != "enp3s0" {
		t.Fatalf("should recommend wired enp3s0, got %s (%s)", rec.Name, reason)
	}
}

func TestRecommendFallsBackToWifi(t *testing.T) {
	adapters := []Adapter{
		{Name: "wlp2s0", Up: true, Wireless: true, IPv4: []string{"192.168.1.20/24"}},
	}
	rec, _, ok := Recommend(adapters)
	if !ok || rec.Name != "wlp2s0" {
		t.Fatalf("should fall back to wifi, got %s", rec.Name)
	}
}

// TestRecommendPrefersRoutableOverLinkLocal locks the real-world fix: a wired
// cable with only a 169.254.x link-local address (no DHCP lease) must NOT be
// preferred over Wi‑Fi with a real routable IPv4.
func TestRecommendPrefersRoutableOverLinkLocal(t *testing.T) {
	adapters := []Adapter{
		{Name: "enp5s0", Up: true, IPv4: []string{"169.254.163.178/16"}},
		{Name: "wlp3s0", Up: true, Wireless: true, IPv4: []string{"192.168.1.36/24"}},
	}
	rec, reason, ok := Recommend(adapters)
	if !ok || rec.Name != "wlp3s0" {
		t.Fatalf("should prefer routable Wi‑Fi over link-local cable, got %s (%s)", rec.Name, reason)
	}
}

func TestHasRoutableIPv4(t *testing.T) {
	routable := Adapter{IPv4: []string{"192.168.1.10/24"}}
	linkLocal := Adapter{IPv4: []string{"169.254.1.1/16"}}
	if !routable.HasRoutableIPv4() {
		t.Error("192.168.x should be routable")
	}
	if linkLocal.HasRoutableIPv4() {
		t.Error("169.254.x link-local must not be routable")
	}
}

func TestRecommendNoneWhenNoIPv4(t *testing.T) {
	adapters := []Adapter{
		{Name: "enp3s0", Up: true}, // no IPv4
	}
	if _, _, ok := Recommend(adapters); ok {
		t.Fatal("no uplink with IPv4 should yield no recommendation")
	}
}

func TestPhysicalUplinksFiltersVirtualAndDown(t *testing.T) {
	adapters := []Adapter{
		{Name: "enp3s0", Up: true},
		{Name: "vpnaw0", Up: true, Virtual: true},
		{Name: "enp4s0", Up: false},
	}
	up := PhysicalUplinks(adapters)
	if len(up) != 1 || up[0].Name != "enp3s0" {
		t.Fatalf("expected only enp3s0, got %+v", up)
	}
}
