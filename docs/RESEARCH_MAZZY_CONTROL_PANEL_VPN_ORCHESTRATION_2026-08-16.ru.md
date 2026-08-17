# Исследование Mazzy Control Panel и orchestration на задаче VPN Recovery

**Дата исследования:** 2026-08-16  
**Статус документа:** итоговый расширенный технический отчёт  
**Исследуемая задача:** `17f6f71d-4d54-438a-91e7-5ecd5d6fa5f3`  
**Название задачи:** `Improve pi-vpn-recovery transport-health checks and doctor surface`  
**Control plane:** Mazzy Control Panel  
**Child runtime:** `pi-subagents`  
**Репозиторий реализации:** `/home/mazurov/RESEARCH/PI/MY_PLUGINS/pi-vpn-recovery`  

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
