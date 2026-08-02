#!/usr/bin/env python3
"""Validate modern-protocol runtime plans without overclaiming lifecycle support."""

from __future__ import annotations

import json
import subprocess
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
REGISTRY = ROOT / "runtime" / "v1" / "adapter-registry.json"
SCHEMA = ROOT / "runtime" / "v1" / "schema.json"
EXPECTED_PROTOCOLS = {
    "vless",
    "hysteria2",
    "mieru",
    "naive",
    "tuic",
    "shadowsocks2022",
    "trojan",
    "anytls",
    "shadowtls",
}
SING_BOX_RENDERED = {
    "vless",
    "hysteria2",
    "tuic",
    "shadowsocks2022",
    "trojan",
    "anytls",
}


def fail(message: str) -> None:
    raise AssertionError(message)


def load(path: Path) -> dict:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        fail(f"{path.relative_to(ROOT)} is not an object")
    return value


def main() -> None:
    registry = load(REGISTRY)
    load(SCHEMA)
    if registry.get("schema_version") != 1:
        fail("unsupported runtime adapter registry schema")
    adapters = registry.get("adapters", [])
    by_id = {adapter.get("id"): adapter for adapter in adapters}
    if len(by_id) != 4 or len(by_id) != len(adapters):
        fail("runtime adapter IDs are missing, duplicated or unexpected")
    protocol_owners: dict[str, str] = {}
    for adapter_id, adapter in by_id.items():
        if adapter.get("engine_bundled") is not False:
            fail(f"{adapter_id}: unverified engine is claimed as bundled")
        support = adapter.get("support", {})
        if support.get("lifecycle") != "planned" or support.get("integration_tests") != "planned":
            fail(f"{adapter_id}: unaudited lifecycle is overclaimed")
        if not str(adapter.get("source", "")).startswith("https://"):
            fail(f"{adapter_id}: authoritative source is missing")
        for protocol in adapter.get("protocols", []):
            if protocol in protocol_owners:
                fail(f"{protocol}: owned by multiple runtime adapters")
            protocol_owners[protocol] = adapter_id
    if set(protocol_owners) != EXPECTED_PROTOCOLS:
        fail("runtime adapter protocol coverage differs from managed profile coverage")
    sing_box = by_id.get("sing-box-managed-v1", {})
    if set(sing_box.get("protocols", [])) != SING_BOX_RENDERED:
        fail("sing-box renderer protocol set drifted")
    if sing_box.get("required_version") != "1.13.12":
        fail("sing-box adapter version and executable pin differ")
    if sing_box.get("support", {}).get("engine_config_render") != "implemented":
        fail("tested sing-box renderer lost implemented component status")
    for adapter_id in {"mieru-sidecar-v1", "naive-sidecar-v1", "shadowtls-chain-v1"}:
        if by_id[adapter_id]["support"]["engine_config_render"] != "planned":
            fail(f"{adapter_id}: missing supervisor or chain is overclaimed")
    if "iroh" in protocol_owners:
        fail("iroh was incorrectly mixed into the egress VPN runtime registry")
    required_gates = set(registry.get("hard_release_gates", []))
    if not {
        "rollback-restores-route-dns-firewall-and-previous-tunnel",
        "network-namespace-ipv4-ipv6-dns-and-process-crash-tests-pass",
        "two-process-failure-is-atomic-for-sidecar-bridges",
    }.issubset(required_gates):
        fail("runtime release gates omit rollback or sidecar failure testing")

    result = subprocess.run(
        [str(ROOT / "mazzy-vpn"), "protocols", "adapters", "--json"],
        cwd=ROOT,
        check=False,
        capture_output=True,
        text=True,
        timeout=10,
    )
    if result.returncode != 0 or json.loads(result.stdout) != registry:
        fail("CLI runtime adapter output differs from source of truth")

    print("RUNTIME ADAPTER REGISTRY OK: 9 protocols, 4 truthful execution graphs")


if __name__ == "__main__":
    main()
