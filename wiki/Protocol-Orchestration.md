# Протоколы и AI-оркестрация

Подробная версия: [docs/PROTOCOL_ORCHESTRATION.ru.md](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/PROTOCOL_ORCHESTRATION.ru.md).

Проверяемый каталог содержит 13 протоколов. Linux backends сейчас готовы для
AmneziaWG, WireGuard, OpenVPN и L2TP/IPsec. VLESS/REALITY, Hysteria 2, Mieru,
NaiveProxy, TUIC v5, Shadowsocks 2022, Trojan, AnyTLS и ShadowTLS v3 добавлены в
registry и roadmap. Однозначные share URI распознаются без вывода credential,
но connect остаётся `planned` до TUN/routing/rollback tests.

```bash
mazzy-vpn protocols list --json
mazzy-vpn protocols diagnose --json
printf '%s\n' "$SHARE_URI" | mazzy-vpn protocols detect --stdin --json
```

Агент получает только opaque IDs, readiness и evidence. LLM text не становится
shell command, mutation требует `action_id`, deadline, audit и rollback.

---

<a id="english"></a>

# Protocol and AI orchestration

Full document: [docs/PROTOCOL_ORCHESTRATION.en.md](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/PROTOCOL_ORCHESTRATION.en.md).

The validated catalog contains 13 protocols. Linux connection backends are
currently implemented for four. Nine censorship-oriented additions are
cataloged and their unambiguous share URIs are detected, but connection remains
`planned` until TUN, routing, rollback and real connection tests pass.

Agents receive opaque IDs, readiness and evidence only. Model text never
becomes a shell command, and no `planned` backend is selected automatically.

Implementation is tracked in [#36](https://github.com/mazurovn/mazzy-vpn/issues/36),
[#37](https://github.com/mazurovn/mazzy-vpn/issues/37),
[#38](https://github.com/mazurovn/mazzy-vpn/issues/38) and
[#39](https://github.com/mazurovn/mazzy-vpn/issues/39).
