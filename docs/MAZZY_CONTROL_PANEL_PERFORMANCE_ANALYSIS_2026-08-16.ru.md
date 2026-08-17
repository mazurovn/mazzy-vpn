# Почему Mazzy работает медленно

Главный вывод: **сам Control Panel и SQLite работают быстро**. Медленно работает прежде всего **LLM-orchestration: модели, повторные волны review, большой контекст, tool calls, provider failures и parent overhead**.

## 1. Что не является bottleneck

Synthetic benchmark Control Panel:

| Операция | p50 | p95 | max |
|---|---:|---:|---:|
| Create task | 1,36 мс | 2,52 мс | 20,72 мс |
| List 500 tasks | 2,35 мс | 3,55 мс | 6,19 мс |
| Task detail | 0,30 мс | 0,52 мс | 1,07 мс |
| HTTP snapshot | 4,85 мс | 8,41 мс | 28,73 мс |
| Unauthorized rejection | 0,58 мс | 0,93 мс | 1,31 мс |

Даже HTTP snapshot имеет p95 около 8 мс. Поэтому задержки в минуты создаются не SQLite и не API.

Оговорка: это localhost synthetic sequential benchmark на 500 задачах, а не production SLA.

---

# 2. Первый VPN-консилиум

Общий task cycle составил около **55 минут**.

## Критический путь child runs

| Фаза | Wall time |
|---|---:|
| Неуспешные Opus/Sonnet | 5,18 мин |
| Fallback architecture/review | 4,62 мин |
| Terra writer | 6,34 мин |
| Первая validation wave | 4,13 мин |
| Fix-writer | 2,75 мин |
| Focused validation | 1,40 мин |
| Final validation | 1,65 мин |
| **Критический child path** | **≈26 минут** |

Оставшиеся примерно **29 минут** ушли на:

- первоначальный запуск и TMPDIR-проблему;
- ожидание/resume;
- синтез результатов parent-моделью;
- создание и изменение Mazzy task;
- импорт комментариев;
- запуск новых workflow;
- чтение повторных артефактов;
- локальные проверки и подготовку handoff.

## Главные потери

### 1. Provider failures

Два Anthropic run:

- выполнялись параллельно по 5,18 минуты;
- сделали 45 turns и 48 tool calls;
- использовали более 1,6 млн cache-read токенов;
- стоили `$1.655`;
- завершились поздним HTTP 403.

Это около 29% полной стоимости.

### 2. Слишком много validation waves

Было:

- четыре успешных planner runs;
- четыре успешных reviewer runs;
- два writer runs.

Для этой задачи достаточно одного planner, одного writer, двух reviewer и условного fix-writer.

### 3. Повторный контекст

Финальные runs имели огромный input при небольшом output:

| Run | Input | Output |
|---|---:|---:|
| Final reviewer | 211 596 | 1 824 |
| Final planner | 113 609 | 1 133 |
| Focused reviewer | 94 754 | 2 564 |
| Focused planner | 95 111 | 2 963 |

Это признак того, что агенту передавался почти весь accumulated context, хотя для проверки требовался только актуальный diff и список findings.

---

# 3. Второй консилиум отчёта

| Агент | Время | Input | Cache read | Tools |
|---|---:|---:|---:|---:|
| `ops-planner-sol` | 8,25 мин | 175 399 | 1 973 760 | 112 |
| Benchmark reviewer | 5,85 мин | 128 248 | 986 624 | 58 |
| Terra writer | 3,77 мин | 70 682 | 259 072 | 20 |

Первые два агента работали параллельно, поэтому child critical path:

```text
8,25 + 3,77 ≈ 12 минут
```

## Почему planner занял 8 минут

Planner:

- прочитал большой отчёт;
- исследовал Control Panel source;
- проверял SQLite;
- запускал тесты;
- воспроизводил три дополнительных дефекта;
- анализировал процессы и серверы;
- выполнил 112 tool calls;
- работал на `high` thinking.

Исследование получилось полезным, но слишком широким для одной lane.

## Почему writer выпал из workflow

Runtime usage-budget accounting остановил Stage 2 после дорогой read-only wave. Writer пришлось запускать отдельным fallback workflow.

Это добавило:

- новый запуск;
- повторную загрузку контекста;
- новый mission/run lifecycle;
- дополнительные parent actions;
- потерю единого workflow graph.

---

# 4. Долгие локальные тесты

| Проверка | Время |
|---|---:|
| Control Panel tests | 6,65 с |
| Control Panel typecheck | 7,65 с |
| VPN plugin tests | 1,30 с |
| VPN plugin typecheck | 3,41 с |
| MAZZY_VPN full suite | **279,15 с** |

Полный MAZZY_VPN suite занимает **4 минуты 39 секунд**. Это реальный локальный bottleneck, но он запускался только один раз и не объясняет 55 минут первого консилиума.

Его не следует повторять после документационных изменений, если:

- исходный код не изменился;
- hash тестируемого source не изменился;
- предыдущий log привязан к source digest.

---

# 5. Почему web может казаться медленным

Хотя snapshot API быстрый, UI может поздно показывать обновления из-за архитектуры событий:

1. одновременно работают несколько Mazzy server processes;
2. SSE listeners находятся только в памяти процесса;
3. процесс A не знает о записи процесса B;
4. браузер при активном SSE прекращает fallback polling;
5. обновление появляется только после refresh/reconciliation.

Это не медленный SQL-запрос, а **задержка доставки событий**.

## Решение

- один canonical server owner на project DB;
- owner registry;
- heartbeat;
- event cursor;
- DB revision/last event ID в snapshot;
- обязательный snapshot reconciliation после reconnect;
- fallback polling при подозрении на stale cursor.

---

# 6. Что улучшить в первую очередь

## P0 — самые эффективные изменения

| Улучшение | Ожидаемый эффект |
|---|---|
| Model/provider preflight | Не запускать явно недоступную модель |
| Checkpoint после каждой фазы | Не терять работу при позднем 403 |
| Provider circuit breaker | Не повторять non-retryable provider |
| Hard task budget | Не превышать установленную стоимость |
| Резерв бюджета для writer | Writer не выпадет после дорогого review |
| Максимум две review waves | Исключить orchestration-loop |
| Diff-only context | Снизить input tokens в 2–4 раза |
| Stop rule | Завершать после PASS и отсутствия blockers |
| Idempotent resume | Не повторять завершённые стадии |
| Single server owner | Устранить задержку dashboard events |

---

# 7. Оптимальный workflow

```text
Preflight: модели, storage, budget
               ↓
Один planner
               ↓
Один writer
               ↓
Два reviewer параллельно
               ↓
Условный fix-writer
               ↓
Детерминированные проверки
               ↓
DONE
```

## Ограничения

```text
maxTotalChildRuns = 6
maxArchitectureRuns = 1
maxReviewWaves = 2
maxFixWaves = 1
maxTaskCostUsd = 3
maxFailedCostUsd = 0.25
maxAgentMinutes = 25
```

---

# 8. Как уменьшить контекст

## Planner

Передавать:

- acceptance criteria;
- связанные source-файлы;
- конкретные риски;
- тестовые seams.

Не передавать:

- полную историю чата;
- все предыдущие отчёты;
- повторяющиеся logs;
- весь repository context.

## Reviewer

Передавать:

- актуальный diff;
- acceptance criteria;
- результаты тестов;
- изменённые функции;
- известные residual risks.

## Fix-writer

Передавать только:

- accepted findings;
- текущий diff;
- связанные source slices;
- необходимые regression tests.

---

# 9. Рекомендуемые технические лимиты

Для read-only planner/reviewer допустимы жёсткие ограничения:

| Параметр | Цель |
|---|---:|
| Input на agent | ≤50 000 |
| Output | ≤10 000 |
| Tool calls | ≤30–40 |
| Turns | ≤6–8 |
| Runtime | ≤5 минут |
| Повторные чтения одного файла | 0–1 |

Для writer лучше использовать узкий scope и elapsed monitoring, а не слишком жёсткий tool cap.

---

# 10. Кэширование проверок

Нужен ключ:

```text
verificationKey =
  hash(source)
  + hash(test configuration)
  + runtime version
  + environment class
  + command
```

Если ключ не изменился, можно переиспользовать:

- полный MAZZY_VPN suite;
- typecheck;
- package tests;
- benchmark;
- reviewer verdict для неизменённого diff.

Нельзя переиспользовать результат после изменения source, acceptance revision или environment.

---

# 11. Целевые показатели

| Метрика | Сейчас | Цель |
|---|---:|---:|
| VPN task cycle | ~55 мин | 20–25 мин |
| Report consilium | ~17 мин | 8–10 мин |
| Child runs | 12 | 4–6 |
| Child cost | `$5.72` | `$1.8–$3.0` |
| Failed cost share | 28,94% | <5% |
| Input/output tokens | 971 585 | <450 000 |
| Read-only runs | 8 | 2–4 |
| Maximum planner tools | 112 | ≤40 |
| Dashboard snapshot p95 | 8,4 мс | уже достаточно |
| MAZZY_VPN full suite | 279 с | кэшировать по source digest |
| Closure latency | задача осталась REVIEW | ≤2 минуты после gate |

# Итог

Система медленная не из-за базы данных и не из-за HTTP API. Основные причины:

1. top models с `high` thinking;
2. слишком широкий scope агентов;
3. огромный повторный контекст;
4. поздние provider failures;
5. много повторных validation waves;
6. отсутствие task-level budget;
7. неидемпотентный resume;
8. повторные parent synthesis и lifecycle actions;
9. долгий полный MAZZY_VPN suite;
10. межпроцессная задержка SSE в dashboard.

Самые выгодные изменения: **checkpoint salvage, hard task budget, writer budget reservation, diff-only context, ограничение review waves, idempotent resume и один canonical dashboard owner**.
