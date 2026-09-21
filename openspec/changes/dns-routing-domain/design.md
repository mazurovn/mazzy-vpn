# Design

## Context
`core/dns.Manager` — тонкая обёртка над `resolvectl`/`resolvconf` через `netexec.Runner` (ADR-0005: только базовые утилиты). Выбор бэкенда шёл через пакетную функцию `netexec.Available`, что не давало протестировать ветку resolvectl.

## Goals / Non-Goals
**Goals:** без утечки DNS на аплинк при systemd-resolved; тест на точный набор команд.
**Non-Goals:** DoT/DNSSEC (есть отдельная проверка privacy), изменение бэкенда resolvconf, DNS для IPv6-серверов.

## Decisions
1. `~.` + `default-route yes` вместо `default-route no` на аплинке: не трогаем чужой линк, `revert` одного интерфейса полностью откатывает изменения.
2. Поле `Available func(string) bool` вместо глобальной подмены — без гонок в тестах.
3. Ошибка на `domain`/`default-route` — фатальна для Up (туннель без правильного DNS = утечка).

## Risks / Trade-offs
- На хостах без resolved поведение не меняется (resolvconf `-x` уже делает exclusive).
- Если профиль не задаёт DNS, Up — no-op, и запросы по-прежнему идут на аплинк; это существующее поведение, вне scope.
