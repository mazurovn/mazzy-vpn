#!/usr/bin/env python3
"""Isolated LAN-WSS conformance test for the Mazzy egress agent daemon."""

from __future__ import annotations

import base64
import datetime as dt
import hashlib
import json
import os
import socket
import sqlite3
import ssl
import struct
import subprocess
import tempfile
import threading
import time
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
AGENTD = ROOT / "mazzy-agentd"
PROFILE_ID = "profile-11111111111111111111111111111111"
PROTOCOL = "mazzy-agent-egress/1"
SCOPES = [
    "vpn.status",
    "vpn.select",
    "vpn.connect",
    "vpn.disconnect",
    "vpn.verify",
    "planner.evaluate",
    "region.check",
]


def fail(message: str) -> None:
    raise AssertionError(message)


def run(*arguments: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        arguments,
        check=True,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def create_certificates(directory: Path) -> tuple[Path, Path, Path, Path, Path]:
    ca_key = directory / "ca.key"
    ca_cert = directory / "ca.crt"
    server_key = directory / "server.key"
    server_csr = directory / "server.csr"
    server_cert = directory / "server.crt"
    client_key = directory / "client.key"
    client_csr = directory / "client.csr"
    client_cert = directory / "client.crt"
    run(
        "openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes",
        "-days", "1", "-subj", "/CN=Mazzy test CA",
        "-keyout", str(ca_key), "-out", str(ca_cert),
    )
    run(
        "openssl", "req", "-newkey", "rsa:2048", "-nodes",
        "-subj", "/CN=127.0.0.1", "-addext", "subjectAltName=IP:127.0.0.1",
        "-addext", "extendedKeyUsage=serverAuth",
        "-keyout", str(server_key), "-out", str(server_csr),
    )
    run(
        "openssl", "x509", "-req", "-days", "1", "-sha256",
        "-in", str(server_csr), "-CA", str(ca_cert), "-CAkey", str(ca_key),
        "-CAcreateserial", "-copy_extensions", "copy", "-out", str(server_cert),
    )
    run(
        "openssl", "req", "-newkey", "rsa:2048", "-nodes",
        "-subj", "/CN=Mazzy paired client", "-addext", "extendedKeyUsage=clientAuth",
        "-keyout", str(client_key), "-out", str(client_csr),
    )
    run(
        "openssl", "x509", "-req", "-days", "1", "-sha256",
        "-in", str(client_csr), "-CA", str(ca_cert), "-CAkey", str(ca_key),
        "-CAserial", str(directory / "ca.srl"), "-copy_extensions", "copy",
        "-out", str(client_cert),
    )
    for key in (ca_key, server_key, client_key):
        key.chmod(0o600)
    return ca_cert, server_cert, server_key, client_cert, client_key


def fingerprint(certificate: Path) -> str:
    der = ssl.PEM_cert_to_DER_cert(certificate.read_text(encoding="ascii"))
    return hashlib.sha256(der).hexdigest()


class FakeApi:
    def __init__(self, path: Path):
        self.path = path
        self.stop = threading.Event()
        self.ready = threading.Event()
        self.requests: list[dict[str, Any]] = []
        self.malicious_status = False
        self.malformed_region = False
        self.next_connect_state = "succeeded"
        self.lock = threading.Lock()
        self.thread = threading.Thread(target=self.serve, daemon=True)

    def start(self) -> None:
        self.thread.start()
        if not self.ready.wait(5):
            fail("fake local API did not become ready")

    def close(self) -> None:
        self.stop.set()
        try:
            with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as wake:
                wake.connect(str(self.path))
        except OSError:
            pass
        self.thread.join(5)

    def serve(self) -> None:
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as listener:
            listener.bind(str(self.path))
            listener.listen(8)
            listener.settimeout(0.2)
            self.ready.set()
            while not self.stop.is_set():
                try:
                    connection, _ = listener.accept()
                except TimeoutError:
                    continue
                with connection:
                    raw = bytearray()
                    while len(raw) <= 65_536 and b"\n" not in raw:
                        chunk = connection.recv(8192)
                        if not chunk:
                            break
                        raw.extend(chunk)
                    if not raw:
                        continue
                    request = json.loads(bytes(raw).decode("utf-8"))
                    if not isinstance(request, dict):
                        fail("fake API request is not an object")
                    with self.lock:
                        self.requests.append(request)
                    response = {
                        "api_version": "1.0",
                        "request_id": request["request_id"],
                        "status": "ok",
                        "result": self.result(request),
                    }
                    connection.sendall(
                        json.dumps(response, separators=(",", ":")).encode("utf-8")
                    )

    def result(self, request: dict[str, Any]) -> dict[str, Any]:
        operation = request.get("operation")
        payload = request.get("payload")
        if operation == "api.capabilities":
            return {
                "schema_version": 1,
                "api_version": "1.0",
                "engine_version": "1.4.6",
                "operations": [
                    "api.capabilities",
                    "lifecycle.connect",
                    "lifecycle.disconnect",
                    "planner.evaluate",
                    "profiles.list",
                    "region.check",
                    "status.get",
                    "tests.verify-service-egress",
                ],
            }
        if operation == "profiles.list":
            return {
                "profiles": [{
                    "profile_id": PROFILE_ID,
                    "display_name": "Austria test",
                    "protocol": "amneziawg",
                    "country_code": "AT",
                    "selected": False,
                    "validation": "valid",
                }]
            }
        if operation == "planner.evaluate":
            if payload.get("provider") != "antigravity" or payload.get(
                "required_country"
            ) != "AT":
                fail("agentd lost provider/country constraints at planner boundary")
            if [item.get("profile_id") for item in payload.get("candidates", [])] != [
                PROFILE_ID
            ]:
                fail("agentd did not derive planner candidates from backend profiles")
            gates = [
                "backend-ready", "profile-valid",
                "secrets-readable-only-by-backend", "rollback-storage-ready",
                "platform-supported", "provider-country-supported",
                "required-country-match",
            ]
            factors = [
                "recent-success", "censorship-fit", "reachability",
                "latency-loss", "workload-fit",
            ]
            return {
                "schema_version": 1,
                "policy_version": 1,
                "catalog_version": "2026-08-11",
                "evaluated_at": "2026-08-11T08:00:00Z",
                "dry_run": True,
                "workload": "llm-streaming",
                "provider": "antigravity",
                "required_country": "AT",
                "ordered_profile_ids": [PROFILE_ID],
                "candidates": [{
                    "profile_id": PROFILE_ID,
                    "protocol": "amneziawg",
                    "eligible": True,
                    "rank": 1,
                    "score": 100,
                    "hard_gates": [
                        {"id": gate, "passed": True, "reason_code": f"planner.gate.{gate}"}
                        for gate in gates
                    ],
                    "factors": [
                        {
                            "id": factor,
                            "points": 20,
                            "max_points": 20,
                            "reason_code": f"planner.factor.{factor}",
                        }
                        for factor in factors
                    ],
                    "reason_codes": [],
                }],
            }
        if operation == "lifecycle.connect":
            if request.get("authorization") != "system-mutate":
                fail("mutating local API call omitted system authorization marker")
            if payload != {"profile_id": PROFILE_ID}:
                fail("connect did not use the durable selected profile")
            state = self.next_connect_state
            self.next_connect_state = "succeeded"
            return {
                "action_id": request["action_id"],
                "state": state,
                "state_changed": True,
                "rollback": {
                    "required": state != "succeeded",
                    "state": "not-needed" if state == "succeeded" else "completed",
                    "message_key": (
                        "api.rollback.not-needed"
                        if state == "succeeded"
                        else "api.rollback.completed"
                    ),
                },
                "message_key": (
                    "api.lifecycle.succeeded"
                    if state == "succeeded"
                    else "api.lifecycle.rolled-back"
                ),
            }
        if operation == "lifecycle.disconnect":
            return {
                "action_id": request["action_id"],
                "state": "succeeded",
                "state_changed": True,
                "rollback": {
                    "required": False,
                    "state": "not-needed",
                    "message_key": "api.rollback.not-needed",
                },
            }
        if operation == "tests.verify-service-egress":
            if payload != {"service": "antigravity", "timeout_seconds": 10}:
                fail("verify payload drifted")
            return {
                "schema_version": 1,
                "checked_at": "2026-08-11T08:00:00Z",
                "scope": "unauthenticated-egress",
                "results": [{
                    "service_id": "antigravity",
                    "probe_version": 1,
                    "reachability": "reachable",
                    "egress_eligibility": "eligible",
                    "reason_code": "service.antigravity.boundary-reached",
                    "http_status": 401,
                }],
            }
        if operation == "region.check":
            if payload != {"provider": "antigravity", "target_country": "AT"}:
                fail("region-check payload drifted")
            result = {
                "schema_version": 1,
                "provider": "antigravity",
                "egress_country": "AT",
                "system_timezone": "Europe/Vienna",
                "timezone_country": "AT",
                "supported_by_provider": True,
                "country_consistent": True,
                "account_region_hint": "set account country association to AT (manual L2)",
                "verdict": "ready",
                "mismatches": [],
            }
            if self.malformed_region:
                self.malformed_region = False
                result["mismatches"] = [{}]
            return result
        if operation == "status.get":
            if self.malicious_status:
                self.malicious_status = False
                return {
                    "product": "Mazzy VPN",
                    "engine_version": "1.4.6",
                    "generated_at": "2026-08-11T08:00:00Z",
                    "available": True,
                    "connection": {
                        "state": "connected",
                        "access_token": "SENTINEL-DO-NOT-LEAK",
                        "workspace": "/secret/local/path",
                    },
                }
            return {
                "product": "Mazzy VPN",
                "engine_version": "1.4.6",
                "generated_at": "2026-08-11T08:00:00Z",
                "available": True,
                "connection": {
                    "state": "connected",
                    "protocol": "amneziawg",
                    "profile_id": PROFILE_ID,
                    "internet_reachable": True,
                    "healthy": True,
                },
            }
        fail(f"unexpected fake API operation: {operation}")
        return {}


def free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
        listener.bind(("127.0.0.1", 0))
        return int(listener.getsockname()[1])


def recv_exact(connection: ssl.SSLSocket, size: int) -> bytes:
    result = bytearray()
    while len(result) < size:
        chunk = connection.recv(size - len(result))
        if not chunk:
            fail("LAN-WSS connection closed unexpectedly")
        result.extend(chunk)
    return bytes(result)


def send_json(connection: ssl.SSLSocket, value: dict[str, Any]) -> None:
    payload = json.dumps(value, separators=(",", ":")).encode("utf-8")
    mask = os.urandom(4)
    header = bytearray([0x81])
    if len(payload) < 126:
        header.append(0x80 | len(payload))
    else:
        header.append(0x80 | 126)
        header.extend(struct.pack("!H", len(payload)))
    masked = bytes(item ^ mask[index % 4] for index, item in enumerate(payload))
    connection.sendall(bytes(header) + mask + masked)


def receive_json(connection: ssl.SSLSocket) -> dict[str, Any]:
    first, second = recv_exact(connection, 2)
    if first != 0x81 or second & 0x80:
        fail("daemon returned a malformed WebSocket text frame")
    length = second & 0x7F
    if length == 126:
        length = struct.unpack("!H", recv_exact(connection, 2))[0]
    elif length == 127:
        length = struct.unpack("!Q", recv_exact(connection, 8))[0]
    value = json.loads(recv_exact(connection, length).decode("utf-8"))
    if not isinstance(value, dict):
        fail("daemon WebSocket message is not an object")
    return value


def connect_wss(
    port: int, ca_cert: Path, client_cert: Path, client_key: Path
) -> ssl.SSLSocket:
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
    context.minimum_version = ssl.TLSVersion.TLSv1_3
    context.maximum_version = ssl.TLSVersion.TLSv1_3
    context.load_verify_locations(cafile=ca_cert)
    context.load_cert_chain(client_cert, client_key)
    key = base64.b64encode(os.urandom(16)).decode("ascii")
    deadline = time.monotonic() + 5
    while True:
        try:
            raw = socket.create_connection(("127.0.0.1", port), timeout=1)
            connection = context.wrap_socket(raw, server_hostname="127.0.0.1")
            break
        except OSError:
            if time.monotonic() >= deadline:
                raise
            time.sleep(0.05)
    connection.settimeout(5)
    connection.sendall(
        (
            "GET /v1/agent HTTP/1.1\r\n"
            "Host: 127.0.0.1\r\n"
            "Upgrade: websocket\r\n"
            "Connection: Upgrade\r\n"
            "Sec-WebSocket-Version: 13\r\n"
            f"Sec-WebSocket-Key: {key}\r\n\r\n"
        ).encode("ascii")
    )
    response = bytearray()
    while b"\r\n\r\n" not in response:
        response.extend(connection.recv(4096))
    if not response.startswith(b"HTTP/1.1 101 Switching Protocols\r\n"):
        fail("LAN-WSS upgrade failed")
    return connection


def command(sequence: int, capability: str, arguments: dict[str, Any]) -> dict[str, Any]:
    now = dt.datetime.now(dt.timezone.utc)
    high = capability in {"vpn.connect", "vpn.disconnect"}
    return {
        "schema_version": 1,
        "protocol": PROTOCOL,
        "message_type": "command",
        "command_id": f"command-egress-{sequence:04d}",
        "session_id": "session-egress-test-0001",
        "actor_id": "test-operator",
        "source_channel": "cli-first-party",
        "sequence": sequence,
        "issued_at": now.isoformat().replace("+00:00", "Z"),
        "expires_at": (now + dt.timedelta(seconds=30)).isoformat().replace(
            "+00:00", "Z"
        ),
        "capability": capability,
        "target_id": "egress",
        "risk": "high" if high else ("low" if capability == "vpn.select" else "read-only"),
        "confirmation": {
            "required": True,
            "method": "paired-device",
            "confirmation_id": "approval-00000000000000000000000000000000",
        }
        if high
        else {"required": False, "method": "none"},
        "arguments": arguments,
    }


def approve(
    state: Path, certificate: Path, value: dict[str, Any]
) -> dict[str, Any]:
    request = json.loads(json.dumps(value))
    request["message_type"] = "approval-request"
    request["confirmation"].pop("confirmation_id", None)
    projection = json.loads(json.dumps(request))
    projection["message_type"] = "command"
    digest = hashlib.sha256(
        json.dumps(
            projection, ensure_ascii=False, sort_keys=True, separators=(",", ":")
        ).encode("utf-8")
    ).hexdigest()
    result = subprocess.run(
        [
            str(AGENTD), "approve", "--state-dir", str(state),
            "--certificate", str(certificate), "--confirm-digest", digest,
            "--stdin", "--json",
        ],
        input=json.dumps(request, separators=(",", ":")),
        check=True,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    approval = json.loads(result.stdout)
    if approval.get("command_digest") != digest or approval.get("single_use") is not True:
        fail("trusted local approval is not bound to the command digest")
    return approval


def exchange(
    connection: ssl.SSLSocket, value: dict[str, Any]
) -> tuple[dict[str, Any], dict[str, Any]]:
    send_json(connection, value)
    receipt = receive_json(connection)
    result = receive_json(connection)
    if receipt.get("message_type") != "receipt" or result.get("message_type") != "result":
        fail("command did not produce receipt followed by typed result")
    if receipt.get("phase") != "DURABLY_RECEIVED":
        fail("receipt was emitted before durable command journal state")
    return receipt, result


def main() -> None:
    if not AGENTD.is_file() or not os.access(AGENTD, os.X_OK):
        fail("mazzy-agentd executable is missing")
    with tempfile.TemporaryDirectory(prefix="mazzy-agentd-test-") as temporary:
        root = Path(temporary)
        unobserved = root / "diagnose-must-not-create"
        empty_diagnosis = json.loads(
            run(
                str(AGENTD), "diagnose", "--state-dir", str(unobserved), "--json"
            ).stdout
        )
        if unobserved.exists() or empty_diagnosis.get("runtime_ready") is not False:
            fail("read-only diagnostics created state or overclaimed readiness")
        state = root / "state"
        state.mkdir(mode=0o700)
        ca_cert, server_cert, server_key, client_cert, client_key = (
            create_certificates(root)
        )
        stale_socket = root / "stale-api.sock"
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as stale_listener:
            stale_listener.bind(str(stale_socket))
        stale_diagnosis = json.loads(
            run(
                str(AGENTD), "diagnose", "--state-dir", str(state),
                "--certificate", str(server_cert), "--private-key", str(server_key),
                "--client-ca", str(ca_cert), "--api-socket", str(stale_socket),
                "--json",
            ).stdout
        )
        if stale_diagnosis.get("api_socket_ready") is not False:
            fail("unlistened stale local API socket was reported ready")
        stale_socket.unlink()
        api = FakeApi(root / "api.sock")
        api.start()
        pair_args = [
            str(AGENTD), "pair", "--state-dir", str(state),
            "--certificate", str(client_cert),
            "--confirm-fingerprint", fingerprint(client_cert),
            "--actor-id", "test-operator", "--source-channel", "cli-first-party",
        ]
        for scope in SCOPES:
            pair_args.extend(("--scope", scope))
        paired = json.loads(run(*pair_args).stdout)
        if (
            paired.get("paired") is not True
            or paired.get("scopes") != sorted(SCOPES)
            or paired.get("origins") != []
            or paired.get("revoked") is not False
        ):
            fail("explicit certificate pairing did not persist the requested grant")

        unresolved_command = command(0, "vpn.connect", {})
        unresolved_fingerprint = hashlib.sha256(
            json.dumps(
                unresolved_command, ensure_ascii=False, sort_keys=True, separators=(",", ":")
            ).encode("utf-8")
        ).hexdigest()
        database = sqlite3.connect(state / "state.sqlite3")
        try:
            database.execute(
                """
                INSERT INTO commands (
                    fingerprint, command_id, command_fingerprint, sequence,
                    capability, state, result_json, created_at
                ) VALUES (?, ?, ?, 0, 'vpn.connect', 'STARTED', NULL, ?)
                """,
                (
                    fingerprint(client_cert),
                    unresolved_command["command_id"],
                    unresolved_fingerprint,
                    unresolved_command["issued_at"],
                ),
            )
            database.execute(
                "INSERT INTO peer_sequences (fingerprint, last_sequence) VALUES (?, 0)",
                (fingerprint(client_cert),),
            )
            database.commit()
        finally:
            database.close()

        port = free_port()
        daemon = subprocess.Popen(
            [
                str(AGENTD), "serve", "--state-dir", str(state),
                "--listen", "127.0.0.1", "--port", str(port),
                "--certificate", str(server_cert), "--private-key", str(server_key),
                "--client-ca", str(ca_cert), "--api-socket", str(api.path),
            ],
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        response_messages: list[dict[str, Any]] = []
        try:
            connection = connect_wss(port, ca_cert, client_cert, client_key)
            with connection:
                diagnosis = json.loads(
                    run(
                        str(AGENTD), "diagnose", "--state-dir", str(state),
                        "--certificate", str(server_cert),
                        "--private-key", str(server_key),
                        "--client-ca", str(ca_cert),
                        "--api-socket", str(api.path), "--json",
                    ).stdout
                )
                if diagnosis.get("runtime_ready") is not True or not all(
                    diagnosis.get(field) is True
                    for field in (
                        "state_ready", "tls_ready", "api_socket_ready", "daemon_active"
                    )
                ):
                    fail("agentd diagnostics overclaimed or missed configured runtime readiness")
                select_command = command(
                    1, "vpn.select", {"country": "AT", "provider": "antigravity"}
                )
                receipt, result = exchange(connection, select_command)
                response_messages.extend((receipt, result))
                if result.get("status") != "completed" or result.get("result", {}).get(
                    "profile_id"
                ) != PROFILE_ID:
                    fail("vpn.select did not choose the AT profile")

                connect_command = command(2, "vpn.connect", {})
                approval = approve(state, client_cert, connect_command)
                if approval.get("selection_revision") != 1:
                    fail("connect approval is not bound to the selected egress revision")
                connect_command["confirmation"]["confirmation_id"] = approval[
                    "approval_id"
                ]
                receipt, result = exchange(connection, connect_command)
                response_messages.extend((receipt, result))
                if result.get("status") != "completed":
                    fail("vpn.connect did not complete through the local API owner")

                verify_command = command(3, "vpn.verify", {"provider": "antigravity"})
                receipt, result = exchange(connection, verify_command)
                response_messages.extend((receipt, result))
                results = result.get("result", {}).get("results", [])
                if not results or results[0].get("egress_eligibility") != "eligible":
                    fail("vpn.verify did not preserve eligible provider evidence")

                secondary_session = "session-egress-test-0002"
                secondary_select = command(
                    4, "vpn.select", {"country": "AT", "provider": "antigravity"}
                )
                secondary_select["session_id"] = secondary_session
                exchange(connection, secondary_select)
                secondary_connect = command(5, "vpn.connect", {})
                secondary_connect["session_id"] = secondary_session
                secondary_approval = approve(state, client_cert, secondary_connect)
                secondary_connect["confirmation"]["confirmation_id"] = secondary_approval[
                    "approval_id"
                ]
                _, secondary_result = exchange(connection, secondary_connect)
                if secondary_result.get("status") != "completed":
                    fail("second session could not mutate the host-global egress")
                database = sqlite3.connect(state / "state.sqlite3")
                try:
                    stale_evidence = database.execute(
                        """
                        SELECT verified_provider, verification_eligibility,
                               verified_generation
                        FROM egress_selections
                        WHERE fingerprint=? AND session_id=?
                        """,
                        (fingerprint(client_cert), "session-egress-test-0001"),
                    ).fetchone()
                finally:
                    database.close()
                if stale_evidence != (None, None, None):
                    fail("host-global egress mutation did not invalidate every session proof")

                region_command = command(
                    6,
                    "region.check",
                    {"provider": "antigravity", "target_country": "AT"},
                )
                receipt, result = exchange(connection, region_command)
                response_messages.extend((receipt, result))
                handoff = result.get("result", {}).get("handoff")
                if result.get("result", {}).get("workflow_ready") is not True or handoff != {
                    "owner": "browser-agent",
                    "action": "account-region-form",
                    "provider": "antigravity",
                    "country": "AT",
                    "manual_l2_required": True,
                }:
                    fail("ready/eligible workflow did not return the browser L2 handoff")

                api.next_connect_state = "rolled-back"
                failed_connect = command(7, "vpn.connect", {})
                failed_connect["session_id"] = secondary_session
                failed_approval = approve(state, client_cert, failed_connect)
                failed_connect["confirmation"]["confirmation_id"] = failed_approval[
                    "approval_id"
                ]
                _, failed_result = exchange(connection, failed_connect)
                response_messages.append(failed_result)
                if failed_result.get("status") != "failed" or failed_result.get(
                    "error", {}
                ).get("code") != "operation_failed":
                    fail("rolled-back lifecycle outcome was reported as completed")
                database = sqlite3.connect(state / "state.sqlite3")
                try:
                    host_generation = database.execute(
                        "SELECT generation FROM host_egress_state WHERE singleton=1"
                    ).fetchone()
                    verification_count = database.execute(
                        """
                        SELECT COUNT(*) FROM egress_selections
                        WHERE verified_provider IS NOT NULL
                           OR verification_eligibility IS NOT NULL
                           OR verified_generation IS NOT NULL
                        """
                    ).fetchone()
                finally:
                    database.close()
                if host_generation != (3,) or verification_count != (0,):
                    fail("failed lifecycle attempt retained stale host egress evidence")

                duplicate_receipt, duplicate_result = exchange(connection, connect_command)
                response_messages.extend((duplicate_receipt, duplicate_result))
                if duplicate_receipt.get("duplicate") is not True or duplicate_result != response_messages[3]:
                    fail("duplicate command did not return its durable cached outcome")

                unresolved_receipt, unresolved_result = exchange(
                    connection, unresolved_command
                )
                response_messages.extend((unresolved_receipt, unresolved_result))
                if unresolved_receipt.get("duplicate") is not True or unresolved_result.get(
                    "status"
                ) != "in_doubt":
                    fail("crash-recovered unresolved mutation was executed blindly")

                race_session = "session-approval-race-0002"
                first_race_select = command(
                    8, "vpn.select", {"country": "AT", "provider": "antigravity"}
                )
                first_race_select["session_id"] = race_session
                exchange(connection, first_race_select)
                race_connect = command(10, "vpn.connect", {})
                race_connect["session_id"] = race_session
                race_approval = approve(state, client_cert, race_connect)
                race_connect["confirmation"]["confirmation_id"] = race_approval[
                    "approval_id"
                ]
                second_race_select = command(
                    9, "vpn.select", {"country": "AT", "provider": "antigravity"}
                )
                second_race_select["session_id"] = race_session
                exchange(connection, second_race_select)
                _, changed_selection_result = exchange(connection, race_connect)
                if changed_selection_result.get("status") != "failed" or changed_selection_result.get(
                    "error", {}
                ).get("message_key") != "agent.approval.selection-changed":
                    fail("connect approval survived a changed egress selection revision")

                api.malicious_status = True
                _, unsafe_result = exchange(connection, command(11, "vpn.status", {}))
                response_messages.append(unsafe_result)
                if unsafe_result.get("status") != "failed" or unsafe_result.get(
                    "error", {}
                ).get("message_key") != "agent.vpn.status-result-invalid":
                    fail("frontend-forbidden local API result was not rejected")

                api.malformed_region = True
                malformed_region = command(
                    12,
                    "region.check",
                    {"provider": "antigravity", "target_country": "AT"},
                )
                _, malformed_result = exchange(connection, malformed_region)
                response_messages.append(malformed_result)
                if malformed_result.get("status") != "failed" or malformed_result.get(
                    "error", {}
                ).get("message_key") != "agent.region.result-invalid":
                    fail("malformed nested region result escaped the closed API boundary")

                replay = command(3, "vpn.status", {})
                replay["command_id"] = "command-replay-0013"
                send_json(connection, replay)
                replay_error = receive_json(connection)
                response_messages.append(replay_error)
                if replay_error.get("message_type") != "error" or replay_error.get(
                    "error", {}
                ).get("code") != "replay_rejected":
                    fail("new command with a replayed sequence was not rejected")

            revoked_connection = connect_wss(port, ca_cert, client_cert, client_key)
            with revoked_connection:
                revoked = json.loads(
                    run(
                        str(AGENTD), "revoke", "--state-dir", str(state),
                        "--fingerprint", fingerprint(client_cert),
                    ).stdout
                )
                if revoked.get("revoked") is not True:
                    fail("paired certificate was not revoked")
                send_json(revoked_connection, command(13, "vpn.status", {}))
                revoked_error = receive_json(revoked_connection)
                response_messages.append(revoked_error)
                if revoked_error.get("error", {}).get("code") != "revoked_device":
                    fail("revocation did not stop an already-open LAN-WSS session")
        finally:
            daemon.terminate()
            try:
                daemon.wait(5)
            except subprocess.TimeoutExpired:
                daemon.kill()
                daemon.wait(5)
            api.close()
        if daemon.returncode not in {-15, 0}:
            stderr = daemon.stderr.read() if daemon.stderr is not None else ""
            fail(f"mazzy-agentd exited unexpectedly: {stderr[:500]}")
        connect_calls = [
            request for request in api.requests
            if request.get("operation") == "lifecycle.connect"
        ]
        if len(connect_calls) != 3:
            fail("duplicate command caused more than one VPN mutation")
        database = sqlite3.connect(state / "state.sqlite3")
        try:
            audit_count = int(database.execute("SELECT COUNT(*) FROM audit").fetchone()[0])
            command_count = int(
                database.execute("SELECT COUNT(*) FROM commands").fetchone()[0]
            )
            approval_state = database.execute(
                "SELECT used, selection_revision FROM approvals WHERE approval_id=?",
                (approval["approval_id"],),
            ).fetchone()
            race_approval_state = database.execute(
                "SELECT used, selection_revision FROM approvals WHERE approval_id=?",
                (race_approval["approval_id"],),
            ).fetchone()
        finally:
            database.close()
        if audit_count != 12 or command_count != 13:
            fail("durable command/audit journal does not match accepted commands")
        if approval_state != (1, 1):
            fail("command-bound approval was not consumed exactly once")
        if race_approval_state != (0, 1):
            fail("selection-changed approval was consumed or bound to the wrong revision")
        serialized = json.dumps(response_messages, sort_keys=True)
        persisted = (state / "state.sqlite3").read_bytes()
        if (
            str(root) in serialized
            or "SENTINEL-DO-NOT-LEAK" in serialized
            or str(root).encode() in persisted
            or b"PRIVATE KEY" in persisted
            or b"SENTINEL-DO-NOT-LEAK" in persisted
        ):
            fail("agent result or durable state leaked a local path/private key")
    print(
        "MAZZY AGENTD OK: TLS1.3 mTLS, pairing/policy/revocation, command-bound "
        "approval, session isolation, select->connect->verify->region-check->"
        "browser handoff, dedupe/replay"
    )


if __name__ == "__main__":
    main()
