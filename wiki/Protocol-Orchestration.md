# Протоколы и AI-оркестрация

Подробная версия: [docs/PROTOCOL_ORCHESTRATION.ru.md](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/PROTOCOL_ORCHESTRATION.ru.md).

Stable `v1.3.2` содержит registry, redacted detection и `protocols.list`.
Read-only planner находится в draft
[#43](https://github.com/mazurovn/mazzy-vpn/pull/43) и появится в пакетах только
после merge и отдельного release gate.

Проверяемый каталог содержит 13 протоколов. Linux backends сейчас готовы для
AmneziaWG, WireGuard, OpenVPN и L2TP/IPsec. VLESS/REALITY, Hysteria 2, Mieru,
NaiveProxy, TUIC v5, Shadowsocks 2022, Trojan, AnyTLS и ShadowTLS v3 добавлены в
registry и roadmap. Однозначные share URI распознаются без вывода credential,
но connect остаётся `planned` до TUN/routing/rollback tests.
Текущая unreleased ветка также классифицирует ограниченные sing-box/Xray,
официальные Mieru и NaiveProxy JSON shapes; это не импорт и не запуск config.

```bash
mazzy-vpn protocols list --json
mazzy-vpn protocols diagnose --json
printf '%s\n' "$SHARE_URI" | mazzy-vpn protocols detect --stdin --json
printf '%s\n' "$PLANNER_JSON" | mazzy-vpn planner evaluate --stdin --json
```

URI допускает один завершающий `LF` или `CRLF`, но не встроенные переводы строк.
JSON допускает стандартный whitespace; duplicate keys, несколько документов и
неоднозначные multi-protocol configs отклоняются. Лимит обоих форматов — 64 КиБ.

Read-only planner принимает до 128 уникальных opaque profile IDs, применяет
пять backend-owned hard gates и детерминированную policy v1 на 100 баллов.
Gate rollback проверяет только защищённый storage для journal/snapshot, а не
готовность rollback конкретного backend. Наблюдаемое health evidence старше 900
секунд даёт ноль баллов; OpenVPN parser ограничен общим monotonic deadline.
`censorship-fit` и `workload-fit` вычисляет backend из catalog/workload, а не
принимает назначенными агентом.
Результат всегда `dry_run: true`; он не подключает VPN и не выполняет failover.
Агент получает только opaque IDs, readiness, factors и reason codes. LLM text
не становится shell command, mutation требует `action_id`, deadline, audit и
rollback. History и authorized execution остаются в issue #39.

---

<a id="english"></a>

# Protocol and AI orchestration

Full document: [docs/PROTOCOL_ORCHESTRATION.en.md](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/PROTOCOL_ORCHESTRATION.en.md).

Stable `v1.3.2` contains the registry, redacted detection and `protocols.list`.
The read-only planner remains in draft
[#43](https://github.com/mazurovn/mazzy-vpn/pull/43) until merge and a separate
release gate.

The validated catalog contains 13 protocols. Linux connection backends are
currently implemented for four. Nine censorship-oriented additions are
cataloged and their unambiguous share URIs are detected, but connection remains
`planned` until TUN, routing, rollback and real connection tests pass.
The current unreleased branch also classifies bounded sing-box/Xray, official
Mieru and NaiveProxy JSON shapes; classification is not import or execution.

A URI accepts one terminal `LF` or `CRLF` but no embedded line terminators. JSON
accepts standard whitespace; duplicate keys, multiple documents and ambiguous
multi-protocol configurations are rejected. Both formats are limited to 64 KiB.

The read-only planner accepts up to 128 unique opaque profile IDs, applies five
backend-owned hard gates and the deterministic 100-point policy v1. Results are
always `dry_run: true`; no connection or failover is performed. Its rollback
gate checks protected journal/snapshot storage, not candidate-specific rollback
capability. Observed health evidence older than 900 seconds scores zero, and the
OpenVPN parser shares the absolute monotonic deadline. Agents receive opaque
IDs, factors and reason codes only. Model text never becomes a shell command,
and no `planned` backend is selected. History and authorized execution remain
in issue #39.
The backend derives censorship and workload fit from the catalog and workload;
an agent cannot self-assign these factors.

Implementation is tracked in [#36](https://github.com/mazurovn/mazzy-vpn/issues/36),
[#37](https://github.com/mazurovn/mazzy-vpn/issues/37),
[#38](https://github.com/mazurovn/mazzy-vpn/issues/38) and
[#39](https://github.com/mazurovn/mazzy-vpn/issues/39).
