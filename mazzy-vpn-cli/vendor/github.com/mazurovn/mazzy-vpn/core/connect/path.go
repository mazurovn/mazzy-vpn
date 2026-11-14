// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package connect

import (
	"time"

	"github.com/mazurovn/mazzy-vpn/core/engine/wireguard"
	"github.com/mazurovn/mazzy-vpn/core/profile"
)

// peerLink adapts a running engine to transport.Link.
type peerLink struct {
	eng *wireguard.Engine
	pub string
	ka  int
}

func (p peerLink) SetEndpoint(endpoint string) error {
	return p.eng.SetPeerEndpoint(p.pub, endpoint, p.ka)
}

func (p peerLink) LastHandshake() time.Time {
	got, err := p.eng.LastHandshake(p.pub)
	if err != nil {
		return time.Time{}
	}
	return got
}

// withEndpoint returns a shallow copy whose first endpoint is replaced.
// The caller's config is left unchanged.
func withEndpoint(cfg *profile.Config, endpoint string) *profile.Config {
	clone := *cfg
	clone.Peers = append([]profile.Peer(nil), cfg.Peers...)
	for i := range clone.Peers {
		if clone.Peers[i].Endpoint != "" {
			clone.Peers[i].Endpoint = endpoint
			break
		}
	}
	return &clone
}

func endpointPeer(cfg *profile.Config) (pub string, keepalive int, ok bool) {
	for _, p := range cfg.Peers {
		if p.Endpoint != "" && p.PublicKey != "" {
			return p.PublicKey, p.PersistentKeepalive, true
		}
	}
	return "", 0, false
}
