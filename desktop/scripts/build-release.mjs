// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: AGPL-3.0-or-later

import { homedir, tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import {
  chmodSync,
  mkdtempSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs";

const tauriCli = join(
  "node_modules",
  "@tauri-apps",
  "cli",
  "tauri.js",
);
const remapHome = `--remap-path-prefix=${homedir()}=/build/home`;
const rustflags = [process.env.RUSTFLAGS, remapHome].filter(Boolean).join(" ");
const tauriArgs = [tauriCli, "build"];
let updaterConfigDir;
if (process.env.TAURI_SIGNING_PRIVATE_KEY) {
  updaterConfigDir = mkdtempSync(join(tmpdir(), "mazzy-tauri-config-"));
  const updaterConfig = join(updaterConfigDir, "updater.json");
  writeFileSync(
    updaterConfig,
    `${JSON.stringify({ bundle: { createUpdaterArtifacts: true } })}\n`,
    { mode: 0o600 },
  );
  tauriArgs.push("--config", updaterConfig);
}
const metadata = spawnSync("cargo", [
  "metadata",
  "--format-version", "1",
  "--no-deps",
  "--manifest-path", join("src-tauri", "Cargo.toml"),
], {
  cwd: process.cwd(),
  encoding: "utf8",
});
if (metadata.error || metadata.status !== 0) {
  console.error(metadata.stderr || metadata.error?.message || "cargo metadata failed");
  process.exit(metadata.status ?? 1);
}
const releaseDir = join(JSON.parse(metadata.stdout).target_directory, "release");
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
  result = spawnSync(process.execPath, tauriArgs, {
    cwd: process.cwd(),
    env: { ...process.env, RUSTFLAGS: rustflags },
    shell: false,
    stdio: "inherit",
  });
} finally {
  for (const [path, mode] of originalModes) {
    chmodSync(path, mode);
  }
  if (updaterConfigDir) {
    rmSync(updaterConfigDir, { recursive: true, force: true });
  }
}

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}
process.exit(result.status ?? 1);
