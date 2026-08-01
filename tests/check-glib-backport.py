#!/usr/bin/env python3
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
"""Prove the vendored glib crate contains only the reviewed soundness fix."""

from __future__ import annotations

import hashlib
import io
from pathlib import Path, PurePosixPath
import sys
import tarfile
import urllib.request


ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / "desktop" / "src-tauri" / "Cargo.toml"
LOCKFILE = ROOT / "desktop" / "src-tauri" / "Cargo.lock"
VENDOR = ROOT / "desktop" / "src-tauri" / "vendor" / "glib-0.18.5"
ARCHIVE_NAME = "glib-0.18.5.crate"
ARCHIVE_SHA256 = "233daaf6e83ae6a12a52055f568f9d7cf4671dabb78ff9560ab6da230ce00ee5"
ARCHIVE_URL = "https://static.crates.io/crates/glib/glib-0.18.5.crate"
PATCHED_FILE = PurePosixPath("src/variant_iter.rs")
PROVENANCE_FILE = PurePosixPath("PATCH-PROVENANCE.md")
CARGO_MARKER = PurePosixPath(".cargo-ok")
FIX_COMMIT = "b5a4071e439bef2b5eea76c3aa25e5ae84839e34"


def fail(message: str) -> None:
    raise RuntimeError(message)


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def load_archive() -> bytes:
    request = urllib.request.Request(
        ARCHIVE_URL,
        headers={"User-Agent": "Mazzy-VPN-glib-provenance/1"},
    )
    with urllib.request.urlopen(request, timeout=30) as response:
        data = response.read()

    actual = sha256(data)
    if actual != ARCHIVE_SHA256:
        fail(f"upstream archive checksum mismatch from crates.io: {actual}")
    return data


def archive_files(data: bytes) -> dict[PurePosixPath, bytes]:
    files: dict[PurePosixPath, bytes] = {}
    with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as archive:
        for member in archive.getmembers():
            path = PurePosixPath(member.name)
            if path.is_absolute() or ".." in path.parts:
                fail(f"unsafe path in upstream archive: {member.name}")
            if not path.parts or path.parts[0] != "glib-0.18.5":
                fail(f"unexpected archive root: {member.name}")
            relative = PurePosixPath(*path.parts[1:])
            if member.isdir():
                continue
            if not member.isfile():
                fail(f"unexpected non-file archive member: {member.name}")
            extracted = archive.extractfile(member)
            if extracted is None:
                fail(f"cannot read archive member: {member.name}")
            files[relative] = extracted.read()
    return files


def vendored_files() -> dict[PurePosixPath, bytes]:
    if not VENDOR.is_dir():
        fail(f"vendored crate is missing: {VENDOR}")
    return {
        PurePosixPath(path.relative_to(VENDOR).as_posix()): path.read_bytes()
        for path in VENDOR.rglob("*")
        if path.is_file()
    }


def verify_exact_backport(upstream: dict[PurePosixPath, bytes]) -> None:
    vendored = vendored_files()
    allowed_extra = {PROVENANCE_FILE, CARGO_MARKER}
    unexpected = set(vendored) - set(upstream) - allowed_extra
    missing = set(upstream) - set(vendored)
    if unexpected:
        fail(f"unexpected files in vendored glib: {sorted(map(str, unexpected))}")
    if missing:
        fail(f"files missing from vendored glib: {sorted(map(str, missing))}")

    for path, expected in upstream.items():
        if path == PATCHED_FILE:
            continue
        actual = vendored[path]
        if actual != expected:
            fail(f"unreviewed vendored change outside the backport: {path}")

    original = upstream[PATCHED_FILE].decode("utf-8")
    first_old = "            let p: *mut libc::c_char = std::ptr::null_mut();"
    first_new = "            let mut p: *mut libc::c_char = std::ptr::null_mut();"
    second_old = "                &p,"
    second_new = "                &mut p,"
    if original.count(first_old) != 1 or original.count(second_old) != 1:
        fail("upstream glib source no longer matches the reviewed patch context")
    expected_patch = original.replace(first_old, first_new, 1).replace(second_old, second_new, 1)
    actual_patch = vendored[PATCHED_FILE].decode("utf-8")
    if actual_patch != expected_patch:
        fail("VariantStrIter backport differs from upstream fix b5a4071")

    provenance = vendored[PROVENANCE_FILE].decode("utf-8")
    for required in (ARCHIVE_SHA256, FIX_COMMIT, "RUSTSEC-2024-0429"):
        if required not in provenance:
            fail(f"PATCH-PROVENANCE.md is missing {required}")


def verify_cargo_graph() -> None:
    manifest = MANIFEST.read_text(encoding="utf-8")
    patch = '[patch.crates-io]\nglib = { path = "vendor/glib-0.18.5" }'
    if manifest.count(patch) != 1:
        fail("Cargo.toml does not contain the reviewed glib path patch")

    lockfile = LOCKFILE.read_text(encoding="utf-8")
    entries = lockfile.split("[[package]]")[1:]
    glib_entries = []
    for entry in entries:
        fields = {}
        for line in entry.splitlines():
            if " = " not in line:
                continue
            key, value = line.split(" = ", 1)
            if key in {"name", "version", "source", "checksum"}:
                fields[key] = value.strip().strip('"')
        if fields.get("name") == "glib":
            glib_entries.append(fields)

    if len(glib_entries) != 1:
        fail(f"expected one glib entry in Cargo.lock, found {len(glib_entries)}")
    package = glib_entries[0]
    if package.get("version") != "0.18.5":
        fail(f"unexpected vendored glib version: {package.get('version')}")
    if "source" in package or "checksum" in package:
        fail("Cargo.lock still resolves glib from an external registry/source")


def main() -> int:
    upstream = archive_files(load_archive())
    verify_exact_backport(upstream)
    verify_cargo_graph()
    print(
        "GLIB BACKPORT OK: crates.io checksum, exact upstream fix and local Cargo graph verified."
    )
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, RuntimeError, ValueError, tarfile.TarError) as error:
        print(f"GLIB BACKPORT FAIL: {error}", file=sys.stderr)
        sys.exit(1)
