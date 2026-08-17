# Раунд 18 аудита — CLI connect/disconnect/status + качество

- Дата: 2026-08
- Область: `mazzy-vpn-cli` (connect/status), go-workspace, качество кода.
- Метод: живая проверка non-privileged путей + adversarial self-audit.

## Реализация

CLI получил привилегированные команды:
- `connect PROFILE` — поднимает туннель в FOREGROUND (держит до Ctrl+C), затем
  чистый teardown. Single-flight lock + персист intent в state.
- `status [--json]` — показывает сохранённое намерение.

Go-workspace (`go.work`, git-ignored) связывает модули `core` и `cli` без
публикации локальных путей.

## Self-audit (запрос: dead code / hardcode / spaghetti / логика / ущербность)

### CLI-1 — порядок проверок в connect — КОРРЕКТЕН
`requireRoot` вызывается ПЕРВЫМ, до `loadProfile`. Значит non-root пользователь
даже не читает файл профиля (не может спровоцировать парсинг секретного файла).
Проверено чтением кода + живым запуском (`connect` без root → отказ, файл не
читается).

### CLI-2 — хардкод путей — НЕТ (env-overridable)
`/var/lib/mazzy-vpn` и `/run/mazzy-vpn` — документированные системные дефолты
с override через `MAZZY_STATE_DIR`/`MAZZY_RUN_DIR` (паритет с bash
VPNCTL_STATE_DIR/RUN_DIR). Это не хардкод, а конфигурируемый fallback.

### CLI-4 — избыточный `filepath.Join(runDir())` — ИСПРАВЛЕНО
Найден code smell: `filepath.Join` с одним аргументом бессмысленен и тянул
лишний импорт. Упрощено до `return runDir()`, импорт `path/filepath` убран.

### Прочее — ЧИСТО
- Пустых/заглушечных функций нет (18 функций, все с телом).
- staticcheck нашёл `S1023` (redundant return в printUsage) — исправлено.
- go vet чисто.

## Безопасность (запрос заказчика)

| Проверка | Результат |
|---|---|
| секреты в выводе status/connect | НЕТ (только state/protocol/profile-basename) |
| profile basename, не полный путь | да (state хранит basename) |
| non-root не читает секретный файл | подтверждено (root-check первым) |
| go.work с локальными путями в git | НЕТ (git-ignored) |
| public audit | OK |

## Лицензия/авторство

Все 4 CLI-файла несут SPDX `AGPL-3.0-or-later` + `Copyright © 2026 Nik m
(@mazurovn)`.

## Проверки

- CLI собирается статически (`statically linked`).
- core: 19 пакетов PASS. CLI: vet/staticcheck чисто.
- Живо: status (down/json), connect без root — корректный отказ, help.

## Ограничение (честно)

`connect` пока foreground (движок — библиотека в процессе). Демонизация/systemd
и `disconnect` для фонового туннеля — следующий шаг (P3-1 продолжение). Живой
root-туннель проверяется `core/scripts/smoke-c1-8.sh` (нужен sudo).
