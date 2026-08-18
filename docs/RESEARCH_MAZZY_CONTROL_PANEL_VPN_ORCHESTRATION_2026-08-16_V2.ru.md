# Mazzy Control Panel и VPN orchestration — полный отчёт, версия 2

**Дата:** 2026-08-16  
**Статус:** расширенная версия; исходный отчёт сохранён полностью и byte-for-byte  
**Назначение:** объединить исходное исследование, дополнительный консилиум, полный issue log, benchmarks, тестовые логи и разбор нарушения директивы сохранности.

## Навигация по версии 2

1. **Часть I** — полный исходный отчёт без сокращений.
2. **Часть II** — доказательная дельта дополнительного консилиума.
3. **Часть III** — полный журнал проблем, включая причины ошибочного сокращения.
4. Воспроизводимые логи и agent handoffs находятся в `docs/MAZZY_CONTROL_PANEL_AUDIT_2026-08-16/`.
5. Исходная и ошибочно сокращённая версии находятся в `docs/REPORT_RECOVERY_2026-08-16/`.

## Контроль сохранности

| Объект | Строки | Байты | Статус |
|---|---:|---:|---|
| Исходный полный отчёт | 1134 | 48 514 | сохранён полностью |
| Ошибочно сокращённая версия | 87 | 11 432 | сохранена только для аудита |
| Версия 2 | вычисляется manifest | вычисляется manifest | исходник + дополнения |

> Важно: сокращённая версия не является заменой исходного отчёта. Она включена ниже только как дополнительный evidence-led summary и объект анализа orchestration-регрессии.

---

# Часть I. Полный исходный отчёт

# Исследование Mazzy Control Panel и orchestration на задаче VPN Recovery

**Дата исследования:** 2026-08-16  
**Статус документа:** итоговый расширенный технический отчёт  
**Исследуемая задача:** `17f6f71d-4d54-438a-91e7-5ecd5d6fa5f3`  
**Название задачи:** `Improve pi-vpn-recovery transport-health checks and doctor surface`  
**Control plane:** Mazzy Control Panel  
**Child runtime:** `pi-subagents`  
**Репозиторий реализации:** `<local-workspace>/pi-vpn-recovery`  

---

## 1. Резюме для руководителя

Mazzy Control Panel и `pi-subagents` довели рискованное изменение механизма VPN recovery до технически сильного результата:

- реализованы режимы `fast`, `balanced`, `deep`;
- добавлен недеструктивный `/vpn-recovery doctor`;
- статус VPN отделён от реального транспортного здоровья;
- provider-only сбой больше не должен вызывать destructive reconnect;
- сохранены lock, cooldown, rate window, circuit breaker и deadline-защита;
- тестовый набор вырос с 11 до 19 тестов;
- финальные тесты, typecheck и live doctor прошли;
- две финальные независимые проверки не нашли блокирующих дефектов.

Главная ценность orchestration состоит в том, что независимые валидаторы действительно обнаружили ошибки, которые прошли исходные тесты:

1. ложный `transport=PASS` из-за включения служебных сигналов в quorum;
2. отсутствие `WARN` при релевантной деградации IPv6;
3. недостаточный deadline для долгого custom verify;
4. отдельный недочёт lease при reconciliation.

Однако orchestration была существенно дороже и сложнее необходимого:

- 12 измеренных child runs;
- 10 успешных и 2 завершившихся ошибкой;
- 971 585 input/output токенов без учёта 5 379 242 cache-read токенов;
- 39,96 суммарных agent-минут;
- известная child-стоимость `$5.719125` без стоимости parent-orchestrator;
- 28,94% стоимости ушло на неуспешные Anthropic runs;
- только 12,82% общей стоимости пришлось на writer/fix-writer;
- задача осталась в `REVIEW`, несмотря на финальный отчёт без блокеров;
- в durable state обнаружено 0 записей evidence и только 1 run binding при 12 измеренных child runs;
- web dashboard и SQLite state показали разные количества задач.

### Итоговая оценка

| Область | Оценка | Краткое объяснение |
|---|---:|---|
| Качество итогового VPN-решения | 9/10 | Реальные safety-дефекты найдены и устранены |
| Независимость review | 9/10 | Несколько fresh-context проверок с воспроизводимыми findings |
| Безопасность мутаций | 8/10 | Один writer на фазу, узкий scope, без commit/push и без Orca |
| Надёжность orchestration | 6/10 | Timeout/resume, Anthropic 403, незавершённый FSM |
| Трассируемость control plane | 6/10 | Комментарии и review есть, но run/evidence coverage неполная |
| Скорость | 5/10 | Около 55 минут task cycle и почти 40 agent-минут |
| Cost-efficiency | 4/10 | Слишком много read-only волн и дорогие failed runs |
| Web/control-plane UX | 5/10 | Port ownership и расхождение snapshot/state |
| **Общая оценка** | **7/10** | Качественный результат, но orchestration требует оптимизации |

При выполнении рекомендаций из этого отчёта реалистичная целевая оценка Mazzy для подобных задач — **8,5–9/10**, а ожидаемая child-стоимость — **$1.8–$3.0** при сохранении двух независимых проверок.

---

## 2. Цель и границы исследования

### 2.1. Цель

Исследование отвечает на вопросы:

1. насколько хорошо Mazzy Control Panel выполнил роль durable control plane;
2. насколько качественно `pi-subagents` выполнил планирование, реализацию и review;
3. какие дефекты были предотвращены за счёт multi-agent orchestration;
4. почему стоимость и длительность оказались высокими;
5. какие изменения нужны в routing, budget control, FSM, dashboard и run lifecycle;
6. какие метрики следует использовать для последующих orchestration-задач.

### 2.2. В область исследования входят

- задача Mazzy `17f6f71d-4d54-438a-91e7-5ecd5d6fa5f3`;
- routing policy `.pi/mazzy/routing.json`;
- профиль `.pi/agents/mazzy-orchestrator.md`;
- subagent metadata и transcripts;
- writer/fix-writer reports;
- три волны validation outputs;
- SQLite state `.pi-ops/state.db`;
- web/status snapshot Mazzy;
- итоговое состояние `pi-vpn-recovery`.

### 2.3. Не входят

- стоимость parent-модели, так как единая подтверждённая метрика отсутствует;
- оценка всего backlog по содержанию каждой из 62 задач;
- destructive проверка VPN reconnect в рамках настоящего отчёта;
- управление браузером или Orca;
- изменение Mazzy Control Panel — данный документ только исследует и предлагает backlog.

---

## 3. Источники и методика

### 3.1. Исследованные источники

| Источник | Назначение |
|---|---|
| `.pi/mazzy/routing.json` | модели, lanes, бюджеты и escalation policy |
| `.pi/agents/mazzy-orchestrator.md` | контракт parent orchestration и комментариев |
| `.mazzy/work/sessions/subagent-artifacts/*_meta.json` | model, usage, duration, tool count, error |
| `.mazzy/outputs/pi-vpn-recovery-*.md` | writer и validation reports |
| `.mazzy/work/tmp/pi-subagents-*/status.json` | workflow lifecycle и acceptance data |
| `.pi-ops/state.db` | tasks, events, comments, run bindings, review reports, evidence |
| `MY_PLUGINS/pi-vpn-recovery/**` | фактический изменённый scope |
| live `/mazzy` status | web snapshot и backlog counters |

### 3.2. Расчёт метрик

Для 12 VPN-consilium runs агрегированы:

- `durationMs`;
- `usage.input`;
- `usage.output`;
- `usage.cacheRead`;
- `usage.cost`;
- `usage.turns`;
- `toolCount`;
- `exitCode` и `error`.

Стоимость в отчёте — provider-reported metadata cost. Она не включает parent agent и инфраструктурные процессы. Cache-read токены показаны отдельно и не добавлены повторно к input/output total.

### 3.3. Ограничения точности

1. `processSignal=SIGTERM` встречается и у успешных runs с `exitCode=0`; это похоже на штатное завершение harness, поэтому успех определялся по `exitCode`, acceptance report и status.
2. Failed Anthropic runs успели сделать 45 turns и 48 tool calls. Это не простой отказ при первом запросе: ошибка 403 возникла на поздней стадии либо после частично выполненной работы.
3. Web snapshot и SQLite state расходятся; оба значения сохранены как наблюдения, а не искусственно согласованы.
4. Некоторые validator runs не представлены отдельными `run_bindings` в control-plane DB, хотя их metadata и outputs существуют.

---

## 4. Исходная архитектура orchestration

### 4.1. Routing policy

Настроенные роли:

| Lane | Агент | Запрошенная модель | Risk ceiling |
|---|---|---|---|
| bounded-recon | `ops-scout-mini` | `openai-codex/gpt-5.4-mini` | medium |
| review | `ops-review-sonnet` | `anthropic/claude-sonnet-5` | high |
| implementation | `ops-implement-terra` | `openai-codex/gpt-5.6-terra` | critical |
| architecture | `ops-planner-opus5` | `anthropic/claude-opus-5` | critical |
| package-research | `ops-research-opus48` | `anthropic/claude-opus-4-8` | medium |

Configured budgets:

```json
{
  "defaultUsd": 0.25,
  "highRiskUsd": 2,
  "maxChildrenPerWave": 2
}
```

### 4.2. Фактическая модель выполнения

```text
Mazzy durable task
       |
       v
parent/orchestrator
       |
       +--> architecture + review wave
       |       |
       |       +--> Anthropic runs завершились 403
       |       +--> OpenAI GPT-5.4 fallback
       |
       +--> single Terra writer
       |
       +--> fresh validation wave
       |       |
       |       +--> quorum и deadline findings
       |
       +--> Terra fix-writer
       |
       +--> focused validation
       |       |
       |       +--> reconciliation lease finding
       |
       +--> residual fix
       |
       +--> final fresh validation pair
               |
               +--> no blockers
```

### 4.3. Соблюдение организационных ограничений

| Ограничение | Результат |
|---|---|
| Использовать Mazzy как control plane | Соблюдено |
| Использовать только `pi-subagents` как child runtime | Соблюдено |
| Не запускать Orca | Соблюдено |
| Один writer одновременно | Соблюдено по фазам |
| Изменять только plugin scope | Соблюдено |
| Не commit/push | Соблюдено |
| Не выполнять лишний destructive reconnect | Соблюдено агентами; live auto-recovery произошёл по реальному событию |
| Хранить новые artifacts в проекте | После коррекции соблюдено |

---

## 5. Хронология

### 5.1. Durable task lifecycle

| Время UTC | Событие |
|---|---|
| 17:00:08 | Создана задача в `BACKLOG` |
| 17:00:18 | `BACKLOG → READY` |
| 17:00:21 | `READY → CLAIMED` |
| 17:23:51 | Зафиксирован результат первой read-only wave и проблемы запуска |
| 17:31:25 | Принят синтезированный implementation plan |
| 17:32:01 | Назначен writer run |
| 17:39:01 | Зафиксировано завершение основной реализации |
| 17:44:43 | Validators нашли fix-now defects |
| 17:47:57 | Fix-writer завершил основную коррекцию |
| 17:51:52 | Focused validation нашла residual reconciliation issue |
| 17:54:36 | Parent повторно запустил тесты/typecheck |
| 17:55:18 | Completion attested и создан review report |
| После завершения | Задача осталась в `REVIEW` |

Task cycle от создания до последнего update: примерно **55 минут 11 секунд**.

### 5.2. Эволюция тестов

| Этап | Количество тестов | Статус |
|---|---:|---|
| До redesign | 11 | PASS |
| Основной writer | 14 | PASS |
| После quorum/IPv6/deadline fix | 18 | PASS |
| После reconciliation regression | 19 | PASS |

Прирост: **8 тестов**, или примерно **72,7%** относительно исходного набора.

---

## 6. Полная таблица child runs

| Run | Роль | Фактическая модель | Результат | Мин | Input | Output | Cache read | Cost USD | Turns | Tools |
|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `35ac5f57` | planner | Anthropic Opus 5 | FAIL 403 | 5.18 | 2 138 | 13 409 | 738 678 | 1.043129 | 23 | 25 |
| `30881af4` | reviewer | Anthropic Sonnet 5 | FAIL 403 | 5.18 | 1 994 | 22 617 | 876 852 | 0.611871 | 22 | 23 |
| `608b5f54` | planner | OpenAI GPT-5.4 | PASS | 4.62 | 38 227 | 14 168 | 129 024 | 0.340344 | 8 | 14 |
| `4c2a161d` | reviewer | OpenAI GPT-5.4 | PASS | 4.31 | 33 995 | 12 526 | 167 424 | 0.314734 | 10 | 21 |
| `88ffcb7d` | writer | GPT-5.6 Terra | PASS | 6.34 | 44 824 | 18 149 | 376 832 | 0.382802 | 15 | 20 |
| `ef4ac338` | reviewer | OpenAI GPT-5.4 | PASS | 4.13 | 74 140 | 11 849 | 447 488 | 0.474957 | 9 | 14 |
| `91879dcd` | planner | OpenAI GPT-5.4 | PASS | 2.41 | 61 827 | 5 704 | 375 296 | 0.333952 | 9 | 12 |
| `26a7f7a8` | fix-writer | GPT-5.6 Terra | PASS | 2.75 | 85 114 | 7 350 | 458 752 | 0.350178 | 7 | 6 |
| `db478ce8` | reviewer | OpenAI GPT-5.4 | PASS | 1.24 | 94 754 | 2 564 | 330 752 | 0.358033 | 5 | 8 |
| `5dc87452` | planner | OpenAI GPT-5.4 | PASS | 1.40 | 95 111 | 2 963 | 420 352 | 0.387311 | 6 | 8 |
| `68e730db` | reviewer | OpenAI GPT-5.4 | PASS | 1.65 | 211 596 | 1 824 | 872 960 | 0.774590 | 10 | 12 |
| `957014b5` | planner | OpenAI GPT-5.4 | PASS | 0.73 | 113 609 | 1 133 | 184 832 | 0.347226 | 3 | 6 |

### 6.1. Сводка

| Метрика | Все runs | Успешные | Неуспешные |
|---|---:|---:|---:|
| Runs | 12 | 10 | 2 |
| Agent-время, мин | 39.96 | 29.59 | 10.37 |
| Input tokens | 857 329 | 853 197 | 4 132 |
| Output tokens | 114 256 | 78 230 | 36 026 |
| Input + output | 971 585 | 931 427 | 40 158 |
| Cache read | 5 379 242 | 3 763 712 | 1 615 530 |
| Turns | 127 | 82 | 45 |
| Tool calls | 169 | 121 | 48 |
| Cost USD | 5.719125 | 4.064125 | 1.655000 |

### 6.2. По ролям

| Роль | Runs | Success | Мин | Tokens | Cache read | Cost USD |
|---|---:|---:|---:|---:|---:|---:|
| `ops-implement-terra` | 2 | 2 | 9.09 | 155 437 | 835 584 | 0.732981 |
| `ops-planner-opus5` | 5 | 4 | 14.35 | 348 289 | 1 848 182 | 2.451960 |
| `ops-review-sonnet` | 5 | 4 | 16.52 | 467 859 | 2 695 476 | 2.534184 |

### 6.3. По providers

| Provider | Runs | Success | Мин | Tokens | Cache read | Cost USD |
|---|---:|---:|---:|---:|---:|---:|
| Anthropic | 2 | 0 | 10.37 | 40 158 | 1 615 530 | 1.655000 |
| OpenAI Codex | 10 | 10 | 29.59 | 931 427 | 3 763 712 | 4.064125 |

---

## 7. Анализ стоимости

### 7.1. Почему `$5.72` получило низкую оценку cost-efficiency

Абсолютно `$5.72` — умеренная сумма. Низкая оценка относится к эффективности для задачи, затронувшей три файла.

Основные коэффициенты:

| Показатель | Значение |
|---|---:|
| Failed cost share | 28.94% |
| Writer share общей стоимости | 12.82% |
| Planner + reviewer share общей стоимости | 87.18% |
| Successful read-only share successful cost | около 81.96% |
| Средняя стоимость успешного run | `$0.4064` |
| Стоимость на изменённый файл | около `$1.91` без parent cost |
| Runs на изменённый файл | 4.0 |

Review был полезен, но количество повторных read-only запусков оказалось чрезмерным.

### 7.2. Budget-policy mismatch

Routing policy декларирует `highRiskUsd: 2`, однако наблюдаемая child-стоимость задачи — `$5.719125`.

Возможные объяснения:

1. бюджет применяется на отдельную wave или run, а не на task;
2. это advisory policy без runtime enforcement;
3. fallback/retry runs не входят в исходный budget envelope;
4. parent resume создал новый budget context;
5. provider-reported cost не сверяется с Mazzy task budget.

Независимо от причины пользователь воспринимает бюджет как task-level ограничение. Поэтому семантика должна быть явной:

```text
budget.scope = task | wave | run
budget.enforcement = hard | soft | observe-only
```

### 7.3. Почему failed runs были особенно дорогими

Оба Anthropic run не отказали мгновенно. Вместе они успели выполнить:

- 45 turns;
- 48 tool calls;
- 36 026 output tokens;
- 1 615 530 cache-read tokens;
- 10,37 agent-минут.

Это означает, что простого startup preflight недостаточно. Нужны:

- checkpointing после каждого значимого finding;
- salvage partial output при поздней provider error;
- retry с continuation context, а не полный restart;
- circuit breaker после первой неповторимой 403;
- provider health state на уровне task/session.

---

## 8. Качество работы агентов

### 8.1. Terra writer

**Сильные стороны:**

- удержал scope в трёх разрешённых файлах;
- реализовал структурированный probe runner;
- добавил modes, doctor и documentation;
- выполнил тесты/typecheck/diff checks;
- предоставил acceptance report;
- fix-writer быстро устранил конкретные findings.

**Недостатки:**

- первая версия содержала high-severity quorum bug;
- deadline coverage был исправлен не во всех путях сразу;
- fix-writer получил 85 114 input tokens для относительно компактной коррекции;
- metadata fix-run указала file mutation как `not-applicable/expected=false`, хотя run фактически был writer — это несовпадение launch/effect contract.

**Оценка:** 8/10.

### 8.2. Planner/reviewer fallback на GPT-5.4

**Сильные стороны:**

- нашли воспроизводимые, а не гипотетические дефекты;
- приложили минимальные repro harnesses;
- правильно разделили transport и advisory evidence;
- обнаружили отдельный reconciliation edge case после основной коррекции;
- финальные validators подтвердили отсутствие блокеров.

**Недостатки:**

- четыре planner и четыре reviewer success runs — слишком много;
- последние runs имели очень высокий input при малом output;
- роли назывались `opus5`/`sonnet`, хотя фактическая модель была GPT-5.4;
- повторные validation waves не были достаточно delta-oriented.

**Оценка качества:** 9/10.  
**Оценка эффективности:** 5/10.

### 8.3. Anthropic Opus/Sonnet runs

Оба завершились 403, но после значительного числа turns и tool calls. Полезный финальный artifact не был принят как успешный результат.

**Оценка доступности:** 2/10.  
**Оценка fault tolerance orchestration:** 4/10.

### 8.4. Parent orchestrator

**Сильные стороны:**

- сохранил single-writer discipline;
- синтезировал findings и ограничивал fix scope;
- импортировал milestones в task discussion;
- продолжил работу после timeout;
- добился финальных независимых `no blockers` verdicts.

**Недостатки:**

- первый run потребовал resume;
- повторил больше validation waves, чем требовалось;
- не довёл FSM до `DONE`;
- не обеспечил полное durable binding/evidence coverage;
- fallback был организационно успешным, но не автоматическим и не бюджетно ограниченным.

**Оценка:** 7/10.

---

## 9. Что именно обнаружил review

| Finding | Severity | Почему опасно | Исправление | Regression |
|---|---|---|---|---|
| Служебные probes участвовали в fast quorum | High | Возможен `transport=PASS` без реального интернета | Quorum ограничен DNS/ICMP/Google/Cloudflare | Fast false-pass test |
| Base signals могли спасать balanced/deep round | High | Ложное подтверждение стабильности | Строгий general probe membership для всех rounds | Balanced/deep first-round test |
| Relevant IPv6 failure не влиял на aggregate | Medium | Doctor показывал PASS при IPv6 degradation | Aggregate `WARN`, IPv4 transport остаётся отдельным | IPv6 WARN test |
| Recovery lease не покрывал long custom verify | Medium | Второй process мог reclaim lock во время verify | Deadline из actual verify budget + headroom | Long verify deadline test |
| Reconciliation lease оставался 45 секунд | Medium | One-writer нарушался в interrupted reconciliation | Shared `verificationBudgetMs(...)` | Reconciliation lease test |

Это подтверждает, что независимое review было не формальностью. Без него итоговая реализация могла бы неверно считать интернет здоровым и нарушать lock ownership.

---

## 10. Состояние Mazzy durable control plane

### 10.1. Состояние задачи

| Поле | Значение |
|---|---|
| Task ID | `17f6f71d-4d54-438a-91e7-5ecd5d6fa5f3` |
| State | `REVIEW` |
| Priority | 55 |
| Risk | high |
| Revision | 5 |
| Executor | `ops-implement-terra` |
| Events | 18 |
| Comments | 12 |
| Run bindings | 1 |
| Evidence rows | 0 |
| Review reports | 1 |

### 10.2. Положительные стороны

- task и история сохраняются в SQLite;
- есть строгая state machine;
- milestones импортированы в comments;
- completion attestation существует;
- review report содержит acceptance criteria и результаты;
- writer binding содержит model, parent session и lifecycle.

### 10.3. Проблемы трассируемости

#### Проблема A: задача не закрыта

Финальные validators сообщили `no blockers`, tests/typecheck прошли, review report создан, но state остаётся `REVIEW`.

Вероятная причина: FSM требует verifier evidence или явное independent acceptance действие, которое не было материализовано в expected table/transition.

#### Проблема B: evidence table пуста

`evidence=0`, хотя фактически существуют:

- test results;
- typecheck results;
- diff check;
- live doctor;
- two final validator reports;
- review report.

Комментарии по контракту «не являются evidence», поэтому отсутствие evidence логически блокирует автоматический `DONE`.

#### Проблема C: неполное run binding coverage

В DB есть один binding для writer workflow `d75c4fbe...`, тогда как исследовано 12 child runs. Если дизайн намеренно связывает только workflow, UI должен отображать дочерние runs внутри binding. Если ожидается child-level traceability, покрытие составляет только 1/12, или 8,3%.

#### Проблема D: fix-writer traceability

Fix-writer имел отдельный workflow run, но worker comments привязаны к основному `d75c4fbe...`. Это затрудняет ответ на вопрос «какой именно run внёс конкретную коррекцию».

---

## 11. Web dashboard и backlog

### 11.1. Наблюдение web status

Один live `/mazzy` snapshot сообщил:

```text
Mazzy: 53 tasks
backlog 40
ready 1
running 0
review 4
blocked 0
```

### 11.2. Наблюдение SQLite после этого

Прямой read-only query `.pi-ops/state.db` показал:

| State | Count |
|---|---:|
| BACKLOG | 47 |
| READY | 2 |
| REVIEW | 4 |
| DONE | 6 |
| CANCELLED | 3 |
| **Всего** | **62** |

### 11.3. Интерпретация расхождения

Разница составляет 9 задач по total count. Возможные причины:

1. dashboard server держит stale in-memory snapshot;
2. разные sessions используют разные database resolution/context;
3. websocket/reactive refresh не получил новые events;
4. dashboard показывает фильтр, но label ошибочно говорит total;
5. server ownership привёл к подключению к другому process/port.

### 11.4. Port ownership problem

Наблюдалось:

- существующий server на `127.0.0.1:4320`, принадлежащий другой Pi session;
- `/mazzy-server status`: `server is not owned by this session`;
- `/mazzy` возвращал canonical dashboard `127.0.0.1:4319`, где listener отсутствовал;
- для безопасного показа был поднят отдельный project-local owner на `4322` с one-time bootstrap на `4323`.

Это UX и consistency issue: discovery command должен возвращать работающий endpoint либо явно перечислять active servers.

---

## 12. Каталог проблем

| ID | Проблема | Severity | Влияние | Корневая причина |
|---|---|---|---|---|
| MZ-01 | Late Anthropic 403 после частичной работы | High | Потеря времени и `$1.655` | Нет checkpoint salvage/circuit breaker |
| MZ-02 | Task budget `$2` не ограничил `$5.72` | High | Непредсказуемый расход | Неясный budget scope/enforcement |
| MZ-03 | 8 успешных read-only runs | Medium | Высокие токены/стоимость | Нет max validation waves и delta review |
| MZ-04 | Resume недостаточно идемпотентен | High | Повторные runs/context | Нет diff-hash phase keys |
| MZ-05 | Финальная задача зависла в REVIEW | High | Backlog не отражает завершение | Evidence/FSM integration gap |
| MZ-06 | Evidence table пуста | High | Нельзя автоматически принять результат | Reports/comments не материализуются как evidence |
| MZ-07 | Только один durable run binding | Medium | Неполный audit trail | Workflow-child mapping не отражён |
| MZ-08 | Agent name не соответствует actual model | Medium | Ошибочная интерпретация качества | Role и model смешаны в имени |
| MZ-09 | Dashboard snapshot расходится с DB | High | Неверный backlog | Stale state или context mismatch |
| MZ-10 | Dashboard URL указывает на dead port | Medium | Плохой UX/доступ | Session-local ownership discovery |
| MZ-11 | Fix-writer contract metadata inconsistent | Medium | Неверный effects audit | Launch contract пометил mutation как unexpected |
| MZ-12 | Большой input в коротких final runs | Medium | Cost inflation | Полный контекст вместо diff-only packet |
| MZ-13 | Outputs разных стадий не помечены superseded | Low | Можно прочитать старые 14/18 test reports как финальные | Нет artifact lineage |
| MZ-14 | Failed work не переиспользуется | High | Полный fallback restart | Нет partial-result salvage |

---

## 13. Рекомендуемая целевая orchestration

### 13.1. Оптимальный pipeline для high-risk VPN change

```text
0. Preflight
   - storage/TMPDIR
   - provider/model health
   - dashboard/control-plane DB
   - budget reservation

1. One architecture run
   - read-only
   - invariants + acceptance + test plan

2. One writer run
   - one writer
   - restricted file allowlist
   - deterministic tests

3. One parallel review wave
   - safety reviewer
   - correctness/test reviewer

4. Conditional fix-writer
   - only if blocker/high finding
   - receives findings + current diff only

5. Deterministic final gate
   - tests/typecheck/diff/doctor

6. Optional final reviewer
   - only if fix changed architecture or safety invariant

7. Evidence materialization
   - attach command results and review verdict
   - transition REVIEW → DONE
```

### 13.2. Stop rule

```text
IF tests == PASS
AND typecheck == PASS
AND diff_check == PASS
AND live_doctor IN {PASS, WARN-with-documented-advisory}
AND independent_reviewers >= 2
AND blocker_count == 0
AND high_finding_count == 0
THEN materialize evidence
AND close task
ELSE launch at most one fix wave
```

### 13.3. Anti-loop rule

```text
maxArchitectureRuns = 1
maxInitialReviewers = 2
maxFixWaves = 1
maxFinalReviewers = 2
maxTotalChildRuns = 6
```

Для исключений требуется записанный parent decision с дополнительным budget reservation.

---

## 14. Рекомендации по budget control

### 14.1. Task-level budget contract

```json
{
  "budget": {
    "scope": "task",
    "enforcement": "hard",
    "maxCostUsd": 3.0,
    "maxFailedCostUsd": 0.25,
    "maxRuns": 6,
    "maxInputOutputTokens": 450000,
    "maxAgentMinutes": 25
  }
}
```

### 14.2. Budget reservation по фазам

| Фаза | Target USD | Hard max USD |
|---|---:|---:|
| Preflight | 0.03 | 0.05 |
| Architecture | 0.30 | 0.50 |
| Writer | 0.50 | 0.80 |
| Two reviewers | 0.60 | 1.00 |
| Conditional fix | 0.25 | 0.45 |
| Conditional final reviewer | 0.25 | 0.40 |
| Reserve | 0.20 | 0.30 |
| **Итого** | **2.13** | **3.00** |

### 14.3. Runtime enforcement

Перед каждым run:

```text
estimatedCost = modelPrice × estimatedContext × expectedTurns
remainingBudget = taskBudget - observedCost - reservedCost

if estimatedCost > remainingBudget:
    select cheaper eligible model
    or reduce context
    or request explicit budget override
```

### 14.4. Failed-cost breaker

```text
on non-retryable 401/403:
    mark provider unavailable for task
    salvage latest checkpoint
    do not retry same model

on failedCost > maxFailedCostUsd:
    stop automatic retries
    require parent decision
```

---

## 15. Рекомендации по context efficiency

### 15.1. Context packets по ролям

**Planner получает:**

- acceptance criteria;
- relevant source files;
- existing tests;
- safety invariants;
- не весь historical transcript.

**Writer получает:**

- approved plan;
- file allowlist;
- current source/tests;
- required commands;
- explicit non-goals.

**Reviewer получает:**

- current diff;
- acceptance criteria;
- changed functions;
- test output;
- known residual risks.

**Fix-writer получает:**

- только accepted findings;
- minimal source slices;
- current diff;
- target regressions.

### 15.2. Delta validation

Ключ review:

```text
reviewKey = sha256(diff + acceptanceRevision + reviewerRole)
```

Если diff не изменился, повторный reviewer не запускается. Если изменились только две функции, validator получает эти hunks, а не полный initial consilium context.

### 15.3. Target token reductions

| Фаза | Наблюдалось | Цель |
|---|---:|---:|
| Общий input/output | 971 585 | ≤450 000 |
| Fix-writer input | 85 114 | ≤35 000 |
| Финальный reviewer input | до 211 596 | ≤45 000 |
| Финальный planner input | до 113 609 | ≤45 000 |
| Cache read | 5 379 242 | отслеживать отдельно, снижать дублирование |

---

## 16. Рекомендации по model routing

### 16.1. Разделить role и requested model

Вместо:

```text
ops-review-sonnet
ops-planner-opus5
```

использовать:

```text
role: ops-safety-reviewer
requestedModel: anthropic/claude-sonnet-5
actualModel: openai-codex/gpt-5.4
fallbackReason: provider-403
```

### 16.2. Явная fallback chain

```json
{
  "review": {
    "models": [
      "anthropic/claude-sonnet-5",
      "openai-codex/gpt-5.4",
      "openai-codex/gpt-5.4-mini"
    ]
  },
  "architecture": {
    "models": [
      "anthropic/claude-opus-5",
      "openai-codex/gpt-5.4"
    ]
  }
}
```

### 16.3. Checkpoint salvage

Каждый child после значимого этапа пишет структурированный checkpoint:

```json
{
  "runId": "...",
  "phase": "findings-complete",
  "diffDigest": "...",
  "findings": [],
  "commandsRun": [],
  "remainingWork": []
}
```

Если provider падает поздно, fallback продолжает с checkpoint, а не повторяет 20+ turns.

---

## 17. Рекомендации по FSM и evidence

### 17.1. Materialization pipeline

```text
subagent acceptance report
        |
        v
validator parses structured fields
        |
        +--> evidence: tests
        +--> evidence: typecheck
        +--> evidence: diff-check
        +--> evidence: live-doctor
        +--> review_reports: reviewer verdicts
        |
        v
acceptance gate
        |
        +--> PASS → DONE
        +--> FAIL → BLOCKED or RUNNING fix wave
```

### 17.2. Минимальный evidence set для high-risk plugin

| Evidence | Required |
|---|---:|
| Changed-file allowlist | Да |
| Tests with exact count | Да |
| Typecheck | Да |
| Diff/staged-file check | Да |
| Independent reviewer verdict | Да, минимум 2 |
| Safe live doctor | Да |
| Destructive live test | Нет, только по отдельному разрешению |
| Residual-risk statement | Да |

### 17.3. Автоматическое закрытие текущей задачи

Для задачи уже имеются фактические данные, достаточные для закрытия:

- 19 tests PASS;
- typecheck PASS;
- diff check PASS;
- doctor PASS;
- two final validators: no blockers;
- review report создан.

Нужно:

1. материализовать перечисленное в `evidence`;
2. привязать final validator runs;
3. отметить superseded intermediate reports;
4. выполнить acceptance gate;
5. перевести `REVIEW → DONE`.

---

## 18. Рекомендации по dashboard

### 18.1. Обязательные панели

#### Task summary

- state и state age;
- acceptance revision;
- executor;
- blocker count;
- evidence completeness;
- actual vs budget cost.

#### Run graph

```text
architecture
  └─ fallback
writer
  └─ validation pair
      └─ fix writer
          └─ focused validation
              └─ final validation pair
```

#### Cost panel

| Поле | Значение |
|---|---:|
| Successful cost | `$4.064125` |
| Failed cost | `$1.655000` |
| Total child cost | `$5.719125` |
| Configured high-risk budget | `$2.00` |
| Over budget | `$3.719125` / 185.96% |

Здесь `185.96%` — величина превышения относительно `$2`, а total составил `285.96%` исходного бюджета.

#### Model panel

- requested model;
- actual model;
- fallback chain;
- provider health;
- retry count;
- failed spend.

### 18.2. Consistency requirements

1. Dashboard показывает DB revision/last event ID.
2. При websocket reconnect выполняется snapshot reconciliation.
3. `/mazzy` возвращает только реально слушающий URL.
4. Active servers публикуются в local registry с PID, port, project root и owner session.
5. Если UI использует фильтр, label не должен называться total.
6. Stale snapshot заметно маркируется.

---

## 19. Предлагаемый backlog доработок Mazzy

### P0 — обязательно

| ID | Задача | Acceptance criteria | Ожидаемый эффект |
|---|---|---|---|
| MZ-P0-01 | Hard task-level budget | Run не стартует сверх maxCost без override | Предсказуемая стоимость |
| MZ-P0-02 | Provider failure checkpoint salvage | Late 403 сохраняет findings и продолжает fallback | Снижение failed cost |
| MZ-P0-03 | Evidence materialization | Test/review reports создают evidence rows | Автоматический acceptance |
| MZ-P0-04 | REVIEW→DONE gate | No blockers + required evidence закрывают задачу | Нет зависших completed tasks |
| MZ-P0-05 | Idempotent phase keys | Resume не повторяет неизменённую wave | Нет duplicate runs |
| MZ-P0-06 | Dashboard/DB reconciliation | Counts совпадают после refresh | Достоверный backlog |

### P1 — высокая важность

| ID | Задача | Acceptance criteria | Эффект |
|---|---|---|---|
| MZ-P1-01 | Actual model identity | UI показывает requested/actual/fallback | Честный audit trail |
| MZ-P1-02 | Child run graph bindings | Все child runs видны под task/workflow | Полная трассируемость |
| MZ-P1-03 | Delta context packets | Reviewer получает diff-scoped context | Снижение tokens |
| MZ-P1-04 | Max validation waves | Default ≤2 review waves | Anti-loop |
| MZ-P1-05 | Failed-cost breaker | 401/403 блокирует provider в task | Нет повторной потери бюджета |
| MZ-P1-06 | Artifact lineage | Intermediate report marked superseded | Нет путаницы 14/18/19 tests |
| MZ-P1-07 | Mutation contract validation | Writer всегда expected=true | Корректный effects audit |

### P2 — улучшения UX и аналитики

| ID | Задача | Acceptance criteria | Эффект |
|---|---|---|---|
| MZ-P2-01 | Active server registry | `/mazzy` находит живой owner/port | Удобный web access |
| MZ-P2-02 | Cost by phase/role | Dashboard строит breakdown | Оптимизация routing |
| MZ-P2-03 | Context efficiency warnings | Warning при input/output ratio > threshold | Раннее выявление bloated context |
| MZ-P2-04 | Review reuse by diff hash | Неизменённый diff переиспользует verdict | Быстрее validation |
| MZ-P2-05 | KPI dashboard | Cycle time, failed spend, closure latency | Управляемое улучшение |

---

## 20. Целевые KPI

| KPI | Наблюдалось | Цель | Порог предупреждения |
|---|---:|---:|---:|
| Task cycle time | ~55 мин | ≤30 мин | >40 мин |
| Total child runs | 12 | ≤6 | >7 |
| Successful read-only runs | 8 | 2–4 | >4 |
| Failed run share | 16.67% | <5% | >10% |
| Failed cost share | 28.94% | <5% | >10% |
| Total child cost | `$5.72` | `$1.8–$3.0` | >`$3.0` |
| Budget utilization | 285.96% от `$2` | ≤100% | >90% до final gate |
| Input/output tokens | 971 585 | ≤450 000 | >550 000 |
| Evidence completeness | 0 rows | 100% required evidence | <100% при REVIEW |
| Child traceability | 1 binding на 12 runs* | 100% visible graph | <95% |
| Final closure latency | задача осталась REVIEW | ≤2 мин после gate | >10 мин |
| Test growth | +8 | по рискам | нет жёсткого порога |
| Escaped blockers after final gate | 0 | 0 | >0 |

`*` Если binding намеренно workflow-level, KPI должен считать видимые child nodes, а не только строки `run_bindings`.

---

## 21. Рекомендуемые тесты для самого Mazzy

### 21.1. Budget tests

1. Task с `$1` budget не может породить runs на `$1.20` без override.
2. Failed provider cost входит в task budget.
3. Resume использует остаток исходного budget, а не новый полный budget.
4. Fallback reservation учитывается до запуска.

### 21.2. Provider failure tests

1. Provider падает 403 на первом turn — немедленный fallback.
2. Provider падает 403 после 20 turns — checkpoint импортируется.
3. Non-retryable 403 не повторяется на том же provider.
4. Partial findings доступны reviewer/fallback.

### 21.3. FSM/evidence tests

1. Comments не считаются evidence.
2. Structured acceptance report создаёт evidence rows.
3. Два no-blocker reviews + deterministic PASS переводят task в DONE.
4. Missing evidence оставляет REVIEW с понятной причиной.
5. Superseded test report не участвует в final gate.

### 21.4. Resume/idempotency tests

1. Resume с тем же diff не повторяет architecture.
2. Resume после writer повторяет только незавершённый review.
3. Изменение diff инвалидирует только зависимые review keys.
4. Один run не импортируется дважды.

### 21.5. Dashboard consistency tests

1. Snapshot total совпадает с SQLite total.
2. Websocket disconnect + reconnect выполняет reconciliation.
3. `/mazzy` не возвращает dead port.
4. Несколько session-owned servers корректно перечисляются.
5. Dashboard показывает source project root и DB revision.

---

## 22. План внедрения

### Этап 1: 1–2 дня

- определить точную семантику budget scope;
- добавить actual model/fallback metadata;
- исправить dashboard active server discovery;
- материализовать текущий VPN task evidence и закрыть его.

### Этап 2: 3–5 дней

- hard task budget;
- failed-cost circuit breaker;
- idempotent phase keys;
- max validation waves;
- artifact lineage.

### Этап 3: 5–8 дней

- child run graph в durable state;
- checkpoint salvage;
- delta context builder;
- automatic REVIEW→DONE acceptance gate.

### Этап 4: 2–4 дня

- cost/KPI dashboard;
- DB/web consistency monitoring;
- regression suite по сценариям раздела 21.

---

## 23. Критерии успешности доработки

Mazzy можно считать улучшенным для high-risk задач, если на трёх последовательных задачах выполняются все условия:

1. ни один task не превышает hard budget без explicit override;
2. failed cost share ниже 5%;
3. не более одной architecture wave;
4. не более двух validation waves;
5. все child runs видимы в task graph;
6. все required evidence materialized;
7. no-blocker task автоматически закрывается;
8. dashboard и DB counts совпадают;
9. requested и actual model отображаются раздельно;
10. качество результата не снижается: zero escaped high-severity defects после final gate.

---

## 24. Вывод

Эксперимент подтвердил, что концепция Mazzy Control Panel как durable control plane и `pi-subagents` как единственного child runtime жизнеспособна. Система обеспечила реальную инженерную ценность: независимые агенты обнаружили несколько safety-дефектов, которые не были пойманы первоначальной реализацией и тестами.

Основная проблема находится не в качестве reasoning агентов, а в операционной оболочке:

- budget не ограничивает task end-to-end;
- поздние provider failures не salvage-ятся;
- resume и validation недостаточно идемпотентны;
- большие context packets повторяются;
- durable evidence и child bindings неполны;
- FSM не завершает технически принятую задачу;
- dashboard может показывать stale или другой session-owned state.

Поэтому правильная стратегия — не уменьшать качество review, а сделать review **условным, delta-oriented, бюджетно ограниченным и полностью трассируемым**.

Целевой результат после доработок:

| Параметр | Сейчас | Цель |
|---|---:|---:|
| Качество | 9/10 | 9/10 или выше |
| Надёжность | 6/10 | 8.5/10 |
| Cost-efficiency | 4/10 | 8/10 |
| Web/control-plane UX | 5/10 | 8.5/10 |
| Общая оценка | 7/10 | 8.5–9/10 |
| Child cost | `$5.72` | `$1.8–$3.0` |
| Child runs | 12 | 4–6 |
| Cycle time | ~55 мин | 20–30 мин |

Итоговая рекомендация: сохранить multi-agent safety review, но немедленно внедрить P0 backlog — hard task budget, checkpoint salvage, idempotent phase keys, evidence materialization, automatic closure и dashboard/DB reconciliation.

---

## Приложение A. Файлы результатов

Основные artifacts исследования находятся в sibling project `RESEARCH/PI`:

- `.mazzy/outputs/pi-vpn-recovery-writer.md`
- `.mazzy/outputs/pi-vpn-recovery-fix-writer.md`
- `.mazzy/outputs/pi-vpn-recovery-validation-correctness.md`
- `.mazzy/outputs/pi-vpn-recovery-validation-doctor.md`
- `.mazzy/outputs/pi-vpn-recovery-focused-validation-correctness.md`
- `.mazzy/outputs/pi-vpn-recovery-focused-validation-doctor.md`
- `.mazzy/outputs/pi-vpn-recovery-final-validation-correctness.md`
- `.mazzy/outputs/pi-vpn-recovery-final-validation-doctor.md`
- `.mazzy/work/results/vpn-consilium-metrics.json`
- `.pi-ops/state.db`

## Приложение B. Итоговый безопасный VPN status

По завершении работ наблюдалось:

```text
AdGuard VPN: connected to London
Mode: TUN
Interface: tun0
Recovery mode: balanced
Doctor deep aggregate: PASS
Action: skip-provider-only
```

Это состояние было проверено без ручного destructive disconnect в финальной validation фазе.


---

# Часть II. Дополнительный консилиум: доказательная дельта

Ниже сохранён текст второй исследовательской волны. Он уточняет исторический и текущий статус тестов, benchmark-границы и новые control-plane defects, но не отменяет Часть I.

# Исследование Mazzy Control Panel и оркестрации VPN

**Дата:** 2026-08-16 · **Статус:** доказательный отчёт, без изменения исходного кода.  
**Объект:** задача VPN Recovery и связанный `@mazzy/control`. Идентификаторы capability, токены и локальные абсолютные пути намеренно не публикуются.

## Исполнительное резюме: дельта и текущая доказательная позиция

Последний сохранённый измерительный набор фиксирует: **`@mazzy/control`: 52 PASS за 6,65 с, но typecheck FAIL (exit 2, три ошибки storage-policy); VPN-плагин: 19 PASS и typecheck PASS; MAZZY_VPN: 105 PASS.** Поэтому утверждение «все контрольные гейты зелёные» недопустимо. Числа относятся к разным раннерам и не должны суммироваться как 176 независимых тестов.

После подготовки документа родитель повторил проверки в текущем dirty/untracked worktree: `npm test` дал **55/55 PASS за 6,06 с**, `npm run typecheck` — **PASS**. Это наблюдение сохранено в `parent-control-validation.log`, но не отменяет исходный измеренный FAIL: три storage-policy файла остаются untracked, а результат не привязан к принятому commit/diff digest. Следовательно, корректная хронология такова: **исходный набор — 52 PASS + typecheck FAIL (3 ошибки); текущий worktree — 55 PASS + typecheck PASS; интегрированный/release-гейт всё ещё не аттестован**.

Подтверждённая польза orchestration: VPN-плагин получил 19 тестов после обнаружения quorum, IPv6 и lease-дефектов; недеструктивная проверка осталась безопасной. Подтверждённые риски control plane: повторное использование устаревшего worker acceptance, неучёт generic FAIL при закрытии, drift ревизии RUNNING, описательный бюджет, неполные evidence/binding/lineage и межпроцессная доставка событий. Предложения ниже — минимальные безопасные срезы, **не реализация**.

| Гейт/артефакт | Результат | Граница вывода |
|---|---:|---|
| `@mazzy/control` tests | 52/52 PASS, 6,65 с | сохранённый запуск; не заменяет typecheck |
| `@mazzy/control` typecheck | FAIL, exit 2, 7,65 с | исходный артефакт: 3 ошибки `maxAgeSeconds: unknown` |
| `@mazzy/control`, parent rerun | 55/55 PASS, 6,06 с / typecheck PASS | текущий dirty/untracked worktree; не release provenance |
| VPN plugin tests/typecheck | 19/19 PASS, 1,30 с / PASS, 3,41 с | mock/локальная проверка, не live-VPN гарантия |
| MAZZY_VPN tests | 105/105 PASS, 279,15 с | shell/synthetic suite, не destructive staging |
| benchmark | PASS smoke-полей | единичный localhost, synthetic, последовательный |

## Область, метод и ограничения

Изучены сохранённые логи и JSON benchmark, metadata 12 child-runs, отчёты VPN, policy, SQLite/FSM и исходные seams `@mazzy/control`. Не выполнялись destructive reconnect, публикация, commit, изменение БД либо production-кода. Стоимость `$5.719125` — provider-reported child cost без parent/инфраструктуры; 39,96 agent-minutes — сумма длительностей, не wall-clock.

Факты VPN-задачи: 12 запусков (10 success, 2 поздних 403), input+output 971 585, cache-read 5 379 242 отдельно, известная стоимость `$5.719125`; `$1.655000` (28,94%) пришлись на неуспешные runs. Policy имеет `defaultUsd`, `highRiskUsd`, `maxChildrenPerWave`, но это не доказывает task-level hard cap.

## Исправленные и опровергнутые ранние утверждения

| Раннее утверждение | Исправление |
|---|---|
| Control typecheck «сейчас» падает | Сохранённый гейт падает; поздний untracked-worktree осмотр сообщает PASS. Ни одно из них не является доказательством интегрированного релиза. |
| «52 tests — текущий полный suite» | Это сохранённый набор; поздний осмотр сообщил 55. В отчёте фиксируется хронология, а не смешение. |
| Dashboard хранит stale snapshot в памяти | Опровергнуто: endpoint читает SQLite live. Подтверждён другой риск: два процесса и process-local SSE могут не уведомить браузер другого процесса. |
| Расхождение dashboard/DB доказывает одну причину | Не доказано: измерения не атомарны. Причина требует коррелированного snapshot/revision/event-ID опыта. |
| Report автоматически даёт PASS evidence и закрывает задачу | Опровергнуто: generic evidence не участвует в текущем closure-query; child-report не может быть авторитетным PASS. |
| `$2` был нарушенным hard task budget | Подтверждено лишь превышение reference-поля: scope/enforcement не реализованы в route/dispatch. |
| final «no blockers» сам закрывает задачу | Нет: нужны current reviewer-bound evidence и явное parent closure; политика required evidence пока неполна. |

## Benchmark: результаты и допустимое толкование

| Операция | n | p50, мс | p95, мс | max, мс | Условия |
|---|---:|---:|---:|---:|---|
| Store create | 500 | 1,362 | 2,523 | 20,723 | рост synthetic DB 0→499 задач |
| Store list | 200 | 2,348 | 3,545 | 6,194 | около 500 synthetic задач |
| Store detail | 200 | 0,295 | 0,522 | 1,067 | повтор одного synthetic task |
| HTTP snapshot | 100 | 4,846 | 8,405 | 28,728 | localhost, serial |
| Unauthorized snapshot | 50 | 0,583 | 0,925 | 1,309 | только missing-token rejection |

Это один запуск на localhost с project-local SQLite, последовательными `await`, без concurrency, холодного/тёплого разделения, повторов, raw samples, host/Node/SQLite manifest и привязки к source digest. Скрипт использует `floor(n×p)`: p50/p95 — его собственные order statistics, а не восстановимые стандартные перцентили. `max` — один наблюдённый outlier, **не SLA, timeout, throughput или худший гарантированный предел**. Все числа выше — не универсальные SLA.

## Матрица функционального покрытия

| Область | Имеющееся доказательство | Пробел/нужный тест |
|---|---|---|
| CRUD/FSM | sequential stale revision, transitions | two-store equal-revision CAS; stale acceptance в REVIEW |
| Closure/evidence | reviewer PASS/FAIL, comments isolation | current deterministic FAIL blocks DONE; required-kind omission |
| Run control | lifecycle tests | priority/no-op RUNNING update не блокирует PAUSE/STOP |
| Auth/privacy | Host, API auth, token not in URL | Origin matrix, malformed/64KiB/slow/aborted body, redacted errors |
| SSE | replay/reset/two clients | cross-process write, slow consumer/backpressure, snapshot-stream race |
| Routing/budget | policy parse/route | reservation ledger, task remainder, failed-cost accounting |
| Model/binding | one workflow binding | requested/actual/fallback attempt graph |
| Artifact state | отчёты существуют | digest, phase, current/superseded relation |
| Storage policy | сохранённые 52 runtime tests | type narrowing + accepted typecheck/provenance |
| VPN recovery | 19 tests quorum/IPv6/lease | deep/boundary, malformed locks, real staging only by approval |
| MAZZY_VPN | 105 сценариев | явная synthetic граница; staging VM отдельно |

## Решения и гипотезы

| Решение | Статус | Основание |
|---|---|---|
| Не называть benchmark производительностью/SLA | принято | метод serial/synthetic/one-run |
| Сохранить typecheck FAIL в принятом состоянии | принято | измерительный log exit 2 |
| Нормализовать `maxAgeSeconds` один раз после `isRecord` | рекомендовано P0 | наименьшее безопасное narrowing; без реализации |
| Автоматически материализовать child report как PASS | отклонено | current generic FAIL не блокирует DONE |
| Budget enforcement внутри dashboard scheduler | отложено/неверный владелец | child dispatch/billing принадлежат parent/runtime |
| Причина dashboard-count mismatch | гипотеза | two-process/SSE подтверждены, атомарного сравнения нет |
| Stage-B lost update | подтверждён source-level риск, не воспроизведён барьером | read-before-transaction и отсутствие проверки affected rows |

## Целевой порядок работ

Приоритеты, ownership и acceptance-гейты приведены в issue log. Сначала нужны integrity/CAS/evidence-контракты; затем budget, попытки и lineage; затем owner registry/SSE/migration. Все schema-изменения — additive nullable, с backup/`quick_check`, без удаления legacy полей в первом релизе.

## Итог

VPN-плагин имеет полезный зелёный локальный тестовый результат. Для Control Panel исходный compiler gate красный, а текущий parent-rerun зелёный только в dirty/untracked worktree и не имеет release provenance. Следующая безопасная работа — не расширение orchestration, а узкое исправление P0 по issue log с воспроизводимыми тестами и новым, привязанным к ревизии измерением.


---

# Часть III. Полный журнал расследования и проблем

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
