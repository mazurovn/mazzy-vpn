# Contributing to Mazzy VPN

Thank you for helping improve Mazzy VPN.

The project is maintained by [Nik m (@mazurovn)](https://github.com/mazurovn)
and licensed under `AGPL-3.0-or-later`. By contributing, you agree that your
contribution may be distributed under that license. Keep existing copyright,
author and SPDX notices intact.

## Before opening a pull request

1. Never commit real VPN profiles, private keys, passwords, preshared keys,
   personal paths, logs or machine-specific state.
2. Run:

   ```bash
   bash -n mazzy-vpn vpnctl install.sh tests/run.sh
   shellcheck mazzy-vpn vpnctl install.sh tests/run.sh \
     setup_amnezia_vpn.sh stop_amnezia_vpn.sh \
     completions/mazzy-vpn completions/vpnctl
   ./tests/run.sh
   ./tests/audit-public.sh
   ```

3. Explain the user impact and the checks performed.
4. Keep changes focused and preserve compatibility aliases and internal
   systemd unit names unless a migration is included.

Security issues should follow [SECURITY.md](SECURITY.md), not a public issue.
