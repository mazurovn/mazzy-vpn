# Частые вопросы

## Что такое Mazzy VPN?

Это open-source менеджер VPN-профилей и подключений. Он не продаёт VPN-доступ и
не предоставляет серверы. Пользователь импортирует конфигурации своего
провайдера или собственной инфраструктуры.

## Кто автор и какая лицензия?

Автор и сопровождающий — [Nik m (@mazurovn)](https://github.com/mazurovn).
Copyright © 2026 Nik m. Код распространяется по GNU AGPL-3.0-or-later.
Изменённую сборку нельзя выдавать за официальный релиз или публиковать без
соблюдения условий лицензии и сохранения авторства.

## Какие версии сейчас рабочие?

- Опубликованная release line: CLI/TUI 1.3.2 и unsigned Linux Desktop 0.3.2
  preview. Issue #31 закрыт проверенным `glib` backport и чистыми release checks.
- Windows и macOS: только UI preview, нативные VPN backend ещё не готовы.
- Android и iOS: planned native clients; готовых мобильных пакетов пока нет.

Опубликованной считается только версия с tag и страницей в
[GitHub Releases](https://github.com/mazurovn/mazzy-vpn/releases). Проверяйте
[release gates](Releases-and-Roadmap), а не только номер версии или changelog.

## Desktop самодостаточен?

Linux Desktop 0.3 содержит совместимый engine и installer. Предварительная
ручная установка CLI не нужна. Для системного VPN backend и зависимостей
приложение использует стандартное разрешение ОС и после установки повторяет
проверку. Релиз 0.3 опубликован; проверяйте приложенный SHA-256 manifest. До
закрытия `desktop-linux-1.0` пакет остаётся preview.

## Почему `preview-release` пропущен в обычном PR?

Это ожидаемое условие, а не сбой. Release job запускается только при создании
tag `desktop-v*`. Обычный push или PR выполняет тесты и собирает временные
artifacts, но ничего не публикует в Releases.

## Какие профили поддерживаются?

AmneziaWG, WireGuard, OpenVPN и NetworkManager L2TP/IPsec. Поддержка зависит от
платформы: протокол нельзя считать доступным на Windows/macOS/mobile только
потому, что UI умеет распознать файл.

Versioned каталог дополнительно описывает VLESS/REALITY, Hysteria 2, Mieru,
NaiveProxy, TUIC v5, Shadowsocks 2022, Trojan, AnyTLS и ShadowTLS v3. Сейчас для
них реализовано безопасное catalog/URI detection, а unreleased ветка также
классифицирует ограниченный JSON; import/connect ещё не реализованы. Многие из
них являются proxy/transport и требуют отдельного TUN adapter. Точная
матрица: [[Protocol Orchestration]].

## Почему Dashboard видит профили, а экран «Профили» был пустым?

Desktop 0.3.0 строго ожидал новое поле `profile_id` и целиком отвергал cache от
старого CLI 1.2, хотя Dashboard мог прочитать отдельный status cache. В 0.3.2
legacy ID вычисляется совместимо, а недоступный cache отличается от пустого.
После package update проверьте `mazzy-vpn version`, перезапустите приложение и,
если доступ к группе был добавлен впервые, выполните новый login.

## Где находятся профили и логи?

Linux-профили хранятся под root в `/etc/vpnctl/profiles` с ограниченными
правами. Desktop получает только очищенную метаинформацию. Systemd journal
остаётся локальным; Desktop выводит ограниченный объём. Никогда не вставляйте
ключи, пароли, PSK или полный профиль в публичный issue/discussion.

## Есть ли телеметрия?

Нет обязательного аккаунта и нет телеметрии. Проверка endpoint или внешней
доступности создаёт сетевой запрос только при явном тесте либо включённом
пользователем health monitor. См. [Безопасность и приватность](Security-and-Privacy).

## Почему выбран Берлин, а сайт всё равно видит другую страну?

Название выбранного профиля подтверждает только выбор конфигурации. Запустите
`mazzy-vpn verify` или **Проверить VPN** в Desktop: проверка сравнит
interface-bound и default IPv4 egress, два geo provider для того же IP,
настроенный DNS route и возможную IPv6-утечку. `--speed` отдельно добавляет
явный ограниченный 5-МБ sample.

Даже `verified` — это evidence сетевого маршрута в момент проверки, а не
гарантия конкретного сайта. Регион аккаунта, cookies, WebRTC, язык браузера,
геолокация устройства, политика организации и risk scoring сайта могут дать
другой результат.

## Как проверить ping всех локаций и выбрать быструю?

Используйте `mazzy-vpn probe all --timeout 3 --jobs 4` или **Проверить все
локации**. Desktop показывает статус и latency, позволяет сортировать по ping и
подключить самый быстрый из `reachable`. `unknown` для UDP не означает
нерабочий VPN: сервер мог заблокировать ICMP. Авторизацию, handshake и routes
подтверждает только live test с rollback.

## Что делают Doctor и «Исправить»?

Doctor проверяет версии, зависимости, профили, systemd, desired state,
интерфейс и соединение и показывает полный результат. Исправление — отдельное
действие: оно требует подтверждения и стандартной системной авторизации.

## Безопасны ли live tests?

Тест временно меняет активный VPN-маршрут, проверяет профиль и выполняет
transactional rollback. Всё равно сохраните важную сетевую работу и внимательно
прочитайте подтверждение перед тестом.

## Как сообщить о проблеме?

Сначала запустите `mazzy-vpn doctor`, `mazzy-vpn self-test --offline` и
посмотрите ограниченный журнал. Создайте
[issue](https://github.com/mazurovn/mazzy-vpn/issues) или тему в
[Discussions](https://github.com/mazurovn/mazzy-vpn/discussions), указав ОС,
версию, протокол, шаги и очищенный вывод. Секреты отправляйте только по процессу
из `SECURITY.md`.

## Как предложить и выбрать следующую функцию?

Предложения публикуются в
[Ideas](https://github.com/mazurovn/mazzy-vpn/discussions/categories/ideas), а
голосования — в
[Polls](https://github.com/mazurovn/mazzy-vpn/discussions/categories/polls).
Результат помогает выбрать порядок пользовательских функций. Обязательные
security, platform и release gates остаются обязательными независимо от числа
голосов.

---

<a id="english"></a>

# Frequently asked questions

## What is Mazzy VPN?

It is an open-source VPN profile and connection manager. It does not sell VPN
access or provide servers. Users import configurations from their provider or
their own infrastructure.

## Who is the author and what is the license?

The author and maintainer is
[Nik m (@mazurovn)](https://github.com/mazurovn). Copyright © 2026 Nik m.
The code is licensed under GNU AGPL-3.0-or-later. A modified build must not be
presented as an official release or distributed without following the license
and preserving authorship.

## Which versions work today?

- Published release line: CLI/TUI 1.3.2 and the unsigned Linux Desktop 0.3.2
  preview. Issue #31 is closed with a verified `glib` backport and clean release
  checks.
- Windows and macOS: UI preview only; native VPN backends are not complete.
- Android and iOS: planned native clients; no working mobile packages yet.

Only a version with a tag and a
[GitHub Release](https://github.com/mazurovn/mazzy-vpn/releases) is published.
Use the [release gates](Releases-and-Roadmap#english), not a version number or
changelog entry alone, to determine readiness.

## Is Desktop self-contained?

Linux Desktop 0.3 bundles a compatible engine and installer, so no prior manual
CLI install is required. System VPN backends and dependencies use standard OS
authorization and are checked again afterwards. The package remains preview
until `desktop-linux-1.0` passes. Release 0.3 is published; verify downloads
with its attached SHA-256 manifest.

## Why is `preview-release` skipped on an ordinary PR?

This is an expected condition, not a failure. The release job runs only when a
`desktop-v*` tag is created. An ordinary push or PR runs tests and builds
temporary artifacts but publishes nothing to Releases.

## Which profiles are supported?

AmneziaWG, WireGuard, OpenVPN and NetworkManager L2TP/IPsec. Support is
platform-specific: recognizing a file in the UI does not make that protocol
functional on Windows, macOS or mobile.

The versioned catalog additionally describes VLESS/REALITY, Hysteria 2, Mieru,
NaiveProxy, TUIC v5, Shadowsocks 2022, Trojan, AnyTLS and ShadowTLS v3. Safe
catalog/URI detection exists now, and the unreleased branch also classifies
bounded JSON; import/connect does not. Many entries are proxy/transport
protocols and require a separate TUN adapter. See
[[Protocol Orchestration]] for the exact matrix.

## Why did Dashboard see profiles while the Profiles screen was empty?

Desktop 0.3.0 strictly required the new `profile_id` field and rejected the
entire cache produced by an older 1.2 CLI, while Dashboard could still read a
separate status cache. Version 0.3.2 derives a compatible legacy ID and
distinguishes an unavailable cache from an empty one. After package update,
check `mazzy-vpn version`, restart the app and start a new login session if
group access was just granted.

## Where are profiles and logs stored?

Linux profiles are root-protected under `/etc/vpnctl/profiles`. Desktop receives
sanitized metadata only. The systemd journal remains local and Desktop reads a
bounded amount. Never paste keys, passwords, PSKs or a complete profile into a
public issue or discussion.

## Is there telemetry?

There is no mandatory account and no telemetry. An endpoint/connectivity probe
makes a network request only during an explicit test or a health monitor the
user enabled. See [Security and privacy](Security-and-Privacy#english).

## Why does a Berlin profile still look like another country to a site?

A selected profile name proves only which configuration was chosen. Run
`mazzy-vpn verify` or **Verify VPN** in Desktop. It compares interface-bound
and default IPv4 egress, two location providers for that exact IP, configured
DNS routing and a potential IPv6 leak. `--speed` separately adds an explicit
bounded five-megabyte sample.

Even `verified` is time-of-check network evidence, not a promise about a
specific site. Account region, cookies, WebRTC, browser language, device
location, organization policy and site risk scoring may still affect results.

## How do I ping every location and choose a fast one?

Use `mazzy-vpn probe all --timeout 3 --jobs 4` or **Check all locations**.
Desktop shows status/latency, sorts by ping and can connect the fastest
`reachable` entry. UDP `unknown` does not prove failure: the server may block
ICMP. Only a transactional live test proves authentication, handshake and
routes.

## What do Doctor and Repair do?

Doctor checks versions, dependencies, profiles, systemd, desired state,
interface and connectivity and shows the complete result. Repair is a separate
action that requires confirmation and standard system authorization.

## Are live tests safe?

A test temporarily changes the active VPN route, validates the profile and
performs transactional rollback. Save important network work and read the
confirmation before running it.

## How do I report a problem?

Run `mazzy-vpn doctor`, `mazzy-vpn self-test --offline` and inspect bounded
logs first. Open an [issue](https://github.com/mazurovn/mazzy-vpn/issues) or a
[Discussion](https://github.com/mazurovn/mazzy-vpn/discussions) with OS,
version, protocol, reproduction steps and redacted output. Use the private
process in `SECURITY.md` for secrets.

## How do I propose and choose the next feature?

Post proposals in
[Ideas](https://github.com/mazurovn/mazzy-vpn/discussions/categories/ideas) and
vote in
[Polls](https://github.com/mazurovn/mazzy-vpn/discussions/categories/polls).
Results help order user-facing work. Mandatory security, platform and release
gates remain mandatory regardless of vote count.
