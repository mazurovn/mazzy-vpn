# Agent Control Gateway

Текущий обзор: [docs/AGENT_CONTROL_ARCHITECTURE.ru.md](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/AGENT_CONTROL_ARCHITECTURE.ru.md).
Детальная целевая архитектура, P0 findings, protocol/state machines и delivery
DAG: [docs/TARGET_ARCHITECTURE_2026-08-02.ru.md](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/TARGET_ARCHITECTURE_2026-08-02.ru.md).

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

Сейчас реализованы draft contract, diagnostics и partial Desktop ingress. Новый экран
обнаруживает Codex/Claude, показывает discovery/catalog status и выполняет через фиксированный
Rust adapter официальный experimental Codex Remote Control `start|pair|stop`. Pairing code
не сохраняется. Это vendor-native relay, а не first-party Mazzy runtime:
`mazzy-agentd` — каноническое имя будущего per-user daemon, `relay` — серверной
opaque queue. Отдельный репозиторий `mazzy-agent-control` ещё не создан. Все
семь transport runtimes, `mazzy-agentd`, Web и Telegram не готовы.
`runtime_ready=true` разрешён только при наличии pinned runtime, успешных
probe/conformance/lifecycle/rollback/security gates и evidence ID; одного
contract row недостаточно.

Обычный Telegram Bot ограничен low-risk командами, поскольку Bot API не даёт
first-party E2EE. Полный control открывается через paired Web/Telegram Mini App.
Draft v1 не объявляет arbitrary shell execution и резервирует capability, TTL,
signature и anti-replay fields для будущего E2EE runtime. Canonical crypto
input, vectors, key lifecycle и trusted high-risk approval ещё не реализованы.

Исследование Happy, Claude Bridge, Paseo, Yep Anywhere и официальных
Codex/Claude interfaces: [docs/RESEARCH_AGENT_REMOTE_CONTROL_2026-08-02.ru.md](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/RESEARCH_AGENT_REMOTE_CONTROL_2026-08-02.ru.md).
Целевая цепочка разделяет ingress, E2EE transport и provider adapter; raw PTY
не является wire/auth protocol.

Production порядок уточнён повторным ревью: reverse HTTPS/WSS с H2 fallback
является durable baseline, LAN WSS и iroh идут как accelerators. WebTransport
является client-server path, Tailscale/Headscale включается только явно,
libp2p отложен до конкретной interoperability потребности.

`agent-control/v1/registry.json` теперь закрепляет тот же порядок. Каталог из
семи paths не означает одновременную реализацию: reverse WSS идёт первым,
LAN/iroh после общей conformance suite, остальные не блокируют ранние релизы.

## Когда не использовать

Текущий preview нельзя использовать как доверенный remote terminal или для
high-risk unattended actions. В scope первого релиза не входят arbitrary
`shell.exec`, raw PTY как auth/wire protocol, автоматические LLM approvals и
синхронизация provider API keys. Пока R0-2 не заменит renderer confirmation на
native command-bound approval proof, start/pair требуют локального внимания,
но не считаются достаточной границей для опасных удалённых команд.

---

<a id="english"></a>

# Agent Control Gateway (English)

Mazzy keeps provider/corporate egress VPN separate from reverse agent control.
The target design uses LAN WSS, iroh, libp2p, WebRTC, WebTransport, mesh and
reverse WSS beneath a future signed E2EE envelope. Current files are draft
catalog/schema declarations, not an E2EE runtime. Runtime
support for all seven paths is still planned. The current branch also has a
partial Desktop provider integration for discovery and typed official experimental Codex
Remote Control lifecycle/pairing. It is not a first-party `mazzy-agentd`.

The canonical planned components are the per-user `mazzy-agentd` and an opaque
server `relay` in a future separate `mazzy-agent-control` repository. Reverse
WSS/H2 is the durable baseline; LAN and iroh are later accelerators. The
current preview must not be treated as a trusted remote terminal: arbitrary
shell execution, raw PTY authorization and unattended high-risk approvals are
out of scope, and native command-bound confirmation is still an open R0 gate.
