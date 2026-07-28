#!/usr/bin/env bash
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

fail() {
    printf 'PUBLIC AUDIT FAIL: %s\n' "$*" >&2
    exit 1
}

filter_safe_system_config() {
    grep -Fvx \
        -e 'systemd/mazzy-vpn-tmpfiles.conf' \
        -e 'packaging/linux/systemd/mazzy-vpn-api.socket.d/10-package-docs.conf' \
        -e 'packaging/linux/systemd/mazzy-vpn-api@.service.d/10-package-exec.conf' \
        -e 'packaging/linux/systemd/vpnctl-health.service.d/10-package-exec.conf' \
        -e 'packaging/linux/systemd/vpnctl-test-recovery.service.d/10-package-exec.conf' \
        -e 'packaging/linux/systemd/vpnctl.service.d/10-package-exec.conf'
}

if git ls-files |
   filter_safe_system_config |
   grep -Eiq \
    '(^|/)conf/|(^|/)(id_rsa|id_ed25519|credentials|secrets?)([.]|$)|[.](conf|ovpn|nmconnection|key|pem|p12|pfx)$'; then
    fail "a VPN profile, credential file or private-key extension is tracked"
fi

scan_files=()
while IFS= read -r file; do
    case "$file" in
        tests/run.sh|tests/audit-public.sh) continue ;;
    esac
    scan_files+=("$file")
done < <(git ls-files --cached --others --exclude-standard)

((${#scan_files[@]} > 0)) || fail "no tracked files to audit"

if grep -InE \
    'BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY|PrivateKey[[:space:]]*=|PresharedKey[[:space:]]*=|<key>|auth-user-pass[[:space:]]+[^[:space:]]+' \
    "${scan_files[@]}"; then
    fail "private-key or credential material pattern found"
fi

if grep -InE \
    '/home/[^/<[:space:]]+|/run/media/[^/<[:space:]]+|@[[:alnum:]._-]+\.(local|lan)\b' \
    "${scan_files[@]}"; then
    fail "personal machine path or local hostname found"
fi

if git log --all --format= --name-only |
   filter_safe_system_config |
   grep -Eiq \
    '(^|/)conf/|(^|/)(id_rsa|id_ed25519|credentials|secrets?)([.]|$)|[.](conf|ovpn|nmconnection|key|pem|p12|pfx)$'; then
    fail "VPN profiles or credential-file extensions occur in Git history"
fi

printf 'PUBLIC AUDIT OK: tracked files and Git history contain no known VPN secrets or personal paths.\n'
