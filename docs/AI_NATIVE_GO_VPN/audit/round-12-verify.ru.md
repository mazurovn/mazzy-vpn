# Раунд 12 аудита — verify (Фаза 2, C2-6 завершение)

- Дата: 2026-08
- Область: `core/verify`.
- Метод: adversarial проверка verdict-precedence + паритет с bash
  verify_connection_json и уроками Раунда 1.

## Назначение

`core/verify` — фактическая проверка защищённого egress активного соединения:
тоннель активен, IPv4-egress идёт через тоннель (route match), нет IPv6-leak,
страна совпадает с ожидаемой из профиля. Machine-first: структурированный
verdict/findings (verified/warning/failed) для агентов, CLI и Desktop.

## Проверенные уроки (в т.ч. из Раунда 1)

### V1 — IPv6-degradation ДОЛЖНА влиять на verdict (R3 из Раунда 1) — СОБЛЮДЕНО
В прошлом bash-аудите (Раунд 1, R3) отмечалось, что IPv6-degradation не влияла
на doctor-verdict. Здесь IPv6-leak вызывает `escalate(Warning)` — реально меняет
verdict, а не только добавляет finding. Зафиксировано `TestIPv6LeakIsWarning`.

### V2 — precedence failed > warning > verified — КОРРЕКТНО
`escalate` никогда не понижает `Failed` до `Warning`. Комбинация no-egress
(fail) + IPv6-leak (warn) остаётся `failed`, но leak всё равно репортится как
finding. Зафиксировано `TestFailedBeatsWarning`.

### V3 — route match (bound==default) — ПАРИТЕТ
`RouteMatch=true` только когда bound IPv4 == default IPv4 (трафик реально идёт
через тоннель), иначе `verify.ipv4.route-mismatch` + warning. Паритет с bash
`route_same`.

## Модель verdict (паритет)

| Условие | Verdict | message_key |
|---|---|---|
| тоннель неактивен | failed | verify.failed.inactive |
| нет IPv4-egress | failed | verify.failed.no-egress |
| route mismatch | warning | verify.warning.route-mismatch |
| IPv6 leak | warning | verify.warning.ipv6-leak |
| country mismatch | warning | verify.warning.country-mismatch |
| DNS не через VPN | warning | verify.warning.dns-leak |
| всё ок | verified | verify.verified |

Неизвестные (geo/DNS недоступны) → не эскалируют (unknown), чтобы отсутствие
данных не превращалось в ложный warning.

## Дизайн

`Evaluate(Observation) Result` — чистая функция вердикта (детерминирована,
тестируется без сети). `Observer` инъектируется; в проде — обёртка над
`core/probe` + geo-lookup. JSON-вывод машиночитаем (проверен: verdict +
findings[] + echoed observations).

## C2-6 завершён

`core/probe` (Раунд 7) + `core/verify` (этот раунд) закрывают C2-6:
- probe — health-проверки (link/internet-through-tunnel/grace);
- verify — egress/route/IPv6/geo/DNS verdict.

## Проверки

- `core/verify` — 9 тестов PASS.
- Полный прогон: 14 пакетов, `go test -race ./...` — PASS.
- Статическая сборка сохранена. Дефектов не найдено.
