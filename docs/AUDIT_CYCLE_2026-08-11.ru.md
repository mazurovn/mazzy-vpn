# Цикл аудита Mazzy VPN — 2026-08-11

## Метод

Цикл проведён в корне репозитория MAZZY_VPN на commits
`60b9b2d..5398c94`. Координация — Orca orchestration; поиск контекста — реальный
Mazzy Brain MCP (Nim stdio, `mazzy_search` и `mazzy_context_read`).
Consilium через connector завершился transport backstop без payload и не
считается подтверждением. VPN не подключался/отключался, `sudo` не применялся.

Проверены два независимых направления:

1. архитектура, контракты, алгоритмы, registry boundaries, concurrency,
   Bash/Python/JSON/Kotlin;
2. boot/runtime, systemd, local API, Desktop, recovery и package provenance.

## Найдено и исправлено в этом цикле

- `vpnctl-health.timer`: `OnUnitActiveSec + Persistent` зависал на systemd 259
  в `active/elapsed` с пустым next trigger. Для oneshot health-сервиса применён
  `OnUnitInactiveSec`; package drop-in использует один reset в начале и затем
  восстанавливает `OnBootSec=15s` и повторный интервал. Это важно: любой пустой
  `On*` сбрасывает весь список timer expressions.
- Desktop больше не считает отсутствие volatile cache причиной privileged
  bootstrap; cache warm-up отделён от engine repair.
- Desktop installation readiness использует bounded typed `api.capabilities`
  request, а не connect-and-close probe.
- Managed-profile validator запрещает ALPN при `tls.enabled=false`.
- Rollback не публикуется как завершённый без bounded observed-state refresh.
- Android validator теперь принимает только lowercase protocol IDs и проверяет
  закрытые TLS/DNS/routing формы; для восстановления `DESIRED=up` дополнительно
  требуется успешный `systemctl is-active --quiet`.

## Подтверждение

```text
bash tests/run.sh                         1..105
python3 tests/check-managed-protocol-adapter.py
  MANAGED PROTOCOL ADAPTER OK: 9 validated, 6 closed renderers, atomic import
python3 tests/check-android-contract.py
  PASS: Android foundation contract; runtime/device gates remain open
python3 tests/check-desktop-ui.py
  Desktop UI contract OK: v0.4.7, 110 IDs, 160 localized labels
```

## Оставшиеся release blockers

- Android import использует раздельные хранилища без crash-atomic journal;
  полноценная generated/schema-pinned проверка и device/Gradle gate ещё не закрыты.
- Provider и runtime-adapter registry дублируются в Bash, schemas и тестах;
  добавление нового provider требует синхронных ручных правок.
- Полный rollback пока не доказывает route/DNS/firewall/previous-tunnel
  invariants — нужен отдельный observed-state adapter и regression matrix.
- Установленная система всё ещё содержит старый `/usr/bin/mazzy-vpn`; DEB/RPM/
  AppImage artifacts отсутствуют, tag/version provenance не закрыт.
- Mazzy Brain connector в этой сессии вернул `Transport closed`; прямой Nim
  stdio MCP был проверен и отработал для поиска, но consilium не вернул payload.

Следующий цикл должен закрыть schema/runtime parity Android, вынести registry в
versioned source-of-truth и добавить package release gate `tag == embedded version`
до объявления production release.
