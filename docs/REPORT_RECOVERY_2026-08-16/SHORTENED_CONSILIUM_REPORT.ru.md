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
