# Раунд 22 — VPN recovery переписан на Mazzy VPN (AdGuard в резерв)

- Дата: 2026-08
- Область: `mazzy-vpn-recover`, pi-vpn-recovery config, sudoers.
- Метод: установка + живой тест на работающем VPN.

## Что сделано

Переписана логика автовосстановления VPN: теперь при сбое сети первым
восстанавливается **наш Mazzy VPN**, а **AdGuard — только резерв**.

### Приоритет восстановления
```
сбой сети → recovery →
  1. Mazzy VPN (наш): recover (чистка) → up --best (лучшая живая зона)
     └ проверка protected egress до 25s
  2. если Mazzy не поднялся → AdGuard (резерв): reconnect
     └ проверка tun0 до 20s
```

## Компоненты

| Файл | Роль |
|---|---|
| `/usr/local/sbin/mazzy-vpn-recover` | recover-скрипт (--check/--present/--healthy/--recover/--verify) |
| `/etc/sudoers.d/mazzy-vpn-recover` | NOPASSWD только для --check/--recover (узкая привилегия) |
| `~/.config/pi-vpn-recovery/config.json` | selectedClient=mazzy, clients=[mazzy, adguard] |

### Режимы recover-скрипта
- `--present` — есть ли Mazzy-интерфейс ИЛИ AdGuard активен;
- `--healthy` — egress через Mazzy-тоннель ИЛИ AdGuard tun0 up;
- `--recover` — Mazzy VPN первым, AdGuard резервом (под flock);
- `--verify` — подтверждение защищённого egress (Mazzy ИЛИ AdGuard).

## Живой тест (наш VPN активен)

```
--present : exit 0 (vpnaw0 up)
--healthy : exit 0 (egress 95.211.225.232 protected)
--check   : exit 0 (через NOPASSWD sudoers)
config    : selectedClient=mazzy, enabled=true
status    : ✔ PROTECTED egress=95.211.225.232
```

## Безопасность

- sudoers даёт NOPASSWD **только** на `--check` и `--recover` (не на произвольные
  команды). `visudo -cf` = parsed OK.
- Скрипт root-owned 0755, проверяет `$EUID==0` и наличие бинарников.
- flock предотвращает параллельные восстановления.
- Пароль нигде не сохранён (использован транзитно для установки).

## В пакете

- `dist/mazzy-vpn-go/mazzy-vpn-recover`
- `dist/mazzy-vpn-go/pi-vpn-recovery-mazzy.json`

## Результат

Теперь при разрыве связи система автоматически поднимает **наш** VPN (лучшую
живую зону), а AdGuard используется только если наш не смог. Это завершает
переход на собственный VPN как основной с AdGuard в резерве.
