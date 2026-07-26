# Быстрый старт

## 1. Получить исходный код

```bash
git clone https://github.com/mazurovn/mazzy-vpn.git
cd mazzy-vpn
```

## 2. Установить CLI

```bash
sudo ./install.sh --yes --lang ru
```

Установщик запускает regression-тесты, устанавливает зависимости и systemd
units, проверяет профили, создаёт status cache для Desktop и выполняет offline
self-test.

## 3. Импортировать конфиги

```bash
mazzy-vpn import-dir ~/VPN-configs --dry-run
sudo mazzy-vpn import-dir ~/VPN-configs
```

Поддерживаются AmneziaWG, WireGuard, OpenVPN и NetworkManager L2TP/IPsec.
Копируются только распознанные и прошедшие проверку файлы.

## 4. Подключиться и проверить

```bash
sudo mazzy-vpn connect amneziawg 1
sudo mazzy-vpn quick
sudo mazzy-vpn diagnose
mazzy-vpn dashboard
```

Без аргументов `mazzy-vpn` открывает TUI. В нём доступны выбор сервера, быстрый
default, диагностика, тесты, импорт папок и выбор языка.

## 5. Включить автоподключение

```bash
sudo mazzy-vpn autostart on
systemctl status vpnctl.service
systemctl status vpnctl-health.timer
```

---

<a id="english"></a>

# Quick start

## 1. Get the source

```bash
git clone https://github.com/mazurovn/mazzy-vpn.git
cd mazzy-vpn
```

## 2. Install the CLI

```bash
sudo ./install.sh --yes --lang en
```

The installer runs regression tests, installs dependencies and systemd units,
validates profiles, creates the Desktop status cache and performs an offline
self-test.

## 3. Import configs

```bash
mazzy-vpn import-dir ~/VPN-configs --dry-run
sudo mazzy-vpn import-dir ~/VPN-configs
```

AmneziaWG, WireGuard, OpenVPN and NetworkManager L2TP/IPsec are supported. Only
recognized and validated files are copied.

## 4. Connect and verify

```bash
sudo mazzy-vpn connect amneziawg 1
sudo mazzy-vpn quick
sudo mazzy-vpn diagnose
mazzy-vpn dashboard
```

Running `mazzy-vpn` without arguments opens the TUI with server selection,
quick default, diagnostics, tests, folder import and language selection.

## 5. Enable automatic connection

```bash
sudo mazzy-vpn autostart on
systemctl status vpnctl.service
systemctl status vpnctl-health.timer
```
