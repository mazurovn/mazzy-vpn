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
mazzy-vpn probe all --timeout 3
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
mazzy-vpn probe all --timeout 3
```

`Too many connections` is a provider/account rejection. Stop extra sessions,
wait for stale server state to expire and retry.

If Desktop has no status, refresh the cache and inspect the health timer. If the
tray is absent, verify StatusNotifier/AppIndicator support and use right-click
on Linux. The systemd VPN remains operational without the GUI.
