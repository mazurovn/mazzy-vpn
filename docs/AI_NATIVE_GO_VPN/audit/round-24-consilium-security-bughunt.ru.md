# Раунд 24 — консилиум-аудит: безопасность, баг-хант, код-ревью, логика

- Дата: 2026-08. Ветка: `feat/go-rewrite`.
- Запрос: детальный аудит, баг-хант, код-ревью, поиск логических проблем,
  спагетти-кода, хардкода, проблемных функций — «консилиумом».
- Честная оговорка: реальный `pi-subagents` runtime недоступен (в соседнем
  PI-проекте). Провёл **ролевой консилиум сам** (безопасность / конкурентность
  / сеть-корректность / надёжность / стиль) + полный набор инструментов.

## Инструментальный слой (объективные проверки)

| Инструмент | core | cli |
|---|---|---|
| `go build` (CGO=0, static) | ✅ | ✅ |
| `go test ./...` | 34 ok / 0 fail | ✅ |
| `go test -race ./...` | 34 ok | ✅ |
| `go vet` | 0 | 0 |
| `staticcheck` | 0 | 0 |
| `gofmt -l` | 0 | 0 |
| **`govulncheck`** | **0** (было 16) | **0** |

## Роль «Безопасность» — НАЙДЕНО и ИСПРАВЛЕНО

**S1 (высокий): 16 уязвимостей stdlib + 1 в x/net.**
`govulncheck` показал уязвимости в `crypto/tls`, `crypto/x509`, `net/http`,
`net/url`, `encoding/asn1`, `os`, `net` (go1.25.5), достижимые через HTTP-egress
в `livecheck`, плюс `golang.org/x/net` idna/http2.
Исправление: toolchain → **go1.25.13**, `x/net` → **v0.58.0**, re-vendor,
`govulncheck` добавлен в CI. Итог: **0 уязвимостей**.

**Проверено чистым:**
- Нет хардкода IP-адресов в коде (кроме `0.0.0.0/0`, `127.0.0.1` по делу).
- Нет секретов/PII (публичный аудит + git-история чисты).
- Приватный ключ identity хранится `0600`, atomic temp+rename.
- Anti-impersonation в control-plane (VerifyID) — покрыт тестами.

## Роль «Конкурентность» — чисто + 1 стиль-фикс

- `-race` зелёный на всех 34 пакетах.
- Горутины в `stealth_cmd`/`connect` ограничены `WaitGroup`+`mutex`+context —
  утечек нет, спиннер-горутина закрывается через `spinDone`.
- **B1 (низкий, исправлено): shadowing.** В `daemon.go` ветка stealth-монитора
  объявляла `sig := gatherStealthSignal(ctx)`, перекрывая канал ОС-сигналов
  `sig`. Ныне безопасно (scope case), но убрано → `stSig`.

## Роль «Сеть-корректность» — чисто

- **fwmark == routing table** (инвариант G1): один и тот же номер в UAPI
  `fwmark=` и в `ip rule not fwmark <t> table <t>` — проверено в
  `uapi.go`/`routes.go`, задокументировано.
- **UAPI base64→hex** ключей: корректно (32 байта, lowercase hex), покрыто.
- **Парсер `.conf`**: неизвестные ключи отвергаются (fail-loud). Все 10 реальных
  конфигов используют только поддерживаемые поля (s1-s4, h1-h4, i1, jc/jmin/jmax).
- Teardown `connect.Down`/`unwind` — симметричный reverse-order, fail-safe
  (IPv6 guard mismatch дропает IPv6, никогда не течёт).

## Роль «Надёжность» — чисто + 1 фикс

- `state.Write`: durable — chmod→write→fsync file→close→rename→fsync dir,
  cleanup на каждой ошибке. Production-grade.
- `lock`: `flock(LOCK_EX|LOCK_NB)` — без гонок, без stale-PID.
- daemon failover: самолечится (`Protected()` сбрасывает fails), reconnect +
  failover-лимиты корректны.
- **B2 (низкий, исправлено): stale reason.** В `connect.go` авто-reconnect
  слал `snap.Reason` (первый снимок) вместо текущего `s.Reason`.

## Роль «Стиль/спагетти/хардкод» — здоровая база

- Файлы компактны: макс 408 строк (tui.go), core-файлы ≤270.
- Функции: 2 крупных (`cmdConnect` 148, `cmdDaemon` 147) — просмотрены построчно,
  логика корректна; дробить не требуется (линейные, читаемые).
- **Хардкод внешних сервисов** собран в каталогах, не разбросан:
  `aiProviders` (12 шт) в `providers.go`, `DefaultProbeURL` в `livecheck`,
  cloudflare/ipify — именованные константы. Приемлемо.
- Нет `TODO/FIXME/HACK/XXX` в рабочем коде. `panic` только в helper-скрипте.
  `os.Exit` только в `cmd/*/main`. Все `fmt.Errorf` используют `%w`.
- Игнор ошибок (`_ =`) — только в teardown/best-effort путях (осознанно).

## Замечания без исправления (backlog)

| ID | Наблюдение | Приоритет |
|---|---|---|
| N1 | `connect.Down` и `unwind` — почти дубли; можно свести к одной reverse-цепочке | P3 (косметика) |
| N2 | amneziawg-go v3 поддерживает новые поля (`content_padding_addition`, `header_protection_key`) — наш парсер их отвергнет; реальные конфиги их не используют | P2 (forward-compat) |

## Итог

Найдено **3 исправленных дефекта** (1 высокий security, 2 низких логических) +
2 backlog-заметки. Кодовая база **зрелая**: durable I/O, race-free,
fail-safe teardown, 0 уязвимостей, 0 lint/vet/staticcheck, богатое покрытие
тестами. Хардкод дисциплинирован (каталоги), спагетти нет.
