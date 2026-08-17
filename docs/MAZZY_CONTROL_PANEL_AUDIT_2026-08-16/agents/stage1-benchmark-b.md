# Role B — benchmark/test-quality review

## Verdict

**Not ready for an unqualified “all gates passed” claim.** The VPN plugin and Mazzy VPN suites passed, but the measured Mazzy Control Panel typecheck failed. The benchmark is useful only as a single-host synthetic microbenchmark, not as throughput, concurrency, production, or cross-host evidence.

## Measured validation facts

| Artifact | Result | Elapsed | Max RSS |
|---|---:|---:|---:|
| `.mazzy/work/results/mazzy-consilium-2026-08-16/control-panel-tests.log` | 52/52 PASS | 6.65 s | 133,144 KB |
| `.../control-panel-typecheck.log` | **FAIL, exit 2** | 7.65 s | 700,672 KB |
| `.../vpn-plugin-tests.log` | 19/19 PASS | 1.30 s | 85,660 KB |
| `.../vpn-plugin-typecheck.log` | PASS | 3.41 s | 342,432 KB |
| `.../mazzy-vpn-tests.log` | 105/105 PASS | 279.15 s | 45,388 KB |

These counts belong to different runners and should not be combined into “176 independent tests.”

## Concrete findings

### 1. High — Control Panel compiler gate fails

**Evidence:**  
`control-panel-typecheck.log` reports exit 2 and three errors in `mazzy-control-panel/src/storage-policy.ts`, where `rule.maxAgeSeconds` remains `unknown`.

The 52 runtime tests passing does not override a failed typecheck. The report’s executive wording at `docs/RESEARCH_MAZZY_CONTROL_PANEL_VPN_ORCHESTRATION_2026-08-16.ru.md:23` must remain explicitly scoped to the VPN plugin; it cannot imply that current Control Panel validation is fully green.

**Smallest safe fix:** Check `typeof maxAgeSeconds === "number"` before numeric predicates or otherwise narrow once into a number, then rerun Control Panel typecheck and tests.

---

### 2. High — Cross-process stale-revision update can be acknowledged as successful

**Evidence:**  
`../PI/mazzy-control-panel/src/store.ts:64` calls `requireRevision` before entering `BEGIN IMMEDIATE`. Its subsequent `UPDATE ... WHERE revision=?` does not check `changes`. `src/store.ts:110-111` confirms the transaction starts only afterward.

Two stores/processes can therefore both read revision N; the first commits N+1, while the second updates zero rows but still emits an event, commits, and returns the already-current task. Existing tests exercise sequential stale revisions, not this cross-process race. The benchmark is entirely sequential.

**Smallest safe fix:** Move revision lookup and transition validation inside the immediate transaction and require exactly one updated row; otherwise throw a revision conflict before recording an event. Add a two-store/process barrier test proving exactly one same-revision PATCH succeeds.

---

### 3. High — Proposed two-reviewer validation contract is not implemented

**Evidence:**  
The report requires `independent_reviewers >= 2` and “минимум 2” at report lines 612 and 841. Current `src/store.ts:126` accepts the **latest single** independent reviewer PASS. Tests in `test/store.test.ts` do not prove that one PASS is insufficient and two distinct reviewer attestations are required.

**Smallest safe fix:** Either clearly label the two-reviewer rule as future policy, or enforce two distinct current-acceptance reviewer bindings before DONE. Add negative one-PASS and positive two-PASS tests; preserve the rule that a later current FAIL blocks closure.

---

### 4. Medium — Percentile labels use a nonstandard order-statistic convention

**Evidence:**  
`.mazzy/work/benchmarks/mazzy-control-benchmark.mjs` calculates:

```text
values[floor(n × p)]
```

For zero-indexed samples this selects:

- p50: item 101 of 200, not the conventional nearest-rank item 100;
- p95: item 191 of 200;
- p95 for n=50: item 48, effectively the 96th percentile rank.

The JSON does not define this convention and retains no raw samples, so conventional p50/p95 values cannot be reconstructed.

**Smallest safe fix:** Use a documented method such as nearest rank `ceil(n × p) - 1`, add unit cases for even and small sample sizes, and retain raw samples or a histogram.

**Correct interpretation of current values:** They are per-operation sample latencies from one run. p50 is a median-like upper order statistic; p95 is the script-defined tail statistic; `max` is one observed outlier—not a bound, timeout, SLO, or repeatable worst case.

---

### 5. Medium — Benchmark is not reproducibly bound to source or environment

**Evidence:**  
`control-panel-benchmark.json` records counts and synthetic/sequential notes, but not:

- timestamp or benchmark-script hash;
- source commit/diff digest;
- dirty-worktree state;
- Node/SQLite version;
- OS, CPU, memory, filesystem, SQLite pragmas, or host load;
- warmup and independent-process repetitions;
- raw samples or confidence intervals.

The script imports sibling source directly. Direct inspection found that source worktree dirty, so the repository commit alone cannot identify the measured implementation.

**Smallest safe fix:** Emit a metadata envelope with source and script digests, dirty diff digest, tool/runtime and environment details, database settings, seed, warmup, repetitions, and raw-sample artifact hash. Refuse baseline comparison if code identity is missing.

---

### 6. Medium — Methodology cannot support throughput or concurrent-capacity claims

**Evidence:**  
The JSON correctly says “Sequential localhost benchmark,” “Synthetic project-local database,” and “Not a cross-host throughput benchmark.” The script performs each awaited operation serially in one process.

Additional limitations:

- no concurrency or second DB process;
- no cold/warm separation;
- no randomized operation ordering;
- create samples progressively grow the DB from 0 to 499 tasks;
- list/detail/snapshot run only at approximately 500-task cardinality;
- HTTP fetch likely reuses connections;
- no repeated forks or uncertainty estimate;
- no event/comment/evidence-heavy dataset;
- no slow SSE client or backpressure workload.

**Smallest safe fix:** Keep these numbers as a labeled synthetic microbenchmark only. Add separate mixed-load, multi-process, and SSE benchmarks before publishing throughput or scaling conclusions.

---

### 7. Medium — Benchmark “functional PASS” fields are under-asserted

**Evidence:**  
The benchmark checks authenticated PATCH only by HTTP status. Comment idempotency checks matching IDs but not exactly one durable row/event or changed-payload conflict. These are smoke checks, not replacements for the functional suite.

**Smallest safe fix:** Rename them `smokeChecks`, or assert resulting state/revision, durable row/event cardinality, replay identity, and changed-payload rejection.

---

### 8. Medium — Important security/performance negative tests are absent

**Evidence:**  
Control tests cover Host rejection, token omission, SSE authentication, idempotency and bounded task-detail arrays. Direct search found no explicit tests for:

- foreign `Origin` rejection on mutations;
- malformed and over-64-KiB JSON;
- authentication across every actual API route;
- slow request bodies and aborted uploads;
- token/capability absence from errors and logs;
- SSE slow-consumer/backpressure memory bounds;
- equal-revision cross-process mutation races.

The unauthorized latency benchmark tests only missing-token rejection. It is not a token-timing or security benchmark.

**Smallest safe fix:** Add route-table-driven negative tests and bounded slow-client/backpressure tests. Keep security correctness separate from latency reporting.

---

### 9. Low — Published report exposes a host-local implementation path

**Evidence:**  
The report metadata at line 9 includes an absolute user/home path. No capability URL or obvious credential was found in the measured outputs, and the benchmark does not print its generated token.

**Smallest safe fix:** Replace the absolute location with a project-relative label such as `PI/MY_PLUGINS/pi-vpn-recovery`.

## Arithmetic audit

The existing orchestration arithmetic is correct within rounding:

- duration: 2,397,322 ms = **39.9554 min**, reported 39.96;
- input + output: 857,329 + 114,256 = **971,585**;
- failed cost share: **28.9380%**, reported 28.94%;
- writer/fix-writer cost share: **12.8163%**, reported 12.82%;
- successful read-only share of successful cost: **81.9646%**;
- total cost: **$5.7191252**;
- over a $2 reference budget: **$3.7191252 / 185.956%**;
- total as a percentage of $2: **285.956%**;
- test growth 11→19: 8/11 = **72.727%**.

No cache-read double counting was found: the report presents 5,379,242 cache-read tokens separately from input+output.

Wording corrections:

- “39.96 agent-minutes” is summed run time, not wall-clock time; parallel runs overlap.
- `$1.8–$3.0` is an unvalidated target, not an evidence-based expectation until the optimized pipeline is measured.
- “cost per changed file” and percentage test growth are descriptive, not quality/efficiency KPIs.
- Replace “escaped blockers = 0” with “blockers reported by final validators = 0”; no production escape study was performed.
- Numerical 9/10-style ratings should be marked as expert judgment, not measured KPI.

## Benchmark results and allowed interpretation

| Operation | n | p50 ms | p95 ms | max ms | Scope |
|---|---:|---:|---:|---:|---|
| Store create | 500 | 1.362 | 2.523 | 20.723 | Progressive DB growth |
| Store list | 200 | 2.348 | 3.545 | 6.194 | About 500 synthetic tasks |
| Store detail | 200 | 0.295 | 0.522 | 1.067 | Repeated same task |
| HTTP snapshot | 100 | 4.846 | 8.405 | 28.728 | Localhost, serial |
| Unauthorized snapshot | 50 | 0.583 | 0.925 | 1.309 | Missing-token rejection |

All ordering invariants `min ≤ p50 ≤ p95 ≤ max` and `min ≤ mean ≤ max` hold. Without samples, the individual summary statistics cannot be independently recomputed.

## Improved KPI/benchmark plan

### Reproducibility

1. Capture source commit, dirty diff digest, script hash and schema.
2. Capture Node, TypeScript, SQLite, OS/kernel, CPU, memory, filesystem and SQLite pragmas.
3. Use deterministic dataset seeds and cardinalities: 0, 100, 500, 5k and 50k tasks.
4. Run warmup separately, then at least 10 fresh-process repetitions.
5. Store raw samples or HDR histograms with artifact hashes.
6. Use documented nearest-rank percentiles and confidence intervals.

### Workloads

| Lane | Workload |
|---|---|
| Micro-latency | create/list/detail/snapshot at fixed cardinalities |
| Mixed load | 80/20 read/write at concurrency 1, 4, 16 and 64 |
| Optimistic concurrency | Two stores/processes PATCH same revision |
| SSE | 1, 10 and 100 clients; slow consumer, reconnect and reset |
| Durability | Lost response, restart, DB busy/locked and interrupted notification |
| Security negatives | Host, Origin, auth, body bounds, malformed JSON, redaction |
| Dataset scaling | Long descriptions plus events/comments/evidence histories |

### KPIs

- p50/p95/p99 latency with confidence intervals;
- throughput and successful-operation rate;
- expected conflict rate versus silent lost-update rate;
- SSE commit-to-delivery lag and reconnect recovery time;
- error/timeout rate;
- peak RSS, CPU and database growth;
- test flake rate over repeated clean processes;
- exact compiler/test gate status;
- evidence completeness and task closure latency.

Regression thresholds should combine absolute product SLOs with statistically stable baseline deltas; single-run `max` should not gate releases.

## Functional coverage matrix

| Area | Existing evidence | Required delta |
|---|---|---|
| Control build gate | 52 tests pass; typecheck fails | Fix compiler error; make both mandatory |
| CRUD/FSM | Basic create/update, stale sequential revision, UI transitions | Cross-process CAS race and all terminal paths |
| Idempotency | Create replay/conflict, comments, GO coalescing | Concurrent replay across stores and restart |
| Auth/privacy | Host rejection, token omission, no token URL | Foreign Origin, all-route auth matrix, error/log redaction |
| Request bounds | Task ID array bound | Malformed/oversized/slow/aborted body tests |
| SSE | Two clients, replay, reset, cleanup | Backpressure, snapshot/stream race, reconnect under writes |
| Evidence/review | Current reviewer PASS/FAIL and comment isolation | Enforce or retract minimum-two-reviewer contract |
| Storage policy | Dry-run/apply, symlinks, scan/quota bounds | Typecheck; permission/disk-full/interrupted-write faults |
| VPN recovery | 19 mocked tests cover quorum, IPv6 warning and leases | Deep-mode positive/boundary cases, irrelevant IPv6 absence, malformed locks, probe timeout/disable races |
| Mazzy VPN | 105 shell scenarios pass | Explicitly label fake-binary/synthetic nature; staging VM tests require separate approval |
| Live/destructive behavior | Safe doctor evidence only | No destructive reconnect claim; retain as authorized staging-only test |

## Residual limitations

- No benchmark or long suite was rerun.
- No raw benchmark samples exist.
- The cross-process update race was established from direct source inspection but not reproduced with a process-barrier harness.
- VPN plugin tests mock command execution; they do not prove real network, systemd, sudo, DNS or provider behavior.
- No destructive VPN reconnect was performed.
- Current numbers do not establish production throughput, cross-host behavior, or NFS/shared-filesystem lock safety.

```acceptance-report
{
  "criteriaSatisfied": [
    {
      "id": "criterion-1",
      "status": "satisfied",
      "evidence": "Concrete severity findings cite the report, measured logs, benchmark script/JSON, Control Panel source/tests, and VPN plugin source/tests; residual risks and smallest fixes are included."
    }
  ],
  "changedFiles": [],
  "testsAddedOrUpdated": [],
  "commandsRun": [
    {
      "command": "Parse benchmark JSON and measured log exit/elapsed/RSS fields",
      "result": "passed",
      "summary": "All five logs parsed; four exit 0 and Control Panel typecheck exits 2."
    },
    {
      "command": "Recompute orchestration metric totals and percentages from vpn-consilium-metrics.json",
      "result": "passed",
      "summary": "Reported totals, cost shares, budget percentages and duration rounding were confirmed."
    },
    {
      "command": "Inspect source identity, worktree status, runtime version and artifact metadata",
      "result": "passed",
      "summary": "Benchmark source was found unbound to a dirty source state; no staged files were observed."
    }
  ],
  "validationOutput": [
    "Control Panel tests: 52 pass; typecheck: exit 2.",
    "VPN plugin tests: 19 pass; typecheck: pass.",
    "Mazzy VPN tests: 105 pass.",
    "Benchmark summaries satisfy ordering invariants but cannot be independently recomputed without raw samples."
  ],
  "residualRisks": [
    "No long suites or benchmark were rerun.",
    "No raw benchmark samples or environment manifest exist.",
    "Cross-process stale-revision race requires a process-barrier regression test.",
    "No destructive or production VPN validation was performed."
  ],
  "noStagedFiles": true,
  "diffSummary": "Read-only review; no files changed.",
  "reviewFindings": [
    "high: control-panel-typecheck.log - Control Panel compiler gate exits 2.",
    "high: mazzy-control-panel/src/store.ts:64 - updateTask performs revision validation before its transaction and ignores a zero-row CAS update.",
    "high: report lines 612/841 versus store.ts:126 - proposed two-reviewer gate is not enforced.",
    "medium: mazzy-control-benchmark.mjs - nonstandard percentile indexing and no raw samples.",
    "medium: control-panel-benchmark.json - source/environment identity and concurrent methodology are absent.",
    "medium: Control Panel tests - security, body-bound and SSE backpressure negative coverage is incomplete.",
    "low: report line 9 - host-local absolute path should be redacted."
  ],
  "manualNotes": "Existing orchestration cost arithmetic is correct; the optimized cost range and quality scores must be labeled targets/judgment rather than measured outcomes."
}
```

TASK_COMMENT
role: benchmark/test-quality reviewer  
run ID: none  
accomplishment: audited report arithmetic, five measured logs, benchmark JSON/script, and relevant Control Panel/VPN tests; supplied corrections, KPI plan, and coverage matrix  
checks/verdict: arithmetic confirmed; VPN gates pass; Control Panel typecheck fails; benchmark is synthetic/sequential only  
blockers/next step: fix typecheck and transactional revision CAS, resolve the two-reviewer contract, then capture a reproducible concurrent benchmark