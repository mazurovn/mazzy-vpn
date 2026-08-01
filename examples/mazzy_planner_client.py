#!/usr/bin/env python3
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
"""Call Mazzy VPN's read-only planner without exposing profile secrets."""

from __future__ import annotations

import argparse
import json
import socket
import sys
import uuid
from pathlib import Path
from typing import Any, BinaryIO


API_VERSION = "1.0"
DEFAULT_SOCKET = Path("/run/mazzy-vpn/api-v1.sock")
MAX_REQUEST_BYTES = 65_536
MAX_RESPONSE_BYTES = 1_048_576
SOCKET_TIMEOUT_SECONDS = 40


class PlannerClientError(RuntimeError):
    """A bounded input, transport or API contract failure."""


def reject_duplicate_keys(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    value: dict[str, Any] = {}
    for key, item in pairs:
        if key in value:
            raise ValueError(f"duplicate JSON key: {key}")
        value[key] = item
    return value


def decode_one_object(raw: bytes, label: str) -> dict[str, Any]:
    try:
        text = raw.decode("utf-8")
        decoder = json.JSONDecoder(object_pairs_hook=reject_duplicate_keys)
        value, end = decoder.raw_decode(text)
    except (UnicodeDecodeError, json.JSONDecodeError, ValueError) as error:
        raise PlannerClientError(f"{label} is not strict UTF-8 JSON: {error}") from error
    if text[end:].strip():
        raise PlannerClientError(f"{label} contains multiple JSON documents")
    if not isinstance(value, dict):
        raise PlannerClientError(f"{label} must be a JSON object")
    return value


def read_bounded_payload(stream: BinaryIO) -> dict[str, Any]:
    raw = stream.read(MAX_REQUEST_BYTES + 1)
    if not raw or len(raw) > MAX_REQUEST_BYTES:
        raise PlannerClientError("planner payload must contain 1..65536 bytes")
    return decode_one_object(raw, "planner payload")


class PlannerClient:
    def __init__(self, socket_path: Path = DEFAULT_SOCKET) -> None:
        self.socket_path = socket_path

    def evaluate(self, payload: dict[str, Any]) -> dict[str, Any]:
        request_id = f"request-{uuid.uuid4().hex}"
        request = {
            "api_version": API_VERSION,
            "request_id": request_id,
            "operation": "planner.evaluate",
            "deadline_ms": 30_000,
            "payload": payload,
        }
        encoded = json.dumps(
            request,
            ensure_ascii=False,
            separators=(",", ":"),
        ).encode("utf-8")
        if len(encoded) > MAX_REQUEST_BYTES:
            raise PlannerClientError("API envelope exceeds 65536 bytes")

        response_raw = self._exchange(encoded + b"\n")
        response = decode_one_object(response_raw, "planner response")
        if (
            response.get("api_version") != API_VERSION
            or response.get("request_id") != request_id
        ):
            raise PlannerClientError("planner response identity does not match request")
        if response.get("status") == "error":
            error = response.get("error")
            if not isinstance(error, dict):
                raise PlannerClientError("planner returned a malformed error")
            raise PlannerClientError(
                f"{error.get('code', 'unknown')}: "
                f"{error.get('message_key', 'api.error.unknown')}"
            )
        result = response.get("result")
        if response.get("status") != "ok" or not isinstance(result, dict):
            raise PlannerClientError("planner returned a malformed success response")
        if result.get("dry_run") is not True:
            raise PlannerClientError("planner response is not marked dry-run")
        return result

    def _exchange(self, request: bytes) -> bytes:
        chunks: list[bytes] = []
        total = 0
        try:
            with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as connection:
                connection.settimeout(SOCKET_TIMEOUT_SECONDS)
                connection.connect(str(self.socket_path))
                connection.sendall(request)
                connection.shutdown(socket.SHUT_WR)
                while True:
                    chunk = connection.recv(65_536)
                    if not chunk:
                        break
                    total += len(chunk)
                    if total > MAX_RESPONSE_BYTES:
                        raise PlannerClientError("planner response exceeds 1 MiB")
                    chunks.append(chunk)
        except (OSError, TimeoutError) as error:
            raise PlannerClientError(f"local planner transport failed: {error}") from error
        if not chunks:
            raise PlannerClientError("local planner returned no response")
        return b"".join(chunks)


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Evaluate an opaque-ID Mazzy VPN planner payload from stdin."
    )
    parser.add_argument(
        "--socket",
        type=Path,
        default=DEFAULT_SOCKET,
        help=argparse.SUPPRESS,
    )
    args = parser.parse_args()
    try:
        payload = read_bounded_payload(sys.stdin.buffer)
        result = PlannerClient(args.socket).evaluate(payload)
    except PlannerClientError as error:
        print(f"planner client: {error}", file=sys.stderr)
        return 1
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
