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
   python3 tests/check-capabilities.py
   ./tests/audit-public.sh
   ```

3. Explain the user impact and the checks performed.
4. Keep changes focused and preserve compatibility aliases and internal
   systemd unit names unless a migration is included.

## Cross-surface Definition of Done

Every feature must name an ID from `docs/capabilities.json`. A change is
complete only when:

1. shared core/API behavior and failure semantics are defined;
2. every applicable CLI, TUI, Desktop Linux, macOS and Windows surface is
   implemented or explicitly marked `partial`, `planned` or `not-applicable`;
3. success, failure, timeout and rollback paths have automated coverage;
4. Russian and English docs, Wiki sources and capability statuses agree;
5. the computed release gate remains accurate.

Use the pull-request checklist. Do not copy VPN logic into a frontend to close
one checkbox: clients must share the same core contract.

Security issues should follow [SECURITY.md](SECURITY.md), not a public issue.
