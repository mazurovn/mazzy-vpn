#!/bin/sh
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
set -eu

start_opted_in_engine() {
    if systemctl start vpnctl.service; then
        return 0
    fi
    status="$(systemctl show vpnctl.service -p ExecMainStatus --value 2>/dev/null || true)"
    case "$status" in
        75|77)
            printf '%s\n' "Mazzy VPN: engine start deferred by recovery gate (status $status)" >&2
            return 0
            ;;
    esac
    return 1
}

migrate_legacy_cli() {
    root="$1"
    legacy_dir="$root/usr/local/bin"
    package_cli="$root/usr/bin/mazzy-vpn"
    backup_dir="$root/var/lib/vpnctl/package-migration"
    expected_owner=root:root
    [ "$root" = / ] || expected_owner="$(id -un):$(id -gn)"

    [ -x "$package_cli" ] || return 1
    for name in mazzy-vpn vpnctl mazzyvpn; do
        legacy="$legacy_dir/$name"
        backup="$backup_dir/$name.pre-package"
        if [ -L "$legacy" ] && [ "$(readlink "$legacy")" = /usr/bin/mazzy-vpn ]; then
            continue
        fi
        if [ ! -f "$legacy" ] || [ -L "$legacy" ]; then
            continue
        fi
        owner_mode="$(stat -c '%U:%G:%a' "$legacy" 2>/dev/null || true)"
        case "$owner_mode" in
            "$expected_owner":*[2367][0-7]|"$expected_owner":[0-7]*[2367])
                printf '%s\n' "Mazzy VPN: unsafe legacy CLI permissions left unchanged: $legacy" >&2
                continue
                ;;
            "$expected_owner":*) ;;
            *)
                printf '%s\n' "Mazzy VPN: unowned legacy CLI left unchanged: $legacy" >&2
                continue
                ;;
        esac
        if ! grep -Fqx '# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0' "$legacy" ||
           ! grep -Fqx 'PRODUCT_NAME="Mazzy VPN"' "$legacy" ||
           ! grep -Eq '^VERSION="[0-9]+\.[0-9]+\.[0-9]+"$' "$legacy"; then
            printf '%s\n' "Mazzy VPN: unrelated /usr/local command left unchanged: $legacy" >&2
            continue
        fi
        if [ "$root" = / ]; then
            install -d -o root -g root -m 700 "$backup_dir"
        else
            install -d -m 700 "$backup_dir"
        fi
        if [ -e "$backup" ]; then
            printf '%s\n' "Mazzy VPN: existing migration backup prevents replacing $legacy" >&2
            continue
        fi
        mv "$legacy" "$backup"
        if ! ln -s /usr/bin/mazzy-vpn "$legacy"; then
            mv "$backup" "$legacy"
            return 1
        fi
    done
}

migrate_legacy_agentd() {
    root="$1"
    legacy="$root/usr/local/bin/mazzy-agentd"
    package_agentd="$root/usr/bin/mazzy-agentd"
    backup_dir="$root/var/lib/vpnctl/package-migration"
    backup="$backup_dir/mazzy-agentd.pre-package"
    expected_owner=root:root
    [ "$root" = / ] || expected_owner="$(id -un):$(id -gn)"

    [ -x "$package_agentd" ] || return 1
    if [ -L "$legacy" ] && [ "$(readlink "$legacy")" = /usr/bin/mazzy-agentd ]; then
        return 0
    fi
    if [ ! -f "$legacy" ] || [ -L "$legacy" ]; then
        return 0
    fi
    owner_mode="$(stat -c '%U:%G:%a' "$legacy" 2>/dev/null || true)"
    case "$owner_mode" in
        "$expected_owner":*[2367][0-7]|"$expected_owner":[0-7]*[2367])
            printf '%s\n' "Mazzy VPN: unsafe legacy agent daemon permissions left unchanged: $legacy" >&2
            return 0
            ;;
        "$expected_owner":*) ;;
        *)
            printf '%s\n' "Mazzy VPN: unowned legacy agent daemon left unchanged: $legacy" >&2
            return 0
            ;;
    esac
    if ! grep -Fqx '# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0' "$legacy" ||
       ! grep -Fqx 'PROTOCOL = "mazzy-agent-egress/1"' "$legacy" ||
       ! grep -Eq '^VERSION = "[0-9]+\.[0-9]+\.[0-9]+"$' "$legacy"; then
        printf '%s\n' "Mazzy VPN: unrelated /usr/local command left unchanged: $legacy" >&2
        return 0
    fi
    if [ "$root" = / ]; then
        install -d -o root -g root -m 700 "$backup_dir"
    else
        install -d -m 700 "$backup_dir"
    fi
    if [ -e "$backup" ]; then
        printf '%s\n' "Mazzy VPN: existing migration backup prevents replacing $legacy" >&2
        return 0
    fi
    mv "$legacy" "$backup"
    if ! ln -s /usr/bin/mazzy-agentd "$legacy"; then
        mv "$backup" "$legacy"
        return 1
    fi
}

if [ "${1:-}" = --test-migrate ]; then
    test_root="${2:-}"
    case "$test_root" in
        /*/../*|*/..|/|"") exit 2 ;;
        /*) ;;
        *) exit 2 ;;
    esac
    migrate_legacy_cli "$test_root"
    migrate_legacy_agentd "$test_root"
    exit
fi

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

    if [ -z "$access_user" ]; then
        access_user="$(logname 2>/dev/null || true)"
        case "$access_user" in
            ""|root|*[!A-Za-z0-9_.-]*) access_user="" ;;
        esac
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

    migration_root=/var/lib/vpnctl/package-migration/systemd
    # Older source installs placed Mazzy-owned units in /etc, which has higher
    # systemd precedence than the DEB payload. Keep only those exact project
    # paths synchronized so an upgrade cannot boot an obsolete dependency
    # graph while /usr/lib contains the fixed units.
    for relative in \
        vpnctl.service \
        vpnctl-health.service \
        vpnctl-test-recovery.service \
        mazzy-vpn-api.socket \
        mazzy-vpn-api@.service \
        mazzy-vpn-api-recovery.service \
        vpnctl.service.d/10-package-exec.conf \
        vpnctl-health.service.d/10-package-exec.conf \
        vpnctl-test-recovery.service.d/10-package-exec.conf \
        mazzy-vpn-api.socket.d/10-package-docs.conf \
        mazzy-vpn-api@.service.d/10-package-exec.conf \
        mazzy-vpn-api-recovery.service.d/10-package-exec.conf; do
        if [ -e "/etc/systemd/system/$relative" ]; then
            backup="$migration_root/$relative.pre-package"
            checksum="$migration_root/$relative.package-sha256"
            if [ ! -e "$backup" ]; then
                install -D -m 0600 "/etc/systemd/system/$relative" "$backup"
            fi
            install -D -m 0644 "/usr/lib/systemd/system/$relative" \
                "/etc/systemd/system/$relative"
            install -d -m 0700 "$(dirname "$checksum")"
            sha256sum "/usr/lib/systemd/system/$relative" | awk '{print $1}' >"$checksum"
            chmod 0600 "$checksum"
        fi
    done

    systemctl daemon-reload
    systemctl enable mazzy-vpn-api-recovery.service
    systemctl enable mazzy-vpn-api.socket
    systemctl enable vpnctl-test-recovery.service
    systemctl enable vpnctl-health.timer
    # RemainAfterExit may still describe the previous package's recovery pass.
    # Re-run the newly installed coordinator before any opted-in managed VPN.
    systemctl reset-failed mazzy-vpn-api-recovery.service 2>/dev/null || true
    systemctl restart mazzy-vpn-api-recovery.service
    systemctl restart mazzy-vpn-api.socket
    systemctl restart vpnctl-health.timer

    # Preserve an existing user opt-in across package upgrades without
    # enabling the VPN service on a fresh install.
    if systemctl is-enabled --quiet vpnctl.service; then
        systemctl enable vpnctl.service
        start_opted_in_engine
    fi

    /usr/lib/mazzy-vpn/mazzy-vpn _refresh-dashboard-cache
}

verify_payload() {
    test -x /usr/lib/mazzy-vpn/mazzy-vpn
    test -x /usr/bin/mazzy-vpn
    cmp -s /usr/lib/mazzy-vpn/mazzy-vpn /usr/bin/mazzy-vpn
    test -x /usr/bin/mazzy-agentd
    test -x /usr/lib/mazzy-vpn/runtime/mazzy-sing-box-adapter
    test -r /usr/lib/mazzy-vpn/protocols/v1/managed-profile.schema.json
    test -r /usr/lib/mazzy-vpn/runtime/v1/adapter-registry.json
    /usr/lib/mazzy-vpn/mazzy-vpn version
    /usr/lib/mazzy-vpn/mazzy-vpn api-info --json |
        cmp -s - /usr/lib/mazzy-vpn/api/v1/manifest.json
    /usr/lib/mazzy-vpn/mazzy-vpn protocols list --json |
        cmp -s - /usr/lib/mazzy-vpn/protocols/v1/registry.json
    /usr/lib/mazzy-vpn/mazzy-vpn protocols adapters --json |
        cmp -s - /usr/lib/mazzy-vpn/runtime/v1/adapter-registry.json
    /usr/lib/mazzy-vpn/mazzy-vpn agent-transports list --json |
        cmp -s - /usr/lib/mazzy-vpn/agent-control/v1/registry.json
    /usr/bin/mazzy-agentd --version | grep -Eq '^mazzy-agentd [0-9]+\.[0-9]+\.[0-9]+$'
    printf '%s\n' '{"api_version":"1.0","request_id":"request-package-capabilities","operation":"api.capabilities","deadline_ms":1000,"payload":{}}' |
        /usr/lib/mazzy-vpn/mazzy-vpn _api-dispatch |
        jq -e '
            .status == "ok"
            and ([
                "status.get", "profiles.list", "planner.evaluate",
                "lifecycle.connect", "lifecycle.disconnect",
                "tests.verify-service-egress", "region.check"
            ] - .result.operations | length) == 0
        ' >/dev/null
}

ensure_access_group
ensure_state_layout
grant_installer_access
verify_payload
migrate_legacy_cli /
migrate_legacy_agentd /
activate_services
