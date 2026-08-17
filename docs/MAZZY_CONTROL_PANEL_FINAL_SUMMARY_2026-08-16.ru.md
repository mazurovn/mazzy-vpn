# Итоговая сводка

## Общая оценка

| Область | Оценка | Итог |
|---|---:|---|
| VPN Recovery | 9/10 | Работает безопаснее, тесты проходят |
| Качество работы reviewers | 9/10 | Найдены реальные архитектурные дефекты |
| Mazzy Control Panel | 6.5/10 | Хороший pilot, но есть критические FSM/evidence-проблемы |
| Оркестрация агентов | 6/10 | Результат качественный, но процесс дорогой и нестабильный |
| Контроль бюджета | 3/10 | Бюджет описан, но практически не ограничивает расходы |
| Трассируемость runs | 5/10 | Child runs существуют, но не полностью отражены в task graph |
| Dashboard/Web UX | 5/10 | Есть проблемы ownership, портов и межпроцессного обновления |
| Сохранность документов | 4/10 | Writer смог заменить отчёт вопреки директиве |
| **Общая оценка системы** | **7/10** | Полезная система, но P0-проблемы нужно устранить до production |

---

# 1. Что получилось хорошо

## VPN Recovery

| Функция | Состояние |
|---|---|
| Режимы `fast/balanced/deep` | Реализованы |
| `/vpn-recovery doctor` | Реализован, недеструктивный |
| Проверка реального транспорта | Реализована |
| Provider-only suppression | Реализован |
| Проверка маршрута через `tun0` | Реализована |
| Physical carrier precheck | Реализован |
| Helper readiness | Реализована |
| IPv6 degradation | Отображается как `WARN` |
| Lock/deadline coverage | Исправлена |
| Reconciliation lease | Исправлена |
| Cooldown/rate limit/circuit breaker | Сохранены |
| Тесты | **19/19 PASS** |
| TypeScript | PASS |

Главный результат: один сбой OpenAI или CLI status больше не должен сам по себе вызывать destructive VPN reconnect.

## Работа reviewers

Reviewers нашли ошибки, которые не поймала первая реализация:

1. служебные probes ошибочно участвовали в transport quorum;
2. `fast` мог вернуть ложный PASS без рабочего интернета;
3. IPv6 degradation не влияла на doctor verdict;
4. recovery lease не покрывал долгий verify;
5. reconciliation имел отдельный короткий lock;
6. в Mazzy обнаружены дополнительные acceptance/FSM-проблемы.

Это подтверждает пользу независимого fresh-context review.

---

# 2. Текущее состояние тестов

| Компонент | Тесты | Typecheck | Ограничение |
|---|---:|---:|---|
| `pi-vpn-recovery` | 19/19 PASS | PASS | Mock/локальные проверки |
| MAZZY_VPN | 105/105 PASS | Не основной gate | Shell/synthetic suite |
| Mazzy Control Panel — сохранённый запуск | 52/52 PASS | FAIL, 3 ошибки | Исторический снимок |
| Mazzy Control Panel — текущий worktree | 55/55 PASS | PASS | Изменения untracked, нет release provenance |

Важно: зелёные 55 тестов не означают отсутствие архитектурных дефектов. Несколько проблем были воспроизведены отдельными harness-проверками и пока не исправлены в source.

---

# 3. Критические проблемы Mazzy Control Panel

## P0 — исправить в первую очередь

| ID | Проблема | Риск | Что нужно сделать |
|---|---|---|---|
| MZ-CP-04 | Старый worker может закрыть новую acceptance revision | Critical | Привязать report/reviewer/evidence к acceptance revision и digest |
| MZ-CP-05 | Current deterministic `FAIL` не блокирует `DONE` | High | Ввести типы evidence и closure policy |
| MZ-CP-06 | Изменение RUNNING-task отключает PAUSE/STOP | High | Разделить assignment epoch и текущую task revision |
| MZ-CP-14 | Возможна cross-process CAS-гонка | High | Перенести revision check внутрь transaction и проверять affected rows |
| MZ-CP-01 | Историческая ошибка narrowing `unknown` | High | Зафиксировать уже найденное исправление в tracked source и прогнать gate |
| MZ-CP-15 | Writer заменил отчёт вместо дополнения | High | Добавить document-preservation gate |
| MZ-CP-16 | Acceptance не проверял сохранность отчёта | High | Проверять baseline hash, headings и deletion ratio |

## Самая опасная проблема: stale acceptance

Сценарий:

```text
Worker завершил acceptance revision 1
        ↓
Описание задачи изменили → acceptance revision 2
        ↓
Старый report revision 1 повторно принимается
        ↓
Reviewer ставит PASS
        ↓
Задача может перейти в DONE
```

### Исправление

Для report, reviewer assignment и reviewer evidence необходимо проверять:

```text
worker.acceptanceRevision == task.acceptanceRevision
worker.acceptanceDigest == task.acceptanceDigest
```

Если title или description меняются в `REVIEW`, задача должна возвращаться в `READY`.

---

# 4. Проблемы evidence и закрытия задач

Сейчас comments, reports и evidence существуют как разные сущности, но политика закрытия неполна.

| Сигнал | Сейчас | Как должно быть |
|---|---|---|
| Child comment | Не evidence | Оставить как есть |
| Child acceptance report | Report, но не authoritative PASS | Требует parent verification |
| Reviewer PASS | Может разрешить DONE | Должен быть current и independent |
| Generic typecheck FAIL | Может не блокировать DONE | Обязан блокировать |
| Missing required evidence | Не всегда явно объясняется | Должен показываться exact blocker |
| Полный набор PASS | Не закрывает автоматически | Parent atomic completion decision |

## Рекомендуемые типы evidence

```text
parent-observed-deterministic
independent-review
child-reported-claim
informational-artifact
```

Только первые два должны влиять на `DONE`.

## Рекомендуемый closure gate

```text
current acceptance
AND completion report current
AND required deterministic checks PASS
AND no required FAIL/UNCERTAIN
AND independent reviewer PASS
AND parent completion decision
→ DONE
```

---

# 5. Проблемы управления PAUSE/STOP

Текущая логика смешивает:

- revision, при которой worker был назначен;
- текущую mutable task revision.

Из-за этого изменение priority или даже no-op update может увеличить revision и сделать активный run «неподходящим» для STOP.

## Исправление

Для control command проверять:

- task ID;
- current expected task revision;
- exact active run ID;
- binding state `active`;
- owner/session authority.

Не нужно требовать, чтобы immutable assignment revision всегда совпадала с текущей mutable revision.

---

# 6. Cross-process CAS race

Сейчас revision может проверяться до `BEGIN IMMEDIATE`.

Возможный сценарий:

```text
Process A читает revision 5
Process B читает revision 5
Process A записывает revision 6
Process B делает UPDATE WHERE revision=5
UPDATE затрагивает 0 строк
Process B не проверяет affected rows и может сообщить успех
```

## Исправление

```text
BEGIN IMMEDIATE
    read current revision
    validate transition
    UPDATE ... WHERE revision = expected
    require changes == 1
    record event
COMMIT
```

Нужен двухпроцессный barrier test: один PATCH проходит, второй получает conflict, создаётся только одно событие.

---

# 7. Проблемы бюджета orchestration

## Фактические метрики

| Метрика | Значение |
|---|---:|
| Child runs | 12 |
| Успешные | 10 |
| Неуспешные | 2 |
| Суммарное agent-время | 39,96 минуты |
| Input + output tokens | 971 585 |
| Cache read | 5 379 242 |
| Child cost | `$5.719125` |
| Failed cost | `$1.655` |
| Доля failed cost | 28,94% |
| Доля writer-cost | 12,82% |
| Доля planner/reviewer cost | 87,18% |

Configured `highRiskUsd: 2` не являлся реальным hard task budget.

## Почему дорого

1. четыре успешных planner runs;
2. четыре успешных reviewer runs;
3. два writer runs;
4. поздние Anthropic 403 после десятков turns;
5. повторная передача большого контекста;
6. resume и fallback повторяли часть работы;
7. обязательная writer-stage не получила заранее зарезервированный бюджет.

## Что улучшить

```json
{
  "scope": "task",
  "enforcement": "hard",
  "maxCostUsd": 3.0,
  "maxFailedCostUsd": 0.25,
  "maxRuns": 6,
  "maxAgentMinutes": 25
}
```

Перед запуском каждой стадии резервировать бюджет. Обязательный writer должен получить резерв до planner/reviewer fanout.

---

# 8. Проблема late provider failure

Anthropic runs завершились 403 не сразу:

- 45 turns;
- 48 tool calls;
- более 10 agent-минут;
- `$1.655` расходов.

Финальные checkpoints с полезными findings отсутствовали.

## Решение

После каждого значимого этапа сохранять:

```json
{
  "phase": "findings-complete",
  "artifactDigest": "...",
  "completedWork": [],
  "remainingWork": [],
  "provider": "...",
  "attempt": 1
}
```

При late 403:

1. сохранить checkpoint;
2. открыть provider circuit breaker;
3. не повторять тот же provider;
4. запустить fallback с checkpoint, а не с полным контекстом;
5. записать requested и actual model.

---

# 9. Run bindings и модели

Для 12 child runs в VPN task отражён только один основной workflow binding.

| Проблема | Последствие |
|---|---|
| Child attempts не видны | Неполный audit trail |
| Agent называется `sonnet`, фактически GPT | Неверная интерпретация модели |
| Одно поле `model` перезаписывается | Нельзя восстановить fallback history |
| Fix-writer связан неоднозначно | Нельзя точно определить автора изменения |

## Нужная модель данных

```text
workflow_binding
  ├── child_attempt 1
  │     requested_model
  │     actual_model
  │     provider
  │     fallback_reason
  │     lifecycle
  │     artifact_digest
  └── child_attempt 2
```

---

# 10. Dashboard и web

## Подтверждённые проблемы

| Проблема | Статус |
|---|---|
| Несколько server owners одного проекта | Подтверждено |
| Port discovery session-local | Подтверждено |
| Default URL может не соответствовать живому owner | Подтверждено как design gap |
| SSE notifications process-local | Подтверждено |
| Dashboard имеет server-side cached snapshot | Опровергнуто |
| Исторические counts 53 и 62 доказывают stale cache | Не доказано |

`/api/snapshot` читает SQLite напрямую. Основной дефект другой: процесс A не получает SSE notification о записи процесса B.

## Исправление

Нужен token-free owner registry:

```text
project digest
endpoint
PID + process-start nonce
owner session
heartbeat
DB revision / last event ID
```

Capability token в registry сохранять нельзя.

Dashboard при reconnect должен сравнивать event cursor и выполнять snapshot reconciliation.

---

# 11. Artifact lineage

Существуют отчёты с результатами:

- 14 тестов;
- 18 тестов;
- 19 тестов.

Но Control Panel не знает, какой из них final, а какой superseded.

## Нужные поля

```text
task
acceptance revision
run/attempt
phase
artifact kind
relative path
SHA-256
state: current | superseded | failed
supersedes
```

В acceptance gate должны участвовать только `current` artifacts.

---

# 12. Почему был уменьшен отчёт

Это отдельная orchestration-регрессия.

| Уровень | Что произошло |
|---|---|
| Директива | Writer получил требование сохранить исходный материал |
| Чтение | Writer прочитал полный файл на 1134 строки |
| Запись | Использовал полную `write` примерно на 7777 символов |
| Результат | Основной отчёт уменьшился до 87 строк |
| Acceptance | Не проверял размер, headings и deletion ratio |
| Review | Независимый post-write reviewer отсутствовал |
| Parent | Проверил Markdown и числа, но не сравнил размер |
| Workflow | Writer выполнялся отдельным fallback после budget stop |
| Git | Файл был untracked, поэтому Git не мог восстановить его |

## Что добавить в orchestration

Для задач `extend`, `supplement`, `preserve`:

```text
baseline hash
baseline line count
baseline byte count
required headings
max deletion ratio
append-only/targeted-edit mode
post-write independent reviewer
```

Если было 1134 строки, а стало 87, acceptance должен автоматически завершаться FAIL и восстанавливать baseline.

---

# 13. Benchmark

## Результаты

| Операция | n | p50, мс | p95, мс | max, мс |
|---|---:|---:|---:|---:|
| Store create | 500 | 1,362 | 2,523 | 20,723 |
| Store list | 200 | 2,348 | 3,545 | 6,194 |
| Store detail | 200 | 0,295 | 0,522 | 1,067 |
| HTTP snapshot | 100 | 4,846 | 8,405 | 28,728 |
| Unauthorized snapshot | 50 | 0,583 | 0,925 | 1,309 |

## Ограничения

Это:

- один localhost-run;
- synthetic SQLite database;
- последовательные операции;
- без concurrency;
- без cold/warm разделения;
- без raw samples;
- без environment manifest;
- без source/diff digest;
- не throughput benchmark;
- не SLA.

## Что улучшить

1. использовать documented nearest-rank percentile;
2. сохранять raw samples или histogram;
3. выполнять минимум 10 fresh-process runs;
4. тестировать cardinalities 100, 500, 5k, 50k;
5. добавить concurrency 1/4/16/64;
6. добавить mixed 80/20 read/write;
7. отдельно тестировать SSE lag и slow consumers;
8. сохранять CPU, RAM, Node, SQLite, filesystem и source digest.

---

# 14. Приоритетный план улучшений

## P0 — безопасность и корректность

| Работа | Effort | Acceptance |
|---|---:|---|
| Current-acceptance enforcement | M | Старый worker не может закрыть новую revision |
| Deterministic evidence policy | M | Current FAIL блокирует DONE |
| PAUSE/STOP revision drift | S | Control работает после non-content edit |
| Transactional CAS | M | В гонке проходит только один PATCH |
| Storage-policy source provenance | S | Tracked fix, tests и typecheck PASS |
| Document-preservation gate | S–M | Отчёт нельзя незаметно сократить |

## P1 — orchestration и audit trail

| Работа | Effort | Результат |
|---|---:|---|
| Hard task budget | L | Нет запуска сверх лимита |
| Checkpoint salvage | M | Late failure не теряет работу |
| Child attempt graph | M | Все runs видны |
| Requested/actual model | M | Честная fallback history |
| Artifact lineage | M | Известен current report |
| Atomic completion decision | M | REVIEW закрывается корректно |
| Cross-process SSE reconciliation | M | Dashboard не пропускает события |

## P2 — UX и performance

| Работа | Effort |
|---|---:|
| Canonical server owner registry | M |
| Reproducible benchmark harness | M |
| Cost/KPI dashboard | M |
| Transactional numbered migrations | M–L |
| Slow-client/backpressure tests | M |

---

# 15. Рекомендуемый оптимальный workflow

```text
Preflight
  ↓
Один planner
  ↓
Один writer
  ↓
Два параллельных reviewer
  ↓
Условный fix-writer
  ↓
Детерминированные проверки
  ↓
Evidence materialization
  ↓
Parent completion decision
```

Ограничения:

```text
max child runs: 6
max architecture runs: 1
max review waves: 2
max fix waves: 1
hard budget: $3
failed budget: $0.25
```

---

# 16. Что делать сейчас

1. Исправить `MZ-CP-04`, `05`, `06`, `14`.
2. Зафиксировать storage-policy изменения в tracked source и повторить gates.
3. Добавить preservation acceptance для документов.
4. Реализовать task-level budget reservation.
5. Добавить checkpoint salvage в `pi-subagents`.
6. Расширить run binding до child attempts.
7. Добавить requested/actual model.
8. Добавить artifact lineage.
9. Исправить canonical server ownership и SSE reconciliation.
10. После этого провести один ограниченный контрольный workflow и сравнить:
   - стоимость;
   - количество runs;
   - duration;
   - failed cost;
   - evidence completeness;
   - closure latency;
   - сохранность документов.

Главный вывод: **VPN Recovery уже находится в хорошем состоянии. Основные риски теперь сосредоточены в Mazzy Control Panel — acceptance integrity, evidence policy, concurrency, budget enforcement, traceability и orchestration preservation gates.**
