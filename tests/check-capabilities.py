#!/usr/bin/env python3
"""Validate the Mazzy VPN cross-surface capability and release-gate registry."""

# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later

from __future__ import annotations

import json
from pathlib import Path
import sys


ROOT = Path(__file__).resolve().parents[1]
REGISTRY = ROOT / "docs" / "capabilities.json"
PARITY_DOC = ROOT / "docs" / "FEATURE_PARITY.md"


def fail(message: str) -> None:
    print(f"CAPABILITY MATRIX ERROR: {message}", file=sys.stderr)
    raise SystemExit(1)


def main() -> None:
    data = json.loads(REGISTRY.read_text(encoding="utf-8"))
    if data.get("schema_version") != 1:
        fail("schema_version must be 1")

    surfaces = data.get("surfaces")
    statuses = set(data.get("status_values", []))
    if not isinstance(surfaces, list) or len(surfaces) != len(set(surfaces)):
        fail("surfaces must be a unique list")
    if statuses != {"implemented", "partial", "planned", "not-applicable"}:
        fail("status_values are incomplete")

    capabilities = data.get("capabilities")
    if not isinstance(capabilities, list) or not capabilities:
        fail("capabilities must be a non-empty list")

    by_id: dict[str, dict] = {}
    parity_text = PARITY_DOC.read_text(encoding="utf-8")
    for capability in capabilities:
        capability_id = capability.get("id")
        if not isinstance(capability_id, str) or not capability_id:
            fail("every capability needs an id")
        if capability_id in by_id:
            fail(f"duplicate capability id: {capability_id}")
        by_id[capability_id] = capability

        states = capability.get("surfaces")
        if not isinstance(states, dict) or set(states) != set(surfaces):
            fail(f"{capability_id}: surface set does not match registry")
        invalid = set(states.values()) - statuses
        if invalid:
            fail(f"{capability_id}: invalid statuses {sorted(invalid)}")
        if f"`{capability_id}`" not in parity_text:
            fail(f"{capability_id}: missing from FEATURE_PARITY.md")

        for field in ("tests", "docs"):
            references = capability.get(field)
            if not isinstance(references, list) or not references:
                fail(f"{capability_id}: {field} must be a non-empty list")
            for reference in references:
                path = reference.split("#", 1)[0]
                if not (ROOT / path).exists():
                    fail(f"{capability_id}: missing {field} reference {reference}")

    gate_ids: set[str] = set()
    for gate in data.get("release_gates", []):
        gate_id = gate.get("id")
        if not isinstance(gate_id, str) or not gate_id or gate_id in gate_ids:
            fail(f"invalid or duplicate release gate: {gate_id}")
        gate_ids.add(gate_id)
        if f"`{gate_id}`" not in parity_text:
            fail(f"{gate_id}: missing from FEATURE_PARITY.md")

        computed_ready = True
        requirements = gate.get("requirements")
        if not isinstance(requirements, list) or not requirements:
            fail(f"{gate_id}: requirements must be a non-empty list")
        for requirement in requirements:
            capability_id = requirement.get("capability")
            if capability_id not in by_id:
                fail(f"{gate_id}: unknown capability {capability_id}")
            required_surfaces = requirement.get("surfaces")
            if not isinstance(required_surfaces, list) or not required_surfaces:
                fail(f"{gate_id}: empty surfaces for {capability_id}")
            for surface in required_surfaces:
                if surface not in surfaces:
                    fail(f"{gate_id}: unknown surface {surface}")
                if by_id[capability_id]["surfaces"][surface] != "implemented":
                    computed_ready = False

        if gate.get("declared_ready") is not computed_ready:
            fail(
                f"{gate_id}: declared_ready={gate.get('declared_ready')} "
                f"but computed readiness is {computed_ready}"
            )

    print(
        "CAPABILITY MATRIX OK: "
        f"{len(capabilities)} capabilities, {len(gate_ids)} release gates"
    )


if __name__ == "__main__":
    main()
