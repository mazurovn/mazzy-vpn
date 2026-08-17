# Журнал расследования и проблем Mazzy Control Panel — 2026-08-16

**Режим:** доказательная документация; production/source/DB не менялись. Все команды ниже выполнялись только для чтения либо анализировали уже сохранённые артефакты. «Деструктивно» означает потенциальную VPN/данную мутацию; во всех строках — `нет`.

## Хронология

| Дата/UTC | Команда/действие | Результат | Артефакт | Деструктивно |
|---|---|---|---|---|
| 2026-08-16 17:00–17:55 | VPN task lifecycle и final validation | задача дошла до REVIEW; 19 plugin tests и typecheck сообщены PASS; DONE не материализован | workflow reports, SQLite task/review | нет |
| 2026-08-16, сохранённый запуск | `@mazzy/control` test | 52/52 PASS, 6,65 с, RSS 133144 KiB | `control-panel-tests.log` | нет |
| 2026-08-16, сохранённый запуск | `@mazzy/control` typecheck | exit 2, 7,65 с, 3 TS ошибки storage-policy | `control-panel-typecheck.log` | нет |
| 2026-08-16, сохранённый запуск | VPN plugin test/typecheck | 19/19 PASS за 1,30 с; typecheck PASS за 3,41 с | plugin logs | нет |
| 2026-08-16, сохранённый запуск | MAZZY_VPN tests | 105/105 PASS за 279,15 с | `mazzy-vpn-tests.log` | нет |
| 2026-08-16 | parse benchmark JSON/script | serial localhost synthetic; summary инварианты порядка соблюдены | `control-panel-benchmark.json`, script | нет |
| 2026-08-16 | read-only source/SQLite/process inspection | выявлены acceptance/evidence/control seams, два server process и process-local SSE | source seams, DB `quick_check` | нет |
| 2026-08-16, Stage A | `npm run typecheck`, `npm test` в текущем worktree | PASS и 55/55 после появления untracked storage-policy файлов; не подменяет сохранённый гейт | `stage1-architecture-a.md` | нет |
| 2026-08-16 | isolated store probes | воспроизведены stale acceptance closure, ignored generic FAIL, RUNNING control drift | architecture handoff | нет |
| 2026-08-16, parent validation | `npm test && npm run typecheck` для `@mazzy/control` | 55/55 PASS за 6,06 с; typecheck PASS; dirty/untracked source, поэтому не release provenance | `parent-control-validation.log` | нет |
| 2026-08-16, parent validation | `npm test && npm run typecheck` для VPN plugin | 19/19 PASS; typecheck PASS | `parent-vpn-plugin-validation.log` | нет |
| 2026-08-16, parent validation | проверка полного `mazzy-vpn-tests.log` без повтора 279-секундного suite | TAP завершён `1..105`, 105/105 PASS | `mazzy-vpn-tests.log` | нет |

## Реестр проблем

### MZ-CP-01 — narrowing `unknown` в storage policy
**Статус:** исторический FAIL подтверждён сохранённым гейтом; текущий parent-rerun PASS, но исправление остаётся untracked и не имеет интеграционной provenance. **Severity:** P0 до привязки к принятой source revision.  
**Evidence/repro:** `tsc -p tsconfig.json` завершился exit 2: два TS18046 и один TS2322 для `rule.maxAgeSeconds` в `src/storage-policy.ts:34–35`; одновременно runtime suite 52 PASS.  
**Минимальное безопасное исправление:** после `isRecord(rule)` один раз извлечь `maxAgeSeconds`, проверить `typeof === "number"`, и лишь затем сравнивать/присваивать. Не применять assertion/cast, не менять policy semantics.  
**Source seam/тест:** `src/storage-policy.ts`; `test/storage-policy.test.ts`: absent/string/NaN/negative/valid number плюс `npm run typecheck`.  
**Acceptance:** `npm run typecheck && npm test`, с привязанными commit/diff digest и отсутствием новых ошибок.

### MZ-CP-02 — budget scope и enforcement неоднозначны
**Статус:** подтверждено. **Severity:** P0 expectation mismatch.  
**Evidence/repro:** `routing.ts:17,50–57,93,126–154` валидирует/загружает `defaultUsd`, `highRiskUsd`, `maxChildrenPerWave`, но `RouteResult` не несёт budget; нет task ledger, reservation либо dispatch gate. Наблюдаемые `$5.719125` при `$2` доказывают только отсутствие hard task cap.  
**Минимальное безопасное исправление:** schema явно содержит `scope`, `enforcement`, currency и failed-cost rule; `ops_route` возвращает декларацию. Parent/runtime атомарно attests prelaunch reservation и фактическую стоимость; dashboard хранит/показывает это, но не планирует child и не биллингует provider.  
**Тест:** task с hard остатком ниже reservation не получает launch без durable override; failed run списывается; resume использует остаток.  
**Acceptance:** parent/runtime integration test + durable ledger inspection; target: 0 запусков сверх hard budget без override.

### MZ-CP-03 — late-provider error не сохраняет полезный checkpoint
**Статус:** отсутствие подтверждено; распределение ownership — архитектурная рекомендация. **Severity:** P1.  
**Evidence/repro:** два 403 возникли после 45 turns/48 tools суммарно, но финальные artifacts не содержат структурированных findings/checkpoint.  
**Минимальное безопасное исправление:** runtime пишет ограниченный checkpoint (phase, digest, относительный artifact ID, remaining work); классифицирует non-retryable 403 и размыкает same-provider circuit. Control panel принимает только parent-attested reference/attempt/fallback reason, не transcript/credential.  
**Тест/acceptance:** simulated late 403 открывает latest checkpoint, не повторяет provider, fallback ссылается на digest; target: 100% late terminal errors имеют структурированный checkpoint.

### MZ-CP-04 — stale worker может закрыть новую acceptance revision
**Статус:** воспроизведено изолированно. **Severity:** P0 critical.  
**Evidence/repro:** worker завершает acceptance=1; content edit в REVIEW оставляет REVIEW с acceptance=2; старый completed worker импортирует report и reviewer PASS приводит к DONE.  
**Seams:** `src/store.ts:64` updateTask, `:66` assignRun, `:71` importReviewReport, `:120` latestCompletedWorker; `test/store.test.ts`.  
**Минимальное безопасное исправление:** title/description edit в REVIEW → READY; import, reviewer assignment и reviewer evidence требуют равенство `acceptanceRevision` и `acceptanceDigest`; latest completed worker фильтруется по current acceptance.  
**Acceptance:** store test отвергает stale import/assignment/DONE до нового completion; target: 0 cross-acceptance closure.

### MZ-CP-05 — generic FAIL evidence не блокирует task closure
**Статус:** воспроизведено изолированно. **Severity:** P0 high.  
**Evidence/repro:** после reviewer PASS добавлен current `typecheck: FAIL`; `updateTask(...DONE...)` всё равно succeeds.  
**Seams:** `src/store.ts:73–74` evidence, `:125` `requireCurrentReviewerPass`; `src/index.ts` exposes reviewer-only path.  
**Минимальное безопасное исправление:** до materialization определить evidence kinds: parent-observed deterministic, independent reviewer, child claim, informational. Closure блокируется current required FAIL/UNCERTAIN и required-kind omission; child claim не авторитетен.  
**Acceptance:** current deterministic FAIL blocks DONE, stale FAIL не блокирует следующую revision, child report не создаёт PASS. Target: 0 closure при required FAIL.

### MZ-CP-06 — RUNNING revision drift отключает PAUSE/STOP
**Статус:** воспроизведено изолированно. **Severity:** P0 high.  
**Evidence/repro:** priority/no-op update увеличивает task revision, binding остаётся assignment revision; STOP/PAUSE отклоняются как не current.  
**Seams:** `src/store.ts:64,124`, `src/server.ts:140–153`, `test/store.test.ts`, `test/server.test.ts`.  
**Минимальное безопасное исправление:** control applicability проверяет current expected task revision, active state, exact run ID и task identity, но не сравнивает mutable lifecycle revision с immutable assignment epoch; альтернативно атомарно синхронизировать lifecycle revision без переписывания epoch.  
**Acceptance:** priority/no-op RUNNING update сохраняет STOP/PAUSE; wrong/superseded run отклоняется; pre-update request получает optimistic conflict. Target: 100% active control после non-content edit.

### MZ-CP-07 — closure требует явного решения и неполно задана policy
**Статус:** подтверждено. **Severity:** P1.  
**Evidence/repro:** DONE отсутствует из UI transitions; current reviewer-bound PASS необходим, но не переводит автоматически; VPN task REVIEW имеет один report и 0 evidence.  
**Seams:** `src/store.ts:125`, task transitions/UI read model, `src/index.ts`.  
**Минимальное безопасное исправление:** parent-only atomic `decide-completion`: вычисляет required evidence, пишет reasons/decision, закрывает только при policy satisfied. Не делать каждый reviewer PASS auto-DONE.  
**Acceptance:** decision сообщает missing evidence; valid policy closes exactly once. Target: closure latency ≤2 min после полного gate (целевой KPI, не текущий SLA).

### MZ-CP-08 — run binding не отражает child attempts
**Статус:** подтверждено. **Severity:** P1.  
**Evidence/repro:** для 12 измеренных child runs у VPN task один workflow binding; artifacts вне task graph.  
**Минимальное безопасное исправление:** immutable child-attempt records под binding: workflow/child ID, role, requested/actual model, provider, attempt index, lifecycle, checkpoint/artifact references. Parent — единственный attestor; child не пишет control state.  
**Acceptance:** каждая observed attempt отображается один раз с parent binding; target: ≥95% visible child traceability.

### MZ-CP-09 — requested и actual model смешаны
**Статус:** подтверждено. **Severity:** P1.  
**Evidence/repro:** route возвращает одну model; `run_bindings.model` mutable и перезаписывается `updateRunActivity`; role labels `opus5`/`sonnet` не совпали с фактическим GPT-5.4.  
**Seams:** `src/routing.ts`, `src/store.ts` binding/activity.  
**Минимальное безопасное исправление:** attempt/nullable additive fields `requested_model`, `actual_model`, `provider`, `fallback_reason`, `attempt_index`; legacy `model` маркировать legacy, не backfill без evidence.  
**Acceptance:** fallback UI/API одновременно показывает request и fact; target: 100% новых attempts имеют оба идентификатора.

### MZ-CP-10 — dashboard snapshot/DB discrepancy и межпроцессный SSE
**Статус:** SSE defect подтверждён; историческая разница count причинно не доказана. **Severity:** P1.  
**Evidence/repro:** `src/server.ts:130` вызывает live `store.snapshot()`, значит server cache не причина сама по себе. Два PI-owned process работают с одной DB; `OpsStore.subscribeEvents` process-local, browser прекращает fallback polling при healthy SSE. Write process B не будит SSE client process A.  
**Минимальное безопасное исправление:** один canonical owner на DB либо cursor-based reset/reconciliation; snapshot отвечает revision/event ID. Проверка count должна быть атомарной по revision.  
**Acceptance:** cross-process mutation доставляется либо вызывает reset+snapshot; target: dashboard count совпадает с DB при одинаковой revision.

### MZ-CP-11 — dead/canonical port discovery session-local
**Статус:** подтверждено. **Severity:** P2.  
**Evidence/repro:** default 4319 может не слушать при владельцах на иных портах; status знает лишь server object текущей session. Короткоживущий запуск не доказывает, что canonical URL был dead в тот же миг.  
**Seams:** `src/server.ts:58–70`, `src/index.ts:331–338`, `src/project.ts:9–17`.  
**Минимальное безопасное исправление:** token-free local owner registry: project digest, endpoint, PID+start nonce, session ID, heartbeat; non-owner показывает owner, не стартует второй. Никогда не хранить capability token/абсолютный sensitive path.  
**Acceptance:** два процесса не становятся canonical; stale PID-reuse safe recovery; registry redaction test. Target: 0 advertised non-listening canonical endpoints.

### MZ-CP-12 — artifact lineage отсутствует
**Статус:** подтверждено. **Severity:** P1.  
**Evidence/repro:** stage reports 14/18/19 tests выглядят независимо final; DB не хранит relative ref, SHA-256, phase/current/supersedes; legacy/new storage locations расходятся.  
**Минимальное безопасное исправление:** bounded manifest/table: task, acceptance, binding/attempt, kind/phase, relative ID, SHA-256, current/superseded/failed, predecessor. Payload и absolute path не сохранять.  
**Acceptance:** final view выбирает only-current; superseded report не входит gate. Target: 100% gate artifacts имеют digest+phase.

### MZ-CP-13 — benchmark не воспроизводим и не измеряет нагрузку
**Статус:** подтверждено. **Severity:** P2.  
**Evidence/repro:** JSON не содержит timestamp, script/source/diff hash, dirty state, Node/SQLite/OS/CPU/RAM/FS/pragma, warmup/repetitions/raw samples. Script serially awaits, создаёт 500 tasks, проверяет smoke только status/id.  
**Минимальное безопасное исправление:** nearest-rank `ceil(n×p)-1` с unit cases; metadata envelope + raw/histogram digest; 10 fresh-process repetitions, fixed seed/cardinality, separate mixed/concurrent/SSE workloads.  
**Acceptance:** artifact позволяет идентифицировать code/environment и пересчитать percentile. Target: воспроизводимый baseline, не universal SLA.

### MZ-CP-14 — Stage-B stale-revision lost-update race
**Статус:** подтверждён static inspection; **не воспроизведён process barrier**, поэтому severity P0 и implementation требуют теста.  
**Evidence/repro:** `src/store.ts:64` выполняет `requireRevision` до `BEGIN IMMEDIATE` (`:110–111`); последующий `UPDATE ... WHERE revision=?` не проверяет `changes`. Два process могут прочитать N; первый записывает N+1, второй обновляет 0 rows, но может emit/commit и вернуть current task. Existing tests — sequential.  
**Минимальное безопасное исправление:** lookup/transition/CAS внутри immediate transaction; требовать ровно одну изменённую строку, иначе conflict до event.  
**Acceptance:** two-store barrier: ровно один PATCH с revision N success, второй conflict, один event. Target: silent lost updates = 0.

## Backlog

| Приоритет | Работа/owner role | Effort | Risk/dependencies | Acceptance command/evidence | Target metric |
|---|---|---:|---|---|---|
| P0 | CP-01 type narrowing — control maintainer | S | untracked provenance | `npm run typecheck && npm test`; bound logs/digest | 0 TS errors |
| P0 | CP-04 current-acceptance enforcement — store maintainer | M | CP-07 policy | focused store test + full suite | 0 stale closure |
| P0 | CP-05 evidence kinds/closure — policy+store maintainer | M | classify legacy tasks | deterministic FAIL test + suite | 0 required-FAIL closure |
| P0 | CP-06 control drift — store/server maintainer | S | preserve immutable epoch | focused server/store tests | active control retained |
| P0 | CP-14 transactional CAS — store maintainer | M | two-store barrier harness | barrier regression + suite | 0 silent lost update |
| P1 | CP-02 budget attestation — parent/runtime + control maintainer | L | provider cost contract | reservation/override integration evidence | 0 hard-cap bypass |
| P1 | CP-03 checkpoint salvage — runtime owner | M | secure artifact contract | late-403 simulation | checkpoint coverage 100% |
| P1 | CP-07 decision closure — parent/control maintainer | M | CP-05 | decision tests | ≤2 min target closure |
| P1 | CP-08/09 attempt identity — schema/control maintainer | M | additive migration | API/UI attempt fixture | ≥95% traceability |
| P1 | CP-10 reconciliation — server/control maintainer | M | owner policy | two-process SSE/reset test | count parity at same rev |
| P1 | CP-12 lineage — artifact/control maintainer | M | relative storage contract | manifest/current selection test | 100% gate digests |
| P2 | CP-11 owner registry — server maintainer | M | PID reuse/secret redaction | multi-process registry test | 0 dead advertised URL |
| P2 | CP-13 reproducible benchmark — perf owner | M | controlled host | repeatable metadata/raw artifacts | baseline only, no SLA |

## Нарушение директивы «дополнить, не сокращать»

### MZ-CP-15 — writer заменил исходный отчёт вместо дополнения
**Статус:** подтверждено журналом инструментов. **Severity:** P0 для document-preservation workflow.  
**Требование:** sole-writer получил прямую инструкцию `Preserve useful existing material and facts`. Он прочитал полный исходный файл размером 48 514 байт и 1134 строки, после чего вызвал полную операцию `write` с содержимым около 7 777 символов. Итоговый основной документ сократился до 87 строк и 11 432 байт. Формулировка handoff «переписано executive summary» также показывает смену семантики с дополнения на замену.  
**Почему это нарушение:** инструкция дошла до writer без искажения, поэтому причиной не является потеря prompt. Причина — отсутствие механического preservation gate и неверный выбор полной перезаписи вместо append/targeted edit.  
**Минимальное безопасное исправление процесса:** перед write вычислять baseline hash/lines/bytes/headings; для задачи `extend/preserve` разрешать только append или targeted edits. Полная перезапись требует отдельного explicit approval.  
**Acceptance:** после writer `new_lines >= baseline_lines`, все обязательные baseline headings присутствуют, baseline body доступен в V2 либо сохранён byte-for-byte; при нарушении run получает FAIL, а исходный файл автоматически восстанавливается.

### MZ-CP-16 — acceptance contract не проверял сохранность содержания
**Статус:** подтверждено metadata writer-run. **Severity:** P0 process/control.  
**Evidence:** runtime checks проверили только changed-files, commands, residual risks, staged state и diff summary. Ни один criterion не требовал минимального размера, сохранения headings, сравнения semantic coverage либо отсутствия крупного удаления. Поэтому run был помечен `review-required`, но локальные criteria одновременно считались satisfied.  
**Минимальное безопасное исправление:** добавить evidence kinds `baseline-digest`, `retained-sections`, `deletion-ratio`, `document-structure`; для документационного расширения установить `maxDeletionRatio=0` или явный allowlist заменяемых секций.  
**Acceptance:** fixture с 1000-строчным отчётом и 100-строчной заменой блокируется до review; append с сохранением baseline проходит.

### MZ-CP-17 — post-write reviewer отсутствовал, а parent не остановил регрессию до handoff
**Статус:** подтверждено. **Severity:** P1 orchestration.  
**Evidence:** writer metadata честно имеет `review-required`; Mazzy task оставлен в REVIEW; независимый post-write reviewer PASS не выполнялся. Parent validation проверила Markdown, числа, staged files и sensitive patterns, но не сравнила 87 строк с baseline 1134 до представления результата как завершённого.  
**Минимальное безопасное исправление:** обязательный parent diff-stat gate сразу после writer, затем независимый reviewer с вопросом «сохранены ли все baseline sections и выполнен ли именно requested operation». При удалении более заданного порога parent не публикует результат и восстанавливает baseline.  
**Acceptance:** destructive document shrink обнаруживается автоматически; task остаётся BLOCKED/REVIEW, canonical файл не заменяется.

### MZ-CP-18 — исходный single-workflow contract был нарушен fallback-запуском
**Статус:** подтверждено parent final report. **Severity:** P1 orchestration reliability.  
**Evidence:** исходно требовался ровно один workflow: две read-only lanes, затем sole writer. Runtime usage-budget accounting остановил Stage 2; parent запустил writer отдельным fallback run. Fallback сохранил single-writer безопасность, но нарушил атомарность и единый workflow graph.  
**Минимальное безопасное исправление:** резервировать budget на обязательную writer stage до fanout; при недостатке бюджета не запускать read-only wave либо запрашивать override. Resume должен продолжать тот же mission/workflow с сохранёнными outputs.  
**Acceptance:** scripted workflow либо завершает все declared mandatory stages, либо останавливается до fanout с явным budget error; отдельный fallback помечается deviation и не считается исходным workflow.

## Корневая причина сокращения отчёта

| Уровень | Причина | Подтверждение | Контрмера |
|---|---|---|---|
| Prompt | Директива была корректной | В writer task есть `Preserve useful existing material` | Не считать это prompt-loss |
| Agent action | Выбрана полная `write`, а не append/edit | Tool log: read полного файла → write 7 777 символов | Запрет full rewrite для preserve-mode |
| Acceptance | Не было retention criteria | Metadata содержит только generic evidence | Baseline/heading/deletion gates |
| Review | Независимый post-write review не выполнен | `review-required`, task REVIEW | Reviewer до публикации результата |
| Parent validation | Проверялись формат и числа, не сохранность | PASS для Markdown/numeric checks при 87 строках | Обязательный diff-stat/baseline check |
| Runtime | Stage 2 выпала из исходного workflow | Usage-budget accounting, отдельный fallback | Budget reservation и resumable workflow |
| Git/recovery | Документ был untracked | Git не мог восстановить предыдущую версию | Project-local pre-write snapshot |

## Непринятые риски и границы

Benchmark и 279-секундный MAZZY_VPN suite не повторялись; не выполнялись real VPN/systemd/provider run, destructive reconnect, конкурентная legacy-migration гонка или атомарное historical dashboard-vs-DB сравнение. Current storage-policy parent-rerun PASS остаётся непригоден как release доказательство, пока untracked files не получат provenance. Ни один performance number этого журнала не является SLA.
