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

- CLI/TUI 1.2: Linux.
- Desktop 0.2: функциональный Linux preview.
- Windows и macOS: только UI preview, нативные VPN backend ещё не готовы.
- Android и iOS: planned native clients; готовых мобильных пакетов пока нет.

Проверяйте [release gates](Releases-and-Roadmap), а не только номер версии.

## Desktop самодостаточен?

Linux Desktop 0.2 содержит совместимый engine и installer. Предварительная
ручная установка CLI не нужна. Для системного VPN backend и зависимостей
приложение использует стандартное разрешение ОС и после установки повторяет
проверку. До закрытия `desktop-linux-1.0` пакет остаётся preview.

## Какие профили поддерживаются?

AmneziaWG, WireGuard, OpenVPN и NetworkManager L2TP/IPsec. Поддержка зависит от
платформы: протокол нельзя считать доступным на Windows/macOS/mobile только
потому, что UI умеет распознать файл.

## Где находятся профили и логи?

Linux-профили хранятся под root в `/etc/vpnctl/profiles` с ограниченными
правами. Desktop получает только очищенную метаинформацию. Systemd journal
остаётся локальным; Desktop выводит ограниченный объём. Никогда не вставляйте
ключи, пароли, PSK или полный профиль в публичный issue/discussion.

## Есть ли телеметрия?

Нет обязательного аккаунта и нет телеметрии. Проверка endpoint или внешней
доступности создаёт сетевой запрос только при явном тесте либо включённом
пользователем health monitor. См. [Безопасность и приватность](Security-and-Privacy).

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

- CLI/TUI 1.2: Linux.
- Desktop 0.2: functional Linux preview.
- Windows and macOS: UI preview only; native VPN backends are not complete.
- Android and iOS: planned native clients; no working mobile packages yet.

Use the [release gates](Releases-and-Roadmap#english), not a version number
alone, to determine readiness.

## Is Desktop self-contained?

Linux Desktop 0.2 bundles a compatible engine and installer, so no prior manual
CLI install is required. System VPN backends and dependencies use standard OS
authorization and are checked again afterwards. The package remains preview
until `desktop-linux-1.0` passes.

## Which profiles are supported?

AmneziaWG, WireGuard, OpenVPN and NetworkManager L2TP/IPsec. Support is
platform-specific: recognizing a file in the UI does not make that protocol
functional on Windows, macOS or mobile.

## Where are profiles and logs stored?

Linux profiles are root-protected under `/etc/vpnctl/profiles`. Desktop receives
sanitized metadata only. The systemd journal remains local and Desktop reads a
bounded amount. Never paste keys, passwords, PSKs or a complete profile into a
public issue or discussion.

## Is there telemetry?

There is no mandatory account and no telemetry. An endpoint/connectivity probe
makes a network request only during an explicit test or a health monitor the
user enabled. See [Security and privacy](Security-and-Privacy#english).

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
