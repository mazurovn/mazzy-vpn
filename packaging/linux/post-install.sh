#!/bin/sh
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
set -eu

ensure_access_group() {
    if ! getent group mazzy-vpn >/dev/null 2>&1; then
        groupadd --system mazzy-vpn
    fi
}

ensure_state_layout() {
    install -d -o root -g root -m 700 \
        /etc/vpnctl \
        /etc/vpnctl/profiles \
        /etc/vpnctl/profiles/amneziawg \
        /etc/vpnctl/profiles/wireguard \
        /etc/vpnctl/profiles/openvpn \
        /etc/vpnctl/profiles/l2tp \
        /var/lib/vpnctl

    if [ ! -e /etc/vpnctl/locale ]; then
        locale_tmp="$(mktemp /etc/vpnctl/.locale.XXXXXX)"
        trap 'rm -f "$locale_tmp"' EXIT HUP INT TERM
        printf 'ru\n' >"$locale_tmp"
        chmod 644 "$locale_tmp"
        mv "$locale_tmp" /etc/vpnctl/locale
        trap - EXIT HUP INT TERM
    fi
}

grant_installer_access() {
    access_user="${SUDO_USER:-}"
    case "$access_user" in
        ""|root|*[!A-Za-z0-9_.-]*) access_user="" ;;
    esac

    if [ -z "$access_user" ] &&
       [ "${PKEXEC_UID:-}" -ge 1000 ] 2>/dev/null; then
        access_user="$(getent passwd "$PKEXEC_UID" | cut -d: -f1 || true)"
    fi

    if [ -n "$access_user" ] &&
       id "$access_user" >/dev/null 2>&1 &&
       ! id -nG "$access_user" | tr ' ' '\n' | grep -Fxq mazzy-vpn; then
        usermod -a -G mazzy-vpn "$access_user"
    fi
}

activate_services() {
    systemd-tmpfiles --create /usr/lib/tmpfiles.d/mazzy-vpn.conf

    if [ ! -d /run/systemd/system ]; then
        return 0
    fi

    systemctl daemon-reload
    systemctl enable mazzy-vpn-api.socket
    systemctl enable vpnctl-test-recovery.service
    systemctl enable vpnctl-health.timer
    systemctl restart mazzy-vpn-api.socket
    systemctl restart vpnctl-health.timer

    if systemctl is-active --quiet vpnctl.service; then
        systemctl try-restart vpnctl.service
    fi

    /usr/bin/mazzy-vpn _refresh-dashboard-cache
}

verify_payload() {
    /usr/bin/mazzy-vpn version
    /usr/bin/mazzy-vpn api-info --json |
        cmp -s - /usr/lib/mazzy-vpn/api/v1/manifest.json
}

ensure_access_group
ensure_state_layout
grant_installer_access
verify_payload
activate_services
