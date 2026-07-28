# Mazzy VPN privacy principles

Copyright © 2026 Nik m ([@mazurovn](https://github.com/mazurovn)).

[Русский](#русский) · [English](#english)

## Русский

Mazzy VPN не содержит телеметрии, рекламных SDK, аналитики и собственной
облачной учётной записи. Профили, ключи, выбранное состояние и журналы остаются
на устройстве пользователя.

- VPN-профили хранятся в `/etc/vpnctl/profiles` и доступны только root.
- Desktop читает только строго проверенные status/profile caches без ключей,
  endpoint и системных путей.
- Импортированный файл передаётся локальному валидатору и не загружается автору
  проекта или в сторонний сервис.
- Для проверки интернета и публичного VPN IP engine обращается к `api.ipify.org`
  и `api6.ipify.org`. Явно запущенная проверка фактической локации обращается
  через VPN-интерфейс к `ipapi.co` и `ipwho.is`; эти сервисы получают публичный
  VPN IP и стандартные данные HTTPS-запроса.
- Включённый health monitor проверяет IPv4 через VPN-интерфейс. Для профиля,
  который явно запрашивает full tunnel, auto policy также сравнивает default
  IPv4 egress. Отсутствие только default-egress observation не запускает
  recovery. Но если оба ограниченных connectivity observer не отвечают,
  повторный failure всё ещё запускает recovery. Geo и speed providers в фоне
  не вызываются.
- OpenVPN использует DNS, полученный от сервера/профиля. Публичный resolver
  молча не подставляется; администратор может явно задать
  `VPNCTL_OPENVPN_FALLBACK_DNS` для профиля без DNS.
- Проверка скорости не запускается в фоне. Только после явного нажатия
  `Проверить + скорость` загружается ограниченный 5 MB sample с
  `speed.cloudflare.com` через VPN-интерфейс. В результат не включается история
  посещений или содержимое VPN-профиля.
- URL диагностических сервисов задаются открытыми переменными окружения и
  видны в исходном коде. Организация может заменить их собственными
  совместимыми endpoints.
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
- To verify Internet access and the public VPN address, the engine contacts
  `api.ipify.org` and `api6.ipify.org`. An explicitly started actual-location
  check contacts `ipapi.co` and `ipwho.is` through the VPN interface; those
  services receive the public VPN address and standard HTTPS request metadata.
- An enabled health monitor checks IPv4 through the VPN interface. For a
  profile that explicitly requests a full tunnel, the automatic policy also
  compares default IPv4 egress. A missing default-egress observation alone
  does not trigger recovery. If both bounded connectivity observers fail,
  however, repeated failures still trigger recovery. Geo and speed providers
  are never contacted by the background monitor.
- OpenVPN uses DNS received from the server/profile. No public resolver is
  silently substituted; an administrator may set
  `VPNCTL_OPENVPN_FALLBACK_DNS` explicitly for profiles that provide no DNS.
- Speed is never sampled in the background. Only an explicit
  `Verify + speed` action downloads a bounded 5 MB sample from
  `speed.cloudflare.com` through the VPN interface. No browsing history or VPN
  profile content is included.
- Diagnostic URLs are visible as environment-overridable settings in the open
  source. An organization can replace them with compatible internal endpoints.
- The systemd journal may contain technical profile labels and VPN runtime
  messages. It remains in the device's system log store.
- Privileged changes run only after standard OS authorization through
  `sudo`/`pkexec`.

The VPN server operator, Internet provider, operating system and third-party
protocol runtimes have their own data practices. Mazzy VPN cannot replace their
privacy policies. Before importing a configuration, the user must assess its
source and the legality of VPN use in their jurisdiction.
