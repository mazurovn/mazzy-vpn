#!/usr/bin/env python3
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

"""Resolve updater-feed entries to downloaded artifacts for signature audit."""

from __future__ import annotations

import json
import os
from pathlib import Path, PurePosixPath
import re
import sys
from typing import NoReturn
from urllib.parse import unquote, urlsplit


PLATFORM_RE = re.compile(r"[A-Za-z0-9._-]{1,64}\Z")
WORKSPACE = Path.cwd().resolve()
MANIFEST_PATH = WORKSPACE / "updater-metadata" / "latest.json"
ASSETS_DIR = WORKSPACE / "updater-assets"
OUTPUT_DIR = WORKSPACE / "updater-signatures"


def fail(message: str) -> NoReturn:
    raise SystemExit(f"updater-feed-audit: {message}")


def main() -> None:
    if not MANIFEST_PATH.is_file() or MANIFEST_PATH.stat().st_size > 1024 * 1024:
        fail("manifest is missing or exceeds 1 MiB")
    try:
        manifest = json.loads(MANIFEST_PATH.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as error:
        fail(f"cannot parse manifest: {error}")

    platforms = manifest.get("platforms")
    if not isinstance(platforms, dict) or not 1 <= len(platforms) <= 16:
        fail("manifest platforms must be a bounded object")
    if not ASSETS_DIR.is_dir():
        fail("downloaded assets directory is missing")
    asset_entries = list(ASSETS_DIR.iterdir())
    if not 1 <= len(asset_entries) <= 128:
        fail("downloaded assets directory has an invalid entry count")
    if any(entry.is_symlink() or not entry.is_file() for entry in asset_entries):
        fail("downloaded assets directory contains an unsafe entry")
    assets = {entry.name: entry for entry in asset_entries}
    OUTPUT_DIR.mkdir(mode=0o700, parents=True, exist_ok=True)

    arguments: list[Path] = []
    for index, (platform, entry) in enumerate(sorted(platforms.items())):
        if not isinstance(platform, str) or not PLATFORM_RE.fullmatch(platform):
            fail("manifest contains an invalid platform key")
        if not isinstance(entry, dict):
            fail(f"platform {platform} is not an object")
        url = entry.get("url")
        signature = entry.get("signature")
        if not isinstance(url, str) or len(url) > 2048:
            fail(f"platform {platform} has an invalid URL")
        if not isinstance(signature, str) or not 32 < len(signature) <= 8192:
            fail(f"platform {platform} has an invalid signature")

        artifact_name = unquote(PurePosixPath(urlsplit(url).path).name)
        if (
            not artifact_name
            or len(artifact_name.encode("utf-8")) > 255
            or Path(artifact_name).name != artifact_name
        ):
            fail(f"platform {platform} has an unsafe artifact name")
        artifact = assets.get(artifact_name)
        if artifact is None or artifact.stat().st_size == 0:
            fail(f"downloaded artifact is missing for {platform}: {artifact_name}")

        signature_path = OUTPUT_DIR / f"signature-{index:03d}.sig"
        signature_path.write_text(f"{signature.strip()}\n", encoding="utf-8")
        os.chmod(signature_path, 0o600)
        arguments.extend((artifact, signature_path))

    for argument in arguments:
        sys.stdout.buffer.write(os.fsencode(argument) + b"\0")


if __name__ == "__main__":
    main()
