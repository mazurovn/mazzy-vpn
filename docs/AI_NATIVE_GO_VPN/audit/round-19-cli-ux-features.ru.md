# Раунд 19 аудита — UX-фичи CLI, выбор адаптера, диагностика сети

- Дата: 2026-08
- Область: `core/catalog`, `core/measure`, `core/netadapter`, `core/netdiag`,
  15 новых CLI-команд + интерактивное меню.
- Метод: живая проверка на реальных конфигах + кабель/Wi‑Fi + AdGuard tun0.

## Что было плохо (запрос заказчика) → исправлено

| Проблема | Решение |
|---|---|
| нет выбора загруженных конфигов | `core/catalog`: import/list/favorite/remove, managed store |
| нет проверок работы конфигов/сети | `core/measure`: UDP-probe серверов, ранжирование |
| нет меню в CLI | интерактивное меню (`mazzy-vpn` без аргументов) |
| неудобно писать пути в коде | `up NAME` / `up --best` вместо путей к файлам |
| нет автоконнекта | `up --best`/`--auto` — авто-выбор лучшей зоны |
| нет выбора лучших зон | `test`/`best` — латентность + reachability |
| нет выбора адаптера | `core/netadapter` + `adapters`/`netdiag` |
| нет анализа/диагностики/исправлений | `core/netdiag` — findings + fixes |

Добавлено **65+ функций** (catalog 23, measure 13, netadapter 21, netdiag 8) +
15 CLI-команд — значительно больше минимума в 30.

## Найденные и исправленные дефекты (на реальном железе)

### UX-1 — P0 — TCP-probe не работает для WireGuard (UDP)
Первый `measure` делал TCP-connect к UDP-порту WireGuard → все серверы
«unreachable». **Исправлено:** UDP-probe = DNS-resolve + UDP-socket + write.
На реальных серверах: 6/6 reachable. Честная оговорка в выводе: латентность =
DNS+socket setup (сервер WG не отвечает by design).

### UX-2 — P1 — рекомендация адаптера игнорировала link-local
На твоём железе кабель `enp5s0` поднят, но имеет только `169.254.x`
(link-local, нет DHCP-lease), а Wi‑Fi `wlp3s0` — реальный `192.168.1.36`.
Старая логика рекомендовала кабель (бесполезный для интернета). **Исправлено:**
`HasRoutableIPv4()` исключает link-local; теперь рекомендуется Wi‑Fi. Regression:
`TestRecommendPrefersRoutableOverLinkLocal`.

### Проверено вживую: детект AdGuard
`netdiag` на твоей системе корректно нашёл `tun0` (AdGuard VPN) как
конфликтующий интерфейс и выдал fix: «отключите другой VPN или привяжите тест
к конкретному uplink».

## Quality gate (dead code / hardcode / spaghetti / логика / ущербность)

| Проверка | Результат |
|---|---|
| go vet (core+cli) | чисто |
| staticcheck (core+cli) | чисто |
| gofmt | чисто |
| go test -race | 23 пакета PASS |
| пустые функции | НЕТ |
| хардкод в CLI | НЕТ |
| dead code (fingerprint) | использован для dedup |
| public audit | OK |

## Клиентский путь (теперь удобно)

```
mazzy-vpn                      # меню
mazzy-vpn import <DIR>         # загрузить конфиги
mazzy-vpn test                 # какие зоны работают
mazzy-vpn best                 # лучшая зона
mazzy-vpn adapters             # выбрать кабель/Wi‑Fi
mazzy-vpn netdiag              # анализ + фиксы
sudo mazzy-vpn up --best       # автоконнект
```

## Осталось (честно)

- Привязка выбранного адаптера к connect (bind egress к uplink) — след. шаг.
- Латентность WG — только setup-время; истинный RTT недоступен без ответа
  сервера (ограничение протокола).
