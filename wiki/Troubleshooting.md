# Устранение проблем

## VPN не подключился

```bash
sudo mazzy-vpn diagnose
sudo mazzy-vpn doctor
systemctl status vpnctl.service
mazzy-vpn logs
```

Проверьте endpoint:

```bash
mazzy-vpn validate all
mazzy-vpn probe all --timeout 3 --jobs 4
```

## OpenVPN: Too many connections

Это отказ сервера/аккаунта, а не ошибка парсера. Остановите лишние сессии,
подождите, пока сервер забудет старую сессию, и повторите. Mazzy VPN переводит
ошибку в retryable status и не оставляет тестовый туннель вместо рабочего.

## Dashboard не показывает статус

```bash
sudo mazzy-vpn _refresh-dashboard-cache
mazzy-vpn status --json
ls -l /run/mazzy-vpn/status.json
systemctl status vpnctl-health.timer
```

## Нет tray

Проверьте поддержку StatusNotifier/AppIndicator в desktop environment. На Linux
используйте правую кнопку. VPN продолжает работать под systemd даже без GUI.

## Конфликт с другим VPN

Не держите два TUN/WireGuard-like VPN одновременно. Mazzy VPN умеет безопасно
останавливать распознанный AdGuard/legacy fallback перед своим туннелем и
восстанавливать его только при rollback. Обычная работа fallback не требует.

## NotebookLM или OpenAI не открывается через рабочий VPN

Сначала отделите обычный network egress от eligibility конкретного сервиса:

```bash
mazzy-vpn verify
mazzy-vpn verify-service all --timeout 5 --json
```

`Network egress verified` не обещает доступ NotebookLM/OpenAI. Service check
явно и без credentials классифицирует только выбранный VPN egress. Для
NotebookLM exact unsupported-location означает `reachable/ineligible`, exact
home redirect — `reachable/eligible`; остальные ответы indeterminate. Для
OpenAI 401 или 405 с `Allow: POST` означает reachable auth boundary, 403 —
reachable/ineligible edge denial, а 429/5xx/неизвестный ответ — indeterminate.
Проверка не тестирует account, login, subscription, organization или content.

**Incident note, 2026-08-02.** На Germany AmneziaWG generic probes корректно
показали DE egress, а NotebookLM отдельно вернул unsupported location. Позднее
Belgium OpenVPN нормально достиг NotebookLM и Codex service: Google не отказывал
на рабочем OpenVPN egress. Независимая OpenVPN-проблема выглядела как повторная
инициализация control plane и создание interface при недоступном HTTPS data
plane; в одной provider session также задерживался `PUSH_REPLY` и встречались
unknown control opcodes. Health restart того же профиля примерно каждые 40
секунд усиливал цикл, поэтому watchdog теперь ограничен одним threshold restart
с последующим pause, оставляющим native retry OpenVPN. В заметке намеренно нет
адресов, endpoints, credentials или account данных. Automatic service-aware
failover не заявлен: смена профиля остаётся явным действием пользователя.

---

<a id="english"></a>

# Troubleshooting

Start with:

```bash
sudo mazzy-vpn diagnose
sudo mazzy-vpn doctor
systemctl status vpnctl.service
mazzy-vpn logs
mazzy-vpn validate all
mazzy-vpn probe all --timeout 3 --jobs 4
```

`Too many connections` is a provider/account rejection. Stop extra sessions,
wait for stale server state to expire and retry.

If network egress works but NotebookLM or OpenAI does not, run:

```bash
mazzy-vpn verify
mazzy-vpn verify-service all --timeout 5 --json
```

`Network egress verified` is not a service-access claim. The explicit,
credential-free classifier reports exact NotebookLM unsupported-location as
reachable/ineligible and the exact home redirect as reachable/eligible.
OpenAI 401, or 405 with `Allow: POST`, reaches the authentication boundary;
403 is reachable/ineligible, while 429, 5xx and unknown responses are
indeterminate. Account, login, subscription, organization and content access
are not tested.

**2026-08-02 incident note.** Generic probes correctly reported DE egress on a
Germany AmneziaWG profile while NotebookLM separately returned unsupported
location. A later Belgium OpenVPN profile reached NotebookLM and the Codex
service normally; Google did not fail on that working OpenVPN egress. A separate
OpenVPN problem repeatedly initialized its control plane and created an
interface while HTTPS data-plane traffic remained unavailable; one provider
session also showed delayed `PUSH_REPLY` and unknown control opcodes. Health
restarting the same profile about every 40 seconds amplified that cycle, so the
watchdog now performs one threshold restart and then pauses while OpenVPN native
retry continues. No address, endpoint, credential or account data is recorded
here or in the diagnostic result. Mazzy VPN does not claim automatic
service-aware failover; changing a profile remains an explicit user action.

If Desktop has no status, refresh the cache and inspect the health timer. If the
tray is absent, verify StatusNotifier/AppIndicator support and use right-click
on Linux. The systemd VPN remains operational without the GUI.

### IPv6 and account eligibility

For an active managed tunnel Mazzy installs `mazzy_vpn_ipv6_guard`. IPv4 is
unchanged; IPv6 is allowed only through loopback or the VPN interface. If
nftables cannot install the guard, tunnel startup fails closed.

```bash
ip -6 route
mazzy-vpn verify --timeout 10 --json
nft list table inet mazzy_vpn_ipv6_guard
```

Meta HTTP 403 and Antigravity eligibility errors are not equivalent to an IPv6
location leak. They can depend on account country, staged rollout, cookies,
provider policy, or VPN address reputation. Mazzy can prove transport egress
and block IPv6 leaks, but cannot change provider-side account eligibility.
