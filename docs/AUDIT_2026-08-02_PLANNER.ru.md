# Повторный аудит planner, архитектуры и секретов — 2026-08-02

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

## Область проверки

Проверены изменения draft
[#43](https://github.com/mazurovn/mazzy-vpn/pull/43), API/schema, policy
оркестрации, CLI и Python examples, repository Wiki, состояние stable release и
история Git. Аудит не объявляет issue #39 завершённым: history, authorized
connect/failover и Desktop/mobile integration отсутствуют.

## Исправленные находки

| Severity | Находка | Исправление |
|---|---|---|
| High | `deadline_ms` проверялся между кандидатами, но OpenVPN parser внутри одного кандидата имел отдельный timeout 10 секунд | Абсолютный monotonic deadline передаётся в `validate_profile` и parser; `124/137` нормализуются в `deadline-exceeded`; добавлен fault regression с зависающим parser |
| Medium | `recent_outcome=success` продолжал давать 30 баллов при `evidence_age_seconds > 900` | Всё наблюдаемое health evidence старше 900 секунд даёт ноль и reason `planner.factor.recent-stale` |
| Medium | Gate `rollback-available` переоценивал гарантию: код проверял только каталоги state/action | Gate и policy переименованы в `rollback-storage-ready`; документация запрещает считать его доказательством rollback конкретного backend |
| Medium | Public JSON Schema не требовал planner deadline и не связывал operation с `PlannerRequest` | Добавлен двусторонний conditional discriminator и planner-specific диапазон `100..30000 ms` |
| Low | Python example отвергал допустимый leading whitespace и принимал `NaN`/`Infinity` как JSON | Переход на strict `json.loads` с duplicate-key hook и запретом non-finite constants; добавлен отдельный validator |
| Low | Localization regression был flaky из-за `grep -q`/`SIGPIPE` при `pipefail` | Help сначала полностью считывается, затем проверяется marker |
| Medium | Regression suite при запущенном установленном API читал системный `/run/mazzy-vpn/api-v1.sock` и реальные профили вместо test fixture | `VPNCTL_API_SOCKET` изолирован внутри временного test root; воспроизводимость больше не зависит от состояния хоста |

## Секреты

- `./tests/audit-public.sh`: tracked tree и Git history не содержат известных
  VPN credentials или персональных absolute paths.
- `gitleaks detect --log-opts=--all --redact`: проверены 73 коммита, leaks не
  найдены.
- GitHub endpoint `secret-scanning/alerts` вернул `404`. Это означает, что
  server-side alerts недоступны текущему repository/token; результат нельзя
  трактовать как дополнительный чистый scan.
- Локальные untracked `.crew/` и `.pi/` не являются частью проекта, не
  добавлялись в index и не публикуются.

## Архитектурная оценка

Planner сохраняет правильную privilege boundary: caller и LLM передают только
opaque IDs и bounded health evidence, а eligibility вычисляет root backend из runtime,
текущего parse профиля, прав secret storage, platform support и защищённого
rollback storage. Result read-only, credential-free и всегда `dry_run: true`.
Повторное ревью нашло лишнее доверие к caller: он мог сам назначать
`censorship_fit` и `workload_fit`. Эти поля удалены из request schema; backend
теперь выводит их из versioned protocol catalog, workload и transport class.

Детерминированны rank, score и reason codes для одинакового local snapshot и
evidence; поле `evaluated_at` намеренно меняется. Внешний parser теперь включён
в общий deadline. Это необходимая граница до любых mutation/failover операций.

Оставшийся performance risk: Bash evaluator запускает несколько `jq`/validation
процессов на кандидата. Лимиты 128 кандидатов, 64 КиБ и 30 секунд заставляют
операцию fail closed, но перед authorized execution нужны benchmark/soak на
слабом hardware и перенос истории/агрегации evidence в backend-owned storage.

## Почему нет нового релиза

Опубликованы stable `v1.3.2` и Desktop preview `desktop-v0.3.2`. Planner после
этого baseline находится в draft PR #43. До merge, повторного CI на исправленном
commit и отдельного version/release gate публиковать planner как stable нельзя.
Issue #39 остаётся открытым намеренно: текущий срез не подключает VPN и не
выполняет failover.

Девять новых protocol entries — validated catalog/detection roadmap, а не
готовые connection backends. VLESS/REALITY, Hysteria 2, Mieru, NaiveProxy, TUIC,
Shadowsocks 2022, Trojan, AnyTLS и ShadowTLS не должны продвигаться в release как
рабочие подключения до задач #36–#38 и реальных TUN/DNS/route/leak/rollback
tests.

Capability contract отдельно учитывает Linux, Windows, macOS и Android, но
strict protocol registry v1 содержит только Linux, Windows и Android. Добавлять
обязательный `macos` в опубликованную v1-схему нельзя: старые strict consumers
отклонят новое поле. Per-protocol macOS status требует registry v2. Windows
preview не содержит Windows Service/Wintun, macOS preview не содержит Network
Extension, а Android client source отсутствует; сборка UI не закрывает gates.

Detector расширен на bounded JSON classification для sing-box/Xray, официальных
Mieru и NaiveProxy shapes. Он не отражает secrets, отклоняет duplicate keys,
несколько JSON documents и неоднозначные multi-protocol configs. Классификация
не является import, полной валидацией или разрешением root execution.

## Проверки

- `./tests/run.sh`: `81/81`;
- Rust unit tests: `25/25`;
- API contract: `28 operations`, `14 errors` в development branch;
- protocol registry: `13 protocols`, `9 share URI schemes`;
- Python planner example strict-boundary validator: пройден;
- `git diff --check`, Bash syntax и Python bytecode compilation: пройдены;
- public leak audit и gitleaks history scan: пройдены.
- локально собраны DEB, RPM и AppImage; package audit проверил payload,
  dependencies, lifecycle, embedded source и GUI launch;
- установленный DEB `0.3.2` проходит `dpkg -V`; `/usr/local/bin/mazzy-vpn` и
  `/usr/bin/mazzy-vpn` разрешаются в один package-owned файл. Локальный
  unreleased кандидат с тем же version не устанавливался и не публиковался;
- все 12 screenshots проверены как `1680×951`. Frontend в этом срезе не
  менялся, поэтому повторная генерация не требуется.
