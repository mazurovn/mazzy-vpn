// SPDX-License-Identifier: AGPL-3.0-or-later
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
	"strconv"
	"strings"
	"time"

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

// SetPeerEndpoint retargets an existing peer and elicits a handshake.
// endpoint must be a numeric IP:port; amneziawg-go rejects hostnames.
func (e *Engine) SetPeerEndpoint(publicKeyB64, endpoint string, keepalive int) error {
	if e == nil || e.dev == nil {
		return fmt.Errorf("wireguard engine: not running")
	}
	hexKey, err := profile.KeyHex(publicKeyB64)
	if err != nil {
		return fmt.Errorf("wireguard engine: peer key: %w", err)
	}
	return e.dev.IpcSet(profile.RetargetUAPI(hexKey, endpoint, keepalive))
}

// LastHandshake reads the peer's latest handshake time. The zero time means
// the peer has never completed one.
func (e *Engine) LastHandshake(publicKeyB64 string) (time.Time, error) {
	if e == nil || e.dev == nil {
		return time.Time{}, fmt.Errorf("wireguard engine: not running")
	}
	hexKey, err := profile.KeyHex(publicKeyB64)
	if err != nil {
		return time.Time{}, err
	}
	text, err := e.dev.IpcGet()
	if err != nil {
		return time.Time{}, err
	}
	return ParseHandshake(text, hexKey), nil
}

// ParseHandshake finds last_handshake_time_* for pubHex in a UAPI get blob.
// A missing peer or a 0/0 timestamp is the zero time.
func ParseHandshake(uapi, pubHex string) time.Time {
	want := strings.ToLower(strings.TrimSpace(pubHex))
	var current string
	var sec, nsec int64
	var seen bool
	flush := func() time.Time {
		if !seen || current != want || (sec == 0 && nsec == 0) {
			return time.Time{}
		}
		return time.Unix(sec, nsec)
	}
	for _, line := range strings.Split(uapi, "\n") {
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "public_key":
			if current == want && seen {
				return flush()
			}
			current = strings.ToLower(val)
			sec, nsec, seen = 0, 0, false
		case "last_handshake_time_sec":
			if current == want {
				sec, _ = strconv.ParseInt(val, 10, 64)
				seen = true
			}
		case "last_handshake_time_nsec":
			if current == want {
				nsec, _ = strconv.ParseInt(val, 10, 64)
				seen = true
			}
		}
	}
	return flush()
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
