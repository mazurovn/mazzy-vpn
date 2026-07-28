// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: AGPL-3.0-or-later

import { join } from "node:path";
import { spawnSync } from "node:child_process";

const [command, ...args] = process.argv.slice(2);
if (command !== "build") {
  console.error("tauri-audited only permits the Tauri build command");
  process.exit(2);
}

const build = spawnSync(
  process.execPath,
  [join("scripts", "build-release.mjs"), ...args],
  {
    cwd: process.cwd(),
    stdio: "inherit",
  },
);
if (build.error) {
  console.error(build.error.message);
  process.exit(1);
}
if (build.status !== 0) {
  process.exit(build.status ?? 1);
}

if (process.platform === "linux") {
  const audit = spawnSync(join("..", "tests", "check-linux-packages.sh"), {
    cwd: process.cwd(),
    stdio: "inherit",
  });
  if (audit.error) {
    console.error(audit.error.message);
    process.exit(1);
  }
  if (audit.status !== 0) {
    process.exit(audit.status ?? 1);
  }
}
