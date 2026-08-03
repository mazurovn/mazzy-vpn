#!/usr/bin/env python3
"""Validate the strict JSON boundary used by the planner SDK example."""

from __future__ import annotations

import importlib.util
import sys
from io import BytesIO
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
CLIENT_PATH = ROOT / "examples" / "mazzy_planner_client.py"


def fail(message: str) -> None:
    raise AssertionError(message)


def load_client():
    sys.dont_write_bytecode = True
    spec = importlib.util.spec_from_file_location("mazzy_planner_client", CLIENT_PATH)
    if spec is None or spec.loader is None:
        fail("cannot load planner client example")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def must_reject(client, raw: bytes, label: str) -> None:
    try:
        client.read_bounded_payload(BytesIO(raw))
    except client.PlannerClientError:
        return
    fail(f"planner client accepted {label}")


def main() -> None:
    client = load_client()
    parsed = client.read_bounded_payload(BytesIO(b"\n {\"workload\":\"general\"} \n"))
    if parsed != {"workload": "general"}:
        fail("planner client rejected legal surrounding JSON whitespace")

    must_reject(client, b'{"evidence":{},"evidence":{}}', "duplicate keys")
    must_reject(client, b'{"value":NaN}', "NaN")
    must_reject(client, b'{"value":Infinity}', "Infinity")
    must_reject(client, b'{} {}', "multiple JSON documents")
    must_reject(client, b'[]', "a non-object payload")
    must_reject(client, b"x" * (client.MAX_REQUEST_BYTES + 1), "an oversized payload")

    print("PLANNER EXAMPLES OK: strict whitespace, duplicate, finite and size checks")


if __name__ == "__main__":
    main()
