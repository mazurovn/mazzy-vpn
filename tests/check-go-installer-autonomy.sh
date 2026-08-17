#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright © 2026 Nik m (@mazurovn). All rights reserved.
#
# Verify the Go installer is autonomous (P3-2): it must NOT install VPN backends
# or build anything (no apt/dnf/pacman/zypper install, no go build, no git clone,
# no make). The engine is embedded; only base OS tools are checked at runtime.
set -euo pipefail

installer="mazzy-vpn-cli/packaging/install.sh"
[[ -f "$installer" ]] || { echo "SKIP: $installer not present"; exit 0; }

fail=0
check() {
  local pattern="$1" label="$2"
  # Ignore comment lines and help text (# ...), only real command lines.
  if grep -nE "$pattern" "$installer" | grep -vE '^\s*[0-9]+:\s*#' | grep -vqE 'no awg|are required|# '; then
    echo "FAIL: installer contains $label:"
    grep -nE "$pattern" "$installer" | grep -vE '^\s*[0-9]+:\s*#' | head
    fail=1
  fi
}

check '\b(apt-get|apt) +install\b' "apt install"
check '\bdnf +install\b'           "dnf install"
check '\bpacman +-S\b'             "pacman install"
check '\bzypper +install\b'        "zypper install"
check '\bgit +clone\b'             "git clone"
check '\bgo +build\b'              "go build"
check '\bmake\b'                   "make"

if [[ "$fail" -eq 0 ]]; then
  echo "P3-2 OK: installer is autonomous (no VPN-backend install or build)."
fi
exit "$fail"
