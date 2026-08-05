#!/usr/bin/env python3
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Resolve updater-feed entries to downloaded artifacts for signature audit."""

from __future__ import annotations

import json
import os
from pathlib import Path
import re
import sys
from typing import NoReturn
from urllib.parse import unquote, urlsplit


PLATFORM_RE = re.compile(r"[A-Za-z0-9._-]{1,64}\Z")


def fail(message: str) -> NoReturn:
    raise SystemExit(f"updater-feed-audit: {message}")


def main() -> None:
    if len(sys.argv) != 4:
        fail("usage: prepare-updater-signature-audit.py MANIFEST ASSETS_DIR OUTPUT_DIR")

    manifest_path = Path(sys.argv[1])
    assets_dir = Path(sys.argv[2])
    output_dir = Path(sys.argv[3])
    if not manifest_path.is_file() or manifest_path.stat().st_size > 1024 * 1024:
        fail("manifest is missing or exceeds 1 MiB")
    try:
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as error:
        fail(f"cannot parse manifest: {error}")

    platforms = manifest.get("platforms")
    if not isinstance(platforms, dict) or not 1 <= len(platforms) <= 16:
        fail("manifest platforms must be a bounded object")
    output_dir.mkdir(mode=0o700, parents=True, exist_ok=True)

    arguments: list[Path] = []
    for platform, entry in sorted(platforms.items()):
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

        artifact_name = unquote(Path(urlsplit(url).path).name)
        if (
            not artifact_name
            or len(artifact_name.encode("utf-8")) > 255
            or Path(artifact_name).name != artifact_name
        ):
            fail(f"platform {platform} has an unsafe artifact name")
        artifact = assets_dir / artifact_name
        if not artifact.is_file() or artifact.stat().st_size == 0:
            fail(f"downloaded artifact is missing for {platform}: {artifact_name}")

        signature_path = output_dir / f"{platform}.sig"
        signature_path.write_text(f"{signature.strip()}\n", encoding="utf-8")
        os.chmod(signature_path, 0o600)
        arguments.extend((artifact, signature_path))

    for argument in arguments:
        sys.stdout.buffer.write(os.fsencode(argument) + b"\0")


if __name__ == "__main__":
    main()
