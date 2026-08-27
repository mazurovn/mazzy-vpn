#!/usr/bin/env python3
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

"""Replace draft GitHub API asset URLs with public versioned release URLs."""

from __future__ import annotations

import json
import os
from pathlib import Path
import re
from typing import NoReturn
from urllib.parse import quote


PLATFORM_RE = re.compile(r"[A-Za-z0-9._-]{1,64}\Z")
REPOSITORY_RE = re.compile(r"[A-Za-z0-9_.-]{1,100}/[A-Za-z0-9_.-]{1,100}\Z")
TAG_RE = re.compile(r"desktop-v(?P<version>[0-9]+\.[0-9]+\.[0-9]+)\Z")
WORKSPACE = Path.cwd().resolve()
MANIFEST_PATH = WORKSPACE / "updater-metadata" / "latest.json"
OUTPUT_PATH = WORKSPACE / "updater-metadata" / "latest.canonical.json"
ASSETS_DIR = WORKSPACE / "updater-assets"


def fail(message: str) -> NoReturn:
    raise SystemExit(f"updater-feed-canonicalize: {message}")


def bounded_text(path: Path, limit: int) -> str:
    if path.is_symlink() or not path.is_file():
        fail(f"unsafe or missing file: {path.name}")
    size = path.stat().st_size
    if size == 0 or size > limit:
        fail(f"invalid bounded file: {path.name}")
    try:
        return path.read_text(encoding="utf-8")
    except (OSError, UnicodeError) as error:
        fail(f"cannot read {path.name}: {error}")


def main() -> None:
    repository = os.environ.get("GITHUB_REPOSITORY", "")
    release_tag = os.environ.get("RELEASE_TAG", "")
    tag_match = TAG_RE.fullmatch(release_tag)
    if not REPOSITORY_RE.fullmatch(repository) or tag_match is None:
        fail("invalid repository or Desktop release tag")

    try:
        manifest = json.loads(bounded_text(MANIFEST_PATH, 1024 * 1024))
    except json.JSONDecodeError as error:
        fail(f"cannot parse manifest: {error}")
    if not isinstance(manifest, dict) or manifest.get("version") != tag_match.group(
        "version"
    ):
        fail("manifest version does not match the release tag")
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

    signature_artifacts: dict[str, str] = {}
    for signature_path in asset_entries:
        if not signature_path.name.endswith(".sig"):
            continue
        artifact_name = signature_path.name.removesuffix(".sig")
        artifact = assets.get(artifact_name)
        if artifact is None or artifact.stat().st_size == 0:
            fail(f"signature has no downloaded artifact: {signature_path.name}")
        signature = bounded_text(signature_path, 8192).strip()
        if not 32 < len(signature) <= 8192:
            fail(f"invalid signature asset: {signature_path.name}")
        previous = signature_artifacts.get(signature)
        if previous is not None and previous != artifact_name:
            fail("one signature ambiguously selects multiple artifacts")
        signature_artifacts[signature] = artifact_name
    if not signature_artifacts:
        fail("no signed updater artifacts were downloaded")

    prefix = (
        f"https://github.com/{repository}/releases/download/{release_tag}/"
    )
    for platform, entry in platforms.items():
        if not isinstance(platform, str) or not PLATFORM_RE.fullmatch(platform):
            fail("manifest contains an invalid platform key")
        if not isinstance(entry, dict):
            fail(f"platform {platform} is not an object")
        signature = entry.get("signature")
        if not isinstance(signature, str):
            fail(f"platform {platform} has no signature")
        artifact_name = signature_artifacts.get(signature.strip())
        if artifact_name is None:
            fail(f"platform {platform} does not select a signed release artifact")
        entry["url"] = prefix + quote(artifact_name, safe="._-")

    if OUTPUT_PATH.exists() or OUTPUT_PATH.is_symlink():
        fail("canonical manifest output already exists")
    try:
        OUTPUT_PATH.write_text(
            json.dumps(manifest, ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8",
        )
        os.replace(OUTPUT_PATH, MANIFEST_PATH)
    except OSError as error:
        fail(f"cannot replace manifest: {error}")


if __name__ == "__main__":
    main()
