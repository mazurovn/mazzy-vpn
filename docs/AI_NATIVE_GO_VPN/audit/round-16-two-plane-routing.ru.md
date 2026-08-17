# Раунд 16 аудита — AI-native двухплоскостная маршрутизация (Фаза 4 aванс)

- Дата: 2026-08
- Область: `core/provider` (Плоскость 2), `core/control` (Плоскость 1).
- Метод: adversarial проверка security-логики обеих плоскостей + quality gate.
- Контекст: заказчик подчеркнул проработать роутинг в двух плоскостях —
  (1) подключение к агентам извне, (2) агенты→провайдеры/LLM обходя блокировки.

## Реализация

- `core/control.Plane` — Плоскость 1: реестр участников (agent/harness/user/
  app/peer) + направленные deny-by-default маршруты.
- `core/provider.CheckRegion` — Плоскость 2: region-readiness verdict, чтобы
  агент не нарвался на гео-блок/челлендж LLM-провайдера.

## Проверенные security-инварианты

### AI-1 — control deny-by-default — ПОДТВЕРЖДЕНО
`CanReach` возвращает `true` ТОЛЬКО при явном `Allow`. Нет ни одного
default-allow пути (проверено grep + `TestDenyByDefault`). Маршруты
направленные: `Allow(h,a)` не даёт `a→h`.

### AI-2 — provider Ready консервативен — ПОДТВЕРЖДЕНО
`Ready` требует ВСЕ: `Supported && Consistent && len(mismatches)==0`. Одного
несоответствия достаточно для `NotReady`. Это правильная сторона ошибки:
лучше сказать агенту «не готово», чем дать нарваться на блок.

### AI-3 — target-mismatch блокирует даже при поддержке — ПОДТВЕРЖДЕНО
Если egress-страна поддержана, но не равна target — добавляется
`region.target.egress-mismatch`, и verdict падает в NotReady.

### AI-4 — таймзона-mismatch = блокер — ПОДТВЕРЖДЕНО
Согласованность egress-страны с системной таймзоной обязательна (классический
триггер VPN-детекта). `TestRegionTimezoneMismatchIsBlocker`.

### AI-5 — Deregister не оставляет висячих прав — ПОДТВЕРЖДЕНО
Удаление участника чистит маршруты и к нему, и от него
(`TestDeregisterRemovesRoutes`).

## Quality gate (запрос: dead code / hardcode / spaghetti / логика / ущербность)

| Проверка | Результат |
|---|---|
| go vet | чисто |
| staticcheck | чисто |
| gofmt | отформатировано |
| go test -race | 19 пакетов PASS (control race-clean) |
| coverage | control 100%, provider 93.6% |
| хардкод URL/IP в логике | НЕТ (endpoints — данные Provider, не хардкод) |
| спагетти | НЕТ (CheckRegion линеен, control — простые map-операции) |
| пустые функции | НЕТ |

## Интеграция обеих плоскостей

`TestScenarioAINativeTwoPlanes`: harness→agent (Плоскость 1, deny→allow) +
egress region-check для OpenAI (Плоскость 2, DE=ready, RU=blocked). Это сквозная
демонстрация «AI-native VPN»: подключить агента извне и надёжно свести его с
LLM, минуя блокировки.

## Дальше

- Плоскость 2: живой региональный probe (egress-страна из реального geo) —
  обёртка над `core/verify`+geo, задача L4-2.
- Плоскость 1: привязка `control.Participant.Endpoint` к реальному data-plane
  адресу туннеля, задача L4-3.
