# Раунд 11 аудита — livetest транзакция с откатом (Фаза 2, C2-4)

- Дата: 2026-08
- Область: `core/livetest`.
- Метод: adversarial проверка транзакционных инвариантов и rollback-семантики.

## Назначение

Транзакционный live-тест профиля с гарантированным откатом (паритет cmd_test /
save_test_backup / restore_test_backup): snapshot → up под deadline → verify →
commit ИЛИ rollback. Система ВСЕГДА остаётся в зафиксированном состоянии.

## Проверенные транзакционные инварианты

| Сценарий | Ожидание | Тест |
|---|---|---|
| verify PASS | commit кандидата, backup отброшен, teardown НЕ вызывается | `TestPassCommitsCandidate` |
| verify FAIL | teardown кандидата + restore previous (+ up если был up) | `TestVerifyFailureRollsBackToPrevious` |
| candidate up FAIL | rollback к previous | `TestCandidateUpFailureRollsBack` |
| нет previous | teardown, НЕ фабрикуется фейковое состояние | `TestNoPreviousLeavesCleanDown` |
| timeout | rollback на ЖИВОМ parent-контексте | `TestTimeoutRollsBackWithLiveParentContext` |

## Ключевые тонкости (self-audit)

### T3 — P1-класс — rollback на живом контексте (не на истёкшем)
Кандидат поднимается под `cctx` (WithTimeout). При срабатывании deadline
teardown и restore ОБЯЗАНЫ выполняться на РОДИТЕЛЬСКОМ `ctx`, иначе они были бы
мгновенно отменены истёкшим `cctx` — и система осталась бы с полуподнятым
кандидатом. Код уже использует `ctx` в `rollback`; зафиксировано тестом
`TestTimeoutRollsBackWithLiveParentContext` (кандидат блокируется до deadline,
проверяется, что teardown+restore реально произошли).

### T1 — commit-fail не оставляет неоднозначности
Если запись committed-состояния падает, выполняется полный rollback (down
кандидата + restore previous), а не «состояние previous при поднятом
кандидате».

### T2 — timeout флаг на правильном контексте
`deadlineHit(cctx)` проверяет именно дочерний контекст с deadline.

## Дизайн

Connector/Verifier/Store инъектируются → транзакционная логика детерминирована
и тестируется без сети. В проде Connector — обёртка над `core/connect`,
Verifier — над `core/probe`.

Отличие от bash: не фабрикуем «none»-профиль при отсутствии previous (bash
писал явный clean-state; наш вариант оставляет кандидата снятым и не пишет
битую запись, которая не прошла бы валидацию state).

## Проверки

- `core/livetest` — 6 тестов PASS, `-race` чист.
- Полный прогон: 13 пакетов, `go test -race ./...` — PASS.
- Статическая сборка сохранена.

Дефектов не найдено; тонкость T3 подтверждена корректной и покрыта тестом.
