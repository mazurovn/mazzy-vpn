#!/usr/bin/env python3
"""Validate closed managed profiles and generated sing-box runtime configs."""

from __future__ import annotations

import copy
import base64
import json
import os
import stat
import subprocess
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
CLI = ROOT / "mazzy-vpn"
ADAPTER = ROOT / "runtime" / "mazzy-sing-box-adapter"
SCHEMA = ROOT / "protocols" / "v1" / "managed-profile.schema.json"
UUID = "11111111-1111-4111-8111-111111111111"


def fail(message: str) -> None:
    raise AssertionError(message)


def base_profile(protocol: str) -> dict[str, object]:
    tls_enabled = protocol not in {"shadowsocks2022", "mieru"}
    return {
        "schema_version": 1,
        "profile_id": f"{protocol}-test",
        "display_name": f"{protocol} test",
        "protocol": protocol,
        "endpoint": {"host": "192.0.2.10", "port": 443},
        "credentials": {"password": f"{protocol}-secret"},
        "tls": {
            "enabled": tls_enabled,
            "insecure": False,
            "server_name": "example.test" if tls_enabled else "",
        },
        "options": {},
        "dns": {
            "strategy": "prefer_ipv4",
            "servers": [
                {
                    "server": "192.0.2.53",
                    "server_port": 443,
                    "server_name": "dns.example.test",
                    "path": "/dns-query",
                }
            ],
        },
        "routing": {"mode": "full-tunnel", "allow_lan": True},
    }


def profile_for(protocol: str) -> dict[str, object]:
    profile = base_profile(protocol)
    if protocol == "vless":
        profile["credentials"] = {"uuid": UUID}
        profile["tls"] = {
            "enabled": True,
            "insecure": False,
            "server_name": "example.test",
            "utls_fingerprint": "chrome",
            "reality_public_key": "A" * 43,
            "reality_short_id": "0123456789abcdef",
        }
        profile["options"] = {"flow": "xtls-rprx-vision", "network": "both"}
    elif protocol == "hysteria2":
        profile["options"] = {
            "up_mbps": 50,
            "down_mbps": 100,
            "obfs_type": "salamander",
            "obfs_password": "obfs-secret",
        }
    elif protocol == "tuic":
        profile["credentials"] = {"uuid": UUID, "password": "tuic-secret"}
        profile["options"] = {
            "congestion_control": "bbr",
            "udp_relay_mode": "native",
            "heartbeat": "10s",
        }
    elif protocol == "mieru":
        profile["credentials"] = {"username": "mieru-user", "password": "mieru-secret"}
        profile["options"] = {
            "transport": "TCP",
            "mtu": 1400,
            "multiplexing": "MULTIPLEXING_LOW",
            "handshake_mode": "HANDSHAKE_STANDARD",
        }
    elif protocol == "naive":
        profile["credentials"] = {"username": "naive-user", "password": "naive-secret"}
        profile["options"] = {
            "proxy_protocol": "https",
            "tunnel_timeout": 1800,
            "idle_timeout": 600,
        }
    elif protocol == "shadowsocks2022":
        profile["credentials"] = {
            "password": base64.b64encode(b"S" * 32).decode("ascii")
        }
        profile["options"] = {"method": "2022-blake3-aes-256-gcm"}
    elif protocol == "trojan":
        profile["options"] = {"network": "both"}
    elif protocol == "anytls":
        profile["options"] = {
            "idle_session_check_interval": "30s",
            "idle_session_timeout": "30s",
            "min_idle_session": 2,
        }
    elif protocol == "shadowtls":
        profile["options"] = {"version": 3, "network": "tcp"}
    else:
        fail(f"unknown fixture protocol: {protocol}")
    return profile


def validate(profile: dict[str, object] | str) -> tuple[int, dict[str, object]]:
    payload = profile if isinstance(profile, str) else json.dumps(profile)
    result = subprocess.run(
        [str(CLI), "protocols", "managed-validate", "--stdin", "--json"],
        cwd=ROOT,
        input=payload,
        capture_output=True,
        text=True,
        check=False,
        timeout=10,
    )
    try:
        response = json.loads(result.stdout)
    except json.JSONDecodeError as error:
        fail(f"validator returned invalid JSON: {error}: {result.stdout!r}")
    return result.returncode, response


def validate_bytes(payload: bytes) -> tuple[int, dict[str, object]]:
    result = subprocess.run(
        [str(CLI), "protocols", "managed-validate", "--stdin", "--json"],
        cwd=ROOT,
        input=payload,
        capture_output=True,
        check=False,
        timeout=10,
    )
    try:
        response = json.loads(result.stdout.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as error:
        fail(f"validator returned invalid JSON for byte input: {error}")
    return result.returncode, response


def recursive_keys(value: object) -> set[str]:
    keys: set[str] = set()
    if isinstance(value, dict):
        keys.update(value)
        for child in value.values():
            keys.update(recursive_keys(child))
    elif isinstance(value, list):
        for child in value:
            keys.update(recursive_keys(child))
    return keys


def main() -> None:
    requested_sing_box = os.environ.get("MAZZY_TEST_SING_BOX", "")
    if requested_sing_box not in {"", "sing-box"}:
        fail("MAZZY_TEST_SING_BOX may only enable the literal sing-box command")
    check_with_sing_box = requested_sing_box == "sing-box"
    protocols = [
        "vless",
        "hysteria2",
        "mieru",
        "naive",
        "tuic",
        "shadowsocks2022",
        "trojan",
        "anytls",
        "shadowtls",
    ]
    expected_types = {
        "vless": "vless",
        "hysteria2": "hysteria2",
        "tuic": "tuic",
        "shadowsocks2022": "shadowsocks",
        "trojan": "trojan",
        "anytls": "anytls",
    }
    schema = json.loads(SCHEMA.read_text(encoding="utf-8"))
    schema_protocols = set(schema["properties"]["protocol"]["enum"])
    branch_protocols = {
        branch["properties"]["protocol"]["const"] for branch in schema["oneOf"]
    }
    if schema_protocols != set(protocols) or branch_protocols != set(protocols):
        fail("managed profile schema and validator protocol sets differ")
    if schema.get("additionalProperties") is not False:
        fail("managed profile schema is not closed at the root")
    with tempfile.TemporaryDirectory(prefix="mazzy-managed-") as directory_name:
        directory = Path(directory_name)
        environment = {
            **os.environ,
            "MAZZY_ADAPTER_ALLOW_UNPRIVILEGED": "1",
            "MAZZY_VPN_CLI": str(CLI),
            "VPNCTL_RUN_DIR": str(directory / "run"),
        }
        for protocol in protocols:
            profile = profile_for(protocol)
            status, response = validate(profile)
            if status != 0 or response.get("valid") is not True:
                fail(f"{protocol}: valid managed profile was rejected: {response}")
            serialized_response = json.dumps(response)
            for secret in [
                "192.0.2.10",
                "192.0.2.53",
                f"{protocol}-secret",
                "obfs-secret",
                "tuic-secret",
                UUID,
            ]:
                if secret in serialized_response:
                    fail(f"{protocol}: validator reflected profile secrets")
            if response.get("connection_enabled") is not False:
                fail(f"{protocol}: validator overclaims lifecycle integration")

            profile_path = directory / f"{protocol}.json"
            output_path = directory / "rendered" / f"{protocol}.json"
            profile_path.write_text(json.dumps(profile), encoding="utf-8")
            rendered = subprocess.run(
                [str(ADAPTER), "render", str(profile_path), str(output_path)],
                cwd=ROOT,
                env=environment,
                capture_output=True,
                text=True,
                check=False,
                timeout=10,
            )
            if protocol in {"mieru", "naive", "shadowtls"}:
                if response.get("render_supported") is not False:
                    fail(f"{protocol}: separate adapter incorrectly claims a renderer")
                if rendered.returncode == 0:
                    fail(f"{protocol}: rendered without an audited sidecar or inner chain")
                continue
            if rendered.returncode != 0:
                fail(f"{protocol}: render failed: {rendered.stderr}")
            config = json.loads(output_path.read_text(encoding="utf-8"))
            if set(config) != {"log", "dns", "inbounds", "outbounds", "route"}:
                fail(f"{protocol}: generated top-level config is not closed")
            if len(config["inbounds"]) != 1 or config["inbounds"][0].get("type") != "tun":
                fail(f"{protocol}: generated config does not own exactly one TUN")
            if config["inbounds"][0].get("interface_name") != "mazzytun0":
                fail(f"{protocol}: profile controlled the TUN interface")
            if config["outbounds"][0].get("type") != expected_types[protocol]:
                fail(f"{protocol}: generated outbound type is wrong")
            if config["route"].get("final") != "proxy-out":
                fail(f"{protocol}: generated routing does not fail into the proxy")
            if config["dns"]["servers"][0].get("detour") != "proxy-out":
                fail(f"{protocol}: application DNS bypasses the proxy")
            forbidden = {
                "listen",
                "listen_port",
                "certificate_path",
                "client_certificate_path",
                "client_key_path",
                "key_path",
                "netns",
                "routing_mark",
                "bind_interface",
            }
            leaked = recursive_keys(config) & forbidden
            if leaked:
                fail(f"{protocol}: generated config contains forbidden keys: {leaked}")
            if stat.S_IMODE(output_path.stat().st_mode) != 0o600:
                fail(f"{protocol}: generated secret config is not mode 0600")
            if check_with_sing_box:
                engine_check = subprocess.run(
                    ["sing-box", "check", "-c", str(output_path)],
                    cwd=ROOT,
                    capture_output=True,
                    text=True,
                    check=False,
                    timeout=10,
                )
                if engine_check.returncode != 0:
                    fail(
                        f"{protocol}: sing-box rejected generated config: "
                        f"{engine_check.stderr or engine_check.stdout}"
                    )

        malicious = profile_for("vless")
        malicious["inbounds"] = [{"type": "mixed", "listen": "0.0.0.0"}]
        status, response = validate(malicious)
        if status != 2 or response.get("reason") != "invalid-managed-profile":
            fail("managed validator accepted an injected listener")

        insecure = profile_for("trojan")
        insecure["tls"]["insecure"] = True  # type: ignore[index]
        status, response = validate(insecure)
        if status != 2 or response.get("reason") != "invalid-managed-profile":
            fail("managed validator accepted insecure TLS")

        invalid_tls_profiles = []
        arbitrary_utls = profile_for("vless")
        arbitrary_utls["tls"]["utls_fingerprint"] = "custom-client"  # type: ignore[index]
        invalid_tls_profiles.append(("arbitrary uTLS fingerprint", arbitrary_utls))
        numeric_utls = profile_for("vless")
        numeric_utls["tls"]["utls_fingerprint"] = 7  # type: ignore[index]
        invalid_tls_profiles.append(("non-string uTLS fingerprint", numeric_utls))
        duplicate_alpn = profile_for("trojan")
        duplicate_alpn["tls"]["alpn"] = ["h2", "h2"]  # type: ignore[index]
        invalid_tls_profiles.append(("duplicate ALPN", duplicate_alpn))
        short_pin = profile_for("trojan")
        short_pin["tls"]["certificate_public_key_sha256"] = ["x"]  # type: ignore[index]
        invalid_tls_profiles.append(("short certificate pin", short_pin))
        duplicate_pin = profile_for("trojan")
        duplicate_pin["tls"]["certificate_public_key_sha256"] = [  # type: ignore[index]
            "A" * 43,
            "A" * 43,
        ]
        invalid_tls_profiles.append(("duplicate certificate pin", duplicate_pin))
        for label, invalid_profile in invalid_tls_profiles:
            status, response = validate(invalid_profile)
            if status != 2 or response.get("reason") != "invalid-managed-profile":
                fail(f"managed validator accepted {label}")

        disabled_tls_alpn = profile_for("shadowsocks2022")
        disabled_tls_alpn["tls"]["alpn"] = ["h2"]  # type: ignore[index]
        status, response = validate(disabled_tls_alpn)
        if status != 2 or response.get("reason") != "invalid-managed-profile":
            fail("managed validator accepted ALPN while TLS is disabled")

        duplicate = json.dumps(profile_for("vless"))
        duplicate = duplicate.replace(
            '"schema_version": 1,',
            '"schema_version": 1, "schema_version": 2,',
            1,
        )
        status, response = validate(duplicate)
        if status != 2 or response.get("reason") != "duplicate-key":
            fail("managed validator accepted a duplicate key")

        spoofed = profile_for("trojan")
        spoofed["display_name"] = "safe\u202eevil"
        status, response = validate(spoofed)
        if status != 2 or response.get("reason") != "unsafe-display-name":
            fail("managed validator accepted a bidi-spoofed display name")

        nul_payload = json.dumps(profile_for("trojan")).encode("utf-8")
        nul_payload = nul_payload.replace(b', "profile_id"', b',\x00 "profile_id"', 1)
        status, response = validate_bytes(nul_payload)
        if status != 2 or response.get("reason") != "invalid-input":
            fail("managed validator accepted a raw NUL discarded by the shell")

        hostname = profile_for("trojan")
        hostname["endpoint"] = {"host": "vpn.example.test", "port": 443}
        status, response = validate(hostname)
        if status != 0 or response.get("valid") is not True:
            fail("managed validator should classify a hostname profile")
        profile_path = directory / "hostname.json"
        output_path = directory / "rendered" / "hostname.json"
        profile_path.write_text(json.dumps(hostname), encoding="utf-8")
        rendered = subprocess.run(
            [str(ADAPTER), "render", str(profile_path), str(output_path)],
            cwd=ROOT,
            env=environment,
            capture_output=True,
            text=True,
            check=False,
            timeout=10,
        )
        if rendered.returncode == 0 or "bootstrap DNS leakage" not in rendered.stderr:
            fail("v1 renderer accepted an endpoint requiring bootstrap DNS")

        import_source = directory / "managed-import.json"
        imported_profile = profile_for("vless")
        import_source.write_text(json.dumps(imported_profile), encoding="utf-8")
        import_environment = {
            **environment,
            "VPNCTL_ALLOW_UNPRIVILEGED": "1",
            "VPNCTL_CONFIG_DIR": str(directory / "profiles"),
        }
        dry_run = subprocess.run(
            [str(CLI), "protocols", "managed-import", str(import_source), "--dry-run", "--json"],
            cwd=ROOT,
            env=import_environment,
            capture_output=True,
            text=True,
            check=False,
            timeout=10,
        )
        dry_response = json.loads(dry_run.stdout)
        if dry_run.returncode != 0 or dry_response.get("dry_run") is not True:
            fail(f"managed import dry-run failed: {dry_run.stderr}")
        target = directory / "profiles" / "vless" / "vless-test.json"
        if target.exists():
            fail("managed import dry-run wrote a profile")

        imported = subprocess.run(
            [str(CLI), "protocols", "managed-import", str(import_source), "--json"],
            cwd=ROOT,
            env=import_environment,
            capture_output=True,
            text=True,
            check=False,
            timeout=10,
        )
        import_response = json.loads(imported.stdout)
        if imported.returncode != 0 or import_response.get("imported") is not True:
            fail(f"managed profile import failed: {imported.stderr}")
        if target.read_text(encoding="utf-8") != import_source.read_text(encoding="utf-8"):
            fail("managed import changed the validated profile snapshot")
        if stat.S_IMODE(target.stat().st_mode) != 0o600:
            fail("managed imported profile is not mode 0600")
        if stat.S_IMODE(target.parent.stat().st_mode) != 0o700:
            fail("managed protocol directory is not mode 0700")
        if any(secret in imported.stdout for secret in [UUID, "192.0.2.10", "example.test"]):
            fail("managed import reflected profile secrets")

        conflict = subprocess.run(
            [str(CLI), "protocols", "managed-import", str(import_source), "--json"],
            cwd=ROOT,
            env=import_environment,
            capture_output=True,
            text=True,
            check=False,
            timeout=10,
        )
        if conflict.returncode == 0:
            fail("managed import overwrote an existing profile without --force")

        imported_profile["display_name"] = "updated managed profile"
        import_source.write_text(json.dumps(imported_profile), encoding="utf-8")
        replaced = subprocess.run(
            [str(CLI), "protocols", "managed-import", str(import_source), "--force", "--json"],
            cwd=ROOT,
            env=import_environment,
            capture_output=True,
            text=True,
            check=False,
            timeout=10,
        )
        if replaced.returncode != 0 or json.loads(replaced.stdout).get("replaced") is not True:
            fail(f"managed import --force failed: {replaced.stderr}")
        if json.loads(target.read_text(encoding="utf-8"))["display_name"] != "updated managed profile":
            fail("managed import --force did not atomically replace the profile")

        symlink_source = directory / "managed-import-link.json"
        symlink_source.symlink_to(import_source)
        symlink_result = subprocess.run(
            [str(CLI), "protocols", "managed-import", str(symlink_source), "--json"],
            cwd=ROOT,
            env=import_environment,
            capture_output=True,
            text=True,
            check=False,
            timeout=10,
        )
        if symlink_result.returncode == 0:
            fail("managed import accepted a symlink source")

        fake_bin = directory / "fake-bin"
        fake_bin.mkdir()
        fake_sing_box = fake_bin / "sing-box"
        fake_sing_box.write_text(
            "#!/usr/bin/env bash\n"
            "case \"${1:-}\" in\n"
            "  version) printf 'sing-box version 1.13.12\\n' ;;\n"
            "  check|run) exit 0 ;;\n"
            "  *) exit 2 ;;\n"
            "esac\n",
            encoding="utf-8",
        )
        fake_sing_box.chmod(0o755)
        run_environment = {
            **environment,
            "PATH": f"{fake_bin}:{os.environ.get('PATH', '')}",
        }
        run_profile = directory / "vless.json"
        check_result = subprocess.run(
            [str(ADAPTER), "check", str(run_profile)],
            cwd=ROOT,
            env=run_environment,
            capture_output=True,
            text=True,
            check=False,
            timeout=10,
        )
        if check_result.returncode != 0:
            fail(f"managed adapter check failed: {check_result.stderr}")
        leaked_checks = list((directory / "run" / "sing-box").glob("check.*.json"))
        if leaked_checks:
            fail("managed adapter left a secret runtime config after engine check")
        run_result = subprocess.run(
            [str(ADAPTER), "run", str(run_profile)],
            cwd=ROOT,
            env=run_environment,
            capture_output=True,
            text=True,
            check=False,
            timeout=10,
        )
        if run_result.returncode != 0:
            fail(f"managed adapter run supervision failed: {run_result.stderr}")
        leaked_configs = list((directory / "run" / "sing-box").glob("config.*.json"))
        if leaked_configs:
            fail("managed adapter left a secret runtime config after engine exit")

    print("MANAGED PROTOCOL ADAPTER OK: 9 validated, 6 closed renderers, atomic import")


if __name__ == "__main__":
    main()
