#!/usr/bin/env bash
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
CLI="$ROOT/mazzy-vpn"
COMPAT_CLI="$ROOT/vpnctl"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
REAL_PYTHON3="$(command -v python3)"
export REAL_PYTHON3

pass=0
fail() {
    printf 'FAIL: %s\n' "$*" >&2
    exit 1
}
ok() {
    pass=$((pass + 1))
    printf 'ok %d - %s\n' "$pass" "$*"
}

mkdir -p "$TMP/config/openvpn" "$TMP/config/wireguard" "$TMP/state" \
    "$TMP/run" "$TMP/fakebin"

cat >"$TMP/config/openvpn/Test Server.ovpn" <<'EOF'
# mazzy-country-code: BE
client
dev tun
proto udp
remote 192.0.2.10 1194
cipher AES-256-CBC
<ca>
test
</ca>
EOF
chmod 600 "$TMP/config/openvpn/Test Server.ovpn"

cat >"$TMP/config/wireguard/Unsafe.conf" <<'EOF'
[Interface]
PrivateKey = test
PostUp = touch /tmp/should-not-run
[Peer]
PublicKey = test
AllowedIPs = 0.0.0.0/0
Endpoint = 192.0.2.20:51820
EOF
chmod 600 "$TMP/config/wireguard/Unsafe.conf"

cat >"$TMP/fakebin/systemctl" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${FAKE_SYSTEMCTL_LOG:?}"
[[ -z "${FAKE_TRANSITION_LOG:-}" ]] ||
    printf 'systemctl %s\n' "$*" >>"$FAKE_TRANSITION_LOG"
[[ -z "${FAKE_DURABILITY_LOG:-}" ]] ||
    printf 'systemctl %s\n' "$*" >>"$FAKE_DURABILITY_LOG"

fake_activate_api_recovery() {
    printf 'activate mazzy-vpn-api-recovery.service\n' \
        >>"${FAKE_SYSTEMCTL_RECOVERY_LOG:?}"
    [[ "${FAKE_SYSTEMCTL_RECOVERY_FAIL:-0}" != "1" ]] || return 1
    if grep -Fxq 'RemainAfterExit=yes' \
            "${FAKE_SYSTEMCTL_RECOVERY_UNIT:?}"; then
        : >"${FAKE_SYSTEMCTL_RECOVERY_ACTIVE_FILE:?}"
    fi
}

if [[ "${FAKE_SYSTEMCTL_MODEL_RECOVERY_DEPENDENCY:-0}" == "1" ]]; then
    case "$*" in
        "start mazzy-vpn-api-recovery.service")
            [[ -e "${FAKE_SYSTEMCTL_RECOVERY_ACTIVE_FILE:?}" ]] ||
                fake_activate_api_recovery || exit 1
            ;;
        "start vpnctl.service"|"restart vpnctl.service"|\
        "start --no-block vpnctl.service"|"restart --no-block vpnctl.service"|\
        "start mazzy-vpn-api.socket"|"restart mazzy-vpn-api.socket"|\
        "start vpnctl-health.service"|"restart vpnctl-health.service"|\
        "start vpnctl-test-recovery.service"|"restart vpnctl-test-recovery.service")
            [[ -e "${FAKE_SYSTEMCTL_RECOVERY_ACTIVE_FILE:?}" ]] ||
                fake_activate_api_recovery || exit 1
            ;;
    esac
fi
if [[ -n "${FAKE_SYSTEMCTL_DELAY_ONCE_FILE:-}" &&
      ( -z "${FAKE_SYSTEMCTL_DELAY_ACTION:-}" ||
        "$*" == "${FAKE_SYSTEMCTL_DELAY_ACTION}" ) &&
      ! -e "$FAKE_SYSTEMCTL_DELAY_ONCE_FILE" ]]; then
    touch "$FAKE_SYSTEMCTL_DELAY_ONCE_FILE"
    printf '%s\n' "$$" >"${FAKE_SYSTEMCTL_DELAY_PID_FILE:?}"
    sleep "${FAKE_SYSTEMCTL_DELAY_ONCE_SECONDS:-10}"
fi
if [[ "${FAKE_SYSTEMCTL_DELAY_SECONDS:-0}" != "0" ]]; then
    sleep "$FAKE_SYSTEMCTL_DELAY_SECONDS"
fi
case "$*" in
    "show --property=ActiveEnterTimestampMonotonic vpnctl.service")
        [[ "${FAKE_SYSTEMCTL_ACTIVE_AGE_SECONDS:-}" =~ ^[0-9]+$ ]] || exit 1
        now_usec="${FAKE_MONOTONIC_NOW_USEC:-9000000000000}"
        [[ "$now_usec" =~ ^[1-9][0-9]*$ ]] || exit 1
        if [[ "${FAKE_SYSTEMCTL_MONOTONIC_MALFORMED:-0}" == "1" ]]; then
            printf 'ActiveEnterTimestampMonotonic=%s\nunexpected=true\n' \
                "$((now_usec - FAKE_SYSTEMCTL_ACTIVE_AGE_SECONDS * 1000000))"
        else
            printf 'ActiveEnterTimestampMonotonic=%s\n' \
                "$((now_usec - FAKE_SYSTEMCTL_ACTIVE_AGE_SECONDS * 1000000))"
        fi
        ;;
    reset-failed*vpnctl.service*)
        rm -f "${FAKE_SYSTEMCTL_COUNTER:?}"
        ;;
    "start vpnctl.service"|"restart vpnctl.service"|\
    "start --no-block vpnctl.service"|"restart --no-block vpnctl.service")
        [[ "${FAKE_SYSTEMCTL_START_FAIL:-0}" == "1" ]] && exit 1
        if [[ -n "${FAKE_SYSTEMCTL_FAIL_ONCE_FILE:-}" &&
              ! -e "$FAKE_SYSTEMCTL_FAIL_ONCE_FILE" ]]; then
            touch "$FAKE_SYSTEMCTL_FAIL_ONCE_FILE"
            exit 1
        fi
        if [[ "${FAKE_SYSTEMCTL_START_LIMIT:-0}" =~ ^[0-9]+$ ]] &&
           ((FAKE_SYSTEMCTL_START_LIMIT > 0)); then
            count=0
            [[ ! -r "${FAKE_SYSTEMCTL_COUNTER:?}" ]] ||
                read -r count <"$FAKE_SYSTEMCTL_COUNTER"
            count=$((count + 1))
            printf '%s\n' "$count" >"$FAKE_SYSTEMCTL_COUNTER"
            ((count <= FAKE_SYSTEMCTL_START_LIMIT)) || exit 1
        fi
        if [[ "${FAKE_SYSTEMCTL_OPENVPN_LIMIT:-0}" == "1" ]]; then
            printf '%s\n' openvpn-too-many-connections \
                >"${VPNCTL_RUN_DIR:?}/test.failure"
        fi
        [[ -z "${FAKE_SYSTEMCTL_ACTIVATE_FILE:-}" ]] ||
            : >"$FAKE_SYSTEMCTL_ACTIVATE_FILE"
        ;;
    "stop vpnctl.service")
        [[ "${FAKE_SYSTEMCTL_STOP_FAIL:-0}" == "1" ]] && exit 1
        ;;
    "is-active vpnctl.service")
        if [[ "${FAKE_SYSTEMCTL_INACTIVE:-0}" == "1" &&
              ( -z "${FAKE_SYSTEMCTL_ACTIVATE_FILE:-}" ||
                ! -e "$FAKE_SYSTEMCTL_ACTIVATE_FILE" ) ]]; then
            exit 3
        fi
        exit 0
        ;;
    *is-active*) exit 0 ;;
    *cat*) exit 0 ;;
esac
exit 0
EOF

cat >"$TMP/fakebin/python3" <<'EOF'
#!/usr/bin/env bash
if [[ "${1:-}" == "-I" && "${2:-}" == "-c" &&
      "${3:-}" == *"time.CLOCK_MONOTONIC"* ]]; then
    [[ "${FAKE_PYTHON_MONOTONIC_FAIL:-0}" != "1" ]] || exit 1
    if [[ "${FAKE_PYTHON_MONOTONIC_MALFORMED:-0}" == "1" ]]; then
        printf 'not-a-timestamp\n'
    else
        printf '%s\n' "${FAKE_MONOTONIC_NOW_USEC:-9000000000000}"
    fi
    exit 0
fi
exec "${REAL_PYTHON3:?}" "$@"
EOF

cat >"$TMP/fakebin/nft" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${FAKE_NFT_LOG:?}"
case "$*" in
    "delete table inet mazzy_vpn_transition")
        [[ -z "${FAKE_TRANSITION_LOG:-}" ]] ||
            printf 'guard remove\n' >>"$FAKE_TRANSITION_LOG"
        exit 0
        ;;
    "-f -")
        rules="$(cat)"
        printf '%s\n' "$rules" >>"${FAKE_NFT_RULES_LOG:?}"
        [[ -z "${FAKE_TRANSITION_LOG:-}" ]] ||
            printf 'guard install\n' >>"$FAKE_TRANSITION_LOG"
        [[ "${FAKE_NFT_INSTALL_FAIL:-0}" != "1" ]] || exit 1
        ;;
esac
EOF

cat >"$TMP/fakebin/timeout" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${FAKE_TIMEOUT_LOG:?}"
exec /usr/bin/timeout "$@"
EOF

cat >"$TMP/fakebin/systemd-run" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${FAKE_SYSTEMD_RUN_LOG:?}"
EOF

cat >"$TMP/fakebin/journalctl" <<'EOF'
#!/usr/bin/env bash
printf 'fake journal: %s\n' "$*"
EOF

cat >"$TMP/fakebin/curl" <<'EOF'
#!/usr/bin/env bash
[[ "${FAKE_CURL_FAIL:-0}" == "1" ]] && exit 28
has_interface=false
url=""
for argument in "$@"; do
    [[ "$argument" == "--interface" ]] && has_interface=true
    url="$argument"
done
case "$url" in
    *\[2606:4700:4700::1111\]/cdn-cgi/trace*)
        [[ -z "${FAKE_TRANSITION_CURL_LOG:-}" ]] ||
            printf '%s\n' "$*" >>"$FAKE_TRANSITION_CURL_LOG"
        [[ -z "${FAKE_TRANSITION_LOG:-}" ]] ||
            printf 'curl transition probe v6\n' >>"$FAKE_TRANSITION_LOG"
        printf 'fl=test\nh=2606:4700:4700::1111\nip=%s\nvisit_scheme=https\nwarp=off\n' \
            "${FAKE_BOUND_IPV6:-2001:db8::7}"
        [[ "${FAKE_TRANSITION_PROBE_DUP_IP:-0}" != "1" ]] ||
            printf 'ip=2001:db8::9\n'
        ;;
    *1.1.1.1/cdn-cgi/trace*)
        [[ -z "${FAKE_TRANSITION_CURL_LOG:-}" ]] ||
            printf '%s\n' "$*" >>"$FAKE_TRANSITION_CURL_LOG"
        [[ -z "${FAKE_TRANSITION_LOG:-}" ]] ||
            printf 'curl transition probe\n' >>"$FAKE_TRANSITION_LOG"
        printf 'fl=test\nh=1.1.1.1\nip=%s\n' \
            "${FAKE_BOUND_IPV4:-203.0.113.7}"
        [[ "${FAKE_TRANSITION_PROBE_DUP_IP:-0}" != "1" ]] ||
            printf 'ip=198.51.100.9\n'
        printf 'visit_scheme=https\nwarp=off\n'
        ;;
    *probe.invalid*)
        [[ -z "${FAKE_TRANSITION_CURL_LOG:-}" ]] ||
            printf '%s\n' "$*" >>"$FAKE_TRANSITION_CURL_LOG"
        [[ -z "${FAKE_TRANSITION_LOG:-}" ]] ||
            printf 'curl hostname probe\n' >>"$FAKE_TRANSITION_LOG"
        [[ "${FAKE_DNS_UNAVAILABLE:-0}" != "1" ]] || exit 6
        if [[ "$has_interface" == true ]]; then
            printf '%s' "${FAKE_BOUND_IPV4:-203.0.113.7}"
        else
            printf '%s' "${FAKE_DEFAULT_IPV4:-203.0.113.7}"
        fi
        ;;
    *notebooklm.google.com*|*chatgpt.com/backend-api/codex/responses*)
        [[ -z "${FAKE_SERVICE_CURL_LOG:-}" ]] ||
            printf '%s\n' "$*" >>"$FAKE_SERVICE_CURL_LOG"
        if [[ -n "${ALL_PROXY:-}${all_proxy:-}${HTTP_PROXY:-}${http_proxy:-}${HTTPS_PROXY:-}${https_proxy:-}${NO_PROXY:-}${no_proxy:-}" ]]; then
            exit 96
        fi
        [[ "$has_interface" == true ]] || exit 97
        [[ -z "${FAKE_SERVICE_PID_FILE:-}" ]] ||
            printf '%s\n' "$$" >"$FAKE_SERVICE_PID_FILE"
        if [[ -n "${FAKE_SERVICE_DELAY_SECONDS:-}" ]]; then
            sleep "$FAKE_SERVICE_DELAY_SECONDS"
        fi
        status="${FAKE_SERVICE_STATUS:-302}"
        printf 'HTTP/2 %s\r\n' "$status"
        if [[ "${FAKE_SERVICE_OVERSIZE:-0}" == "1" ]]; then
            printf 'X-Oversized: '
            head -c 9000 /dev/zero | tr '\0' x
            printf '\r\n'
        fi
        if [[ -n "${FAKE_SERVICE_LOCATION:-}" ]]; then
            printf 'Location: %s\r\n' "$FAKE_SERVICE_LOCATION"
            [[ "${FAKE_SERVICE_DUP_LOCATION:-0}" != "1" ]] ||
                printf 'Location: %s\r\n' "$FAKE_SERVICE_LOCATION"
        fi
        [[ -z "${FAKE_SERVICE_ALLOW:-}" ]] ||
            printf 'Allow: %s\r\n' "$FAKE_SERVICE_ALLOW"
        printf '\r\n'
        ;;
    *api6.ipify.org*)
        if [[ "$has_interface" == true ]]; then
            value="${FAKE_BOUND_IPV6:-}"
        else
            value="${FAKE_DEFAULT_IPV6:-}"
        fi
        [[ -n "$value" ]] || exit 7
        printf '%s' "$value"
        ;;
    *ipapi.co*)
        [[ "${FAKE_GEO_FAIL:-0}" == "1" ]] && exit 28
        printf '{"ip":"%s","country_code":"BE","country_name":"Belgium","region":"Brussels","city":"Brussels"}' \
            "${FAKE_GEO_IPV4:-203.0.113.7}"
        ;;
    *ipwho.is*)
        [[ "${FAKE_GEO_FAIL:-0}" == "1" ]] && exit 28
        if [[ "${FAKE_GEO_MISMATCH:-0}" == "1" ]]; then
            country_code=DE
            country=Germany
            region=Berlin
            city=Berlin
        else
            country_code=BE
            country=Belgium
            region=Brussels
            city=Brussels
        fi
        printf '{"success":true,"ip":"%s","country_code":"%s","country":"%s","region":"%s","city":"%s"}' \
            "${FAKE_GEO_IPV4:-203.0.113.7}" \
            "$country_code" "$country" "$region" "$city"
        ;;
    *speed.cloudflare.com*)
        [[ "${FAKE_SPEED_FAIL:-0}" == "1" ]] && exit 28
        printf '12500000\t0.040'
        ;;
    *)
        if [[ "$has_interface" == true ]]; then
            printf '%s' "${FAKE_BOUND_IPV4:-203.0.113.7}"
        else
            printf '%s' "${FAKE_DEFAULT_IPV4:-203.0.113.7}"
        fi
        ;;
esac
EOF

cat >"$TMP/fakebin/ping" <<'EOF'
#!/usr/bin/env bash
[[ "${FAKE_PING_FAIL:-0}" == "1" ]] && exit 1
if [[ -n "${FAKE_PING_STARTED_FILE:-}" ]]; then
    : >"$FAKE_PING_STARTED_FILE"
fi
if [[ -n "${FAKE_PING_PID_FILE:-}" ]]; then
    printf '%s\n' "$$" >"$FAKE_PING_PID_FILE"
fi
if [[ -n "${FAKE_PING_DELAY_SECONDS:-}" ]]; then
    sleep "$FAKE_PING_DELAY_SECONDS"
fi
printf '64 bytes from 192.0.2.10: icmp_seq=1 ttl=58 time=12.4 ms\n'
exit 0
EOF

cat >"$TMP/fakebin/getent" <<'EOF'
#!/usr/bin/env bash
[[ "${FAKE_GETENT_FAIL:-0}" == "1" ]] && exit 2
/usr/bin/getent "$@"
EOF

cat >"$TMP/fakebin/sync" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${FAKE_SYNC_LOG:?}"
[[ -z "${FAKE_DURABILITY_LOG:-}" ]] ||
    printf 'sync %s\n' "$*" >>"$FAKE_DURABILITY_LOG"
[[ "${FAKE_SYNC_FAIL_TARGET:-}" != "${*: -1}" ]] || exit 1
exec /usr/bin/sync "$@"
EOF

cat >"$TMP/fakebin/openvpn" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >"${FAKE_OPENVPN_LOG:?}"
if [[ -n "${FAKE_OPENVPN_DELAY_SECONDS:-}" ]]; then
    sleep "$FAKE_OPENVPN_DELAY_SECONDS"
fi
if [[ "${FAKE_OPENVPN_TOO_MANY:-0}" == "1" ]]; then
    printf "Halt command was pushed by server ('Too many connections')\n" >&2
fi
if [[ "${FAKE_OPENVPN_AUTH_FAILED:-0}" == "1" ]]; then
    printf 'AUTH_FAILED\n' >&2
fi
EOF

cat >"$TMP/fakebin/resolvectl" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${FAKE_RESOLVECTL_LOG:?}"
case "${1:-}" in
    dns)
        [[ "${FAKE_DNS_EMPTY:-0}" == "1" ]] ||
            printf 'Link 7 (%s): 9.9.9.9\n' "${2:-vpnovpn0}"
        ;;
    domain)
        if [[ "${FAKE_DNS_NO_DEFAULT_ROUTE:-0}" == "1" ]]; then
            printf 'Link 7 (%s): ~corp.example\n' "${2:-vpnovpn0}"
        else
            printf 'Link 7 (%s): ~.\n' "${2:-vpnovpn0}"
        fi
        ;;
esac
EOF

cat >"$TMP/fakebin/socat" <<'EOF'
#!/usr/bin/env bash
if [[ "${FAKE_SOCAT_OVERSIZED:-0}" == "1" ]]; then
    head -c 2048 /dev/zero | tr '\0' x
    exit 0
fi
if [[ "${FAKE_SOCAT_MULTIPLE_RESPONSES:-0}" == "1" ]]; then
    request="$(cat)"
    for _ in 1 2; do
        printf '%s\n' "$request" | "${FAKE_SOCAT_DISPATCH:?}" _api-dispatch
    done
    exit 0
fi
if [[ -n "${FAKE_SOCAT_LOST_RESPONSE_FILE:-}" &&
      ! -e "$FAKE_SOCAT_LOST_RESPONSE_FILE" ]]; then
    touch "$FAKE_SOCAT_LOST_RESPONSE_FILE"
    "${FAKE_SOCAT_DISPATCH:?}" _api-dispatch >/dev/null
    exit 1
fi
exec "${FAKE_SOCAT_DISPATCH:?}" _api-dispatch
EOF

cat >"$TMP/fakebin/ip" <<'EOF'
#!/usr/bin/env bash
case "$*" in
    "-4 rule show")
        [[ ! -e "${FAKE_IP_RULES:?}" ]] || cat "$FAKE_IP_RULES"
        exit 0
        ;;
    "-4 rule delete priority "*)
        printf '%s\n' "$*" >>"${FAKE_IP_LOG:?}"
        exit 0
        ;;
    "-4 route flush cache") exit 0 ;;
    "link show wg0")
        [[ -e "${FAKE_LEGACY_ACTIVE:?}" ]] && exit 0
        exit 1
        ;;
    "link show vpnwg0") exit 0 ;;
    "link show adguard0") exit 0 ;;
    "link show vpnovpn0")
        [[ "${FAKE_OPENVPN_INTERFACE_MISSING:-0}" != "1" ]]
        exit $?
        ;;
    "-o -4 address show dev "*" scope global")
        [[ "${FAKE_IPV6_ONLY:-0}" != "1" ]] || exit 0
        ;;
    "-o -6 address show dev "*" scope global")
        if [[ "${FAKE_IPV6_ONLY:-0}" == "1" ]]; then
            printf '7: vpnovpn0 inet6 2001:db8::2/64 scope global\n'
        fi
        exit 0
        ;;
    "route show default") printf 'default via 192.0.2.1 dev eth0\n'; exit 0 ;;
    "-o route get 1.1.1.1") printf '1.1.1.1 dev adguard0 src 10.0.0.2\n'; exit 0 ;;
    "-o route get 192.0.2.40") printf '192.0.2.40 dev eth0 src 198.51.100.2\n'; exit 0 ;;
esac
/usr/sbin/ip "$@"
EOF

cat >"$TMP/fakebin/ss" <<'EOF'
#!/usr/bin/env bash
pid_file="${VPNCTL_ADGUARD_PID_FILE:-}"
if [[ "$*" == "-H -n -t -u -p" && -n "$pid_file" && -r "$pid_file" ]]; then
    read -r pid <"$pid_file"
    if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
        printf 'udp ESTAB 0 0 10.0.0.2:51820 192.0.2.40:443 users:(("adguardvpn-cli",pid=%s,fd=3))\n' \
            "$pid"
    fi
fi
EOF

cat >"$TMP/fakebin/awg" <<'EOF'
#!/usr/bin/env bash
[[ "$*" == "show interfaces" ]] && exit 0
if [[ "$*" == "show wg0 endpoints" ]]; then
    printf 'test-peer 192.0.2.41:51820\n'
    exit 0
fi
exit 0
EOF

cat >"$TMP/fakebin/wg" <<'EOF'
#!/usr/bin/env bash
[[ "$*" == "show interfaces" ]] && exit 0
exit 0
EOF

cat >"$TMP/fakebin/nmcli" <<'EOF'
#!/usr/bin/env bash
if [[ "$*" == "-t -f NAME connection show" ]]; then
    printf 'Test connection\n'
fi
exit 0
EOF

for mock_command in wg-quick awg-quick nm-l2tp-service \
    charon pppd xl2tpd; do
    ln -s /bin/true "$TMP/fakebin/$mock_command"
done
mkdir -p "$TMP/doctorbin"
ln -s /bin/true "$TMP/doctorbin/amneziawg-go"

cat >"$TMP/fakebin/adguardvpn-cli" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${FAKE_ADGUARD_LOG:?}"
case "${1:-}" in
    status)
        if [[ -e "${FAKE_ADGUARD_ACTIVE:?}" ]]; then
            printf 'Connected to TEST in TUN mode, running on tun0\n'
            exit 0
        fi
        printf 'VPN is disconnected\n'
        exit 0
        ;;
    disconnect)
        rm -f "${FAKE_ADGUARD_ACTIVE:?}"
        ;;
    connect)
        touch "${FAKE_ADGUARD_ACTIVE:?}"
        if [[ "${FAKE_ADGUARD_FORK:-0}" == "1" ]]; then
            (sleep 5) >/dev/null 2>&1 &
        fi
        ;;
esac
EOF

cat >"$TMP/fallback-start" <<'EOF'
#!/usr/bin/env bash
printf 'start\n' >>"${FAKE_LEGACY_LOG:?}"
touch "${FAKE_LEGACY_ACTIVE:?}"
EOF

cat >"$TMP/fallback-stop" <<'EOF'
#!/usr/bin/env bash
printf 'stop\n' >>"${FAKE_LEGACY_LOG:?}"
rm -f "${FAKE_LEGACY_ACTIVE:?}"
EOF

find "$TMP/fakebin" -maxdepth 1 -type f -exec chmod +x {} +
chmod +x "$TMP/fallback-start" "$TMP/fallback-stop"
export PATH="$TMP/fakebin:$PATH"
export FAKE_SYSTEMCTL_LOG="$TMP/systemctl.log"
export FAKE_SYSTEMCTL_RECOVERY_ACTIVE_FILE="$TMP/systemd-recovery.active"
export FAKE_SYSTEMCTL_RECOVERY_LOG="$TMP/systemd-recovery.log"
export FAKE_SYSTEMCTL_RECOVERY_UNIT="$ROOT/systemd/mazzy-vpn-api-recovery.service"
export FAKE_TRANSITION_LOG="$TMP/transition.log"
export FAKE_TRANSITION_CURL_LOG="$TMP/transition-curl.log"
export FAKE_NFT_LOG="$TMP/nft.log"
export FAKE_NFT_RULES_LOG="$TMP/nft-rules.log"
export FAKE_SYSTEMCTL_COUNTER="$TMP/systemctl.counter"
export FAKE_TIMEOUT_LOG="$TMP/timeout.log"
export FAKE_SYSTEMD_RUN_LOG="$TMP/systemd-run.log"
export FAKE_SERVICE_CURL_LOG="$TMP/service-curl.log"
export FAKE_SERVICE_PID_FILE="$TMP/service-curl.pid"
export FAKE_SYNC_LOG="$TMP/sync.log"
export FAKE_DURABILITY_LOG="$TMP/durability.log"
export FAKE_OPENVPN_LOG="$TMP/openvpn.log"
export FAKE_RESOLVECTL_LOG="$TMP/resolvectl.log"
export FAKE_SOCAT_DISPATCH="$CLI"
export FAKE_IP_LOG="$TMP/ip.log"
export FAKE_IP_RULES="$TMP/ip.rules"
export FAKE_ADGUARD_LOG="$TMP/adguard.log"
export FAKE_ADGUARD_ACTIVE="$TMP/adguard.active"
export VPNCTL_ADGUARD_PID_FILE="$TMP/adguard.pid"
export FAKE_LEGACY_LOG="$TMP/legacy.log"
export FAKE_LEGACY_ACTIVE="$TMP/legacy.active"
export VPNCTL_ALLOW_UNPRIVILEGED=1
export VPNCTL_CONFIG_DIR="$TMP/config"
export VPNCTL_STATE_DIR="$TMP/state"
export VPNCTL_API_ACTION_DIR="$TMP/state/api-actions"
export VPNCTL_API_AUDIT_FILE="$TMP/state/api-audit.jsonl"
export VPNCTL_RUN_DIR="$TMP/run"
export VPNCTL_API_SOCKET="$TMP/run/api-v1.sock"
export VPNCTL_DASHBOARD_DIR="$TMP/dashboard"
export VPNCTL_PROBE_URL="https://probe.invalid"
export VPNCTL_ADGUARD_CLI="$TMP/fakebin/adguardvpn-cli"
export VPNCTL_LOCALE_FILE="$TMP/system-language"
export VPNCTL_USER_LOCALE_FILE="$TMP/user-language"
VPNCTL_FALLBACK_USER="$(id -un)"
export VPNCTL_FALLBACK_USER
export VPNCTL_LEGACY_START="$TMP/fallback-start"
export VPNCTL_LEGACY_STOP="$TMP/fallback-stop"
export NO_COLOR=1

"$CLI" version | grep -q '^Mazzy VPN 1\.4\.1 (mazzy-vpn; alias: vpnctl)$'
"$COMPAT_CLI" version | grep -q '^Mazzy VPN 1\.4\.1 ' ||
    fail "vpnctl compatibility wrapper is broken"
ok "Mazzy VPN branding and compatibility alias"

language_list="$("$CLI" language list)"
for language_code in ru en de zh ja ko; do
    grep -q " $language_code —" <<<" $language_list" ||
        fail "language list is missing $language_code"
done
for language_check in \
    'ru:Использование:' \
    'en:Usage:' \
    'de:Verwendung:' \
    'zh:用法：' \
    'ja:使い方:' \
    'ko:사용법:'; do
    language_code="${language_check%%:*}"
    language_marker="${language_check#*:}"
    localized_help="$(VPNCTL_LANG="$language_code" "$CLI" help)" ||
        fail "localized help failed for $language_code"
    grep -q "$language_marker" <<<"$localized_help" ||
        fail "localized help is missing for $language_code"
done
"$CLI" language de >/dev/null
[[ "$(<"$VPNCTL_USER_LOCALE_FILE")" == "de" ]] ||
    fail "language command did not persist German"
"$CLI" language ru >/dev/null
if "$CLI" language unsupported >/dev/null 2>&1; then
    fail "unsupported language was accepted"
fi
ok "six interface languages are selectable and persisted"

list_output="$("$CLI" list openvpn)"
grep -q 'Test Server' <<<"$list_output" || fail "profile with spaces was not listed"
grep -q '192.0.2.10:1194' <<<"$list_output" || fail "endpoint was not shown"
ok "list handles spaces"

metadata_profile="$TMP/config/openvpn/opaque-profile.ovpn"
cat >"$metadata_profile" <<'EOF'
# mazzy-name-extra: must not override the exact directive
# mazzy-name: AI Workspace
# mazzy-location-note: must not override the exact directive
# mazzy-location: Belgium — Brussels
# mazzy-country-code-backup: RU
# mazzy-country-code: be
client
dev tun
proto tcp
remote 192.0.2.11 443
<ca>
test
</ca>
EOF
chmod 600 "$metadata_profile"
metadata_list="$("$CLI" list openvpn)"
grep -q 'AI Workspace' <<<"$metadata_list" ||
    fail "profile display name was not parsed from config metadata"
rm -f -- "$VPNCTL_DASHBOARD_DIR/profiles.json"
metadata_json="$("$CLI" profiles --json)"
jq -e '
    .profiles[]
    | select(.file_name == "opaque-profile.ovpn")
    | .name == "AI Workspace"
      and .location == "Belgium — Brussels"
      and .country_code == "BE"
' <<<"$metadata_json" >/dev/null ||
    fail "profile location/country metadata was not parsed from the config"
if grep -Eq '192\.0\.2\.11|remote' <<<"$metadata_json"; then
    fail "profile catalog exposed an endpoint while parsing location metadata"
fi
rm -f -- "$metadata_profile" "$VPNCTL_DASHBOARD_DIR/profiles.json"
ok "profile names and locations come from config metadata with filename fallback"

unsafe_profile_name="$TMP/config/openvpn/"$'Bell\aServer.ovpn'
cp "$TMP/config/openvpn/Test Server.ovpn" "$unsafe_profile_name"
unsafe_name_list="$("$CLI" list openvpn 2>&1)"
grep -q 'имя содержит управляющие символы' <<<"$unsafe_name_list" ||
    fail "profile catalog did not report a control-character filename"
grep -q 'Bell' <<<"$unsafe_name_list" &&
    fail "profile catalog exposed a control-character filename"
if "$CLI" import openvpn "$unsafe_profile_name" >/dev/null 2>&1; then
    fail "profile import accepted a control-character filename"
fi
rm -f -- "$unsafe_profile_name"
ok "profile catalog rejects terminal control characters"

bidi_profile_name="$TMP/config/openvpn/"$'office\u202Econf.ovpn'
cp "$TMP/config/openvpn/Test Server.ovpn" "$bidi_profile_name"
bidi_name_list="$("$CLI" list openvpn 2>&1)"
grep -q 'имя содержит управляющие символы или скрытые Unicode-маркеры' \
    <<<"$bidi_name_list" ||
    fail "profile catalog did not report a Unicode direction marker"
rm -f -- "$bidi_profile_name"
ok "profile catalog rejects Unicode direction and zero-width spoofing markers"

validate_output="$("$CLI" validate openvpn)"
grep -q 'profiles=1 passed=1 failed=0' <<<"$validate_output" ||
    fail "valid profile was not accepted by validate"
ok "validate checks every selected profile"

probe_output="$("$CLI" probe openvpn --timeout 1)"
grep -q 'endpoints=1 reachable=1 unknown=0 unreachable=0 invalid=0' \
    <<<"$probe_output" ||
    fail "endpoint DNS/ping probe did not pass"
grep -q '12 ms (ICMP)' <<<"$probe_output" ||
    fail "endpoint probe did not report measured ICMP latency"
probe_json="$("$CLI" probe openvpn --timeout 1 --jobs 2 --json)"
jq -e '
    .schema_version == 1
    and .concurrency == 2
    and .summary == {
        total: 1,
        reachable: 1,
        unknown: 0,
        unreachable: 0,
        invalid: 0,
        active: 0
    }
    and .results[0].display_name == "Test Server"
    and .results[0].reachability == "reachable"
    and .results[0].latency_ms == 12
    and .results[0].latency_source == "icmp"
' <<<"$probe_json" >/dev/null ||
    fail "structured endpoint probe omitted reachability or latency"
if grep -Eq '192\.0\.2\.10|remote|endpoint' <<<"$probe_json"; then
    fail "structured endpoint probe leaked the VPN endpoint"
fi
probe_unknown="$(
    FAKE_PING_FAIL=1 "$CLI" probe openvpn --timeout 1 --json
)"
jq -e '
    .summary.unknown == 1
    and .summary.unreachable == 0
    and .results[0].reachability == "unknown"
    and .results[0].latency_ms == null
' <<<"$probe_unknown" >/dev/null ||
    fail "blocked ICMP incorrectly marked a UDP VPN endpoint unavailable"
if probe_dns_failure="$(
    FAKE_GETENT_FAIL=1 "$CLI" probe openvpn --timeout 1 --json
)"; then
    fail "DNS failure did not fail the endpoint probe"
fi
jq -e '
    .summary.unreachable == 1
    and .results[0].message_key == "probe.unreachable.dns"
' <<<"$probe_dns_failure" >/dev/null ||
    fail "DNS failure did not produce a structured unreachable result"
cp "$TMP/config/openvpn/Test Server.ovpn" \
    "$TMP/config/openvpn/Test Server Parallel.ovpn"
sed -i 's/192\.0\.2\.10/192.0.2.11/' \
    "$TMP/config/openvpn/Test Server Parallel.ovpn"
parallel_probe_json="$(
    FAKE_PING_DELAY_SECONDS=2 \
        "$CLI" probe openvpn --timeout 3 --jobs 2 --json
)"
jq -e '
    .summary.total == 2
    and .summary.reachable == 2
    and .concurrency == 2
    and .duration_ms < 3500
' <<<"$parallel_probe_json" >/dev/null ||
    fail "two-worker endpoint probe ran sequentially or lost a location"
rm -f -- "$TMP/config/openvpn/Test Server Parallel.ovpn"
ok "bounded endpoint probe reports per-location reachability and latency"

cat >"$TMP/config/openvpn/Unsafe Include.ovpn" <<'EOF'
client
dev tun
remote 192.0.2.11 1194
config /tmp/unsafe-openvpn.conf
EOF
chmod 600 "$TMP/config/openvpn/Unsafe Include.ovpn"
if "$CLI" validate openvpn >/dev/null 2>&1; then
    fail "nested OpenVPN config directive bypassed validation"
fi
rm "$TMP/config/openvpn/Unsafe Include.ovpn"
ok "OpenVPN nested config is rejected"

mkdir -p "$TMP/import-source/mixed/nested" "$TMP/import-target"
cat >"$TMP/import-source/mixed/AWG.conf" <<'EOF'
[Interface]
PrivateKey = test
Address = 10.0.0.2/32
Jc = 4
Jmin = 40
Jmax = 70
S1 = 1
S2 = 2
H1 = 3
H2 = 4
H3 = 5
H4 = 6
[Peer]
PublicKey = test
AllowedIPs = 0.0.0.0/0
Endpoint = 192.0.2.30:51820
EOF
cat >"$TMP/import-source/mixed/WG.conf" <<'EOF'
[Interface]
PrivateKey = test
Address = 10.0.0.3/32
[Peer]
PublicKey = test
AllowedIPs = 0.0.0.0/0
Endpoint = 192.0.2.31:51820
EOF
cat >"$TMP/import-source/mixed/nested/Auto OpenVPN.conf" <<'EOF'
client
dev tun
proto udp
remote 192.0.2.32 1194
EOF
cat >"$TMP/import-source/mixed/Office.nmconnection" <<'EOF'
[connection]
id=Office
type=vpn
[vpn]
service-type=org.freedesktop.NetworkManager.l2tp
gateway=192.0.2.33
EOF
cat >"$TMP/import-source/mixed/Unsafe.ovpn" <<'EOF'
client
dev tun
remote 192.0.2.34 1194
up /tmp/unsafe-script
EOF
cat >"$TMP/import-source/mixed/Unknown.conf" <<'EOF'
not a vpn profile
EOF
find "$TMP/import-source" -type f -exec chmod 644 {} +

if import_scan="$(
    VPNCTL_CONFIG_DIR="$TMP/import-target" \
        "$CLI" import-dir "$TMP/import-source" --dry-run 2>&1
)"; then
    fail "folder scan accepted a recognized unsafe profile"
fi
grep -q 'detected=5' <<<"$import_scan" || fail "folder scan protocol detection count is wrong"
grep -q 'invalid=1' <<<"$import_scan" || fail "folder scan did not reject unsafe config"
if VPNCTL_CONFIG_DIR="$TMP/import-target" \
    "$CLI" import-dir "$TMP/import-source" >/dev/null 2>&1; then
    fail "folder import accepted a recognized unsafe profile"
fi
[[ -f "$TMP/import-target/amneziawg/AWG.conf" ]] ||
    fail "AmneziaWG folder auto-detection failed"
[[ -f "$TMP/import-target/wireguard/WG.conf" ]] ||
    fail "WireGuard folder auto-detection failed"
[[ -f "$TMP/import-target/openvpn/Auto OpenVPN.conf" ]] ||
    fail "OpenVPN .conf folder auto-detection failed"
[[ -f "$TMP/import-target/l2tp/Office.nmconnection" ]] ||
    fail "L2TP folder auto-detection failed"
[[ "$(stat -c %a "$TMP/import-target/amneziawg/AWG.conf")" == "600" ]] ||
    fail "import-dir did not close target permissions"
[[ ! -e "$TMP/import-target/openvpn/Unsafe.ovpn" ]] ||
    fail "unsafe folder profile was copied"

mkdir -p "$TMP/import-files-target"
printf '3\n' >"$VPNCTL_RUN_DIR/health.failures"
VPNCTL_CONFIG_DIR="$TMP/import-files-target" \
    "$CLI" import-files \
    "$TMP/import-source/mixed/AWG.conf" \
    "$TMP/import-source/mixed/WG.conf" >/dev/null
[[ -f "$TMP/import-files-target/amneziawg/AWG.conf" &&
   -f "$TMP/import-files-target/wireguard/WG.conf" ]] ||
    fail "multi-file import did not detect both VPN protocols"
[[ "$(stat -c %a "$TMP/import-files-target/amneziawg/AWG.conf")" == "600" ]] ||
    fail "multi-file import did not close target permissions"
[[ ! -e "$VPNCTL_RUN_DIR/health.failures" ]] ||
    fail "explicit profile mutation did not reset the health recovery counter"
if VPNCTL_CONFIG_DIR="$TMP/import-files-target" \
    "$CLI" import-files \
    "$TMP/import-source/mixed/AWG.conf" \
    "$TMP/import-source/mixed/WG.conf" >/dev/null 2>&1; then
    fail "multi-file import overwrote an existing profile without --force"
fi
ok "safe multi-file profile import"

"$CLI" init-config-dir "$TMP/new-config-tree" >/dev/null
for protocol_dir in amneziawg wireguard openvpn l2tp; do
    [[ -d "$TMP/new-config-tree/$protocol_dir" ]] ||
        fail "init-config-dir did not create $protocol_dir"
done
ok "folder structure and safe protocol auto-import"

cat >"$FAKE_IP_RULES" <<'EOF'
215:	from all lookup main suppress_prefixlength 0
216:	not from all fwmark 0xca6c lookup 51820
217:	from all lookup main suppress_prefixlength 0
218:	not from all fwmark 0xca6c lookup 51820
219:	not from all fwmark 0xca6d lookup 51821
220:	from all lookup 220
EOF
: >"$FAKE_IP_LOG"
: >"$FAKE_TRANSITION_LOG"
: >"$FAKE_TRANSITION_CURL_LOG"
FAKE_DNS_UNAVAILABLE=1 "$CLI" connect openvpn "Test Server" >/dev/null
grep -q '^PROTOCOL=openvpn$' "$TMP/state/active" || fail "protocol state missing"
grep -q '^PROFILE=Test Server.ovpn$' "$TMP/state/active" || fail "profile state missing"
grep -q '^DESIRED=up$' "$TMP/state/active" || fail "desired state missing"
if grep -q 'enable.*vpnctl-health.timer' "$TMP/systemctl.log"; then
    fail "ordinary connect enabled watchdog at boot"
fi
for priority in 215 216 217 218 219; do
    grep -q -- "-4 rule delete priority $priority" "$FAKE_IP_LOG" ||
        fail "stale policy priority $priority was not removed"
done
if grep -q -- '-4 rule delete priority 220' "$FAKE_IP_LOG"; then
    fail "unrelated IPsec priority 220 was removed"
fi
guard_install_line="$(grep -n -m1 '^guard install$' "$FAKE_TRANSITION_LOG" | cut -d: -f1)"
service_stop_line="$(grep -n -m1 '^systemctl stop vpnctl.service$' "$FAKE_TRANSITION_LOG" | cut -d: -f1)"
service_start_line="$(grep -n -m1 '^systemctl start vpnctl.service$' "$FAKE_TRANSITION_LOG" | cut -d: -f1)"
guard_remove_line="$(grep -n '^guard remove$' "$FAKE_TRANSITION_LOG" | tail -n1 | cut -d: -f1)"
[[ "$guard_install_line" =~ ^[0-9]+$ && "$service_stop_line" =~ ^[0-9]+$ &&
   "$service_start_line" =~ ^[0-9]+$ && "$guard_remove_line" =~ ^[0-9]+$ &&
   "$guard_install_line" -lt "$service_stop_line" &&
   "$service_start_line" -lt "$guard_remove_line" ]] ||
    fail "connect did not retain the leak guard across the tunnel transition"
grep -q 'ip daddr 192.0.2.10 udp dport 1194 accept' \
    "$FAKE_NFT_RULES_LOG" ||
    fail "transition guard does not allow the selected VPN endpoint"
if grep -q 'meta skuid 0 accept' "$FAKE_NFT_RULES_LOG"; then
    fail "transition guard permits unrestricted privileged egress"
fi
grep -q 'chain forward' "$FAKE_NFT_RULES_LOG" ||
    fail "transition guard does not cover forwarded traffic"
grep -q 'reject with icmpx type admin-prohibited' "$FAKE_NFT_RULES_LOG" ||
    fail "transition guard does not reject direct IPv4/IPv6 user traffic"
grep -Eq -- '--interface vpnovpn0 .*https://1\.1\.1\.1/cdn-cgi/trace$' \
    "$FAKE_TRANSITION_CURL_LOG" ||
    fail "transition readiness is not bound to the selected VPN interface"
awk '
    /^guard install$/ { guarded = 1; next }
    /^guard remove$/ { guarded = 0; next }
    guarded && /^curl transition probe$/ { fixed_probe = 1 }
    guarded && /^curl hostname probe$/ { hostname_probe = 1 }
    END { exit !(fixed_probe && !hostname_probe) }
' "$FAKE_TRANSITION_LOG" ||
    fail "guarded transition readiness still depends on local DNS"
if awk '
    /dport[[:space:]]+53/ &&
        $0 !~ /(^|[[:space:]])ip6?[[:space:]]+daddr/ { unsafe = 1 }
    END { exit !unsafe }
' "$FAKE_NFT_RULES_LOG"; then
    fail "transition guard permits DNS egress without a resolver address"
fi
rm -f "$FAKE_IP_RULES"
ok "connect persists profile and prevents direct traffic during the transition"

: >"$FAKE_TRANSITION_CURL_LOG"
FAKE_IPV6_ONLY=1 "$CLI" reconnect >/dev/null
grep -Eq -- '-6 .*--interface vpnovpn0 .*https://\[2606:4700:4700::1111\]/cdn-cgi/trace$' \
    "$FAKE_TRANSITION_CURL_LOG" ||
    fail "transition readiness did not verify an IPv6-only tunnel"
ok "transition readiness supports IPv6-only protected egress"

: >"$FAKE_TRANSITION_LOG"
if FAKE_TRANSITION_PROBE_DUP_IP=1 VPNCTL_TRANSITION_READY_TIMEOUT=3 \
    "$CLI" reconnect >/dev/null 2>&1; then
    fail "transition readiness accepted an ambiguous IP-literal probe response"
fi
grep -q '^systemctl stop vpnctl.service$' "$FAKE_TRANSITION_LOG" ||
    fail "failed transition readiness did not stop the unverified tunnel"
if grep -q '^guard remove$' "$FAKE_TRANSITION_LOG"; then
    fail "failed primary and rollback readiness removed the traffic guard"
fi
jq -e '
    .state == "recovery-required" and
    .reason == "managed-vpn-restore-failed"
' "$TMP/state/transition-recovery-required.json" >/dev/null ||
    fail "failed rollback did not persist transition recovery state"
"$CLI" reconnect >/dev/null
[[ ! -e "$TMP/state/transition-recovery-required.json" ]] ||
    fail "successful reconnect did not clear transition recovery state"
ok "transition readiness fails closed on ambiguous responses and recovers cleanly"

: >"$FAKE_SYSTEMCTL_LOG"
: >"$FAKE_TRANSITION_LOG"
if FAKE_NFT_INSTALL_FAIL=1 "$CLI" reconnect >/dev/null 2>&1; then
    fail "reconnect continued after the leak guard failed"
fi
if grep -Eq '^(stop|start|restart) vpnctl.service$' "$FAKE_SYSTEMCTL_LOG"; then
    fail "failed leak guard allowed a tunnel lifecycle mutation"
fi
ok "tunnel mutation fails before stop when the leak guard is unavailable"

cat >"$FAKE_IP_RULES" <<'EOF'
214:	from all lookup main suppress_prefixlength 0
215:	not from all fwmark 0xca6c lookup 51820
216:	from all lookup main suppress_prefixlength 0
217:	not from all fwmark 0xca6c lookup 51820
218:	from all lookup main suppress_prefixlength 0
219:	not from all fwmark 0xca6c lookup 51820
220:	from all lookup 220
EOF
: >"$FAKE_IP_LOG"
"$CLI" _dedupe-quick-policy >/dev/null
for priority in 216 217 218 219; do
    grep -q -- "-4 rule delete priority $priority" "$FAKE_IP_LOG" ||
        fail "duplicate policy priority $priority was not removed"
done
for priority in 214 215 220; do
    if grep -q -- "-4 rule delete priority $priority" "$FAKE_IP_LOG"; then
        fail "active or unrelated policy priority $priority was removed"
    fi
done
rm -f "$FAKE_IP_RULES"
ok "duplicate quick policy rules are removed without touching the active pair"

menu_output="$(
    printf '2\n3\n1\n0\n' |
        VPNCTL_FORCE_INTERACTIVE=1 "$CLI" 2>&1
)"
grep -q 'Выберите протокол' <<<"$menu_output" || fail "protocol menu was not shown"
grep -q 'Выбран OpenVPN' <<<"$menu_output" || fail "TUI did not connect selected protocol"
grep -q 'PROTOCOL=openvpn' "$TMP/state/active" || fail "TUI selection corrupted protocol"
ok "interactive TUI protocol/profile selection"

menu_live_output="$TMP/menu-live.out"
(
    {
        printf '2\n3\n1\n'
        sleep 5
        printf '0\n'
    } | VPNCTL_FORCE_INTERACTIVE=1 "$CLI" >"$menu_live_output" 2>&1
) &
menu_live_pid=$!
for _ in {1..30}; do
    grep -q 'Подключение запущено' "$menu_live_output" 2>/dev/null && break
    sleep 0.1
done
grep -q 'Подключение запущено' "$menu_live_output" ||
    fail "live TUI did not complete the connect action"
exec 6>"$TMP/run/.mutation.lock"
flock -n 6 || fail "idle TUI retained the mutation lock after connect"
flock -u 6
wait "$menu_live_pid"
ok "idle TUI releases the watchdog mutation lock"

printf '15\n3\n%s\n0\n' "$TMP/tui-config-tree" |
    VPNCTL_FORCE_INTERACTIVE=1 "$CLI" >/dev/null 2>&1
[[ -d "$TMP/tui-config-tree/amneziawg" && -d "$TMP/tui-config-tree/openvpn" ]] ||
    fail "TUI folder-structure action did not work"
ok "interactive TUI config-folder action"

language_menu_output="$(
    printf '16\n2\n0\n' |
        VPNCTL_FORCE_INTERACTIVE=1 "$CLI" 2>&1
)"
grep -q 'Language saved: English (en)' <<<"$language_menu_output" ||
    fail "TUI language selection did not switch immediately"
grep -q 'Quick-connect the default config' <<<"$language_menu_output" ||
    fail "TUI did not redraw in the selected language"
"$CLI" language ru >/dev/null
ok "interactive TUI language selection"

"$CLI" _service-run
grep -q -- '--dev vpnovpn0' "$TMP/openvpn.log" || fail "fixed OpenVPN device missing"
grep -q -- '--data-ciphers-fallback AES-256-CBC' "$TMP/openvpn.log" ||
    fail "legacy cipher fallback missing"
grep -q -- '_openvpn-dns-up' "$TMP/openvpn.log" || fail "OpenVPN DNS hook missing"
if grep -q -- '--verb 6' "$TMP/openvpn.log"; then
    fail "normal OpenVPN mode unexpectedly enabled verbose logging"
fi
ok "OpenVPN service arguments"

sed -i 's/^MODE=normal$/MODE=test/' "$TMP/state/active"
printf 'TEST_TOKEN=test-token\nTEST_DEADLINE=9999999999\n' >>"$TMP/state/active"
"$CLI" _service-run
grep -q -- '--verb 6' "$TMP/openvpn.log" || fail "test mode did not enable OpenVPN verb 6"
export FAKE_OPENVPN_TOO_MANY=1
openvpn_limit_rc=0
"$CLI" _service-run >/dev/null 2>&1 || openvpn_limit_rc=$?
unset FAKE_OPENVPN_TOO_MANY
[[ "$openvpn_limit_rc" -eq 75 ]] ||
    fail "OpenVPN server-pushed connection limit was not converted to a retryable failure"
grep -qx 'openvpn-too-many-connections' "$TMP/run/test.failure" ||
    fail "OpenVPN connection-limit reason was not recorded"
rm -f "$TMP/run/test.failure"
export FAKE_OPENVPN_AUTH_FAILED=1
openvpn_auth_rc=0
"$CLI" _service-run >/dev/null 2>&1 || openvpn_auth_rc=$?
unset FAKE_OPENVPN_AUTH_FAILED
[[ "$openvpn_auth_rc" -eq 77 ]] ||
    fail "permanent OpenVPN authentication rejection did not return exit 77"
grep -qx 'openvpn-auth-failed' "$TMP/run/test.failure" ||
    fail "OpenVPN authentication rejection reason was not recorded"
rm -f "$TMP/run/test.failure"
sed -i '/^TEST_/d; s/^MODE=test$/MODE=normal/' "$TMP/state/active"
ok "test mode logging and permanent/retryable OpenVPN exit classification"

foreign_option_1='dhcp-option DNS 9.9.9.9' \
    "$CLI" _openvpn-dns-up vpnovpn0
"$CLI" _openvpn-dns-down vpnovpn0
grep -q '^dns vpnovpn0 9.9.9.9$' "$TMP/resolvectl.log" || fail "DNS server was not applied"
grep -q '^domain vpnovpn0 ~\.$' "$TMP/resolvectl.log" || fail "DNS route was not applied"
grep -q '^revert vpnovpn0$' "$TMP/resolvectl.log" || fail "DNS was not reverted"
dns_log_lines="$(wc -l <"$TMP/resolvectl.log")"
"$CLI" _openvpn-dns-up vpnovpn1 >/dev/null 2>&1
[[ "$(wc -l <"$TMP/resolvectl.log")" -eq "$dns_log_lines" ]] ||
    fail "OpenVPN silently substituted a hard-coded public DNS server"
VPNCTL_OPENVPN_FALLBACK_DNS='149.112.112.112' \
    "$CLI" _openvpn-dns-up vpnovpn2
grep -q '^dns vpnovpn2 149.112.112.112$' "$TMP/resolvectl.log" ||
    fail "explicit OpenVPN fallback DNS was not applied"
ok "OpenVPN DNS lifecycle avoids a hard-coded resolver"

status_output="$("$CLI" status)"
grep -q '^VPN:       active$' <<<"$status_output" || fail "status is not active"
grep -q '^Autostart: enabled$' <<<"$status_output" || fail "autostart status missing"
grep -q '^Public IP: 203.0.113.7$' <<<"$status_output" || fail "public IP missing"
ok "status"

status_json="$("$CLI" status --json)"
python3 -m json.tool <<<"$status_json" >/dev/null ||
    fail "structured dashboard status is not valid JSON"
grep -q '"profile":"Test Server"' <<<"$status_json" ||
    fail "structured dashboard status is missing the selected profile"
grep -Eq '"profile_id":"profile-[a-f0-9]{32}"' <<<"$status_json" ||
    fail "structured dashboard status is missing the exact profile identity"
grep -q '"profile_file_name":"Test Server.ovpn"' <<<"$status_json" ||
    fail "structured dashboard status is missing the exact profile filename"
grep -q '"protocol":"openvpn","protocol_name":"OpenVPN"' <<<"$status_json" ||
    fail "structured dashboard status has inconsistent active protocol metadata"
grep -q '"healthy":true' <<<"$status_json" ||
    fail "structured dashboard status did not report a healthy tunnel"
grep -q '"openvpn":1' <<<"$status_json" ||
    fail "structured dashboard status is missing profile counts"
grep -q '192\.0\.2\.10' <<<"$status_json" &&
    fail "structured dashboard status leaked a VPN endpoint"
[[ "$(stat -c %a "$VPNCTL_DASHBOARD_DIR")" == "750" ]] ||
    fail "dashboard cache directory is not restricted"
[[ "$(stat -c %a "$VPNCTL_DASHBOARD_DIR/status.json")" == "640" ]] ||
    fail "dashboard status cache is not group-restricted"
"$CLI" _refresh-dashboard-cache
ok "safe structured dashboard status"

verify_json="$("$CLI" verify --json)"
jq -e '
    .schema_version == 1
    and .verdict == "verified"
    and .tunnel.active == true
    and .tunnel.interface == "vpnovpn0"
    and .ipv4.interface_ip == "203.0.113.7"
    and .ipv4.default_ip == "203.0.113.7"
    and .ipv4.same_egress == true
    and .ipv6.potential_leak == false
    and .geo.expected_country_code == "BE"
    and .geo.observed_country_code == "BE"
    and .geo.country_match == "match"
    and .geo.providers_agree == true
    and (.geo.providers | length) == 2
    and .dns.state == "vpn-full-tunnel"
    and .speed.requested == false
    and .findings == []
' <<<"$verify_json" >/dev/null ||
    fail "VPN egress verification did not prove the bounded happy path"
if grep -Eq '192\.0\.2\.10|PrivateKey|PublicKey|Test Server\.ovpn' \
    <<<"$verify_json"; then
    fail "VPN egress verification leaked engine-only profile data"
fi

verify_speed_json="$("$CLI" verify --speed --json)"
jq -e '
    .verdict == "verified"
    and .speed.requested == true
    and .speed.measured == true
    and .speed.mbps == 100
    and .speed.connect_ms == 40
' <<<"$verify_speed_json" >/dev/null ||
    fail "explicit bounded speed sample is invalid"

ru_numeric_locale="$(
    locale -a | LC_ALL=C awk 'tolower($0) ~ /^ru_ru\.utf-?8$/ { print; exit }'
)"
if [[ -n "$ru_numeric_locale" ]]; then
    [[ "$(LC_ALL="$ru_numeric_locale" locale decimal_point)" == ',' ]] ||
        fail "$ru_numeric_locale does not provide the required comma decimal separator"
    verify_speed_non_c_json="$(
        LC_ALL="$ru_numeric_locale" "$CLI" verify --speed --json
    )"
    jq -e '
        .speed.measured == true
        and .speed.mbps == 100
        and .speed.connect_ms == 40
    ' <<<"$verify_speed_non_c_json" >/dev/null ||
        fail "speed JSON depends on the caller numeric locale"
else
    printf 'notice: ru_RU UTF-8 locale unavailable; comma-decimal coverage skipped\n'
fi
verify_human_output="$("$CLI" verify)"
grep -Fxq 'Network egress verified' <<<"$verify_human_output" ||
    fail "generic verification still overclaims VPN or service eligibility"
ok "network egress verification preserves schema v1 and locale-independent JSON"

: >"$FAKE_SERVICE_CURL_LOG"
service_notebook_unsupported="$(
    ALL_PROXY=http://proxy.invalid:8080 \
    HTTP_PROXY=http://proxy.invalid:8080 \
    HTTPS_PROXY=http://proxy.invalid:8080 \
    NO_PROXY=localhost \
    FAKE_SERVICE_STATUS=302 \
    FAKE_SERVICE_LOCATION='https://notebooklm.google/?location=unsupported' \
        "$CLI" verify-service notebooklm --timeout 3 --json
)"
jq -e '
    .schema_version == 1
    and .scope == "unauthenticated-egress"
    and (.results | length) == 1
    and .results[0] == {
        service_id: "notebooklm",
        probe_version: 1,
        reachability: "reachable",
        egress_eligibility: "ineligible",
        reason_code: "service.notebooklm.unsupported-location",
        http_status: 302
    }
' <<<"$service_notebook_unsupported" >/dev/null ||
    fail "NotebookLM unsupported-location redirect was misclassified"
service_notebook_eligible="$(
    FAKE_SERVICE_STATUS=302 \
    FAKE_SERVICE_LOCATION='https://notebook.google.com/' \
        "$CLI" verify-service notebooklm --timeout 3 --json
)"
jq -e '
    .results[0].reachability == "reachable"
    and .results[0].egress_eligibility == "eligible"
    and .results[0].reason_code == "service.notebooklm.home-reached"
' <<<"$service_notebook_eligible" >/dev/null ||
    fail "NotebookLM exact eligible redirect was misclassified"
service_notebook_unknown="$(
    FAKE_SERVICE_STATUS=302 \
    FAKE_SERVICE_LOCATION='https://accounts.google.com/unsafe-variable-target' \
        "$CLI" verify-service notebooklm --timeout 3 --json
)"
jq -e '
    .results[0].reachability == "reachable"
    and .results[0].egress_eligibility == "indeterminate"
    and .results[0].reason_code == "service.notebooklm.unrecognized-response"
' <<<"$service_notebook_unknown" >/dev/null ||
    fail "NotebookLM unrecognized redirect was trusted"

for service_status_case in \
    '401|eligible|service.openai.auth-boundary-reached' \
    '405|eligible|service.openai.auth-boundary-reached' \
    '403|ineligible|service.openai.edge-denied' \
    '429|indeterminate|service.openai.rate-limited' \
    '500|indeterminate|service.openai.service-unavailable' \
    '418|indeterminate|service.openai.unrecognized-response'; do
    service_status="${service_status_case%%|*}"
    service_expected="${service_status_case#*|}"
    service_eligibility="${service_expected%%|*}"
    service_reason="${service_expected#*|}"
    service_allow=""
    [[ "$service_status" != 405 ]] || service_allow=POST
    service_openai_result="$(
        FAKE_SERVICE_STATUS="$service_status" \
        FAKE_SERVICE_LOCATION='' FAKE_SERVICE_ALLOW="$service_allow" \
            "$CLI" verify-service openai --timeout 3 --json
    )"
    jq -e \
        --argjson status "$service_status" \
        --arg eligibility "$service_eligibility" \
        --arg reason "$service_reason" '
        .results[0].service_id == "openai"
        and .results[0].reachability == "reachable"
        and .results[0].egress_eligibility == $eligibility
        and .results[0].reason_code == $reason
        and .results[0].http_status == $status
    ' <<<"$service_openai_result" >/dev/null ||
        fail "OpenAI status $service_status was misclassified"
done
service_openai_405_without_allow="$(
    FAKE_SERVICE_STATUS=405 FAKE_SERVICE_ALLOW='' \
        "$CLI" verify-service openai --timeout 3 --json
)"
jq -e '
    .results[0].egress_eligibility == "indeterminate"
    and .results[0].reason_code == "service.openai.unrecognized-response"
' <<<"$service_openai_405_without_allow" >/dev/null ||
    fail "OpenAI 405 without exact Allow: POST was trusted"
service_all_result="$(
    FAKE_SERVICE_STATUS=401 FAKE_SERVICE_LOCATION='' FAKE_SERVICE_ALLOW='' \
        "$CLI" verify-service all --timeout 3 --json
)"
jq -e '
    (.results | length) == 2
    and (.results | map(.service_id)) == ["notebooklm", "openai"]
    and .results[0].egress_eligibility == "indeterminate"
    and .results[1].egress_eligibility == "eligible"
' <<<"$service_all_result" >/dev/null ||
    fail "verify-service all did not run both allowlisted probes"
service_network_error="$(
    FAKE_CURL_FAIL=1 "$CLI" verify-service openai --timeout 3 --json
)"
jq -e '
    .results[0].reachability == "unreachable"
    and .results[0].egress_eligibility == "indeterminate"
    and .results[0].reason_code == "service.network-unreachable"
    and .results[0].http_status == null
' <<<"$service_network_error" >/dev/null ||
    fail "service network failure was not sanitized"
rm -f -- "$FAKE_SERVICE_PID_FILE"
service_oversized="$(
    FAKE_SERVICE_OVERSIZE=1 FAKE_SERVICE_STATUS=302 \
        "$CLI" verify-service notebooklm --timeout 3 --json
)"
jq -e '
    .results[0].reason_code == "service.response-too-large"
    and .results[0].http_status == null
' <<<"$service_oversized" >/dev/null ||
    fail "service probe accepted oversized response headers"
[[ -s "$FAKE_SERVICE_PID_FILE" ]] ||
    fail "oversized-header test did not observe the curl worker"
oversized_probe_pid="$(<"$FAKE_SERVICE_PID_FILE")"
sleep 0.1
if [[ -r "/proc/$oversized_probe_pid/cmdline" ]] &&
   tr '\0' ' ' <"/proc/$oversized_probe_pid/cmdline" |
       grep -Fq -- "$TMP/fakebin/curl"; then
    fail "oversized service headers left a curl worker running"
fi
service_duplicate_location="$(
    FAKE_SERVICE_STATUS=302 FAKE_SERVICE_DUP_LOCATION=1 \
    FAKE_SERVICE_LOCATION='https://notebook.google.com/' \
        "$CLI" verify-service notebooklm --timeout 3 --json
)"
jq -e '
    .results[0].reason_code == "service.response-invalid"
    and .results[0].egress_eligibility == "indeterminate"
' <<<"$service_duplicate_location" >/dev/null ||
    fail "service probe accepted duplicate Location headers"

grep -Fq -- '--head --interface vpnovpn0' "$FAKE_SERVICE_CURL_LOG" ||
    fail "service probe is not a HEAD request bound to the selected VPN interface"
grep -Fq -- '--max-redirs 0' "$FAKE_SERVICE_CURL_LOG" ||
    fail "service probe does not explicitly disable redirect traversal"
if grep -Eq '(^| )(-L|--location|--cookie|--user|--netrc|--cert)( |$)' \
    "$FAKE_SERVICE_CURL_LOG"; then
    fail "service probe enabled redirects, credentials, cookies or client certificates"
fi
if grep -Eq 'https://|location=|[Cc]ookie|[Aa]uthorization|account' \
    <<<"$service_notebook_unsupported"; then
    fail "service result leaked a URL, header or account field"
fi
if grep -R -E 'location=unsupported|backend-api/codex/responses' \
    "$VPNCTL_STATE_DIR" "$VPNCTL_RUN_DIR" >/dev/null 2>&1; then
    fail "service probe persisted a raw URL or Location header"
fi
: >"$FAKE_SERVICE_CURL_LOG"
if "$CLI" verify-service 'https://example.invalid/' --json >/dev/null 2>&1; then
    fail "verify-service accepted an arbitrary URL"
fi
if "$CLI" verify-service openai --timeout 2 --json >/dev/null 2>&1 ||
   "$CLI" verify-service openai --timeout 16 --json >/dev/null 2>&1 ||
   "$CLI" verify-service all notebooklm --json >/dev/null 2>&1; then
    fail "verify-service accepted an invalid timeout or duplicate scope"
fi
[[ ! -s "$FAKE_SERVICE_CURL_LOG" ]] ||
    fail "invalid verify-service input reached curl"
ok "allowlisted service egress classifiers are credential-free, bounded and sanitized"

cp "$TMP/config/openvpn/Test Server.ovpn" "$TMP/profile-with-country.ovpn"
sed -i '/^[#;][[:space:]]*mazzy-country-code[[:space:]]*:/d' \
    "$TMP/config/openvpn/Test Server.ovpn"
verify_missing_country="$("$CLI" _verify-egress-json 3 false)"
mv -f -- "$TMP/profile-with-country.ovpn" "$TMP/config/openvpn/Test Server.ovpn"
jq -e '
    .verdict == "warning"
    and .geo.expected_country_code == null
    and .geo.country_match == "unknown"
    and (
        .findings
        | index("verify.geo.expected-country-unavailable")
    ) != null
' <<<"$verify_missing_country" >/dev/null ||
    fail "VPN verification claimed verified location without explicit country metadata"

verify_route_warning="$(
    FAKE_DEFAULT_IPV4=198.51.100.9 "$CLI" _verify-egress-json 3 false
)"
jq -e '
    .verdict == "warning"
    and .ipv4.same_egress == false
    and (.findings | index("verify.ipv4.route-mismatch")) != null
' <<<"$verify_route_warning" >/dev/null ||
    fail "VPN verification missed a default-route egress mismatch"

verify_ipv6_warning="$(
    FAKE_DEFAULT_IPV6=2001:db8::10 "$CLI" _verify-egress-json 3 false
)"
jq -e '
    .verdict == "warning"
    and .ipv6.potential_leak == true
    and (.findings | index("verify.ipv6.potential-leak")) != null
' <<<"$verify_ipv6_warning" >/dev/null ||
    fail "VPN verification missed a potential IPv6 leak"

verify_geo_warning="$(
    FAKE_GEO_MISMATCH=1 "$CLI" _verify-egress-json 3 false
)"
jq -e '
    .verdict == "warning"
    and .geo.providers_agree == false
    and (.findings | index("verify.geo.providers-disagree")) != null
' <<<"$verify_geo_warning" >/dev/null ||
    fail "VPN verification trusted disagreeing geolocation providers"

verify_geo_ip_warning="$(
    FAKE_GEO_IPV4=198.51.100.22 "$CLI" _verify-egress-json 3 false
)"
jq -e '
    .verdict == "warning"
    and (.geo.providers | length) == 0
    and (.findings | index("verify.geo.ip-mismatch")) != null
    and (.findings | index("verify.geo.unavailable")) != null
' <<<"$verify_geo_ip_warning" >/dev/null ||
    fail "VPN verification trusted location data for a different egress IP"

verify_dns_warning="$(
    FAKE_DNS_NO_DEFAULT_ROUTE=1 "$CLI" _verify-egress-json 3 false
)"
jq -e '
    .verdict == "warning"
    and .dns.state == "vpn-interface"
    and (
        .findings
        | index("verify.dns.not-confirmed-full-tunnel")
    ) != null
' <<<"$verify_dns_warning" >/dev/null ||
    fail "VPN verification treated partial DNS routing as full-tunnel DNS"

verify_inactive="$(
    FAKE_SYSTEMCTL_INACTIVE=1 "$CLI" _verify-egress-json 3 false
)"
jq -e '
    .verdict == "failed"
    and .tunnel.active == false
    and (.findings | index("verify.tunnel.inactive")) != null
' <<<"$verify_inactive" >/dev/null ||
    fail "VPN verification accepted an inactive tunnel"
ok "VPN egress verification rejects route, IPv6, geo, DNS and tunnel failures"

profiles_json="$("$CLI" profiles --json)"
python3 -m json.tool <<<"$profiles_json" >/dev/null ||
    fail "structured Desktop profile cache is not valid JSON"
grep -q '"file_name":"Test Server.ovpn","name":"Test Server"' <<<"$profiles_json" ||
    fail "Desktop profile cache is missing a sanitized profile"
grep -q '"selected":true' <<<"$profiles_json" ||
    fail "Desktop profile cache did not mark the selected profile"
grep -q '192\.0\.2\.10' <<<"$profiles_json" &&
    fail "Desktop profile cache leaked a VPN endpoint"
grep -Eq 'PrivateKey|PublicKey|private.key|public.key' <<<"$profiles_json" &&
    fail "Desktop profile cache leaked key material"
profile_id="$(
    jq -er '.profiles[] | select(.name == "Test Server") | .profile_id' \
        <<<"$profiles_json"
)"
[[ "$profile_id" =~ ^profile-[a-f0-9]{32}$ ]] ||
    fail "Desktop profile cache does not expose an opaque stable profile ID"
[[ "$(stat -c %a "$VPNCTL_DASHBOARD_DIR/profiles.json")" == "640" ]] ||
    fail "Desktop profile cache is not group-restricted"
ok "sanitized Desktop profile library cache"

api_status_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-status-0001",
        operation: "status.get",
        payload: {}
    }'
)"
api_status_response="$(printf '%s\n' "$api_status_request" | "$CLI" _api-dispatch)"
jq -e --arg profile_id "$profile_id" '
    .api_version == "1.0"
    and .request_id == "request-status-0001"
    and .status == "ok"
    and .result.product == "Mazzy VPN"
    and .result.connection.profile_id == $profile_id
    and .result.connection.state == "connected"
    and .result.connection.interface == "vpnovpn0"
    and .result.connection.public_ip == "203.0.113.7"
    and .result.desired == "up"
    and .result.mode == "normal"
    and .result.autostart == true
    and .result.health_monitor == true
    and .result.health_failures == 0
    and .result.fallback.active == false
' <<<"$api_status_response" >/dev/null ||
    fail "local API status.get response is invalid"
if grep -Eq '192\.0\.2\.10|endpoint|file_name|Test Server\.ovpn' \
    <<<"$api_status_response"; then
    fail "local API status.get detail leaked engine-only profile data"
fi

api_profiles_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-profiles-0001",
        operation: "profiles.list",
        payload: {}
    }'
)"
api_profiles_response="$(printf '%s\n' "$api_profiles_request" | "$CLI" _api-dispatch)"
jq -e --arg profile_id "$profile_id" '
    .status == "ok"
    and any(
        .result.profiles[];
        .profile_id == $profile_id
        and .display_name == "Test Server"
        and .protocol == "openvpn"
        and .selected == true
    )
' <<<"$api_profiles_response" >/dev/null ||
    fail "local API profiles.list response is invalid"

api_protocols_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-protocols-0001",
        operation: "protocols.list",
        payload: {}
    }'
)"
api_protocols_response="$(printf '%s\n' "$api_protocols_request" | "$CLI" _api-dispatch)"
jq -e '
    .status == "ok"
    and (.result.protocols | length) == 13
    and any(
        .result.protocols[];
        .id == "vless"
        and .support.detection == "implemented"
        and .support.linux == "planned"
    )
    and (.result.orchestration.agent_rules | index(
        "credentials-never-enter-prompts-events-or-audit"
    )) != null
' <<<"$api_protocols_response" >/dev/null ||
    fail "local API protocols.list response is invalid"
if jq -e '
    .. | objects | keys[] |
    select(test("^(password|credentials?|secret|endpoint|file_path|private_key)$"))
' <<<"$api_protocols_response" >/dev/null; then
    fail "local API protocols.list leaked a frontend-forbidden field"
fi

chmod 700 "$VPNCTL_STATE_DIR"
[[ ! -e "$VPNCTL_API_ACTION_DIR" ]] ||
    fail "planner fresh-state test unexpectedly has an action directory"
unknown_profile_id="profile-ffffffffffffffffffffffffffffffff"
planner_payload="$(
    jq -cn \
        --arg profile_id "$profile_id" \
        --arg unknown_profile_id "$unknown_profile_id" '{
        workload: "llm-streaming",
        candidates: [
            {
                profile_id: $profile_id,
                evidence: {
                    recent_outcome: "success",
                    consecutive_failures: 0,
                    reachability: "reachable",
                    latency_ms: 50,
                    loss_percent: 0,
                    evidence_age_seconds: 30
                }
            },
            {
                profile_id: $unknown_profile_id,
                evidence: {
                    recent_outcome: "unknown",
                    consecutive_failures: 0,
                    reachability: "unknown",
                    latency_ms: null,
                    loss_percent: 0,
                    evidence_age_seconds: 30
                }
            }
        ]
    }'
)"
planner_request="$(
    jq -cn --argjson payload "$planner_payload" '{
        api_version: "1.0",
        request_id: "request-planner-0001",
        operation: "planner.evaluate",
        deadline_ms: 5000,
        payload: $payload
    }'
)"
planner_response="$(printf '%s\n' "$planner_request" | "$CLI" _api-dispatch)"
jq -e \
    --arg profile_id "$profile_id" \
    --arg unknown_profile_id "$unknown_profile_id" '
    .api_version == "1.0"
    and .request_id == "request-planner-0001"
    and .status == "ok"
    and .result.schema_version == 1
    and .result.policy_version == 1
    and .result.dry_run == true
    and .result.workload == "llm-streaming"
    and .result.ordered_profile_ids == [$profile_id]
    and (.result.candidates | length) == 2
    and .result.candidates[0].profile_id == $profile_id
    and .result.candidates[0].protocol == "openvpn"
    and .result.candidates[0].eligible == true
    and .result.candidates[0].rank == 1
    and .result.candidates[0].score == 92
    and (.result.candidates[0].hard_gates | length) == 5
    and all(.result.candidates[0].hard_gates[]; .passed == true)
    and (.result.candidates[0].factors | map(.points) | add) == 92
    and .result.candidates[1].profile_id == $unknown_profile_id
    and .result.candidates[1].eligible == false
    and .result.candidates[1].rank == null
    and .result.candidates[1].score == null
    and (.result.candidates[1] | has("protocol") | not)
' <<<"$planner_response" >/dev/null ||
    fail "local API planner.evaluate did not apply hard gates and scoring"
if jq -e '
    .. | objects | keys[] |
    select(test("^(display_name|password|credentials?|secret|endpoint|file_name|file_path|private_key)$"))
' <<<"$planner_response" >/dev/null; then
    fail "local API planner.evaluate leaked frontend-forbidden profile data"
fi

planner_repeat_request="$(
    jq -c '.request_id = "request-planner-0002"' <<<"$planner_request"
)"
planner_repeat_response="$(
    printf '%s\n' "$planner_repeat_request" | "$CLI" _api-dispatch
)"
jq -S '.result | del(.evaluated_at)' <<<"$planner_response" \
    >"$TMP/planner-first.json"
jq -S '.result | del(.evaluated_at)' <<<"$planner_repeat_response" \
    >"$TMP/planner-second.json"
cmp -s "$TMP/planner-first.json" "$TMP/planner-second.json" ||
    fail "local API planner.evaluate is not deterministic for equal evidence"

planner_stale_request="$(
    jq -c '
        .request_id = "request-planner-stale-0001"
        | .payload.candidates = [.payload.candidates[0]]
        | .payload.candidates[0].evidence.evidence_age_seconds = 901
    ' <<<"$planner_request"
)"
planner_stale_response="$(
    printf '%s\n' "$planner_stale_request" | "$CLI" _api-dispatch
)"
jq -e '
    .status == "ok"
    and .result.candidates[0].score == 27
    and (.result.candidates[0].reason_codes | index(
        "planner.factor.recent-stale"
    )) != null
    and (.result.candidates[0].reason_codes | index(
        "planner.factor.reachability-stale"
    )) != null
    and (.result.candidates[0].reason_codes | index(
        "planner.factor.latency-loss-stale"
    )) != null
' <<<"$planner_stale_response" >/dev/null ||
    fail "local API planner.evaluate trusted stale network evidence"

: >"$FAKE_TIMEOUT_LOG"
planner_deadline_request="$(
    jq -c '
        .request_id = "request-planner-deadline-0001"
        | .deadline_ms = 500
        | .payload.candidates = [.payload.candidates[0]]
    ' <<<"$planner_request"
)"
planner_deadline_started=$SECONDS
planner_deadline_response="$(
    printf '%s\n' "$planner_deadline_request" |
        FAKE_OPENVPN_DELAY_SECONDS=5 "$CLI" _api-dispatch
)"
planner_deadline_elapsed=$((SECONDS - planner_deadline_started))
jq -e '
    .status == "error"
    and .error.code == "deadline-exceeded"
    and .error.message_key == "api.planner.deadline-exceeded"
' <<<"$planner_deadline_response" >/dev/null ||
    fail "local API planner did not preserve its candidate validation deadline"
((planner_deadline_elapsed < 3)) ||
    fail "OpenVPN planner validation exceeded its sub-second deadline"
if grep -Eq -- '(^| )(10|10\.000s) openvpn --config' "$FAKE_TIMEOUT_LOG"; then
    fail "OpenVPN planner validation used the unbounded default parser timeout"
fi

chmod 644 "$TMP/config/openvpn/Test Server.ovpn"
planner_unsafe_request="$(
    jq -c '
        .request_id = "request-planner-unsafe-0001"
        | .payload.candidates = [.payload.candidates[0]]
    ' <<<"$planner_request"
)"
planner_unsafe_response="$(
    printf '%s\n' "$planner_unsafe_request" | "$CLI" _api-dispatch
)"
chmod 600 "$TMP/config/openvpn/Test Server.ovpn"
jq -e '
    .status == "ok"
    and .result.ordered_profile_ids == []
    and .result.candidates[0].eligible == false
    and .result.candidates[0].score == null
    and any(
        .result.candidates[0].hard_gates[];
        .id == "secrets-readable-only-by-backend" and .passed == false
    )
' <<<"$planner_unsafe_response" >/dev/null ||
    fail "local API planner.evaluate ignored unsafe profile storage"

planner_duplicate_candidate_request="$(
    jq -c '
        .request_id = "request-planner-duplicate-0001"
        | .payload.candidates = [
            .payload.candidates[0], .payload.candidates[0]
        ]
    ' <<<"$planner_request"
)"
planner_duplicate_candidate_response="$(
    printf '%s\n' "$planner_duplicate_candidate_request" |
        "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.code == "invalid-request"
    and .error.message_key == "api.planner.payload-invalid"
' <<<"$planner_duplicate_candidate_response" >/dev/null ||
    fail "local API planner.evaluate accepted duplicate candidate IDs"

planner_untrusted_fit_request="$(
    jq -c '
        .request_id = "request-planner-untrusted-fit-0001"
        | .payload.candidates = [.payload.candidates[0]]
        | .payload.candidates[0].evidence.censorship_fit = "high"
    ' <<<"$planner_request"
)"
planner_untrusted_fit_response="$(
    printf '%s\n' "$planner_untrusted_fit_request" | "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.code == "invalid-request"
    and .error.message_key == "api.planner.payload-invalid"
' <<<"$planner_untrusted_fit_response" >/dev/null ||
    fail "local API planner.evaluate trusted caller-supplied policy fit"

planner_nested_duplicate_response="$(
    printf '%s\n' \
        "{\"api_version\":\"1.0\",\"request_id\":\"request-planner-duplicate-0002\",\"operation\":\"planner.evaluate\",\"deadline_ms\":5000,\"payload\":{\"workload\":\"general\",\"candidates\":[{\"profile_id\":\"$profile_id\",\"evidence\":{},\"evidence\":{\"recent_outcome\":\"success\"}}]}}" |
        "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.code == "invalid-request"
    and .error.message_key == "api.request.duplicate-key"
' <<<"$planner_nested_duplicate_response" >/dev/null ||
    fail "local API accepted a nested duplicate planner key"

planner_cli_response="$(
    printf '%s\n' "$planner_payload" |
        VPNCTL_API_CLIENT_FORCE=1 \
        "$CLI" planner evaluate --stdin --json
)"
jq -e \
    --arg profile_id "$profile_id" '
    .dry_run == true
    and .ordered_profile_ids == [$profile_id]
    and .candidates[0].score == 92
    and (.candidates[0].reason_codes | index(
        "planner.factor.censorship-catalog-medium"
    )) != null
    and (.candidates[0].reason_codes | index(
        "planner.factor.workload-derived-high"
    )) != null
' <<<"$planner_cli_response" >/dev/null ||
    fail "planner CLI did not use the typed local API query"

planner_max_payload="$(
    jq -cn '
        def evidence: {
            recent_outcome: "unknown",
            consecutive_failures: 0,
            reachability: "unknown",
            latency_ms: null,
            loss_percent: 0,
            evidence_age_seconds: 30
        };
        {
            workload: "general",
            candidates: [
                range(0; 128) as $index
                | {
                    profile_id: (
                        "profile-" + (
                            "00000000000000000000000000000000"
                            + ($index | tostring)
                        )[-32:]
                    ),
                    evidence: evidence
                }
            ]
        }
    '
)"
[[ "$(printf '%s' "$planner_max_payload" | wc -c)" -le 65536 ]] ||
    fail "maximum planner fixture exceeds the documented request cap"
planner_max_response="$(
    printf '%s\n' "$planner_max_payload" |
        VPNCTL_API_CLIENT_FORCE=1 \
        "$CLI" planner evaluate --stdin --json
)"
[[ "$(printf '%s' "$planner_max_response" | wc -c)" -le 1048576 ]] ||
    fail "maximum planner response exceeds its bounded CLI cap"
jq -e '
    (.candidates | length) == 128
    and .ordered_profile_ids == []
    and all(
        .candidates[];
        .eligible == false and .rank == null and .score == null
    )
' <<<"$planner_max_response" >/dev/null ||
    fail "planner CLI did not handle the documented 128-candidate bound"
ok "deterministic agent-safe protocol planner"

api_probe_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-probe-0001",
        operation: "tests.probe",
        deadline_ms: 5000,
        payload: {
            protocol: "openvpn",
            timeout_seconds: 1,
            concurrency: 2
        }
    }'
)"
api_probe_response="$(printf '%s\n' "$api_probe_request" | "$CLI" _api-dispatch)"
jq -e --arg profile_id "$profile_id" '
    .api_version == "1.0"
    and .request_id == "request-probe-0001"
    and .status == "ok"
    and .result.summary.total == 1
    and .result.summary.reachable == 1
    and .result.results[0].profile_id == $profile_id
    and .result.results[0].display_name == "Test Server"
    and .result.results[0].latency_ms == 12
    and .result.results[0].active == true
' <<<"$api_probe_response" >/dev/null ||
    fail "local API tests.probe response is invalid"
if grep -Eq '192\.0\.2\.10|file_name|PrivateKey|PublicKey' \
    <<<"$api_probe_response"; then
    fail "local API tests.probe leaked engine-only profile data"
fi
api_probe_bad_payload="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-probe-invalid-0001",
        operation: "tests.probe",
        deadline_ms: 5000,
        payload: {timeout_seconds: 1, concurrency: 99}
    }' |
        "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.code == "invalid-request"
    and .error.message_key == "api.tests.probe.payload-invalid"
' <<<"$api_probe_bad_payload" >/dev/null ||
    fail "local API tests.probe accepted unsafe concurrency"
api_probe_slow_request="$(
    jq -c '.request_id = "request-probe-slow-0001"' <<<"$api_probe_request"
)"
FAKE_PING_STARTED_FILE="$TMP/probe.started" \
FAKE_PING_DELAY_SECONDS=2 \
    "$CLI" _api-dispatch <<<"$api_probe_slow_request" \
    >"$TMP/probe-slow-response.json" &
api_probe_slow_pid=$!
for _ in {1..100}; do
    [[ -e "$TMP/probe.started" ]] && break
    sleep 0.02
done
[[ -e "$TMP/probe.started" ]] ||
    fail "slow API probe did not reach the bounded worker"
api_probe_busy_request="$(
    jq -c '.request_id = "request-probe-busy-0001"' <<<"$api_probe_request"
)"
api_probe_busy_response="$(
    "$CLI" _api-dispatch <<<"$api_probe_busy_request"
)"
jq -e '
    .status == "error"
    and .error.code == "busy"
    and .error.message_key == "api.tests.probe.busy"
    and .error.retryable == true
' <<<"$api_probe_busy_response" >/dev/null ||
    fail "local API allowed concurrent batch probes to multiply network load"
wait "$api_probe_slow_pid"
jq -e '.status == "ok"' "$TMP/probe-slow-response.json" >/dev/null ||
    fail "serialized batch probe did not finish normally"
api_probe_deadline_request="$(
    jq -c '
        .request_id = "request-probe-deadline-0001"
        | .deadline_ms = 1500
    ' <<<"$api_probe_request"
)"
api_probe_deadline_response="$(
    FAKE_PING_DELAY_SECONDS=10 \
    FAKE_PING_PID_FILE="$TMP/probe-ping.pid" \
        "$CLI" _api-dispatch <<<"$api_probe_deadline_request"
)"
jq -e '
    .status == "error"
    and .error.code == "deadline-exceeded"
    and .error.message_key == "api.tests.probe.deadline"
' <<<"$api_probe_deadline_response" >/dev/null ||
    fail "local API batch probe ignored the whole-request deadline"
api_probe_after_deadline_request="$(
    jq -c '.request_id = "request-probe-after-deadline-0001"' \
        <<<"$api_probe_request"
)"
api_probe_after_deadline_response="$(
    "$CLI" _api-dispatch <<<"$api_probe_after_deadline_request"
)"
jq -e '.status == "ok"' <<<"$api_probe_after_deadline_response" >/dev/null ||
    fail "timed-out local API probe descendants retained the serialization lock"
[[ -s "$TMP/probe-ping.pid" ]] ||
    fail "deadline probe did not start its fake ping worker"
probe_ping_pid="$(<"$TMP/probe-ping.pid")"
sleep 0.2
if kill -0 "$probe_ping_pid" 2>/dev/null; then
    fail "timed-out API probe left a ping worker running"
fi
if find "$VPNCTL_API_ACTION_DIR" -maxdepth 1 -type f \
    \( -name '.probe-result.*' -o -name '.verify-result.*' \) |
    grep -q .; then
    fail "local API query left a temporary result file"
fi

api_verify_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-verify-0001",
        operation: "tests.verify-egress",
        deadline_ms: 30000,
        payload: {
            timeout_seconds: 3,
            include_speed: false
        }
    }'
)"
api_verify_response="$(
    printf '%s\n' "$api_verify_request" | "$CLI" _api-dispatch
)"
jq -e '
    .api_version == "1.0"
    and .request_id == "request-verify-0001"
    and .status == "ok"
    and .result.verdict == "verified"
    and .result.ipv4.same_egress == true
    and .result.geo.observed_country_code == "BE"
    and .result.geo.providers_agree == true
    and .result.dns.state == "vpn-full-tunnel"
    and .result.speed.requested == false
' <<<"$api_verify_response" >/dev/null ||
    fail "local API tests.verify-egress response is invalid"
if grep -Eq '192\.0\.2\.10|file_name|PrivateKey|PublicKey|Test Server\.ovpn' \
    <<<"$api_verify_response"; then
    fail "local API tests.verify-egress leaked engine-only profile data"
fi
api_verify_bad_payload="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-verify-invalid-0001",
        operation: "tests.verify-egress",
        deadline_ms: 30000,
        payload: {timeout_seconds: 3, include_speed: "yes"}
    }' |
        "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.code == "invalid-request"
    and .error.message_key == "api.tests.verify-egress.payload-invalid"
' <<<"$api_verify_bad_payload" >/dev/null ||
    fail "local API tests.verify-egress accepted an unsafe payload"

mkdir -p "$VPNCTL_API_ACTION_DIR"
(
    flock -x 9
    touch "$TMP/verify.locked"
    sleep 1
) 9>"$VPNCTL_API_ACTION_DIR/.verify.lock" &
verify_lock_pid=$!
for _ in {1..100}; do
    [[ -e "$TMP/verify.locked" ]] && break
    sleep 0.01
done
[[ -e "$TMP/verify.locked" ]] ||
    fail "verify concurrency test did not acquire its lock"
api_verify_busy_response="$(
    "$CLI" _api-dispatch <<<"$api_verify_request"
)"
jq -e '
    .status == "error"
    and .error.code == "busy"
    and .error.message_key == "api.tests.verify-egress.busy"
    and .error.retryable == true
' <<<"$api_verify_busy_response" >/dev/null ||
    fail "local API allowed concurrent egress checks to multiply traffic"
wait "$verify_lock_pid"

api_service_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-service-verify-0001",
        operation: "tests.verify-service-egress",
        deadline_ms: 5000,
        payload: {service: "openai", timeout_seconds: 3}
    }'
)"
api_service_response="$(
    FAKE_SERVICE_STATUS=401 "$CLI" _api-dispatch <<<"$api_service_request"
)"
jq -e '
    .status == "ok"
    and .result.schema_version == 1
    and .result.scope == "unauthenticated-egress"
    and (.result.results | length) == 1
    and .result.results[0].service_id == "openai"
    and .result.results[0].reachability == "reachable"
    and .result.results[0].egress_eligibility == "eligible"
    and .result.results[0].reason_code == "service.openai.auth-boundary-reached"
    and .result.results[0].http_status == 401
' <<<"$api_service_response" >/dev/null ||
    fail "local API tests.verify-service-egress response is invalid"
if grep -Eq 'https://|backend-api|[Cc]ookie|[Aa]uthorization|account' \
    <<<"$api_service_response"; then
    fail "local API service-egress result leaked request or account data"
fi
api_service_bad_payload="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-service-invalid-0001",
        operation: "tests.verify-service-egress",
        deadline_ms: 5000,
        payload: {
            service: "openai",
            timeout_seconds: 3,
            url: "https://example.invalid/"
        }
    }' | "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.code == "invalid-request"
    and .error.message_key == "api.tests.verify-service-egress.payload-invalid"
' <<<"$api_service_bad_payload" >/dev/null ||
    fail "local API service-egress accepted an arbitrary URL"

rm -f -- "$FAKE_SERVICE_PID_FILE"
api_service_deadline_request="$(
    jq -c '
        .request_id = "request-service-deadline-0001"
        | .deadline_ms = 1500
    ' <<<"$api_service_request"
)"
api_service_deadline_response="$(
    FAKE_SERVICE_DELAY_SECONDS=10 \
        "$CLI" _api-dispatch <<<"$api_service_deadline_request"
)"
jq -e '
    .status == "error"
    and .error.code == "deadline-exceeded"
    and .error.message_key == "api.tests.verify-service-egress.deadline"
' <<<"$api_service_deadline_response" >/dev/null ||
    fail "local API service-egress ignored the whole-request deadline"
[[ -s "$FAKE_SERVICE_PID_FILE" ]] ||
    fail "service-egress deadline test did not start its bounded worker"
service_probe_pid="$(<"$FAKE_SERVICE_PID_FILE")"
sleep 0.2
if kill -0 "$service_probe_pid" 2>/dev/null; then
    fail "timed-out service-egress API left a curl worker running"
fi
if find "$VPNCTL_API_ACTION_DIR" -maxdepth 1 -type f \
    -name '.service-verify-result.*' | grep -q .; then
    fail "service-egress API left a temporary result file"
fi

(
    flock -x 9
    touch "$TMP/service-verify.locked"
    sleep 1
) 9>"$VPNCTL_API_ACTION_DIR/.verify-service.lock" &
service_verify_lock_pid=$!
for _ in {1..100}; do
    [[ -e "$TMP/service-verify.locked" ]] && break
    sleep 0.01
done
[[ -e "$TMP/service-verify.locked" ]] ||
    fail "service verify concurrency test did not acquire its lock"
api_service_busy_response="$(
    "$CLI" _api-dispatch <<<"$api_service_request"
)"
jq -e '
    .status == "error"
    and .error.code == "busy"
    and .error.message_key == "api.tests.verify-service-egress.busy"
    and .error.retryable == true
' <<<"$api_service_busy_response" >/dev/null ||
    fail "local API allowed concurrent service egress checks"
wait "$service_verify_lock_pid"
ok "local API query envelopes expose sanitized status, profiles, probes and egress"

api_oversized_request="$(printf 'я%.0s' {1..40})"
api_oversized_response="$(
    printf '%s\n' "$api_oversized_request" |
        VPNCTL_API_MAX_REQUEST_BYTES=64 "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.code == "invalid-request"
    and .error.message_key == "api.request.size"
' <<<"$api_oversized_response" >/dev/null ||
    fail "local API did not enforce its request limit in bytes"
ok "local API bounds request memory before JSON parsing"

api_duplicate_operation_response="$(
    printf '%s\n' \
        '{"api_version":"1.0","request_id":"request-duplicate-0001","operation":"status.get","\u006fperation":"profiles.list","payload":{}}' |
        "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.code == "invalid-request"
    and .error.message_key == "api.request.duplicate-key"
' <<<"$api_duplicate_operation_response" >/dev/null ||
    fail "local API accepted a Unicode-escaped duplicate envelope key"

api_duplicate_payload_response="$(
    printf '%s\n' \
        '{"api_version":"1.0","request_id":"request-duplicate-0002","operation":"status.get","payload":{},"payload":{}}' |
        "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.code == "invalid-request"
    and .error.message_key == "api.request.duplicate-key"
' <<<"$api_duplicate_payload_response" >/dev/null ||
    fail "local API accepted duplicate payload objects"

api_multiple_documents_response="$(
    printf '%s\n' \
        '{"api_version":"1.0","request_id":"request-multiple-0001","operation":"status.get","payload":{}} {"api_version":"1.0","request_id":"request-multiple-0001","operation":"profiles.list","payload":{}}' |
        "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.code == "invalid-request"
    and .error.message_key == "api.request.invalid-json"
' <<<"$api_multiple_documents_response" >/dev/null ||
    fail "local API accepted multiple top-level JSON documents"
ok "local API rejects ambiguous JSON envelopes before dispatch"

api_bounded_query_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-bounded-query-0001",
        operation: "status.get",
        deadline_ms: 100,
        payload: {}
    }'
)"
: >"$FAKE_TIMEOUT_LOG"
api_bounded_query_response="$(
    printf '%s\n' "$api_bounded_query_request" |
        FAKE_SYSTEMCTL_DELAY_SECONDS=5 "$CLI" _api-dispatch
)"
jq -e '
    .status == "ok"
    and .request_id == "request-bounded-query-0001"
' <<<"$api_bounded_query_response" >/dev/null ||
    fail "local API did not fall back to its existing cache after a bounded refresh"
grep -Fq -- \
    "--kill-after=2s 0.100s $CLI _refresh-dashboard-cache" \
    "$FAKE_TIMEOUT_LOG" ||
    fail "local API query ignored its refresh deadline"
if find "$VPNCTL_DASHBOARD_DIR" -maxdepth 1 -type f \
    \( -name '.status.*' -o -name '.profiles.*' \) |
    grep -q .; then
    fail "timed-out local API refresh left temporary cache files"
fi
ok "local API bounds query refreshes and cleans interrupted cache writes"

api_busy_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-busy-0001",
        operation: "lifecycle.reconnect",
        action_id: "action-busy-0001",
        authorization: "system-mutate",
        deadline_ms: 5000,
        payload: {}
    }'
)"
mkdir -p "$VPNCTL_API_ACTION_DIR"
exec 8>"$TMP/run/.mutation.lock"
flock 8
api_busy_response="$(
    printf '%s\n' "$api_busy_request" | "$CLI" _api-dispatch
)"
flock -u 8
exec 8>&-
jq -e '
    .status == "error"
    and .error.code == "busy"
    and .error.retryable == true
    and .error.user_action_required == false
' <<<"$api_busy_response" >/dev/null ||
    fail "local API did not reject a concurrent mutation as retryable busy"
ok "local API serializes concurrent mutations"

api_audit_unavailable_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-audit-unavailable-0001",
        operation: "lifecycle.reconnect",
        action_id: "action-audit-unavailable-0001",
        authorization: "system-mutate",
        deadline_ms: 5000,
        payload: {}
    }'
)"
rm -f -- "$VPNCTL_API_AUDIT_FILE" "$VPNCTL_API_AUDIT_FILE.1"
mkdir "$VPNCTL_API_AUDIT_FILE"
api_audit_systemctl_count="$(wc -l <"$FAKE_SYSTEMCTL_LOG")"
api_audit_unavailable_response="$(
    printf '%s\n' "$api_audit_unavailable_request" |
        "$CLI" _api-dispatch 2>/dev/null
)"
jq -e '
    .status == "error"
    and .error.code == "internal-error"
    and .error.message_key == "api.audit.unavailable"
    and .error.user_action_required == true
' <<<"$api_audit_unavailable_response" >/dev/null ||
    fail "local API did not fail closed when its audit log was unavailable"
[[ "$(wc -l <"$FAKE_SYSTEMCTL_LOG")" == "$api_audit_systemctl_count" ]] ||
    fail "local API executed a mutation without a durable start audit event"
[[ ! -e "$VPNCTL_API_ACTION_DIR/action-audit-unavailable-0001.json" ]] ||
    fail "rejected unaudited API mutation left a running action record"
rmdir "$VPNCTL_API_AUDIT_FILE"
ok "local API requires a durable audit event before mutation"

api_terminal_audit_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-terminal-audit-0001",
        operation: "lifecycle.reconnect",
        action_id: "action-terminal-audit-0001",
        authorization: "system-mutate",
        deadline_ms: 5000,
        payload: {}
    }'
)"
rm -f -- "$VPNCTL_API_AUDIT_FILE" "$VPNCTL_API_AUDIT_FILE.1"
api_audit_archive_blocker="$VPNCTL_API_AUDIT_FILE.1/$(basename "$VPNCTL_API_AUDIT_FILE")"
mkdir -p "$api_audit_archive_blocker"
api_terminal_audit_before="$(wc -l <"$FAKE_SYSTEMCTL_LOG")"
api_terminal_audit_response="$(
    printf '%s\n' "$api_terminal_audit_request" |
        VPNCTL_API_AUDIT_MAX_BYTES=1 "$CLI" _api-dispatch 2>/dev/null
)"
jq -e '
    .status == "error"
    and .error.code == "internal-error"
    and .error.message_key == "api.audit.unavailable"
    and .error.user_action_required == true
' <<<"$api_terminal_audit_response" >/dev/null ||
    fail "local API hid a missing terminal audit event"
api_terminal_audit_after="$(wc -l <"$FAKE_SYSTEMCTL_LOG")"
((api_terminal_audit_after > api_terminal_audit_before)) ||
    fail "terminal audit fault was injected before mutation execution"
jq -e '
    .state == "recovery-required"
    and .action_id == "action-terminal-audit-0001"
    and .operation == "lifecycle.reconnect"
    and .reason == "audit-unavailable"
' "$VPNCTL_STATE_DIR/api-recovery-required.json" >/dev/null ||
    fail "missing terminal audit event did not enter recovery-only mode"
jq -e '
    .state == "completed"
    and .outcome.action_id == "action-terminal-audit-0001"
    and .outcome.state == "succeeded"
' "$VPNCTL_API_ACTION_DIR/action-terminal-audit-0001.json" >/dev/null ||
    {
        jq . "$VPNCTL_API_ACTION_DIR/action-terminal-audit-0001.json" >&2 || true
        fail "terminal audit fault lost the idempotent action outcome"
    }
rmdir "$api_audit_archive_blocker" "$VPNCTL_API_AUDIT_FILE.1"
"$CLI" _api-clear-recovery --acknowledge-current-state >/dev/null
api_terminal_audit_retry="$(
    printf '%s\n' "$api_terminal_audit_request" | "$CLI" _api-dispatch
)"
jq -e '
    .status == "ok"
    and .result.action_id == "action-terminal-audit-0001"
    and .result.state == "succeeded"
' <<<"$api_terminal_audit_retry" >/dev/null ||
    fail "acknowledged terminal audit fault lost its stored action result"
[[ "$(wc -l <"$FAKE_SYSTEMCTL_LOG")" == "$api_terminal_audit_after" ]] ||
    fail "terminal audit retry executed the completed mutation twice"
jq -s -e '
    map(select(
        .event_type == "audit.recorded"
        and .action_id == "action-terminal-audit-0001"
        and .payload.operation == "lifecycle.reconnect"
        and .payload.outcome == "succeeded"
    ))
    | length == 1
' "$VPNCTL_API_AUDIT_FILE" >/dev/null ||
    fail "idempotent retry did not repair exactly one missing terminal audit event"
rm -f -- "$VPNCTL_API_AUDIT_FILE"
ok "local API repairs terminal audit persistence without repeating a mutation"

api_connect_request="$(
    jq -cn --arg profile_id "$profile_id" '{
        api_version: "1.0",
        request_id: "request-connect-0001",
        operation: "lifecycle.connect",
        action_id: "action-connect-0001",
        authorization: "system-mutate",
        deadline_ms: 30000,
        payload: {profile_id: $profile_id}
    }'
)"
: >"$FAKE_SYSTEMCTL_LOG"
: >"$FAKE_DURABILITY_LOG"
api_connect_response="$(printf '%s\n' "$api_connect_request" | "$CLI" _api-dispatch)"
jq -e '
    .status == "ok"
    and .result.action_id == "action-connect-0001"
    and .result.state == "succeeded"
    and .result.rollback.state == "not-needed"
' <<<"$api_connect_response" >/dev/null ||
    fail "local API lifecycle.connect did not succeed"
awk -v action="$VPNCTL_API_ACTION_DIR/action-connect-0001.json" \
    -v audit="$VPNCTL_API_AUDIT_FILE" \
    -v state="$VPNCTL_STATE_DIR/active" '
    $0 == "sync -f " action && !running_sync { running_sync = NR }
    $0 == "sync -f " action {
        action_sync_count++
        if (action_sync_count == 2) completed_sync = NR
    }
    $0 == "sync -f " audit && !start_audit_sync { start_audit_sync = NR }
    $0 == "sync -f " state && !state_sync { state_sync = NR }
    /^systemctl (start|stop|restart) vpnctl[.]service$/ && !mutation { mutation = NR }
    END {
        exit !(running_sync && start_audit_sync && mutation &&
            state_sync && completed_sync &&
            running_sync < mutation && start_audit_sync < mutation &&
            state_sync < completed_sync)
    }
' "$FAKE_DURABILITY_LOG" ||
    {
        sed -n '1,120p' "$FAKE_DURABILITY_LOG" >&2
        fail "local API mutation started before its journal and audit were durable"
    }
api_start_count="$(grep -c '^start vpnctl.service$' "$FAKE_SYSTEMCTL_LOG")"
api_replay_request="$(
    jq -c '.request_id = "request-connect-replay-0001"' <<<"$api_connect_request"
)"
api_replay_response="$(printf '%s\n' "$api_replay_request" | "$CLI" _api-dispatch)"
jq -e '
    .request_id == "request-connect-replay-0001"
    and .status == "ok"
    and .result.state == "succeeded"
' <<<"$api_replay_response" >/dev/null ||
    fail "local API did not replay the stored action outcome"
[[ "$(grep -c '^start vpnctl.service$' "$FAKE_SYSTEMCTL_LOG")" == "$api_start_count" ]] ||
    fail "repeated local API action ID executed the mutation twice"
[[ "$(stat -c %a "$VPNCTL_API_ACTION_DIR/action-connect-0001.json")" == "600" ]] ||
    fail "local API action journal is not root-only"
ok "local API mutation IDs are persistent and idempotent"

export FAKE_SYSTEMCTL_FAIL_ONCE_FILE="$TMP/api-start-failed-once"
api_rollback_request="$(
    jq -cn --arg profile_id "$profile_id" '{
        api_version: "1.0",
        request_id: "request-rollback-0001",
        operation: "lifecycle.connect",
        action_id: "action-rollback-0001",
        authorization: "system-mutate",
        deadline_ms: 5000,
        payload: {profile_id: $profile_id}
    }'
)"
api_rollback_response="$(
    printf '%s\n' "$api_rollback_request" | "$CLI" _api-dispatch
)"
unset FAKE_SYSTEMCTL_FAIL_ONCE_FILE
jq -e '
    .status == "ok"
    and .result.state == "rolled-back"
    and .result.state_changed == true
    and .result.rollback.required == true
    and .result.rollback.state == "completed"
' <<<"$api_rollback_response" >/dev/null ||
    fail "local API did not report a completed lifecycle rollback"
grep -q '^DESIRED=up$' "$VPNCTL_STATE_DIR/active" ||
    fail "local API rollback did not restore the previous desired state"
[[ "$(grep -c '^start vpnctl.service$' "$FAKE_SYSTEMCTL_LOG")" -gt "$api_start_count" ]] ||
    fail "local API rollback did not restart the previous managed tunnel"
if grep -Eq 'Test Server|192\.0\.2\.10|PrivateKey|PublicKey' \
    "$VPNCTL_API_AUDIT_FILE"; then
    fail "local API audit log leaked profile or endpoint data"
fi
jq -s -e '
    length == 4
    and all(.[];
        .api_version == "1.0"
        and .event_type == "audit.recorded"
        and (.event_id | startswith("audit-"))
        and .payload.authorization == "system-mutate"
        and (.payload.outcome |
            IN("started", "succeeded", "rolled-back", "timed-out",
               "rollback-failed"))
    )
' "$VPNCTL_API_AUDIT_FILE" >/dev/null ||
    fail "local API audit log does not use sanitized v1 event envelopes"
[[ "$(stat -c %a "$VPNCTL_API_AUDIT_FILE")" == "600" ]] ||
    fail "local API audit log is not root-only"
api_conflict_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-conflict-0001",
        operation: "lifecycle.disconnect",
        action_id: "action-connect-0001",
        authorization: "system-mutate",
        deadline_ms: 5000,
        payload: {}
    }'
)"
api_conflict_response="$(
    printf '%s\n' "$api_conflict_request" | "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.code == "conflict"
    and .error.user_action_required == true
' <<<"$api_conflict_response" >/dev/null ||
    fail "local API accepted one action ID for two different mutations"

api_interrupted_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-interrupted-0001",
        operation: "lifecycle.disconnect",
        action_id: "action-interrupted-0001",
        authorization: "system-mutate",
        deadline_ms: 5000,
        payload: {}
    }'
)"
api_interrupted_fingerprint="$(
    jq -cS 'del(.request_id)' <<<"$api_interrupted_request" |
        sha256sum |
        awk '{ print $1 }'
)"
cp "$VPNCTL_STATE_DIR/active" \
    "$VPNCTL_API_ACTION_DIR/.snapshot.action-interrupted-0001"
sed -i 's/^DESIRED=up$/DESIRED=down/' "$VPNCTL_STATE_DIR/active"
jq -cn \
    --arg fingerprint "$api_interrupted_fingerprint" \
    '{
        fingerprint: $fingerprint,
        state: "running",
        operation: "lifecycle.disconnect",
        started_at: "2026-01-01T00:00:00Z",
        snapshot_existed: true
    }' >"$VPNCTL_API_ACTION_DIR/action-interrupted-0001.json"
api_stop_count="$(grep -c '^stop vpnctl.service$' "$FAKE_SYSTEMCTL_LOG" || true)"
api_interrupted_response="$(
    printf '%s\n' "$api_interrupted_request" | "$CLI" _api-dispatch
)"
jq -e '
    .status == "ok"
    and .result.action_id == "action-interrupted-0001"
    and .result.state == "rolled-back"
    and .result.rollback.state == "completed"
    and .result.message_key == "api.action.interrupted-rolled-back"
' <<<"$api_interrupted_response" >/dev/null ||
    fail "local API did not reconcile an interrupted action"
grep -q '^DESIRED=up$' "$VPNCTL_STATE_DIR/active" ||
    fail "interrupted local API action did not restore its state snapshot"
[[ ! -e "$VPNCTL_API_ACTION_DIR/.snapshot.action-interrupted-0001" ]] ||
    fail "interrupted local API action left its snapshot behind"
[[ "$(grep -c '^stop vpnctl.service$' "$FAKE_SYSTEMCTL_LOG" || true)" == \
   "$api_stop_count" ]] ||
    fail "interrupted local API action was executed again after recovery"

api_missing_snapshot_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-missing-snapshot-0001",
        operation: "lifecycle.disconnect",
        action_id: "action-missing-snapshot-0001",
        authorization: "system-mutate",
        deadline_ms: 5000,
        payload: {}
    }'
)"
api_missing_snapshot_fingerprint="$(
    jq -cS 'del(.request_id)' <<<"$api_missing_snapshot_request" |
        sha256sum |
        awk '{ print $1 }'
)"
jq -cn \
    --arg fingerprint "$api_missing_snapshot_fingerprint" \
    '{
        fingerprint: $fingerprint,
        state: "running",
        operation: "lifecycle.disconnect",
        started_at: "2026-01-01T00:00:00Z",
        snapshot_existed: true
    }' >"$VPNCTL_API_ACTION_DIR/action-missing-snapshot-0001.json"
api_missing_snapshot_response="$(
    printf '%s\n' "$api_missing_snapshot_request" | "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.code == "internal-error"
    and .error.message_key == "api.action.recovery-required"
    and .error.retryable == false
    and .error.user_action_required == true
' <<<"$api_missing_snapshot_response" >/dev/null ||
    fail "missing API snapshot did not enter recovery-only mode"
api_recovery_marker="$VPNCTL_STATE_DIR/api-recovery-required.json"
jq -e '
    .state == "recovery-required"
    and .action_id == "action-missing-snapshot-0001"
    and .operation == "lifecycle.disconnect"
    and .reason == "snapshot-missing"
    and (.created_at | type == "string")
' "$api_recovery_marker" >/dev/null ||
    fail "local API recovery marker is missing or unsafe"
[[ "$(stat -c %a "$api_recovery_marker")" == "600" ]] ||
    fail "local API recovery marker is not root-only"
api_blocked_stop_count="$(grep -c '^stop vpnctl.service$' "$FAKE_SYSTEMCTL_LOG" || true)"
api_blocked_response="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-recovery-blocked-0001",
        operation: "lifecycle.disconnect",
        action_id: "action-recovery-blocked-0001",
        authorization: "system-mutate",
        deadline_ms: 5000,
        payload: {}
    }' | "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.message_key == "api.action.recovery-required"
' <<<"$api_blocked_response" >/dev/null ||
    fail "local API accepted a mutation while recovery-only mode was active"
[[ "$(grep -c '^stop vpnctl.service$' "$FAKE_SYSTEMCTL_LOG" || true)" == \
   "$api_blocked_stop_count" ]] ||
    fail "recovery-only mode executed a blocked mutation"
"$CLI" _api-clear-recovery --acknowledge-current-state >/dev/null
[[ ! -e "$api_recovery_marker" ]] ||
    fail "explicit administrator acknowledgement did not clear recovery-only mode"

api_deadline_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-deadline-0001",
        operation: "lifecycle.reconnect",
        action_id: "action-deadline-0001",
        authorization: "system-mutate",
        deadline_ms: 5000,
        payload: {}
    }'
)"
: >"$FAKE_TIMEOUT_LOG"
api_delayed_systemctl_marker="$TMP/api-systemctl-delay-once"
api_delayed_systemctl_pid_file="$TMP/api-systemctl-delay.pid"
api_deadline_response="$(
    printf '%s\n' "$api_deadline_request" |
        FAKE_SYSTEMCTL_DELAY_ONCE_FILE="$api_delayed_systemctl_marker" \
        FAKE_SYSTEMCTL_DELAY_PID_FILE="$api_delayed_systemctl_pid_file" \
        FAKE_SYSTEMCTL_DELAY_ONCE_SECONDS=10 \
        VPNCTL_API_CACHE_REFRESH_TIMEOUT_SECONDS=2 \
        "$CLI" _api-dispatch
)"
jq -e '
    .status == "ok"
    and .result.action_id == "action-deadline-0001"
    and .result.state == "timed-out"
    and .result.rollback.state == "completed"
' <<<"$api_deadline_response" >/dev/null ||
    fail "local API did not time out and roll back a sub-second mutation"
grep -Eq -- \
    '--kill-after=5s [0-4]\.[0-9]{3}s .*/mazzy-vpn reconnect' \
    "$FAKE_TIMEOUT_LOG" ||
    fail "local API did not account for preflight time in the mutation deadline"
api_delayed_systemctl_pid="$(cat "$api_delayed_systemctl_pid_file")"
for _ in {1..20}; do
    api_delayed_systemctl_state="$(
        ps -o stat= -p "$api_delayed_systemctl_pid" 2>/dev/null || true
    )"
    if ! kill -0 "$api_delayed_systemctl_pid" 2>/dev/null ||
       [[ "$api_delayed_systemctl_state" == Z* ]]; then
        break
    fi
    sleep 0.1
done
api_delayed_systemctl_state="$(
    ps -o stat= -p "$api_delayed_systemctl_pid" 2>/dev/null || true
)"
if kill -0 "$api_delayed_systemctl_pid" 2>/dev/null &&
   [[ "$api_delayed_systemctl_state" != Z* ]]; then
    fail "timed-out local API mutation left a systemctl descendant running"
fi
ok "local API accounts preflight time and terminates timed-out descendants"

sed -i 's/^DESIRED=up$/DESIRED=down/' "$VPNCTL_STATE_DIR/active"
api_stop_rollback_request="$(
    jq -cn --arg profile_id "$profile_id" '{
        api_version: "1.0",
        request_id: "request-stop-rollback-0001",
        operation: "lifecycle.connect",
        action_id: "action-stop-rollback-0001",
        authorization: "system-mutate",
        deadline_ms: 5000,
        payload: {profile_id: $profile_id}
    }'
)"
: >"$FAKE_TIMEOUT_LOG"
api_stop_rollback_response="$(
    printf '%s\n' "$api_stop_rollback_request" |
        FAKE_SYSTEMCTL_START_FAIL=1 FAKE_SYSTEMCTL_STOP_FAIL=1 \
        VPNCTL_API_ROLLBACK_TIMEOUT_SECONDS=120 \
        "$CLI" _api-dispatch
)"
jq -e '
    .status == "ok"
    and .result.state == "rollback-failed"
    and .result.rollback.state == "failed"
' <<<"$api_stop_rollback_response" >/dev/null ||
    fail "local API reported a failed stop rollback as successful: $api_stop_rollback_response"
[[ -s "$api_recovery_marker" ]] ||
    fail "failed stop rollback did not enter recovery-only mode"
grep -Fq -- \
    '--kill-after=5s 30s systemctl stop vpnctl.service' \
    "$FAKE_TIMEOUT_LOG" ||
    fail "rollback timeout exceeded the client completion grace contract"
"$CLI" _api-clear-recovery --acknowledge-current-state >/dev/null
sed -i 's/^DESIRED=down$/DESIRED=up/' "$VPNCTL_STATE_DIR/active"
[[ -s "$VPNCTL_STATE_DIR/transition-recovery-required.json" ]] ||
    fail "failed stop rollback did not preserve the transition recovery marker"
"$CLI" reconnect >/dev/null
[[ ! -e "$VPNCTL_STATE_DIR/transition-recovery-required.json" ]] ||
    fail "explicit verified reconnect did not clear transition recovery state"
ok "local API fails closed when rollback cannot stop the service"

api_retention_request="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-retention-0001",
        operation: "lifecycle.reconnect",
        action_id: "action-retention-0001",
        authorization: "system-mutate",
        deadline_ms: 5000,
        payload: {}
    }'
)"
api_retention_response="$(
    printf '%s\n' "$api_retention_request" |
        VPNCTL_API_ACTION_MAX_RECORDS=3 VPNCTL_API_AUDIT_MAX_BYTES=1 \
            "$CLI" _api-dispatch
)"
jq -e '
    .status == "ok"
    and .result.action_id == "action-retention-0001"
    and .result.state == "succeeded"
' <<<"$api_retention_response" >/dev/null ||
    fail "local API retention test mutation failed"
[[ "$(find "$VPNCTL_API_ACTION_DIR" -maxdepth 1 -type f -name '*.json' |
    wc -l)" -le 3 ]] ||
    fail "local API completed action journal exceeded its configured bound"
[[ -r "$VPNCTL_API_ACTION_DIR/action-retention-0001.json" ]] ||
    fail "local API pruned the current action outcome"
[[ -s "$VPNCTL_API_AUDIT_FILE" && -s "$VPNCTL_API_AUDIT_FILE.1" ]] ||
    fail "local API audit log did not rotate at its configured bound"
[[ "$(stat -c %a "$VPNCTL_API_AUDIT_FILE")" == "600" &&
   "$(stat -c %a "$VPNCTL_API_AUDIT_FILE.1")" == "600" ]] ||
    fail "rotated local API audit files are not root-only"
if grep -Eq 'Test Server|192\.0\.2\.10|PrivateKey|PublicKey' \
    "$VPNCTL_API_AUDIT_FILE" "$VPNCTL_API_AUDIT_FILE.1"; then
    fail "rotated local API audit log leaked profile or endpoint data"
fi
ok "local API recovery fails closed and bounds persistent journals"

boot_action_id="action-boot-recovery-0001"
cp "$VPNCTL_STATE_DIR/active" \
    "$VPNCTL_API_ACTION_DIR/.snapshot.$boot_action_id"
sed -i 's/^DESIRED=up$/DESIRED=down/' "$VPNCTL_STATE_DIR/active"
jq -cn '{
    fingerprint: "boot-recovery-fingerprint",
    state: "running",
    operation: "lifecycle.disconnect",
    started_at: "2026-01-01T00:00:00Z",
    snapshot_existed: true
}' >"$VPNCTL_API_ACTION_DIR/$boot_action_id.json"
: >"$FAKE_SYSTEMCTL_LOG"
"$CLI" _api-recover-interrupted-actions >/dev/null
grep -q '^DESIRED=up$' "$VPNCTL_STATE_DIR/active" ||
    fail "boot API recovery did not restore the interrupted action snapshot"
jq -e '
    .state == "completed"
    and .outcome.state == "rolled-back"
    and .outcome.rollback.state == "completed"
' "$VPNCTL_API_ACTION_DIR/$boot_action_id.json" >/dev/null ||
    fail "boot API recovery did not persist a successful terminal outcome"
[[ ! -e "$VPNCTL_API_ACTION_DIR/.snapshot.$boot_action_id" ]] ||
    fail "boot API recovery left a recovered snapshot behind"
if grep -Eq '^(start|stop|restart) vpnctl\.service$' "$FAKE_SYSTEMCTL_LOG"; then
    fail "boot API recovery recursively waited on an ordered vpnctl.service job"
fi
ok "boot API recovery reconciles interrupted actions without an ordering deadlock"

transition_boot_marker="$VPNCTL_STATE_DIR/transition-recovery-required.json"
jq -cn '{
    state: "recovery-required",
    reason: "simulated-power-loss",
    created_at: "2026-08-03T00:00:00Z",
    guard_table: "mazzy_vpn_transition"
}' >"$transition_boot_marker"
rm -f -- "$VPNCTL_STATE_DIR/test.transaction"
: >"$FAKE_NFT_RULES_LOG"
if "$CLI" _api-recover-interrupted-actions >/dev/null 2>&1; then
    fail "boot recovery accepted an unresolved transition marker"
fi
grep -q 'oifname "lo" accept' "$FAKE_NFT_RULES_LOG" ||
    fail "boot recovery did not restore loopback access in the fail-closed guard"
grep -q 'chain forward' "$FAKE_NFT_RULES_LOG" ||
    fail "boot recovery did not restore the forwarded-traffic guard"
[[ "$(grep -c 'reject with icmpx type admin-prohibited' \
    "$FAKE_NFT_RULES_LOG")" -ge 2 ]] ||
    fail "boot recovery did not reject both local and forwarded direct egress"
if grep -Eq 'ip6?[[:space:]]+daddr|oifname "(eth0|adguard0|vpnwg0|vpnovpn0)" accept' \
    "$FAKE_NFT_RULES_LOG"; then
    fail "boot recovery restored an unverified egress exception"
fi
transition_service_rc=0
"$CLI" _service-run >/dev/null 2>&1 || transition_service_rc=$?
[[ "$transition_service_rc" -eq 77 ]] ||
    fail "managed service did not stop retries behind the transition recovery marker"
: >"$FAKE_SYSTEMCTL_LOG"
FAKE_SYSTEMCTL_INACTIVE=1 "$CLI" _health-check >/dev/null 2>&1 || true
if grep -Eq '^(start|restart) vpnctl\.service$' "$FAKE_SYSTEMCTL_LOG"; then
    fail "health remediation bypassed the transition recovery marker"
fi
transition_blocked_response="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-transition-blocked-0001",
        operation: "lifecycle.reconnect",
        action_id: "action-transition-blocked-0001",
        authorization: "system-mutate",
        deadline_ms: 5000,
        payload: {}
    }' | "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.message_key == "api.action.recovery-required"
    and .error.user_action_required == true
' <<<"$transition_blocked_response" >/dev/null ||
    fail "local API lifecycle bypassed the transition recovery marker"
rm -f -- "$transition_boot_marker"
ok "boot transition recovery restores a process-wide fail-closed guard"

umask_probe_env="$TMP/api-recovery-umask-env"
umask_probe_file="$TMP/api-recovery-umask-probe"
umask_probe_record="$VPNCTL_API_ACTION_DIR/umask-probe!.json"
cat >"$umask_probe_env" <<'EOF'
if [[ -z "${UMASK_PROBE_ROOT_PID:-}" ]]; then
    export UMASK_PROBE_ROOT_PID="$BASHPID"
    umask 022
    trap '[[ "$BASHPID" != "$UMASK_PROBE_ROOT_PID" ]] || : >"${UMASK_PROBE_FILE:?}"' EXIT
fi
EOF
jq -cn '{state: "running"}' >"$umask_probe_record"
if BASH_ENV="$umask_probe_env" UMASK_PROBE_FILE="$umask_probe_file" \
    "$CLI" _api-recover-interrupted-actions >/dev/null 2>&1; then
    fail "boot API recovery accepted a corrupt action journal name"
fi
[[ "$(stat -c %a "$umask_probe_file")" == "644" ]] ||
    fail "api_mark_recovery_required leaked its restrictive umask to the caller"
rm -f -- "$umask_probe_record"
"$CLI" _api-clear-recovery --acknowledge-current-state >/dev/null
ok "API recovery marker creation restores the caller umask"

boot_failed_action_id="action-boot-recovery-failed-0001"
jq -cn '{
    fingerprint: "boot-recovery-failed-fingerprint",
    state: "running",
    operation: "lifecycle.disconnect",
    started_at: "2026-01-01T00:00:00Z",
    snapshot_existed: true
}' >"$VPNCTL_API_ACTION_DIR/$boot_failed_action_id.json"
if "$CLI" _api-recover-interrupted-actions >/dev/null 2>&1; then
    fail "boot API recovery accepted an interrupted action without a snapshot"
fi
jq -e --arg action_id "$boot_failed_action_id" '
    .state == "recovery-required"
    and .action_id == $action_id
    and .reason == "snapshot-missing"
' "$VPNCTL_STATE_DIR/api-recovery-required.json" >/dev/null ||
    fail "failed boot rollback did not preserve the recovery marker"
if "$CLI" _api-recover-interrupted-actions >/dev/null 2>&1; then
    fail "boot API recovery ignored its fail-closed recovery marker"
fi
[[ -s "$VPNCTL_STATE_DIR/api-recovery-required.json" ]] ||
    fail "repeated boot recovery cleared the recovery marker"
if "$CLI" _service-run >/dev/null 2>&1; then
    fail "managed tunnel started while boot recovery required review"
fi
"$CLI" _api-clear-recovery --acknowledge-current-state >/dev/null
ok "boot API recovery failure remains fail closed until administrator acknowledgement"

marker_unavailable_action_id="action-boot-marker-unavailable-0001"
jq -cn '{
    fingerprint: "boot-marker-unavailable-fingerprint",
    state: "running",
    operation: "lifecycle.disconnect",
    started_at: "2026-01-01T00:00:00Z",
    snapshot_existed: true
}' >"$VPNCTL_API_ACTION_DIR/$marker_unavailable_action_id.json"
marker_mv_bin="$TMP/marker-mv-bin"
mkdir -p "$marker_mv_bin"
cat >"$marker_mv_bin/mv" <<'EOF'
#!/usr/bin/env bash
target="${@: -1}"
[[ "$target" != "${FAKE_MV_FAIL_TARGET:?}" ]] || exit 1
exec /usr/bin/mv "$@"
EOF
chmod 700 "$marker_mv_bin/mv"
if PATH="$marker_mv_bin:$PATH" \
    FAKE_MV_FAIL_TARGET="$VPNCTL_STATE_DIR/api-recovery-required.json" \
    "$CLI" _api-recover-interrupted-actions >/dev/null 2>&1; then
    fail "boot API recovery continued when its fail-closed marker could not persist"
fi
[[ ! -e "$VPNCTL_STATE_DIR/api-recovery-required.json" ]] ||
    fail "failed marker persistence left an untrusted recovery marker"
if find "$VPNCTL_STATE_DIR" -maxdepth 1 -name '.api-recovery.*' -print -quit |
   grep -q .; then
    fail "failed marker persistence left a temporary marker behind"
fi
rm -f -- "$VPNCTL_API_ACTION_DIR/$marker_unavailable_action_id.json"
ok "boot recovery reports marker-persistence failure without leaking temporary state"

api_recovery_bad_run_dir="$TMP/api-recovery-run-path"
: >"$api_recovery_bad_run_dir"
if VPNCTL_RUN_DIR="$api_recovery_bad_run_dir" \
    "$CLI" _api-recover-interrupted-actions >/dev/null 2>&1; then
    fail "boot API recovery accepted an unavailable runtime directory"
fi
jq -e '
    .state == "recovery-required"
    and .action_id == "boot-recovery"
    and .operation == "api.interrupted-recovery"
    and .reason == "boot-recovery-directory-unavailable"
' "$VPNCTL_STATE_DIR/api-recovery-required.json" >/dev/null ||
    fail "runtime-directory failure did not persist the API recovery marker"
rm -f -- "$api_recovery_bad_run_dir"
"$CLI" _api-clear-recovery --acknowledge-current-state >/dev/null

api_recovery_chmod_bin="$TMP/api-recovery-chmod-bin"
mkdir -p "$api_recovery_chmod_bin"
cat >"$api_recovery_chmod_bin/chmod" <<'EOF'
#!/usr/bin/env bash
for argument in "$@"; do
    if [[ "$argument" == "${FAKE_CHMOD_FAIL_TARGET:?}" ]]; then
        exit 1
    fi
done
exec /usr/bin/chmod "$@"
EOF
chmod 700 "$api_recovery_chmod_bin/chmod"
if PATH="$api_recovery_chmod_bin:$PATH" \
    FAKE_CHMOD_FAIL_TARGET="$VPNCTL_RUN_DIR" \
    "$CLI" _api-recover-interrupted-actions >/dev/null 2>&1; then
    fail "boot API recovery accepted an unprotected runtime directory"
fi
jq -e '
    .state == "recovery-required"
    and .reason == "boot-recovery-permissions-unavailable"
' "$VPNCTL_STATE_DIR/api-recovery-required.json" >/dev/null ||
    fail "runtime-directory chmod failure did not persist the API recovery marker"
"$CLI" _api-clear-recovery --acknowledge-current-state >/dev/null

exec 6>"$VPNCTL_RUN_DIR/.mutation.lock"
flock 6
if VPNCTL_API_RECOVERY_LOCK_WAIT_SECONDS=1 \
    "$CLI" _api-recover-interrupted-actions >/dev/null 2>&1; then
    fail "boot API recovery bypassed the occupied shared mutation lock"
fi
flock -u 6
exec 6>&-
jq -e '
    .state == "recovery-required"
    and .action_id == "boot-recovery"
    and .operation == "api.interrupted-recovery"
    and .reason == "boot-recovery-lock-unavailable"
' "$VPNCTL_STATE_DIR/api-recovery-required.json" >/dev/null ||
    fail "shared-lock timeout did not persist the API recovery marker"
[[ "$(stat -c %a "$VPNCTL_STATE_DIR/api-recovery-required.json")" == "600" ]] ||
    fail "boot infrastructure failure marker is not root-only"
api_marker_service_rc=0
"$CLI" _service-run >/dev/null 2>&1 || api_marker_service_rc=$?
[[ "$api_marker_service_rc" -eq 77 ]] ||
    fail "managed service did not stop retries behind the API recovery marker"
: >"$FAKE_SYSTEMCTL_LOG"
FAKE_SYSTEMCTL_INACTIVE=1 "$CLI" _health-check >/dev/null 2>&1 || true
if grep -Eq '^(start|restart) vpnctl\.service$' "$FAKE_SYSTEMCTL_LOG"; then
    fail "health remediation bypassed the boot recovery lock failure marker"
fi
api_boot_lock_blocked_response="$(
    jq -cn '{
        api_version: "1.0",
        request_id: "request-boot-lock-blocked-0001",
        operation: "lifecycle.reconnect",
        action_id: "action-boot-lock-blocked-0001",
        authorization: "system-mutate",
        deadline_ms: 5000,
        payload: {}
    }' | "$CLI" _api-dispatch
)"
jq -e '
    .status == "error"
    and .error.message_key == "api.action.recovery-required"
' <<<"$api_boot_lock_blocked_response" >/dev/null ||
    fail "local API lifecycle bypassed the boot recovery lock failure marker"
"$CLI" _api-clear-recovery --acknowledge-current-state >/dev/null
ok "boot API infrastructure failures persist a fail-closed marker"

assert_recovery_marker_blocks_runtime() {
    local request_id="$1" blocked_response blocked_service_rc=0
    "$CLI" _service-run >/dev/null 2>&1 || blocked_service_rc=$?
    [[ "$blocked_service_rc" -eq 77 ]] ||
        fail "managed service did not stop retries behind recovery marker $request_id"
    : >"$FAKE_SYSTEMCTL_LOG"
    FAKE_SYSTEMCTL_INACTIVE=1 "$CLI" _health-check >/dev/null 2>&1 || true
    if grep -Eq '^(start|restart) vpnctl\.service$' "$FAKE_SYSTEMCTL_LOG"; then
        fail "health remediation bypassed recovery marker $request_id"
    fi
    blocked_response="$(
        jq -cn --arg request_id "$request_id" '{
            api_version: "1.0",
            request_id: $request_id,
            operation: "lifecycle.reconnect",
            action_id: ("action-" + $request_id),
            authorization: "system-mutate",
            deadline_ms: 5000,
            payload: {}
        }' | "$CLI" _api-dispatch
    )"
    jq -e '
        .status == "error"
        and .error.message_key == "api.action.recovery-required"
    ' <<<"$blocked_response" >/dev/null ||
        fail "local API bypassed recovery marker $request_id"
}

: >"$FAKE_SYNC_LOG"
corrupt_recovery_record="$VPNCTL_API_ACTION_DIR/broken!.json"
jq -cn '{state: "running"}' >"$corrupt_recovery_record"
if "$CLI" _api-recover-interrupted-actions >/dev/null 2>&1; then
    fail "boot recovery accepted a corrupt action journal name"
fi
jq -e '
    .state == "recovery-required"
    and .action_id == "recovery-scan"
    and .operation == "api.interrupted-recovery"
    and .reason == "journal-corrupt"
' "$VPNCTL_STATE_DIR/api-recovery-required.json" >/dev/null ||
    fail "journal-corrupt recovery marker schema is invalid"
[[ "$(stat -c %a "$VPNCTL_STATE_DIR/api-recovery-required.json")" == "600" ]] ||
    fail "journal-corrupt recovery marker is not mode 600"
grep -Fxq -- "-f $VPNCTL_STATE_DIR/api-recovery-required.json" "$FAKE_SYNC_LOG" ||
    fail "journal-corrupt recovery marker was not durably synced"
assert_recovery_marker_blocks_runtime request-journal-corrupt-0001
rm -f -- "$corrupt_recovery_record"
"$CLI" _api-clear-recovery --acknowledge-current-state >/dev/null

journal_unavailable_id="action-journal-unavailable-0001"
journal_unavailable_record="$VPNCTL_API_ACTION_DIR/$journal_unavailable_id.json"
cp "$VPNCTL_STATE_DIR/active" \
    "$VPNCTL_API_ACTION_DIR/.snapshot.$journal_unavailable_id"
jq -cn '{
    fingerprint: "journal-unavailable-fingerprint",
    state: "running",
    operation: "lifecycle.reconnect",
    started_at: "2026-01-01T00:00:00Z",
    snapshot_existed: true
}' >"$journal_unavailable_record"
journal_mv_bin="$TMP/journal-mv-bin"
mkdir -p "$journal_mv_bin"
cat >"$journal_mv_bin/mv" <<'EOF'
#!/usr/bin/env bash
target="${@: -1}"
[[ "$target" != "${FAKE_MV_FAIL_TARGET:?}" ]] || exit 1
exec /usr/bin/mv "$@"
EOF
chmod 700 "$journal_mv_bin/mv"
: >"$FAKE_SYNC_LOG"
if PATH="$journal_mv_bin:$PATH" \
    FAKE_MV_FAIL_TARGET="$journal_unavailable_record" \
    "$CLI" _api-recover-interrupted-actions >/dev/null 2>&1; then
    fail "boot recovery accepted an unavailable action journal"
fi
jq -e --arg action_id "$journal_unavailable_id" '
    .state == "recovery-required"
    and .action_id == $action_id
    and .operation == "lifecycle.reconnect"
    and .reason == "journal-unavailable"
' "$VPNCTL_STATE_DIR/api-recovery-required.json" >/dev/null ||
    fail "journal-unavailable recovery marker schema is invalid"
[[ "$(stat -c %a "$VPNCTL_STATE_DIR/api-recovery-required.json")" == "600" ]] ||
    fail "journal-unavailable recovery marker is not mode 600"
grep -Fxq -- "-f $VPNCTL_STATE_DIR/api-recovery-required.json" "$FAKE_SYNC_LOG" ||
    fail "journal-unavailable recovery marker was not durably synced"
assert_recovery_marker_blocks_runtime request-journal-unavailable-0001
rm -f -- \
    "$journal_unavailable_record" \
    "$VPNCTL_API_ACTION_DIR/.snapshot.$journal_unavailable_id"
"$CLI" _api-clear-recovery --acknowledge-current-state >/dev/null
ok "all boot journal failure markers are mode 600, synced and fail closed"

real_path="${PATH#"$TMP/fakebin:"}"
real_socat="$(PATH="$real_path" command -v socat)" ||
    fail "real socat is required for the API half-close integration test"
real_api_socket="$TMP/api-client-integration.sock"
real_api_responder="$TMP/api-client-responder"
cat >"$real_api_responder" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
IFS= read -r request
request_id="$(jq -er '.request_id' <<<"$request")"
sleep 1
jq -cn --arg request_id "$request_id" '{
    api_version: "1.0",
    request_id: $request_id,
    status: "ok",
    result: {profiles: []}
}'
EOF
chmod 700 "$real_api_responder"
timeout 20 "$real_socat" "UNIX-LISTEN:$real_api_socket,fork" \
    "EXEC:$real_api_responder" &
real_api_listener_pid=$!
for _ in {1..50}; do
    [[ -S "$real_api_socket" ]] && break
    sleep 0.02
done
[[ -S "$real_api_socket" ]] || fail "real API integration socket did not start"
real_api_response="$(
    PATH="/usr/bin:/bin:$PATH" \
        VPNCTL_API_SOCKET="$real_api_socket" \
        VPNCTL_API_CLIENT_FORCE=1 \
        "$CLI" profiles --api-json
)" || fail "API client closed the response half after request EOF"
kill "$real_api_listener_pid" 2>/dev/null || true
wait "$real_api_listener_pid" 2>/dev/null || true
jq -e '
    .api_version == "1.0"
    and .status == "ok"
    and .result.profiles == []
' <<<"$real_api_response" >/dev/null ||
    fail "real API socket integration returned an invalid response"
ok "local API client preserves the response half after request EOF"

api_client_status="$(
    VPNCTL_API_CLIENT_FORCE=1 "$CLI" status --api-json
)"
jq -e '
    .api_version == "1.0"
    and .status == "ok"
    and .result.connection.state == "connected"
' <<<"$api_client_status" >/dev/null ||
    fail "CLI local API client did not return the status.get envelope"

api_client_human_status="$(
    VPNCTL_API_CLIENT_FORCE=1 "$CLI" status
)"
for expected_status_line in \
    'Desired:   up' \
    'Mode:      normal' \
    'Autostart: enabled' \
    'Monitor:   enabled' \
    'Fallback:  inactive' \
    'Interface: vpnovpn0' \
    'Public IP: 203.0.113.7'; do
    grep -qx "$expected_status_line" <<<"$api_client_human_status" ||
        fail "CLI local API status omitted safe detail: $expected_status_line"
done

api_client_profiles="$(
    VPNCTL_API_CLIENT_FORCE=1 "$CLI" profiles --api-json
)"
jq -e --arg profile_id "$profile_id" '
    .api_version == "1.0"
    and .status == "ok"
    and any(
        .result.profiles[];
        .profile_id == $profile_id
        and .display_name == "Test Server"
        and .protocol == "openvpn"
    )
' <<<"$api_client_profiles" >/dev/null ||
    fail "CLI local API client did not return the profiles.list envelope"
if grep -Eq 'file_name|192\.0\.2\.10|PrivateKey|PublicKey' \
    <<<"$api_client_profiles"; then
    fail "CLI local API profile response leaked engine-only profile data"
fi
api_client_probe="$(
    VPNCTL_API_CLIENT_FORCE=1 "$CLI" probe openvpn --timeout 1 --jobs 2 --json
)"
jq -e '
    .schema_version == 1
    and .summary.total == 1
    and .summary.reachable == 1
    and .results[0].display_name == "Test Server"
    and .results[0].latency_ms == 12
' <<<"$api_client_probe" >/dev/null ||
    fail "CLI did not route the structured endpoint probe through the local API"
if grep -Eq '192\.0\.2\.10|file_name|PrivateKey|PublicKey' \
    <<<"$api_client_probe"; then
    fail "CLI local API probe response leaked engine-only profile data"
fi

api_client_legacy_status="$(
    VPNCTL_API_CLIENT_FORCE=1 "$CLI" status --json
)"
jq -e '
    .schema_version == 1
    and .service_state == "active"
    and (has("api_version") | not)
' <<<"$api_client_legacy_status" >/dev/null ||
    fail "local API availability changed the stable status --json schema"
api_client_legacy_profiles="$(
    VPNCTL_API_CLIENT_FORCE=1 "$CLI" profiles --json
)"
jq -e '
    .schema_version == 1
    and any(.profiles[]; .file_name == "Test Server.ovpn")
    and (has("api_version") | not)
' <<<"$api_client_legacy_profiles" >/dev/null ||
    fail "local API availability changed the stable profiles --json schema"

api_client_list="$(
    VPNCTL_API_CLIENT_FORCE=1 "$CLI" list openvpn
)"
grep -q 'Test Server' <<<"$api_client_list" ||
    fail "CLI local API profile list omitted the display name"
grep -q '192\.0\.2\.10' <<<"$api_client_list" &&
    fail "CLI local API profile list leaked the endpoint"
if FAKE_SOCAT_OVERSIZED=1 VPNCTL_API_CLIENT_FORCE=1 \
    VPNCTL_API_CLIENT_MAX_RESPONSE_BYTES=256 \
    "$CLI" status --api-json >/dev/null 2>&1; then
    fail "CLI local API client accepted an oversized response"
fi
if FAKE_SOCAT_MULTIPLE_RESPONSES=1 VPNCTL_API_CLIENT_FORCE=1 \
    "$CLI" status --api-json >/dev/null 2>&1; then
    fail "CLI local API client accepted multiple JSON response documents"
fi

for api_locale_check in \
    "ru|Профиль 'Missing' не найден" \
    "en|Profile 'Missing' was not found" \
    "de|Profil 'Missing' wurde" \
    "zh|找不到配置 'Missing'（OpenVPN）" \
    "ja|プロファイル 'Missing' が OpenVPN に見つかりません" \
    "ko|'Missing' 프로필을 OpenVPN에서 찾을 수 없습니다"; do
    api_locale="${api_locale_check%%|*}"
    api_locale_marker="${api_locale_check#*|}"
    if api_locale_output="$(
        VPNCTL_LANG="$api_locale" VPNCTL_API_CLIENT_FORCE=1 \
            "$CLI" connect openvpn Missing 2>&1
    )"; then
        fail "localized local API client accepted a missing profile: $api_locale"
    fi
    grep -Fq "$api_locale_marker" <<<"$api_locale_output" ||
        fail "local API client error is not localized for $api_locale"
done

: >"$FAKE_SYSTEMCTL_LOG"
api_client_connect="$(
    VPNCTL_API_CLIENT_FORCE=1 "$CLI" connect openvpn "Test Server"
)"
grep -q 'Подключение запущено' <<<"$api_client_connect" ||
    fail "CLI local API connect did not report success"
[[ "$(grep -c '^start vpnctl.service$' "$FAKE_SYSTEMCTL_LOG")" == "1" ]] ||
    fail "CLI local API connect did not execute exactly once"

: >"$FAKE_SYSTEMCTL_LOG"
: >"$FAKE_TIMEOUT_LOG"
rm -f "$TMP/socat-lost-response"
api_client_reconnect="$(
    FAKE_SOCAT_LOST_RESPONSE_FILE="$TMP/socat-lost-response" \
        VPNCTL_API_CLIENT_COMPLETION_GRACE_SECONDS=15 \
        VPNCTL_API_CLIENT_FORCE=1 "$CLI" reconnect
)"
grep -q 'Переподключение запущено' <<<"$api_client_reconnect" ||
    fail "CLI local API reconnect did not recover from a lost response"
[[ -e "$TMP/socat-lost-response" ]] ||
    fail "CLI local API lost-response scenario was not exercised"
[[ "$(grep -c '^start vpnctl.service$' "$FAKE_SYSTEMCTL_LOG")" == "1" ]] ||
    fail "CLI local API retry executed one reconnect action twice"
grep -Fq -- '--kill-after=2s 90s socat -T 90 STDIO,ignoreeof' \
    "$FAKE_TIMEOUT_LOG" ||
    fail "CLI client did not reserve the bounded rollback completion grace"

api_client_dashboard="$(
    VPNCTL_API_CLIENT_FORCE=1 "$CLI" dashboard
)"
grep -q 'API: 1.0' <<<"$api_client_dashboard" ||
    fail "TUI dashboard did not use the protected local API"
grep -q 'Интерфейс: vpnovpn0' <<<"$api_client_dashboard" ||
    fail "TUI local API dashboard omitted the tunnel interface"
grep -q 'Внешний IP: 203.0.113.7' <<<"$api_client_dashboard" ||
    fail "TUI local API dashboard omitted the current public IP"
grep -q 'Автозапуск: включён' <<<"$api_client_dashboard" ||
    fail "TUI local API dashboard omitted autostart state"
grep -q 'Контроль здоровья: включён' <<<"$api_client_dashboard" ||
    fail "TUI local API dashboard omitted monitor state"
grep -q 'Резерв: внешний VPN не активен' <<<"$api_client_dashboard" ||
    fail "TUI local API dashboard omitted fallback state"
grep -q '192\.0\.2\.10' <<<"$api_client_dashboard" &&
    fail "TUI local API dashboard leaked the endpoint"

api_client_disconnect="$(
    VPNCTL_API_CLIENT_FORCE=1 "$CLI" disconnect
)"
grep -q 'VPN отключён' <<<"$api_client_disconnect" ||
    fail "CLI local API disconnect did not report success"
VPNCTL_API_CLIENT_FORCE=1 "$CLI" connect openvpn "Test Server" >/dev/null
ok "CLI/TUI use opaque local API queries and idempotent lifecycle mutations"

dashboard_output="$("$CLI" dashboard)"
grep -q 'M A Z Z Y' <<<"$dashboard_output" || fail "dashboard artwork is missing"
grep -q 'Локация: Test Server' <<<"$dashboard_output" ||
    fail "dashboard selected location is missing"
grep -q 'Конфиг по умолчанию: Test Server.ovpn' <<<"$dashboard_output" ||
    fail "dashboard default config is missing"
grep -q 'Интернет: ✓ работает' <<<"$dashboard_output" ||
    fail "dashboard connectivity check is missing"
grep -q 'OpenVPN=1' <<<"$dashboard_output" ||
    fail "dashboard profile counts are missing"
quick_output="$("$CLI" quick)"
grep -q 'Уже подключено через default-конфиг' <<<"$quick_output" ||
    fail "quick connect did not reuse the working default config"
ok "dashboard and quick default connection"

mkdir -p "$TMP/selftest-config/openvpn"
cp "$TMP/config/openvpn/Test Server.ovpn" "$TMP/selftest-config/openvpn/"
selftest_output="$(
    PATH="$TMP/doctorbin:$PATH" \
        VPNCTL_CONFIG_DIR="$TMP/selftest-config" "$CLI" self-test --offline
)"
grep -q '^Mazzy VPN self-test: все выбранные проверки пройдены\.$' \
    <<<"$selftest_output" || fail "offline self-test did not complete"
ok "offline self-diagnostics"

test_output="$("$CLI" test openvpn "Test Server" --timeout 2)"
grep -q 'TEST OK' <<<"$test_output" || fail "successful tunnel test was not reported"
grep -q '^MODE=normal$' "$TMP/state/active" || fail "test did not restore previous state"
[[ ! -e "$TMP/state/test.transaction" ]] || fail "test transaction was not cleaned"
grep -q -- '--on-active=2s' "$TMP/systemd-run.log" || fail "timeout guard was not armed"
ok "successful test rolls back"

(exec -a "/opt/adguardvpn_cli/adguardvpn-cli connect" sleep 30) &
fake_adguard_transport_pid=$!
printf '%s\n' "$fake_adguard_transport_pid" >"$VPNCTL_ADGUARD_PID_FILE"
touch "$FAKE_ADGUARD_ACTIVE"
: >"$FAKE_ADGUARD_LOG"
export FAKE_ADGUARD_FORK=1
fallback_test_output="$("$CLI" test openvpn "Test Server" --timeout 2)"
grep -q 'TEST OK' <<<"$fallback_test_output" ||
    fail "test over AdGuard fallback did not pass"
[[ -e "$FAKE_ADGUARD_ACTIVE" ]] || fail "AdGuard fallback was not restored"
grep -q '^disconnect$' "$FAKE_ADGUARD_LOG" ||
    fail "AdGuard fallback was not stopped before the managed tunnel"
grep -q '^connect --no-progress -y$' "$FAKE_ADGUARD_LOG" ||
    fail "AdGuard fallback was not reconnected after rollback"
grep -q '^DESIRED=down$' "$TMP/state/active" ||
    fail "managed watchdog remained armed over restored AdGuard fallback"
grep -q 'ip daddr 192.0.2.40 udp dport 443 accept' \
    "$FAKE_NFT_RULES_LOG" ||
    fail "AdGuard transition guard did not preserve the exact transport endpoint"
grep -q 'oifname "adguard0" accept' "$FAKE_NFT_RULES_LOG" ||
    fail "AdGuard transition guard did not preserve the proven tunnel interface"
if grep -q 'oifname "eth0" accept' "$FAKE_NFT_RULES_LOG"; then
    fail "AdGuard transition guard permitted unrestricted physical-interface egress"
fi
wait "$fake_adguard_transport_pid" 2>/dev/null || true
rm -f "$FAKE_ADGUARD_ACTIVE" "$VPNCTL_ADGUARD_PID_FILE"
if ! "$CLI" connect openvpn "Test Server" >/dev/null; then
    fail "daemonized AdGuard fallback inherited the vpnctl mutation lock"
fi

"$CLI" disconnect >/dev/null
(exec -a "/opt/adguardvpn_cli/adguardvpn-cli connect" sleep 30) &
fake_adguard_transport_pid=$!
printf '%s\n' "$fake_adguard_transport_pid" >"$VPNCTL_ADGUARD_PID_FILE"
touch "$FAKE_ADGUARD_ACTIVE"
export FAKE_SYSTEMCTL_START_FAIL=1
api_fallback_fd_request="$(
    jq -cn --arg profile_id "$profile_id" '{
        api_version: "1.0",
        request_id: "request-fallback-fd-0001",
        operation: "lifecycle.connect",
        action_id: "action-fallback-fd-0001",
        authorization: "system-mutate",
        deadline_ms: 10000,
        payload: {profile_id: $profile_id}
    }'
)"
api_fallback_fd_response="$(
    printf '%s\n' "$api_fallback_fd_request" | "$CLI" _api-dispatch
)"
unset FAKE_SYSTEMCTL_START_FAIL
jq -e '
    .status == "ok"
    and .result.state == "rolled-back"
    and .result.rollback.state == "completed"
' <<<"$api_fallback_fd_response" >/dev/null ||
    fail "API fallback fd regression did not complete a rollback"
wait "$fake_adguard_transport_pid" 2>/dev/null || true
rm -f "$FAKE_ADGUARD_ACTIVE" "$VPNCTL_ADGUARD_PID_FILE"
exec 6>"$TMP/run/.mutation.lock"
flock -n 6 || fail "daemonized API fallback retained inherited mutation fd 7"
flock -u 6
exec 6>&-
"$CLI" connect openvpn "Test Server" >/dev/null
unset FAKE_ADGUARD_FORK
ok "AdGuard fallback cannot retain direct or API mutation locks"

rm -f "$FAKE_ADGUARD_ACTIVE"
(exec -a "/opt/adguardvpn_cli/adguardvpn-cli connect" sleep 30) &
fake_adguard_pid=$!
printf '%s\n' "$fake_adguard_pid" >"$VPNCTL_ADGUARD_PID_FILE"
: >"$FAKE_ADGUARD_LOG"
process_fallback_output="$("$CLI" test openvpn "Test Server" --timeout 2)"
grep -q 'TEST OK' <<<"$process_fallback_output" ||
    fail "AdGuard process fallback test did not pass"
if kill -0 "$fake_adguard_pid" 2>/dev/null; then
    fail "stale AdGuard tunnel process was not terminated"
fi
[[ -e "$FAKE_ADGUARD_ACTIVE" ]] ||
    fail "AdGuard process fallback was not restored"
grep -q '^disconnect$' "$FAKE_ADGUARD_LOG" ||
    fail "AdGuard process fallback did not attempt a clean disconnect"
grep -q '^connect --no-progress -y$' "$FAKE_ADGUARD_LOG" ||
    fail "AdGuard process fallback was not reconnected after rollback"
rm -f "$FAKE_ADGUARD_ACTIVE" "$VPNCTL_ADGUARD_PID_FILE"
ok "AdGuard is detected by its validated PID file when status is unavailable"

export FAKE_SYSTEMCTL_START_LIMIT=1
"$CLI" reconnect >/dev/null
batch_output="$("$CLI" test-all openvpn --timeout 2)"
unset FAKE_SYSTEMCTL_START_LIMIT
grep -q 'tested=1 passed=1 failed=0 skipped=0' <<<"$batch_output" ||
    fail "test-all summary is wrong"
grep -q '^MODE=normal$' "$TMP/state/active" || fail "test-all did not restore state"
ok "explicit profile cycling and test-all reset systemd start limit and roll back"

cp "$TMP/config/openvpn/Test Server.ovpn" "$TMP/config/openvpn/Second Server.ovpn"
export FAKE_SYSTEMCTL_OPENVPN_LIMIT=1
batch_limit_rc=0
batch_limit_output="$("$CLI" test-all openvpn --timeout 2 2>&1)" ||
    batch_limit_rc=$?
unset FAKE_SYSTEMCTL_OPENVPN_LIMIT
rm "$TMP/config/openvpn/Second Server.ovpn"
[[ "$batch_limit_rc" -ne 0 ]] ||
    fail "test-all accepted an OpenVPN server connection limit"
grep -q 'tested=1 passed=0 failed=1' <<<"$batch_limit_output" ||
    fail "test-all did not stop after the first OpenVPN server connection limit"
grep -q 'Пакетная проверка остановлена: OpenVPN-сервер отклонил сессию' \
    <<<"$batch_limit_output" ||
    fail "test-all did not explain the OpenVPN server rejection"
grep -q '^MODE=normal$' "$TMP/state/active" ||
    fail "OpenVPN server rejection did not restore the previous state"
ok "test-all stops safely on an OpenVPN server connection limit"

cp "$TMP/state/active" "$TMP/state/test.previous"
touch "$TMP/state/test.previous.exists" "$TMP/state/test.previous.active" "$TMP/state/test.transaction"
sed -i 's/^MODE=normal$/MODE=test/' "$TMP/state/active"
printf 'TEST_TOKEN=guard-token\nTEST_DEADLINE=1\n' >>"$TMP/state/active"
"$CLI" _test-timeout wrong-token
grep -q '^MODE=test$' "$TMP/state/active" || fail "wrong guard token changed state"
"$CLI" _test-timeout guard-token >/dev/null
grep -q '^MODE=normal$' "$TMP/state/active" || fail "guard did not restore previous state"
[[ ! -e "$TMP/state/test.transaction" ]] || fail "guard transaction was not cleaned"
ok "independent timeout guard"

cp "$TMP/state/active" "$TMP/state/test.previous"
touch "$TMP/state/test.previous.exists" "$TMP/state/test.previous.active" \
    "$TMP/state/test.transaction"
sed -i 's/^MODE=normal$/MODE=test/' "$TMP/state/active"
printf 'TEST_TOKEN=race-token\nTEST_DEADLINE=1\n' >>"$TMP/state/active"
exec 7>"$TMP/run/.mutation.lock"
flock 7
timeout 5 "$CLI" _test-timeout race-token >/dev/null 2>&1 &
guard_pid=$!
sleep 0.2
kill -0 "$guard_pid" 2>/dev/null || fail "timeout guard did not wait for mutation lock"
flock -u 7
wait "$guard_pid"
exec 7>&-
grep -q '^MODE=normal$' "$TMP/state/active" || fail "serialized guard did not restore state"
ok "timeout recovery is serialized with active operations"

cp "$TMP/state/active" "$TMP/state/test.previous"
touch "$TMP/state/test.previous.exists" "$TMP/state/test.previous.active" "$TMP/state/test.transaction"
sed -i 's/^MODE=normal$/MODE=test/' "$TMP/state/active"
printf 'TEST_TOKEN=boot-token\nTEST_DEADLINE=1\n' >>"$TMP/state/active"
"$CLI" _recover-stale-test >/dev/null
grep -q '^MODE=normal$' "$TMP/state/active" || fail "boot recovery did not restore state"
[[ ! -e "$TMP/state/test.transaction" ]] || fail "boot recovery transaction was not cleaned"
ok "boot recovery"

cp "$TMP/state/active" "$TMP/state/test.previous"
touch "$TMP/state/test.previous.exists" "$TMP/state/test.transaction"
printf 'legacy\n' >"$TMP/state/test.previous.fallback"
sed -i 's/^MODE=normal$/MODE=test/' "$TMP/state/active"
printf 'TEST_TOKEN=legacy-boot-token\nTEST_DEADLINE=1\n' >>"$TMP/state/active"
rm -f "$FAKE_LEGACY_ACTIVE"
"$CLI" _recover-stale-test >/dev/null
[[ -e "$FAKE_LEGACY_ACTIVE" ]] ||
    fail "boot recovery did not restart legacy fallback"
grep -q '^DESIRED=down$' "$TMP/state/active" ||
    fail "boot recovery left managed watchdog armed over legacy fallback"
rm -f "$FAKE_LEGACY_ACTIVE"
"$CLI" connect openvpn "Test Server" >/dev/null
ok "boot recovery restores external fallback"

export FAKE_CURL_FAIL=1
failed_test_rc=0
"$CLI" test openvpn "Test Server" --timeout 2 >/dev/null 2>&1 ||
    failed_test_rc=$?
[[ "$failed_test_rc" -eq 2 ]] ||
    fail "unverified test rollback did not return the critical status"
unset FAKE_CURL_FAIL
[[ -e "$TMP/state/test.transaction" ]] ||
    fail "unverified test rollback discarded its recovery transaction"
jq -e '.reason == "test-managed-restore-failed"' \
    "$TMP/state/transition-recovery-required.json" >/dev/null ||
    fail "unverified test rollback did not preserve fail-closed recovery state"
"$CLI" _recover-stale-test >/dev/null
grep -q '^MODE=normal$' "$TMP/state/active" ||
    fail "recovery did not restore the previous test state"
[[ ! -e "$TMP/state/test.transaction" ]] ||
    fail "verified test recovery did not clean its transaction"
ok "test timeout fails closed until rollback is independently verified"

export FAKE_SYSTEMCTL_START_FAIL=1
rollback_rc=0
"$CLI" test openvpn "Test Server" --timeout 2 >/dev/null 2>&1 || rollback_rc=$?
unset FAKE_SYSTEMCTL_START_FAIL
[[ "$rollback_rc" -eq 2 ]] ||
    fail "failed rollback did not return the critical status"
[[ -e "$TMP/state/test.transaction" ]] ||
    fail "failed rollback discarded its recovery transaction"
"$CLI" _recover-stale-test >/dev/null
grep -q '^MODE=normal$' "$TMP/state/active" ||
    fail "recovery after failed rollback did not restore normal state"
ok "rollback failure is critical and recoverable"

touch "$FAKE_LEGACY_ACTIVE"
: >"$FAKE_LEGACY_LOG"
export FAKE_SYSTEMCTL_START_FAIL=1
if "$CLI" connect openvpn "Test Server" >/dev/null 2>&1; then
    fail "simulated service start failure returned success"
fi
unset FAKE_SYSTEMCTL_START_FAIL
[[ -e "$FAKE_LEGACY_ACTIVE" ]] ||
    fail "legacy fallback was not restored after connect failure"
[[ "$(paste -sd, "$FAKE_LEGACY_LOG")" == "stop,start" ]] ||
    fail "legacy fallback stop/start order is wrong"
grep -q '^DESIRED=down$' "$TMP/state/active" ||
    fail "failed connect left watchdog armed over legacy fallback"
rm -f "$FAKE_LEGACY_ACTIVE"
"$CLI" connect openvpn "Test Server" >/dev/null
ok "failed connect restores legacy fallback"

(exec -a "/opt/adguardvpn_cli/adguardvpn-cli connect" sleep 30) &
fake_adguard_transport_pid=$!
printf '%s\n' "$fake_adguard_transport_pid" >"$VPNCTL_ADGUARD_PID_FILE"
touch "$FAKE_ADGUARD_ACTIVE"
"$CLI" test openvpn "Test Server" --timeout 2 --keep >/dev/null
grep -q '^MODE=normal$' "$TMP/state/active" || fail "--keep did not finalize normal state"
grep -q '^DESIRED=up$' "$TMP/state/active" || fail "--keep did not retain connection"
[[ ! -e "$FAKE_ADGUARD_ACTIVE" ]] ||
    fail "--keep restored conflicting AdGuard fallback over managed VPN"
wait "$fake_adguard_transport_pid" 2>/dev/null || true
rm -f "$VPNCTL_ADGUARD_PID_FILE"
ok "test keep leaves external fallback stopped"

"$CLI" emergency --protocol openvpn --timeout 2 >/dev/null
grep -q '^PROTOCOL=openvpn$' "$TMP/state/active" || fail "emergency did not keep working protocol"
grep -q '^MODE=normal$' "$TMP/state/active" || fail "emergency did not finalize state"
ok "emergency keeps first available tunnel"

cp "$TMP/state/active" "$TMP/emergency-before"
export FAKE_CURL_FAIL=1
if "$CLI" emergency --protocol openvpn --timeout 2 >/dev/null 2>&1; then
    fail "unavailable emergency profiles returned success"
fi
unset FAKE_CURL_FAIL
cmp -s "$TMP/emergency-before" "$TMP/state/active" ||
    fail "failed emergency did not restore previous state"
[[ -e "$TMP/state/test.transaction" &&
   -e "$TMP/state/transition-recovery-required.json" ]] ||
    fail "unverified emergency rollback did not preserve recoverable state"
"$CLI" _recover-stale-test >/dev/null
[[ ! -e "$TMP/state/transition-recovery-required.json" ]] ||
    fail "verified emergency recovery did not clear transition recovery state"
ok "emergency failure rolls back"

if "$CLI" connect wireguard Unsafe >/dev/null 2>&1; then
    fail "unsafe WireGuard hook was accepted"
fi
ok "unsafe hooks are rejected"

: >"$TMP/systemctl.log"
FAKE_DEFAULT_IPV4=198.51.100.9 "$CLI" _health-check >/dev/null
if grep -q 'restart vpnctl.service' "$TMP/systemctl.log"; then
    fail "auto health policy forced full-tunnel recovery for a split profile"
fi

: >"$TMP/systemctl.log"
FAKE_DEFAULT_IPV4=198.51.100.9 VPNCTL_HEALTH_REQUIRE_DEFAULT_EGRESS=yes \
    "$CLI" _health-check >/dev/null
FAKE_DEFAULT_IPV4=198.51.100.9 VPNCTL_HEALTH_REQUIRE_DEFAULT_EGRESS=yes \
    "$CLI" _health-check >/dev/null
grep -q '^restart vpnctl.service$' "$TMP/systemctl.log" ||
    fail "strict health policy missed default traffic bypassing the VPN"
ok "watchdog can enforce full-tunnel default egress without breaking split-tunnel auto mode"

cat >"$TMP/config/wireguard/Full.conf" <<'EOF'
[Interface]
PrivateKey = test
[Peer]
PublicKey = test
AllowedIPs=10.0.0.0/8,0.0.0.0/0
Endpoint=192.0.2.21:51820
EOF
chmod 600 "$TMP/config/wireguard/Full.conf"
"$CLI" connect wireguard Full >/dev/null
: >"$TMP/systemctl.log"
FAKE_DEFAULT_IPV4=198.51.100.9 "$CLI" _health-check >/dev/null
FAKE_DEFAULT_IPV4=198.51.100.9 "$CLI" _health-check >/dev/null
grep -q '^restart vpnctl.service$' "$TMP/systemctl.log" ||
    fail "auto health policy missed compact WireGuard full-tunnel AllowedIPs"
"$CLI" connect openvpn "Test Server" >/dev/null
rm -f -- "$TMP/config/wireguard/Full.conf"
ok "watchdog parses compact WireGuard full-tunnel routes"

: >"$TMP/systemctl.log"
rm -f -- "$VPNCTL_RUN_DIR/health.failures"
FAKE_OPENVPN_INTERFACE_MISSING=1 FAKE_SYSTEMCTL_ACTIVE_AGE_SECONDS=30 \
    "$CLI" _health-check >/dev/null 2>&1
[[ ! -e "$VPNCTL_RUN_DIR/health.failures" ]] ||
    fail "watchdog counted a missing OpenVPN interface during startup grace"
if grep -Eq '^(start|restart) vpnctl\.service$' "$TMP/systemctl.log"; then
    fail "watchdog restarted OpenVPN during monotonic startup grace"
fi
[[ "$(grep -c '^show --property=ActiveEnterTimestampMonotonic vpnctl.service$' \
    "$TMP/systemctl.log")" == 1 ]] ||
    fail "watchdog did not obtain the systemd activation timestamp in one bounded query"
FAKE_OPENVPN_INTERFACE_MISSING=1 FAKE_SYSTEMCTL_ACTIVE_AGE_SECONDS=61 \
    "$CLI" _health-check >/dev/null 2>&1
[[ "$(<"$VPNCTL_RUN_DIR/health.failures")" == 1 ]] ||
    fail "watchdog did not resume bounded failure counting after startup grace"
"$CLI" _health-check >/dev/null
[[ ! -e "$VPNCTL_RUN_DIR/health.failures" ]] ||
    fail "successful health tick did not reset the startup failure counter"
FAKE_OPENVPN_INTERFACE_MISSING=1 FAKE_SYSTEMCTL_ACTIVE_AGE_SECONDS=30 \
    FAKE_PYTHON_MONOTONIC_FAIL=1 \
    "$CLI" _health-check >/dev/null 2>&1
[[ "$(<"$VPNCTL_RUN_DIR/health.failures")" == 1 ]] ||
    fail "watchdog granted startup grace without a trustworthy CLOCK_MONOTONIC source"
"$CLI" _health-check >/dev/null
[[ ! -e "$VPNCTL_RUN_DIR/health.failures" ]] ||
    fail "successful health tick did not reset the monotonic-helper failure counter"
FAKE_CURL_FAIL=1 FAKE_SYSTEMCTL_ACTIVE_AGE_SECONDS=30 \
    "$CLI" _health-check >/dev/null 2>&1
[[ ! -e "$VPNCTL_RUN_DIR/health.failures" ]] ||
    fail "watchdog counted OpenVPN control-up/data-down during startup grace"
FAKE_CURL_FAIL=1 FAKE_SYSTEMCTL_ACTIVE_AGE_SECONDS=61 \
    "$CLI" _health-check >/dev/null 2>&1
[[ "$(<"$VPNCTL_RUN_DIR/health.failures")" == 1 ]] ||
    fail "watchdog did not count OpenVPN data-plane failure after startup grace"
"$CLI" _health-check >/dev/null
[[ ! -e "$VPNCTL_RUN_DIR/health.failures" ]] ||
    fail "successful health tick did not reset the data-plane startup counter"
ok "watchdog uses suspend-safe bounded monotonic grace and fails closed without its clock"

: >"$TMP/systemctl.log"
FAKE_CURL_FAIL=1 "$CLI" _health-check >/dev/null 2>&1
health_recovery_failure_output="$(
    FAKE_CURL_FAIL=1 "$CLI" _health-check 2>&1
)"
[[ "$(<"$VPNCTL_RUN_DIR/health.failures")" == 2 ]] ||
    fail "watchdog cleared the failure counter before threshold recovery"
[[ "$(grep -c '^restart vpnctl.service$' "$TMP/systemctl.log")" == 1 ]] ||
    fail "watchdog did not restart exactly once at the failure threshold"
grep -q 'VPN перезапущен, но защищённый egress не подтверждён' \
    <<<"$health_recovery_failure_output" ||
    fail "watchdog misclassified post-restart egress failure"
if grep -q 'systemd отклонил команду' <<<"$health_recovery_failure_output"; then
    fail "watchdog reported a successful systemd restart as rejected"
fi
if grep -q '^reset-failed .*vpnctl.service' "$TMP/systemctl.log"; then
    fail "automatic watchdog recovery reset the systemd start limiter"
fi
health_paused_output="$(
    FAKE_CURL_FAIL=1 "$CLI" _health-check 2>&1
)"
[[ "$(<"$VPNCTL_RUN_DIR/health.failures")" == 3 ]] ||
    fail "watchdog did not saturate at threshold plus one"
health_saturated_output="$(
    FAKE_CURL_FAIL=1 "$CLI" _health-check 2>&1
)"
[[ "$(<"$VPNCTL_RUN_DIR/health.failures")" == 3 ]] ||
    fail "watchdog failure counter exceeded threshold plus one"
[[ "$(grep -c '^restart vpnctl.service$' "$TMP/systemctl.log")" == 1 ]] ||
    fail "watchdog repeated same-profile recovery after the threshold"
paused_warning_count="$(
    printf '%s\n%s\n' "$health_paused_output" "$health_saturated_output" |
        grep -c 'Автовосстановление приостановлено' || true
)"
[[ "$paused_warning_count" == 1 ]] ||
    fail "watchdog did not emit exactly one recovery-paused warning"
if grep -q '^reset-failed .*vpnctl.service' "$TMP/systemctl.log"; then
    fail "multi-cycle automatic recovery reset the systemd start limiter"
fi
"$CLI" _health-check >/dev/null
[[ ! -e "$VPNCTL_RUN_DIR/health.failures" ]] ||
    fail "successful data-plane check did not reset the saturated counter"
printf '3\n' >"$VPNCTL_RUN_DIR/health.failures"
: >"$TMP/systemctl.log"
"$CLI" reconnect >/dev/null
[[ ! -e "$VPNCTL_RUN_DIR/health.failures" ]] ||
    fail "explicit reconnect did not reset the health recovery counter"
grep -q '^reset-failed vpnctl.service$' "$TMP/systemctl.log" ||
    fail "explicit reconnect did not retain its manual start-limit reset"
ok "watchdog caps automatic recovery and explicit recovery resets its counter"

: >"$TMP/systemctl.log"
health_active_file="$TMP/health-systemctl.active"
rm -f -- "$health_active_file"
FAKE_SYSTEMCTL_INACTIVE=1 FAKE_SYSTEMCTL_ACTIVATE_FILE="$health_active_file" \
    "$CLI" _health-check >/dev/null
grep -q '^start vpnctl.service$' "$TMP/systemctl.log" ||
    fail "watchdog did not immediately start an inactive desired service"
ok "watchdog immediately restores an inactive desired service"

: >"$TMP/systemctl.log"
touch "$VPNCTL_STATE_DIR/api-recovery-required.json"
FAKE_SYSTEMCTL_INACTIVE=1 "$CLI" _health-check >/dev/null 2>&1 || true
rm -f -- "$VPNCTL_STATE_DIR/api-recovery-required.json"
if grep -Eq '^(start|restart) vpnctl\.service$' "$TMP/systemctl.log"; then
    fail "health remediation ignored the API recovery marker"
fi
ok "health remediation fails closed behind the recovery marker"

: >"$TMP/systemctl.log"
health_systemctl_delay_marker="$TMP/health-systemctl-delay-once"
health_systemctl_pid_file="$TMP/health-systemctl-delay.pid"
rm -f -- "$health_active_file"
FAKE_SYSTEMCTL_INACTIVE=1 \
FAKE_SYSTEMCTL_ACTIVATE_FILE="$health_active_file" \
FAKE_SYSTEMCTL_DELAY_ACTION="start vpnctl.service" \
FAKE_SYSTEMCTL_DELAY_ONCE_FILE="$health_systemctl_delay_marker" \
FAKE_SYSTEMCTL_DELAY_PID_FILE="$health_systemctl_pid_file" \
FAKE_SYSTEMCTL_DELAY_ONCE_SECONDS=2 \
    "$CLI" _health-check >/dev/null 2>&1 &
health_check_pid=$!
for _ in {1..50}; do
    [[ -s "$health_systemctl_pid_file" ]] && break
    sleep 0.05
done
[[ -s "$health_systemctl_pid_file" ]] || {
    wait "$health_check_pid" || true
    fail "health remediation did not reach the delayed systemd job"
}
exec 6>"$TMP/run/.mutation.lock"
if flock -n 6; then
    flock -u 6
    exec 6>&-
    wait "$health_check_pid" || true
    fail "health remediation released the shared lock before systemd completed"
fi
exec 6>&-
wait "$health_check_pid" || fail "synchronous health remediation failed"
grep -q '^start vpnctl.service$' "$TMP/systemctl.log" ||
    fail "health remediation did not wait for the terminal systemd start result"
if grep -q -- '--no-block' "$TMP/systemctl.log"; then
    fail "health remediation still uses asynchronous systemd jobs"
fi
ok "health remediation retains the shared lock through terminal systemd result"

"$CLI" autostart on >/dev/null
"$CLI" autostart off >/dev/null
grep -q '^disable vpnctl.service$' "$TMP/systemctl.log" || fail "service autostart was not disabled"
grep -q '^disable vpnctl-health.timer$' "$TMP/systemctl.log" || fail "timer autostart was not disabled"
ok "autostart semantics"

: >"$TMP/systemctl.log"
"$CLI" monitor on >/dev/null
"$CLI" monitor off >/dev/null
grep -q '^enable --now vpnctl-health.timer$' "$TMP/systemctl.log" ||
    fail "Desktop monitor control did not enable the health timer"
grep -q '^enable vpnctl-test-recovery.service$' "$TMP/systemctl.log" ||
    fail "Desktop monitor control did not enable boot recovery"
grep -q '^enable mazzy-vpn-api-recovery.service$' "$TMP/systemctl.log" ||
    fail "Desktop monitor control did not enable API action boot recovery"
grep -q '^disable --now vpnctl-health.timer$' "$TMP/systemctl.log" ||
    fail "Desktop monitor control did not stop the health timer"
ok "independent health monitor control"

: >"$TMP/systemctl.log"
exec 6>"$TMP/run/.mutation.lock"
flock 6
if "$CLI" monitor on >/dev/null 2>&1; then
    fail "monitor bypassed the shared mutation lock"
fi
profile_checksum_before="$(sha256sum "$TMP/config/openvpn/Test Server.ovpn")"
if "$CLI" import openvpn "$TMP/config/openvpn/Test Server.ovpn" --force \
    >/dev/null 2>&1; then
    fail "profile import bypassed the shared mutation lock"
fi
[[ "$(sha256sum "$TMP/config/openvpn/Test Server.ovpn")" == \
   "$profile_checksum_before" ]] ||
    fail "blocked profile import changed the active fixture"
flock -u 6
exec 6>&-
if grep -Eq '^(enable|disable|start|stop|restart|kill|daemon-reload)([[:space:]]|$)' \
    "$TMP/systemctl.log"; then
    fail "blocked monitor mutated systemd policy"
fi
if VPNCTL_MUTATION_LOCK_FD=99 "$CLI" autostart off >/dev/null 2>&1; then
    fail "autostart trusted an invalid inherited mutation lock"
fi
if grep -Eq '^(enable|disable|start|stop|restart|kill|daemon-reload)([[:space:]]|$)' \
    "$TMP/systemctl.log"; then
    fail "invalid inherited mutation lock reached mutating systemctl"
fi
ok "service policy mutations fail closed behind the shared lock"

logs_output="$("$CLI" logs --lines 250)"
grep -q 'fake journal: -u vpnctl.service --no-pager -n 250' <<<"$logs_output" ||
    fail "bounded Desktop log retrieval used unexpected journal arguments"
ok "bounded service log retrieval"

"$CLI" disconnect >/dev/null
grep -q '^DESIRED=down$' "$TMP/state/active" || fail "disconnect state missing"
ok "disconnect"

stage="$TMP/stage"
"$ROOT/install.sh" --destdir "$stage" --no-deps --lang de >/dev/null
[[ -x "$stage/usr/local/bin/mazzy-vpn" ]] || fail "Mazzy VPN binary was not staged"
[[ "$(readlink "$stage/usr/local/bin/vpnctl")" == "mazzy-vpn" ]] ||
    fail "vpnctl alias was not staged"
[[ "$(readlink "$stage/usr/local/bin/mazzyvpn")" == "mazzy-vpn" ]] ||
    fail "mazzyvpn alias was not staged"
[[ -f "$stage/usr/local/lib/mazzy-vpn/README.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/README.en.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/README.de.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/README.zh.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/README.ja.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/README.ko.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/DESKTOP.en.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/DESKTOP.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/DESKTOP_ROADMAP.en.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/DESKTOP_ROADMAP.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/PLATFORM_ROADMAP.en.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/PLATFORM_ROADMAP.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/PROTOCOL_ORCHESTRATION.en.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/PROTOCOL_ORCHESTRATION.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/TARGET_ARCHITECTURE_2026-08-02.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/RESEARCH_AGENT_REMOTE_CONTROL_2026-08-02.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/R0_MUTATION_SINGLE_FLIGHT.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/FEATURE_PARITY.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/capabilities.json" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/API_CONTRACT.en.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/API_CONTRACT.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/PROJECT_STATUS.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/AUDIT_2026-07-28.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/AUDIT_2026-08-01.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/AUDIT_2026-08-01_PROTOCOLS.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/AUDIT_2026-08-02_RUNTIME_AND_AGENTS.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/ARCHITECTURE.en.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/ARCHITECTURE.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/api/v1/manifest.json" &&
   -f "$stage/usr/local/lib/mazzy-vpn/api/v1/schema.json" &&
   -f "$stage/usr/local/lib/mazzy-vpn/protocols/v1/registry.json" &&
   -f "$stage/usr/local/lib/mazzy-vpn/protocols/v1/schema.json" &&
   -f "$stage/usr/local/lib/mazzy-vpn/protocols/v1/managed-profile.schema.json" &&
   -x "$stage/usr/local/lib/mazzy-vpn/runtime/mazzy-sing-box-adapter" &&
   -f "$stage/usr/local/lib/mazzy-vpn/runtime/v1/adapter-registry.json" &&
   -f "$stage/usr/local/lib/mazzy-vpn/runtime/v1/schema.json" &&
   -f "$stage/usr/local/lib/mazzy-vpn/LICENSE" &&
   -f "$stage/usr/local/lib/mazzy-vpn/AUTHORS.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/PRIVACY.md" ]] ||
    fail "six-language and architecture documentation was not staged"
cmp -s "$ROOT/api/v1/manifest.json" \
    "$stage/usr/local/lib/mazzy-vpn/api/v1/manifest.json" ||
    fail "staged API manifest differs from the source contract"
cmp -s "$ROOT/api/v1/schema.json" \
    "$stage/usr/local/lib/mazzy-vpn/api/v1/schema.json" ||
    fail "staged API schema differs from the source contract"
cmp -s "$ROOT/protocols/v1/registry.json" \
    "$stage/usr/local/lib/mazzy-vpn/protocols/v1/registry.json" ||
    fail "staged protocol registry differs from the source contract"
cmp -s "$ROOT/protocols/v1/managed-profile.schema.json" \
    "$stage/usr/local/lib/mazzy-vpn/protocols/v1/managed-profile.schema.json" ||
    fail "staged managed profile schema differs from the source contract"
cmp -s "$ROOT/runtime/mazzy-sing-box-adapter" \
    "$stage/usr/local/lib/mazzy-vpn/runtime/mazzy-sing-box-adapter" ||
    fail "staged sing-box adapter differs from source"
cmp -s "$ROOT/runtime/v1/adapter-registry.json" \
    "$stage/usr/local/lib/mazzy-vpn/runtime/v1/adapter-registry.json" ||
    fail "staged runtime adapter registry differs from source"
"$stage/usr/local/bin/mazzy-vpn" api-info --json |
    cmp -s - "$ROOT/api/v1/manifest.json" ||
    fail "staged CLI does not expose the installed API manifest"
[[ -f "$stage/usr/local/share/bash-completion/completions/mazzy-vpn" ]] ||
    fail "Mazzy VPN completion was not staged"
for protocol_dir in amneziawg wireguard openvpn l2tp; do
    [[ -d "$stage/etc/vpnctl/profiles/$protocol_dir" ]] ||
        fail "staged profile structure is missing $protocol_dir"
done
[[ "$(<"$stage/etc/vpnctl/locale")" == "de" ]] ||
    fail "installer did not persist selected language"
[[ -f "$stage/etc/systemd/system/vpnctl-health.timer" ]] || fail "timer not staged"
grep -q '^Restart=always$' "$stage/etc/systemd/system/vpnctl.service" ||
    fail "service does not restart after a clean unexpected exit"
grep -q '^RestartPreventExitStatus=77$' \
    "$stage/etc/systemd/system/vpnctl.service" ||
    fail "service retries permanent OpenVPN authentication failures"
grep -q '^StartLimitIntervalSec=600$' \
    "$stage/etc/systemd/system/vpnctl.service" ||
    fail "service start-limit interval is not the tested ten-minute window"
grep -q '^StartLimitBurst=5$' "$stage/etc/systemd/system/vpnctl.service" ||
    fail "service start-limit burst is not bounded to five attempts"
grep -q '^OnUnitActiveSec=60s$' "$stage/etc/systemd/system/vpnctl-health.timer" ||
    fail "health timer interval is not the expected 60 seconds"
[[ -f "$stage/etc/systemd/system/vpnctl-test-recovery.service" ]] ||
    fail "test recovery unit not staged"
[[ -f "$stage/etc/systemd/system/mazzy-vpn-api-recovery.service" ]] ||
    fail "API action recovery unit not staged"
grep -q '^DefaultDependencies=no$' \
    "$stage/etc/systemd/system/mazzy-vpn-api-recovery.service" ||
    fail "API recovery retains default basic.target ordering and can cycle with sockets.target"
grep -q '^Requires=local-fs.target$' \
    "$stage/etc/systemd/system/mazzy-vpn-api-recovery.service" ||
    fail "early API recovery does not require its filesystem dependency"
grep -q '^Before=shutdown.target vpnctl-test-recovery.service vpnctl.service vpnctl-health.service mazzy-vpn-api.socket$' \
    "$stage/etc/systemd/system/mazzy-vpn-api-recovery.service" ||
    fail "API recovery is not ordered before every mutating/socket-activated consumer"
grep -q '^Requires=mazzy-vpn-api-recovery.service$' \
    "$stage/etc/systemd/system/vpnctl-test-recovery.service" ||
    fail "test recovery does not require successful API recovery"
grep -q '^After=local-fs.target mazzy-vpn-api-recovery.service$' \
    "$stage/etc/systemd/system/vpnctl-test-recovery.service" ||
    fail "test recovery is not serialized after API recovery"
grep -q '^TimeoutStartSec=60s$' \
    "$stage/etc/systemd/system/vpnctl-test-recovery.service" ||
    fail "test recovery has no bounded systemd startup budget"
grep -q '^ExecStart=/usr/local/bin/mazzy-vpn _api-recover-interrupted-actions$' \
    "$stage/etc/systemd/system/mazzy-vpn-api-recovery.service" ||
    fail "API recovery unit does not use the root-only recovery entrypoint"
grep -Fxq 'RemainAfterExit=yes' \
    "$stage/etc/systemd/system/mazzy-vpn-api-recovery.service" ||
    fail "staged API recovery unit does not remain active after boot recovery"
grep -q '^ProtectSystem=strict$' \
    "$stage/etc/systemd/system/mazzy-vpn-api-recovery.service" ||
    fail "API recovery unit does not use strict filesystem protection"
grep -q '^CapabilityBoundingSet=CAP_NET_ADMIN$' \
    "$stage/etc/systemd/system/mazzy-vpn-api-recovery.service" ||
    fail "API recovery unit cannot restore the nftables transition guard"
[[ -f "$stage/etc/systemd/system/mazzy-vpn-api.socket" &&
   -f "$stage/etc/systemd/system/mazzy-vpn-api@.service" ]] ||
    fail "local API systemd units were not staged"
[[ -f "$stage/usr/lib/tmpfiles.d/mazzy-vpn.conf" ]] ||
    fail "local API runtime-directory policy was not staged"
grep -q '^d /run/mazzy-vpn 0750 root mazzy-vpn -$' \
    "$stage/usr/lib/tmpfiles.d/mazzy-vpn.conf" ||
    fail "local API runtime-directory policy is not group-restricted"
grep -q '^SocketMode=0660$' "$stage/etc/systemd/system/mazzy-vpn-api.socket" ||
    fail "local API socket is not protected by mode 0660"
grep -q '^SocketGroup=mazzy-vpn$' "$stage/etc/systemd/system/mazzy-vpn-api.socket" ||
    fail "local API socket is not restricted to the mazzy-vpn group"
grep -q '^DirectoryMode=0750$' "$stage/etc/systemd/system/mazzy-vpn-api.socket" ||
    fail "local API socket directory is not group-restricted"
grep -q '^Requires=mazzy-vpn-api-recovery.service$' \
    "$stage/etc/systemd/system/mazzy-vpn-api.socket" ||
    fail "local API socket can start without successful boot recovery"
grep -q '^After=mazzy-vpn-api-recovery.service$' \
    "$stage/etc/systemd/system/mazzy-vpn-api.socket" ||
    fail "local API socket is not ordered after boot recovery"
grep -q '^Requires=mazzy-vpn-api-recovery.service$' \
    "$stage/etc/systemd/system/vpnctl-health.service" ||
    fail "health remediation can run after boot recovery fails"
grep -q '^After=network-online.target mazzy-vpn-api-recovery.service vpnctl.service$' \
    "$stage/etc/systemd/system/vpnctl-health.service" ||
    fail "health remediation is not serialized after boot recovery"
grep -q '^NoNewPrivileges=yes$' \
    "$stage/etc/systemd/system/mazzy-vpn-api@.service" ||
    fail "local API request service is missing process hardening"
grep -q 'ExecStart=/usr/local/bin/mazzy-vpn' \
    "$stage/etc/systemd/system/vpnctl.service" ||
    fail "staged service does not use Mazzy VPN command"
ok "branded staged installation and aliases"

installer_order_output="$(
    VPNCTL_AMNEZIA_PPA_AVAILABLE=0 \
        "$ROOT/install.sh" --dry-run --yes --skip-tests --skip-checks
)"
installer_dependency_line="$(
    grep -En -m1 \
        '^\+ (env DEBIAN_FRONTEND=noninteractive apt-get install |dnf install |pacman -S |zypper --non-interactive install )' \
        <<<"$installer_order_output" |
        cut -d: -f1
)"
installer_files_line="$(
    grep -n -m1 '^+ install -d -m 755 /usr/local/bin ' \
        <<<"$installer_order_output" |
        cut -d: -f1
)"
[[ "$installer_dependency_line" =~ ^[0-9]+$ &&
   "$installer_files_line" =~ ^[0-9]+$ &&
   "$installer_dependency_line" -lt "$installer_files_line" ]] ||
    fail "installer changes root-owned product files before dependencies succeed"
python3 - "$ROOT/desktop/src-tauri/tauri.conf.json" <<'PY' ||
import json
import sys

with open(sys.argv[1], encoding="utf-8") as stream:
    config = json.load(stream)
linux = config["bundle"]["linux"]
deb = linux["deb"]
rpm = linux["rpm"]
for dependency in (
    "bash", "diffutils", "findutils", "grep", "iproute2", "jq", "pkexec",
    "nftables", "procps", "python3", "sed", "socat", "systemd",
):
    assert dependency in deb["depends"]
for dependency in (
    "netcat-openbsd", "network-manager-l2tp", "openvpn", "wireguard-tools",
):
    assert dependency in deb["recommends"]
for dependency in (
    "bash", "diffutils", "findutils", "gawk", "grep", "iproute", "jq",
    "nftables", "polkit", "procps-ng", "python3", "sed", "socat", "systemd",
):
    assert dependency in rpm["depends"]
for dependency in ("NetworkManager-l2tp", "nmap-ncat", "openvpn", "wireguard-tools"):
    assert dependency in rpm["recommends"]

expected_scripts = {
    "postInstallScript": "../../packaging/linux/post-install.sh",
    "preRemoveScript": "../../packaging/linux/pre-remove.sh",
    "postRemoveScript": "../../packaging/linux/post-remove.sh",
}
for key, value in expected_scripts.items():
    assert deb[key] == value
    assert rpm[key] == value

assert deb["files"] == rpm["files"]
files = deb["files"]
for destination in (
    "/usr/bin/mazzy-vpn",
    "/usr/bin/vpnctl",
    "/usr/lib/mazzy-vpn/api",
    "/usr/lib/mazzy-vpn/desktop/src-tauri/tauri.conf.json",
    "/usr/lib/mazzy-vpn/desktop/ui/app.css",
    "/usr/lib/mazzy-vpn/desktop/ui/app.js",
    "/usr/lib/mazzy-vpn/packaging",
    "/usr/lib/mazzy-vpn/protocols",
    "/usr/lib/mazzy-vpn/wiki",
    "/usr/lib/systemd/system/mazzy-vpn-api.socket",
    "/usr/lib/systemd/system/mazzy-vpn-api.socket.d",
    "/usr/lib/systemd/system/mazzy-vpn-api@.service",
    "/usr/lib/systemd/system/mazzy-vpn-api@.service.d",
    "/usr/lib/systemd/system/vpnctl.service",
    "/usr/lib/systemd/system/vpnctl.service.d",
    "/usr/lib/systemd/system/vpnctl-health.timer.d",
    "/usr/lib/tmpfiles.d/mazzy-vpn.conf",
):
    assert destination in files
assert not any(path.startswith("/usr/local/") for path in files)
assert not any(path.startswith("/etc/vpnctl") for path in files)
PY
    fail "Desktop packages do not declare their privileged bootstrap dependency"
for package_script in \
    "$ROOT/packaging/linux/post-install.sh" \
    "$ROOT/packaging/linux/pre-remove.sh" \
    "$ROOT/packaging/linux/post-remove.sh"; do
    bash -n "$package_script" ||
        fail "invalid package lifecycle script: $package_script"
    if grep -Eq \
        '(^|[[:space:]])(apt-get|dnf|pacman|zypper|curl|wget)([[:space:]]|$)' \
        "$package_script"; then
        fail "package lifecycle script performs network/dependency installation"
    fi
    if grep -Eq 'rm[[:space:]].*/etc/vpnctl' "$package_script"; then
        fail "package lifecycle script can remove user VPN state"
    fi
done

legacy_migration_root="$TMP/legacy-package-migration"
mkdir -p "$legacy_migration_root/usr/local/bin" \
    "$legacy_migration_root/usr/bin" \
    "$legacy_migration_root/var/lib/vpnctl"
install -m 755 "$ROOT/mazzy-vpn" \
    "$legacy_migration_root/usr/bin/mazzy-vpn"
install -m 744 "$ROOT/mazzy-vpn" \
    "$legacy_migration_root/usr/local/bin/mazzy-vpn"
install -m 755 "$ROOT/mazzy-vpn" \
    "$legacy_migration_root/usr/local/bin/vpnctl"
install -m 700 "$ROOT/mazzy-vpn" \
    "$legacy_migration_root/usr/local/bin/mazzyvpn"
declare -A legacy_modes=([mazzy-vpn]=744 [vpnctl]=755 [mazzyvpn]=700)
legacy_checksum="$(sha256sum \
    "$legacy_migration_root/usr/local/bin/mazzy-vpn" | awk '{print $1}')"
"$ROOT/packaging/linux/post-install.sh" --test-migrate \
    "$legacy_migration_root"
for legacy_name in mazzy-vpn vpnctl mazzyvpn; do
    [[ "$(readlink "$legacy_migration_root/usr/local/bin/$legacy_name")" == \
        /usr/bin/mazzy-vpn ]] ||
        fail "package migration did not redirect legacy $legacy_name"
    [[ -f "$legacy_migration_root/var/lib/vpnctl/package-migration/$legacy_name.pre-package" ]] ||
        fail "package migration did not preserve legacy $legacy_name"
done
"$ROOT/packaging/linux/post-remove.sh" --test-restore \
    "$legacy_migration_root"
for legacy_name in mazzy-vpn vpnctl mazzyvpn; do
    [[ ! -L "$legacy_migration_root/usr/local/bin/$legacy_name" ]] ||
        fail "package removal left a legacy $legacy_name symlink"
    [[ "$(sha256sum "$legacy_migration_root/usr/local/bin/$legacy_name" | awk '{print $1}')" == \
        "$legacy_checksum" ]] ||
        fail "package removal did not restore legacy $legacy_name"
    [[ "$(stat -c %a "$legacy_migration_root/usr/local/bin/$legacy_name")" == \
        "${legacy_modes[$legacy_name]}" ]] ||
        fail "package removal did not restore legacy $legacy_name permissions"
done
ok "package lifecycle migrates and restores trusted legacy CLI copies"

for package_dropin in \
    "$ROOT/packaging/linux/systemd/mazzy-vpn-api.socket.d/10-package-docs.conf" \
    "$ROOT/packaging/linux/systemd/mazzy-vpn-api-recovery.service.d/10-package-exec.conf" \
    "$ROOT/packaging/linux/systemd/mazzy-vpn-api@.service.d/10-package-exec.conf" \
    "$ROOT/packaging/linux/systemd/vpnctl-health.service.d/10-package-exec.conf" \
    "$ROOT/packaging/linux/systemd/vpnctl-health.timer.d/10-package-interval.conf" \
    "$ROOT/packaging/linux/systemd/vpnctl-test-recovery.service.d/10-package-exec.conf" \
    "$ROOT/packaging/linux/systemd/vpnctl.service.d/10-package-exec.conf"; do
    if [[ "$package_dropin" == *".service.d/"* ]]; then
        grep -q '^ExecStart=/usr/bin/mazzy-vpn ' "$package_dropin" ||
            fail "package systemd override does not use the package-managed engine"
    fi
    if [[ "$package_dropin" == *"mazzy-vpn-api"* ]]; then
        grep -q '^Documentation=file:/usr/lib/mazzy-vpn/' "$package_dropin" ||
            fail "package local API documentation path is not package-managed"
    fi
done
grep -q '^DefaultDependencies=no$' \
    "$ROOT/packaging/linux/systemd/mazzy-vpn-api-recovery.service.d/10-package-exec.conf" ||
    fail "package recovery drop-in can reintroduce the sockets.target boot cycle"
grep -q '^Requires=mazzy-vpn-api-recovery.service$' \
    "$ROOT/packaging/linux/systemd/mazzy-vpn-api.socket.d/10-package-docs.conf" ||
    fail "package socket drop-in does not harden a legacy /etc unit"
grep -q '^After=mazzy-vpn-api-recovery.service$' \
    "$ROOT/packaging/linux/systemd/mazzy-vpn-api.socket.d/10-package-docs.conf" ||
    fail "package socket drop-in does not serialize a legacy unit after recovery"
grep -q '^Requires=mazzy-vpn-api-recovery.service$' \
    "$ROOT/packaging/linux/systemd/vpnctl.service.d/10-package-exec.conf" ||
    fail "package runtime drop-in does not harden legacy systemd overrides"
grep -q '^StartLimitIntervalSec=600$' \
    "$ROOT/packaging/linux/systemd/vpnctl.service.d/10-package-exec.conf" ||
    fail "package runtime drop-in does not bound legacy service retries"
grep -q '^RestartPreventExitStatus=77$' \
    "$ROOT/packaging/linux/systemd/vpnctl.service.d/10-package-exec.conf" ||
    fail "package runtime drop-in loses permanent authentication failure handling"
grep -q '^Requires=mazzy-vpn-api-recovery.service$' \
    "$ROOT/packaging/linux/systemd/vpnctl-test-recovery.service.d/10-package-exec.conf" ||
    fail "package test recovery drop-in does not harden a legacy unit"
grep -q '^TimeoutStartSec=60s$' \
    "$ROOT/packaging/linux/systemd/vpnctl-test-recovery.service.d/10-package-exec.conf" ||
    fail "package test recovery drop-in has no bounded startup budget"
grep -q '^Requires=mazzy-vpn-api-recovery.service$' \
    "$ROOT/packaging/linux/systemd/vpnctl-health.service.d/10-package-exec.conf" ||
    fail "package health drop-in does not propagate recovery failure"
grep -q '^After=mazzy-vpn-api-recovery.service$' \
    "$ROOT/packaging/linux/systemd/vpnctl-health.service.d/10-package-exec.conf" ||
    fail "package health drop-in is not ordered after recovery"
[[ "$(grep -c '^OnUnitActiveSec=$' \
    "$ROOT/packaging/linux/systemd/vpnctl-health.timer.d/10-package-interval.conf")" == 1 ]] ||
    fail "package health timer drop-in does not reset the legacy interval"
grep -q '^OnUnitActiveSec=60s$' \
    "$ROOT/packaging/linux/systemd/vpnctl-health.timer.d/10-package-interval.conf" ||
    fail "package health timer drop-in does not enforce the one-minute interval"
grep -q '^RandomizedDelaySec=5s$' \
    "$ROOT/packaging/linux/systemd/vpnctl-health.timer.d/10-package-interval.conf" ||
    fail "package health timer drop-in loses bounded jitter"

effective_units="$TMP/effective-systemd-units"
mkdir -p "$effective_units/etc" "$effective_units/usr"
awk '
    /^Requires=mazzy-vpn-api-recovery\.service$/ { next }
    /^After=mazzy-vpn-api-recovery\.service$/ { next }
    { print }
' "$ROOT/systemd/mazzy-vpn-api.socket" \
    >"$effective_units/etc/mazzy-vpn-api.socket"
awk '
    /^Requires=mazzy-vpn-api-recovery\.service$/ { next }
    /^After=network-online\.target mazzy-vpn-api-recovery\.service vpnctl\.service$/ {
        print "After=network-online.target vpnctl.service"
        next
    }
    { print }
' "$ROOT/systemd/vpnctl-health.service" \
    >"$effective_units/etc/vpnctl-health.service"
sed 's/^OnUnitActiveSec=.*/OnUnitActiveSec=20s/' \
    "$ROOT/systemd/vpnctl-health.timer" \
    >"$effective_units/etc/vpnctl-health.timer"
for effective_unit in \
    mazzy-vpn-api.socket \
    mazzy-vpn-api-recovery.service \
    vpnctl-health.service \
    vpnctl-health.timer \
    vpnctl.service \
    vpnctl-test-recovery.service; do
    cp "$ROOT/systemd/$effective_unit" "$effective_units/usr/$effective_unit"
done
for effective_dropin in \
    mazzy-vpn-api.socket.d/10-package-docs.conf \
    mazzy-vpn-api-recovery.service.d/10-package-exec.conf \
    vpnctl-health.service.d/10-package-exec.conf \
    vpnctl-health.timer.d/10-package-interval.conf \
    vpnctl.service.d/10-package-exec.conf \
    vpnctl-test-recovery.service.d/10-package-exec.conf; do
    mkdir -p "$effective_units/usr/${effective_dropin%/*}"
    cp "$ROOT/packaging/linux/systemd/$effective_dropin" \
        "$effective_units/usr/$effective_dropin"
done
effective_verify_log="$TMP/effective-systemd-verify.log"
if ! SYSTEMD_UNIT_PATH="$effective_units/etc:$effective_units/usr:/usr/lib/systemd/system" \
    systemd-analyze verify \
        mazzy-vpn-api.socket \
        mazzy-vpn-api-recovery.service \
        vpnctl-health.service \
        vpnctl-health.timer \
        vpnctl.service \
        vpnctl-test-recovery.service >"$effective_verify_log" 2>&1; then
    cat "$effective_verify_log" >&2
    fail "legacy /etc units and package drop-ins do not form a valid systemd graph"
fi
ok "package drop-ins form a valid recovery graph over legacy /etc units"

grep -q 'chmodSync(path, 0o755)' \
    "$ROOT/desktop/scripts/build-release.mjs" ||
    fail "release builder does not normalize package executable modes"
grep -q 'chmodSync(path, mode)' \
    "$ROOT/desktop/scripts/build-release.mjs" ||
    fail "release builder does not restore checkout modes after packaging"
if grep -q 'process\.argv' \
    "$ROOT/desktop/scripts/tauri-audited.mjs" \
    "$ROOT/desktop/scripts/build-release.mjs"; then
    fail "release scripts forward user-controlled CLI arguments"
fi
grep -Fq '[join("scripts", "build-release.mjs")]' \
    "$ROOT/desktop/scripts/tauri-audited.mjs" ||
    fail "tag release audit does not invoke the fixed release builder"
if grep -q 'check-linux-packages.sh' \
    "$ROOT/desktop/scripts/tauri-audited.mjs"; then
    fail "signing credentials can reach the Linux package audit subprocess"
fi
ok "dependency bootstrap and package-owned lifecycle are declared safely"

if "$ROOT/install.sh" --destdir "$TMP/invalid-language-stage" --no-deps \
    --lang invalid >/dev/null 2>&1; then
    fail "installer accepted an unsupported language"
fi
installer_language_output="$(
    printf '6\n\n' |
        VPNCTL_INSTALL_FORCE_INTERACTIVE=1 \
        "$ROOT/install.sh" --dry-run --no-deps --skip-tests --skip-checks
)"
grep -q '설치 언어: 한국어' <<<"$installer_language_output" ||
    fail "interactive installer language selection did not choose Korean"
ok "installer language selection"

mkdir -p "$TMP/install-import"
cp "$TMP/import-source/mixed/WG.conf" "$TMP/install-import/InstallerWG.conf"
stage_with_configs="$TMP/stage-with-configs"
"$ROOT/install.sh" --destdir "$stage_with_configs" --no-deps \
    --config-dir "$TMP/install-import" >/dev/null
[[ -f "$stage_with_configs/etc/vpnctl/profiles/wireguard/InstallerWG.conf" ]] ||
    fail "installer --config-dir did not auto-import WireGuard profile"
[[ "$(stat -c %a "$stage_with_configs/etc/vpnctl/profiles/wireguard/InstallerWG.conf")" == "600" ]] ||
    fail "installer --config-dir used unsafe permissions"
ok "installer config-folder auto-detection"

# shellcheck disable=SC1091
source "$ROOT/completions/mazzy-vpn"
COMP_WORDS=(mazzy-vpn se)
COMP_CWORD=1
_mazzy_vpn
printf '%s\n' "${COMPREPLY[@]}" | grep -qx 'self-test' ||
    fail "Mazzy VPN completion does not include self-test"
COMP_WORDS=(mazzy-vpn da)
COMP_CWORD=1
_mazzy_vpn
printf '%s\n' "${COMPREPLY[@]}" | grep -qx 'dashboard' ||
    fail "Mazzy VPN completion does not include dashboard"
COMP_WORDS=(mazzy-vpn api-info --)
COMP_CWORD=2
_mazzy_vpn
printf '%s\n' "${COMPREPLY[@]}" | grep -qx -- '--json' ||
    fail "Mazzy VPN completion does not include api-info --json"
COMP_WORDS=(mazzy-vpn probe all --j)
COMP_CWORD=3
_mazzy_vpn
printf '%s\n' "${COMPREPLY[@]}" | grep -qx -- '--jobs' ||
    fail "Mazzy VPN completion does not include probe --jobs"
COMP_WORDS=(mazzy-vpn verify --s)
COMP_CWORD=2
_mazzy_vpn
printf '%s\n' "${COMPREPLY[@]}" | grep -qx -- '--speed' ||
    fail "Mazzy VPN completion does not include verify --speed"
COMP_WORDS=(mazzy-vpn planner e)
COMP_CWORD=2
_mazzy_vpn
printf '%s\n' "${COMPREPLY[@]}" | grep -qx 'evaluate' ||
    fail "Mazzy VPN completion does not include planner evaluate"
COMP_WORDS=(mazzy-vpn language j)
COMP_CWORD=2
_mazzy_vpn
printf '%s\n' "${COMPREPLY[@]}" | grep -qx 'ja' ||
    fail "Mazzy VPN completion does not include language codes"
ok "Mazzy VPN bash completion"

if grep -Eq '^(Wants|After)=.*vpnctl-test-recovery' "$ROOT/systemd/vpnctl.service"; then
    fail "runtime service pulls boot recovery into a locked test transaction"
fi
grep -q '^Requires=mazzy-vpn-api-recovery.service$' \
    "$ROOT/systemd/vpnctl.service" ||
    fail "runtime service can start after API boot recovery fails"
grep -q '^After=network-online.target NetworkManager.service mazzy-vpn-api-recovery.service$' \
    "$ROOT/systemd/vpnctl.service" ||
    fail "runtime service is not ordered after required API boot recovery"
grep -q '^DefaultDependencies=no$' \
    "$ROOT/systemd/mazzy-vpn-api-recovery.service" ||
    fail "boot API recovery can form a basic.target/sockets.target ordering cycle"
grep -q '^Requires=local-fs.target$' \
    "$ROOT/systemd/mazzy-vpn-api-recovery.service" ||
    fail "boot API recovery lost its explicit filesystem requirement"
grep -q '^Before=shutdown.target vpnctl-test-recovery.service vpnctl.service vpnctl-health.service mazzy-vpn-api.socket$' \
    "$ROOT/systemd/mazzy-vpn-api-recovery.service" ||
    fail "boot API recovery ordering drifted"
grep -Fxq 'RemainAfterExit=yes' \
    "$ROOT/systemd/mazzy-vpn-api-recovery.service" ||
    fail "successful boot API recovery is not retained for the boot"
grep -q '^CapabilityBoundingSet=CAP_NET_ADMIN$' \
    "$ROOT/systemd/mazzy-vpn-api-recovery.service" ||
    fail "boot API recovery cannot restore the nftables transition guard"
grep -q '^Requires=mazzy-vpn-api-recovery.service$' \
    "$ROOT/systemd/vpnctl-test-recovery.service" ||
    fail "test boot recovery no longer requires API boot recovery"
grep -q '^After=local-fs.target mazzy-vpn-api-recovery.service$' \
    "$ROOT/systemd/vpnctl-test-recovery.service" ||
    fail "boot recovery units are no longer ordered"
grep -q '^TimeoutStartSec=60s$' \
    "$ROOT/systemd/vpnctl-test-recovery.service" ||
    fail "test boot recovery timeout budget drifted"
grep -q '^WantedBy=multi-user.target$' \
    "$ROOT/systemd/mazzy-vpn-api-recovery.service" ||
    fail "boot API recovery is not enabled through the boot target"
grep -q '^Requires=mazzy-vpn-api-recovery.service$' \
    "$ROOT/systemd/mazzy-vpn-api.socket" ||
    fail "API socket no longer requires the fail-closed recovery gate"
grep -q '^After=mazzy-vpn-api-recovery.service$' \
    "$ROOT/systemd/mazzy-vpn-api.socket" ||
    fail "API socket is no longer serialized after recovery"
grep -q '^Requires=mazzy-vpn-api-recovery.service$' \
    "$ROOT/systemd/vpnctl-health.service" ||
    fail "health remediation no longer propagates boot recovery failure"
grep -q '^After=network-online.target mazzy-vpn-api-recovery.service vpnctl.service$' \
    "$ROOT/systemd/vpnctl-health.service" ||
    fail "health remediation is no longer serialized after recovery"
: >"$FAKE_SYSTEMCTL_RECOVERY_LOG"
rm -f -- "$FAKE_SYSTEMCTL_RECOVERY_ACTIVE_FILE"
FAKE_SYSTEMCTL_MODEL_RECOVERY_DEPENDENCY=1 \
    systemctl start mazzy-vpn-api-recovery.service
(
    exec 9>"$VPNCTL_RUN_DIR/.mutation.lock"
    flock -x 9
    FAKE_SYSTEMCTL_MODEL_RECOVERY_DEPENDENCY=1 \
        systemctl start vpnctl.service
    FAKE_SYSTEMCTL_MODEL_RECOVERY_DEPENDENCY=1 \
        systemctl restart vpnctl.service
)
[[ "$(wc -l <"$FAKE_SYSTEMCTL_RECOVERY_LOG")" == "1" ]] ||
    fail "runtime VPN service activation re-entered boot API recovery"
ok "runtime start cannot deadlock on boot recovery"

for recovery_consumer in \
    mazzy-vpn-api.socket \
    vpnctl.service \
    vpnctl-health.service \
    vpnctl-test-recovery.service; do
    rm -f -- "$FAKE_SYSTEMCTL_RECOVERY_ACTIVE_FILE"
    : >"$FAKE_SYSTEMCTL_RECOVERY_LOG"
    if FAKE_SYSTEMCTL_MODEL_RECOVERY_DEPENDENCY=1 \
        FAKE_SYSTEMCTL_RECOVERY_FAIL=1 \
        systemctl start "$recovery_consumer"; then
        fail "$recovery_consumer started after required API recovery failed"
    fi
    [[ ! -e "$FAKE_SYSTEMCTL_RECOVERY_ACTIVE_FILE" ]] ||
        fail "$recovery_consumer retained a false successful recovery state"
    [[ "$(wc -l <"$FAKE_SYSTEMCTL_RECOVERY_LOG")" == "1" ]] ||
        fail "$recovery_consumer did not attempt exactly one required recovery"
done
ok "API recovery failure blocks every socket and mutating consumer"

mkdir -p "$TMP/installbin"
for install_command in bash dirname uname id sed grep tr cut head tail stat \
    mkdir cp chmod chown find sort cmp python3 awk; do
    install_command_path="$(
        PATH=/usr/bin:/bin command -v "$install_command" 2>/dev/null
    )" || fail "cannot prepare isolated installer PATH: $install_command"
    ln -s "$install_command_path" "$TMP/installbin/$install_command"
done
fallback_output="$(
    PATH="$TMP/installbin" \
        VPNCTL_AMNEZIA_PPA_AVAILABLE=0 \
        "$ROOT/install.sh" --dry-run --yes --deps-only
)"
grep -q 'amneziawg-go.git' <<<"$fallback_output" ||
    fail "unsupported Ubuntu suite did not select userspace AmneziaWG"
grep -q '61e741780e8465a67a7d7fb6cffe14a8a15d624a' <<<"$fallback_output" ||
    fail "AmneziaWG tools source commit is not verified"
grep -q '9f5d948bc72cc554791cfe0fb91527e4acfb6b79' <<<"$fallback_output" ||
    fail "AmneziaWG Go source commit is not verified"
if grep -q 'add-apt-repository' <<<"$fallback_output"; then
    fail "unsupported Ubuntu suite attempted to add the PPA"
fi
ok "Ubuntu without PPA uses commit-verified userspace fallback"

if ! grep -Eq 'id="notifications-toggle" type="checkbox"[[:space:]]*$' \
        "$ROOT/desktop/ui/index.html" ||
   ! grep -Eq '^[[:space:]]+disabled aria-disabled="true">' \
        "$ROOT/desktop/ui/index.html"; then
    fail "Desktop exposes an active notifications control without a backend"
fi
if grep -q 'mazzy-notifications' "$ROOT/desktop/ui/app.js"; then
    fail "Desktop persists a notifications preference that has no effect"
fi
grep -q 'id="installation-type"' "$ROOT/desktop/ui/index.html" ||
    fail "Desktop does not expose package-managed installation state"
grep -q 'report?.package_managed' "$ROOT/desktop/ui/app.js" ||
    fail "Desktop ignores package-managed installation state"
grep -q 'ensure_runtime_reader_access' "$ROOT/mazzy-vpn" ||
    fail "package repair cannot enroll the invoking user into the local API group"
ok "Desktop package state and unavailable notifications are represented honestly"

grep -q 'id="location-health-button"' "$ROOT/desktop/ui/index.html" ||
    fail "Desktop profile list has no batch location check"
grep -q 'invoke("probe_profiles"' "$ROOT/desktop/ui/app.js" ||
    fail "Desktop batch location check does not use the typed backend"
grep -q 'state.profileHealth.set(entry.profile_id, entry)' \
    "$ROOT/desktop/ui/app.js" ||
    fail "Desktop does not attach probe results to individual profile rows"
grep -q 'backend::probe_profiles' "$ROOT/desktop/src-tauri/src/main.rs" ||
    fail "Desktop does not expose the typed probe command"
grep -q 'id="profile-sort"' "$ROOT/desktop/ui/index.html" ||
    fail "Desktop profile list has no latency/status sorting"
grep -q 'id="connect-fastest-button"' "$ROOT/desktop/ui/index.html" ||
    fail "Desktop cannot connect the fastest reachable location"
grep -q 'invoke("verify_connection"' "$ROOT/desktop/ui/app.js" ||
    fail "Desktop real VPN verification does not use the typed backend"
grep -q 'backend::verify_connection' "$ROOT/desktop/src-tauri/src/main.rs" ||
    fail "Desktop does not expose the typed egress verification command"
[[ "$(grep -c 'desktop/ui/app.css' "$ROOT/desktop/src-tauri/tauri.conf.json")" -eq 3 ]] ||
    fail "Desktop CSS corresponding source is incomplete in package payloads"
python3 "$ROOT/tests/check-desktop-ui.py" >/dev/null ||
    fail "Desktop HTML/JavaScript/Rust contract is inconsistent"
ok "Desktop verifies egress and exposes sortable sanitized per-location health"

python3 "$ROOT/tests/check-capabilities.py" >/dev/null ||
    fail "cross-surface capability registry is inconsistent"
ok "CLI/TUI/Desktop capability parity and release gates"

python3 "$ROOT/tests/check-api-contract.py" >/dev/null ||
    fail "versioned local API contract is inconsistent"
ok "versioned local API contract"

python3 "$ROOT/tests/check-protocol-registry.py" >/dev/null ||
    fail "protocol registry and orchestration policy are inconsistent"
ok "protocol registry and AI orchestration policy"

python3 "$ROOT/tests/check-agent-control-registry.py" >/dev/null ||
    fail "agent-control transport registry and security policy are inconsistent"
ok "reverse agent-control transports and ingress policy"

grep -q 'agent_control::get_agent_integrations' "$ROOT/desktop/src-tauri/src/main.rs" ||
    fail "Desktop agent diagnostics are not exposed through a typed command"
if grep -ERq 'run_agent_operation|codex-remote-(start|pair|stop)|pairingGrant' \
    "$ROOT/desktop/ui" "$ROOT/desktop/src-tauri/src"; then
    fail "Desktop retains executable Agent Control renderer authority"
fi
if grep -q 'Command::new' "$ROOT/desktop/src-tauri/src/agent_control.rs"; then
    fail "Desktop diagnostics execute an untrusted discovered agent binary"
fi
ok "Desktop Agent Control is diagnostics-only and exposes no renderer mutation"

python3 "$ROOT/tests/check-managed-protocol-adapter.py" >/dev/null ||
    fail "managed proxy profile and sing-box renderer boundary are inconsistent"
ok "managed proxy profiles generate a closed sing-box TUN graph"

python3 "$ROOT/tests/check-runtime-adapter-registry.py" >/dev/null ||
    fail "runtime adapter registry overclaims an unverified lifecycle"
ok "modern protocol runtime adapters have explicit execution and release gates"

python3 "$ROOT/tests/check-planner-examples.py" >/dev/null ||
    fail "planner SDK examples do not enforce strict bounded JSON"
ok "planner SDK examples enforce strict bounded JSON"

declare -A protocol_uri_schemes=(
    [vless]=vless
    [hysteria2]=hysteria2
    [hy2]=hysteria2
    [mieru]=mieru
    [mierus]=mieru
    [tuic]=tuic
    [ss]=shadowsocks2022
    [trojan]=trojan
    [anytls]=anytls
)
for protocol_scheme in "${!protocol_uri_schemes[@]}"; do
    protocol_detection="$({
        printf '%s://user:password@example.invalid:443?id=secret#private' \
            "$protocol_scheme"
    } | "$ROOT/mazzy-vpn" protocols detect --stdin --json)" ||
        fail "protocol detector rejected $protocol_scheme"
    jq -e --arg protocol "${protocol_uri_schemes[$protocol_scheme]}" '
        .recognized == true
        and .protocol == $protocol
        and .contains_secrets == true
    ' <<<"$protocol_detection" >/dev/null ||
        fail "protocol detector mislabeled $protocol_scheme"
    if jq -r '.. | strings' <<<"$protocol_detection" |
        grep -Eq 'password|example\.invalid|secret|private'; then
        fail "protocol detector exposed $protocol_scheme credentials"
    fi
done
declare -A protocol_json_types=(
    [vless]=vless
    [hysteria2]=hysteria2
    [tuic]=tuic
    [trojan]=trojan
    [anytls]=anytls
    [shadowtls]=shadowtls
    [naive]=naive
    [mieru]=mieru
)
for protocol_type in "${!protocol_json_types[@]}"; do
    protocol_detection="$(
        jq -nc --arg type "$protocol_type" '{
            outbounds: [{
                type: $type,
                server: "example.invalid",
                password: "json-secret"
            }]
        }' | "$ROOT/mazzy-vpn" protocols detect --stdin --json
    )" || fail "JSON protocol detector rejected $protocol_type"
    jq -e --arg protocol "${protocol_json_types[$protocol_type]}" '
        .recognized == true
        and .protocol == $protocol
        and .input_kind == "configuration-json"
        and .contains_secrets == true
    ' <<<"$protocol_detection" >/dev/null ||
        fail "JSON protocol detector mislabeled $protocol_type"
    if jq -r '.. | strings' <<<"$protocol_detection" |
        grep -Eq 'example\.invalid|json-secret'; then
        fail "JSON protocol detector exposed $protocol_type secrets"
    fi
done
protocol_detection="$(
    jq -nc '{
        outbounds: [{
            type: "shadowsocks",
            method: "2022-blake3-aes-256-gcm",
            server: "example.invalid",
            password: "json-secret"
        }]
    }' | "$ROOT/mazzy-vpn" protocols detect --stdin --json
)" || fail "JSON protocol detector rejected Shadowsocks 2022"
jq -e '.recognized == true and .protocol == "shadowsocks2022"' \
    <<<"$protocol_detection" >/dev/null ||
    fail "JSON protocol detector mislabeled Shadowsocks 2022"
protocol_detection="$(
    printf '%s\n' '{' \
        '  "listen": "socks://127.0.0.1:1080",' \
        '  "proxy": "https://user:json-secret@example.invalid"' \
        '}' | "$ROOT/mazzy-vpn" protocols detect --stdin --json
)" || fail "JSON protocol detector rejected official NaiveProxy shape"
jq -e '.recognized == true and .protocol == "naive"' \
    <<<"$protocol_detection" >/dev/null ||
    fail "JSON protocol detector mislabeled official NaiveProxy shape"
protocol_detection="$(
    jq -nc '{
        profiles: [{
            profileName: "default",
            user: {name: "user", password: "json-secret"},
            servers: [{domainName: "example.invalid", portBindings: []}]
        }],
        activeProfile: "default"
    }' | "$ROOT/mazzy-vpn" protocols detect --stdin --json
)" || fail "JSON protocol detector rejected official Mieru shape"
jq -e '.recognized == true and .protocol == "mieru"' \
    <<<"$protocol_detection" >/dev/null ||
    fail "JSON protocol detector mislabeled official Mieru shape"
set +e
protocol_detection="$(
    printf '%s' '{"type":"vless","type":"trojan"}' |
        "$ROOT/mazzy-vpn" protocols detect --stdin --json
)"
protocol_detection_status=$?
set -e
[[ "$protocol_detection_status" -eq 2 ]] ||
    fail "JSON protocol detector accepted duplicate keys"
jq -e '.recognized == false and .reason == "duplicate-key"' \
    <<<"$protocol_detection" >/dev/null ||
    fail "JSON protocol detector returned an unsafe duplicate-key response"
set +e
protocol_detection="$(
    printf '%s' '{"type":"vless","\u0074ype":"trojan"}' |
        "$ROOT/mazzy-vpn" protocols detect --stdin --json
)"
protocol_detection_status=$?
set -e
[[ "$protocol_detection_status" -eq 2 ]] ||
    fail "JSON protocol detector accepted an escaped-equivalent duplicate key"
jq -e '.recognized == false and .reason == "duplicate-key"' \
    <<<"$protocol_detection" >/dev/null ||
    fail "JSON protocol detector did not normalize escaped duplicate keys"
set +e
protocol_detection="$(
    printf '%s' '{"outbounds":[{"type":"vless"},{"type":"trojan"}]}' |
        "$ROOT/mazzy-vpn" protocols detect --stdin --json
)"
protocol_detection_status=$?
set -e
[[ "$protocol_detection_status" -eq 2 ]] ||
    fail "JSON protocol detector accepted an ambiguous multi-protocol config"
jq -e '.recognized == false and .reason == "ambiguous-protocol"' \
    <<<"$protocol_detection" >/dev/null ||
    fail "JSON protocol detector returned an unsafe ambiguity response"
protocol_detection="$(
    printf '%s\n' 'vless://user:password@example.invalid:443?id=secret#private' |
        "$ROOT/mazzy-vpn" protocols detect --stdin --json
)" || fail "protocol detector rejected a newline-terminated URI"
jq -e '.recognized == true and .protocol == "vless"' \
    <<<"$protocol_detection" >/dev/null ||
    fail "protocol detector mislabeled a newline-terminated URI"
protocol_detection="$(
    printf '%s\r\n' 'vless://user:password@example.invalid:443?id=secret#private' |
        "$ROOT/mazzy-vpn" protocols detect --stdin --json
)" || fail "protocol detector rejected a CRLF-terminated URI"
jq -e '.recognized == true and .protocol == "vless"' \
    <<<"$protocol_detection" >/dev/null ||
    fail "protocol detector mislabeled a CRLF-terminated URI"
set +e
protocol_detection="$(
    printf 'unknown://user:password@example.invalid' |
        "$ROOT/mazzy-vpn" protocols detect --stdin --json
)"
protocol_detection_status=$?
set -e
[[ "$protocol_detection_status" -eq 2 ]] ||
    fail "protocol detector accepted an unknown scheme"
jq -e '.recognized == false and .reason == "unsupported-scheme"' \
    <<<"$protocol_detection" >/dev/null ||
    fail "protocol detector returned an unsafe unknown-scheme response"
set +e
protocol_detection="$(
    printf 'vless://safe\0hidden' |
        "$ROOT/mazzy-vpn" protocols detect --stdin --json
)"
protocol_detection_status=$?
set -e
[[ "$protocol_detection_status" -eq 2 ]] ||
    fail "protocol detector accepted a control byte"
jq -e '.recognized == false and .reason == "invalid-input"' \
    <<<"$protocol_detection" >/dev/null ||
    fail "protocol detector returned an unsafe control-byte response"
set +e
protocol_detection="$(
    printf 'vless://safe\n\n' |
        "$ROOT/mazzy-vpn" protocols detect --stdin --json
)"
protocol_detection_status=$?
set -e
[[ "$protocol_detection_status" -eq 2 ]] ||
    fail "protocol detector accepted multiple trailing line terminators"
jq -e '.recognized == false and .reason == "invalid-input"' \
    <<<"$protocol_detection" >/dev/null ||
    fail "protocol detector returned an unsafe multi-line response"
ok "protocol URI/JSON classification is bounded and credential-redacted"

python3 "$ROOT/tests/audit-runtime-hardcodes.py" >/dev/null ||
    fail "runtime hard-code boundaries are inconsistent"
ok "runtime hard-code boundaries"

python3 "$ROOT/tests/check-codeql-boundary.py" >/dev/null ||
    fail "CodeQL ownership and vendor provenance boundaries are inconsistent"
ok "CodeQL scans owned code and the excluded vendor tree remains byte-verified"

printf '1..%d\n' "$pass"
