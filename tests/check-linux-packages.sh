#!/usr/bin/env bash
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
set -Eeuo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
BUNDLE_ROOT="${1:-}"
TMP="$(mktemp -d)"
trap 'rm -rf -- "$TMP"' EXIT

fail() {
    echo "linux-package-audit: $*" >&2
    exit 1
}

report_unexpected_error() {
    local rc="$1" line="$2" command="$3"
    trap - ERR
    printf 'linux-package-audit: unexpected failure at line %s (exit %s): %s\n' \
        "$line" "$rc" "$command" >&2
    exit "$rc"
}

trap 'report_unexpected_error "$?" "$LINENO" "$BASH_COMMAND"' ERR

require_command() {
    command -v "$1" >/dev/null 2>&1 ||
        fail "required audit command is missing: $1"
}

single_package() {
    local directory="$1" pattern="$2"
    local -a matches=()
    mapfile -d '' matches < <(
        find "$directory" -maxdepth 1 -type f -name "$pattern" -print0
    )
    ((${#matches[@]} == 1)) ||
        fail "expected exactly one $pattern under $directory, found ${#matches[@]}"
    printf '%s\n' "${matches[0]}"
}

assert_line() {
    local needle="$1" file="$2"
    grep -Fxq "$needle" "$file" ||
        fail "missing package entry: $needle"
}

assert_dependency() {
    local dependency="$1" file="$2"
    grep -Fxq "$dependency" "$file" ||
        fail "missing package dependency: $dependency"
}

find_appimage_offset() {
    local image="$1" offset="" marker
    while IFS=: read -r marker _; do
        if unsquashfs -s -o "$marker" "$image" >/dev/null 2>&1; then
            offset="$marker"
            break
        fi
    done < <(LC_ALL=C grep -abo 'hsqs' "$image")
    [[ -n "$offset" ]] ||
        fail "AppImage does not contain a readable SquashFS payload"
    printf '%s\n' "$offset"
}

assert_gui_launch() {
    local label="$1"
    shift
    local log="$TMP/${label,,}-launch.log" rc=0
    GDK_BACKEND=x11 QT_QPA_PLATFORM=xcb \
        /usr/bin/timeout --kill-after=2s 10s \
        xvfb-run -a "$@" >"$log" 2>&1 || rc=$?
    if ((rc != 124)); then
        cat "$log" >&2
        fail "$label bundle exited before the 10-second GUI launch smoke completed (exit $rc)"
    fi
}

require_command cpio
require_command cargo
require_command dpkg-deb
require_command python3
require_command rpm
require_command rpm2cpio
require_command systemd-analyze
require_command unsquashfs
require_command xvfb-run

if [[ -z "$BUNDLE_ROOT" ]]; then
    BUNDLE_ROOT="$(
        cargo metadata --format-version 1 --no-deps \
            --manifest-path "$ROOT/desktop/src-tauri/Cargo.toml" |
            python3 -c 'import json, sys; print(json.load(sys.stdin)["target_directory"])'
    )/release/bundle"
fi

deb="$(single_package "$BUNDLE_ROOT/deb" '*.deb')"
rpm_package="$(single_package "$BUNDLE_ROOT/rpm" '*.rpm')"
appimage="$(single_package "$BUNDLE_ROOT/appimage" '*.AppImage')"
signature_args=()
signature_count=0
for artifact in "$deb" "$rpm_package" "$appimage"; do
    if [[ -s "$artifact.sig" ]]; then
        signature_args+=("$artifact" "$artifact.sig")
        ((signature_count += 1))
    elif [[ -e "$artifact.sig" ]]; then
        fail "empty updater signature: $artifact.sig"
    fi
done
if ((signature_count != 0 && signature_count != 3)); then
    fail "release artifacts contain only $signature_count of 3 updater signatures"
fi
if ((signature_count == 3)); then
    cargo run --locked --quiet \
        --manifest-path "$ROOT/desktop/src-tauri/Cargo.toml" \
        --example verify-updater-signatures -- \
        "$ROOT/desktop/src-tauri/tauri.conf.json" \
        "${signature_args[@]}" ||
        fail "updater artifact signature verification failed"
fi

deb_control="$TMP/deb-control"
deb_root="$TMP/deb-root"
rpm_root="$TMP/rpm-root"
rpm_db="$TMP/rpm-db"
appimage_root="$TMP/appimage-root"
appimage_stage="$TMP/appimage-stage"
systemd_root="$TMP/systemd-root"
mkdir -p "$deb_control" "$deb_root" "$rpm_root" "$rpm_db"
rpm --dbpath "$rpm_db" --initdb
dpkg-deb -e "$deb" "$deb_control"
dpkg-deb -x "$deb" "$deb_root"
if ! rpm -Kv "$rpm_package" >"$TMP/rpm-verify.log" 2>&1; then
    cat "$TMP/rpm-verify.log" >&2
    fail "RPM header or payload digest verification failed"
fi
rpm_payload="$TMP/rpm-payload.cpio"
rpm2cpio_rc=0
rpm2cpio "$rpm_package" >"$rpm_payload" || rpm2cpio_rc=$?
# Ubuntu 22.04 rpm2cpio 4.17 returns 1 for this digest-valid Tauri RPM after
# writing a complete archive. Accept only that known code; cpio extraction,
# metadata checks and byte comparisons below still have to succeed.
if ((rpm2cpio_rc != 0 && rpm2cpio_rc != 1)); then
    fail "rpm2cpio failed with unsupported exit code: $rpm2cpio_rc"
fi
(
    cd "$rpm_root"
    cpio -idm --quiet <"$rpm_payload"
)
appimage_offset="$(find_appimage_offset "$appimage")"
unsquashfs -q -o "$appimage_offset" -d "$appimage_root" "$appimage" >/dev/null
appimage_engine="$appimage_root/usr/lib/Mazzy VPN Desktop/engine"

mkdir -p \
    "$systemd_root/usr/bin" \
    "$systemd_root/usr/lib/systemd/system" \
    "$systemd_root/etc"
cp -a /usr/lib/systemd/system/. "$systemd_root/usr/lib/systemd/system/"
cp -L /etc/os-release "$systemd_root/etc/os-release"
find "$systemd_root/usr/lib/systemd/system" -xtype l -delete
cp -a "$deb_root/usr/bin/." "$systemd_root/usr/bin/"
cp -a "$deb_root/usr/lib/systemd/system/." \
    "$systemd_root/usr/lib/systemd/system/"
systemd_units=(
    mazzy-vpn-api-recovery.service \
    mazzy-vpn-api.socket \
    mazzy-vpn-api@.service \
    vpnctl.service \
    vpnctl-health.timer \
    vpnctl-health.service \
    vpnctl-test-recovery.service
)
if ! systemd-analyze verify --root="$systemd_root" \
    "${systemd_units[@]}" >"$TMP/systemd-verify.log" 2>&1; then
    if grep -Fq \
        'Option --root is only supported for cat-config right now.' \
        "$TMP/systemd-verify.log"; then
        systemd_verify_dir="$TMP/systemd-verify-units"
        mkdir -p "$systemd_verify_dir"
        cp -a "$deb_root/usr/lib/systemd/system/." "$systemd_verify_dir/"
        find "$systemd_verify_dir" -type f -name '*.conf' -exec \
            sed -i -E \
                's#^ExecStart=/usr/bin/mazzy-vpn .*$#ExecStart=/bin/true#' \
                {} +
        if ! SYSTEMD_UNIT_PATH="$systemd_verify_dir:/usr/lib/systemd/system" \
            systemd-analyze verify "${systemd_units[@]}" \
                >"$TMP/systemd-verify.log" 2>&1; then
            cat "$TMP/systemd-verify.log" >&2
            fail "assembled package systemd units do not verify"
        fi
    else
        cat "$TMP/systemd-verify.log" >&2
        fail "assembled package systemd units do not verify"
    fi
fi

cmp -s "$ROOT/packaging/linux/post-install.sh" "$deb_control/postinst" ||
    fail "DEB postinst differs from the audited source"
grep -q '^    if systemctl is-enabled --quiet vpnctl.service; then$' \
    "$ROOT/packaging/linux/post-install.sh" ||
    fail "package post-install does not preserve the existing engine opt-in"
grep -q '^        systemctl enable vpnctl.service$' \
    "$ROOT/packaging/linux/post-install.sh" ||
    fail "package post-install does not re-enable an opted-in engine"
grep -q '^        systemctl start vpnctl.service$' \
    "$ROOT/packaging/linux/post-install.sh" ||
    fail "package post-install does not start an opted-in engine after upgrade"
if grep -Eq '^    systemctl enable vpnctl\.service$' \
    "$ROOT/packaging/linux/post-install.sh"; then
    fail "package post-install unconditionally enables the VPN engine"
fi
grep -q '^    if systemctl is-enabled --quiet vpnctl.service; then$' \
    "$ROOT/install.sh" ||
    fail "installer does not preserve the existing engine opt-in"
grep -q '^        run systemctl enable vpnctl.service$' \
    "$ROOT/install.sh" ||
    fail "installer does not re-enable an opted-in engine"
grep -q '^        run systemctl start vpnctl.service$' \
    "$ROOT/install.sh" ||
    fail "installer does not start an opted-in engine after upgrade"
if grep -Eq '^    run systemctl enable vpnctl\.service$' "$ROOT/install.sh"; then
    fail "installer unconditionally enables the VPN engine"
fi
cmp -s "$ROOT/packaging/linux/pre-remove.sh" "$deb_control/prerm" ||
    fail "DEB prerm differs from the audited source"
cmp -s "$ROOT/packaging/linux/post-remove.sh" "$deb_control/postrm" ||
    fail "DEB postrm differs from the audited source"

rpm --dbpath "$rpm_db" -qp --qf '%{POSTIN}' "$rpm_package" >"$TMP/rpm-post"
rpm --dbpath "$rpm_db" -qp --qf '%{PREUN}' "$rpm_package" >"$TMP/rpm-preun"
rpm --dbpath "$rpm_db" -qp --qf '%{POSTUN}' "$rpm_package" >"$TMP/rpm-postun"
cmp -s "$ROOT/packaging/linux/post-install.sh" "$TMP/rpm-post" ||
    fail "RPM post-install script differs from the audited source"
cmp -s "$ROOT/packaging/linux/pre-remove.sh" "$TMP/rpm-preun" ||
    fail "RPM pre-uninstall script differs from the audited source"
cmp -s "$ROOT/packaging/linux/post-remove.sh" "$TMP/rpm-postun" ||
    fail "RPM post-uninstall script differs from the audited source"

dpkg-deb -f "$deb" Depends |
    tr ',' '\n' |
    sed -E 's/^[[:space:]]*//; s/[[:space:](].*$//' >"$TMP/deb-depends"
dpkg-deb -f "$deb" Recommends |
    tr ',' '\n' |
    sed -E 's/^[[:space:]]*//; s/[[:space:](].*$//' >"$TMP/deb-recommends"
rpm --dbpath "$rpm_db" -qp --requires "$rpm_package" >"$TMP/rpm-requires"
rpm --dbpath "$rpm_db" -qp --recommends "$rpm_package" >"$TMP/rpm-recommends"

for dependency in \
    bash diffutils findutils grep iproute2 jq nftables pkexec procps python3 sed socat systemd \
    util-linux; do
    assert_dependency "$dependency" "$TMP/deb-depends"
done
for dependency in netcat-openbsd network-manager-l2tp openvpn wireguard-tools; do
    assert_dependency "$dependency" "$TMP/deb-recommends"
done
for dependency in \
    bash diffutils findutils gawk grep iproute jq nftables polkit procps-ng python3 sed socat \
    systemd util-linux; do
    assert_dependency "$dependency" "$TMP/rpm-requires"
done
for dependency in NetworkManager-l2tp nmap-ncat openvpn wireguard-tools; do
    assert_dependency "$dependency" "$TMP/rpm-recommends"
done

find "$deb_root" -type f -o -type l |
    sed "s#^$deb_root##" |
    sort >"$TMP/deb-files"
find "$rpm_root" -type f -o -type l |
    sed "s#^$rpm_root##" |
    sort >"$TMP/rpm-files"

for listing in "$TMP/deb-files" "$TMP/rpm-files"; do
    for path in \
        /usr/bin/mazzy-vpn \
        /usr/bin/mazzyvpn \
        /usr/bin/vpnctl \
        /usr/lib/mazzy-vpn/api/v1/manifest.json \
        /usr/lib/mazzy-vpn/agent-control/v1/registry.json \
        /usr/lib/mazzy-vpn/agent-control/v1/schema.json \
        /usr/lib/mazzy-vpn/agent-control/v1/envelope.schema.json \
        /usr/lib/mazzy-vpn/agent-control/v1/command.schema.json \
        /usr/lib/mazzy-vpn/docs/TARGET_ARCHITECTURE_2026-08-02.ru.md \
        /usr/lib/mazzy-vpn/docs/RESEARCH_AGENT_REMOTE_CONTROL_2026-08-02.ru.md \
        /usr/lib/mazzy-vpn/docs/R0_MUTATION_SINGLE_FLIGHT.ru.md \
        /usr/lib/mazzy-vpn/protocols/v1/registry.json \
        /usr/lib/mazzy-vpn/protocols/v1/schema.json \
        /usr/lib/mazzy-vpn/protocols/v1/managed-profile.schema.json \
        /usr/lib/mazzy-vpn/runtime/mazzy-sing-box-adapter \
        /usr/lib/mazzy-vpn/runtime/v1/adapter-registry.json \
        /usr/lib/mazzy-vpn/runtime/v1/schema.json \
        /usr/lib/mazzy-vpn/deny.toml \
        /usr/lib/mazzy-vpn/desktop/package-lock.json \
        /usr/lib/mazzy-vpn/desktop/package.json \
        /usr/lib/mazzy-vpn/desktop/scripts/build-release.mjs \
        /usr/lib/mazzy-vpn/desktop/scripts/tauri-audited.mjs \
        /usr/lib/mazzy-vpn/desktop/src-tauri/Cargo.lock \
        /usr/lib/mazzy-vpn/desktop/src-tauri/Cargo.toml \
        /usr/lib/mazzy-vpn/desktop/src-tauri/build.rs \
        /usr/lib/mazzy-vpn/desktop/src-tauri/examples/verify-updater-signatures.rs \
        /usr/lib/mazzy-vpn/desktop/src-tauri/capabilities/default.json \
        /usr/lib/mazzy-vpn/desktop/src-tauri/src/updater.rs \
        /usr/lib/mazzy-vpn/desktop/src-tauri/vendor/glib-0.18.5/Cargo.toml \
        /usr/lib/mazzy-vpn/desktop/src-tauri/vendor/glib-0.18.5/PATCH-PROVENANCE.md \
        /usr/lib/mazzy-vpn/desktop/src-tauri/vendor/glib-0.18.5/src/variant_iter.rs \
        /usr/lib/mazzy-vpn/desktop/ui/mazzy-vpn-logo.svg \
        /usr/lib/mazzy-vpn/rust-toolchain.toml \
        /usr/lib/systemd/system/mazzy-vpn-api.socket \
        /usr/lib/systemd/system/mazzy-vpn-api.socket.d/10-package-docs.conf \
        /usr/lib/systemd/system/mazzy-vpn-api-recovery.service \
        /usr/lib/systemd/system/mazzy-vpn-api-recovery.service.d/10-package-exec.conf \
        /usr/lib/systemd/system/mazzy-vpn-api@.service \
        /usr/lib/systemd/system/mazzy-vpn-api@.service.d/10-package-exec.conf \
        /usr/lib/systemd/system/vpnctl.service \
        /usr/lib/systemd/system/vpnctl.service.d/10-package-exec.conf \
        /usr/lib/systemd/system/vpnctl-health.service \
        /usr/lib/systemd/system/vpnctl-health.service.d/10-package-exec.conf \
        /usr/lib/systemd/system/vpnctl-health.timer \
        /usr/lib/systemd/system/vpnctl-health.timer.d/10-package-interval.conf \
        /usr/lib/systemd/system/vpnctl-test-recovery.service \
        /usr/lib/systemd/system/vpnctl-test-recovery.service.d/10-package-exec.conf \
        /usr/lib/tmpfiles.d/mazzy-vpn.conf \
        /usr/share/bash-completion/completions/mazzy-vpn; do
        assert_line "$path" "$listing"
    done
    if grep -q '^/usr/local/' "$listing"; then
        fail "distribution package writes into /usr/local"
    fi
    if grep -q '^/etc/vpnctl/' "$listing"; then
        fail "distribution package owns user profile/configuration state"
    fi
done

for extracted_root in "$deb_root" "$rpm_root"; do
    [[ "$(stat -c '%a' "$extracted_root/usr/bin/mazzy-vpn")" == 755 ]] ||
        fail "package engine mode is not exactly 0755"
    [[ "$(stat -c '%a' "$extracted_root/usr/bin/vpnctl")" == 755 ]] ||
        fail "package compatibility command mode is not exactly 0755"
    cmp -s "$ROOT/mazzy-vpn" "$extracted_root/usr/bin/mazzy-vpn" ||
        fail "package engine differs from source"
    cmp -s "$ROOT/api/v1/manifest.json" \
        "$extracted_root/usr/lib/mazzy-vpn/api/v1/manifest.json" ||
        fail "package API manifest differs from source"
    cmp -s "$ROOT/protocols/v1/registry.json" \
        "$extracted_root/usr/lib/mazzy-vpn/protocols/v1/registry.json" ||
        fail "package protocol registry differs from source"
    cmp -s "$ROOT/protocols/v1/managed-profile.schema.json" \
        "$extracted_root/usr/lib/mazzy-vpn/protocols/v1/managed-profile.schema.json" ||
        fail "package managed profile schema differs from source"
    cmp -s "$ROOT/runtime/mazzy-sing-box-adapter" \
        "$extracted_root/usr/lib/mazzy-vpn/runtime/mazzy-sing-box-adapter" ||
        fail "package sing-box adapter differs from source"
    [[ "$(stat -c '%a' "$extracted_root/usr/lib/mazzy-vpn/runtime/mazzy-sing-box-adapter")" == 755 ]] ||
        fail "package sing-box adapter mode is not exactly 0755"
    cmp -s "$ROOT/runtime/v1/adapter-registry.json" \
        "$extracted_root/usr/lib/mazzy-vpn/runtime/v1/adapter-registry.json" ||
        fail "package runtime adapter registry differs from source"
    cmp -s "$ROOT/agent-control/v1/registry.json" \
        "$extracted_root/usr/lib/mazzy-vpn/agent-control/v1/registry.json" ||
        fail "package agent-control registry differs from source"
    cmp -s "$ROOT/desktop/ui/app.css" \
        "$extracted_root/usr/lib/mazzy-vpn/desktop/ui/app.css" ||
        fail "package Desktop CSS differs from source"
    cmp -s "$ROOT/desktop/ui/app.js" \
        "$extracted_root/usr/lib/mazzy-vpn/desktop/ui/app.js" ||
        fail "package Desktop UI logic differs from source"
    cmp -s "$ROOT/desktop/package-lock.json" \
        "$extracted_root/usr/lib/mazzy-vpn/desktop/package-lock.json" ||
        fail "package npm lockfile differs from source"
    cmp -s "$ROOT/desktop/src-tauri/Cargo.lock" \
        "$extracted_root/usr/lib/mazzy-vpn/desktop/src-tauri/Cargo.lock" ||
        fail "package Cargo lockfile differs from source"
    cmp -s "$ROOT/desktop/src-tauri/src/updater.rs" \
        "$extracted_root/usr/lib/mazzy-vpn/desktop/src-tauri/src/updater.rs" ||
        fail "package Desktop updater source differs from source"
    cmp -s "$ROOT/desktop/src-tauri/vendor/glib-0.18.5/src/variant_iter.rs" \
        "$extracted_root/usr/lib/mazzy-vpn/desktop/src-tauri/vendor/glib-0.18.5/src/variant_iter.rs" ||
        fail "package vendored glib backport differs from source"
    cmp -s "$ROOT/tests/check-glib-backport.py" \
        "$extracted_root/usr/lib/mazzy-vpn/tests/check-glib-backport.py" ||
        fail "package glib provenance gate differs from source"
    cmp -s "$ROOT/desktop/scripts/tauri-audited.mjs" \
        "$extracted_root/usr/lib/mazzy-vpn/desktop/scripts/tauri-audited.mjs" ||
        fail "package audited release wrapper differs from source"
    cmp -s "$ROOT/packaging/linux/systemd/vpnctl.service.d/10-package-exec.conf" \
        "$extracted_root/usr/lib/systemd/system/vpnctl.service.d/10-package-exec.conf" ||
        fail "package service override differs from source"
    cmp -s "$ROOT/systemd/vpnctl.service" \
        "$extracted_root/usr/lib/systemd/system/vpnctl.service" ||
        fail "package VPN service unit differs from source"
    grep -Fxq 'RestartPreventExitStatus=77' \
        "$extracted_root/usr/lib/systemd/system/vpnctl.service" ||
        fail "package VPN service retries permanent authentication failures"
    grep -Fxq 'StartLimitIntervalSec=600' \
        "$extracted_root/usr/lib/systemd/system/vpnctl.service" ||
        fail "package VPN service has an unexpected start-limit interval"
    grep -Fxq 'StartLimitBurst=5' \
        "$extracted_root/usr/lib/systemd/system/vpnctl.service" ||
        fail "package VPN service has an unexpected start-limit burst"
    grep -Fxq 'Requires=mazzy-vpn-api-recovery.service' \
        "$extracted_root/usr/lib/systemd/system/vpnctl.service" ||
        fail "package VPN service can bypass failed API boot recovery"
    grep -Fxq \
        'After=network-online.target NetworkManager.service mazzy-vpn-api-recovery.service' \
        "$extracted_root/usr/lib/systemd/system/vpnctl.service" ||
        fail "package VPN service is not ordered after API boot recovery"
    cmp -s "$ROOT/systemd/mazzy-vpn-api-recovery.service" \
        "$extracted_root/usr/lib/systemd/system/mazzy-vpn-api-recovery.service" ||
        fail "package API recovery unit differs from source"
    grep -Fxq 'RemainAfterExit=yes' \
        "$extracted_root/usr/lib/systemd/system/mazzy-vpn-api-recovery.service" ||
        fail "package API recovery unit does not remain active after boot recovery"
    cmp -s \
        "$ROOT/packaging/linux/systemd/mazzy-vpn-api-recovery.service.d/10-package-exec.conf" \
        "$extracted_root/usr/lib/systemd/system/mazzy-vpn-api-recovery.service.d/10-package-exec.conf" ||
        fail "package API recovery override differs from source"
    for systemd_source in \
        systemd/mazzy-vpn-api.socket \
        packaging/linux/systemd/mazzy-vpn-api.socket.d/10-package-docs.conf \
        systemd/vpnctl-health.service \
        packaging/linux/systemd/vpnctl-health.service.d/10-package-exec.conf \
        systemd/vpnctl-health.timer \
        packaging/linux/systemd/vpnctl-health.timer.d/10-package-interval.conf \
        systemd/vpnctl-test-recovery.service \
        packaging/linux/systemd/vpnctl-test-recovery.service.d/10-package-exec.conf; do
        package_systemd_path="${systemd_source#packaging/linux/}"
        package_systemd_path="${package_systemd_path#systemd/}"
        cmp -s "$ROOT/$systemd_source" \
            "$extracted_root/usr/lib/systemd/system/$package_systemd_path" ||
            fail "package systemd payload differs from source: $package_systemd_path"
    done
done

for relative in \
    mazzy-vpn \
    install.sh \
    api/v1/manifest.json \
    agent-control/v1/registry.json \
    agent-control/v1/schema.json \
    agent-control/v1/envelope.schema.json \
    agent-control/v1/command.schema.json \
    protocols/v1/registry.json \
    protocols/v1/schema.json \
    protocols/v1/managed-profile.schema.json \
    runtime/mazzy-sing-box-adapter \
    runtime/v1/adapter-registry.json \
    runtime/v1/schema.json \
    deny.toml \
    desktop/package-lock.json \
    desktop/package.json \
    desktop/scripts/build-release.mjs \
    desktop/scripts/tauri-audited.mjs \
    desktop/src-tauri/Cargo.lock \
    desktop/src-tauri/Cargo.toml \
    desktop/src-tauri/build.rs \
    desktop/src-tauri/examples/verify-updater-signatures.rs \
    desktop/src-tauri/capabilities/default.json \
    desktop/src-tauri/icons/icon.png \
    desktop/src-tauri/vendor/glib-0.18.5/Cargo.toml \
    desktop/src-tauri/vendor/glib-0.18.5/PATCH-PROVENANCE.md \
    desktop/src-tauri/vendor/glib-0.18.5/src/variant_iter.rs \
    desktop/src-tauri/src/agent_control.rs \
    desktop/src-tauri/src/backend.rs \
    desktop/src-tauri/src/main.rs \
    desktop/src-tauri/src/updater.rs \
    desktop/src-tauri/tauri.conf.json \
    desktop/ui/app.css \
    desktop/ui/app.js \
    desktop/ui/index.html \
    desktop/ui/mazzy-vpn-logo.svg \
    packaging/linux/post-install.sh \
    packaging/linux/systemd/mazzy-vpn-api-recovery.service.d/10-package-exec.conf \
    packaging/linux/systemd/vpnctl-health.timer.d/10-package-interval.conf \
    rust-toolchain.toml \
    tests/check-capabilities.py \
    tests/check-glib-backport.py \
    tests/check-linux-packages.sh \
    tests/check-protocol-registry.py \
    tests/check-agent-control-registry.py \
    tests/check-managed-protocol-adapter.py \
    tests/check-runtime-adapter-registry.py \
    tests/run.sh \
    systemd/mazzy-vpn-api-recovery.service \
    systemd/vpnctl.service \
    wiki/Desktop-Dashboard-and-Tray.md; do
    [[ -f "$appimage_engine/$relative" ]] ||
        fail "AppImage embedded source is missing: $relative"
done
[[ "$(stat -c '%a' "$appimage_engine/mazzy-vpn")" == 755 ]] ||
    fail "AppImage embedded engine mode is not exactly 0755"
cmp -s "$ROOT/mazzy-vpn" "$appimage_engine/mazzy-vpn" ||
    fail "AppImage embedded engine differs from source"
cmp -s "$ROOT/desktop/ui/app.css" "$appimage_engine/desktop/ui/app.css" ||
    fail "AppImage embedded Desktop CSS differs from source"
cmp -s "$ROOT/desktop/ui/app.js" "$appimage_engine/desktop/ui/app.js" ||
    fail "AppImage embedded Desktop UI logic differs from source"
cmp -s "$ROOT/desktop/package-lock.json" \
    "$appimage_engine/desktop/package-lock.json" ||
    fail "AppImage embedded npm lockfile differs from source"
cmp -s "$ROOT/desktop/src-tauri/Cargo.lock" \
    "$appimage_engine/desktop/src-tauri/Cargo.lock" ||
    fail "AppImage embedded Cargo lockfile differs from source"
cmp -s "$ROOT/desktop/src-tauri/src/updater.rs" \
    "$appimage_engine/desktop/src-tauri/src/updater.rs" ||
    fail "AppImage embedded updater source differs from source"
cmp -s "$ROOT/desktop/src-tauri/vendor/glib-0.18.5/src/variant_iter.rs" \
    "$appimage_engine/desktop/src-tauri/vendor/glib-0.18.5/src/variant_iter.rs" ||
    fail "AppImage vendored glib backport differs from source"
cmp -s "$ROOT/tests/check-glib-backport.py" \
    "$appimage_engine/tests/check-glib-backport.py" ||
    fail "AppImage glib provenance gate differs from source"
cmp -s "$ROOT/desktop/scripts/tauri-audited.mjs" \
    "$appimage_engine/desktop/scripts/tauri-audited.mjs" ||
    fail "AppImage audited release wrapper differs from source"
cmp -s "$ROOT/desktop/src-tauri/tauri.conf.json" \
    "$appimage_engine/desktop/src-tauri/tauri.conf.json" ||
    fail "AppImage package configuration differs from source"
cmp -s "$ROOT/packaging/linux/post-install.sh" \
    "$appimage_engine/packaging/linux/post-install.sh" ||
    fail "AppImage package lifecycle source differs from source"
cmp -s "$ROOT/systemd/mazzy-vpn-api-recovery.service" \
    "$appimage_engine/systemd/mazzy-vpn-api-recovery.service" ||
    fail "AppImage API recovery unit source differs from source"
grep -Fxq 'RemainAfterExit=yes' \
    "$appimage_engine/systemd/mazzy-vpn-api-recovery.service" ||
    fail "AppImage API recovery source does not remain active after boot recovery"
cmp -s "$ROOT/systemd/vpnctl.service" \
    "$appimage_engine/systemd/vpnctl.service" ||
    fail "AppImage VPN service unit source differs from source"
cmp -s "$ROOT/tests/check-linux-packages.sh" \
    "$appimage_engine/tests/check-linux-packages.sh" ||
    fail "AppImage release audit differs from source"

python3 "$appimage_engine/tests/check-capabilities.py" >/dev/null ||
    fail "AppImage embedded capability registry is incomplete"
python3 "$appimage_engine/tests/check-api-contract.py" >/dev/null ||
    fail "AppImage embedded API contract is invalid"
python3 "$appimage_engine/tests/check-protocol-registry.py" >/dev/null ||
    fail "AppImage embedded protocol registry is invalid"
"$appimage_engine/install.sh" \
    --destdir "$appimage_stage" \
    --no-deps \
    --skip-tests \
    --skip-checks \
    --yes \
    --lang en >/dev/null ||
    fail "AppImage embedded installer source validation failed"

assert_gui_launch "DEB" "$deb_root/usr/bin/mazzy-vpn-desktop"
assert_gui_launch "RPM" "$rpm_root/usr/bin/mazzy-vpn-desktop"
assert_gui_launch "AppImage" env APPIMAGE_EXTRACT_AND_RUN=1 "$appimage"

echo "linux-package-audit: AppImage/DEB/RPM payload, dependencies, lifecycle and GUI launch verified"
