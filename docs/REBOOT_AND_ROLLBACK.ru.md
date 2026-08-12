# Установка, перезагрузка и откат DEB 0.4.8

## Установленный кандидат

- Desktop: `0.4.8`
- Engine/CLI: `1.4.7`
- DEB: `dist/desktop-v0.4.8/Mazzy VPN Desktop_0.4.8_amd64.deb`
- SHA-256: `50bb0ba798e5f2c8ab24519d890b282d6c5d232dff9d28c8eae60cfbc9e3b6ed`

Пароль не передаётся скриптам и не сохраняется. Для графической авторизации
используется PolicyKit.

## Проверка перед перезагрузкой

```bash
cd /home/mazurov/RESEARCH/MAZZY_VPN
pkexec ./scripts/pre-reboot-check.sh 0.4.8 1.4.7 pre-reboot
```

Допустимый результат — `Pre-reboot verdict: GO`. Предупреждение о старых
policy rules до первой перезагрузки допустимо: post-reboot gate требует ровно
одну пару правил.

## Перезагрузка

Сохраните открытые документы и выполните перезагрузку штатным способом через
Desktop Environment либо:

```bash
systemctl reboot
```

Не запускайте вручную `mazzy-vpn`, `mazzy-agentd`, `doctor --fix` или команды
очистки recovery marker после входа. Socket, recovery, VPN service и health
timer должны запуститься сами.

## Проверка после входа

```bash
cd /home/mazurov/RESEARCH/MAZZY_VPN
pkexec ./scripts/pre-reboot-check.sh 0.4.8 1.4.7 post-reboot
```

Ожидаемый результат — `Pre-reboot verdict: GO`. Затем запустите Mazzy VPN
Desktop из меню приложений. Desktop использует внутренний
`/usr/lib/mazzy-vpn/mazzy-vpn`; запуск отдельного CLI не требуется.

## Если post-reboot gate не прошёл

Сначала сохраните диагностику, не очищая markers:

```bash
systemctl status --no-pager \
  mazzy-vpn-api-recovery.service mazzy-vpn-api.socket \
  vpnctl.service vpnctl-health.timer
journalctl -b --no-pager \
  -u mazzy-vpn-api-recovery.service \
  -u mazzy-vpn-api.socket \
  -u vpnctl.service \
  -u vpnctl-health.service
```

Не выполняйте `_api-clear-recovery` и не удаляйте nft rules вручную.

## Откат к сохранённому 0.4.7

Rollback bundle создан автоматически в `.mazzy/rollback`; он содержит только
предыдущий DEB и systemd intent, но не VPN-профили, ключи или пароль.

Проверка bundle без изменений системы:

```bash
./scripts/rollback-release-deb.sh --check
```

```bash
cd /home/mazurov/RESEARCH/MAZZY_VPN
pkexec ./scripts/rollback-release-deb.sh
```

Скрипт проверяет SHA-256, выполняет разрешённый downgrade к `0.4.7`,
восстанавливает enablement units и не очищает recovery markers. Он также не
переключает и не останавливает туннель намеренно. Если откат выполняется уже
после неудачной загрузки и service остаётся inactive, сначала изучите journal;
не запускайте VPN поверх неизвестного recovery state.
