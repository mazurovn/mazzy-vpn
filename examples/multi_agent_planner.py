#!/usr/bin/env python3
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
"""Coordinate redacted worker reports into one Mazzy VPN dry-run evaluation.

Input shape:
{
  "workload": "llm-streaming",
  "provider": "antigravity",             # optional
  "required_country": "AT",              # optional
  "workers": [
    {"worker_id": "probe-eu", "candidates": [{"profile_id": "...", "evidence": {...}}]}
  ]
}

Workers must report distinct opaque profile IDs. Worker IDs are validated for
coordination diagnostics and are never sent to the privileged local API.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

from mazzy_planner_client import (
    DEFAULT_SOCKET,
    PlannerClient,
    PlannerClientError,
    read_bounded_payload,
)


IDENTIFIER = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$")
PROVIDER_ID = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._:-]{1,63}$")
COUNTRY_CODE = re.compile(r"^[A-Z]{2}$")
WORKLOADS = {"general", "llm-streaming", "api-calls", "video", "split-routing"}


def build_planner_payload(report: dict[str, Any]) -> dict[str, Any]:
    required_keys = {"workload", "workers"}
    allowed_keys = required_keys | {"provider", "required_country"}
    if not required_keys.issubset(report) or not set(report).issubset(allowed_keys):
        raise PlannerClientError(
            "coordinator input must contain workload/workers and only optional "
            "provider/required_country"
        )
    workload = report["workload"]
    workers = report["workers"]
    if workload not in WORKLOADS:
        raise PlannerClientError("coordinator workload is unsupported")
    if not isinstance(workers, list) or not workers:
        raise PlannerClientError("coordinator requires at least one worker report")

    worker_ids: set[str] = set()
    profile_ids: set[str] = set()
    candidates: list[dict[str, Any]] = []
    for worker in workers:
        if not isinstance(worker, dict) or set(worker) != {"worker_id", "candidates"}:
            raise PlannerClientError("each worker must contain worker_id and candidates")
        worker_id = worker["worker_id"]
        worker_candidates = worker["candidates"]
        if not isinstance(worker_id, str) or IDENTIFIER.fullmatch(worker_id) is None:
            raise PlannerClientError("worker_id is not a safe opaque identifier")
        if worker_id in worker_ids:
            raise PlannerClientError(f"duplicate worker_id: {worker_id}")
        worker_ids.add(worker_id)
        if not isinstance(worker_candidates, list) or not worker_candidates:
            raise PlannerClientError(f"worker {worker_id} returned no candidates")

        for candidate in worker_candidates:
            if not isinstance(candidate, dict) or set(candidate) != {
                "profile_id",
                "evidence",
            }:
                raise PlannerClientError(
                    f"worker {worker_id} returned an invalid candidate shape"
                )
            profile_id = candidate["profile_id"]
            if not isinstance(profile_id, str) or IDENTIFIER.fullmatch(profile_id) is None:
                raise PlannerClientError(
                    f"worker {worker_id} returned an invalid opaque profile ID"
                )
            if profile_id in profile_ids:
                raise PlannerClientError(f"duplicate profile_id across workers: {profile_id}")
            profile_ids.add(profile_id)
            candidates.append(candidate)

    if len(candidates) > 128:
        raise PlannerClientError("coordinator produced more than 128 candidates")
    payload: dict[str, Any] = {"workload": workload, "candidates": candidates}
    if "provider" in report:
        provider = report["provider"]
        if not isinstance(provider, str) or PROVIDER_ID.fullmatch(provider) is None:
            raise PlannerClientError("provider is not a safe registry identifier")
        payload["provider"] = provider
    if "required_country" in report:
        required_country = report["required_country"]
        if (
            not isinstance(required_country, str)
            or COUNTRY_CODE.fullmatch(required_country) is None
        ):
            raise PlannerClientError("required_country must be uppercase ISO alpha-2")
        payload["required_country"] = required_country
    return payload


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Merge redacted worker reports and call Mazzy VPN planner."
    )
    parser.add_argument(
        "--socket",
        type=Path,
        default=DEFAULT_SOCKET,
        help=argparse.SUPPRESS,
    )
    args = parser.parse_args()
    try:
        report = read_bounded_payload(sys.stdin.buffer)
        payload = build_planner_payload(report)
        result = PlannerClient(args.socket).evaluate(payload)
    except PlannerClientError as error:
        print(f"multi-agent planner: {error}", file=sys.stderr)
        return 1
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
