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

Ниже указаны точные dot-normalized имена GitHub Release assets для 0.4;
локальный Tauri build может сохранять пробелы из product name.

```bash
sudo apt install ./Mazzy.VPN.Desktop_0.4.1_amd64.deb
sudo dnf install ./Mazzy.VPN.Desktop-0.4.1-1.x86_64.rpm
chmod +x ./Mazzy.VPN.Desktop_0.4.1_amd64.AppImage
```

DEB и RPM добавляют приложение в системное меню. AppImage можно запускать без
системной установки. Сверяйте artifact с конкретным release/Actions commit.
Installable updater artifacts имеют Tauri-подписи, которые проверяет встроенный
public key. Это не заменяет Authenticode, Apple notarization, подпись RPM или
APT-репозитория; SHA-256 manifest остаётся отдельной проверкой целостности.

Desktop 0.4 release source содержит совместимые installer/engine resources. На экране Settings
он проверяет установленную версию и зависимости и после явного системного
разрешения устанавливает, обновляет или восстанавливает engine. Поэтому сначала
устанавливать CLI вручную не требуется. Статус preview сохраняется до закрытия
критериев [[Desktop Full Application Plan]]. Issue #31 закрыт проверенным
`glib` backport; сверяйте downloads с `Mazzy.VPN.Desktop_0.4.1_SHA256SUMS`.

DEB/RPM владеют `/usr/bin/mazzy-vpn`. Начиная с 0.3.2 доверенные root-owned
копии Mazzy VPN из `/usr/local/bin` сохраняются в закрытом migration-каталоге и
заменяются ссылками на package engine, поэтому старая ручная версия больше не
перекрывает новую. Сторонние или небезопасные файлы installer не меняет;
удаление пакета восстанавливает сохранённые копии с исходными правами.

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

These are the exact dot-normalized names of the 0.4 GitHub Release
assets; a local Tauri build may retain spaces from the product name.

```bash
sudo apt install ./Mazzy.VPN.Desktop_0.4.1_amd64.deb
sudo dnf install ./Mazzy.VPN.Desktop-0.4.1-1.x86_64.rpm
chmod +x ./Mazzy.VPN.Desktop_0.4.1_amd64.AppImage
```

DEB and RPM add an application-menu entry. AppImage is portable. Match the
artifact to its release/Actions commit. Installable updater artifacts carry
Tauri signatures verified by the embedded public key. This does not replace
Authenticode, Apple notarization, RPM signing or APT repository signing; the
SHA-256 manifest remains a separate integrity check.

The Desktop 0.4 release source contains compatible installer/engine resources. Its Settings screen
checks the installed version and dependencies and, after explicit system
authorization, installs, updates or repairs the engine. A prior manual CLI
installation is therefore not required. The package remains a preview until
the [[Desktop Full Application Plan]] release criteria are complete. Issue #31
is closed with a verified `glib` backport; verify downloads with
`Mazzy.VPN.Desktop_0.4.1_SHA256SUMS`.

DEB/RPM own `/usr/bin/mazzy-vpn`. Since 0.3.2, recognized root-owned
Mazzy VPN copies under `/usr/local/bin` are moved into a private migration
directory and replaced with links to the package engine, so an older manual
version cannot shadow the update. Unrelated or unsafe files are unchanged;
package removal restores preserved copies with their original permissions.

## macOS and Windows

Current DMG/app and MSI/NSIS artifacts are UI previews only. They do not create
a VPN tunnel and must not be treated as traffic-protection tools. Native
backends and signed installers are on the roadmap.
