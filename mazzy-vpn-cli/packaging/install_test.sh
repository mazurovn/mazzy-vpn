#!/usr/bin/env bash
# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
# Copyright © 2026 Nik m (@mazurovn). All rights reserved.
#
# Scenario matrix for install.sh. Exercises every rootless code path in a loop
# so regressions are caught systematically instead of one at a time. Privileged
# (system-path) installs are covered by the real installer-autonomy gate; here
# we keep everything under a writable temp prefix so the suite runs unprivileged
# and deterministically in CI.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALLER="$SCRIPT_DIR/install.sh"
BINARY="$SCRIPT_DIR/mazzy-vpn"

PASS=0
FAIL=0

# ok/no record a check result without the A && B || C footgun (SC2015).
ok()  { printf '  \033[32mok\033[0m   %s\n' "$1"; PASS=$((PASS + 1)); }
no()  { printf '  \033[31mFAIL\033[0m %s\n' "$1"; FAIL=$((FAIL + 1)); }

# check <condition-rc> <label>: pass when the last command's rc is 0.
check() {
    local rc="$1" label="$2"
    if [[ "$rc" -eq 0 ]]; then ok "$label"; else no "$label"; fi
}
# check_not <rc> <label>: pass when rc is non-zero.
check_not() {
    local rc="$1" label="$2"
    if [[ "$rc" -ne 0 ]]; then ok "$label"; else no "$label"; fi
}
# want <label>: pass; deny <label>: fail. Used after an explicit if-test.
want() { ok "$1"; }
deny() { no "$1"; }

newenv() { mktemp -d; }

RC=0
OUT=""
# run_installer <sandbox> <stdin-file> [extra args...]
run_installer() {
    local sandbox="$1"; shift
    local stdin_src="$1"; shift
    OUT="$(MAZZY_STATE_DIR="$sandbox/state" MAZZY_RUN_DIR="$sandbox/run" \
        bash "$INSTALLER" --prefix "$sandbox/usr" --no-color "$@" <"$stdin_src" 2>&1)"
    RC=$?
}

need_binary() {
    if [[ ! -f "$BINARY" ]]; then
        echo "building test binary..."
        ( cd "$SCRIPT_DIR/.." && GOWORK=off CGO_ENABLED=0 go build -mod=vendor -o packaging/mazzy-vpn ./cmd/mazzy-vpn )
    fi
}

echo "== install.sh scenario matrix =="
need_binary

# S1: non-interactive install (stdin closed) — must complete, exit 0, no sudo.
s="$(newenv)"
run_installer "$s" /dev/null
check "$RC" "S1 non-interactive: exit 0"
grep -q "Installation complete" <<<"$OUT"; check "$?" "S1: reached completion banner"
if [[ -x "$s/usr/bin/mazzy-vpn" ]]; then want "S1: binary installed"; else deny "S1: binary NOT installed"; fi
if grep -q "sudo:" <<<"$OUT"; then deny "S1: unexpectedly tried sudo"; else want "S1: no sudo in rootless path"; fi
rm -rf "$s"

# S2: blank profile answer (feed an empty line then EOF) — must complete.
s="$(newenv)"
printf '\n' >"$s/in"
run_installer "$s" "$s/in"
check "$RC" "S2 blank-answer: exit 0"
if [[ -x "$s/usr/bin/mazzy-vpn" ]]; then want "S2: binary installed"; else deny "S2: binary NOT installed"; fi
rm -rf "$s"

# S3: install then re-install (idempotency) — both must succeed.
s="$(newenv)"
run_installer "$s" /dev/null; first=$RC
run_installer "$s" /dev/null
if [[ "$first" -eq 0 && "$RC" -eq 0 ]]; then want "S3 re-install idempotent"; else deny "S3 re-install: first=$first second=$RC"; fi
rm -rf "$s"

# S4: uninstall, non-interactive — must complete, remove binary.
s="$(newenv)"
run_installer "$s" /dev/null
run_installer "$s" /dev/null --uninstall
check "$RC" "S4 uninstall: exit 0"
if [[ ! -e "$s/usr/bin/mazzy-vpn" ]]; then want "S4: binary removed"; else deny "S4: binary still present"; fi
rm -rf "$s"

# S5: uninstall with 'y' answer removes state dir too.
s="$(newenv)"
run_installer "$s" /dev/null
printf 'y\n' >"$s/in"
run_installer "$s" "$s/in" --uninstall
check "$RC" "S5 uninstall+state: exit 0"
if [[ ! -e "$s/state" ]]; then want "S5: state dir removed on 'y'"; else deny "S5: state dir kept despite 'y'"; fi
rm -rf "$s"

# S6: missing binary next to installer — must fail fast with a clear message.
s="$(newenv)"
cp "$INSTALLER" "$s/install.sh"   # copied WITHOUT the binary beside it
OUT="$(MAZZY_STATE_DIR="$s/state" MAZZY_RUN_DIR="$s/run" bash "$s/install.sh" --prefix "$s/usr" --no-color </dev/null 2>&1)"; RC=$?
check_not "$RC" "S6 missing-binary: non-zero exit"
grep -qi "Binary not found" <<<"$OUT"; check "$?" "S6: clear 'binary not found' message"
rm -rf "$s"

# S7: --help must exit 0 and print usage without side effects.
OUT="$(bash "$INSTALLER" --help 2>&1)"; RC=$?
check "$RC" "S7 --help: exit 0"
grep -qi "Usage" <<<"$OUT"; check "$?" "S7: usage shown"

# S8: unknown flag must fail with a clear message.
OUT="$(bash "$INSTALLER" --bogus 2>&1)"; RC=$?
check_not "$RC" "S8 unknown-flag: non-zero exit"
grep -qi "Unknown option" <<<"$OUT"; check "$?" "S8: clear unknown-option message"

# S9: custom prefix must NOT try to write /etc/systemd (no forced root).
s="$(newenv)"
run_installer "$s" /dev/null
if grep -q "/etc/systemd" <<<"$OUT"; then deny "S9: custom prefix touched /etc/systemd"; else want "S9: custom prefix skipped system systemd unit"; fi
rm -rf "$s"

# S10: idempotent uninstall when nothing is installed — must not abort.
s="$(newenv)"
run_installer "$s" /dev/null --uninstall
check "$RC" "S10 uninstall-when-absent: exit 0"
rm -rf "$s"

echo
echo "== result: $PASS passed, $FAIL failed =="
[[ "$FAIL" -eq 0 ]]
