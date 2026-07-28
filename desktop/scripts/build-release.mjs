// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: AGPL-3.0-or-later

import { homedir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import { chmodSync, rmSync, statSync } from "node:fs";

const executable = join(
  "node_modules",
  ".bin",
  process.platform === "win32" ? "tauri.cmd" : "tauri",
);
const remapHome = `--remap-path-prefix=${homedir()}=/build/home`;
const rustflags = [process.env.RUSTFLAGS, remapHome].filter(Boolean).join(" ");
const releaseDir = join("src-tauri", "target", "release");
rmSync(join(releaseDir, "bundle"), {
  recursive: true,
  force: true,
});
for (const binary of ["mazzy-vpn-desktop", "mazzy-vpn-desktop.exe"]) {
  rmSync(join(releaseDir, binary), { force: true });
}

const runtimeExecutables = process.platform === "win32"
  ? []
  : [
    join("..", "mazzy-vpn"),
    join("..", "vpnctl"),
    join("..", "install.sh"),
    join("..", "setup_amnezia_vpn.sh"),
    join("..", "stop_amnezia_vpn.sh"),
  ];
const originalModes = new Map();
let result;
try {
  for (const path of runtimeExecutables) {
    const mode = statSync(path).mode & 0o7777;
    originalModes.set(path, mode);
    chmodSync(path, 0o755);
  }
  result = spawnSync(executable, ["build"], {
    cwd: process.cwd(),
    env: { ...process.env, RUSTFLAGS: rustflags },
    shell: process.platform === "win32",
    stdio: "inherit",
  });
} finally {
  for (const [path, mode] of originalModes) {
    chmodSync(path, mode);
  }
}

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}
process.exit(result.status ?? 1);
