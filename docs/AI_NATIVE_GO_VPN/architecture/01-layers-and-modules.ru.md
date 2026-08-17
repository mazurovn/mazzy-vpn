# Архитектура: слои и модули

Статус: DRAFT для аудита. Дата: 2026-08.

## 1. Двухслойная модель (маппинг на код)

```
┌─────────────────────────────────────────────────────────────┐
│ Слой 1 — CONTROL PLANE (оркестрация агентов/подключений)     │
│                                                              │
│  agents ⇄ harnesses ⇄ users ⇄ apps ⇄ peers ⇄ Android        │
│                                                              │
│  core/control/   — реестр участников, маршруты «кто с кем»   │
│  core/api/       — machine-first JSON API (для агентов)      │
└───────────────────────────┬─────────────────────────────────┘
                            │ намерения (intents)
┌───────────────────────────▼─────────────────────────────────┐
│ Слой 2 — SECURE DATA PLANE (транспорт и защита)              │
│                                                              │
│  core/engine/    — mazzy-core: WG/AmneziaWG/OpenVPN/L2TP     │
│  core/health/    — no-drop, auto-reconnect, self-repair      │
│  core/doctor/    — диагностика                               │
│  core/guard/     — fail-closed nftables + IPv6 leak guard    │
│  core/routes/dns/tun/ — автономная замена *-quick            │
│  core/provider/  — провайдеры/точки выхода                   │
│  core/routing/   — policy routing, split-tunnel              │
└─────────────────────────────────────────────────────────────┘
```

## 2. Карта модулей `core/` (общее ядро)

| Модуль | Ответственность | Замещает в bash |
|---|---|---|
| `engine/wireguard` | vendored amneziawg-go как библиотека; up/down | `run_quick_service`, `awg-quick` |
| `engine/openvpn` | запуск OpenVPN (vendored статик, этап 1) | `run_openvpn_service` |
| `engine/l2tp` | L2TP через NM (где неизбежно) | `run_l2tp_service` |
| `tun` | создание/настройка TUN-интерфейса | часть `awg-quick`/`wg-quick` |
| `routes` | таблицы маршрутов, fwmark, ip rule | `transition_route_interface` и др. |
| `dns` | применение DNS (resolvectl/resolv.conf) | `openvpn_dns_up/down` |
| `guard` | fail-closed nftables + IPv6 leak guard | `transition_guard_*`, `ipv6_guard_*` |
| `state` | персистентное намерение (DESIRED/MODE) | `write_state`/`state_get` |
| `lock` | single-flight mutation lock / lease | `acquire_action_lock`, flock |
| `bootrecovery` | boot-recovery gate | `boot_recovery_*` |
| `connect` | транзакция подключения + откат | `cmd_connect`, `transition_restore_previous` |
| `health` | health-loop, auto-reconnect | `health_check`/`health_recover` |
| `test` | live-test с откатом | `cmd_test`, `arm_test_guard` |
| `doctor` | диагностика | `cmd_doctor`, `doctor_*_checks` |
| `profile` | парсинг/валидация конфигов | `load_profiles`, `validate_profile` |
| `probe` | DNS/ping/tcp пробы | `cmd_probe`, `probe_*` |
| `verify` | реальный egress/geo/DNS/IPv6 | `cmd_verify`, `verify_connection_json` |
| `provider` | реестр провайдеров/регионов | `provider_registry_*`, `region_check` |
| `api` | локальный machine-first API | `api_dispatch` + socat |
| `control` | реестр агентов/харнессов, маршрутизация связей | НОВОЕ (Слой 1) |

## 3. Продукты (тонкие обёртки над core)

| Продукт | Точка входа | UI |
|---|---|---|
| CLI | `cmd/mazzy-vpn` | `internal/tui` (bubbletea или построчный) |
| Desktop | `desktop` | gio/Wails (ADR-0004) |
| Android | `android` | Kotlin UI + gomobile-биндинг `core` |

## 4. Инварианты (обязательный паритет с bash)

- Fail-closed при любой транзакции/сбое (нет прямого выхода в интернет).
- Single-flight: одна мутация состояния одновременно.
- Boot-recovery: восстановление последнего рабочего намерения, без «изобретения»
  нового подключения.
- Транзакционный тест с гарантированным откатом.
- Cooldown/rate-limit/circuit-breaker на автовосстановление.

## 5. Machine-first API (для агентов, Слой 1↔2)

Все команды возвращают стабильный JSON-envelope (замена текущего v1 API на
socat+jq). Требования:
- версионирование схемы,
- идемпотентные мутации с client-id,
- разделение «запрошено/выполнено» и «подтверждён защищённый egress».
