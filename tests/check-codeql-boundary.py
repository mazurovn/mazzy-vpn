#!/usr/bin/env python3
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
"""Keep CodeQL focused on owned code while the vendored crate is byte-verified."""

from __future__ import annotations

import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
CONFIG = ROOT / ".github/codeql/codeql-config.yml"
WORKFLOW = ROOT / ".github/workflows/codeql.yml"
ACTION_SHA = "f205ea1c3313d32999d8d6a48b4f6530d4437b38"

EXPECTED_CONFIG = """name: Mazzy VPN CodeQL configuration

paths-ignore:
  - desktop/src-tauri/vendor/**
  - core/vendor/**
  - mazzy-vpn-cli/vendor/**

queries:
  - uses: security-extended

threat-models: local
"""


def fail(message: str) -> None:
    raise RuntimeError(message)


def main() -> int:
    config = CONFIG.read_text(encoding="utf-8")
    if config != EXPECTED_CONFIG:
        fail(
            "CodeQL config must exclude only the byte-verified vendor tree and "
            "retain security-extended/local-source analysis"
        )

    workflow = WORKFLOW.read_text(encoding="utf-8")
    required = (
        "config-file: ./.github/codeql/codeql-config.yml",
        f"github/codeql-action/init@{ACTION_SHA}",
        f"github/codeql-action/analyze@{ACTION_SHA}",
        "security-events: write",
    )
    for marker in required:
        if marker not in workflow:
            fail(f"CodeQL workflow is missing required marker: {marker}")

    languages = set(re.findall(r"^\s+- language: ([\w-]+)$", workflow, re.MULTILINE))
    # actions/js/python/rust use the path-filterable no-build mode; go uses a
    # manual build of the owned core + cli (its vendor tree is byte-verified via
    # go.sum checksums and excluded above, same guarantee as the glib crate).
    expected_no_build = {"actions", "javascript-typescript", "python", "rust"}
    expected_languages = expected_no_build | {"go"}
    if languages != expected_languages:
        fail(f"unexpected CodeQL language matrix: {sorted(languages)}")
    if workflow.count("build-mode: none") != len(expected_no_build):
        fail("every non-Go CodeQL language must use the path-filterable no-build mode")
    if workflow.count("build-mode: manual") != 1:
        fail("Go CodeQL analysis must use a single manual (vendored) build")

    desktop_workflow = (ROOT / ".github/workflows/desktop.yml").read_text(
        encoding="utf-8"
    )
    if "python3 tests/check-glib-backport.py" not in desktop_workflow:
        fail("Desktop CI no longer verifies the excluded vendor source provenance")

    verifier = (ROOT / "tests/check-glib-backport.py").read_text(encoding="utf-8")
    for forbidden in (
        "MAZZY_GLIB_UPSTREAM_CRATE",
        "CARGO_HOME",
        "os.environ",
        "Path.home()",
    ):
        if forbidden in verifier:
            fail(f"glib verifier accepts an untrusted filesystem path: {forbidden}")
    if "urllib.request.Request(\n        ARCHIVE_URL," not in verifier:
        fail("glib verifier no longer downloads only the pinned crates.io URL")

    deny_config = (ROOT / "deny.toml").read_text(encoding="utf-8")
    if re.search(r"^\s*db-path\s*=", deny_config, re.MULTILINE):
        fail("cargo-deny must use its portable CARGO_HOME-aware advisory DB default")

    print(
        "CODEQL BOUNDARY OK: owned languages use security-extended analysis; "
        "only the byte-verified glib vendor tree is excluded."
    )
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, RuntimeError, ValueError) as error:
        print(f"CODEQL BOUNDARY FAIL: {error}", file=sys.stderr)
        sys.exit(1)
