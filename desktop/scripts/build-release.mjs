// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: AGPL-3.0-or-later

import { homedir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import { rmSync } from "node:fs";

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
const result = spawnSync(executable, ["build", ...process.argv.slice(2)], {
  cwd: process.cwd(),
  env: { ...process.env, RUSTFLAGS: rustflags },
  shell: process.platform === "win32",
  stdio: "inherit",
});

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}
process.exit(result.status ?? 1);
