#!/usr/bin/env bash
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
CLI="$ROOT/mazzy-vpn"
COMPAT_CLI="$ROOT/vpnctl"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

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
if [[ -n "${FAKE_SYSTEMCTL_DELAY_ONCE_FILE:-}" &&
      ! -e "$FAKE_SYSTEMCTL_DELAY_ONCE_FILE" ]]; then
    touch "$FAKE_SYSTEMCTL_DELAY_ONCE_FILE"
    printf '%s\n' "$$" >"${FAKE_SYSTEMCTL_DELAY_PID_FILE:?}"
    sleep "${FAKE_SYSTEMCTL_DELAY_ONCE_SECONDS:-10}"
fi
if [[ "${FAKE_SYSTEMCTL_DELAY_SECONDS:-0}" != "0" ]]; then
    sleep "$FAKE_SYSTEMCTL_DELAY_SECONDS"
fi
case "$*" in
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
        ;;
    "stop vpnctl.service")
        [[ "${FAKE_SYSTEMCTL_STOP_FAIL:-0}" == "1" ]] && exit 1
        ;;
    "is-active vpnctl.service")
        [[ "${FAKE_SYSTEMCTL_INACTIVE:-0}" == "1" ]] && exit 3
        exit 0
        ;;
    *is-active*) exit 0 ;;
    *cat*) exit 0 ;;
esac
exit 0
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

cat >"$TMP/fakebin/openvpn" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >"${FAKE_OPENVPN_LOG:?}"
if [[ -n "${FAKE_OPENVPN_DELAY_SECONDS:-}" ]]; then
    sleep "$FAKE_OPENVPN_DELAY_SECONDS"
fi
if [[ "${FAKE_OPENVPN_TOO_MANY:-0}" == "1" ]]; then
    printf "Halt command was pushed by server ('Too many connections')\n" >&2
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
    "link show vpnovpn0") exit 0 ;;
    "route show default") printf 'default via 192.0.2.1 dev eth0\n'; exit 0 ;;
esac
/usr/sbin/ip "$@"
EOF

cat >"$TMP/fakebin/awg" <<'EOF'
#!/usr/bin/env bash
[[ "$*" == "show interfaces" ]] && exit 0
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
export FAKE_SYSTEMCTL_COUNTER="$TMP/systemctl.counter"
export FAKE_TIMEOUT_LOG="$TMP/timeout.log"
export FAKE_SYSTEMD_RUN_LOG="$TMP/systemd-run.log"
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

"$CLI" version | grep -q '^Mazzy VPN 1\.3\.2 (mazzy-vpn; alias: vpnctl)$'
"$COMPAT_CLI" version | grep -q '^Mazzy VPN 1\.3\.2 ' ||
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
VPNCTL_CONFIG_DIR="$TMP/import-files-target" \
    "$CLI" import-files \
    "$TMP/import-source/mixed/AWG.conf" \
    "$TMP/import-source/mixed/WG.conf" >/dev/null
[[ -f "$TMP/import-files-target/amneziawg/AWG.conf" &&
   -f "$TMP/import-files-target/wireguard/WG.conf" ]] ||
    fail "multi-file import did not detect both VPN protocols"
[[ "$(stat -c %a "$TMP/import-files-target/amneziawg/AWG.conf")" == "600" ]] ||
    fail "multi-file import did not close target permissions"
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
"$CLI" connect openvpn "Test Server" >/dev/null
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
rm -f "$FAKE_IP_RULES"
ok "connect persists profile and removes only stale wg-quick policy"

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
exec 6>"$TMP/run/.action.lock"
flock -n 6 || fail "idle TUI retained the action lock after connect"
flock -u 6
wait "$menu_live_pid"
ok "idle TUI releases the watchdog action lock"

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
sed -i '/^TEST_/d; s/^MODE=test$/MODE=normal/' "$TMP/state/active"
ok "test mode logging and OpenVPN server-halt detection"

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
exec 8>"$VPNCTL_API_ACTION_DIR/.mutation.lock"
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
    fail "terminal audit fault lost the idempotent action outcome"
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
rm -f -- "$VPNCTL_API_AUDIT_FILE"
ok "local API preserves idempotency when terminal audit persistence fails"

api_connect_request="$(
    jq -cn --arg profile_id "$profile_id" '{
        api_version: "1.0",
        request_id: "request-connect-0001",
        operation: "lifecycle.connect",
        action_id: "action-connect-0001",
        authorization: "system-mutate",
        deadline_ms: 5000,
        payload: {profile_id: $profile_id}
    }'
)"
: >"$FAKE_SYSTEMCTL_LOG"
api_connect_response="$(printf '%s\n' "$api_connect_request" | "$CLI" _api-dispatch)"
jq -e '
    .status == "ok"
    and .result.action_id == "action-connect-0001"
    and .result.state == "succeeded"
    and .result.rollback.state == "not-needed"
' <<<"$api_connect_response" >/dev/null ||
    fail "local API lifecycle.connect did not succeed"
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
        deadline_ms: 500,
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
    '--kill-after=5s 0\.[0-9]{3}s .*/mazzy-vpn reconnect' \
    "$FAKE_TIMEOUT_LOG" ||
    fail "local API rounded a millisecond deadline up to a full second"
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
ok "local API accounts preflight time and preserves millisecond deadlines"

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
rm -f "$FAKE_ADGUARD_ACTIVE"
if ! "$CLI" connect openvpn "Test Server" >/dev/null; then
    fail "daemonized AdGuard fallback inherited the vpnctl action lock"
fi
unset FAKE_ADGUARD_FORK
ok "AdGuard fallback is restored without inheriting the action lock"

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
batch_output="$("$CLI" test-all openvpn --timeout 2)"
unset FAKE_SYSTEMCTL_START_LIMIT
grep -q 'tested=1 passed=1 failed=0 skipped=0' <<<"$batch_output" ||
    fail "test-all summary is wrong"
grep -q '^MODE=normal$' "$TMP/state/active" || fail "test-all did not restore state"
ok "test-all resets systemd start limit and rolls back"

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
exec 7>"$TMP/run/.action.lock"
flock 7
timeout 5 "$CLI" _test-timeout race-token >/dev/null 2>&1 &
guard_pid=$!
sleep 0.2
kill -0 "$guard_pid" 2>/dev/null || fail "timeout guard did not wait for action lock"
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
if "$CLI" test openvpn "Test Server" --timeout 2 >/dev/null 2>&1; then
    fail "failed tunnel test returned success"
fi
unset FAKE_CURL_FAIL
grep -q '^MODE=normal$' "$TMP/state/active" || fail "timeout did not restore previous state"
[[ ! -e "$TMP/state/test.transaction" ]] || fail "failed test transaction was not cleaned"
ok "test timeout rolls back"

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

touch "$FAKE_ADGUARD_ACTIVE"
"$CLI" test openvpn "Test Server" --timeout 2 --keep >/dev/null
grep -q '^MODE=normal$' "$TMP/state/active" || fail "--keep did not finalize normal state"
grep -q '^DESIRED=up$' "$TMP/state/active" || fail "--keep did not retain connection"
[[ ! -e "$FAKE_ADGUARD_ACTIVE" ]] ||
    fail "--keep restored conflicting AdGuard fallback over managed VPN"
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
ok "emergency failure rolls back"

if "$CLI" connect wireguard Unsafe >/dev/null 2>&1; then
    fail "unsafe WireGuard hook was accepted"
fi
ok "unsafe hooks are rejected"

: >"$TMP/systemctl.log"
FAKE_DEFAULT_IPV4=198.51.100.9 "$CLI" _health-check >/dev/null
if grep -q 'restart --no-block vpnctl.service' "$TMP/systemctl.log"; then
    fail "auto health policy forced full-tunnel recovery for a split profile"
fi

: >"$TMP/systemctl.log"
FAKE_DEFAULT_IPV4=198.51.100.9 VPNCTL_HEALTH_REQUIRE_DEFAULT_EGRESS=yes \
    "$CLI" _health-check >/dev/null
FAKE_DEFAULT_IPV4=198.51.100.9 VPNCTL_HEALTH_REQUIRE_DEFAULT_EGRESS=yes \
    "$CLI" _health-check >/dev/null
grep -q 'restart --no-block vpnctl.service' "$TMP/systemctl.log" ||
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
grep -q 'restart --no-block vpnctl.service' "$TMP/systemctl.log" ||
    fail "auto health policy missed compact WireGuard full-tunnel AllowedIPs"
"$CLI" connect openvpn "Test Server" >/dev/null
rm -f -- "$TMP/config/wireguard/Full.conf"
ok "watchdog parses compact WireGuard full-tunnel routes"

: >"$TMP/systemctl.log"
export FAKE_CURL_FAIL=1
"$CLI" _health-check >/dev/null
"$CLI" _health-check >/dev/null
grep -q 'restart --no-block vpnctl.service' "$TMP/systemctl.log" ||
    fail "watchdog did not restart after threshold"
unset FAKE_CURL_FAIL
ok "watchdog reconnects after two confirmed network failures"

: >"$TMP/systemctl.log"
export FAKE_SYSTEMCTL_INACTIVE=1
"$CLI" _health-check >/dev/null
unset FAKE_SYSTEMCTL_INACTIVE
grep -q '^start --no-block vpnctl.service$' "$TMP/systemctl.log" ||
    fail "watchdog did not immediately start an inactive desired service"
ok "watchdog immediately restores an inactive desired service"

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
grep -q '^disable --now vpnctl-health.timer$' "$TMP/systemctl.log" ||
    fail "Desktop monitor control did not stop the health timer"
ok "independent health monitor control"

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
   -f "$stage/usr/local/lib/mazzy-vpn/docs/FEATURE_PARITY.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/capabilities.json" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/API_CONTRACT.en.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/API_CONTRACT.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/PROJECT_STATUS.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/AUDIT_2026-07-28.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/AUDIT_2026-08-01.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/AUDIT_2026-08-01_PROTOCOLS.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/ARCHITECTURE.en.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/ARCHITECTURE.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/api/v1/manifest.json" &&
   -f "$stage/usr/local/lib/mazzy-vpn/api/v1/schema.json" &&
   -f "$stage/usr/local/lib/mazzy-vpn/protocols/v1/registry.json" &&
   -f "$stage/usr/local/lib/mazzy-vpn/protocols/v1/schema.json" &&
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
grep -q '^OnUnitActiveSec=20s$' "$stage/etc/systemd/system/vpnctl-health.timer" ||
    fail "health timer interval is not the expected 20 seconds"
[[ -f "$stage/etc/systemd/system/vpnctl-test-recovery.service" ]] ||
    fail "test recovery unit not staged"
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
grep -q '^NoNewPrivileges=yes$' \
    "$stage/etc/systemd/system/mazzy-vpn-api@.service" ||
    fail "local API request service is missing process hardening"
grep -q 'ExecStart=/usr/local/bin/mazzy-vpn' \
    "$stage/etc/systemd/system/vpnctl.service" ||
    fail "staged service does not use Mazzy VPN command"
grep -Eq '^StartLimitBurst=([1-9][0-9]{2,}|[1-9][0-9]{3,})$' \
    "$stage/etc/systemd/system/vpnctl.service" ||
    fail "staged service start limit is too low for test-all"
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
    "procps", "sed", "socat", "systemd",
):
    assert dependency in deb["depends"]
for dependency in (
    "netcat-openbsd", "network-manager-l2tp", "openvpn", "wireguard-tools",
):
    assert dependency in deb["recommends"]
for dependency in (
    "bash", "diffutils", "findutils", "gawk", "grep", "iproute", "jq",
    "polkit", "procps-ng", "sed", "socat", "systemd",
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
    "$ROOT/packaging/linux/systemd/mazzy-vpn-api@.service.d/10-package-exec.conf" \
    "$ROOT/packaging/linux/systemd/vpnctl-health.service.d/10-package-exec.conf" \
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
ok "runtime start cannot deadlock on boot recovery"

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
