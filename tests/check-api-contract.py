#!/usr/bin/env python3
"""Validate the public Mazzy VPN local API v1 contract with stdlib only."""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
API_DIR = ROOT / "api" / "v1"
MANIFEST_PATH = API_DIR / "manifest.json"
SCHEMA_PATH = API_DIR / "schema.json"
OPERATION_RE = re.compile(r"^[a-z][a-z0-9-]*\.[a-z][a-z0-9-]*$")


def fail(message: str) -> None:
    raise AssertionError(message)


def reject_duplicate_keys(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    value: dict[str, Any] = {}
    for key, item in pairs:
        if key in value:
            raise ValueError(f"duplicate JSON key: {key}")
        value[key] = item
    return value


def load_json(path: Path) -> dict[str, Any]:
    try:
        value = json.loads(
            path.read_text(encoding="utf-8"),
            object_pairs_hook=reject_duplicate_keys,
        )
    except (OSError, json.JSONDecodeError, ValueError) as error:
        fail(f"{path.relative_to(ROOT)} is not valid JSON: {error}")
    if not isinstance(value, dict):
        fail(f"{path.relative_to(ROOT)} must contain a JSON object")
    return value


def enum(schema_defs: dict[str, Any], name: str) -> list[str]:
    value = schema_defs.get(name, {}).get("enum")
    if not isinstance(value, list) or not all(isinstance(item, str) for item in value):
        fail(f"$defs/{name} must contain a string enum")
    return value


def property_names(value: Any) -> set[str]:
    names: set[str] = set()
    if isinstance(value, dict):
        properties = value.get("properties")
        if isinstance(properties, dict):
            names.update(properties)
        for nested in value.values():
            names.update(property_names(nested))
    elif isinstance(value, list):
        for nested in value:
            names.update(property_names(nested))
    return names


def forbidden_property_names(schema: dict[str, Any], forbidden: list[str]) -> set[str]:
    leaked: set[str] = set()
    for name in property_names(schema):
        for marker in forbidden:
            if (
                name == marker
                or name.startswith(f"{marker}_")
                or name.endswith(f"_{marker}")
            ):
                leaked.add(name)
    return leaked


def validate_manifest(manifest: dict[str, Any], schema: dict[str, Any]) -> None:
    if manifest.get("contract_id") != "com.mazurovn.mazzy-vpn.local-api":
        fail("unexpected contract_id")
    version = manifest.get("api_version")
    if not isinstance(version, str) or re.fullmatch(r"[1-9][0-9]*\.[0-9]+", version) is None:
        fail("api_version must use major.minor")
    if manifest.get("schema") != SCHEMA_PATH.name:
        fail("manifest schema path is not synchronized")
    if manifest.get("status") != "foundation":
        fail("the v1 slice must not claim that the protected service is complete")

    transports = {
        item.get("id"): item.get("status")
        for item in manifest.get("transports", [])
        if isinstance(item, dict)
    }
    if transports.get("cli-contract-metadata") != "implemented":
        fail("CLI contract metadata must be declared implemented")
    if transports.get("cli-json-adapter") != "partial":
        fail("the not-yet-unified CLI JSON adapter must remain partial")
    if transports.get("protected-local-service") != "partial":
        fail("the incremental Linux protected service must remain partial")

    domains = manifest.get("domains")
    operations = manifest.get("operations")
    if not isinstance(domains, list) or len(domains) != len(set(domains)):
        fail("domains must be a unique list")
    if not isinstance(operations, list) or not operations:
        fail("operations must be a non-empty list")

    query_names: list[str] = []
    mutation_names: list[str] = []
    seen_names: set[str] = set()
    authorization_classes: set[str] = set()
    required_fields = {
        "name",
        "domain",
        "kind",
        "authorization",
        "action_id_required",
        "deadline_required",
        "audit_event",
        "rollback",
    }
    for operation in operations:
        if not isinstance(operation, dict) or set(operation) != required_fields:
            fail(f"operation entry has unexpected fields: {operation!r}")
        name = operation["name"]
        if not isinstance(name, str) or OPERATION_RE.fullmatch(name) is None:
            fail(f"invalid operation name: {name!r}")
        if name in seen_names:
            fail(f"duplicate operation: {name}")
        seen_names.add(name)
        if operation["domain"] not in domains or name.split(".", 1)[0] != operation["domain"]:
            fail(f"operation domain is inconsistent: {name}")
        authorization_classes.add(operation["authorization"])

        if operation["kind"] == "query":
            query_names.append(name)
            if operation["action_id_required"] or operation["audit_event"]:
                fail(f"query cannot require action/audit state: {name}")
            if operation["rollback"] != "not-applicable":
                fail(f"query cannot declare rollback: {name}")
        elif operation["kind"] == "mutation":
            mutation_names.append(name)
            if not operation["action_id_required"]:
                fail(f"mutation must require action_id: {name}")
            if not operation["deadline_required"]:
                fail(f"mutation must require a deadline: {name}")
            if not operation["audit_event"]:
                fail(f"mutation must emit an audit event: {name}")
            if operation["authorization"] == "none":
                fail(f"mutation must declare authorization: {name}")
            if operation["rollback"] == "not-applicable":
                fail(f"mutation must declare rollback semantics: {name}")
        else:
            fail(f"unsupported operation kind: {operation['kind']!r}")

    defs = schema.get("$defs")
    if not isinstance(defs, dict):
        fail("schema must contain $defs")
    if defs.get("ApiVersion", {}).get("const") != version:
        fail("schema ApiVersion is not synchronized with the manifest")
    if set(enum(defs, "QueryOperation")) != set(query_names):
        fail("query operation enum is not synchronized with the manifest")
    if set(enum(defs, "MutationOperation")) != set(mutation_names):
        fail("mutation operation enum is not synchronized with the manifest")
    if not authorization_classes.issubset(set(enum(defs, "Authorization"))):
        fail("schema is missing a manifest authorization class")

    error_codes = manifest.get("error_codes")
    if not isinstance(error_codes, list) or len(error_codes) != len(set(error_codes)):
        fail("error_codes must be a unique list")
    if error_codes != enum(defs, "ErrorCode"):
        fail("error code enum is not synchronized with the manifest")

    event_types = enum(
        {"EventType": defs.get("EventEnvelope", {}).get("properties", {}).get("event_type", {})},
        "EventType",
    )
    if "audit.recorded" not in event_types or "AuditEvent" not in defs:
        fail("the schema must define sanitized audit events")

    status_properties = defs.get("PublicStatus", {}).get("properties", {})
    connection_properties = defs.get("ConnectionSummary", {}).get("properties", {})
    if not {
        "desired",
        "mode",
        "autostart",
        "health_monitor",
        "health_failures",
        "fallback",
    }.issubset(status_properties):
        fail("PublicStatus is missing optional CLI/TUI parity fields")
    if not {"interface", "handshake_age", "public_ip"}.issubset(
        connection_properties
    ):
        fail("ConnectionSummary is missing optional runtime detail fields")

    test_request = defs.get("TestRequest", {}).get("properties", {})
    concurrency = test_request.get("concurrency", {})
    if concurrency.get("minimum") != 1 or concurrency.get("maximum") != 8:
        fail("TestRequest must bound endpoint probe concurrency to 1..8")
    if test_request.get("include_speed", {}).get("type") != "boolean":
        fail("TestRequest must make the egress speed sample explicit")
    probe_result = defs.get("ProbeResult", {})
    probe_required = set(probe_result.get("required", []))
    if not {
        "profile_id",
        "display_name",
        "protocol",
        "active",
        "reachability",
        "latency_ms",
        "latency_source",
        "message_key",
    }.issubset(probe_required):
        fail("ProbeResult is missing per-location health fields")
    reachability = (
        probe_result.get("properties", {}).get("reachability", {}).get("enum")
    )
    if reachability != ["reachable", "unknown", "unreachable", "invalid"]:
        fail("ProbeResult must preserve unknown separately from unreachable")
    response_refs = {
        item.get("$ref")
        for item in defs.get("ResponseResult", {}).get("oneOf", [])
        if isinstance(item, dict)
    }
    if "#/$defs/ProbeCollection" not in response_refs:
        fail("ResponseResult does not expose the structured probe collection")
    if "#/$defs/EgressVerification" not in response_refs:
        fail("ResponseResult does not expose structured egress verification")
    if "#/$defs/ServiceEgressVerification" not in response_refs:
        fail("ResponseResult does not expose structured service-egress verification")
    if "#/$defs/ProtocolCatalog" not in response_refs:
        fail("ResponseResult does not expose the sanitized protocol catalog")
    if "#/$defs/PlannerEvaluation" not in response_refs:
        fail("ResponseResult does not expose deterministic planner evaluation")

    planner_request = defs.get("PlannerRequest", {})
    planner_request_properties = planner_request.get("properties", {})
    planner_candidates = planner_request_properties.get("candidates", {})
    if set(planner_request.get("required", [])) != {"workload", "candidates"}:
        fail("PlannerRequest must require workload and candidates only")
    if (
        planner_request.get("additionalProperties") is not False
        or set(planner_request_properties)
        != {"workload", "provider", "required_country", "candidates"}
        or planner_request_properties.get("provider", {}).get("$ref")
        != "#/$defs/Identifier"
        or planner_request_properties.get("required_country", {}).get("pattern")
        != "^[A-Z]{2}$"
        or planner_candidates.get("minItems") != 1
        or planner_candidates.get("maxItems") != 128
        or planner_candidates.get("items", {}).get("$ref")
        != "#/$defs/PlannerCandidateInput"
    ):
        fail("PlannerRequest must be strict and bound candidates to 1..128")

    query_request = defs.get("QueryRequest", {})
    planner_bindings = query_request.get("allOf")
    if not isinstance(planner_bindings, list) or len(planner_bindings) != 4:
        fail("QueryRequest must bind planner and service-egress operations")
    operation_binding, payload_binding, service_operation_binding, service_payload_binding = (
        planner_bindings
    )
    operation_then = operation_binding.get("then", {})
    operation_properties = operation_then.get("properties", {})
    planner_deadline = operation_properties.get("deadline_ms", {})
    if (
        operation_binding.get("if", {})
        .get("properties", {})
        .get("operation", {})
        .get("const")
        != "planner.evaluate"
        or operation_then.get("required") != ["deadline_ms"]
        or planner_deadline.get("minimum") != 100
        or planner_deadline.get("maximum") != 30000
        or operation_properties.get("payload", {}).get("$ref")
        != "#/$defs/PlannerRequest"
        or payload_binding.get("if", {})
        .get("properties", {})
        .get("payload", {})
        .get("$ref")
        != "#/$defs/PlannerRequest"
        or payload_binding.get("then", {})
        .get("properties", {})
        .get("operation", {})
        .get("const")
        != "planner.evaluate"
    ):
        fail("planner schema binding must be bidirectional and deadline-bounded")
    service_operation_then = service_operation_binding.get("then", {})
    service_operation_properties = service_operation_then.get("properties", {})
    if (
        service_operation_binding.get("if", {})
        .get("properties", {})
        .get("operation", {})
        .get("const")
        != "tests.verify-service-egress"
        or service_operation_then.get("required") != ["deadline_ms"]
        or service_operation_properties.get("deadline_ms", {}).get("minimum") != 100
        or service_operation_properties.get("deadline_ms", {}).get("maximum")
        != 32000
        or service_operation_properties.get("payload", {}).get("$ref")
        != "#/$defs/ServiceEgressRequest"
        or service_payload_binding.get("if", {})
        .get("properties", {})
        .get("payload", {})
        .get("$ref")
        != "#/$defs/ServiceEgressRequest"
        or service_payload_binding.get("then", {})
        .get("properties", {})
        .get("operation", {})
        .get("const")
        != "tests.verify-service-egress"
    ):
        fail("service-egress schema binding must be bidirectional and deadline-bounded")

    planner_evidence = defs.get("PlannerEvidence", {})
    required_evidence = {
        "recent_outcome",
        "consecutive_failures",
        "reachability",
        "latency_ms",
        "loss_percent",
        "evidence_age_seconds",
    }
    if (
        planner_evidence.get("additionalProperties") is not False
        or set(planner_evidence.get("required", [])) != required_evidence
        or set(planner_evidence.get("properties", {})) != required_evidence
    ):
        fail("PlannerEvidence must contain only bounded observed health evidence")

    planner_evaluation = defs.get("PlannerEvaluation", {})
    planner_evaluation_properties = planner_evaluation.get("properties", {})
    planner_evaluation_required = {
        "schema_version",
        "policy_version",
        "catalog_version",
        "evaluated_at",
        "dry_run",
        "workload",
        "provider",
        "required_country",
        "ordered_profile_ids",
        "candidates",
    }
    if (
        planner_evaluation.get("additionalProperties") is not False
        or set(planner_evaluation.get("required", []))
        != planner_evaluation_required
        or planner_evaluation_properties.get("dry_run", {}).get("const") is not True
        or planner_evaluation_properties.get("policy_version", {}).get("const") != 1
    ):
        fail("PlannerEvaluation must remain versioned, strict and dry-run only")

    planner_candidate = defs.get("PlannerCandidate", {})
    planner_candidate_properties = planner_candidate.get("properties", {})
    if (
        planner_candidate.get("additionalProperties") is not False
        or "display_name" in planner_candidate_properties
        or not {
            "profile_id",
            "eligible",
            "rank",
            "score",
            "hard_gates",
            "factors",
            "reason_codes",
        }.issubset(set(planner_candidate.get("required", [])))
        or planner_candidate_properties.get("hard_gates", {}).get("minItems") != 5
        or planner_candidate_properties.get("hard_gates", {}).get("maxItems") != 7
        or planner_candidate_properties.get("factors", {}).get("minItems") != 5
        or planner_candidate_properties.get("factors", {}).get("maxItems") != 5
    ):
        fail("PlannerCandidate must expose opaque IDs, bounded gates and five factors")

    verification = defs.get("EgressVerification", {})
    verification_required = set(verification.get("required", []))
    egress_fields = {
        "schema_version",
        "checked_at",
        "verdict",
        "message_key",
        "tunnel",
        "ipv4",
        "ipv6",
        "geo",
        "dns",
        "speed",
        "findings",
    }
    if (
        verification.get("additionalProperties") is not False
        or set(verification.get("properties", {})) != egress_fields
        or verification_required != egress_fields
    ):
        fail("EgressVerification v1 byte shape changed")
    if not {
        "verdict",
        "tunnel",
        "ipv4",
        "ipv6",
        "geo",
        "dns",
        "speed",
        "findings",
    }.issubset(verification_required):
        fail("EgressVerification is missing user-visible trust signals")
    speed_required = set(
        verification.get("properties", {})
        .get("speed", {})
        .get("required", [])
    )
    if not {"requested", "measured", "mbps", "connect_ms"}.issubset(
        speed_required
    ):
        fail("EgressVerification speed sample is not explicit and bounded")

    service_request = defs.get("ServiceEgressRequest", {})
    if (
        service_request.get("additionalProperties") is not False
        or set(service_request.get("required", [])) != {"service", "timeout_seconds"}
        or set(service_request.get("properties", {}))
        != {"service", "timeout_seconds"}
        or service_request.get("properties", {}).get("service", {}).get("enum")
        != ["notebooklm", "openai", "google", "antigravity", "all"]
        or service_request.get("properties", {})
        .get("timeout_seconds", {})
        .get("minimum")
        != 3
        or service_request.get("properties", {})
        .get("timeout_seconds", {})
        .get("maximum")
        != 15
    ):
        fail("ServiceEgressRequest must be strict, allowlisted and bounded")
    service_probe = defs.get("ServiceEgressProbe", {})
    service_probe_fields = {
        "service_id",
        "probe_version",
        "reachability",
        "egress_eligibility",
        "reason_code",
        "http_status",
    }
    service_probe_properties = service_probe.get("properties", {})
    expected_reasons = {
        "service.notebooklm.unsupported-location",
        "service.notebooklm.home-reached",
        "service.notebooklm.unrecognized-response",
        "service.openai.auth-boundary-reached",
        "service.openai.edge-denied",
        "service.openai.rate-limited",
        "service.openai.service-unavailable",
        "service.openai.unrecognized-response",
        "service.google.boundary-reached",
        "service.google.rate-limited",
        "service.google.service-unavailable",
        "service.google.unrecognized-response",
        "service.antigravity.boundary-reached",
        "service.antigravity.rate-limited",
        "service.antigravity.service-unavailable",
        "service.antigravity.unrecognized-response",
        "service.network-unreachable",
        "service.response-invalid",
        "service.response-too-large",
    }
    if (
        service_probe.get("additionalProperties") is not False
        or set(service_probe.get("required", [])) != service_probe_fields
        or set(service_probe_properties) != service_probe_fields
        or service_probe_properties.get("service_id", {}).get("enum")
        != ["notebooklm", "openai", "google", "antigravity"]
        or service_probe_properties.get("reachability", {}).get("enum")
        != ["reachable", "unreachable"]
        or service_probe_properties.get("egress_eligibility", {}).get("enum")
        != ["eligible", "ineligible", "indeterminate"]
        or set(service_probe_properties.get("reason_code", {}).get("enum", []))
        != expected_reasons
    ):
        fail("ServiceEgressProbe must expose only strict sanitized evidence")
    service_verification = defs.get("ServiceEgressVerification", {})
    service_results = service_verification.get("properties", {}).get("results", {})
    if (
        service_verification.get("additionalProperties") is not False
        or set(service_verification.get("required", []))
        != {"schema_version", "checked_at", "scope", "results"}
        or service_results.get("minItems") != 1
        or service_results.get("maxItems") != 4
        or service_results.get("uniqueItems") is not True
        or service_results.get("items", {}).get("$ref")
        != "#/$defs/ServiceEgressProbe"
    ):
        fail("ServiceEgressVerification must be strict and bounded to four services")

    security = manifest.get("security")
    if not isinstance(security, dict):
        fail("manifest security policy is missing")
    forbidden = security.get("frontend_forbidden_fields")
    if not isinstance(forbidden, list) or not all(isinstance(item, str) for item in forbidden):
        fail("frontend_forbidden_fields must be a string list")
    leaked = forbidden_property_names(schema, forbidden)
    if leaked:
        fail(f"frontend schema exposes forbidden fields: {sorted(leaked)}")


def validate_cli(manifest: dict[str, Any]) -> None:
    environment = os.environ.copy()
    environment.update(
        {
            "NO_COLOR": "1",
            "VPNCTL_ALLOW_UNPRIVILEGED": "1",
            "VPNCTL_API_DIR": str(API_DIR),
        }
    )
    result = subprocess.run(
        [str(ROOT / "mazzy-vpn"), "api-info", "--json"],
        cwd=ROOT,
        env=environment,
        check=False,
        capture_output=True,
        text=True,
        timeout=10,
    )
    if result.returncode != 0:
        fail(f"api-info --json failed: {result.stderr.strip()}")
    try:
        cli_manifest = json.loads(result.stdout)
    except json.JSONDecodeError as error:
        fail(f"api-info --json returned invalid JSON: {error}")
    if cli_manifest != manifest:
        fail("CLI api-info output differs from api/v1/manifest.json")


def validate_docs(version: str) -> None:
    for relative in ("docs/API_CONTRACT.en.md", "docs/API_CONTRACT.ru.md"):
        text = (ROOT / relative).read_text(encoding="utf-8")
        for required in (
            f"`api_version`",
            version,
            "`mazzy-vpn api-info --json`",
            "`cli-json-adapter`",
            "`planned`",
        ):
            if required not in text:
                fail(f"{relative} is missing contract marker {required!r}")


def main() -> int:
    manifest = load_json(MANIFEST_PATH)
    schema = load_json(SCHEMA_PATH)
    validate_manifest(manifest, schema)
    validate_cli(manifest)
    validate_docs(manifest["api_version"])
    print(
        "API contract OK: "
        f"{manifest['api_version']}, {len(manifest['operations'])} operations, "
        f"{len(manifest['error_codes'])} errors"
    )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (AssertionError, OSError, subprocess.SubprocessError) as error:
        print(f"API contract check failed: {error}", file=sys.stderr)
        raise SystemExit(1) from error
