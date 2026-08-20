// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package profile

import "strings"

// Endpoint returns the first peer endpoint (host:port), or "" if none. Used by
// the measure package to probe server reachability without connecting.
func (c *Config) Endpoint() string {
	for _, p := range c.Peers {
		if p.Endpoint != "" {
			return p.Endpoint
		}
	}
	return ""
}

// EndpointHost returns just the host part of the first peer endpoint.
func (c *Config) EndpointHost() string {
	ep := c.Endpoint()
	if ep == "" {
		return ""
	}
	// Handle IPv6 [::]:port and host:port.
	if strings.HasPrefix(ep, "[") {
		if i := strings.LastIndex(ep, "]"); i > 0 {
			return ep[1:i]
		}
	}
	if i := strings.LastIndex(ep, ":"); i > 0 {
		return ep[:i]
	}
	return ep
}
