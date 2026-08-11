#!/usr/bin/env python3
"""Validate reverse agent-control contracts and audited LAN-WSS Linux support."""

from __future__ import annotations

import json
import os
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
    implemented = {
        platform for platform in PLATFORMS if support[platform] == "implemented"
    }
    if implemented and not (owner == "lan-wss" and implemented == {"linux"}):
        fail(f"{owner}: runtime is overclaimed beyond audited LAN-WSS Linux support")


def main() -> None:
    registry = load(REGISTRY_PATH)
    schema = load(CONTRACT_DIR / "schema.json")
    envelope = load(CONTRACT_DIR / "envelope.schema.json")
    command = load(CONTRACT_DIR / "command.schema.json")
    ack = load(CONTRACT_DIR / "ack.schema.json")
    result_schema = load(CONTRACT_DIR / "result.schema.json")
    event = load(CONTRACT_DIR / "event.schema.json")
    error = load(CONTRACT_DIR / "error.schema.json")
    pairing = load(CONTRACT_DIR / "pairing.schema.json")
    approval = load(CONTRACT_DIR / "approval.schema.json")
    approval_request = load(CONTRACT_DIR / "approval-request.schema.json")
    transport_error = load(CONTRACT_DIR / "transport-error.schema.json")
    user_unit = (ROOT / "systemd" / "user" / "mazzy-agentd.service").read_text(
        encoding="utf-8"
    )
    if schema.get("additionalProperties") is not False:
        fail("agent-control registry schema is not closed")
    if envelope.get("additionalProperties") is not False:
        fail("agent-control envelope schema is not closed")
    if command.get("additionalProperties") is not False:
        fail("agent-control command schema is not closed")
    for name, contract in {
        "ack": ack,
        "result": result_schema,
        "event": event,
        "error": error,
        "pairing": pairing,
        "approval": approval,
        "approval-request": approval_request,
        "transport-error": transport_error,
    }.items():
        if contract.get("additionalProperties") is not False:
            fail(f"agent-control {name} schema is not closed")
    if registry.get("schema_version") != 1:
        fail("unsupported agent-control registry schema")
    for required_unit_line in {
        "UMask=0077",
        "NoNewPrivileges=yes",
        "ProtectSystem=strict",
        "ProtectHome=read-only",
        "RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6",
        "ConditionPathExists=%h/.config/mazzy-agentd/server.key",
    }:
        if required_unit_line not in user_unit:
            fail(f"mazzy-agentd user service lost hardening: {required_unit_line}")
    if "--listen 127.0.0.1" not in user_unit or "--api-socket /run/mazzy-vpn/api-v1.sock" not in user_unit:
        fail("mazzy-agentd user service exposes an unsafe or untyped boundary")
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
    lan = by_id["lan-wss"]
    if lan.get("runtime_probes") != ["mazzy-agentd"]:
        fail("lan-wss: runtime probe is not the owned daemon")
    if lan.get("support", {}).get("diagnostics") != "implemented":
        fail("lan-wss: implemented Linux slice lacks diagnostics")
    evidence = lan.get("evidence")
    if evidence != {
        "id": "agentd-lan-wss-linux-2026-08-11",
        "package_version": "0.1.0",
        "test_matrix": "tests/check-agentd.py",
        "reviewed_at": "2026-08-11",
    }:
        fail("lan-wss: implementation evidence is missing or inconsistent")
    if any(
        transport.get("evidence") is not None
        for transport_id, transport in by_id.items()
        if transport_id != "lan-wss"
    ):
        fail("planned transport carries unearned implementation evidence")

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
        "end-to-end-channel-valid",
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
    command_rules = command.get("allOf")
    if not isinstance(command_rules, list):
        fail("agent-control command policy is missing")
    serialized_rules = json.dumps(command_rules, sort_keys=True)
    if "^approval-[a-f0-9]{32}$" not in serialized_rules:
        fail("high-risk command schema does not require a trusted approval proof id")
    if approval_request.get("properties", {}).get("message_type", {}).get(
        "const"
    ) != "approval-request" or "confirmation_id" in approval_request.get(
        "properties", {}
    ).get("confirmation", {}).get("properties", {}):
        fail("approval request is not a distinct proof-free local contract")
    for capability, required_risk in {
        "vpn.status": "read-only",
        "vpn.select": "low",
        "vpn.connect": "high",
        "vpn.disconnect": "high",
        "vpn.verify": "read-only",
        "planner.evaluate": "read-only",
        "region.check": "read-only",
    }.items():
        if capability not in serialized_rules or required_risk not in serialized_rules:
            fail(f"{capability}: command schema does not pin its risk")
    if set(capabilities) != {
        "vpn.status", "vpn.select", "vpn.connect", "vpn.disconnect",
        "vpn.verify", "planner.evaluate", "region.check",
    }:
        fail("egress command schema and mazzy-agentd capability surface drifted")
    planner_agent_bound = next(
        (
            rule.get("then", {})
            .get("properties", {})
            .get("arguments", {})
            .get("allOf")
            for rule in command_rules
            if rule.get("if", {})
            .get("properties", {})
            .get("capability", {})
            .get("const")
            == "planner.evaluate"
        ),
        None,
    )
    if (
        not isinstance(planner_agent_bound, list)
        or not any(
            item.get("properties", {}).get("candidates", {}).get("maxItems") == 16
            for item in planner_agent_bound
            if isinstance(item, dict)
        )
    ):
        fail("agent planner input is not bounded below the LAN-WSS response cap")
    source_channels = set(
        command.get("properties", {}).get("source_channel", {}).get("enum", [])
    )
    if "telegram-bot" in source_channels:
        fail("gateway-visible Telegram Bot entered the direct egress command protocol")
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
        env={**os.environ, "PATH": f"{ROOT}:{os.environ.get('PATH', '')}"},
        check=False,
        capture_output=True,
        text=True,
        timeout=10,
    )
    if diagnosis.returncode != 0:
        fail("CLI agent transport diagnostics failed")
    diagnosed = json.loads(diagnosis.stdout)
    ready = {
        item.get("id")
        for item in diagnosed.get("transports", [])
        if item.get("runtime_ready")
    }
    if ready:
        fail("unconfigured agent transport was incorrectly reported runtime-ready")
    runtime_by_id = {
        item.get("id"): item for item in diagnosed.get("runtimes", [])
        if isinstance(item, dict)
    }
    agentd_runtime = runtime_by_id.get("mazzy-agentd", {})
    if agentd_runtime.get("available") is not True or agentd_runtime.get(
        "ready"
    ) is not False:
        fail("agent transport diagnostics lost installed-vs-configured readiness")
    diagnostics = agentd_runtime.get("diagnostics")
    if not isinstance(diagnostics, dict) or diagnostics.get("runtime_ready") is not False:
        fail("mazzy-agentd diagnostics are absent or overclaim readiness")

    print(
        f"AGENT CONTROL REGISTRY OK: {len(transports)} transports, "
        f"{len(channels)} ingress channels"
    )


if __name__ == "__main__":
    main()
