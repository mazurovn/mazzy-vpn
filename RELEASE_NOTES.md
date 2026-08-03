# Mazzy VPN 1.4.0 / Desktop 0.4.0

Release source prepared 2026-08-03 by [Nik m (@mazurovn)](https://github.com/mazurovn).
Publication is confirmed only by the corresponding GitHub tag and Release page.

## English

Mazzy VPN 1.4.0 advances the audited Linux control plane for resilient access
to AI services, the web and corporate networks. Desktop 0.4.0 is a functional
unsigned Linux preview built on the same engine. It is not a cross-platform VPN
release: macOS and Windows bundles remain UI-only previews, and no Android
package is produced.

### Highlights

- A deterministic, read-only `planner.evaluate` ranks up to 128 opaque
  candidates using backend-owned runtime, profile, rollback-storage, platform,
  protocol and evidence gates. It cannot connect or switch a tunnel.
- The versioned catalog covers 13 protocols. AmneziaWG, WireGuard, OpenVPN and
  L2TP/IPsec remain the four implemented Linux connection backends.
- VLESS/REALITY, Hysteria 2, Mieru, NaiveProxy, TUIC v5, Shadowsocks 2022,
  Trojan, AnyTLS and ShadowTLS v3 now have closed profile schemas, bounded
  credential-redacted detection and atomic `0700/0600` Linux import.
- Six modern protocols have a closed sing-box TUN/config renderer foundation.
  It is deliberately not wired into connect, DNS/routing ownership, rollback
  or release support. Mieru, NaiveProxy and ShadowTLS still require audited
  sidecar/chain adapters.
- Managed connect, reconnect, live test, emergency selection and watchdog
  recovery use a dual-stack nftables transition guard. It covers host output
  and forwarded traffic, permits only exact transport/resolver endpoints, and
  remains fail-closed if both the new tunnel and rollback cannot be verified.
- An unresolved transition marker restores a minimal output/forward deny guard
  at boot before the API socket, health monitor or managed tunnel can start.
- AdGuard fallback preserves only its exact live transport peer and a tunnel
  interface proven distinct from that peer's physical route.
- IPv4-only and IPv6-only interface-bound readiness paths are supported.
- API snapshots, active state, action journals, audit records and recovery
  markers are synced before lifecycle completion; retries repair a missing
  terminal audit event without repeating the mutation.
- `verify-service` and API `tests.verify-service-egress` provide bounded,
  unauthenticated NotebookLM/OpenAI egress classification without cookies,
  credentials, arbitrary URLs or automatic profile selection.
- The reverse Agent Control architecture catalogs reverse WSS, LAN WSS, iroh,
  libp2p, WebRTC, WebTransport and Tailscale/Headscale. Capability risk is fixed
  by schema and Telegram Bot is low-risk only. These are contracts and
  diagnostics, not a remote-control or E2EE runtime claim.
- Desktop preserves explicit profile-cache failure reasons, reads legacy caches
  with opaque derived IDs, and discovers Codex/Claude executables without
  running them. Discovery uses allowlisted names and fixed system directories;
  Unix executable permissions and a fixed Windows executable suffix set are
  enforced without trusting `PATH`, `HOME` or `PATHEXT`.
- CI now syntax-checks and shellchecks the runtime adapter and verifies the API
  recovery unit together with the other systemd units.
- Recovery-marker startup blocks return systemd stop status `77`, so
  `Restart=always` does not loop while manual recovery is required. Desktop
  provider readiness is computed from reported adapter state instead of a
  fixed badge value.

### Platform status

- **Linux CLI/TUI 1.4.0:** functional release; DEB/RPM/source artifacts.
- **Linux Desktop 0.4.0:** functional unsigned preview; AppImage, DEB and RPM.
- **Windows Desktop 0.4.0:** unsigned UI preview only; no Windows Service or
  Wintun backend and therefore no traffic protection.
- **macOS Desktop 0.4.0:** unsigned UI preview only; no Network Extension or
  privileged helper and therefore no traffic protection.
- **Android/iOS:** no application package. There is no Gradle project,
  `VpnService`, Packet Tunnel backend, signed AAB/APK or mobile release gate.

### Important limits

- Modern protocol import/render support is not equivalent to working
  import/connect/TUN/DNS/routing/rollback/leak-tested lifecycle support.
- Agent-control transport schemas are not an implemented client-to-client
  gateway, Telegram controller or remote agent runtime.
- `verify-service` reaches only a public unauthenticated boundary. Provider
  acceptance can still depend on account region, cookies, browser/device
  signals, organization policy and provider risk scoring.
- The transition guard is not a persistent always-on kill switch.
- Preview artifacts are unsigned. Hashes protect download integrity but do not
  establish publisher identity.

### Upgrade

Linux packages declare `nftables` and `python3` in addition to the existing
runtime dependencies. Package installation enables the hardened API recovery
unit before tunnel/test recovery. DEB/RPM continue to migrate trusted legacy
`/usr/local/bin` copies to package-owned `/usr/bin` commands with reversible
backups.

## Русский

Mazzy VPN 1.4.0 развивает проверенный Linux control plane для устойчивого
доступа к AI-сервисам, открытому интернету и корпоративным сетям. Desktop 0.4.0
является функциональным, но неподписанным Linux preview на том же движке. Это
не кроссплатформенный VPN-релиз: сборки macOS и Windows остаются только preview
интерфейса, Android-пакет не выпускается.

### Основные изменения

- Детерминированный read-only `planner.evaluate` ранжирует до 128 кандидатов по
  backend-owned gates и наблюдаемым данным. Planner не подключает и не
  переключает VPN.
- Каталог содержит 13 протоколов. Рабочими Linux connection backend остаются
  AmneziaWG, WireGuard, OpenVPN и L2TP/IPsec.
- Для VLESS/REALITY, Hysteria 2, Mieru, NaiveProxy, TUIC v5,
  Shadowsocks 2022, Trojan, AnyTLS и ShadowTLS v3 добавлены закрытые схемы,
  безопасное распознавание без утечки credentials и атомарный Linux import с
  правами `0700/0600`.
- Для шести современных протоколов есть закрытая основа sing-box TUN/config
  renderer. Она намеренно не подключена к lifecycle, DNS/routing, rollback и
  release support. Mieru, NaiveProxy и ShadowTLS ещё требуют проверенных
  sidecar/chain adapter.
- Connect, reconnect, live test, emergency и watchdog recovery защищены
  dual-stack nftables transition guard. Он закрывает output и forward,
  разрешает только точные endpoint транспорта и resolver и остаётся
  fail-closed, если не подтверждены ни новый туннель, ни rollback.
- При unresolved transition marker boot recovery восстанавливает минимальный
  output/forward deny guard до запуска API socket, health monitor и tunnel.
- Для AdGuard разрешаются только точный live transport peer и tunnel interface,
  доказанно отличный от physical route этого peer.
- Проверяется защищённый IPv4-only и IPv6-only egress через выбранный интерфейс.
- API snapshot, active state, action journal, audit и recovery marker
  синхронизируются до lifecycle completion; повтор action восстанавливает
  отсутствующий terminal audit event без второй mutation.
- `verify-service` и `tests.verify-service-egress` выполняют ограниченную
  unauthenticated-проверку NotebookLM/OpenAI без cookies, credentials,
  произвольных URL и автоматического выбора профиля.
- Архитектура reverse Agent Control каталогизирует reverse WSS, LAN WSS, iroh,
  libp2p, WebRTC, WebTransport и Tailscale/Headscale. Risk capability закреплён
  схемой, Telegram Bot ограничен low-risk командами. Это контракты и
  диагностика, а не заявление о готовом remote-control/E2EE runtime.
- Desktop сохраняет причины ошибки profile cache, читает legacy cache через
  производные opaque ID и обнаруживает Codex/Claude без запуска. На Unix
  проверяются executable permissions, на Windows разрешён фиксированный набор
  executable suffixes. Имена и каталоги ограничены allowlist, а `PATH`, `HOME`
  и `PATHEXT` не управляют файловым поиском.
- CI проверяет runtime adapter через `bash -n`/ShellCheck и валидирует API
  recovery unit вместе с остальными systemd unit.
- Блокировка запуска recovery marker возвращает systemd stop status `77`,
  поэтому `Restart=always` не создаёт цикл до ручного восстановления. Desktop
  вычисляет badge provider readiness из состояния adapter, а не из hardcode.

### Статус платформ

- **Linux CLI/TUI 1.4.0:** функциональный релиз; DEB/RPM/source artifacts.
- **Linux Desktop 0.4.0:** функциональный unsigned preview; AppImage, DEB, RPM.
- **Windows Desktop 0.4.0:** только unsigned UI preview; нет Windows Service и
  Wintun, поэтому защита трафика отсутствует.
- **macOS Desktop 0.4.0:** только unsigned UI preview; нет Network Extension и
  privileged helper, поэтому защита трафика отсутствует.
- **Android/iOS:** application package отсутствует. Нет Gradle-проекта,
  `VpnService`, Packet Tunnel backend, подписанного AAB/APK и mobile gate.

### Важные ограничения

- Import/render современных протоколов не означает полноценные
  import/connect/TUN/DNS/routing/rollback/leak-tested lifecycle.
- Схемы Agent Control не являются готовым client-to-client gateway,
  Telegram-контроллером или runtime удалённого управления агентами.
- `verify-service` проверяет только публичную unauthenticated-границу. Решение
  provider может зависеть от региона аккаунта, cookies, browser/device
  signals, политики организации и risk scoring.
- Transition guard не является постоянным always-on kill switch.
- Preview artifacts не подписаны. Hash подтверждает целостность загрузки, но
  не личность издателя.

### Обновление

Linux-пакеты объявляют `nftables` и `python3` вместе с остальными runtime
dependencies. Установка включает hardened API recovery unit перед tunnel/test
recovery. DEB/RPM продолжают безопасно переносить доверенные legacy-команды из
`/usr/local/bin` к package-owned `/usr/bin` с обратимой резервной копией.

Licensed under the GNU Affero General Public License v3.0 or later.
