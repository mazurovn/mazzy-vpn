# FAQ: installation, profiles, Doctor and releases

## Is this a VPN provider?

No. Mazzy VPN manages your profiles and connections; it does not issue an
account or provide a VPN server.

## Does Desktop require a separate CLI install?

Linux Desktop 0.2 bundles a compatible engine and installer. No prior manual
CLI install is required. System changes run only after standard OS
authorization.

## What can I import?

AmneziaWG, WireGuard, OpenVPN and NetworkManager L2TP/IPsec. Backend support is
platform-specific; recognizing a file does not make that protocol functional
on Windows, macOS or mobile.

## Where are profiles stored?

On Linux they are root-protected under `/etc/vpnctl/profiles`. The frontend
receives sanitized metadata, not private keys.

## What does Doctor check?

Versions, dependencies, profiles, systemd, desired state, VPN interface and
connectivity. Desktop 0.2 shows the complete result and bounded logs. Repair is
a separate, confirmed action.

## Is there telemetry?

There is no mandatory account, analytics or telemetry. Details:
[PRIVACY.md](https://github.com/mazurovn/mazzy-vpn/blob/main/PRIVACY.md).

## Where can I download Windows, macOS, Android or iOS?

Windows/macOS artifacts are UI previews and must not be used to protect
traffic. Android/iOS are planned. Follow the
[release gates](https://github.com/mazurovn/mazzy-vpn/wiki/Releases-and-Roadmap).

Full FAQ:
https://github.com/mazurovn/mazzy-vpn/wiki/FAQ
