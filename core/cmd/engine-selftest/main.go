// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Command engine-selftest proves the autonomous mazzy-core config path works
// with zero external tools: parse a wg-quick .conf, validate it, and render
// the amneziawg-go UAPI — all in-process. It intentionally does NOT bring up a
// TUN device (that needs root/CAP_NET_ADMIN); a separate privileged smoke test
// covers that (backlog C1-8).
package main

import (
	"fmt"
	"os"

	"github.com/mazurovn/mazzy-vpn/core"
	"github.com/mazurovn/mazzy-vpn/core/profile"
)

const sample = `[Interface]
PrivateKey = AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=
Address = 10.8.0.2/32
DNS = 1.1.1.1
MTU = 1420
Jc = 4
Jmin = 40
Jmax = 70
S1 = 50
S2 = 100
H1 = 1
H2 = 2
H3 = 3
H4 = 4
[Peer]
PublicKey = AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = vpn.example.com:51820
PersistentKeepalive = 25
`

func main() {
	c, err := profile.Parse(sample)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse:", err)
		os.Exit(1)
	}
	if problems := profile.Validate(core.AmneziaWG, c); len(problems) != 0 {
		fmt.Fprintln(os.Stderr, "validate:", problems)
		os.Exit(1)
	}
	uapi, err := profile.ToUAPI(core.AmneziaWG, c)
	if err != nil {
		fmt.Fprintln(os.Stderr, "uapi:", err)
		os.Exit(1)
	}
	fmt.Println("mazzy-core autonomous config path OK")
	fmt.Printf("protocol=%s interface=%s mtu=%d peers=%d\n",
		core.AmneziaWG, core.AmneziaWG.Interface(), c.MTU, len(c.Peers))
	fmt.Print("--- UAPI ---\n", uapi)
}
