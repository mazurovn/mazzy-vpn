# Mazzy VPN privacy principles

Copyright © 2026 Nik m ([@mazurovn](https://github.com/mazurovn)).

[Русский](#русский) · [English](#english)

## Русский

Mazzy VPN не содержит телеметрии, рекламных SDK, аналитики и собственной
облачной учётной записи. Профили, ключи, выбранное состояние и журналы остаются
на устройстве пользователя.

- VPN-профили хранятся в `/etc/vpnctl/profiles` и доступны только root.
- Desktop читает только очищенные status/profile caches без ключей, endpoint и
  системных путей.
- Импортированный файл передаётся локальному валидатору и не загружается автору
  проекта или в сторонний сервис.
- Для проверки интернета и публичного VPN IP engine выполняет сетевые запросы к
  адресам, указанным в исходном коде и документации. Эти запросы можно проверить
  самостоятельно в открытом исходном коде.
- Systemd journal может содержать технические имена профилей и сообщения VPN
  runtime. Журнал остаётся в системном хранилище устройства.
- Привилегированные изменения выполняются только после стандартного разрешения
  ОС через `sudo`/`pkexec`.

Оператор VPN-сервера, интернет-провайдер, ОС и сторонние protocol runtimes имеют
собственные правила обработки данных. Mazzy VPN не может заменить их privacy
policy. Перед импортом конфигурации пользователь должен проверить доверие к её
источнику и законность использования VPN в своей юрисдикции.

## English

Mazzy VPN contains no telemetry, advertising SDK, analytics or project-hosted
cloud account. Profiles, keys, selected state and logs remain on the user's
device.

- VPN profiles live in `/etc/vpnctl/profiles` and are root-readable only.
- Desktop reads sanitized status/profile caches without keys, endpoints or
  system paths.
- An imported file is passed to the local validator and is not uploaded to the
  project author or a third-party service.
- To verify Internet access and the public VPN address, the engine makes network
  requests to targets documented in the open source code and project docs.
- The systemd journal may contain technical profile labels and VPN runtime
  messages. It remains in the device's system log store.
- Privileged changes run only after standard OS authorization through
  `sudo`/`pkexec`.

The VPN server operator, Internet provider, operating system and third-party
protocol runtimes have their own data practices. Mazzy VPN cannot replace their
privacy policies. Before importing a configuration, the user must assess its
source and the legality of VPN use in their jurisdiction.
