# Установка

## Требования Linux

- systemd;
- Bash, iproute2, curl, ping, timeout и flock;
- runtime нужного протокола: AmneziaWG, WireGuard, OpenVPN или
  NetworkManager L2TP/IPsec;
- `pkexec` для действий Desktop с повышенными правами.

## Инсталлятор CLI

```bash
sudo ./install.sh
sudo ./install.sh --yes --lang ru
sudo ./install.sh --config-dir ~/VPN-configs
sudo ./install.sh --live-test
```

Важные параметры:

| Параметр | Назначение |
|---|---|
| `--yes` | не задавать вопросы при установке пакетов |
| `--lang CODE` | `ru`, `en`, `de`, `zh`, `ja` или `ko` |
| `--config-dir DIR` | рекурсивно распознать и импортировать профили |
| `--force-configs` | явно заменить одноимённые изменённые профили |
| `--no-deps` | не устанавливать отсутствующие пакеты |
| `--live-test` | после установки проверить реальные подключения с rollback |
| `--dry-run` | показать действия без изменений |

Основная команда: `/usr/local/bin/mazzy-vpn`. Устанавливаются aliases `vpnctl`
и `mazzyvpn`, Bash completion и units:

- `vpnctl.service`;
- `vpnctl-health.timer`;
- `vpnctl-health.service`;
- `vpnctl-test-recovery.service`.

## Desktop Linux

Сначала должен быть установлен CLI. Затем установите один пакет из Releases:

```bash
sudo apt install "./Mazzy VPN Desktop_VERSION_amd64.deb"
sudo dnf install "./Mazzy VPN Desktop-VERSION-1.x86_64.rpm"
chmod +x "Mazzy VPN Desktop_VERSION_amd64.AppImage"
```

DEB и RPM добавляют приложение в системное меню. AppImage можно запускать без
системной установки. Проверяйте опубликованный SHA-256.

## macOS и Windows

Текущие DMG/app и MSI/NSIS являются только preview интерфейса. Они не поднимают
VPN-туннель и не являются средством защиты трафика. Нативные backends и
подписанные installers находятся в roadmap.

---

<a id="english"></a>

# Installation

## Linux requirements

- systemd;
- Bash, iproute2, curl, ping, timeout and flock;
- the selected protocol runtime: AmneziaWG, WireGuard, OpenVPN or
  NetworkManager L2TP/IPsec;
- `pkexec` for privileged Desktop actions.

## CLI installer

```bash
sudo ./install.sh
sudo ./install.sh --yes --lang en
sudo ./install.sh --config-dir ~/VPN-configs
sudo ./install.sh --live-test
```

Important options:

| Option | Purpose |
|---|---|
| `--yes` | do not prompt while installing packages |
| `--lang CODE` | `ru`, `en`, `de`, `zh`, `ja` or `ko` |
| `--config-dir DIR` | recursively recognize and import profiles |
| `--force-configs` | explicitly replace changed same-name profiles |
| `--no-deps` | do not install missing packages |
| `--live-test` | test real tunnels with rollback after installation |
| `--dry-run` | print actions without changing the system |

The main command is `/usr/local/bin/mazzy-vpn`. The installer also creates
`vpnctl` and `mazzyvpn` aliases, Bash completion and the systemd units listed
above.

## Linux Desktop

Install the CLI first, then one bundle from Releases:

```bash
sudo apt install "./Mazzy VPN Desktop_VERSION_amd64.deb"
sudo dnf install "./Mazzy VPN Desktop-VERSION-1.x86_64.rpm"
chmod +x "Mazzy VPN Desktop_VERSION_amd64.AppImage"
```

DEB and RPM add an application-menu entry. AppImage is portable. Verify the
published SHA-256 checksum.

## macOS and Windows

Current DMG/app and MSI/NSIS artifacts are UI previews only. They do not create
a VPN tunnel and must not be treated as traffic-protection tools. Native
backends and signed installers are on the roadmap.
