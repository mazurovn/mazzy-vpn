# Reverse AI-agent control architecture

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

[Русская версия](AGENT_CONTROL_ARCHITECTURE.ru.md) ·
[deep target architecture and delivery DAG (RU)](TARGET_ARCHITECTURE_2026-08-02.ru.md) ·
[comparison research (RU)](RESEARCH_AGENT_REMOTE_CONTROL_2026-08-02.ru.md) ·
[transport registry](../agent-control/v1/registry.json) ·
[draft future encrypted-envelope schema](../agent-control/v1/envelope.schema.json)

## Status

This branch implements the versioned machine-readable **draft contract**, the
`agent-transports list|diagnose --json` CLI, package payload checks, security
policy and the first embedded Desktop ingress. The AI Agents screen discovers
local Codex and Claude Code installations, displays discovery/catalog status
for all seven paths, and invokes official experimental Codex Remote Control
`start|pair|stop` through a
fixed typed adapter. The manual pairing code lives in UI memory only until its
expiry; the complete opaque vendor secret never enters the WebView.

This is **not** the first-party Mazzy client-to-client gateway. The Claude
adapter is discovery-only, while the transport runtimes, broker, Web/Telegram
clients and `mazzy-agentd` daemon are not implemented. All seven network paths
remain `planned` and must not be presented as release-ready.

The repeat independent review found that this document has the right direction
but is not yet a complete executable protocol. Pairing, result/event/ACK/error,
endpoint-derived policy, canonical cryptographic input, key lifecycle, path
state machines and a single egress mutation owner are specified in the
[deep target architecture (RU)](TARGET_ARCHITECTURE_2026-08-02.ru.md).

## Two independent network planes

The egress VPN data plane connects applications to a provider or corporate
network. The reverse agent-control plane carries commands from Web, CLI or
Telegram to an already-running agent. They have separate trust boundaries,
lifecycle and registries. iroh, libp2p and WebRTC are agent transports, not VPN
entries in `protocols/v1`.

Happy provides durable HTTP/Socket.IO synchronization with opaque encrypted
payloads and first-party Web/mobile clients. Claude Bridge prioritizes pinned
LAN WSS and adds an iroh/QUIC TCP sidecar with direct/relay fallback. Paseo puts
a multi-provider daemon behind Desktop/mobile/Web/CLI clients and an E2EE
relay. Yep Anywhere demonstrates a self-hosted browser-oriented session owner.
Official Codex app-server provides typed JSON-RPC for rich clients; Claude now
provides vendor-native Remote Control and MCP-based Channels. Mazzy combines
these patterns while keeping provider integration separate from transport.

The v1 catalog order follows the delivery architecture: reverse WSS is the
mandatory durable baseline, LAN WSS and iroh are accelerators, explicit
Tailscale/Headscale is optional, WebRTC and WebTransport are later experiments,
and libp2p is deferred until a concrete interoperability need. A transport only
carries encrypted envelopes; it does not parse agent commands.

## Provider adapters and local boundary

`Desktop/Web/Telegram -> command capability -> mazzy-agent policy -> provider
adapter -> Codex app-server / Claude / ACP` is the application path. Provider
credentials stay with the installed provider CLI. Typed app-server/SDK or ACP
interfaces are preferred; a raw PTY may exist only as an isolated legacy
adapter and cannot be the authentication, authorization or wire protocol.

The current Desktop adapter is deliberately narrower than the future gateway:
three enum operations and fixed argument arrays without a shell. Its current
mechanics are a 12-second direct-child timeout and renderer confirmation for
start/pair; process-group/job-object termination, bounded reader joins and
native command-bound approval remain R0 blockers. Pairing is not persisted.

## Security and channels — target, not runtime

The closed command schema exposes specific capabilities and deliberately has
no arbitrary `shell.exec`. The draft envelope schema only declares future HPKE
and Ed25519 identifiers; canonical inputs, vectors, key lifecycle and an E2EE
runtime do not exist yet. In the target design, a broker stores ciphertext and
minimal routing metadata, while the endpoint rechecks authorization.

Standard Telegram Bot API messages are not a first-party E2EE channel. The bot
is therefore limited to read-only and low-risk operations and must not carry
prompts, artifacts, credentials or approvals. Full control is exposed through
a paired first-party Web client or Telegram Mini App using the same E2EE
envelope. High-risk actions require first-party, WebAuthn/TOTP or local
confirmation.

## Routing and delivery — target, not runtime

Hard gates reject paths without runtime readiness, authorized pairing, valid
E2EE, anti-replay state, channel policy or loop-free routing. Reverse WSS/H2 is
the durable baseline; valid LAN may become the foreground accelerator, and
iroh follows only after its conformance gates. At most two internet paths are
raced. Resume starts at the last acknowledged sequence;
duplicate message IDs return the prior result without repeating a side effect.
Changing paths never downgrades encryption or authorization.

Control-plane routing relative to the egress VPN is explicit:
`through-egress`, `independent-underlay` for allowlisted control endpoints, or
`strict-coupled`. The backend prevents route loops and preserves the selected
control path across transactional VPN switches. An LLM cannot select this
policy.

## Separate gateway repository — target, not created

The proposed `mazzy-agent-control` monorepo (formerly drafted as
`mazzy-agent-gateway`) owns the unprivileged per-user `mazzy-agentd`, endpoint
policy and event log, the opaque relay/queue, transport plugins,
first-party Web/PWA and Telegram adapters. It must not run inside the privileged
VPN backend. The first production slice should implement reverse WSS/H2 plus
LAN WSS, pairing, status/list/prompt capabilities, ACK/resume, Linux agent support
and end-to-end tests. Before network access, the local daemon must expose typed
Codex app-server/Claude/ACP adapters over a protected local socket. iroh follows
behind the same contract tests; WebRTC, libp2p and mesh adapters follow later.

Runtime status cannot move from `planned` until pinned dependencies, SBOM,
clean lifecycle tests, key rotation/revocation, NAT/relay matrices,
loss/reconnect/replay/flood tests, channel privilege tests and an independent
security review are complete.
