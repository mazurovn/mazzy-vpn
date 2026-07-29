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

Установите один пакет из Releases:

```bash
sudo apt install "./Mazzy VPN Desktop_VERSION_amd64.deb"
sudo dnf install "./Mazzy VPN Desktop-VERSION-1.x86_64.rpm"
chmod +x "Mazzy VPN Desktop_VERSION_amd64.AppImage"
```

DEB и RPM добавляют приложение в системное меню. AppImage можно запускать без
системной установки. Сверяйте artifact с конкретным release/Actions commit.
Текущие preview artifacts не подписаны; неподписанный SHA-256 обнаруживает
случайное повреждение, но сам по себе не доказывает издателя.

Desktop 0.3 candidate содержит совместимые installer/engine resources. На экране Settings
он проверяет установленную версию и зависимости и после явного системного
разрешения устанавливает, обновляет или восстанавливает engine. Поэтому сначала
устанавливать CLI вручную не требуется. Статус preview сохраняется до закрытия
критериев [[Desktop Full Application Plan]]. Не публикуйте и не устанавливайте
Desktop 0.3 как новый preview, пока не закрыт RustSec gate issue #31 для
Tauri/GTK `glib` 0.18.

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

Install one bundle from Releases:

```bash
sudo apt install "./Mazzy VPN Desktop_VERSION_amd64.deb"
sudo dnf install "./Mazzy VPN Desktop-VERSION-1.x86_64.rpm"
chmod +x "Mazzy VPN Desktop_VERSION_amd64.AppImage"
```

DEB and RPM add an application-menu entry. AppImage is portable. Match the
artifact to its release/Actions commit. Current preview artifacts are unsigned;
an unsigned SHA-256 detects accidental corruption but does not prove the
publisher.

The Desktop 0.3 candidate contains compatible installer/engine resources. Its Settings screen
checks the installed version and dependencies and, after explicit system
authorization, installs, updates or repairs the engine. A prior manual CLI
installation is therefore not required. The package remains a preview until
the [[Desktop Full Application Plan]] release criteria are complete. Do not
publish or install Desktop 0.3 as a new preview while the issue #31 RustSec gate
for Tauri/GTK `glib` 0.18 remains open.

## macOS and Windows

Current DMG/app and MSI/NSIS artifacts are UI previews only. They do not create
a VPN tunnel and must not be treated as traffic-protection tools. Native
backends and signed installers are on the roadmap.
