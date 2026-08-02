# Reverse AI-agent control architecture

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

[Русская версия](AGENT_CONTROL_ARCHITECTURE.ru.md) ·
[transport registry](../agent-control/v1/registry.json) ·
[encrypted envelope schema](../agent-control/v1/envelope.schema.json)

## Status

This branch implements the versioned machine-readable contract, the
`agent-transports list|diagnose --json` CLI, package payload checks and security
policy. Transport runtimes, broker, Web/Telegram clients and the agent daemon
are not implemented. Their platform support remains `planned`.

## Two independent network planes

The egress VPN data plane connects applications to a provider or corporate
network. The reverse agent-control plane carries commands from Web, CLI or
Telegram to an already-running agent. They have separate trust boundaries,
lifecycle and registries. iroh, libp2p and WebRTC are agent transports, not VPN
entries in `protocols/v1`.

Happy provides durable HTTP/Socket.IO synchronization with opaque encrypted
payloads and first-party Web/mobile clients. Claude Bridge prioritizes pinned
LAN WSS and adds an iroh/QUIC TCP sidecar with direct/relay fallback. Mazzy uses
both ideas but keeps the application command protocol transport-neutral.

The v1 catalog contains LAN WSS, iroh QUIC, libp2p QUIC/Circuit Relay, WebRTC
DataChannel, WebTransport, explicit Tailscale/Headscale mesh integration and a
reverse WSS encrypted broker. A transport only carries encrypted envelopes; it
does not parse agent commands.

## Security and channels

The closed command schema exposes specific capabilities and deliberately has
no arbitrary `shell.exec`. The encrypted envelope uses recipient HPKE, an
Ed25519 sender signature, message IDs, sequence numbers and expiry. A broker may
store ciphertext and minimal routing metadata but must not receive plaintext.
The agent rechecks actor, capability, risk and confirmation before execution.

Standard Telegram Bot API messages are not a first-party E2EE channel. The bot
is therefore limited to read-only and low-risk operations and must not carry
prompts, artifacts, credentials or approvals. Full control is exposed through
a paired first-party Web client or Telegram Mini App using the same E2EE
envelope. High-risk actions require first-party, WebAuthn/TOTP or local
confirmation.

## Routing and delivery

Hard gates reject paths without runtime readiness, authorized pairing, valid
E2EE, anti-replay state, channel policy or loop-free routing. Valid LAN is
preferred. At most two internet paths are raced. The registry score considers
health, privacy, censorship fit, latency and battery cost. Reverse WSS may stay
warm for durable fallback. Resume starts at the last acknowledged sequence;
duplicate message IDs return the prior result without repeating a side effect.
Changing paths never downgrades encryption or authorization.

Control-plane routing relative to the egress VPN is explicit:
`through-egress`, `independent-underlay` for allowlisted control endpoints, or
`strict-coupled`. The backend prevents route loops and preserves the selected
control path across transactional VPN switches. An LLM cannot select this
policy.

## Separate gateway repository

The proposed `mazzy-agent-gateway` repository owns the unprivileged agent
daemon, policy gateway, opaque durable event log, transport plugins,
first-party Web/PWA and Telegram adapters. It must not run inside the privileged
VPN backend. The first production slice should implement LAN WSS plus reverse
WSS, pairing, status/list/prompt capabilities, ACK/resume, Linux agent support
and end-to-end tests. iroh follows behind the same contract tests; WebRTC,
libp2p and mesh adapters follow later.

Runtime status cannot move from `planned` until pinned dependencies, SBOM,
clean lifecycle tests, key rotation/revocation, NAT/relay matrices,
loss/reconnect/replay/flood tests, channel privilege tests and an independent
security review are complete.
