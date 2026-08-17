# Раунд 5 аудита — self-audit state/lock (Фаза 1, C1-7)

- Дата: 2026-08
- Область: `core/state`, `core/lock`.
- Метод: adversarial проверка durability и concurrency против bash-паритета.

## Проверенные риски

### L1 — flock-контенция внутри одного процесса — ПОДТВЕРЖДЕНО КОРРЕКТНО
Опасение: `flock` работает на open file description; два `Acquire` из одного
процесса могли бы «оба взять» лок. Эмпирическая проверка (два независимых
`os.OpenFile` одного файла): второй `Flock(LOCK_EX|LOCK_NB)` возвращает
`EWOULDBLOCK`. Значит single-flight держится и внутри процесса. Тест
`TestSingleFlightBusy` это фиксирует.

### L2 — temp-файл при сбое rename — ЧИСТО
`atomicWrite` вызывает `cleanup()` на всех путях ошибки (5 мест): chmod, write,
sync, close, rename. Хвостов не остаётся.

### L3 — data race — ЧИСТО
`go test -race ./lock ./state` — PASS (в т.ч. тест с горутиной, освобождающей
лок).

### L4 — целостность vendor/модулей — ЧИСТО
`go mod verify` = all modules verified. Случайный upgrade `x/sys` v0.36→v0.47
через `go get` откачен назад к версии amneziawg-go, чтобы не плодить дубли в
vendor. go.mod пинит `x/sys v0.36.0`.

### L5 — автономность не сломана — ЧИСТО
`x/sys/unix` использует сырые syscalls, не cgo. Статическая сборка
(`CGO_ENABLED=0`) сохранена: `engine-selftest` — statically linked.

## Паритет с bash (подтверждён)

| Свойство bash | Реализация Go | Тест |
|---|---|---|
| atomic temp+rename+`sync -f` | `atomicWrite` + fsync файла и каталога | round-trip |
| 0700 dir / 0600 file | `os.MkdirAll(0700)`+`Chmod`; temp `Chmod(0600)` | `TestFilePermissions` |
| PROFILE = basename | `filepath.Base` при записи | `TestWriteReadRoundTrip` |
| test-поля только в MODE=test | условная запись/чтение | `TestTestModePersists...` |
| flock -n (single-flight) | `LOCK_EX\|LOCK_NB`→ErrBusy | `TestSingleFlightBusy` |
| flock -w (recovery, bounded) | `AcquireRecovery(wait)` | `TestRecoveryWaitTimesOut` |

## Дефектов не найдено

В отличие от Раунда 4 (где был P0 G1), в state/lock новых дефектов нет:
логика проще и ближе к прямому переносу проверенной семантики.

## Примечание про наследуемый lock FD

Bash передаёт `VPNCTL_MUTATION_LOCK_FD` дочерним процессам
(`inherited_mutation_lock_valid`). В Go-архитектуре мутации выполняются
в одном процессе (движок — библиотека, не внешний бинарник), поэтому передача
FD между процессами не нужна. Если позже появится сценарий privileged-helper,
эту семантику добавим отдельной задачей.
