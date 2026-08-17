# Раунд 23 — автономная сессия: CI, U7, CONNMARK, control plane, DNS, install

- Дата: 2026-08. Ветка: `feat/go-rewrite`.
- Метод: автономная работа по бэклогу с аудитом на каждом шаге; ops_route для
  выбора политики (opus5 для архитектуры, terra для реализации), CI-гейт на
  каждый push.

## Выполнено (9 коммитов, все через гейты)

| Задача | Что | Тесты |
|---|---|---|
| **P3-CI** | Go CI workflow (build/test/race/vet/staticcheck/gofmt/static/audit) на go-rewrite | CI зелёный |
| **U7** | routes.Uplink: пин egress к выбранному адаптеру (host-route к endpoint) | 3 теста |
| **C1-4a2** | guard: CONNMARK save/restore (wg-quick паритет) | 1 тест |
| **L4-0** | SDD control-plane контракт (arch/04) | — |
| **L4-0a** | self-auth Ed25519 identity | 6 тестов |
| **L4-0b** | trust store (untrusted/paired/owned), non-transitive, anti-impersonation | 7 тестов |
| **L4-0c/d** | signed Grants (scope/expiry/sig), cascade revoke, prune | 8 тестов |
| **C1-4b2** | netexec RunInput + resolvconf DNS backend | 3 теста |
| **P3-2** | автономный installer + CI-гейт (нет apt/git/go build) | CI-гейт |
| **CLI control** | control id/pair/list (identity+pairing usable) | live |
| **integration** | end-to-end control plane (identity→pair→grant→revoke) | 1 тест |

## Найденные баги (bug-hunt)

- **BUG (partial Read)**: `resp.Body.Read()` за один вызов терял данные для
  chunked/TLS (Cloudflare trace, IP). Исправлено `io.ReadAll(LimitReader)` в
  stealth/probe/livecheck (в прошлой сессии), проверено здесь.
- **CI-breaking**: CLI через `replace ../core` не собирался с `-mod=vendor`;
  завендорил core в CLI. Найдено аудитом ДО первого CI-запуска.
- **nil rand**: `ed25519.GenerateKey(nil)` в control_cmd → `rand.Reader`.
- **dead code**: убраны неиспользуемые loadOrCreateIdentity/nodeIdentity.

## Инструментальные гейты (каждый коммит)

- `go build -mod=vendor` (static, CGO_ENABLED=0)
- `go test ./...` + `-race`
- `go vet`, `staticcheck` (core + cli)
- `gofmt -l`
- `tests/audit-public.sh` (секреты/PII)
- `tests/check-go-installer-autonomy.sh` (P3-2)

Итог: **34 core-пакета, все тесты + race clean, staticcheck clean, CI зелёный
на каждом push, публичный аудит чист.**

## Модели/маршрутизация (экономия токенов)

- `ops_route` перед сложными задачами: opus5 для архитектуры (L4-0), terra для
  реализации (U7). Реальный dispatch субагентов недоступен (pi-subagents
  runtime в соседнем проекте) — реализация выполнена напрямую с аудитом,
  граница честно зафиксирована.

## Осталось (крупное, вне быстрых итераций)

- P3-3 Desktop (gio) — отдельный продукт, многочасовой.
- P3-4 Android (gomobile) — отдельный продукт.
- P3-5 OpenVPN vendored — движок.
- C1-4d native-netlink (P2).
- F0-6b многоагентный консилиум (нужен pi-subagents runtime).
