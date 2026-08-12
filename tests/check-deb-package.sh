#!/usr/bin/env bash
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
# DEB-only release gate. It never installs the package or touches live VPN state.

set -Eeuo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
BUNDLE_ROOT="${1:-${MAZZY_BUNDLE_ROOT:-}}"

fail() {
    printf 'deb-package-audit: %s\n' "$*" >&2
    exit 1
}

for command in dpkg-deb jq sha256sum stat cmp grep find; do
    command -v "$command" >/dev/null 2>&1 || fail "missing command: $command"
done

if [[ -z "$BUNDLE_ROOT" ]]; then
    command -v cargo >/dev/null 2>&1 || fail "missing command: cargo"
    BUNDLE_ROOT="$(
        cargo metadata --format-version 1 --no-deps \
            --manifest-path "$ROOT/desktop/src-tauri/Cargo.toml" |
            python3 -c 'import json, sys; print(json.load(sys.stdin)["target_directory"])'
    )/release/bundle"
fi

mapfile -t packages < <(find "$BUNDLE_ROOT/deb" -maxdepth 1 -type f -name '*.deb' -print 2>/dev/null | sort)
(( ${#packages[@]} == 1 )) ||
    fail "expected exactly one DEB in $BUNDLE_ROOT/deb, found ${#packages[@]}"
deb="${packages[0]}"
[[ -s "$deb" ]] || fail "DEB is empty: $deb"

mkdir -p "$ROOT/.mazzy/work/deb-audit"
tmp="$(mktemp -d -p "$ROOT/.mazzy/work/deb-audit" audit.XXXXXX)"
trap 'rm -rf -- "$tmp"' EXIT
root="$tmp/root"
control="$tmp/control"
mkdir -p "$root" "$control"
dpkg-deb --info "$deb" >/dev/null
dpkg-deb -x "$deb" "$root"
dpkg-deb -e "$deb" "$control"

desktop_version="$(jq -er '.version' "$ROOT/desktop/src-tauri/tauri.conf.json")"
package_version="$(dpkg-deb -f "$deb" Version)"
[[ "$package_version" == "$desktop_version" ]] ||
    fail "DEB version $package_version differs from Desktop $desktop_version"
grep -Fxq "version = \"$desktop_version\"" "$ROOT/desktop/src-tauri/Cargo.toml" ||
    fail "Cargo package version differs from Desktop configuration"
[[ "$(jq -er '.version' "$ROOT/desktop/package.json")" == "$desktop_version" ]] ||
    fail "npm package version differs from Desktop configuration"

required=(
    usr/bin/mazzy-vpn-desktop
    usr/bin/mazzy-vpn
    usr/bin/mazzy-agentd
    usr/lib/mazzy-vpn/mazzy-vpn
    usr/lib/mazzy-vpn/providers/v1/registry.json
    usr/lib/mazzy-vpn/api/v1/manifest.json
    usr/lib/mazzy-vpn/protocols/v1/registry.json
    usr/lib/mazzy-vpn/runtime/v1/adapter-registry.json
    usr/lib/systemd/system/vpnctl.service.d/10-package-exec.conf
    usr/lib/systemd/system/vpnctl-health.service.d/10-package-exec.conf
    usr/lib/systemd/system/vpnctl-test-recovery.service.d/10-package-exec.conf
    usr/lib/systemd/system/mazzy-vpn-api@.service.d/10-package-exec.conf
    usr/lib/systemd/system/mazzy-vpn-api-recovery.service.d/10-package-exec.conf
    usr/lib/systemd/system/mazzy-vpn-api.socket
    usr/lib/systemd/system/vpnctl-health.timer
)
for relative in "${required[@]}"; do
    [[ -e "$root/$relative" ]] || fail "missing DEB payload: /$relative"
done

for executable in usr/bin/mazzy-vpn-desktop usr/bin/mazzy-vpn usr/bin/mazzy-agentd usr/lib/mazzy-vpn/mazzy-vpn; do
    [[ "$(stat -c '%a' "$root/$executable")" == 755 ]] ||
        fail "/$executable mode is not 0755"
done

cmp -s "$ROOT/mazzy-vpn" "$root/usr/lib/mazzy-vpn/mazzy-vpn" ||
    fail "package-internal engine differs from source"
cmp -s "$ROOT/mazzy-vpn" "$root/usr/bin/mazzy-vpn" ||
    fail "public CLI differs from source"
cmp -s "$ROOT/mazzy-agentd" "$root/usr/bin/mazzy-agentd" ||
    fail "mazzy-agentd differs from source"
cmp -s "$ROOT/providers/v1/registry.json" "$root/usr/lib/mazzy-vpn/providers/v1/registry.json" ||
    fail "provider registry differs from source"
cmp -s "$ROOT/packaging/linux/post-install.sh" "$control/postinst" ||
    fail "DEB postinst differs from audited source"
bash -n "$control/postinst" || fail "DEB postinst has invalid shell syntax"

for relative in \
    vpnctl.service.d/10-package-exec.conf \
    vpnctl-health.service.d/10-package-exec.conf \
    vpnctl-test-recovery.service.d/10-package-exec.conf \
    mazzy-vpn-api@.service.d/10-package-exec.conf \
    mazzy-vpn-api-recovery.service.d/10-package-exec.conf; do
    unit="$root/usr/lib/systemd/system/$relative"
    grep -Fq '/usr/lib/mazzy-vpn/mazzy-vpn' "$unit" ||
        fail "$relative does not use the package-internal engine"
    if grep -Eq 'Exec(Start|Condition)=.*(/usr/bin|/usr/local/bin)/mazzy-vpn([[:space:]]|$)' "$unit"; then
        fail "$relative has a hard dependency on the public CLI"
    fi
done

for legacy in mazzy-vpn-api@.service mazzy-vpn-api@.service.d/10-package-exec.conf; do
    grep -Fq "$legacy" "$control/postinst" ||
        fail "DEB postinst does not synchronize legacy $legacy"
done
grep -Fq 'start_opted_in_engine' "$control/postinst" ||
    fail "DEB postinst does not defer an opted-in engine behind recovery"
grep -Eq '75\|77' "$control/postinst" ||
    fail "DEB postinst does not recognize recovery gate exit statuses"

if command -v systemd-analyze >/dev/null 2>&1; then
    # Older Ubuntu systemd-analyze cannot use --root with verify. Build an
    # isolated copy of the package unit graph, excluding host /etc overrides.
    # The real extracted engine is mode/byte/execution-checked separately; in
    # the verification copy only, /bin/true supplies an existing ExecStart.
    verify_units="$tmp/systemd-units"
    mkdir -p "$verify_units"
    cp -a "$root/usr/lib/systemd/system/." "$verify_units/"
    while IFS= read -r -d '' unit_file; do
        sed -i 's|/usr/lib/mazzy-vpn/mazzy-vpn|/bin/true|g' "$unit_file"
    done < <(find "$verify_units" -type f -print0)
    SYSTEMD_UNIT_PATH="$verify_units:/usr/lib/systemd/system" \
      systemd-analyze --man=no verify \
        mazzy-vpn-api-recovery.service \
        mazzy-vpn-api.socket \
        mazzy-vpn-api@internal.service \
        vpnctl.service \
        vpnctl-health.service \
        vpnctl-health.timer \
        vpnctl-test-recovery.service \
        >/dev/null 2>"$tmp/systemd.log" || {
            cat "$tmp/systemd.log" >&2
            fail "DEB systemd units do not verify"
        }
fi

engine_version="$("$root/usr/lib/mazzy-vpn/mazzy-vpn" version)"
[[ -n "$engine_version" ]] || fail "package-internal engine did not execute"
"$root/usr/lib/mazzy-vpn/mazzy-vpn" api-info --json |
    cmp -s - "$root/usr/lib/mazzy-vpn/api/v1/manifest.json" ||
    fail "package-internal engine/API manifest mismatch"

if command -v dbus-run-session >/dev/null 2>&1 &&
   command -v xvfb-run >/dev/null 2>&1 && command -v timeout >/dev/null 2>&1; then
    if command -v pgrep >/dev/null 2>&1 && pgrep -f '(^|/)mazzy-vpn-desktop($| )' >/dev/null; then
        printf '%s\n' 'deb-package-audit: GUI smoke skipped because Desktop is already running'
    else
        rc=0
        NO_AT_BRIDGE=1 GDK_BACKEND=x11 dbus-run-session -- \
            timeout --kill-after=2s 10s xvfb-run -a \
            "$root/usr/bin/mazzy-vpn-desktop" >"$tmp/gui.log" 2>&1 || rc=$?
        (( rc == 124 )) || {
            cat "$tmp/gui.log" >&2
            fail "DEB Desktop exited before the 10-second GUI smoke completed (exit $rc)"
        }
    fi
fi

printf 'deb-package-audit: OK version=%s sha256=%s artifact=%s\n' \
    "$desktop_version" "$(sha256sum "$deb" | awk '{print $1}')" "$deb"
