# Раунд 25 — полное повторное ревью: исправление N1–N6 + новые баги

- Дата: 2026-08. Ветка: `feat/go-rewrite`.
- Запрос: исправить баги и сделать полное повторное ревью.
- Метод: исправил backlog-заметки (N1, N2) + провёл второй, более глубокий
  проход по не покрытым ранее модулям (routes/guard/recover/control/bootrecovery)
  → нашёл и исправил 4 новых дефекта.

## Исправлено из прошлого раунда

| ID | Что | Тест |
|---|---|---|
| **N1** | `connect`: `Down` и `unwind` сведены к единой `teardown()` — не могут разойтись | idempotency + flag-clear |
| **N2** | `profile`: приём + проброс AmneziaWG 1.5 полей (`header_protection_key`, `content_padding_addition`) вместо reject | parse + UAPI passthrough |

## Новые баги, найденные повторным ревью

| ID | Severity | Баг | Исправление | Тест |
|---|---|---|---|---|
| **N3** | средний | `routes.addEndpointRoute`: при hostname-endpoint (пропуск) всё равно писался `appliedEndpoint`, teardown пытался удалить несуществующий маршрут | функция возвращает `installed bool`; запись только при реальной установке | no spurious `route del` |
| **N4** | низкий | `guard.RemoveConnmark` дублировал raw-вызов вместо общего `deleteTableErr` | использует общий хелпер | — |
| **N5** | **высокий** | `recover`/`disconnect` НЕ чистили `mazzy_vpn_connmark` (добавлен в C1-4a2) — таблица утекала после «панической кнопки» | чистятся все 3 таблицы через `guard.*` константы; fwmark через `routes.DefaultMark` | — |
| **N6** | низкий | `control pair` писал `trust.json` неатомарным `os.WriteFile` — краш мог оставить обрезанный файл | `writeJSONAtomic` (temp+rename, 0600) для trust.json И identity.json | live re-test |
| **DRY** | стиль | хардкод `{vpnaw0,vpnwg0}` в 2 местах мог разойтись с `Protocol.Interface()` (и оба пропускали `vpnovpn0`) | `core.ManagedInterfaces()` — единый источник | покрытие всех протоколов |

## Проверено чистым (второй проход)

- **bootrecovery.Read()**: hardened — отвергает симлинки (anti-symlink-attack),
  не-regular файлы, неизвестные состояния → всё → `Unknown` (блокирует мутации).
  Fail-closed по дизайну.
- **measure.RankBest**: race-safe (каждая горутина пишет свой `results[idx]`,
  sem-пул, done-канал), корректная сортировка ICMP-alive → reachable → latency.
- **state.Write**: durable (chmod→write→fsync→close→rename→fsync dir + cleanup).
- **lock**: `flock(LOCK_EX|LOCK_NB)`, single-writer гарантия для state.
- **livecheck.egress**: bounded read, `net.ParseIP` валидирует контент.
- **recover** намеренно lock-free (панич-кнопка должна работать при зависшем lock).
- **profile.parseFwMark**: hex/decimal + 32-bit bound (overflow отлавливается).
- **endpoint.EndpointHost**: корректный разбор `[IPv6]:port` и `host:port`.
- **mimicry**: timezone↔country 1:1, обратный маппинг без коллизий.

## Инструментальный гейт (core + cli)

```
core:  build OK | tests 35ok/0fail | race 35ok | vet 0 | sc 0 | fmt 0 | vuln 0
cli:   build OK | vet 0 | sc 0 | vuln 0
public-audit OK | installer-autonomy OK
```

## Итог

Повторное ревью нашло **4 новых дефекта** (1 высокий — утечка connmark-таблицы
после recover) + завершило 2 backlog-заметки + DRY-рефактор имён интерфейсов.
Всего за две сессии аудита: **7 исправленных багов**, 0 уязвимостей, 0 lint.
Модули bootrecovery/measure/state/lock подтверждены как hardened.
