#!/usr/bin/env python3
"""Validate the reverse agent-control contract without claiming runtime support."""

from __future__ import annotations

import json
import subprocess
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
CONTRACT_DIR = ROOT / "agent-control" / "v1"
REGISTRY_PATH = CONTRACT_DIR / "registry.json"
TRANSPORTS = {
    "lan-wss",
    "iroh-quic",
    "libp2p-quic",
    "webrtc-datachannel",
    "webtransport",
    "tailscale-headscale",
    "reverse-wss-broker",
}
CHANNELS = {
    "desktop-first-party",
    "web-first-party",
    "cli-first-party",
    "telegram-bot",
    "telegram-mini-app",
}
PLATFORMS = {"linux", "windows", "macos", "android", "web"}
STATUSES = {"implemented", "partial", "planned"}


def fail(message: str) -> None:
    raise AssertionError(message)


def reject_duplicate_keys(pairs: list[tuple[str, object]]) -> dict[str, object]:
    result: dict[str, object] = {}
    for key, value in pairs:
        if key in result:
            fail(f"duplicate JSON key: {key}")
        result[key] = value
    return result


def load(path: Path) -> dict[str, object]:
    value = json.loads(
        path.read_text(encoding="utf-8"),
        object_pairs_hook=reject_duplicate_keys,
    )
    if not isinstance(value, dict):
        fail(f"{path.relative_to(ROOT)} is not an object")
    return value


def validate_support(owner: str, support: object) -> None:
    required = {"contract", "diagnostics", *PLATFORMS}
    if not isinstance(support, dict) or set(support) != required:
        fail(f"{owner}: incomplete support matrix")
    if not set(support.values()).issubset(STATUSES):
        fail(f"{owner}: invalid support status")
    if any(support[platform] == "implemented" for platform in PLATFORMS):
        fail(f"{owner}: runtime is overclaimed before an audited implementation exists")


def main() -> None:
    registry = load(REGISTRY_PATH)
    schema = load(CONTRACT_DIR / "schema.json")
    envelope = load(CONTRACT_DIR / "envelope.schema.json")
    command = load(CONTRACT_DIR / "command.schema.json")
    if schema.get("additionalProperties") is not False:
        fail("agent-control registry schema is not closed")
    if envelope.get("additionalProperties") is not False:
        fail("agent-control envelope schema is not closed")
    if command.get("additionalProperties") is not False:
        fail("agent-control command schema is not closed")
    if registry.get("schema_version") != 1:
        fail("unsupported agent-control registry schema")
    if registry.get("layer") != "reverse-agent-control":
        fail("agent-control registry is mixed with the VPN data plane")
    if set(registry.get("platforms", [])) != PLATFORMS:
        fail("agent-control platform set drifted")
    if set(registry.get("status_values", [])) != STATUSES:
        fail("agent-control status values drifted")

    transports = registry.get("transports")
    if not isinstance(transports, list):
        fail("agent-control transports are missing")
    by_id = {item.get("id"): item for item in transports if isinstance(item, dict)}
    if set(by_id) != TRANSPORTS or len(by_id) != len(transports):
        fail("agent-control transport catalog is incomplete or has duplicate IDs")
    for transport_id, transport in by_id.items():
        validate_support(transport_id, transport.get("support"))
        if transport.get("support", {}).get("contract") != "implemented":
            fail(f"{transport_id}: machine-readable contract regressed")
        source = transport.get("source")
        if not isinstance(source, str) or not source.startswith("https://"):
            fail(f"{transport_id}: primary source URL is missing")

    channels = registry.get("ingress_channels")
    if not isinstance(channels, list):
        fail("agent-control ingress channels are missing")
    channels_by_id = {
        item.get("id"): item for item in channels if isinstance(item, dict)
    }
    if set(channels_by_id) != CHANNELS or len(channels_by_id) != len(channels):
        fail("agent-control ingress catalog is incomplete or has duplicate IDs")
    for channel_id, channel in channels_by_id.items():
        validate_support(channel_id, channel.get("support"))
    desktop = channels_by_id["desktop-first-party"]
    if desktop.get("support", {}).get("diagnostics") != "implemented":
        fail("Desktop agent integration diagnostics regressed")
    if desktop.get("support", {}).get("linux") != "partial":
        fail("Desktop agent integration overclaims or hides its partial Linux slice")
    if desktop.get("risk_ceiling") != "low":
        fail("Diagnostics-only Desktop ingress exceeds its current low-risk ceiling")
    telegram = channels_by_id["telegram-bot"]
    if telegram.get("risk_ceiling") != "low":
        fail("Telegram Bot may authorize actions above the low-risk ceiling")
    if telegram.get("e2ee_mode") != "gateway-visible":
        fail("Telegram Bot incorrectly claims first-party E2EE")

    policy = registry.get("orchestration")
    if not isinstance(policy, dict) or policy.get("policy_version") != 1:
        fail("unsupported agent-control orchestration policy")
    priority = policy.get("path_priority")
    if not isinstance(priority, list) or set(priority) != TRANSPORTS:
        fail("transport priority does not cover the complete catalog")
    if priority != [
        "reverse-wss-broker",
        "lan-wss",
        "iroh-quic",
        "tailscale-headscale",
        "webrtc-datachannel",
        "webtransport",
        "libp2p-quic",
    ]:
        fail("transport priority contradicts the reverse-HTTPS-first architecture")
    constraints = set(policy.get("hard_constraints", []))
    if not {
        "runtime-ready",
        "paired-peer-authorized",
        "e2ee-envelope-valid",
        "anti-replay-valid",
        "channel-risk-policy-valid",
        "no-vpn-route-loop",
    }.issubset(constraints):
        fail("agent-control hard constraints are incomplete")
    channel_rules = set(policy.get("channel_rules", []))
    if "arbitrary-shell-execution-is-not-a-v1-capability" not in channel_rules:
        fail("agent-control policy permits an arbitrary shell boundary")
    if (
        "desktop-provider-actions-disabled-until-native-approval-and-process-containment"
        not in channel_rules
    ):
        fail("agent-control policy does not contain executable Desktop authority")

    capabilities = command.get("properties", {}).get("capability", {}).get("enum", [])
    if any("shell" in item or "exec" in item for item in capabilities):
        fail("agent-control v1 exposes arbitrary process execution")
    if "iroh" in {
        item.get("id")
        for item in load(ROOT / "protocols" / "v1" / "registry.json").get(
            "protocols", []
        )
        if isinstance(item, dict)
    }:
        fail("iroh was incorrectly added to the VPN protocol catalog")

    result = subprocess.run(
        [str(ROOT / "mazzy-vpn"), "agent-transports", "list", "--json"],
        cwd=ROOT,
        check=False,
        capture_output=True,
        text=True,
        timeout=10,
    )
    if result.returncode != 0 or json.loads(result.stdout) != registry:
        fail("CLI agent-control registry output differs from source of truth")
    diagnosis = subprocess.run(
        [str(ROOT / "mazzy-vpn"), "agent-transports", "diagnose", "--json"],
        cwd=ROOT,
        check=False,
        capture_output=True,
        text=True,
        timeout=10,
    )
    if diagnosis.returncode != 0:
        fail("CLI agent transport diagnostics failed")
    diagnosed = json.loads(diagnosis.stdout)
    if any(item.get("runtime_ready") for item in diagnosed.get("transports", [])):
        fail("agent transport diagnostics overclaim runtime readiness")

    print(
        f"AGENT CONTROL REGISTRY OK: {len(transports)} transports, "
        f"{len(channels)} ingress channels"
    )


if __name__ == "__main__":
    main()
