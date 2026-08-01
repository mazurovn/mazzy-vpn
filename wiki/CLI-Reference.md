# Справочник CLI

| Команда | Назначение |
|---|---|
| `mazzy-vpn` | интерактивное TUI |
| `mazzy-vpn dashboard` | терминальный dashboard |
| `mazzy-vpn status` | текстовый статус |
| `mazzy-vpn status --json` | очищенный структурированный статус |
| `mazzy-vpn status --api-json` | raw response envelope local API v1 |
| `mazzy-vpn profiles --api-json` | каталог с opaque ID без engine filenames |
| `mazzy-vpn protocols list --json` | versioned каталог 13 протоколов и readiness |
| `mazzy-vpn protocols diagnose --json` | candidate runtimes и готовность backend |
| `... \| mazzy-vpn protocols detect --stdin --json` | redacted распознавание share URI |
| `... \| mazzy-vpn planner evaluate --stdin --json` | draft PR #43: deterministic dry-run rank; ещё не в stable 1.3.2 |
| `mazzy-vpn quick` | подключить сохранённый default через local API |
| `mazzy-vpn connect PROTOCOL PROFILE` | выбрать и подключить профиль |
| `mazzy-vpn reconnect` | безопасно перезапустить выбранный туннель |
| `mazzy-vpn disconnect` | записать `DESIRED=down` и отключить |
| `mazzy-vpn list [PROTOCOL]` | список профилей |
| `mazzy-vpn validate all` | проверить формат, директивы и права |
| `mazzy-vpn probe all --timeout 3 --jobs 4 [--json]` | массовый DNS/ICMP/TCP probe с latency и active |
| `mazzy-vpn verify [--speed] [--json]` | фактические egress/geo/DNS/IPv6 и optional 5-МБ sample |
| `sudo mazzy-vpn test ...` | транзакционный тест одного профиля |
| `sudo mazzy-vpn test-all all` | тест всех профилей с rollback |
| `sudo mazzy-vpn emergency` | найти первый реально работающий профиль |
| `sudo mazzy-vpn diagnose` | маршрут, DNS, service, interface, handshake, IP |
| `sudo mazzy-vpn doctor [--fix]` | зависимости, units, профили, repair |
| `mazzy-vpn self-test [--offline\|--live]` | объединённая самодиагностика |
| `sudo mazzy-vpn autostart on\|off` | автоподключение |
| `mazzy-vpn import-dir DIR [--dry-run]` | распознавание и импорт папки |
| `mazzy-vpn init-config-dir DIR` | создать структуру каталогов |
| `mazzy-vpn language CODE` | изменить язык |
| `mazzy-vpn logs [-f]` | журнал сервиса |

Подключение сейчас реализовано для `amneziawg`/`awg`, `wireguard`/`wg`,
`openvpn`/`ovpn`, `l2tp`. Каталог 13 протоколов не означает 13 готовых backend.

---

<a id="english"></a>

# CLI reference

| Command | Purpose |
|---|---|
| `mazzy-vpn` | interactive TUI |
| `mazzy-vpn dashboard` | terminal dashboard |
| `mazzy-vpn status` | text status |
| `mazzy-vpn status --json` | sanitized structured status |
| `mazzy-vpn status --api-json` | raw local API v1 response envelope |
| `mazzy-vpn profiles --api-json` | opaque-ID catalog without engine filenames |
| `mazzy-vpn protocols list --json` | versioned 13-protocol catalog and readiness |
| `mazzy-vpn protocols diagnose --json` | candidate runtimes and backend readiness |
| `... \| mazzy-vpn protocols detect --stdin --json` | redacted share-URI detection |
| `... \| mazzy-vpn planner evaluate --stdin --json` | draft PR #43: deterministic dry-run rank; not in stable 1.3.2 yet |
| `mazzy-vpn quick` | connect the saved default through the local API |
| `mazzy-vpn connect PROTOCOL PROFILE` | select and connect a profile |
| `mazzy-vpn reconnect` | safely restart the selected tunnel |
| `mazzy-vpn disconnect` | write `DESIRED=down` and disconnect |
| `mazzy-vpn list [PROTOCOL]` | list profiles |
| `mazzy-vpn validate all` | validate format, directives and permissions |
| `mazzy-vpn probe all --timeout 3 --jobs 4 [--json]` | bounded batch DNS/ICMP/TCP probe with latency and active state |
| `mazzy-vpn verify [--speed] [--json]` | actual egress/geo/DNS/IPv6 and optional 5 MB sample |
| `sudo mazzy-vpn test ...` | transactional single-profile test |
| `sudo mazzy-vpn test-all all` | test all profiles with rollback |
| `sudo mazzy-vpn emergency` | find the first working profile |
| `sudo mazzy-vpn diagnose` | route, DNS, service, interface, handshake and IP |
| `sudo mazzy-vpn doctor [--fix]` | dependencies, units, profiles and repair |
| `mazzy-vpn self-test [--offline\|--live]` | combined self-diagnostics |
| `sudo mazzy-vpn autostart on\|off` | automatic connection |
| `mazzy-vpn import-dir DIR [--dry-run]` | folder recognition and import |
| `mazzy-vpn init-config-dir DIR` | create a folder structure |
| `mazzy-vpn language CODE` | change language |
| `mazzy-vpn logs [-f]` | service log |

Connection is currently implemented for `amneziawg`/`awg`, `wireguard`/`wg`,
`openvpn`/`ovpn` and `l2tp`. A 13-entry catalog is not 13 working backends.
