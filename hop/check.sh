#!/bin/sh
# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright © 2026 Nik m (@mazurovn). All rights reserved.
#
# Local check for the hop. CI does not require the Nim, MoonBit, or Mojo
# toolchains. Missing tools are an error here, because this script is the
# manual gate.

set -eu
cd "$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"

if [ -n "${HOME:-}" ] && [ -x "${HOME}/.moon/bin/moon" ]; then
  PATH="${HOME}/.moon/bin:${PATH}"
fi

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'missing %s\n' "$1" >&2
    exit 1
  fi
}
need moon
need mojo
need nim
need go

( cd hop/toeac && moon test --target native && moon build --target native )
( cd hop/front && moon test --target native && moon build --target native )

mojo build hop/protect/protect.mojo -o /tmp/mazzy-hop-protect
printf 'decide 1 1 1 0 2\n' | /tmp/mazzy-hop-protect | grep -qx open
printf 'decide 0 1 1 0 2\n' | /tmp/mazzy-hop-protect | grep -qx refuse
printf 'trace established 3 4 5\n' | /tmp/mazzy-hop-protect | grep -qx 'status=established latency_ms=3 bytes_up=4 bytes_down=5 egress_clean=false'
printf 'trace nope 1 1 1\n' | /tmp/mazzy-hop-protect | grep -qx 'status=refused latency_ms=1 bytes_up=1 bytes_down=1 egress_clean=false'

nim c -d:ssl --nimcache:/tmp/mazzy-nimcache -o:/tmp/mazzy-core --hints:off hop/core/mazzy_core.nim
code=0
/tmp/mazzy-core trust-config /tmp/no-such-hop.toeac >/dev/null 2>&1 || code=$?
if [ "$code" -ne 78 ]; then
  printf 'trust-config exit %s\n' "$code" >&2
  exit 1
fi

(
  cd hop/backend
  fmt=$(gofmt -l .)
  if [ -n "$fmt" ]; then
    printf '%s\n' "$fmt" >&2
    exit 1
  fi
  go test -count=1 -timeout 180s .
)
