# Раунд 7 аудита — health / auto-reconnect + probe (Фаза 2, C2-2)

- Дата: 2026-08
- Область: `core/health`, `core/probe`.
- Метод: adversarial проверка против ключевых уроков предыдущего консилиума
  (`MAZZY_CONTROL_PANEL_AUDIT_2026-08-16`).

## Проверенные уроки прошлого аудита

### Урок «один сбой не должен вызывать destructive reconnect» — СОБЛЮДЁН
`core/health` считает ТОЛЬКО последовательные сбои; лимит по умолчанию 2.
Любой healthy-тик сбрасывает счётчик и флаг восстановления. Зафиксировано
тестом `TestSingleFailureDoesNotRecover`.

### Урок «false PASS без рабочего интернета» — СОБЛЮДЁН
`core/probe` при заданном интерфейсе биндит исходящий сокет на IPv4 туннеля
(`dialer.LocalAddr`), форсируя egress через тоннель, а для full-tunnel
дополнительно требует `default egress == tunnel egress`. Ответ валидируется как
IP (`TestInternetOKRejectsNonIP`). Недоступность → провал
(`TestInternetOKFailsWhenUnreachable`).

### Урок «recovery ровно один раз» — СОБЛЮДЁН
`recoveredAtLimit` гарантирует один destructive reconnect на серию; последующие
сбои не рестартят повторно (`TestRecoversExactlyOnceAtLimit`).

## Найденные тонкости (H2) — корректно

**H2 — grace между двумя сбоями.** Startup-grace-тик между двумя реальными
сбоями НЕ обнуляет и НЕ увеличивает счётчик — два сбоя остаются
последовательными. Это точный паритет с bash (ранний `return` на grace без
`health_reset`). Зафиксировано `TestGraceBetweenFailuresDoesNotResetStreak`.

Новых дефектов не найдено.

## Архитектурное решение

`core/health` не знает о сети: он зависит только от интерфейсов `Probe` и
`Recoverer`. Реальные сетевые проверки — в `core/probe`. Это даёт:
- детерминированные, быстрые unit-тесты health-логики без сети;
- одну точку правды для «интернет через тоннель» (`NetProbe`).

## Паритет

| bash | Go | тест |
|---|---|---|
| health.failures счётчик, лимит 2 | `Monitor.failures` / `Config.limit()` | `TestDefaultLimitIsTwo` |
| health_reset при healthy | `reset()` | `TestHealthyResetsCounter` |
| recover при достижении лимита | `fail()` at limit | `TestRecoversExactlyOnceAtLimit` |
| startup grace (15..180s, деф.60) | `InStartupGrace` clock | `TestStartupGraceClock` |
| health_internet_ok + default egress | `NetProbe.InternetOK` | `TestInternetOK*` |
| connection_local_ok (link exists) | `NetProbe.LinkPresent` | `TestLinkPresent*` |

## Проверки

- `go test -race ./...` — 9 пакетов PASS.
- Статическая сборка сохранена (net/http не тянет cgo при CGO_ENABLED=0).
