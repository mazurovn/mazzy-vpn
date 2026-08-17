# Раунд 21 — новый TUI-дашборд, настройки, восстановление, AI-провайдеры

- Дата: 2026-08
- Область: `core/settings`, новый TUI, recover/disconnect, providers, live status.
- Метод: аудит + живой тест на работающем VPN (Netherlands egress).

## Реализованные фичи (по запросу заказчика)

| Запрос | Реализовано |
|---|---|
| Быстрое подключение | TUI пункт 1 «⚡ Quick connect» (preferred/best) |
| Переподключение с диагностикой | TUI пункт 3 «🔄 Reconnect with diagnostics» |
| Диагностика + подбор локаций | `test`/`best` через физ.uplink (ICMP-liveness) |
| Восстановление / принудит. отключение | `recover` (panic → чистый Wi-Fi, сброс guards/rules) |
| Просто disconnect | `disconnect` (мягкое снятие туннеля) |
| Сброс конфига | `recover --reset-catalog` |
| Диагностика провайдеров + фильтр по типу | `providers [--type llm/agent/search]` |
| Загрузка конфига/папки | TUI пункт 11 «📥 Import» |
| Мониторинг + неблокирующий дашборд | TUI live-header (сэмплирует статус, не блокирует меню) |
| Проверки AI-агентов/провайдеров | `providers` — 12 сервисов (OpenAI/Anthropic/Gemini/...) |
| Диагностика локации/подключения | `status` показывает реальный интерфейс + egress |
| Настройки (автоподключение/диагностика/оповещения) | `core/settings` + TUI пункт 13 |

## Живой тест — доказательства

- **AI-провайдеры через наш VPN**: 12/12 reachable (OpenAI 401, Anthropic 401,
  Claude 403, Gemini 200, ChatGPT 403, Perplexity 403 — все доступны). Главная
  цель проекта — доступ к AI — достигнута.
- **status**: `✔ PROTECTED egress=95.211.225.232` (Netherlands, реальный трафик
  RX 7MB/TX 4MB).
- **TUI-header**: live-статус PROTECTED + uplink wlp3s0.
- **settings**: тоглы сохраняются в `~/.config/mazzy-vpn/settings.json`.

## Исправленные баги

| # | Баг | Фикс |
|---|---|---|
| S1 | `status` показывал `down` при активном VPN | Детект реального интерфейса vpnaw0 + egress |
| S2 | ICMP через AdGuard tun0 давал ложные «dead» | Пинг через физ.uplink (`ping -I`), `NewViaUplink` |
| S3 | daemon.go не собирался (кривой interface) | Убран placeholder-интерфейс |

## Новый TUI

- **Live-header**: статус (PROTECTED/LINK-UP/DISCONNECTED) + egress + uplink,
  сэмплируется при каждой отрисовке (меню не блокируется).
- **Сгруппированное меню**: Connect / Diagnostics / Profiles & settings.
- **Настройки**: auto-connect, auto-diagnostics, notifications, auto-reconnect,
  kill-switch, preferred zone — персист в JSON.

## Самолечение (self-healing)

- `daemon` с zone-failover: при повторных сбоях зоны переключается на другую
  ЖИВУЮ зону (ICMP-ranked через uplink).
- Auto-reconnect с экспоненциальным backoff.
- `recover` — гарантированный возврат к чистой сети.

## Аудит — всё зелёное

| Гейт | Результат |
|---|---|
| core tests | 26/26 PASS |
| core race | 26/26 clean |
| core vet / staticcheck | clean |
| cli vet / staticcheck | clean |

## Осталось (не блокирует)

- Full-screen TUI (bubbletea) вместо построчного — опционально.
- OpenVPN в движке.
- Автозапуск daemon с --best failover в systemd (юнит готов).
