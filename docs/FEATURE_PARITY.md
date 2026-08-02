# Mazzy VPN capability parity / Паритет функций

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

Machine-readable source of truth:
[`capabilities.json`](capabilities.json). CI validates that every capability,
test/document reference and release gate remains consistent.

Status: **I** implemented · **P** partial · **R** planned · **—** not applicable.

Matrix scope: **current unreleased worktree**. Published
`desktop-v0.3.2` does not contain the AI Agents screen or
`agent-provider-integration`; `P` below describes source in this branch, not a
shipped artifact. That capability is an explicit Desktop 1.0 gate on Linux,
macOS and Windows.

| Capability ID | CLI | TUI | Linux | macOS | Windows | Android | iOS |
|---|---:|---:|---:|---:|---:|---:|---:|
| `connection-lifecycle` | I | I | I | R | R | R | R |
| `profile-import` | I | I | I | R | R | R | R |
| `profile-location-selection` | I | I | I | R | R | R | R |
| `validation-probe-test` | I | I | I | R | R | R | R |
| `egress-location-verification` | I | I | I | R | R | R | R |
| `operating-modes` | I | I | P | R | R | R | R |
| `service-control` | I | P | I | R | R | — | — |
| `dependency-bootstrap` | I | — | P | R | R | — | — |
| `dashboard-tray` | I | I | I | P | P | R | R |
| `localization-six-languages` | I | I | P | P | P | R | R |
| `automatic-recovery` | I | I | I | R | R | R | R |
| `self-contained-runtime` | I | I | P | R | R | R | R |
| `privilege-boundary` | I | I | P | R | R | R | R |
| `protocol-catalog-detection` | I | R | R | R | R | R | R |
| `ai-orchestration-contract` | P | R | R | R | R | R | R |
| `agent-provider-integration` | R | R | P | P | P | R | R |
| `versioned-local-api` | P | P | P | P | P | R | R |
| `mobile-vpn-lifecycle` | — | — | — | — | — | R | R |

## Release gates

| Gate | Declared ready | Meaning |
|---|---:|---|
| `cli-tui-1.3` | yes | Current Linux CLI/TUI release gate |
| `desktop-linux-1.0` | **no** | Standalone Linux application; no separate CLI install |
| `desktop-macos-1.0` | **no** | Standalone signed macOS application with native backend |
| `desktop-windows-1.0` | **no** | Standalone signed Windows application with native backend |
| `mobile-android-1.0` | **no** | Native signed Android client using `VpnService` |
| `mobile-ios-1.0` | **no** | Native signed iOS client using Network Extension |

The validator calculates readiness from the matrix. A gate cannot be marked
ready while any required capability is `partial`, `planned` or
`not-applicable`.

## Русский

Desktop 0.3 — Linux control-center preview. Он уже включает установщик общего
движка, импорт и выбор профилей, validate/probe/live-test, Doctor с полным
выводом, фактическую проверку egress/DNS/IPv6/локации, сортировку локаций,
расширенный tray, журнал и управление службами. Desktop 0.3 опубликован как
unsigned preview, а gate Desktop 1.0 всё ещё закрыт: issue #31 закрыт точным
provenance-verified backport `glib`, но не завершены fallback-policy UI, полный перевод новых экранов на шесть языков и
переход всех typed `pkexec`-операций к локальному versioned daemon API.
Контракт API `1.0`, manifest, безопасные envelopes и ограниченный protected
service уже реализованы в `main` и release source, но большинство operation
domains и native caller identity отсутствуют. DEB/RPM владеют engine/service
payload и bootstrap, а AppImage всё ещё зависит от уже установленного `pkexec`.
Android и iOS пока являются только планом: UI preview или Desktop wrapper не
считаются мобильным VPN-клиентом.

Каталог из 13 протоколов и redacted CLI/API detection уже реализованы, но это
не девять новых connection backends. VLESS/REALITY, Hysteria 2, Mieru,
NaiveProxy, TUIC v5, Shadowsocks 2022, Trojan, AnyTLS и ShadowTLS v3 остаются
`planned` для Linux/macOS/Windows/Android до platform adapter, TUN/routing/DNS,
secret storage, rollback и реальных integration tests. AI orchestration пока
`partial`: CLI/API уже исполняют детерминированную read-only оценку с hard
gates; censorship/workload fit выводится из trusted catalog/workload, а не
назначается агентом. History store, authorized execution/failover и Desktop/mobile
integration ещё не реализованы.

Agent provider integration также `partial`: unreleased Desktop обнаруживает
Codex/Claude и управляет официальным experimental Codex Remote Control через три
типизированные операции с memory-only pairing. Это не готовый
client-to-client `mazzy-agentd`; семь network paths, Web,
Telegram и Claude lifecycle остаются `planned`.

Новая функция считается завершённой не после добавления одной кнопки, а после
обновления общего API/core, всех применимых интерфейсов, автоматических тестов,
матрицы и документации на русском и английском.

## English

Desktop 0.3 is a Linux control-center preview. It bundles the shared-engine
installer and now exposes profile import/selection, validation, probes, live
tests, actual egress/DNS/IPv6/location verification, location sorting,
an expanded tray, full Doctor output, logs and service controls. Desktop 0.3 is
published as an unsigned preview, while the Desktop 1.0 gate remains closed:
issue #31 is closed with a provenance-verified `glib` backport, but fallback-policy UI, full six-language coverage for the new screens and
the versioned local daemon API still need to replace every typed `pkexec`
operation. The API `1.0` contract, manifest, frontend-safe envelopes and a
limited protected service are implemented on `main` and in the released source,
but most operation domains and native caller identity are still missing.
DEB/RPM own the engine/service payload and bootstrap, while AppImage still
requires `pkexec` to be present already.
Android and iOS are currently plans only: a UI preview or wrapped Desktop
frontend does not count as a mobile VPN client.

The 13-protocol catalog and redacted CLI/API detection are implemented, but
this is not a claim of nine new connection backends. VLESS/REALITY, Hysteria 2,
Mieru, NaiveProxy, TUIC v5, Shadowsocks 2022, Trojan, AnyTLS and ShadowTLS v3
remain `planned` on Linux/macOS/Windows/Android until platform adapters,
TUN/routing/DNS, secret storage, rollback and real integration tests pass. AI
orchestration is `partial`: CLI/API now execute deterministic read-only
evaluation with hard gates; censorship/workload fit is derived from the trusted
catalog/workload rather than assigned by an agent. History storage, authorized execution/failover
and Desktop/mobile integration are not implemented.

Agent provider integration is also `partial`: the unreleased Desktop discovers
Codex/Claude and controls official experimental Codex Remote Control through three typed
operations with memory-only pairing. This is not a complete client-to-client
`mazzy-agentd`; all seven network paths, Web, Telegram and Claude
lifecycle remain `planned`.

A feature is complete only after its shared API/core, every applicable
interface, automated tests, this registry and both Russian and English
documentation are updated.
