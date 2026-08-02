# Повторный аудит protocol runtime и agent control — 2026-08-02

## Проверенная область

Проверены protocol registry и detector, managed profile boundary, Linux
import/storage, sing-box renderer, package payload, Mieru и NaiveProxy upstream,
ShadowTLS composition, два сетевых слоя продукта, аналоги iroh, Web/CLI/Telegram
ingress, release claims и platform roadmap.

Источники истины:

- `protocols/v1/registry.json` — пользовательские egress-протоколы и status;
- `protocols/v1/managed-profile.schema.json` — закрытый neutral profile;
- `runtime/v1/adapter-registry.json` — engine graph и release gates;
- `agent-control/v1/registry.json` — reverse agent-control transports;
- `docs/capabilities.json` — platform release gates.

## Findings и исправления

### High: detection ошибочно воспринимался как import/connect

У девяти современных entries работали registry и redacted classification, но
не существовало закрытого формата хранения. Добавлены protocol-specific schema,
строгая проверка одного JSON до 64 КиБ и `managed-import`. Duplicate keys,
symlink source, control/bidi text, insecure TLS, произвольные listeners, file
paths, route rules и marks отклоняются. Import повторно проверяет snapshot под
блокировкой, атомарно создаёт `PROTOCOL/PROFILE_ID.json`, использует `700/600`,
не перезаписывает без `--force` и не возвращает endpoint/credential.

Статус import повышен только до `partial`: прямой conversion всех vendor share
formats и platform keystore ещё отсутствуют. Connection остаётся `planned`.

### High: произвольный sing-box JSON расширял root attack surface

Добавлен `mazzy-sing-box-adapter`, который никогда не принимает готовый engine
graph. Для VLESS, Hysteria 2, TUIC, Shadowsocks 2022, Trojan и AnyTLS он
синтезирует ровно один fixed TUN, proxy-routed DoH, DNS hijack и final proxy
route. Пользователь не управляет listener, interface name, path, bind, mark или
route rules; TLS verification нельзя отключить. Runtime config создаётся `0600`
и удаляется после process exit. Renderer v1 требует literal IPv4 proxy/DoH
bootstrap, иначе отказывает из-за риска DNS bootstrap leak.

Это config-render foundation, а не production connection: sing-box 1.13.12 не
поставляется, service lifecycle и network tests ещё не подключены.

### High: Mieru и NaiveProxy требуют двух процессов

Официальный Mieru 3.32.0 поддерживает foreground `mieru run` и изолированный
`MIERU_CONFIG_JSON_FILE`; глобальный `apply config` не нужен. NaiveProxy
148.0.7778.96-5 принимает отдельный JSON и поднимает loopback SOCKS. В обоих
случаях full-tunnel требует TUN-to-SOCKS процесса. Supervisor обязан считать
оба процесса одной транзакцией: crash любого закрывает второй, маршруты/DNS
откатываются, engine endpoint исключён из TUN.

Managed validation/import для Mieru и NaiveProxy реализованы. Renderer,
supervisor и lifecycle намеренно остаются `planned`.

### High: ShadowTLS не является самостоятельным VPN outbound

ShadowTLS v3 — TCP camouflage transport. Без typed inner proxy его нельзя
превратить в L3 egress. Validator/import существуют, но renderer fail-closed.
Будущий profile должен ссылаться на opaque ID совместимого inner proxy, а не
принимать вложенный произвольный JSON.

### High: iroh нельзя смешивать с VPN protocol registry

iroh, libp2p, WebRTC, WebTransport, Tailscale/Headscale и reverse WSS решают
обратную доставку команд к агентам. Они не меняют системный egress сами по себе.
Поэтому второй слой имеет отдельные schema/envelope/command/transport registry.
Это исключает ошибку, при которой отказ agent relay мог бы мутировать VPN route
или VPN credential оказался бы в control broker.

### Medium: Telegram Bot не является first-party E2EE transport

Bot ingress ограничен read-only/low-risk capability. Prompt, artifact,
credential и high-risk approval идут только через paired first-party Web или
Telegram Mini App с тем же signed encrypted envelope. В command v1 нет
arbitrary `shell.exec`; agent повторно проверяет actor, risk, TTL и confirmation.

## Сравнение с Happy и Claude Bridge

Happy полезен durable sequence/update sync, optimistic versions, E2EE opaque
payload и first-party Web/mobile surfaces. Claude Bridge полезен pinned LAN WSS,
mDNS discovery, allowlisted peer IDs и iroh QUIC direct/relay sidecar. Mazzy
объединяет эти свойства под transport-neutral command protocol и добавляет
libp2p, WebRTC, WebTransport, explicit mesh и reverse-WSS fallback. Ни один
transport не получает plaintext command или право выполнить shell text.

## Почему нового production release ещё нет

Текущие изменения нельзя выпускать как «13 рабочих протоколов» до выполнения
всех hard gates из runtime registry:

1. checksum-pinned engine assets, reproducible source, provenance и SBOM;
2. service integration и единая ownership-модель TUN/DNS/routes/firewall;
3. loop prevention и explicit engine endpoint bypass;
4. start/health/stop/crash transaction с восстановлением предыдущего tunnel;
5. реальные IPv4, IPv6, DNS, bootstrap, process-crash и leak tests;
6. Windows Service/Wintun backend и signed installer;
7. macOS Network Extension, signing и notarization;
8. Android `VpnService`, Keystore, embedded engines и real-device tests;
9. независимый security review agent gateway, pairing, E2EE и relay abuse.

Linux stable 1.3.2 и Desktop preview 0.3.2 остаются последними опубликованными
линиями. Windows/macOS packages являются UI preview, Android source отсутствует.
Issue #31 не блокирует релиз: он закрыт. Текущие blockers — новые lifecycle,
platform backends и supply chain.

## Проверяемые регрессии

- schema/validator protocol sets совпадают для всех девяти profiles;
- шесть generated sing-box configs имеют закрытый graph и mode `0600`;
- Mieru, NaiveProxy и ShadowTLS fail closed на render;
- managed import проверяет dry-run, conflict, force replacement и symlink;
- runtime registry запрещает `implemented` lifecycle без integration tests;
- agent-control registry проверяет E2EE, anti-replay и channel risk policy;
- source installer, DEB/RPM/AppImage payload gates требуют schemas и adapters.

Фактический verification run текущего дерева:

- `./tests/run.sh`: `84/84`;
- Rust unit tests: `25/25`, Clippy с `-D warnings` для owned crate;
- npm audit: `0 vulnerabilities`;
- ShellCheck, JSON contracts и public secret/history audit: успешно;
- официальный sing-box 1.13.12 linux-amd64 загружен во временный каталог,
  archive SHA-256 `1540533adb3df24f5ad5f14b5c7ca3dbc2401b10a1c1eb278fcadcada47ec6c4`
  совпал с GitHub release metadata, все шесть generated configs прошли
  `sing-box check`;
- AppImage/DEB/RPM пересобраны, unpacked payload/corresponding source,
  dependencies, lifecycle и Xvfb GUI launch audit прошли;
- `cargo-deny` в текущем host PATH отсутствует, поэтому локально повторно не
  запускался; Rust dependency graph в этом срезе не менялся.

Скриншоты Desktop не менялись: в этом срезе нет нового UI. Предыдущие 12
скриншотов остаются актуальными для опубликованного Desktop 0.3.2; изображать в
них неработающие modern-protocol connect controls было бы ложным claim.
