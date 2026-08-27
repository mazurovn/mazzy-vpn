// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package profile parses and validates WireGuard/AmneziaWG wg-quick .conf
// files into a structured model and the wireguard-go UAPI representation.
//
// This closes audit finding R2/C1-6a: amneziawg-go itself does NOT parse
// wg-quick .conf (it only accepts UAPI via IpcSet). The interface-level fields
// (Address/DNS/MTU) and routing are our responsibility and live here.
package profile

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// Peer is a single [Peer] section of a wg-quick config.
type Peer struct {
	PublicKey           string
	PresharedKey        string
	Endpoint            string
	AllowedIPs          []string
	PersistentKeepalive int
}

// AmneziaParams holds the AmneziaWG obfuscation parameters (Jc/Jmin/Jmax,
// S1/S2, H1..H4, and the newer S3/S4 and i1..i5 magic-packet chains).
//
// Values are kept as opaque strings keyed by their lowercase UAPI name, because
// amneziawg-go's IpcSet is the authoritative parser: H1..H4 are integer RANGES
// ("127765534-127831069"), i1..i5 are obfuscation-chain strings, and S1..S4/Jc
// are integers. Storing them verbatim lets us pass them straight through to
// UAPI without re-implementing (and risking divergence from) that parser.
type AmneziaParams struct {
	// values maps a UAPI key (jc/jmin/jmax/s1..s4/h1..h4/i1..i5) to its raw
	// string value exactly as written in the .conf.
	values map[string]string
}

// amneziaKeys is the full set of AmneziaWG obfuscation keys we accept in an
// [Interface] section and forward to UAPI. It includes the classic S/J/H/i
// parameters plus the newer AmneziaWG 1.5+ fields that amneziawg-go v3 accepts
// via UAPI (header_protection_key, content_padding_addition, random_trailers,
// disable_cookies) so a forward config is passed through instead of rejected
// (audit N2).
var amneziaKeys = map[string]bool{
	"jc": true, "jmin": true, "jmax": true,
	"s1": true, "s2": true, "s3": true, "s4": true,
	"h1": true, "h2": true, "h3": true, "h4": true,
	"i1": true, "i2": true, "i3": true, "i4": true, "i5": true,
	"header_protection_key": true, "content_padding_addition": true,
	"random_trailers": true, "disable_cookies": true,
}

// Get returns the raw value for a UAPI key and whether it was present.
func (a AmneziaParams) Get(key string) (string, bool) {
	v, ok := a.values[key]
	return v, ok
}

// Has reports whether an AmneziaWG key was present.
func (a AmneziaParams) Has(key string) bool {
	_, ok := a.values[key]
	return ok
}

// Config is a parsed wg-quick [Interface] + [Peer] configuration.
type Config struct {
	// [Interface]
	PrivateKey string
	Addresses  []string // interface-level, applied by our tun/routes code
	DNS        []string // interface-level, applied by our dns code
	MTU        int
	ListenPort int
	FwMark     uint32
	Amnezia    AmneziaParams // only meaningful for AmneziaWG

	Peers []Peer

	// HasAmneziaFields reports whether any Jc/S1/H1... field was present.
	HasAmneziaFields bool
}

// unsafeInterfaceKeys are wg-quick keys that execute commands. We reject them:
// mazzy-core performs routing/DNS natively and never runs profile scripts.
// Parity with the bash PreUp/PostUp/PreDown/PostDown rejection.
var unsafeInterfaceKeys = []string{"preup", "postup", "predown", "postdown"}

// ParseError describes a config parsing/validation failure.
type ParseError struct{ Msg string }

func (e *ParseError) Error() string { return e.Msg }

func perr(format string, a ...any) error { return &ParseError{Msg: fmt.Sprintf(format, a...)} }

// Parse reads a wg-quick style config. It is strict: unknown executable
// directives are rejected rather than ignored.
func Parse(text string) (*Config, error) {
	cfg := &Config{}
	var section string
	var cur *Peer

	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			if section == "peer" {
				cfg.Peers = append(cfg.Peers, Peer{})
				cur = &cfg.Peers[len(cfg.Peers)-1]
			}
			continue
		}
		key, val, ok := splitKV(line)
		if !ok {
			return nil, perr("malformed line: %q", line)
		}
		lkey := strings.ToLower(key)

		switch section {
		case "interface":
			for _, u := range unsafeInterfaceKeys {
				if lkey == u {
					return nil, perr("config contains executable directive %s", key)
				}
			}
			if err := cfg.setInterface(lkey, val); err != nil {
				return nil, err
			}
		case "peer":
			if cur == nil {
				return nil, perr("peer key %q before [Peer] section", key)
			}
			if err := setPeer(cur, lkey, val); err != nil {
				return nil, err
			}
		default:
			return nil, perr("key %q outside any section", key)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, perr("read config: %v", err)
	}
	return cfg, nil
}

func (c *Config) setInterface(key, val string) error {
	switch key {
	case "privatekey":
		c.PrivateKey = val
	case "address":
		c.Addresses = append(c.Addresses, splitList(val)...)
	case "dns":
		c.DNS = append(c.DNS, splitList(val)...)
	case "mtu":
		n, err := atoiField("MTU", val)
		if err != nil {
			return err
		}
		c.MTU = n
	case "listenport":
		n, err := atoiField("ListenPort", val)
		if err != nil {
			return err
		}
		c.ListenPort = n
	case "fwmark":
		n, err := parseFwMark(val)
		if err != nil {
			return err
		}
		c.FwMark = n
	default:
		if amneziaKeys[key] {
			c.HasAmneziaFields = true
			if c.Amnezia.values == nil {
				c.Amnezia.values = map[string]string{}
			}
			if val == "" {
				return perr("empty AmneziaWG value for %q", key)
			}
			c.Amnezia.values[key] = val
			return nil
		}
		return perr("unknown [Interface] key %q", key)
	}
	return nil
}

func setPeer(p *Peer, key, val string) error {
	switch key {
	case "publickey":
		p.PublicKey = val
	case "presharedkey":
		p.PresharedKey = val
	case "endpoint":
		p.Endpoint = val
	case "allowedips":
		p.AllowedIPs = append(p.AllowedIPs, splitList(val)...)
	case "persistentkeepalive":
		n, err := atoiField("PersistentKeepalive", val)
		if err != nil {
			return err
		}
		p.PersistentKeepalive = n
	default:
		return perr("unknown [Peer] key %q", key)
	}
	return nil
}

func splitKV(line string) (key, val string, ok bool) {
	i := strings.IndexByte(line, '=')
	if i < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), true
}

func splitList(val string) []string {
	parts := strings.Split(val, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func atoiField(name, val string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		return 0, perr("invalid %s value %q", name, val)
	}
	return n, nil
}

func parseFwMark(val string) (uint32, error) {
	v := strings.TrimSpace(val)
	base := 10
	if strings.HasPrefix(v, "0x") || strings.HasPrefix(v, "0X") {
		v = v[2:]
		base = 16
	}
	n, err := strconv.ParseUint(v, base, 32)
	if err != nil {
		return 0, perr("invalid FwMark value %q", val)
	}
	return uint32(n), nil
}
