// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

//go:build ignore

// genkey prints a fresh WireGuard X25519 keypair (private then public) as two
// base64 lines. Used by smoke-c1-8.sh so no external `wg genkey` is needed.
package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

func main() {
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		panic(err)
	}
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64
	pub, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		panic(err)
	}
	fmt.Println(base64.StdEncoding.EncodeToString(priv[:]))
	fmt.Println(base64.StdEncoding.EncodeToString(pub))
}
