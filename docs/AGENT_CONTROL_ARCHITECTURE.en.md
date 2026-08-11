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
policy and read-only Desktop diagnostics. The AI Agents screen discovers local
Codex and Claude Code candidates and displays discovery/catalog status for all
seven paths. It does not execute discovered binaries, and renderer/Tauri IPC
expose no provider start, pair, stop or pairing state.

This branch also implements a bounded Linux P0 `mazzy-agentd` slice for
`lan-wss`: TLS 1.3 mTLS, explicit client-certificate fingerprint confirmation,
scopes/channel policy, immediate revocation, anti-replay, durable dedupe, a
single-use local approval bound to the command digest and egress revision, and
seven egress capabilities. The protected channel terminates directly at the
daemon with no relay or intermediary; it does not claim the future HPKE envelope.

This is **not** the complete first-party Mazzy gateway. The Claude adapter stays
discovery-only; broker/relay E2EE, PAKE/QR pairing, Web/Telegram clients, agent
provider sessions, offline queue, resume, key rotation and the other six paths
remain `planned`. `linux=implemented` is limited to this local egress LAN-WSS
profile; an instance is `runtime_ready` only with the daemon running, valid TLS
files, an active pairing and the protected local API available.

The complete remote protocol still needs canonical cryptographic input, key
lifecycle, relay path state machines and provider session actors, as specified in the
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

The current Desktop adapter is deliberately diagnostics-only. Native
command-bound approval, trusted executable resolution,
process-group/job-object termination and bounded reader joins remain R0 gates;
provider lifecycle authority must not return before all of them are proven.

## Security and channels — target, not runtime

The closed command schema exposes specific capabilities and deliberately has
no arbitrary `shell.exec`. Every capability is bound to one fixed risk class;
a caller cannot downgrade risk. Telegram Bot accepts only argument-free
status/list/pause/cancel commands. The draft envelope schema only declares future HPKE
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

## Separate gateway repository — target beyond the local egress slice

The proposed `mazzy-agent-control` monorepo (formerly drafted as
`mazzy-agent-gateway`) owns the unprivileged per-user `mazzy-agentd`, endpoint
policy and event log, the opaque relay/queue, transport plugins,
first-party Web/PWA and Telegram adapters. It must not run inside the privileged
VPN backend. The repository-local `mazzy-agentd` is deliberately narrower: it
owns only the paired LAN-WSS egress capability surface and delegates privileged
network changes to the protected Mazzy VPN API. Reverse WSS/H2, agent-provider
status/list/prompt, ACK/resume and Codex/Claude/ACP adapters belong in that
future monorepo. iroh follows behind the same contract tests; WebRTC, libp2p and
mesh adapters follow later.

The remaining runtime paths cannot move from `planned` until pinned dependencies, SBOM,
clean lifecycle tests, key rotation/revocation, NAT/relay matrices,
loss/reconnect/replay/flood tests, channel privilege tests and an independent
security review are complete.

## Implemented LAN-WSS egress profile

`mazzy-agentd` is packaged with Desktop/CLI and can run as a `systemd --user`
service. The pairing administrator provisions a server certificate, a `0600`
private key and the trusted client-certificate CA under
`~/.config/mazzy-agentd/`; keys never enter commands, results or audit. Pairing
requires explicit confirmation of the full SHA-256 client certificate
fingerprint. High-risk `vpn.connect|vpn.disconnect` cannot self-approve over the
network: a trusted local UI/CLI confirms the canonical command digest and creates
a proof from a separate closed
[`approval-request`](../agent-control/v1/approval-request.schema.json), which has
no `confirmation_id`, with `mazzy-agentd approve --stdin --json`. The returned
single-use proof is inserted only into the final LAN-WSS command and binds peer,
actor, session, capability, target, TTL and the selected egress revision. Every
connect/disconnect mutation advances a host-global egress generation and
invalidates all session eligibility; `region.check` repeats the provider probe
and compares the fresh country with the selected/target country before browser
handoff. Revocation also stops commands on an already-open WSS session. Read-only
`diagnose --json` verifies a runtime-owned version record plus a bounded
`api.capabilities` round trip containing every required operation, and distinguishes an
installed binary from a configured, running `runtime_ready` instance. Desktop
discovery never executes discovered binaries and therefore reports readiness
fail-closed.
The LAN-WSS planner request is capped at 16 candidates: the full local API keeps
its 128-candidate limit, while this smaller transport boundary keeps the closed
PlannerEvaluation within the 48 KiB local-API and 60 KiB WebSocket envelopes
without truncation.
