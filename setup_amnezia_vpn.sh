#!/usr/bin/env bash
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
MAZZY_VPN="$(command -v mazzy-vpn 2>/dev/null || true)"
[[ -n "$MAZZY_VPN" ]] || MAZZY_VPN="$SCRIPT_DIR/mazzy-vpn"

echo "Совместимый запуск AmneziaWG через Mazzy VPN."
exec "$MAZZY_VPN" connect amneziawg "$@"
