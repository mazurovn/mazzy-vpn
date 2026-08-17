# Architecture/control-plane audit delta

**Role:** Architecture/control-plane lane  
**Run ID:** none visible  
**Mode:** Read-only; no source, documentation, database, index, or remote changes.

## 1. Executive delta for the writer

The existing report is directionally strong, but fresh inspection changes several conclusions:

1. **The stored three-error typecheck failure is historical, not current.**  
   The saved log records three errors in `src/storage-policy.ts`, but that source was modified later. Fresh `npm run typecheck` passes, and the current full suite passes **55/55**, not 52/52. The three storage-policy files remain untracked, so release provenance is still unresolved.

2. **A critical acceptance-binding flaw was newly reproduced.**  
   Editing task content in `REVIEW` leaves it in `REVIEW`; the old completed worker can then be used to import a report for the new acceptance revision, assign a reviewer, record PASS, and reach `DONE`.

3. **A high-severity control flaw was newly reproduced.**  
   A non-content update while `RUNNING` increments the task revision without updating the active binding. Subsequent `PAUSE`/`STOP` requests are rejected as not targeting the current binding.

4. **Evidence materialization requires a safer design than the report currently proposes.**  
   Generic current-acceptance `FAIL` evidence does not block `DONE`; only reviewer-bound evidence participates in the closure query. Automatically converting child reports into evidence would therefore be unsafe without first defining evidence kinds and closure policy.

5. **The dashboard does not hold an in-memory task snapshot.**  
   `/api/snapshot` queries SQLite live. The historical dashboard/DB count difference does not prove a stale server-side snapshot. However, multiple live server processes against the same DB were confirmed, and SSE notifications are process-local; cross-process writes can remain invisible to a connected browser until a manual or visibility refresh.

6. **Budget fields are validated configuration, not enforcement.**  
   `defaultUsd`, `highRiskUsd`, and `maxChildrenPerWave` are not consumed by routing or dispatch. Since `@mazzy/control` does not own child execution or provider accounting, a hard-budget implementation requires a parent/runtime attestation seam rather than a scheduler inside the control panel.

---

## 2. Fresh measured state versus stored evidence

| Check | Stored artifact | Fresh result | Interpretation |
|---|---|---|---|
| Control tests | 52/52, 6.65 s | **55/55**, ~6.08 s | Three storage-policy tests are now included |
| Control typecheck | 3 errors | **PASS** | Source changed after the failing log |
| Focused storage tests | Not separate | **3/3 PASS** | Runtime validation is covered |
| PI control DB | Historical 62 tasks | **74 tasks**, `quick_check=ok` | Backlog continued changing |
| Target VPN task | REVIEW, 1 binding, 0 evidence | Same state and coverage | Report assertion remains current |
| Target-repository control DB | Not distinguished | Empty DB, `quick_check=ok` | Project-root DB identity matters |
| Dashboard listeners | Historical ports | Two live PI-owned listeners; default port absent | Multi-owner condition is current |

### Typecheck chronology

Measured facts:

- The stored typecheck log was written before the current `storage-policy.ts`.
- It records:
  - two `TS18046` errors because `rule.maxAgeSeconds` remained `unknown`;
  - one `TS2322` assignment of `unknown` to `number`.
- Current `mazzy-control-panel/src/storage-policy.ts:60-62` first extracts:
  ```ts
  const maxAgeSeconds = isRecord(rule) ? rule.maxAgeSeconds : undefined;
  ```
  then performs `typeof maxAgeSeconds !== "number"` validation before assignment.
- Fresh typecheck passes.

This is the smallest safe narrowing pattern. It avoids depending on control-flow narrowing of a repeatedly accessed property from `Record<string, unknown>`.

**Required report correction:** do not call the three errors a current failure. Record them as a reproduced historical gate that is fixed in the present worktree but not yet attestably integrated because these files are untracked:

- `mazzy-control-panel/src/storage-policy.ts`
- `mazzy-control-panel/test/storage-policy.test.ts`
- `mazzy-control-panel/resources/storage-policy.json`

---

## 3. Confirmed findings

### CP-A01 — Stale worker acceptance can be promoted to a new acceptance revision

**Severity: Critical**  
**Status: Confirmed with isolated reproduction**

**Seams**

- `mazzy-control-panel/src/store.ts:64` — `updateTask`
- `mazzy-control-panel/src/store.ts:66` — reviewer `assignRun`
- `mazzy-control-panel/src/store.ts:71` — `importReviewReport`
- `mazzy-control-panel/src/store.ts:120` — `latestCompletedWorker`
- `mazzy-control-panel/test/store.test.ts`

**Measured behavior**

1. Worker completes acceptance revision 1.
2. Task title is edited while in `REVIEW`.
3. Task remains `REVIEW` at acceptance revision 2.
4. `importReviewReport` accepts the revision-1 worker and writes its report under revision 2.
5. Reviewer assignment uses the stale completed worker.
6. Reviewer PASS permits `DONE`.

Observed result: old worker acceptance `1`, current acceptance `2`, stale report import accepted, closure reached `DONE`.

**Cause**

- Content edits force `RUNNING → READY`, but not `REVIEW → READY`.
- `importReviewReport` checks completed worker/task identity but not the worker’s acceptance revision or digest.
- `latestCompletedWorker` does not filter by current acceptance.
- Reviewer assignment consequently accepts a stale worker.

**Smallest safe fix**

1. Any title/description edit in `REVIEW` must transition to `READY`.
2. `importReviewReport`, reviewer assignment, and reviewer evidence must require:
   - worker `acceptanceRevision === task.acceptanceRevision`;
   - worker `acceptanceDigest === task.acceptanceDigest`.
3. Query the latest completed worker for the **current acceptance**, not merely the latest worker by time.

**Acceptance test**

Add a store test that completes revision 1, edits content in `REVIEW`, and asserts:

- task becomes `READY`;
- stale report import is rejected;
- reviewer assignment is rejected until a new worker completes;
- task cannot reach `DONE` using old worker/report data.

---

### CP-A02 — Generic current FAIL evidence does not block closure

**Severity: High**  
**Status: Confirmed with isolated reproduction**

**Seams**

- `mazzy-control-panel/src/store.ts:73-74` — reviewer and generic evidence
- `mazzy-control-panel/src/store.ts:125` — `requireCurrentReviewerPass`
- `mazzy-control-panel/src/index.ts` — only reviewer evidence is exposed through `ops_assignment`
- `mazzy-control-panel/test/store.test.ts`

**Measured behavior**

After reviewer PASS, a generic `typecheck: FAIL` was recorded at the same acceptance revision. `updateTask(... DONE ...)` still succeeded.

**Cause**

`requireCurrentReviewerPass` joins only evidence with a reviewer binding. Unbound deterministic evidence is neither required nor checked for failure.

**Implication**

The report’s proposed automatic evidence materialization would be unsafe if it merely inserted generic rows: those rows would not influence closure.

**Smallest safe recommendation**

Before adding automatic materialization:

1. Define evidence classes:
   - deterministic parent-observed check;
   - independent reviewer verdict;
   - child-reported claim;
   - informational artifact.
2. Define required kinds per acceptance policy.
3. Block closure on any current required `FAIL` or unresolved `UNCERTAIN`.
4. Keep child self-reports non-authoritative until parent-observed or independently verified.
5. Add a parent-only import path for deterministic evidence with bounded payload and provenance.

**Acceptance tests**

- Current deterministic FAIL blocks `DONE`.
- Stale FAIL does not block a later acceptance revision.
- Child report alone cannot produce authoritative PASS.
- Required evidence omission leaves an explicit closure reason.
- Latest independent reviewer PASS remains necessary.

---

### CP-A03 — RUNNING revision drift disables PAUSE/STOP

**Severity: High**  
**Status: Confirmed with isolated reproduction**

**Seams**

- `mazzy-control-panel/src/store.ts:64` — every update increments revision
- `mazzy-control-panel/src/store.ts:124` — `controlApplicable`
- `mazzy-control-panel/src/server.ts:140-153` — orchestration endpoint
- `mazzy-control-panel/test/store.test.ts`
- `mazzy-control-panel/test/server.test.ts`

**Measured behavior**

A RUNNING task with binding revision 3 received a priority update and became revision 4. A STOP request targeting the still-active run was rejected because the binding remained revision 3.

This also occurs with a no-op update because `updateTask` increments revision unconditionally.

**Smallest safe fix**

The HTTP request already validates the task’s current expected revision and targets the active run ID. Therefore, `controlApplicable` should not additionally require binding assignment revision equality for an otherwise current active target. Retain:

- current expected task revision;
- active binding state;
- exact target run ID;
- task identity.

Alternatively, update active binding lifecycle revision atomically, but do not overwrite its immutable assignment epoch.

**Acceptance tests**

- Priority/risk update while RUNNING does not disable STOP.
- No-op update does not disable PAUSE.
- A superseded or wrong run ID remains rejected.
- A request created before a later task revision still fails optimistic concurrency at claim.

---

### CP-A04 — Budget policy is descriptive only

**Severity: High for expectation mismatch; not itself a runtime bypass**  
**Status: Confirmed**

**Seams**

- `mazzy-control-panel/src/routing.ts:17, 50-57, 93`
- `mazzy-control-panel/src/routing.ts:32-40` — `RouteResult` has no budget
- `mazzy-control-panel/src/routing.ts:126-154` — route selection
- `.pi/mazzy/routing.json`
- `mazzy-control-panel/test/routing.test.ts`

The budget fields are validated and returned as part of the loaded policy, but:

- `route()` does not consume or return them;
- no task budget is persisted;
- no reservation/cost ledger exists;
- no dispatch gate exists;
- `maxChildrenPerWave` is not enforced by `@mazzy/control`.

**Report correction**

The observed `$5.719125` versus configured `$2` proves the field was not a hard task cap. It does not by itself establish whether some external parent treated it as per-run, per-wave, advisory, or ignored.

**Smallest measurable slice**

1. Extend policy schema with explicit:
   - `scope`;
   - `enforcement`;
   - currency;
   - whether failed cost is charged.
2. Return the selected budget declaration from `ops_route`.
3. Persist task-level budget and parent-attested reservations/actual usage.
4. Refuse to label policy “hard” until the parent/runtime supplies atomic pre-launch reservation and post-run actual-cost attestation.

Do **not** put child scheduling or provider billing logic inside the dashboard server.

**Acceptance gate**

A task with a hard remaining balance below the requested reservation cannot obtain a launch reservation without a durable override event; failed runs consume the same ledger; resume uses the original task balance.

---

### CP-A05 — Late-provider-error salvage is absent, but belongs primarily to the child runtime

**Severity: High operationally**  
**Status: Confirmed absence; proposed ownership is architectural**

Measured metadata confirms two failed provider runs accumulated substantial turns, tools, time, and cost before a terminal 403. Their final output artifacts contain only a start sentence or the terminal error—not structured findings/checkpoints.

`@mazzy/control` currently has only generic activity fields and no checkpoint, attempt, or fallback-chain model.

**Smallest safe ownership split**

- **pi-subagents/runtime:** checkpoint capture, provider error classification, attempt continuation, same-provider circuit breaker.
- **@mazzy/control:** durable parent-attested references to checkpoints, attempts, fallback reasons, and final disposition.
- Checkpoint records should contain digest, phase, relative artifact identifier, and remaining work—not transcript contents or credentials.

**Acceptance tests**

- Late non-retryable 403 exposes the latest structured checkpoint.
- Same provider is not retried automatically.
- Fallback references the prior attempt and checkpoint digest.
- No raw transcript, prompt, capability URL, or credential enters control events.

---

### CP-A06 — Evidence is not automatically materialized

**Severity: High for automation; current behavior is internally honest**  
**Status: Confirmed**

Completion creates a report and moves the worker to completed/REVIEW. It does not create deterministic evidence. Reviewer evidence requires a separately active reviewer binding.

For the VPN task, fresh DB inspection still shows:

- state `REVIEW`;
- one completed worker binding;
- one report;
- zero evidence rows.

**Nuance for the report**

This is not evidence silently being lost by a parser: no materialization pipeline exists. Comments correctly remain non-evidence.

**Recommendation**

Add parent-observed deterministic evidence import only after CP-A02’s policy fix. Do not infer PASS merely because an acceptance-report JSON field says “passed.”

---

### CP-A07 — Task closure is explicit and currently under-specified, not simply “broken”

**Severity: Medium/High process gap**  
**Status: Confirmed behavior**

- `DONE` is excluded from UI transitions.
- `DONE` requires the latest current independent reviewer-bound PASS.
- Reviewer PASS does not automatically transition the task.
- A parent must explicitly call task update after evidence.

The VPN task remained in REVIEW primarily because no bound reviewer evidence exists, not because a valid reviewer PASS failed to auto-close it.

**Recommendation**

Prefer an atomic parent-only `decide-completion` operation that:

1. evaluates required evidence;
2. records the decision and reasons;
3. closes only when policy is satisfied.

Do not make every reviewer PASS automatically close a task before evidence policy exists.

---

### CP-A08 — Run binding coverage is an integration gap

**Severity: Medium**  
**Status: Confirmed**

The schema can retain multiple historical worker/reviewer bindings, but the VPN task has one workflow-level worker binding for twelve measured child runs. Child metadata and report artifacts exist outside the task graph.

**Smallest safe extension**

Add immutable run-attempt/child-node records beneath a binding:

- workflow run ID;
- child run ID;
- parent binding ID;
- role;
- requested and actual model;
- attempt number;
- lifecycle;
- checkpoint/artifact references.

The parent remains the attestation authority; child processes must not write control state directly.

---

### CP-A09 — Requested and actual model identities are conflated

**Severity: Medium**  
**Status: Confirmed**

- Routing returns one selected model.
- `run_bindings.model` is a mutable single string.
- `updateRunActivity` overwrites that field.
- Agent names such as `ops-review-sonnet` encode a desired model while measured runs used another provider/model.
- The VPN worker binding reports Terra, while its review report has a GPT model under the same worker agent, demonstrating ambiguous report-model semantics.

**Smallest migration**

Add nullable immutable fields/attempt rows:

- `requested_model`;
- `actual_model`;
- `provider`;
- `fallback_reason`;
- `attempt_index`.

Do not backfill legacy `model` as requested or actual without evidence; label it `legacy_model` during migration.

---

### CP-A10 — Dashboard count discrepancy needs corrected attribution

**Severity: High design gap; historical count mismatch remains unproven causally**  
**Status: Partially confirmed**

**What source refutes**

`mazzy-control-panel/src/server.ts:130` calls `store.snapshot()` for every request; task data is not held in a server-side snapshot cache.

**What is confirmed**

- Two live server processes currently run from the PI project.
- Neither inspected process has an explicit DB override.
- Both therefore resolve to the same Git-root DB.
- `OpsStore.subscribeEvents` uses an in-process listener set.
- A write committed by process B does not notify process A’s SSE listeners.
- The browser stops fallback polling while SSE is healthy.

Thus a browser connected to A can miss B’s writes until manual refresh or visibility refresh even though a direct `/api/snapshot` request would be current.

**Smallest safe recommendation**

Prefer one server owner per project DB, with a token-free local registry containing only:

- project identity digest;
- endpoint;
- PID plus process-start nonce;
- owning session identifier;
- last heartbeat.

Never persist the capability token. A non-owner should report the live owner rather than start another server.

**Acceptance tests**

- Two processes cannot both become canonical owners for one DB.
- Stale registry entries recover safely despite PID reuse.
- Cross-process mutation either reaches SSE or forces cursor-based snapshot reset.
- Registry output contains no token or absolute sensitive path.

---

### CP-A11 — Port discovery is session-local

**Severity: Medium**  
**Status: Confirmed**

Seams:

- `mazzy-control-panel/src/server.ts:58-70`
- `mazzy-control-panel/src/index.ts:331-338`
- `mazzy-control-panel/src/project.ts:9-17`

The default remains 4319, while current live owners use other ports. `/mazzy-server status` only knows whether the current session owns its local server object. No project-wide owner registry exists.

Historical “dead port” evidence should be worded carefully: a short-lived command may have started a server that disappeared when its process exited. The durable issue is lack of owner/lifetime discovery, not proof that `canonicalUrl()` returned an already-dead socket at that exact instant.

---

### CP-A12 — Artifact lineage is absent

**Severity: Medium**  
**Status: Confirmed**

Eight VPN stage reports exist with distinguishable filenames, but the control DB stores no:

- relative artifact reference;
- content digest;
- phase;
- current/superseded status;
- `supersedes` relation.

Reports for 14, 18, and 19 tests can therefore all look independently final. Artifacts also remain under a legacy output location while the new storage policy describes `work/outputs`.

**Smallest safe slice**

Create a bounded artifact-reference manifest/table with:

- task, acceptance revision, binding/attempt;
- kind and phase;
- relative identifier;
- SHA-256;
- state: current/superseded/failed;
- optional predecessor ID.

Never store arbitrary absolute host paths or artifact payloads in events.

---

### CP-A13 — SQLite migration is not concurrency-safe enough for multi-owner startup

**Severity: Medium**  
**Status: Confirmed design risk; race not reproduced**

`OpsStore.migrate()` performs repeated table inspection, `ALTER TABLE`, backfills, and index creation without one explicit numbered migration transaction. Two processes starting against an older schema could both observe a missing column before one `ALTER` wins.

`schema_migrations` records versions 1 and 8 but does not model every applied step.

**Recommendation**

- One `BEGIN IMMEDIATE` migration transaction.
- Ordered, individually recorded migration versions.
- Backup/restore rehearsal before non-additive migration.
- Two-process legacy-DB startup regression.
- Additive nullable columns first; retain old columns for rollback.

---

## 4. Report assertions: retain, weaken, or correct

| Existing assertion | Verdict |
|---|---|
| 12 measured child runs, 10 success/2 failed, cost/token totals | Retain; metrics artifact supports it |
| Late provider failures consumed substantial work | Retain |
| Meaningful partial findings could have been salvaged | Weaken: transcripts had activity, but final checkpoint artifacts did not contain usable findings |
| VPN task has 1 binding, 0 evidence, 1 report, remains REVIEW | Retain; freshly confirmed |
| `$2` budget was exceeded | Reword: configured field was not a hard task budget |
| Dashboard held a stale in-memory snapshot | Correct: server snapshots are live DB queries |
| Dashboard/DB mismatch proves a control-plane bug | Weaken: historical measurements were not simultaneous; cross-process SSE blindness is the confirmed defect |
| Final no-blocker comments should automatically close task | Correct: current contract requires bound reviewer evidence plus explicit parent closure |
| Structured reports should directly create PASS evidence | Reject as unsafe until evidence policy/provenance is added |
| Requested agent labels differ from actual models | Retain |
| Intermediate artifacts lack lineage | Retain |
| Current control typecheck fails | Correct to historical failure/current pass with untracked integration caveat |
| Current suite is 52/52 | Correct to fresh 55/55 |

---

## 5. Smallest prioritized implementation slices

### P0 — Acceptance integrity and control safety

1. **Current-acceptance worker enforcement**  
   Files: `src/store.ts`, `test/store.test.ts`  
   Gate: stale worker/report cannot cross acceptance revisions.

2. **PAUSE/STOP revision-drift fix**  
   Files: `src/store.ts`, `test/store.test.ts`, `test/server.test.ts`  
   Gate: non-content RUNNING update preserves control of the active target.

3. **Evidence closure policy**  
   Files: `src/types.ts`, `src/store.ts`, `src/index.ts`, tests  
   Gate: current required FAIL blocks DONE; child claims alone never pass.

### P1 — Durable accounting and identity

4. Explicit observe-only/hard budget schema plus reservation ledger.  
5. Requested/actual model attempt records.  
6. Child run graph beneath workflow binding.  
7. Artifact lineage manifest.

### P2 — Ownership and migration hardening

8. Single canonical server-owner registry without capability tokens.  
9. Cross-process event reconciliation.  
10. Transactional numbered SQLite migrations.

### Rollback requirements

- Use additive nullable schema changes.
- Keep legacy `model` and existing report rows readable.
- Feature-flag new closure policy until legacy tasks are classified.
- Registry can be disabled without deleting the DB.
- Never drop old columns in the first release.
- Back up and `PRAGMA quick_check` before migration; verify restored DB with the previous package.

---

## 6. Validation commands and acceptance gates

Executed read-only or temporary-project-local checks:

- `cd mazzy-control-panel && npm run typecheck` — PASS.
- `cd mazzy-control-panel && npm test` — PASS, 55 tests.
- Focused `storage-policy.test.ts` — PASS, 3 tests.
- Read-only SQLite `quick_check` and target-task counts — PASS.
- Isolated stale-acceptance reproduction — defect reproduced.
- Isolated generic-FAIL closure reproduction — defect reproduced.
- Isolated RUNNING control-drift reproduction — defect reproduced.
- Listener/process inspection — two same-project owners confirmed; no capability URL inspected or recorded.
- Git staged-file count — zero.

Recommended mandatory gate after implementation:

```bash
cd mazzy-control-panel
npm run typecheck
npm test
git diff --check
git diff --cached --name-only
```

Additionally require focused multi-process and legacy-migration tests; the current suite does not cover those contracts.

---

## 7. Residual risks and gaps

- No same-instant authenticated dashboard-vs-DB comparison was performed, so the historical numerical discrepancy remains observational.
- Provider checkpoint behavior is outside `@mazzy/control`; pi-subagents source was not audited in this lane.
- No destructive VPN operation or network mutation was performed.
- No concurrent legacy migration race was executed.
- Current storage-policy source/tests/resources are untracked, preventing commit-level lineage for the typecheck fix.
- The current DB is a live shared control plane and continued changing during the investigation; counts are point-in-time facts.
- SQLite and process inspection confirm local pilot behavior only, not multi-user or distributed safety.

```acceptance-report
{
  "criteriaSatisfied": [
    {
      "id": "criterion-1",
      "status": "satisfied",
      "evidence": "Concrete severity-ranked findings identify exact @mazzy/control source and test seams, including three isolated reproductions and report corrections."
    }
  ],
  "changedFiles": [],
  "testsAddedOrUpdated": [],
  "commandsRun": [
    {
      "command": "cd mazzy-control-panel && npm run typecheck",
      "result": "passed",
      "summary": "Fresh typecheck passed; the stored three-error log predates the current source."
    },
    {
      "command": "cd mazzy-control-panel && npm test",
      "result": "passed",
      "summary": "Fresh full suite passed 55/55."
    },
    {
      "command": "focused storage-policy test",
      "result": "passed",
      "summary": "Three storage policy tests passed."
    },
    {
      "command": "read-only SQLite quick_check and control-state queries",
      "result": "passed",
      "summary": "Both inspected databases passed quick_check; the target task remains REVIEW with one binding, one report, and zero evidence."
    },
    {
      "command": "isolated acceptance-revision, evidence-gate, and control-revision probes",
      "result": "passed",
      "summary": "Reproduced stale-worker closure, generic FAIL being ignored, and STOP rejection after RUNNING revision drift."
    }
  ],
  "validationOutput": [
    "Current control suite: 55 passed, 0 failed.",
    "Current control typecheck: passed.",
    "Stored earlier typecheck artifact: three errors caused by unknown maxAgeSeconds narrowing.",
    "Target task: REVIEW; one completed worker binding; zero evidence; one review report.",
    "Current PI control DB quick_check: ok."
  ],
  "residualRisks": [
    "Historical dashboard and DB counts were not captured atomically.",
    "Provider checkpoint implementation belongs partly to pi-subagents and was not inspected here.",
    "Concurrent legacy migration race remains unexecuted.",
    "Storage-policy implementation files are untracked, so commit-level integration is not attested."
  ],
  "noStagedFiles": true,
  "diffSummary": "Read-only architecture audit; no repository diff created.",
  "reviewFindings": [
    "critical: mazzy-control-panel/src/store.ts:64,66,71,120 - REVIEW content edits can reuse a stale completed worker/report and still reach DONE.",
    "high: mazzy-control-panel/src/store.ts:73-74,125 - current generic FAIL evidence does not block DONE.",
    "high: mazzy-control-panel/src/store.ts:64,124 - non-content RUNNING updates disable PAUSE/STOP for the active binding.",
    "high: mazzy-control-panel/src/routing.ts:17,126-154 - configured budgets are not enforced or returned by routing.",
    "high: mazzy-control-panel/src/server.ts:101-118 - process-local SSE listeners miss writes committed by another server process.",
    "medium: mazzy-control-panel/src/index.ts:331-338 - dashboard ownership and port discovery are session-local.",
    "medium: mazzy-control-panel/src/store.ts migration seam - schema migration is not one explicit numbered transaction.",
    "medium: review reports and output artifacts have no durable lineage or superseded relation."
  ],
  "manualNotes": "The writer should replace the inherited 'current 52-pass plus typecheck failure' statement with a chronology: stored 52-pass/three-error evidence, later source change, and fresh 55-pass/typecheck-pass state."
}
```

TASK_COMMENT
role: ops-planner-sol  
run_id: none  
accomplishment: Audited routing, SQLite/FSM, evidence, bindings, dashboard ownership, reports, logs, and current typecheck state; reproduced three control-plane defects.  
checks/verdict: Current tests and typecheck pass; acceptance and control-safety blockers remain in store logic.  
blockers/next step: Correct stale-acceptance reuse first, then evidence closure and RUNNING control drift; update the report’s historical/current test chronology.