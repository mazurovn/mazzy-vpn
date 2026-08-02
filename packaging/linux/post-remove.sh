#!/bin/sh
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
set -eu

restore_legacy_cli() {
    root="$1"
    legacy_dir="$root/usr/local/bin"
    backup_dir="$root/var/lib/vpnctl/package-migration"

    for name in mazzy-vpn vpnctl mazzyvpn; do
        legacy="$legacy_dir/$name"
        backup="$backup_dir/$name.pre-package"
        [ -f "$backup" ] || continue
        if [ -L "$legacy" ] && [ "$(readlink "$legacy")" = /usr/bin/mazzy-vpn ]; then
            rm -f "$legacy"
        elif [ -e "$legacy" ] || [ -L "$legacy" ]; then
            printf '%s\n' "Mazzy VPN: $legacy changed after installation; backup retained at $backup" >&2
            continue
        fi
        mv "$backup" "$legacy"
    done
    rmdir "$backup_dir" 2>/dev/null || true
}

if [ "${1:-}" = --test-restore ]; then
    [ "$(id -u)" -ne 0 ] || exit 2
    test_root="${2:-}"
    case "$test_root" in
        /*/../*|*/..|/|"") exit 2 ;;
        /*) ;;
        *) exit 2 ;;
    esac
    restore_legacy_cli "$test_root"
    exit
fi

case "${1:-}" in
    remove|purge|0) ;;
    *) exit 0 ;;
esac

restore_legacy_cli /

if [ -d /run/systemd/system ]; then
    systemctl daemon-reload
    systemctl reset-failed vpnctl.service mazzy-vpn-api.socket \
        vpnctl-health.service vpnctl-health.timer \
        vpnctl-test-recovery.service mazzy-vpn-api-recovery.service || true
fi

# /etc/vpnctl and /var/lib/vpnctl are user state and are intentionally kept.
rmdir /run/mazzy-vpn 2>/dev/null || true
