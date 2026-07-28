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
    "is-active vpnctl.service")
        [[ "${FAKE_SYSTEMCTL_INACTIVE:-0}" == "1" ]] && exit 3
        exit 0
        ;;
    *is-active*) exit 0 ;;
    *cat*) exit 0 ;;
esac
exit 0
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
printf '203.0.113.7'
EOF

cat >"$TMP/fakebin/ping" <<'EOF'
#!/usr/bin/env bash
[[ "${FAKE_PING_FAIL:-0}" == "1" ]] && exit 1
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
if [[ "${FAKE_OPENVPN_TOO_MANY:-0}" == "1" ]]; then
    printf "Halt command was pushed by server ('Too many connections')\n" >&2
fi
EOF

cat >"$TMP/fakebin/resolvectl" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${FAKE_RESOLVECTL_LOG:?}"
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
export FAKE_SYSTEMD_RUN_LOG="$TMP/systemd-run.log"
export FAKE_OPENVPN_LOG="$TMP/openvpn.log"
export FAKE_RESOLVECTL_LOG="$TMP/resolvectl.log"
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

"$CLI" version | grep -q '^Mazzy VPN 1\.2\.0 (mazzy-vpn; alias: vpnctl)$'
"$COMPAT_CLI" version | grep -q '^Mazzy VPN 1\.2\.0 ' ||
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
    VPNCTL_LANG="$language_code" "$CLI" help | grep -q "$language_marker" ||
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

validate_output="$("$CLI" validate openvpn)"
grep -q 'profiles=1 passed=1 failed=0' <<<"$validate_output" ||
    fail "valid profile was not accepted by validate"
ok "validate checks every selected profile"

probe_output="$("$CLI" probe openvpn --timeout 1)"
grep -q 'endpoints=1 ping_ok=1' <<<"$probe_output" ||
    fail "endpoint DNS/ping probe did not pass"
ok "endpoint probe"

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
ok "OpenVPN DNS lifecycle"

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
ok "local API query envelopes expose only sanitized status and profiles"

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
ok "local API recovers interrupted actions and bounds persistent journals"

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
   -f "$stage/usr/local/lib/mazzy-vpn/docs/FEATURE_PARITY.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/capabilities.json" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/API_CONTRACT.en.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/API_CONTRACT.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/PROJECT_STATUS.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/ARCHITECTURE.en.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/docs/ARCHITECTURE.ru.md" &&
   -f "$stage/usr/local/lib/mazzy-vpn/api/v1/manifest.json" &&
   -f "$stage/usr/local/lib/mazzy-vpn/api/v1/schema.json" &&
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
grep -q '^SocketMode=0660$' "$stage/etc/systemd/system/mazzy-vpn-api.socket" ||
    fail "local API socket is not protected by mode 0660"
grep -q '^SocketGroup=mazzy-vpn$' "$stage/etc/systemd/system/mazzy-vpn-api.socket" ||
    fail "local API socket is not restricted to the mazzy-vpn group"
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

python3 "$ROOT/tests/check-capabilities.py" >/dev/null ||
    fail "cross-surface capability registry is inconsistent"
ok "CLI/TUI/Desktop capability parity and release gates"

python3 "$ROOT/tests/check-api-contract.py" >/dev/null ||
    fail "versioned local API contract is inconsistent"
ok "versioned local API contract"

printf '1..%d\n' "$pass"
