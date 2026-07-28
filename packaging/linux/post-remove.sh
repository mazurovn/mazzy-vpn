#!/bin/sh
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
set -eu

case "${1:-}" in
    remove|purge|0) ;;
    *) exit 0 ;;
esac

if [ -d /run/systemd/system ]; then
    systemctl daemon-reload
    systemctl reset-failed vpnctl.service mazzy-vpn-api.socket \
        vpnctl-health.service vpnctl-health.timer \
        vpnctl-test-recovery.service || true
fi

# /etc/vpnctl and /var/lib/vpnctl are user state and are intentionally kept.
rmdir /run/mazzy-vpn 2>/dev/null || true
