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
- Явная `verify-service` отправляет только HEAD-запросы без credentials,
  redirects, cookies, body и proxy на фиксированные allowlisted endpoints
  NotebookLM и OpenAI через выбранный VPN-интерфейс. Возвращаются только
  enum-классификация и ограниченный HTTP status; URL, headers, IP, account и
  body не сохраняются. Проверка не запускается в фоне, не влияет на
  health/planner и не доказывает доступ учётной записи или к контенту.
- URL общих geo/speed-проверок видны в исходном коде и могут быть явно
  переопределены оператором. Security-классификатор `verify-service` намеренно
  использует только фиксированные allowlisted NotebookLM/OpenAI endpoints:
  окружение не может перенаправить эту проверку на произвольный адрес.
- Systemd journal может содержать технические имена профилей и сообщения VPN
  runtime. Журнал остаётся в системном хранилище устройства.
- Desktop ведёт отдельный пользовательский lifecycle log в стандартном app log
  directory (`Mazzy VPN Desktop.log`). `KeepOne` сохраняет только один активный
  файл размером до `1 000 000` байт. В него попадают версия/OS, повторный
  запуск, тип и результат операции, число профилей и агрегаты
  probe/verify/updater; имена профилей, endpoint, credentials, конфигурации и IP
  не записываются. После остановки Desktop файл можно удалить вручную.
- Привилегированные изменения выполняются только после стандартного разрешения
  ОС через `sudo`/`pkexec`.
- Desktop по умолчанию один раз при запуске запрашивает фиксированный
  `desktop-updater` asset на GitHub Releases. Запрос раскрывает GitHub обычные
  сетевые метаданные, включая исходящий IP. Версия, профили, ключи, диагностика
  и история действий проекту не отправляются. Проверку можно отключить в
  Settings; загрузка и установка требуют отдельного подтверждения в диалоге.

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
- Explicit `verify-service` sends credential-free HEAD requests without
  redirects, cookies, bodies or proxy inheritance to fixed NotebookLM/OpenAI
  allowlisted endpoints through the selected VPN interface. Only enum
  classification and a bounded HTTP status are returned; URLs, headers,
  addresses, accounts and bodies are not persisted. It never runs in the
  background, affects neither health nor planner, and does not prove account or
  content access.
- General geo/speed diagnostic URLs are visible in the source and may be
  explicitly overridden by an operator. The `verify-service` security
  classifier intentionally uses fixed allowlisted NotebookLM/OpenAI endpoints;
  the environment cannot redirect those probes to an arbitrary destination.
- The systemd journal may contain technical profile labels and VPN runtime
  messages. It remains in the device's system log store.
- Desktop also keeps a separate user-owned lifecycle log in the standard
  platform app log directory (`Mazzy VPN Desktop.log`). `KeepOne` retains one
  active file up to `1,000,000` bytes. It contains version/OS, repeated-launch,
  operation type/outcome, profile-count and probe/verify/updater aggregates;
  profile names, endpoints, credentials, configurations and IP addresses are
  not written. The file can be deleted manually while Desktop is stopped.
- Privileged changes run only after standard OS authorization through
  `sudo`/`pkexec`.
- By default, Desktop requests the fixed `desktop-updater` GitHub Releases asset
  once at startup. GitHub receives ordinary network metadata, including the
  egress address. Profiles, keys, diagnostics and action history are not sent
  to the project. The check can be disabled in Settings; download and install
  require separate confirmation in the modal dialog.

The VPN server operator, Internet provider, operating system and third-party
protocol runtimes have their own data practices. Mazzy VPN cannot replace their
privacy policies. Before importing a configuration, the user must assess its
source and the legality of VPN use in their jurisdiction.
