# Раунд 10 аудита — boot-recovery gate (Фаза 2, C2-3)

- Дата: 2026-08
- Область: `core/bootrecovery`, интеграция в `core/connect`.
- Метод: adversarial проверка fail-closed после перезагрузки + проверка, что
  gate реально применяется (а не просто существует).

## Назначение

После перезагрузки система обязана НЕЗАВИСИМО подтвердить безопасный
защищённый egress ДО любой сетевой мутации. До этого status/profiles читаются,
но connect/disconnect/reconnect заблокированы. Это не даёт демону «изобрести»
новое подключение только потому, что юнит стартовал при загрузке.

## НАЙДЕННЫЙ ДЕФЕКТ (то, ради чего аудит)

### B1 — P0 — gate был реализован, но НЕ применялся в connect
`core/bootrecovery` прошёл 7 тестов, но `grep` показал: `connect.Up` его
**не вызывал** (0 ссылок). Классический «зелёный, но не подключённый»
предохранитель — как P0 из прошлого консилиума (MZ-CP-05: FAIL не блокирует
DONE).

**Исправление:** `Options.BootGate`; `Up` проверяет `MutationAllowed()`
**первым делом**, до арминга guard и создания интерфейса. При блокировке —
`ErrBootRecoveryPending` и НОЛЬ kernel-команд.

**Regression-тест** `TestBootRecoveryGateBlocksBeforeAnyKernelAction`:
заблокированный connect выдаёт **пустой** список вызовов runner (проверяется
`len(sr.calls) != 0`). Плюс `TestBootRecoveryReadyAllowsProceeding`: ready-gate
пропускает до kernel-действий.

## Проверенные свойства fail-closed

| Состояние | Мутация | Service gate | Exit code |
|---|---|---|---|
| нет файла (свежий boot) | БЛОК | БЛОК | 77 |
| running / test-recovery | БЛОК | БЛОК | 75 (retry) |
| awaiting-egress / awaiting-cleanup | БЛОК | разрешён | 0 |
| ready | РАЗРЕШЕНА | разрешён | — |
| recovery-only | БЛОК | БЛОК | 77 (manual) |

Точный паритет с bash `boot_recovery_mutation_gate` / `*_service_gate` /
`*_service_exit_code`.

## Дополнительная защита

- **Symlink refusal** (`! -L` паритет): подсунутый симлинк на «ready» файл
  трактуется как Unknown → блок (`TestSymlinkStateRefused`). Защита от
  state-redirection.
- Атомарная запись (temp+rename+fsync), 0700/0600.
- Non-required режим (unprivileged/dev) — gate пропускает всё.

## Проверки

- `core/bootrecovery` — 7 тестов; `core/connect` — +2 регрессии; все PASS.
- `go test -race ./...` — 12 пакетов PASS.
- Статическая сборка сохранена.

## Урок

Предохранитель без подключения бесполезен. Аудит поймал именно это: сам класс
корректен, но его надо было ВСТАВИТЬ в путь мутации. Теперь вставлен и покрыт
тестом «ноль kernel-команд при блокировке».
