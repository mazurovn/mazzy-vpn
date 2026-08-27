// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package wireguard runs the AmneziaWG/WireGuard crypto + TUN data path by
// embedding amneziawg-go as a library (ADR-0003). It does NOT shell out to
// awg-quick/wg-quick.
//
// Scope of THIS package (per audit R1): the crypto engine and TUN device only.
// Interface addressing, routes, DNS, policy routing and the kill-switch are
// applied by the sibling core/tun, core/routes, core/dns and core/guard
// packages against the interface this engine creates.
package wireguard

import (
	"fmt"

	"github.com/amnezia-vpn/amneziawg-go/v3/conn"
	"github.com/amnezia-vpn/amneziawg-go/v3/device"
	"github.com/amnezia-vpn/amneziawg-go/v3/tun"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/profile"
)

// Engine is a running AmneziaWG/WireGuard userspace device.
type Engine struct {
	proto     core.Protocol
	name      string
	tun       tun.Device
	dev       *device.Device
	Interface string // actual kernel interface name
	MTU       int
}

// LogLevel controls the embedded device logger verbosity.
type LogLevel int

const (
	LogSilent  LogLevel = LogLevel(device.LogLevelSilent)
	LogError   LogLevel = LogLevel(device.LogLevelError)
	LogVerbose LogLevel = LogLevel(device.LogLevelVerbose)
)

// Up creates the TUN interface for proto, configures the crypto engine from
// cfg via UAPI, and brings the device up. The caller is responsible for
// applying addressing/routes/DNS afterward.
//
// mark is the socket fwmark to apply; it MUST equal the routing table used by
// core/routes for a full-tunnel config (audit G1). Pass 0 for split-tunnel
// configs that do not use policy routing.
func Up(proto core.Protocol, cfg *profile.Config, mark uint32, level LogLevel) (*Engine, error) {
	if proto != core.AmneziaWG && proto != core.WireGuard {
		return nil, fmt.Errorf("wireguard engine: unsupported protocol %q", proto)
	}
	name := proto.Interface()
	mtu := cfg.MTU
	if mtu <= 0 {
		mtu = device.DefaultMTU
	}

	tdev, err := tun.CreateTUN(name, mtu)
	if err != nil {
		return nil, fmt.Errorf("create TUN %q: %w", name, err)
	}
	realName, err := tdev.Name()
	if err == nil && realName != "" {
		name = realName
	}

	logger := device.NewLogger(int(level), fmt.Sprintf("(%s) ", name))
	dev := device.NewDevice(tdev, conn.NewDefaultBind(), logger)

	uapiConf, err := profile.ToUAPIWithMark(proto, cfg, mark)
	if err != nil {
		dev.Close()
		return nil, fmt.Errorf("render UAPI: %w", err)
	}
	if err := dev.IpcSet(uapiConf); err != nil {
		dev.Close()
		return nil, fmt.Errorf("apply config: %w", err)
	}
	if err := dev.Up(); err != nil {
		dev.Close()
		return nil, fmt.Errorf("bring device up: %w", err)
	}

	return &Engine{
		proto:     proto,
		name:      name,
		tun:       tdev,
		dev:       dev,
		Interface: name,
		MTU:       mtu,
	}, nil
}

// Wait blocks until the device stops (e.g. via Down or a fatal error).
func (e *Engine) Wait() {
	if e.dev != nil {
		e.dev.Wait()
	}
}

// Down tears down the crypto engine and TUN interface.
func (e *Engine) Down() error {
	if e.dev != nil {
		e.dev.Close() // also closes the TUN device
		e.dev = nil
		e.tun = nil
	}
	return nil
}
