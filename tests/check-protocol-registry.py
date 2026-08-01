#!/usr/bin/env python3
"""Validate protocol claims, support tiers and the agent orchestration policy."""

from __future__ import annotations

import json
import re
import subprocess
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
REGISTRY = ROOT / "protocols" / "v1" / "registry.json"
SCHEMA = ROOT / "protocols" / "v1" / "schema.json"
STATUSES = {"implemented", "partial", "planned"}
PLATFORMS = {"linux", "windows", "android"}
CURRENT = {"amneziawg", "wireguard", "openvpn", "l2tp"}
REQUESTED = {"vless", "hysteria2", "mieru", "naive"}
ANTI_CENSORSHIP = {"tuic", "shadowsocks2022", "trojan", "anytls", "shadowtls"}
HARD_CONSTRAINTS = [
    "backend-ready",
    "profile-valid",
    "secrets-readable-only-by-backend",
    "rollback-available",
    "platform-supported",
]
SCORE_FACTORS = [
    ("recent-success", 30),
    ("censorship-fit", 25),
    ("reachability", 20),
    ("latency-loss", 15),
    ("workload-fit", 10),
]


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
        fail("unsupported protocol registry schema")
    if set(registry.get("status_values", [])) != STATUSES:
        fail("protocol support statuses drifted")
    if set(registry.get("platforms", [])) != PLATFORMS:
        fail("protocol platform set drifted")

    protocols = registry.get("protocols")
    if not isinstance(protocols, list) or not 10 <= len(protocols) <= 15:
        fail("the target protocol catalog must contain 10..15 entries")
    by_id = {item.get("id"): item for item in protocols if isinstance(item, dict)}
    if len(by_id) != len(protocols) or None in by_id:
        fail("protocol IDs must be present and unique")
    required = CURRENT | REQUESTED | ANTI_CENSORSHIP
    if not required.issubset(by_id):
        fail(f"required protocol entries are missing: {sorted(required - set(by_id))}")

    schemes: dict[str, str] = {}
    for protocol_id, protocol in by_id.items():
        if re.fullmatch(r"[a-z][a-z0-9-]+", protocol_id) is None:
            fail(f"invalid protocol ID: {protocol_id}")
        support = protocol.get("support")
        if not isinstance(support, dict) or set(support) != {
            "detection", "import", "diagnostics", *PLATFORMS
        }:
            fail(f"{protocol_id}: incomplete support matrix")
        if not set(support.values()).issubset(STATUSES):
            fail(f"{protocol_id}: invalid support status")
        if protocol_id in CURRENT and support["linux"] != "implemented":
            fail(f"{protocol_id}: released Linux backend lost implemented status")
        if protocol_id not in CURRENT and support["linux"] == "implemented":
            fail(f"{protocol_id}: catalog overclaims an unverified Linux backend")
        if protocol.get("kind") in {"proxy", "transport"} and protocol_id not in CURRENT:
            if support["windows"] == "implemented" or support["android"] == "implemented":
                fail(f"{protocol_id}: unreleased native backend is marked implemented")
        for scheme in protocol.get("uri_schemes", []):
            if scheme in schemes:
                fail(f"URI scheme {scheme} is ambiguous")
            schemes[scheme] = protocol_id
        source = protocol.get("source", "")
        if not source.startswith("https://"):
            fail(f"{protocol_id}: primary source URL is missing")

    policy = registry.get("orchestration", {})
    if policy.get("policy_version") != 1:
        fail("unsupported orchestration policy version")
    weights = policy.get("score_factors", [])
    factor_pairs = [
        (item.get("id"), item.get("weight"))
        for item in weights
        if isinstance(item, dict)
    ]
    if factor_pairs != SCORE_FACTORS:
        fail("orchestration score factors drifted without a policy version change")
    if policy.get("hard_constraints") != HARD_CONSTRAINTS:
        fail("orchestration hard gates drifted without a policy version change")
    agent_rules = set(policy.get("agent_rules", []))
    if not {
        "agents-receive-opaque-profile-ids-only",
        "plans-are-dry-run-by-default",
        "mutations-require-authorized-action-id",
        "llm-output-never-becomes-a-shell-command",
        "credentials-never-enter-prompts-events-or-audit",
    }.issubset(agent_rules):
        fail("agent policy does not close the credential/command boundary")

    result = subprocess.run(
        [str(ROOT / "mazzy-vpn"), "protocols", "list", "--json"],
        cwd=ROOT,
        check=False,
        capture_output=True,
        text=True,
        timeout=10,
    )
    if result.returncode != 0 or json.loads(result.stdout) != registry:
        fail("CLI protocol registry output differs from source of truth")

    print(f"PROTOCOL REGISTRY OK: {len(protocols)} protocols, {len(schemes)} share URI schemes")


if __name__ == "__main__":
    main()
