# Бэклог AI-NATIVE-GO-VPN

Статус: рабочий. Дата: 2026-08. Источник задач синхронизируется с локальным
Control Panel (`.mazzy-vpn-control/`, git-ignored).

Легенда приоритетов: **P0** блокер, **P1** высокий, **P2** средний.

## Фаза 0 — Фундамент и решения

| ID | Задача | Приор | Статус |
|---|---|---|---|
| F0-1 | Зафиксировать язык Go (ADR-0001) | P0 | DONE |
| F0-2 | Три автономных продукта (ADR-0002) | P0 | DONE |
| F0-3 | Автономный движок (ADR-0003) | P0 | DONE |
| F0-4 | Локальный форк Control Panel + git-исключения | P0 | DONE |
| F0-5 | SDD: концепция, правила, архитектура | P0 | DONE |
| F0-6 | Консилиум-аудит Фазы 0 (решения/риски) — Раунд 1 | P0 | DONE |
| F0-6b | Полный многоагентный консилиум (fresh-context) до Фазы 2 | P0 | TODO |
| F0-7 | Выбор GUI-тулкита Desktop (ADR-0004 финал: **gio**) | P1 | DONE |

Итоги Раунда 1: `../audit/round-01-phase0-decisions.ru.md`. Решения Фазы 0
подтверждены; добавлены P0-уточнения R1/R2 (ниже).

## Фаза 1 — Ядро mazzy-core (движок)

| ID | Задача | Приор | Статус |
|---|---|---|---|
| C1-1 | Go-модуль `core/`, скелет пакетов | P0 | DONE |
| C1-1a | **Pin Go toolchain 1.25.x** (`core/build.sh` авто-выбор) | P0 | DONE |
| C1-2 | Vendored `amneziawg-go` как библиотека (`core/vendor/`) | P0 | DONE |
| C1-3 | `engine/wireguard`: автономный up/down (TUN) | P0 | DONE (код; privileged smoke — C1-8) |
| C1-6a | **Go-парсер `.conf`** WG/AmneziaWG→UAPI (base64→hex) | P0 | DONE |
| C1-6 | `profile`: валидация AmneziaWG/WG конфигов | P0 | DONE |
| C1-4a | `routes`: таблицы/ip rule/fwmark (policy routing wg-quick) | P0 | DONE |
| C1-4a2 | CONNMARK save/restore (паритет wg-quick, G3) | P2 | DONE |
| C1-4b | `dns`: resolvectl | P0 | DONE (resolvectl) |
| C1-4b2 | `dns`: resolvconf stdin backend (G4) | P2 | DONE |
| C1-4d | native-netlink вместо exec `ip` (ADR-0005 этап 2) | P2 | TODO |
| C1-5 | `guard`: fail-closed nftables + IPv6 leak guard | P0 | DONE |
| C1-7 | `state`+`lock`: намерение + single-flight lease | P0 | DONE |
| C1-9 | `connect`-оркестратор (guard→engine→routes→dns, fail-closed) | P0 | DONE |
| C1-8 | Privileged smoke: поднять AmneziaWG-туннель без внешних *-quick | P0 | DONE (code+local); живой root-прогон PENDING SUDO |

C1-9/C1-8: Раунд 6 (`../audit/round-06-connect-c1-8.ru.md`). Найден/исправлен
C4 (P1: guard по реальному имени интерфейса). `tun.CreateTUN` доходит до
ядра (operation not permitted на границе прав). Скрипт: `core/scripts/smoke-c1-8.sh`.

C1-7: само-аудит Раунд 5 (`../audit/round-05-state-lock-selfaudit.ru.md`) — дефектов нет.
flock-контенция проверена эмпирически; `-race` PASS; atomic write+fsync;
0700/0600; static build сохранён (x/sys/unix — syscalls, не cgo).

Само-аудит (Раунд 4, `../audit/round-04-routes-guard-selfaudit.ru.md`) нашёл и исправил
**G1 (P0)**: рассинхрон fwmark сокета и таблицы → обрыв связи; **G2 (P1)**:
`src_valid_mark`. Закрыты regression-тестами. G3/G4 — техдолг (выше).

Прогресс Фазы 1: `core/` собирается **статически** (`CGO_ENABLED=0`,
`ldd` → not a dynamic executable). `engine-selftest` доказывает автономный
путь parse+validate+UAPI без jq/awg/awg-quick. Тесты `profile` — PASS.

R1/R2 (Раунд 1): `amneziawg-go` даёт только крипто+TUN. Маршруты, DNS,
policy-routing, парсинг `.conf` — НАШ код. C1-4 разбит на C1-4a/b/c + C1-6a.

## Фаза 2 — Паритет логики CLI

| ID | Задача | Приор | Статус |
|---|---|---|---|
| C2-1 | `connect`: транзакция подключения + откат | P0 | DONE (C1-9) |
| C2-2 | `health`: health-loop + auto-reconnect (паритет) | P0 | DONE |
| C2-3 | `bootrecovery`: boot-recovery gate | P0 | DONE |

C2-3: Раунд 10 (`../audit/round-10-bootrecovery.ru.md`). Найден/исправлен **B1 (P0)**:
gate был реализован, но НЕ применялся в `connect.Up` (зелёный, но не подключённый
предохранитель). Теперь `Options.BootGate` блокирует мутацию до любого
kernel-действия (тест «ноль команд при блоке»). Symlink-refusal, exit codes — паритет.
| C2-4 | `test`: live-test с откатом (`core/livetest`) | P1 | DONE |

C2-4: Раунд 11 (`../audit/round-11-livetest.ru.md`). Транзакция snapshot→up→
verify→commit/rollback; система всегда в зафиксированном состоянии. Тонкость T3:
rollback на живом parent-контексте (не на истёкшем) — покрыто тестом. 6 тестов.
| C2-5 | `doctor`: диагностика под новый движок | P0 | DONE |

C2-5: Раунд 8 (`../audit/round-08-doctor.ru.md`). Автономность измерима:
legacy doctor требовал 20 внешних тулов (awg/awg-quick/wg/jq/socat/python3...),
`core/doctor` — только `ip`/`nft`/`resolvectl`. Живой прогон: OK=5 WARN=1 FAIL=0.
Regression-тест против возврата legacy-зависимостей.
| C2-6 | `probe`/`verify`: пробы, egress/geo/DNS/IPv6 | P1 | DONE |

C2-6: Раунд 12 (`../audit/round-12-verify.ru.md`). `core/verify` — чистая
function вердикта (verified/warning/failed), precedence failed>warning>verified.
IPv6-leak реально влияет на verdict (урок R3). +`core/probe` (Раунд 7) = C2-6 закрыт.

C2-2: Раунд 7 (`../audit/round-07-health-probe.ru.md`). `core/health` считает
только последовательные сбои (один сбой ≠ reconnect), recovery ровно один
раз, startup-grace; `core/probe` биндит egress на тоннель (нет false-PASS).
8+6 тестов, `-race` PASS.
| C2-7 | `api`: machine-first JSON API (замена socat+jq) | P0 | DONE (логика; транспорт — Фаза 3) |
| C2-8 | TUI-меню + i18n (6 языков) | P1 | DONE (модель; рендер — Фаза 3) |
| C2-9 | Тесты паритета vs bash (guard/lease/recovery) | P0 | DONE |

C2-9: Раунд 15 (`../audit/round-15-integration-c2-9.ru.md`). `core/integration` —
6 сквозных сценариев на РЕАЛЬНЫХ пакетах (health-recover, live-rollback,
boot-gate blocks connect, guard-first, CLI-composition). Compile-time interface
assertions ловят дрейф. Устранёны orphan-пакеты (Q2), netexec 25%→96%.

**ФАЗА 2 ЗАКРЫТА.** 23 пакета, 104 тест-функции, 15 раундов аудита.

C2-8: Раунд 14 (`../audit/round-14-tui-i18n.ru.md`). 6 языков, data-driven
каталог (не switch-спагетти), меню отделено от рендера. Исправлен I3 (P1:
рассинхрон каталога с verify message_keys). Анти-«ущербность» гейты: полнота
переводов на 6 языков тестом. 14 тестов.

C2-7: Раунд 9 (`../audit/round-09-api.ru.md`). Версионированный JSON-конверт,
read-only/mutation политика, защита от request-smuggling (дубли ключей),
net request_id-spoofing, нет утечки внутренних ошибок. 9 тестов, 6 векторов безопасности.

Качество кода (Раунд 13, `../audit/round-13-code-quality.ru.md`): staticcheck/vet/gofmt/
-race — всё зелёное. Исправлен dead code (Q1: `lastReason`, `ipArgs`). Нет
хардкода в логике, нет спагетти, нет пустых пакетов, ошибки игнорируются
только намеренно (best-effort cleanup/rollback).

## Фаза 3.5 — UX CLI (клиентский путь)

| ID | Задача | Приор | Статус |
|---|---|---|---|
| U1 | `catalog`: managed store конфигов (import/list/fav/remove) | P0 | DONE |
| U2 | `measure`: UDP-probe серверов + ранжирование зон | P0 | DONE |
| U3 | `up NAME` / `up --best` (автоконнект по имени/лучшей зоне) | P0 | DONE |
| U4 | Интерактивное меню (mazzy-vpn без аргументов) | P0 | DONE |
| U5 | `netadapter`: выбор адаптера (кабель/Wi‑Fi), рекомендация | P0 | DONE |
| U6 | `netdiag`: анализ сети + фиксы (детект AdGuard tun0) | P0 | DONE |
| U7 | bind egress к выбранному uplink в connect | P1 | DONE |

U1-U6 (Раунд 19, `../audit/round-19-cli-ux-features.ru.md`): 65+ функций,
15 CLI-команд. Исправлены UX-1 (TCP→UDP probe для WG) и UX-2 (link-local
рекомендация). Проверено на реальных конфигах + кабель/Wi‑Fi + AdGuard.

## Фаза 3 — Продукты

| ID | Задача | Приор | Статус |
|---|---|---|---|
| P3-1 | CLI: один автономный статический бинарник | P0 | В РАБОТЕ (doctor/list/validate/connect/status) |

P3-1: Раунд 17 (`../audit/round-17-cli-real-configs.ru.md`). `mazzy-vpn-cli/`
собран, работает на РЕАЛЬНЫХ конфигах (6 AmneziaWG + OpenVPN). Исправлен P0:
парсер отвергал S3/S4/i1-i5/H-диапазоны — переписан на opaque-строки (amneziawg-go
парсит). SPDX AGPL-3.0 + Copyright Nik m (@mazurovn) в 53 файла. Секреты в вывод
не утекают. Осталось: демонизация/systemd для фонового туннеля.

P3-1 (Раунд 18, `../audit/round-18-cli-connect.ru.md`): добавлены `connect`
(foreground, single-flight lock + persist state) и `status --json`. root-check
первым (non-root не читает секретный файл). go.work git-ignored. Исправлены
CLI-4 (redundant filepath.Join) и S1023.
| P3-CI | CI-gate: staticcheck + vet + gofmt + -race | P1 | DONE (.github/workflows/go-ci.yml) |
| P3-2 | install без git clone/apt VPN-бэкендов | P0 | TODO |
| P3-3 | Desktop на **gio**, линкует core, работает без CLI, без webkit/GTK | P0 | TODO |
| P3-3a | Desktop: трей-иконка + автозапуск (gio трей не даёт) | P1 | TODO |
| P3-3b | Desktop: проверка рендера Wayland+X11 на целевых DE | P1 | TODO |
| P3-4 | Android: gomobile-биндинг core, без CLI | P1 | TODO |
| P3-5 | OpenVPN vendored статик в пакетах | P1 | TODO |

## Фаза 4 — Control Plane (Слой 1, AI-native)

| ID | Задача | Приор | Статус |
|---|---|---|---|
| L4-0 | **SDD контракт Control Plane** (идентификация/доверие/авторизация) (R6) | P1 | DONE (arch/04) |
| L4-0a | self-auth Ed25519 identity | P1 | DONE |
| L4-0b | trust store (untrusted/paired/owned) + pairing | P1 | DONE |
| L4-0c | signed Grants (scopes/expiry/signature) | P1 | DONE |
| L4-0d | каскадный отзыв при unpair (RevokeAllTo) | P1 | DONE |
| L4-1 | `control`: реестр агентов/харнессов/устройств | P1 | DONE |
| L4-2 | Маршрутизация связей «кто с кем» (deny-by-default) | P1 | DONE |
| L4-2p | Плоскость 2: `provider` region-check (LLM без блокировок) | P1 | DONE |
| L4-3 | Peer-to-peer защищённые каналы между агентами (Endpoint→data plane) | P2 | TODO |
| L4-4 | Multi-target routing к точкам выхода | P2 | TODO |

L4-1/L4-2/L4-2p: Раунд 16 (`../audit/round-16-two-plane-routing.ru.md`).
Две плоскости: control (deny-by-default, 100% cov) + provider
(region-readiness, 93.6% cov). Сквозной `TestScenarioAINativeTwoPlanes`.
Arch: `../architecture/03-two-plane-routing.ru.md`.

## Аудит-контроль

Каждая фаза завершается циклом проверок + консилиумом (см. `../audit/`).
P0-задачи не закрываются без прохождения аудита.
