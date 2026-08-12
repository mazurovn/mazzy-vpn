#!/bin/sh
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
set -eu

restore_legacy_cli() {
    root="$1"
    legacy_dir="$root/usr/local/bin"
    backup_dir="$root/var/lib/vpnctl/package-migration"

    for name in mazzy-vpn vpnctl mazzyvpn mazzy-agentd; do
        legacy="$legacy_dir/$name"
        backup="$backup_dir/$name.pre-package"
        package_target=/usr/bin/mazzy-vpn
        [ "$name" = mazzy-agentd ] && package_target=/usr/bin/mazzy-agentd
        [ -f "$backup" ] || continue
        if [ -L "$legacy" ] && [ "$(readlink "$legacy")" = "$package_target" ]; then
            rm -f "$legacy"
        elif [ -e "$legacy" ] || [ -L "$legacy" ]; then
            printf '%s\n' "Mazzy VPN: $legacy changed after installation; backup retained at $backup" >&2
            continue
        fi
        mv "$backup" "$legacy"
    done
    rmdir "$backup_dir" 2>/dev/null || true
}

restore_legacy_systemd() {
    root="$1"
    migration_root="$root/var/lib/vpnctl/package-migration/systemd"
    [ -d "$migration_root" ] || return 0
    find "$migration_root" -type f -name '*.pre-package' -print |
    while IFS= read -r backup; do
        relative="${backup#"$migration_root/"}"
        relative="${relative%.pre-package}"
        current="$root/etc/systemd/system/$relative"
        checksum="$migration_root/$relative.package-sha256"
        expected="$(cat "$checksum" 2>/dev/null || true)"
        actual="$(sha256sum "$current" 2>/dev/null | awk '{print $1}')"
        if [ -n "$expected" ] && [ "$actual" = "$expected" ]; then
            install -D -m 0644 "$backup" "$current"
            rm -f "$backup" "$checksum"
        else
            printf '%s\n' "Mazzy VPN: $current changed after installation; legacy backup retained at $backup" >&2
        fi
    done
    find "$migration_root" -depth -type d -empty -delete 2>/dev/null || true
}

if [ "${1:-}" = --test-restore ]; then
    test_root="${2:-}"
    case "$test_root" in
        /*/../*|*/..|/|"") exit 2 ;;
        /*) ;;
        *) exit 2 ;;
    esac
    restore_legacy_cli "$test_root"
    restore_legacy_systemd "$test_root"
    exit
fi

case "${1:-}" in
    remove|purge|0) ;;
    *) exit 0 ;;
esac

restore_legacy_cli /
restore_legacy_systemd /

if [ -d /run/systemd/system ]; then
    systemctl daemon-reload
    systemctl reset-failed vpnctl.service mazzy-vpn-api.socket \
        vpnctl-health.service vpnctl-health.timer \
        vpnctl-test-recovery.service mazzy-vpn-api-recovery.service || true
fi

# /etc/vpnctl and /var/lib/vpnctl are user state and are intentionally kept.
rmdir /run/mazzy-vpn 2>/dev/null || true
