#!/usr/bin/env bash
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
# Strict read-only acceptance gate before and after a real reboot.

set -Eeuo pipefail

EXPECTED_DESKTOP_VERSION="${1:-0.4.8}"
EXPECTED_ENGINE_VERSION="${2:-1.4.7}"
PHASE="${3:-pre-reboot}"
[[ "$PHASE" == pre-reboot || "$PHASE" == post-reboot ]] || {
    printf 'usage: %s [desktop-version] [engine-version] [pre-reboot|post-reboot]\n' "$0" >&2
    exit 2
}
failures=0

ok() { printf '[ OK ] %s\n' "$*"; }
fail() { printf '[FAIL] %s\n' "$*" >&2; failures=$((failures + 1)); }

for command in dpkg-query systemctl jq nft stat grep runuser getent ip awk; do
    command -v "$command" >/dev/null 2>&1 || fail "missing command: $command"
done
((failures == 0)) || exit 1

version="$(dpkg-query -W -f='${Version}' mazzy-vpn-desktop 2>/dev/null || true)"
if [[ "$version" == "$EXPECTED_DESKTOP_VERSION" ]]; then
    ok "Desktop package $version"
else
    fail "Desktop package version=${version:-missing}, expected $EXPECTED_DESKTOP_VERSION"
fi
if dpkg -V mazzy-vpn-desktop | grep -q .; then
    fail "dpkg payload verification reports modified files"
else
    ok "dpkg payload verification is clean"
fi

for path in /usr/bin/mazzy-vpn-desktop /usr/bin/mazzy-agentd /usr/lib/mazzy-vpn/mazzy-vpn; do
    if [[ -x "$path" && ! -L "$path" ]]; then
        ok "executable $path"
    else
        fail "missing executable $path"
    fi
done
engine_version="$(/usr/lib/mazzy-vpn/mazzy-vpn version 2>/dev/null || true)"
if [[ "$engine_version" == "Mazzy VPN $EXPECTED_ENGINE_VERSION (mazzy-vpn; alias: vpnctl)" ]]; then
    ok "package engine $EXPECTED_ENGINE_VERSION"
else
    fail "unexpected engine version: $engine_version"
fi

units=(mazzy-vpn-api-recovery.service mazzy-vpn-api.socket vpnctl.service vpnctl-health.timer)
for unit in "${units[@]}"; do
    if systemctl is-enabled --quiet "$unit"; then ok "$unit enabled"; else fail "$unit not enabled"; fi
    if systemctl is-active --quiet "$unit"; then ok "$unit active"; else fail "$unit not active"; fi
done

for unit in mazzy-vpn-api.socket vpnctl.service vpnctl-health.service vpnctl-test-recovery.service; do
    requires="$(systemctl show "$unit" -p Requires --value 2>/dev/null || true)"
    if [[ " $requires " != *' mazzy-vpn-api-recovery.service '* ]]; then
        ok "$unit has no hard recovery dependency"
    else
        fail "$unit still hard-requires recovery"
    fi
done
effective_units="$(systemctl cat mazzy-vpn-api@.service vpnctl.service vpnctl-health.service vpnctl-test-recovery.service 2>/dev/null || true)"
if grep -Eq 'Exec(Start|Condition)=.*(/usr/bin|/usr/local/bin)/mazzy-vpn([[:space:]]|$)' <<<"$effective_units"; then
    fail "effective systemd graph depends on a public CLI path"
else
    ok "effective systemd graph uses the package engine"
fi

timer_line="$(systemctl list-timers --all --no-legend vpnctl-health.timer 2>/dev/null || true)"
if [[ -n "$timer_line" && "${timer_line%% *}" != '-' ]]; then
    ok "health timer has a finite NEXT"
else
    fail "health timer has no finite NEXT"
fi

api_user=""
if ((EUID == 0)); then
    if [[ "${PKEXEC_UID:-}" =~ ^[1-9][0-9]*$ ]]; then
        api_user="$(getent passwd "$PKEXEC_UID" | awk -F: '{print $1; exit}')"
    elif [[ -n "${SUDO_USER:-}" && "${SUDO_USER:-}" != root ]]; then
        api_user="$SUDO_USER"
    fi
fi
api_call() {
    if [[ -n "$api_user" ]]; then
        timeout 20 runuser -u "$api_user" -- /usr/bin/mazzy-vpn "$@"
    else
        timeout 20 /usr/bin/mazzy-vpn "$@"
    fi
}

status_json="$(api_call status --api-json 2>/dev/null || true)"
if jq -e --arg version "$EXPECTED_ENGINE_VERSION" '
    .status == "ok"
    and .result.engine_version == $version
    and .result.desired == "up"
    and .result.connection.state == "connected"
    and .result.connection.healthy == true
    and .result.connection.internet_reachable == true
' <<<"$status_json" >/dev/null 2>&1; then
    ok "local API reports a healthy connected tunnel"
else
    fail "local API tunnel acceptance failed"
fi
profiles_json="$(api_call profiles --api-json 2>/dev/null || true)"
if jq -e '
    .status == "ok"
    and ([.result.profiles[] | select(.protocol == "amneziawg")] | length) > 0
    and ([.result.profiles[] | select(.protocol == "openvpn")] | length) > 0
' <<<"$profiles_json" >/dev/null 2>&1; then
    ok "local API exposes AmneziaWG and OpenVPN profiles"
else
    fail "profile catalog is missing AmneziaWG or OpenVPN"
fi

if ((EUID != 0)); then
    fail "root is required to inspect protected recovery state; run through pkexec or sudo"
else
    marker_found=0
    for marker in \
        /var/lib/vpnctl/api-recovery-required.json \
        /var/lib/vpnctl/transition-recovery-required.json \
        /var/lib/vpnctl/test.transaction; do
        if [[ -e "$marker" ]]; then
            fail "protected recovery marker exists: $(basename -- "$marker")"
            marker_found=1
        fi
    done
    ((marker_found == 0)) && ok "no protected recovery marker is active"
    boot_state="$(cat /run/vpnctl/api-recovery.state 2>/dev/null || true)"
    if [[ "$boot_state" == ready ]]; then
        ok "boot recovery state is ready"
    else
        fail "boot recovery state=${boot_state:-missing}"
    fi
    if nft list table inet mazzy_vpn_transition >/dev/null 2>&1; then
        fail "transition guard remains installed"
    else
        ok "transition guard is absent after verified connection"
    fi
fi

if systemctl --failed --no-legend 2>/dev/null | grep -Eq 'mazzy-vpn|vpnctl'; then
    fail "a Mazzy VPN systemd unit is failed"
else
    ok "no Mazzy VPN systemd unit is failed"
fi

managed_rule_count="$(
    ip -4 rule show 2>/dev/null |
        awk '/suppress_prefixlength 0|fwmark 0xca6c.*lookup 51820/{count++} END{print count+0}'
)"
if [[ "$managed_rule_count" == 2 ]]; then
    ok "one managed policy-rule pair is present"
elif [[ "$PHASE" == post-reboot ]]; then
    fail "post-reboot managed policy-rule count=$managed_rule_count, expected 2"
else
    printf '[WARN] pre-reboot policy rules contain %s matching entries; post-reboot gate requires 2\n' \
        "$managed_rule_count"
fi

if ((failures > 0)); then
    printf 'Pre-reboot verdict: NO-GO (%d failures)\n' "$failures" >&2
    exit 1
fi
printf 'Pre-reboot verdict: GO\n'
