#!/usr/bin/env bash
# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
# Copyright © 2026 Nik m (@mazurovn). All rights reserved.
#
# Mazzy VPN (Go) installer.
# Installs the autonomous static mazzy-vpn binary, prepares directories,
# runs preflight checks + self-diagnostics, and optionally imports a profile.
#
# Usage:
#   ./install.sh                 interactive install
#   ./install.sh --prefix DIR    install binary under DIR/bin (default /usr/local)
#   ./install.sh --uninstall     remove the binary and (optionally) state
#   ./install.sh --no-color      plain output
set -euo pipefail

# ---------------------------------------------------------------------------
# Presentation
# ---------------------------------------------------------------------------
USE_COLOR=1
[[ -t 1 ]] || USE_COLOR=0
for a in "$@"; do [[ "$a" == "--no-color" ]] && USE_COLOR=0; done

if [[ "$USE_COLOR" == 1 ]]; then
    C_G=$'\033[32m'; C_Y=$'\033[33m'; C_R=$'\033[31m'; C_B=$'\033[1m'; C_0=$'\033[0m'
else
    C_G=""; C_Y=""; C_R=""; C_B=""; C_0=""
fi
ok()   { printf '%s[ OK ]%s %s\n' "$C_G" "$C_0" "$*"; }
warn() { printf '%s[WARN]%s %s\n' "$C_Y" "$C_0" "$*"; }
err()  { printf '%s[FAIL]%s %s\n' "$C_R" "$C_0" "$*" >&2; }
info() { printf '%s%s%s\n' "$C_B" "$*" "$C_0"; }
die()  { err "$*"; exit 1; }

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_NAME="mazzy-vpn"
BINARY_SRC="$SCRIPT_DIR/$BINARY_NAME"
PREFIX="/usr/local"
# State/run dirs are overridable for rootless/test installs (mirrors the CLI's
# MAZZY_STATE_DIR / MAZZY_RUN_DIR runtime overrides).
STATE_DIR="${MAZZY_STATE_DIR:-/var/lib/mazzy-vpn}"
RUN_DIR="${MAZZY_RUN_DIR:-/run/mazzy-vpn}"
DO_UNINSTALL=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --prefix) PREFIX="${2:?--prefix needs a directory}"; shift 2 ;;
        --uninstall) DO_UNINSTALL=1; shift ;;
        --no-color) shift ;;
        -h|--help)
            grep '^#' "$0" | sed 's/^# \{0,1\}//' | head -20; exit 0 ;;
        *) die "Unknown option: $1" ;;
    esac
done
BIN_DIR="$PREFIX/bin"
INSTALLED_BIN="$BIN_DIR/$BINARY_NAME"

# ---------------------------------------------------------------------------
# Privilege helper: re-run individual steps with sudo when needed.
# ---------------------------------------------------------------------------
# writable_ancestor walks up from a (possibly not-yet-existing) target path to
# the first directory that exists, and reports whether we can write there. This
# is what actually determines if `install -d` will succeed without elevation.
writable_ancestor() {
    local p
    p="$1"
    while [[ -n "$p" && ! -e "$p" ]]; do
        p="$(dirname "$p")"
    done
    [[ -n "$p" && -w "$p" ]]
}

# needs_root decides whether we need elevation: if the bin dir, the state dir
# AND the runtime dir are all creatable without root, a rootless install into a
# custom --prefix works without sudo.
needs_root() {
    writable_ancestor "$PREFIX/bin" &&
        writable_ancestor "$STATE_DIR" &&
        writable_ancestor "$RUN_DIR" && return 1
    return 0
}

SUDO=""
if [[ $EUID -ne 0 ]] && needs_root; then
    if command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    else
        die "Elevated privileges required for $PREFIX / $STATE_DIR and sudo not found. Re-run as root or use --prefix to a writable location."
    fi
fi
priv() { if [[ -n "$SUDO" ]]; then "$SUDO" "$@"; else "$@"; fi; }

# ---------------------------------------------------------------------------
# Uninstall
# ---------------------------------------------------------------------------
if [[ "$DO_UNINSTALL" == 1 ]]; then
    info "Uninstalling Mazzy VPN (Go)"
    if [[ -e "$INSTALLED_BIN" ]]; then
        priv rm -f "$INSTALLED_BIN" && ok "Removed $INSTALLED_BIN"
    else
        warn "Binary not found at $INSTALLED_BIN"
    fi
    printf 'Remove state directory %s (keeps no profiles)? [y/N] ' "$STATE_DIR"
    # `read` returns non-zero on EOF (non-interactive/piped stdin); under set -e
    # that would abort. Default to "no" and keep going.
    read -r ans || ans=""
    if [[ "${ans,,}" == y* ]]; then
        priv rm -rf "$STATE_DIR" && ok "Removed $STATE_DIR"
    else
        info "Kept $STATE_DIR"
    fi
    ok "Uninstall complete"
    exit 0
fi

# ---------------------------------------------------------------------------
# 1. Preflight: verify the binary and base OS tools
# ---------------------------------------------------------------------------
info "Mazzy VPN (Go) — installer"
echo
info "1) Preflight checks"

[[ -f "$BINARY_SRC" ]] || die "Binary not found next to installer: $BINARY_SRC"
[[ -x "$BINARY_SRC" ]] || chmod +x "$BINARY_SRC"

if file "$BINARY_SRC" 2>/dev/null | grep -q "statically linked"; then
    ok "Binary is statically linked (self-contained)"
else
    warn "Binary may not be static; runtime libraries could be required"
fi

# The Go engine is embedded; only base OS tools are needed.
PREFLIGHT_FAIL=0
for tool in ip nft; do
    if command -v "$tool" >/dev/null 2>&1; then
        ok "Base tool present: $tool"
    else
        err "Missing base tool: $tool (from iproute2/nftables)"
        PREFLIGHT_FAIL=1
    fi
done
if command -v resolvectl >/dev/null 2>&1 || command -v resolvconf >/dev/null 2>&1; then
    ok "DNS backend present (resolvectl/resolvconf)"
else
    warn "No resolvectl/resolvconf; VPN DNS may not apply"
fi
if [[ -e /dev/net/tun ]]; then
    ok "TUN device present (/dev/net/tun)"
else
    err "/dev/net/tun missing; cannot create tunnels (load 'tun' module)"
    PREFLIGHT_FAIL=1
fi
[[ "$PREFLIGHT_FAIL" == 0 ]] || die "Preflight failed; install the missing base tools and retry"

# ---------------------------------------------------------------------------
# 2. Self-test the binary BEFORE installing it
# ---------------------------------------------------------------------------
echo
info "2) Binary self-test"
if "$BINARY_SRC" version >/dev/null 2>&1; then
    ok "Binary runs: $("$BINARY_SRC" version)"
else
    die "Binary failed to execute"
fi

# ---------------------------------------------------------------------------
# 3. Install binary + directories
# ---------------------------------------------------------------------------
echo
info "3) Installing"
priv install -d -m 0755 "$BIN_DIR"
priv install -m 0755 "$BINARY_SRC" "$INSTALLED_BIN"
ok "Installed $INSTALLED_BIN"

priv install -d -m 0700 "$STATE_DIR"
ok "State directory ready: $STATE_DIR"

# Install the systemd template unit for permanent operation (optional to use).
# Skipped for a rootless/custom --prefix install: /etc/systemd/system is a
# system path, so writing it would force elevation the user did not ask for.
if [[ "$PREFIX" == "/usr/local" || "$PREFIX" == "/usr" ]] &&
    [[ -f "$SCRIPT_DIR/mazzy-vpn@.service" ]] && command -v systemctl >/dev/null 2>&1; then
    priv install -m 0644 "$SCRIPT_DIR/mazzy-vpn@.service" /etc/systemd/system/mazzy-vpn@.service
    priv systemctl daemon-reload 2>/dev/null || true
    ok "systemd unit installed: mazzy-vpn@.service"
else
    info "Skipped systemd unit (custom prefix or no systemd); use 'mazzy-vpn daemon' directly."
fi
# /run is tmpfs; created at connect time, but prepare it now if possible.
if priv install -d -m 0700 "$RUN_DIR" 2>/dev/null; then
    ok "Runtime directory ready: $RUN_DIR"
else
    warn "Could not pre-create $RUN_DIR (created at connect time)"
fi

# ---------------------------------------------------------------------------
# 4. Self-diagnostics via the installed binary
# ---------------------------------------------------------------------------
echo
info "4) Self-diagnostics (mazzy-vpn doctor)"
if "$INSTALLED_BIN" doctor; then
    ok "Doctor reports no blocking problems"
else
    warn "Doctor reported issues above (review before connecting)"
fi

# ---------------------------------------------------------------------------
# 5. Optional profile setup
# ---------------------------------------------------------------------------
echo
info "5) Network analysis"
"$INSTALLED_BIN" netdiag || true

echo
info "6) Profile setup (optional)"
# Only prompt when attached to a terminal; a non-interactive install
# (piped/automation) skips import cleanly instead of aborting on EOF.
pdir=""
if [[ -t 0 ]]; then
    echo "Import your profiles now (AmneziaWG .conf / OpenVPN .ovpn) into the catalog."
    printf 'Path to a profile file or directory (blank to skip): '
    read -r pdir || pdir=""
else
    info "Non-interactive install: skipping profile import (use 'mazzy-vpn import <DIR>')."
fi
if [[ -n "$pdir" ]]; then
    if [[ -e "$pdir" ]]; then
        "$INSTALLED_BIN" import "$pdir" || true
        echo
        "$INSTALLED_BIN" profiles || true
    else
        warn "Path not found: $pdir"
    fi
fi

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------
echo
ok "Installation complete."
cat <<EOF

Next steps:
  $BINARY_NAME                        # interactive menu
  $BINARY_NAME import <DIR>           # import profiles into the catalog
  $BINARY_NAME profiles               # list managed profiles
  $BINARY_NAME test                   # probe all servers (which zones work)
  $BINARY_NAME best                   # pick the best zone
  $BINARY_NAME adapters               # choose wired vs Wi‑Fi uplink
  $BINARY_NAME netdiag                # analyze the network + fixes
  sudo $BINARY_NAME up --best         # connect to the best zone (Ctrl+C stops)
  sudo $BINARY_NAME up <NAME>         # connect to a specific zone

Permanent background operation (auto-reconnect, notifications):
  sudo $BINARY_NAME daemon <NAME>                    # run in this shell
  sudo systemctl enable --now mazzy-vpn@<NAME>       # start at boot
  sudo systemctl status mazzy-vpn@<NAME>             # check
  sudo systemctl disable --now mazzy-vpn@<NAME>      # stop

The engine is embedded: no awg/awg-quick/wg/jq are required.
EOF
