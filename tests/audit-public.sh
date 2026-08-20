#!/usr/bin/env bash
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
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
        -e 'packaging/linux/systemd/mazzy-vpn-api-recovery.service.d/10-package-exec.conf' \
        -e 'packaging/linux/systemd/mazzy-vpn-api.socket.d/10-package-docs.conf' \
        -e 'packaging/linux/systemd/mazzy-vpn-api@.service.d/10-package-exec.conf' \
        -e 'packaging/linux/systemd/user/mazzy-agentd.service.d/10-package-exec.conf' \
        -e 'packaging/linux/systemd/vpnctl-health.service.d/10-package-exec.conf' \
        -e 'packaging/linux/systemd/vpnctl-health.timer.d/10-package-interval.conf' \
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
    # Skip vendored trees, submodule gitlinks and binary assets: they are
    # byte-verified upstream and only produce false positives. A gitlink
    # directory also makes grep return an error, which previously masked real
    # hits and let the audit pass by mistake.
    case "$file" in
        */vendor/*|vendor/*|*/testdata/*) continue ;;
        *.png|*.jpg|*.jpeg|*.gif|*.webp|*.svg|*.ico|*.pdf|*.woff|*.woff2|*.ttf) continue ;;
    esac
    [ -f "$file" ] || continue   # skip non-regular files (submodule gitlinks)
    scan_files+=("$file")
done < <(git ls-files --cached --others --exclude-standard)

((${#scan_files[@]} > 0)) || fail "no tracked files to audit"

# grep returns 2 on any read error; capture hits explicitly (|| true) and fail
# only on a non-empty result, so a read error can never silently pass.
key_hits="$(grep -InE \
    'BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY|<key>|auth-user-pass[[:space:]]+[^[:space:]]+' \
    "${scan_files[@]}" 2>/dev/null || true)"
if [ -n "$key_hits" ]; then
    printf '%s\n' "$key_hits" >&2
    fail "private-key or credential material pattern found"
fi

# Personal machine paths and local hostnames. A redacted placeholder
# (/home/.../ or /home/<...>) is allowed; a concrete /home/<user>/ is not.
path_hits="$(grep -InE \
    '/home/[A-Za-z0-9_-]+/|/run/media/[A-Za-z0-9_-]+/|@[[:alnum:]._-]+\.(local|lan)\b' \
    "${scan_files[@]}" 2>/dev/null | grep -vE '/home/\.\.\.|/home/<' || true)"
if [ -n "$path_hits" ]; then
    printf '%s\n' "$path_hits" >&2
    fail "personal machine path or local hostname found"
fi

# The real production egress IP must never be committed.
ip_hits="$(grep -InE '95\.211\.225\.232' "${scan_files[@]}" 2>/dev/null || true)"
if [ -n "$ip_hits" ]; then
    printf '%s\n' "$ip_hits" >&2
    fail "real production egress IP found"
fi

if git log --all --format= --name-only |
   filter_safe_system_config |
   grep -Eiq \
    '(^|/)conf/|(^|/)(id_rsa|id_ed25519|credentials|secrets?)([.]|$)|[.](conf|ovpn|nmconnection|key|pem|p12|pfx)$'; then
    fail "VPN profiles or credential-file extensions occur in Git history"
fi

printf 'PUBLIC AUDIT OK: tracked files and Git history contain no known VPN secrets or personal paths.\n'
