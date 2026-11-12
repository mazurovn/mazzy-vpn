// mazzy-vpn-desktop: Linux control window for Mazzy VPN (ADR-0004, gio).
// Author: Nik m (@mazurovn). License: AGPL-3.0-or-later.
module github.com/mazurovn/mazzy-vpn/desktop

go 1.26.0

toolchain go1.26.11

require github.com/mazurovn/mazzy-vpn/core v0.0.0

require (
	gioui.org v0.8.0
	gioui.org/shader v1.0.8 // indirect
	github.com/amnezia-vpn/amneziawg-go/v3 v3.1.20260814 // indirect
	github.com/go-text/typesetting v0.2.1 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/exp v0.0.0-20240707233637-46b078467d37 // indirect
	golang.org/x/exp/shiny v0.0.0-20240707233637-46b078467d37 // indirect
	golang.org/x/image v0.18.0 // indirect
	golang.org/x/net v0.60.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
)

replace github.com/mazurovn/mazzy-vpn/core => ../core
