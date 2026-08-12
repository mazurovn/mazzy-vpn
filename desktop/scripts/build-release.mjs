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
  openSync,
  closeSync,
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
if (process.platform === "linux") {
  // Linux releases are DEB-only. Keep this explicit even if a local Tauri
  // configuration is later widened for development experiments.
  tauriArgs.push("--bundles", "deb");
}
const buildEnv = {
  ...process.env,
  RUSTFLAGS: rustflags,
};
const lockPath = join(tmpdir(), "mazzy-vpn-desktop-release.lock");
let lockFd;
try {
  lockFd = openSync(lockPath, "wx");
  writeFileSync(lockFd, `${process.pid}\n`, { encoding: "utf8" });
} catch (error) {
  console.error(`release build already running (${lockPath}); refusing a concurrent build`);
  process.exit(2);
}
const releaseLock = () => {
  try { closeSync(lockFd); } catch {}
  try { rmSync(lockPath, { force: true }); } catch {}
};
process.once("exit", releaseLock);
process.once("SIGINT", () => { releaseLock(); process.exit(130); });
process.once("SIGTERM", () => { releaseLock(); process.exit(143); });
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
    join("..", "mazzy-agentd"),
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
    env: buildEnv,
    shell: false,
    stdio: "inherit",
    // Bound all release builds so a broken packager cannot leave CI hanging.
    timeout: Number(process.env.MAZZY_RELEASE_TIMEOUT_MS ?? 600_000),
    killSignal: "SIGTERM",
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
  if (result.error.code === "ETIMEDOUT") {
    console.error("Tauri DEB release build timed out");
  }
  console.error(result.error.message);
  process.exit(1);
}
if (result.signal) {
  console.error(`Tauri release build terminated by ${result.signal}`);
  process.exit(1);
}
process.exit(result.status ?? 1);
