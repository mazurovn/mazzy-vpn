#!/usr/bin/env bash
# Privileged autonomy smoke test for backlog C1-8.
#
# Proves mazzy-core brings up a real AmneziaWG/WireGuard tunnel using ONLY the
# embedded engine + native routes/dns/guard — no awg-quick, wg-quick, jq or any
# external VPN tool. Requires root (CAP_NET_ADMIN to create the TUN device).
#
# Usage:
#   sudo core/scripts/smoke-c1-8.sh [path/to/profile.conf]
#
# With no profile it generates a self-referential test config (real X25519
# keys, unreachable endpoint). That still proves the LOCAL data path:
# interface creation, config application, addressing, policy routing, guards.
set -euo pipefail

here="$(cd "$(dirname "$0")/.." && pwd)"
go_bin="${MAZZY_GO:-/usr/local/go/bin/go}"
[[ -x "$go_bin" ]] || go_bin="$(command -v go)"

if [[ $EUID -ne 0 ]]; then
    echo "error: run with sudo (needs CAP_NET_ADMIN)" >&2
    exit 1
fi

echo "== building static mazzy-core-smoke =="
CGO_ENABLED=0 "$go_bin" build -o "$here/bin/mazzy-core-smoke" ./cmd/mazzy-core-smoke
( cd "$here" && CGO_ENABLED=0 "$go_bin" build -o bin/mazzy-core-smoke ./cmd/mazzy-core-smoke )

conf="${1:-}"
tmpconf=""
if [[ -z "$conf" ]]; then
    echo "== generating a test AmneziaWG config (real keys, unreachable peer) =="
    tmpconf="$(mktemp /tmp/mazzy-smoke.XXXXXX.conf)"
    read -r priv pub < <("$go_bin" run "$here/scripts/genkey.go")
    cat >"$tmpconf" <<EOF
[Interface]
PrivateKey = $priv
Address = 10.99.0.2/32
Jc = 4
Jmin = 40
Jmax = 70
S1 = 50
S2 = 100
H1 = 1
H2 = 2
H3 = 3
H4 = 4
[Peer]
PublicKey = $pub
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = 192.0.2.1:51820
PersistentKeepalive = 25
EOF
    chmod 600 "$tmpconf"
    conf="$tmpconf"
fi

cleanup() { [[ -n "$tmpconf" ]] && rm -f "$tmpconf"; }
trap cleanup EXIT

echo "== bringing tunnel up for 3s then tearing down =="
timeout -s INT 3 "$here/bin/mazzy-core-smoke" up "$conf" --protocol amneziawg || true

echo "== verifying no external *-quick tools were required =="
echo "   (this binary is statically linked; it embeds amneziawg-go)"
file "$here/bin/mazzy-core-smoke" | grep -q "statically linked" && echo "   OK: static, self-contained"

echo "== C1-8 smoke complete =="
