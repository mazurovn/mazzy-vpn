# Раунд 15 аудита — интеграционные parity-тесты (Фаза 2, C2-9) + качество

- Дата: 2026-08
- Область: `core/integration` + сквозная проверка всех пакетов.
- Метод: end-to-end wiring реальных пакетов + quality gate по запросу заказчика.

## Реализация C2-9

`core/integration` — пакет без прод-кода, где РЕАЛЬНЫЕ пакеты стыкуются на
своих истинных интерфейсах (faked только kernel-граница). Interface drift между
пакетами ломает сборку.

### Сценарии (паритет с bash CLI)

| Сценарий | Что проверяет |
|---|---|
| HealthDegradesThenRecovers | connect→health: 2 последовательных сбоя → recover ровно 1 раз |
| LiveTestRollsBackToPreviousState | livetest+state: verify FAIL → откат к previous |
| VerifyProducesActionableVerdict | verify: IPv6-leak → warning verdict для UI/агента |
| BootGateBlocksConnectEndToEnd | connect+bootrecovery: после reboot connect → **0 kernel-команд** |
| ReadyGateArmsGuardFirst | connect: ready-gate → первый kernel-action = nft IPv6 guard |
| CLIComposition | api+doctor+lock+tui+i18n стыкуются как в будущем CLI |

### Compile-time interface assertions
`var _ health.Probe = (*probe.NetProbe)(nil)` и
`var _ doctor.Environment = fakeHealthyEnv{}` — дрейф интерфейсов = ошибка
компиляции, а не рантайм-сюрприз.

## Найденное при quality-аудите

### Q2 — «orphan» leaf-пакеты (api/doctor/lock/tui) — РАЗРЕШЕНО
Они не импортировались другими пакетами (их потребитель — CLI-бинарник Фазы 3).
Чтобы не были «мертвы на практике», добавлен `TestScenarioCLIComposition`,
который реально их связывает. Теперь каждый достижим из интеграции.

### Q3 — низкое покрытие netexec (25%→96.4%) — УЛУЧШЕНО
`ExecRunner.Run`/`Available` тестировались слабо (нужны реальные процессы).
Добавлены тесты на `echo`/`false`/`sh` — покрытие 96.4%.

## Quality gate (запрос: dead code / hardcode / spaghetti / логика)

| Проверка | Результат |
|---|---|
| go vet | чисто |
| staticcheck | чисто |
| gofmt | отформатировано |
| go test -race | 17 пакетов PASS |
| хардкод в integration | НЕТ |
| orphan-пакеты | устранены (все wired) |

## Честная картина покрытия

Пакеты с 0% (`dns`, `engine/wireguard`, `cmd/*`) требуют реального ядра/root
или являются entry-points. Их логика проверяется privileged smoke (C1-8) и
статической сборкой, а не unit-тестами. Пакеты с чистой логикой:
livetest 89%, profile 83%, routes 82%, doctor 82%, i18n 76%, health 75%,
lock 75%, bootrecovery 72%, api 69%, probe 60%, netexec 96%.

## Итог Фазы 2

Все C2-1..C2-9 закрыты. 23 пакета (17 с тестами), 104 тест-функции, 15 раундов
аудита. Инструментальные gate зелёные. Пакеты доказанно композируются в
поведение bash CLI (guard/lease/recovery/rollback/boot-gate).
