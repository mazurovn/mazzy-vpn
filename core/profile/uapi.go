// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package profile

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/mazurovn/mazzy-vpn/core"
)

// ToUAPI renders the parsed config into the wireguard-go/amneziawg-go UAPI
// "set" string accepted by (*device.Device).IpcSet.
//
// Important detail (verified against amneziawg-go/v3 device/uapi.go): UAPI
// keys are HEX-encoded, while wg-quick .conf files store base64 keys. We
// convert here. Interface-level Address/DNS/MTU are intentionally NOT part of
// UAPI — they are applied by our tun/routes/dns packages.
func ToUAPI(proto core.Protocol, c *Config) (string, error) {
	return ToUAPIWithMark(proto, c, c.FwMark)
}

// ToUAPIWithMark is like ToUAPI but forces a specific socket fwmark. The
// routing layer and the engine MUST use the same effective mark (audit G1),
// otherwise the engine's own encrypted packets match the `not fwmark` policy
// rule and loop back into the tunnel. A mark of 0 emits no fwmark line.
func ToUAPIWithMark(proto core.Protocol, c *Config, mark uint32) (string, error) {
	var b strings.Builder

	priv, err := keyBase64ToHex(c.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("private_key: %w", err)
	}
	fmt.Fprintf(&b, "private_key=%s\n", priv)

	if c.ListenPort > 0 {
		fmt.Fprintf(&b, "listen_port=%d\n", c.ListenPort)
	}
	if mark != 0 {
		fmt.Fprintf(&b, "fwmark=%d\n", mark)
	}

	if proto == core.AmneziaWG {
		if err := writeAmnezia(&b, c.Amnezia); err != nil {
			return "", err
		}
	}

	// Full reconfigure semantics.
	b.WriteString("replace_peers=true\n")

	for i := range c.Peers {
		if err := writePeer(&b, &c.Peers[i]); err != nil {
			return "", fmt.Errorf("peer %d: %w", i, err)
		}
	}
	return b.String(), nil
}

func writePeer(b *strings.Builder, p *Peer) error {
	pub, err := keyBase64ToHex(p.PublicKey)
	if err != nil {
		return fmt.Errorf("public_key: %w", err)
	}
	fmt.Fprintf(b, "public_key=%s\n", pub)

	if p.PresharedKey != "" {
		psk, err := keyBase64ToHex(p.PresharedKey)
		if err != nil {
			return fmt.Errorf("preshared_key: %w", err)
		}
		fmt.Fprintf(b, "preshared_key=%s\n", psk)
	}
	if p.Endpoint != "" {
		fmt.Fprintf(b, "endpoint=%s\n", p.Endpoint)
	}
	if p.PersistentKeepalive > 0 {
		fmt.Fprintf(b, "persistent_keepalive_interval=%d\n", p.PersistentKeepalive)
	}
	for _, aip := range p.AllowedIPs {
		fmt.Fprintf(b, "allowed_ip=%s\n", aip)
	}
	return nil
}

// amneziaUAPIOrder is the deterministic emit order for AmneziaWG fields. The
// newer 1.5+ fields (header_protection_key, content_padding_addition,
// random_trailers, disable_cookies) are emitted last (audit N2) and only when
// present.
var amneziaUAPIOrder = []string{
	"jc", "jmin", "jmax", "s1", "s2", "s3", "s4",
	"h1", "h2", "h3", "h4", "i1", "i2", "i3", "i4", "i5",
	"header_protection_key", "content_padding_addition",
	"random_trailers", "disable_cookies",
}

// writeAmnezia emits each present AmneziaWG field verbatim. amneziawg-go's
// IpcSet is the authoritative parser (H1..H4 ranges, i1..i5 chains, S/J ints),
// so we forward the raw string values unchanged.
func writeAmnezia(b *strings.Builder, a AmneziaParams) error {
	for _, key := range amneziaUAPIOrder {
		if v, ok := a.Get(key); ok {
			fmt.Fprintf(b, "%s=%s\n", key, v)
		}
	}
	return nil
}

// keyBase64ToHex converts a wg-quick base64 X25519 key to the lowercase hex
// form expected by UAPI.
func keyBase64ToHex(b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return "", fmt.Errorf("not valid base64")
	}
	if len(raw) != 32 {
		return "", fmt.Errorf("key must be 32 bytes, got %d", len(raw))
	}
	return hex.EncodeToString(raw), nil
}
