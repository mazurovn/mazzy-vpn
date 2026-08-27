// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

import { join } from "node:path";
import { spawnSync } from "node:child_process";

const build = spawnSync(
  process.execPath,
  [join("scripts", "build-release.mjs")],
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
