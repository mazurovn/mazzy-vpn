#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cli_version="$(sed -n 's/^VERSION="\([^"]*\)"/\1/p' "$ROOT/mazzy-vpn" | head -1)"
desktop_version="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["version"])' "$ROOT/desktop/src-tauri/tauri.conf.json")"
cargo_version="$(sed -n 's/^version = "\([^"]*\)"/\1/p' "$ROOT/desktop/src-tauri/Cargo.toml" | head -1)"
[[ -n "$cli_version" && "$desktop_version" == "$cargo_version" ]] || {
  echo "release-provenance: version mismatch cli=$cli_version desktop=$desktop_version cargo=$cargo_version" >&2
  exit 1
}
for path in "$ROOT/systemd/user/mazzy-agentd.service" "$ROOT/agent-control/v1/registry.json" "$ROOT/agent-control/v1/command.schema.json" "$ROOT/providers/v1/registry.json"; do
  [[ -s "$path" ]] || { echo "release-provenance: missing $path" >&2; exit 1; }
done
grep -Fq 'ExecStart=/usr/bin/mazzy-agentd' "$ROOT/systemd/user/mazzy-agentd.service" || {
  echo "release-provenance: agentd unit is not package-owned" >&2; exit 1;
}
echo "RELEASE PROVENANCE OK: cli=$cli_version desktop=$desktop_version agentd=packaged"
