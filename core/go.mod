// mazzy-core: shared autonomous VPN engine for MAZZY_VPN.
// See docs/AI_NATIVE_GO_VPN/ for the design (ADR-0001..0004).
//
// C1-1a: pin the Go toolchain. amneziawg-go/v3 requires >= 1.25.
// The module is go 1.26 so the stdlib and golang.org/x/net fixes apply.
module github.com/mazurovn/mazzy-vpn/core

go 1.26.0

toolchain go1.26.11

require (
	github.com/amnezia-vpn/amneziawg-go/v3 v3.1.20260814
	golang.org/x/sys v0.48.0
)

require (
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/net v0.60.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
)
