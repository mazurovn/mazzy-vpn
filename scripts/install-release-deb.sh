#!/usr/bin/env bash
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
# Install an audited Mazzy VPN DEB without disconnecting the active tunnel.

set -Eeuo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
DEFAULT_DEB="$ROOT/dist/desktop-v0.4.8/Mazzy VPN Desktop_0.4.8_amd64.deb"
DEB="${1:-$DEFAULT_DEB}"
ROLLBACK_DEB="${2:-}"
ROLLBACK_ROOT="$ROOT/.mazzy/rollback"
EXPECTED_PACKAGE=mazzy-vpn-desktop

die() { printf 'install-release-deb: %s\n' "$*" >&2; exit 1; }
info() { printf 'install-release-deb: %s\n' "$*"; }

for command in dpkg-deb dpkg-query sha256sum stat systemctl awk date install getent; do
    command -v "$command" >/dev/null 2>&1 || die "missing command: $command"
done

DEB="$(realpath -e -- "$DEB")" || die "release DEB not found"
[[ -f "$DEB" && ! -L "$DEB" && -s "$DEB" ]] || die "invalid release DEB"
package="$(dpkg-deb -f "$DEB" Package)"
version="$(dpkg-deb -f "$DEB" Version)"
architecture="$(dpkg-deb -f "$DEB" Architecture)"
[[ "$package" == "$EXPECTED_PACKAGE" ]] || die "unexpected package: $package"
[[ "$architecture" == amd64 ]] || die "unexpected architecture: $architecture"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "invalid package version"

installed_version="$(dpkg-query -W -f='${Version}' "$EXPECTED_PACKAGE" 2>/dev/null || true)"
[[ -n "$installed_version" ]] || die "$EXPECTED_PACKAGE is not currently installed"

if [[ -n "$ROLLBACK_DEB" ]]; then
    ROLLBACK_DEB="$(realpath -e -- "$ROLLBACK_DEB")" || die "rollback DEB not found"
else
    mapfile -t candidates < <(
        find /var/cache/apt/archives -maxdepth 1 -type f \
            -name "${EXPECTED_PACKAGE}_${installed_version}_*.deb" -print 2>/dev/null
    )
    ((${#candidates[@]} == 1)) && ROLLBACK_DEB="${candidates[0]}"
fi
[[ -n "$ROLLBACK_DEB" && -f "$ROLLBACK_DEB" && ! -L "$ROLLBACK_DEB" ]] ||
    die "provide the currently installed rollback DEB as the second argument"
[[ "$(dpkg-deb -f "$ROLLBACK_DEB" Package)" == "$EXPECTED_PACKAGE" ]] ||
    die "rollback DEB contains another package"
[[ "$(dpkg-deb -f "$ROLLBACK_DEB" Version)" == "$installed_version" ]] ||
    die "rollback DEB does not match installed version $installed_version"

mkdir -p -- "$ROLLBACK_ROOT"
chmod 700 -- "$ROOT/.mazzy" "$ROLLBACK_ROOT"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
backup="$ROLLBACK_ROOT/pre-${version}-${stamp}"
mkdir -- "$backup"
chmod 700 -- "$backup"
install -m 600 -- "$ROLLBACK_DEB" "$backup/rollback.deb"
rollback_sha="$(sha256sum "$backup/rollback.deb" | awk '{print $1}')"
release_sha="$(sha256sum "$DEB" | awk '{print $1}')"

units=(mazzy-vpn-api-recovery.service mazzy-vpn-api.socket vpnctl.service vpnctl-health.timer)
{
    printf 'schema=mazzy-vpn.deb-rollback.v1\n'
    printf 'package=%s\n' "$EXPECTED_PACKAGE"
    printf 'previous_version=%s\n' "$installed_version"
    printf 'target_version=%s\n' "$version"
    printf 'release_sha256=%s\n' "$release_sha"
    printf 'rollback_sha256=%s\n' "$rollback_sha"
    for unit in "${units[@]}"; do
        enabled="$(systemctl is-enabled "$unit" 2>/dev/null || true)"
        active="$(systemctl is-active "$unit" 2>/dev/null || true)"
        printf 'unit_%s_enabled=%s\n' "${unit//[^A-Za-z0-9]/_}" "$enabled"
        printf 'unit_%s_active=%s\n' "${unit//[^A-Za-z0-9]/_}" "$active"
    done
} >"$backup/manifest.env"
chmod 600 -- "$backup/manifest.env"
printf '%s\n' "$backup" >"$ROLLBACK_ROOT/latest"
chmod 600 -- "$ROLLBACK_ROOT/latest"
invoking_uid="${PKEXEC_UID:-${SUDO_UID:-}}"
if ((EUID == 0)) && [[ "$invoking_uid" =~ ^[1-9][0-9]*$ ]]; then
    invoking_gid="$(getent passwd "$invoking_uid" | awk -F: '{print $4; exit}')"
    [[ "$invoking_gid" =~ ^[1-9][0-9]*$ ]] || die "cannot resolve invoking user group"
    chown "$invoking_uid:$invoking_gid" "$ROLLBACK_ROOT" "$ROLLBACK_ROOT/latest" "$backup"
    chown "$invoking_uid:$invoking_gid" "$backup/manifest.env" "$backup/rollback.deb"
fi

before_service="$(systemctl is-active vpnctl.service 2>/dev/null || true)"
info "prepared rollback $installed_version at $backup"
info "installing $EXPECTED_PACKAGE $version; active VPN will not be stopped"

if ((EUID == 0)); then
    apt-get install -y -- "$DEB"
else
    sudo -v
    sudo apt-get install -y -- "$DEB"
fi

actual="$(dpkg-query -W -f='${Version}' "$EXPECTED_PACKAGE")"
[[ "$actual" == "$version" ]] || die "installed version is $actual, expected $version"
after_service="$(systemctl is-active vpnctl.service 2>/dev/null || true)"
if [[ "$before_service" == active && "$after_service" != active ]]; then
    die "package installed, but the previously active VPN service is no longer active"
fi
systemctl is-active --quiet mazzy-vpn-api.socket || die "local API socket is not active"
systemctl is-enabled --quiet vpnctl-health.timer || die "health timer is not enabled"

deadline=$((SECONDS + 75))
while ((SECONDS < deadline)); do
    if /usr/bin/mazzy-vpn status --api-json >/dev/null 2>&1 &&
       /usr/bin/mazzy-vpn profiles --api-json >/dev/null 2>&1; then
        info "installation verified: API status and profiles are readable"
        info "rollback command: $ROOT/scripts/rollback-release-deb.sh '$backup'"
        exit 0
    fi
    sleep 1
done
die "package installed, but API status/profile verification did not become ready"
