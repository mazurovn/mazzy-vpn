#!/usr/bin/env bash
# Build mazzy-core with a pinned Go toolchain (backlog C1-1a).
#
# amneziawg-go/v3 requires Go >= 1.25. The default `go` on PATH may be older
# (e.g. 1.22), which fails silently. This script selects a 1.25+ toolchain.
set -euo pipefail

need_major=1
need_minor=25

pick_go() {
    # Prefer an explicit MAZZY_GO, then a known 1.25 install, then PATH.
    local candidates=("${MAZZY_GO:-}" /usr/local/go/bin/go)
    candidates+=("$(command -v go 2>/dev/null || true)")
    for g in "${candidates[@]}"; do
        [[ -x "$g" ]] || continue
        local v
        v="$("$g" env GOVERSION 2>/dev/null || "$g" version 2>/dev/null)"
        # Extract goX.Y
        if [[ "$v" =~ go([0-9]+)\.([0-9]+) ]]; then
            local maj=${BASH_REMATCH[1]} min=${BASH_REMATCH[2]}
            if (( maj > need_major || (maj == need_major && min >= need_minor) )); then
                printf '%s\n' "$g"
                return 0
            fi
        fi
    done
    return 1
}

GO="$(pick_go)" || {
    echo "error: need Go >= ${need_major}.${need_minor} (amneziawg-go). Set MAZZY_GO." >&2
    exit 1
}

echo "Using Go: $GO ($("$GO" version))"
cd "$(dirname "$0")"

# Static, self-contained: no external runtime libraries.
export CGO_ENABLED=0
"$GO" build ./...
"$GO" test ./...
"$GO" build -o ./bin/engine-selftest ./cmd/engine-selftest
echo "OK: mazzy-core built statically at ./bin/engine-selftest"
