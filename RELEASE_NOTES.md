# Mazzy VPN 1.3.0 / Desktop 0.3.0

Copyright (C) 2026 Nik m
([@mazurovn](https://github.com/mazurovn)).

> Release-candidate text. The release is published only after tags `v1.3.0`
> and `desktop-v0.3.0`, their GitHub Release pages and required checks exist.

## English

Mazzy VPN 1.3 strengthens the Linux AI-ready VPN control plane for long-running
human, AI-agent, learning, video, open-web and work sessions. Desktop 0.3
remains a Linux preview: it is functional, but it is not yet a signed Desktop
1.0 product.

### Highlights

- `mazzy-vpn verify [--speed] [--json]` and API v1
  `tests.verify-egress`;
- actual interface-bound versus default IPv4 comparison;
- two distinct geolocation providers, both required to report the exact
  observed egress IP and agree on country before `verified`;
- configured full-tunnel DNS and potential IPv6-leak signals;
- explicit bounded five-megabyte speed sample, never a background transfer;
- whole-list location DNS/ICMP/TCP checks with bounded concurrency, latency,
  `reachable`/`unknown`/`unreachable`/`invalid` and active-tunnel state;
- Desktop sorting by ping/status/name and connect-fastest from measured
  reachable entries;
- expanded tray navigation and controls, clickable retained event detail and
  clearer ON/OFF service state;
- strict typed Desktop response validation and bounded compatibility helpers;
- auto health policy that detects confirmed default-egress mismatch only for
  declared full-tunnel profiles; split-tunnel profiles are not forced into
  full-tunnel recovery;
- updated architecture, privacy, installation, Wiki, platform roadmap and
  AI-ready product description.

### What `verified` means

`verified` is time-of-check network evidence: the managed tunnel is active,
default and interface IPv4 egress match, two validated geo providers agree,
no potential IPv6 leak was observed, configured DNS routing is full-tunnel and
there are no findings. It is not a promise that a particular AI provider,
website or video platform accepts the session. Account region, organization
policy, cookies, browser language, WebRTC, device location and provider-side
risk scoring remain external.

### Platform status

- **Linux CLI/TUI 1.3.0:** release candidate.
- **Linux Desktop 0.3.0:** functional unsigned preview; AppImage, DEB and RPM.
- **Windows/macOS:** unsigned UI previews without native VPN backends; do not
  use them for traffic protection.
- **Android/iOS:** planned native clients; no application packages are in this
  release.

### Known blockers

- most privileged Desktop domains still use the typed `pkexec` compatibility
  adapter instead of a native service with peer identity;
- no signed SBOM/provenance/update chain or reproducible clean-builder proof;
- no complete clean-device distro, sleep/resume, network-change, loss/jitter,
  crash, upgrade/rollback and soak matrix;
- German, Chinese, Japanese and Korean still use English fallback on parts of
  the extended Desktop screens;
- Windows Service/Wintun, macOS Network Extension, Android `VpnService` and iOS
  Packet Tunnel backends are not implemented.

## Русский

Mazzy VPN 1.3 усиливает Linux AI-ready VPN-контур для долгих сессий людей,
AI-агентов, обучения, видео, открытого веба и рабочих систем. Desktop 0.3
остаётся Linux preview: он функционален, но ещё не является подписанным
Desktop 1.0 продуктом.

### Основные изменения

- `mazzy-vpn verify [--speed] [--json]` и API v1
  `tests.verify-egress`;
- сравнение фактических interface-bound и default IPv4;
- два разных geo provider: для `verified` оба должны сообщить точный
  наблюдаемый egress IP и согласовать страну;
- signals настроенного full-tunnel DNS и потенциальной IPv6-утечки;
- только явный bounded speed sample 5 МБ, без фонового трафика;
- массовая DNS/ICMP/TCP-проверка всего списка с bounded concurrency, latency,
  состояниями `reachable`/`unknown`/`unreachable`/`invalid` и active tunnel;
- сортировка Desktop по ping/status/name и connect-fastest только среди
  измеренных `reachable`;
- расширенный tray, прямое открытие экранов, кликабельные детали событий и
  понятное текущее ON/OFF состояние служб;
- strict typed validation ответов Desktop и bounded compatibility helpers;
- auto health policy обнаруживает подтверждённый default-egress mismatch для
  профилей, объявляющих full tunnel, но не ломает split-tunnel;
- обновлены архитектура, privacy, установка, Wiki, platform roadmap и
  AI-ready описание продукта.

### Что означает `verified`

Это evidence сети в момент проверки: managed tunnel активен, default и
interface IPv4 совпадают, два валидированных geo provider согласны,
потенциальная IPv6-утечка не обнаружена, настроен full-tunnel DNS и нет
findings. Это не обещание, что конкретный AI provider, сайт или видео-платформа
примет сессию. Регион аккаунта, политика организации, cookies, язык браузера,
WebRTC, геолокация устройства и provider-side risk scoring остаются внешними.

### Статус платформ

- **Linux CLI/TUI 1.3.0:** release candidate.
- **Linux Desktop 0.3.0:** функциональный неподписанный preview; AppImage, DEB
  и RPM.
- **Windows/macOS:** неподписанные UI preview без native VPN backend; не
  используйте их для защиты трафика.
- **Android/iOS:** planned native clients; application packages в релизе нет.

### Известные blockers

- большинство privileged Desktop domains всё ещё использует typed `pkexec`
  compatibility adapter вместо native service с peer identity;
- нет signed SBOM/provenance/update chain и доказанной clean-builder
  reproducibility;
- нет полной clean-device distro, sleep/resume, network-change, loss/jitter,
  crash, upgrade/rollback и soak matrix;
- части расширенных Desktop экранов для немецкого, китайского, японского и
  корейского используют English fallback;
- Windows Service/Wintun, macOS Network Extension, Android `VpnService` и iOS
  Packet Tunnel backends не реализованы.

## Safety / Безопасность

Live tests temporarily change routes and use transactional rollback. Save
important network work before starting one. Operational profiles, keys and
credentials are intentionally absent from release artifacts. Current preview
artifacts are unsigned; an unsigned hash is not proof of publisher identity.

Live tests временно меняют маршруты и используют transactional rollback.
Сохраните важную сетевую работу до запуска. Рабочие профили, ключи и credentials
намеренно отсутствуют в release artifacts. Текущие preview artifacts не
подписаны; неподписанный hash не доказывает издателя.

Licensed under the GNU Affero General Public License v3.0 or later.
