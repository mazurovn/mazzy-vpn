# Повторный release-аудит Mazzy VPN 1.4.0 / Desktop 0.4.0

Дата: 2026-08-03
Владелец: [Nik m (@mazurovn)](https://github.com/mazurovn)
Scope: PR #43, Linux egress control plane, local API, modern protocol
foundation, Desktop Agent Control, package/CI/release metadata.

## Метод

Проверка выполнена четырьмя независимыми ролями и затем сведена в один
remediation-проход:

- Claude: release/tag/package/platform audit;
- Codex: adversarial code review lifecycle, rollback, API durability and
  cross-platform executable discovery;
- Qwen с двумя subagent: документация, продуктовые claims и consistency;
- Kimi: независимая release-критика CI, unit/package claims и зависимостей.

Агенты не изменяли production source. Исправления проходили через единый
coordinator, ShellCheck, schema checks и полный regression suite.

## Почему релиза не было

Предыдущий HEAD не был отправлен в PR #43, версии оставались `1.3.2`/`0.3.2`,
а Desktop workflow содержал hard-coded release body старого patch-релиза.
Кроме metadata-блокеров review нашёл небезопасные lifecycle-сценарии, поэтому
создание tag до исправлений было бы неправильным:

1. transition guard снимался до доказанного восстановления предыдущего VPN;
2. guard закрывал только host output и разрешал весь root egress;
3. test/emergency/health paths обходили единый безопасный transition;
4. API journal мог существовать в page cache, но не быть durable до mutation;
5. Agent Control позволял caller назначить заниженный risk;
6. managed TLS validator был слабее опубликованной schema;
7. readiness фактически предпочитал IPv4 и не доказывал IPv6-only path;
8. Windows executable discovery не учитывал `PATHEXT`, а Unix принимал
   неисполняемые файлы;
9. sub-second lifecycle test был timing-dependent.

## Исправления

### Egress и rollback

- nftables guard содержит `output` и `forward` chains;
- blanket `meta skuid 0 accept` удалён;
- разрешаются только точные IP/port/transport managed и external-fallback
  endpoint, точные resolver addresses и выбранные tunnel interfaces;
- connect, reconnect, live test, emergency и health recovery устанавливают
  guard до остановки защищённого пути;
- guard снимается только после interface-bound readiness нового tunnel или
  подтверждённого rollback;
- AdGuard endpoint извлекается из реального шестого поля `ss`; его virtual
  interface разрешается только когда маршрут transport endpoint доказан через
  другой physical interface, поэтому blanket `oifname eth0 accept` невозможен;
- при двойном отказе сохраняются guard, root-only recovery marker и fallback
  allowlist;
- при reboot recovery oneshot с `CAP_NET_ADMIN` восстанавливает минимальный
  output/forward deny guard до запуска API, health и tunnel service; без test
  transaction marker требуется ручная проверка;
- stale test recovery использует наличие transaction marker как источник
  истины, даже если desired-state файл уже восстановлен в `MODE=normal`;
- IPv4-only и IPv6-only readiness проверяются отдельно строгим bounded parser.

Это transition safety, а не always-on kill switch. Boot marker восстанавливает
fail-closed guard, но постоянная namespace-owned policy и доказательство всех
route/DNS/firewall ресурсов остаются задачами целевого `mazzy-vpnd`.

### Local API

- snapshot, running/completed action record, audit и recovery marker получают
  file + parent-directory sync до lifecycle mutation;
- active state и rollback restore синхронизируются до durable completed record;
- повтор completed action восстанавливает отсутствующий terminal audit event,
  не выполняя mutation второй раз;
- удаление snapshot синхронизируется до terminal success;
- boot recovery unit включён в CI `systemd-analyze verify`;
- lifecycle timeout regression использует стабильный budget и отдельно
  доказывает завершение descendants;
- watchdog failure path не сбрасывает systemd start limiter.

### Protocol foundation

- uTLS ограничен закрытым enum;
- ALPN и certificate pins должны быть уникальными;
- certificate pin имеет минимальную длину;
- negative tests покрывают arbitrary/numeric uTLS, duplicate ALPN/pins и
  короткий pin;
- import девяти modern profiles остаётся `partial`, а lifecycle `planned`;
- шесть sing-box renderers не подключены к production tunnel service.

### Agent Control

- capability определяет фиксированный `read-only`/`low`/`medium`/`high` risk;
- Telegram Bot принимает только argument-free status/list/pause/cancel;
- high-risk permission response требует first-party confirmation;
- Desktop discovery не запускает provider binary;
- Unix candidate обязан быть executable;
- Windows использует canonical absolute candidate и `PATHEXT`, включая
  `.cmd`/`.bat`; bare extensionless candidate на Windows больше не принимается.

Это не готовое remote agent management. `mazzy-agentd`, pairing, signed E2EE
envelope, ACK/idempotency runtime, relay, Web/Telegram client и transport
implementations остаются отдельным release-gated направлением.

## Проверки

- `tests/run.sh`: 101/101;
- managed protocol adapter: 9 validators, 6 closed renderers, atomic import;
- Agent Control registry: 7 transports, 5 ingress channels;
- Bash syntax + ShellCheck для CLI, tests и runtime adapter;
- Rust format/unit/Clippy и npm audit пройдены локально; package payload/smoke
  для AppImage/DEB/RPM пройден; cargo-deny и CodeQL должны пройти в GitHub CI
  на release commit;
- все 14 документационных screenshots перегенерированы из изолированной
  localhost-only fixture с RFC 5737 данными; снимки показывают CLI `1.4.0`,
  Desktop `0.4.0`, непустую библиотеку профилей и diagnostics-only Agent
  Control без claims о готовом runtime.

## Статус платформ

| Surface | Статус 1.4/0.4 | Blocker |
|---|---|---|
| Linux CLI/TUI | functional release | дальнейший переход к `mazzy-vpnd` |
| Linux Desktop | unsigned functional preview | signing, native peer identity, remaining typed domains |
| Windows | unsigned UI preview | Windows Service, Wintun, signing, integration/fault tests |
| macOS | unsigned UI preview | Network Extension, helper, entitlements, notarization |
| Android | package отсутствует | Gradle, `VpnService`, native core, keystore, AAB and Play gate |
| iOS | package отсутствует | Network Extension, entitlement, signed app and store gate |

## Release gate

Tag и GitHub Release разрешены только после:

1. push текущего release commit в PR #43;
2. зелёных CI, CodeQL и Desktop Linux/macOS/Windows matrices;
3. merge PR #43 в `main`;
4. tag `v1.4.0` и `desktop-v0.4.0` на одном merge commit;
5. проверки SHA-256 manifests и наличия ожидаемых assets;
6. сохранения UI-only/mobile-not-implemented warnings в release pages.

Релиз нельзя описывать как поддержку 13 работающих VPN backend или готовое
управление агентами: фактический connection lifecycle реализован для четырёх
Linux backend.
