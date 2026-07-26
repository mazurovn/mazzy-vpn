#!/usr/bin/env bash
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
MAZZY_VPN="$(command -v mazzy-vpn 2>/dev/null || true)"
[[ -n "$MAZZY_VPN" ]] || MAZZY_VPN="$SCRIPT_DIR/mazzy-vpn"

exec "$MAZZY_VPN" disconnect
