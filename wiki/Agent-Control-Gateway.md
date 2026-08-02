# Agent Control Gateway

Полный документ: [docs/AGENT_CONTROL_ARCHITECTURE.ru.md](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/AGENT_CONTROL_ARCHITECTURE.ru.md).

Mazzy разделяет два слоя:

- **egress VPN** подключает приложения к VPN-провайдеру или корпоративной сети;
- **reverse agent control** доставляет команды из Web, CLI или Telegram к
  агентам на другом устройстве.

iroh, libp2p, WebRTC, WebTransport, Tailscale/Headscale и reverse WSS относятся
ко второму слою и не объявляются VPN-протоколами. Machine-readable catalog:
`agent-control/v1/registry.json`.

```bash
mazzy-vpn agent-transports list --json
mazzy-vpn agent-transports diagnose --json
```

Сейчас реализованы contract и diagnostics, но не transport runtimes. Поэтому
`runtime_ready` остаётся `false`.

Обычный Telegram Bot ограничен low-risk командами, поскольку Bot API не даёт
first-party E2EE. Полный control открывается через paired Web/Telegram Mini App.
В v1 нет arbitrary shell execution; команды имеют capability, TTL, signature,
anti-replay sequence и обязательное подтверждение для high-risk действий.

---

<a id="english"></a>

# Agent Control Gateway (English)

Mazzy keeps provider/corporate egress VPN separate from reverse agent control.
The latter uses interchangeable LAN WSS, iroh, libp2p, WebRTC, WebTransport,
mesh and reverse WSS paths beneath one signed E2EE command envelope. Runtime
support is still planned; the current implementation is the versioned contract
and fail-closed diagnostics.
