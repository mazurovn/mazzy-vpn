# Справочник CLI

| Команда | Назначение |
|---|---|
| `mazzy-vpn` | интерактивное TUI |
| `mazzy-vpn dashboard` | терминальный dashboard |
| `mazzy-vpn status` | текстовый статус |
| `mazzy-vpn status --json` | очищенный структурированный статус |
| `mazzy-vpn status --api-json` | raw response envelope local API v1 |
| `mazzy-vpn profiles --api-json` | каталог с opaque ID без engine filenames |
| `sudo mazzy-vpn quick` | подключить сохранённый default |
| `sudo mazzy-vpn connect PROTOCOL PROFILE` | выбрать и подключить профиль |
| `sudo mazzy-vpn reconnect` | безопасно перезапустить выбранный туннель |
| `sudo mazzy-vpn disconnect` | записать `DESIRED=down` и отключить |
| `mazzy-vpn list [PROTOCOL]` | список профилей |
| `mazzy-vpn validate all` | проверить формат, директивы и права |
| `mazzy-vpn probe all --timeout 3 --jobs 4 [--json]` | массовый DNS/ICMP/TCP probe с latency и active |
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

Протоколы: `amneziawg`/`awg`, `wireguard`/`wg`, `openvpn`/`ovpn`, `l2tp`.

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
| `sudo mazzy-vpn quick` | connect the saved default |
| `sudo mazzy-vpn connect PROTOCOL PROFILE` | select and connect a profile |
| `sudo mazzy-vpn reconnect` | safely restart the selected tunnel |
| `sudo mazzy-vpn disconnect` | write `DESIRED=down` and disconnect |
| `mazzy-vpn list [PROTOCOL]` | list profiles |
| `mazzy-vpn validate all` | validate format, directives and permissions |
| `mazzy-vpn probe all --timeout 3 --jobs 4 [--json]` | bounded batch DNS/ICMP/TCP probe with latency and active state |
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

Protocols: `amneziawg`/`awg`, `wireguard`/`wg`, `openvpn`/`ovpn`, `l2tp`.
