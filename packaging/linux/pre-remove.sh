#!/bin/sh
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
set -eu

case "${1:-}" in
    remove|deconfigure|0) ;;
    *) exit 0 ;;
esac

if [ ! -d /run/systemd/system ]; then
    exit 0
fi

systemctl disable --now vpnctl-health.timer || true
systemctl disable --now mazzy-vpn-api.socket || true
systemctl disable vpnctl-test-recovery.service || true
systemctl disable mazzy-vpn-api-recovery.service || true
systemctl stop vpnctl.service || true
