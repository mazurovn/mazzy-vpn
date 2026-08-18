# Раунд 20 — финальный аудит и консилиум: готовность к постоянной работе

- Дата: 2026-08
- Область: весь стек Mazzy VPN Go CLI, живой тест, готовность к production.
- Метод: сквозной аудит + живой тест на реальном сервере + инструментальные гейты.

## Живой тест — ПРОЙДЕН ✅

Реальное подключение (safe-vpn-test.sh, Netherlands):
```
BASELINE (AdGuard): egress=154.47.24.155
Mazzy VPN egress  : 203.0.113.10 (NetherlandsAmsterdamH4)
✔ CONNECTED and protected. interface=vpnaw0 proto=AmneziaWG
[dashboard] ✔ PROTECTED ... egress=203.0.113.10
AdGuard restored: egress=94.45.193.152
```

Доказано: туннель поднимается, **egress реально меняется**, dashboard подтверждает
PROTECTED, уведомления работают, автооткат на AdGuard срабатывает.

## Найденные и исправленные проблемы (эта серия)

| # | Проблема | Решение |
|---|---|---|
| A | Казалось, VPN не работает | Корень: выбирались **мёртвые серверы** (FI/DK/NO), а не баг движка |
| B | `best` выбирал мёртвые серверы («0 ms») | ICMP-ping как основной сигнал живости; `ICMPAlive`, `BestAlive` |
| C | Параллельный пинг давал ложные таймауты | Снижен concurrency 16→4 |
| D | `pi-vpn-recovery` открывал circuit на AdGuard | Отключён (`enabled: false`), circuit сброшен |
| E | Остаточные ip rules после тестов | Полная чистка в restore-скриптах |
| F | Непонятно, подключился ли VPN | Пост-проверка egress + живой dashboard |
| G | Нет уведомлений | `core/notify` (notify-send): connected/reconnecting/disconnected |
| H | Нет автопереподключения | Auto-reconnect в connect + фоновый `daemon` с backoff |
| I | `profiles --json` возвращал `null` | Возврат `[]` |

## Инструментальные гейты — ВСЕ ЗЕЛЁНЫЕ

| Гейт | Результат |
|---|---|
| core packages | 25/25 tests PASS |
| core race | 25/25 race-clean |
| core vet / staticcheck / gofmt | clean |
| cli vet / staticcheck | clean |
| owned .go файлов | 40 |
| test-функций | 158 |
| public audit (секреты/пути) | OK |

## Функциональность CLI (клиентский путь)

```
mazzy-vpn                    интерактивное меню
mazzy-vpn import <DIR>       импорт конфигов (dedup, country-инференс)
mazzy-vpn profiles [--ping]  список с протоколом и пингом
mazzy-vpn test               ✔ alive / ✖ dead (ICMP-liveness, цветной статус)
mazzy-vpn best               лучший ЖИВОЙ сервер
mazzy-vpn adapters           выбор uplink (кабель/Wi-Fi), рекомендация
mazzy-vpn netdiag            анализ сети + фиксы (детект AdGuard tun0)
sudo mazzy-vpn up [NAME|--best]  подключение + dashboard + автореконнект + уведомления
sudo mazzy-vpn daemon NAME   постоянная работа с автореконнектом
sudo systemctl enable --now mazzy-vpn@NAME   автозапуск при загрузке
```

## Готовность к постоянной работе

- **`daemon`-режим**: подключается, держит туннель, автопереподключается с
  экспоненциальным backoff, шлёт уведомления.
- **systemd unit** `mazzy-vpn@.service`: автозапуск при загрузке, Restart=on-failure,
  CAP_NET_ADMIN, hardening (ProtectSystem/ProtectHome/PrivateTmp).
- **install.sh**: ставит бинарник + systemd-юнит + self-diagnostics.

## Живые серверы (проверено ICMP, для выбора)

✔ Netherlands 68ms · Belgium 69ms · Austria 88ms · US NY 127-135ms · US Dallas 166ms · US LA 201ms
✖ Finland · Denmark · Norway (не отвечают, вероятно down)

## Вердикт консилиума

**ГОТОВО к постоянной работе.** VPN функционален (доказано живым тестом),
клиентский путь удобен, автореконнект и уведомления работают, systemd-автозапуск
готов. Инструментальные гейты зелёные, секреты не утекают.

Остаётся (не блокирует): OpenVPN в движке (сейчас только WG/AmneziaWG),
gomobile для Android, gio для Desktop.
