# Mazzy VPN Desktop: Dashboard и tray

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

[English](DESKTOP.en.md) · [Архитектура](ARCHITECTURE.ru.md) ·
[План самостоятельного Desktop 1.0](DESKTOP_ROADMAP.ru.md) ·
[Паритет функций](FEATURE_PARITY.md) · [Главная страница](../README.md)

![Mazzy VPN Desktop Dashboard — preview с тестовыми данными](images/dashboard-connected-preview.png)

Mazzy VPN Desktop — компактное приложение на Tauri 2 поверх проверяемого
CLI-движка `mazzy-vpn`. Оно показывает соединение в одном окне и оставляет
основные действия в системном tray.

> **Статус 0.1:** это функциональный Linux dashboard/companion, но ещё не
> самостоятельный VPN-клиент. Ему требуется установленный CLI engine. Desktop
> 1.0 будет включать общий core/bootstrap и полный паритет без отдельной
> установки CLI; до закрытия [release gates](FEATURE_PARITY.md) он остаётся
> preview.

## Статус платформ

| Платформа | Статус | Пакеты |
|---|---|---|
| Linux x86_64 | Рабочий Dashboard и tray для установленного Mazzy VPN CLI | AppImage, DEB, RPM |
| macOS | Preview интерфейса, VPN backend ещё не реализован | app, DMG |
| Windows | Preview интерфейса, VPN backend ещё не реализован | MSI, NSIS EXE |

macOS и Windows preview не нужно использовать как средство защиты трафика.
Для полноценной поддержки нужны нативные Network Extension/launchd и Windows
service/Wintun backends, подпись кода и platform-specific тесты.

## Что показывает Dashboard

- активность systemd service и реального VPN-интерфейса;
- доступ в интернет через выбранный туннель;
- локацию/default-профиль, протокол и имя интерфейса;
- возраст последнего WireGuard/AmneziaWG handshake;
- публичный VPN IP с кнопкой локального скрытия;
- автоподключение, health monitor и внешний fallback;
- количество профилей каждого протокола;
- локальный журнал действий текущего сеанса интерфейса;
- свежесть cache, чтобы устаревшие данные не выглядели актуальными.

Статус обновляется каждые пять секунд из `/run/mazzy-vpn/status.json`.
Root CLI/health monitor пересоздаёт этот файл атомарно. Он не содержит endpoint,
пути профилей, приватные ключи, логины, пароли или конфигурационные директивы.

## Действия в окне и tray

| Действие | CLI-команда |
|---|---|
| Quick Connect | `mazzy-vpn quick` |
| Reconnect | `mazzy-vpn reconnect` |
| Disconnect | `mazzy-vpn disconnect` |
| Self-diagnostics | `mazzy-vpn doctor` |
| Refresh Status | `mazzy-vpn _refresh-dashboard-cache` |

GUI не строит shell-строку. Rust backend принимает enum и сопоставляет его
фиксированному массиву аргументов. На Linux изменяющие состояние действия
проходят через системный `pkexec`, поэтому ОС может показать стандартный запрос
прав администратора. Закрытие окна скрывает приложение в tray; окончательный
выход выполняется пунктом **Quit Mazzy VPN**.

На Linux контекстное меню tray открывается правой кнопкой. Поддержка события
обычного клика зависит от desktop environment.

## Установка Linux

Сначала установите и проверьте CLI:

```bash
git clone https://github.com/mazurovn/mazzy-vpn.git
cd mazzy-vpn
sudo ./install.sh --yes
sudo mazzy-vpn diagnose
```

Затем установите один Desktop-пакет со страницы Releases.

DEB:

```bash
sudo apt install "./Mazzy VPN Desktop_0.1.0_amd64.deb"
```

RPM:

```bash
sudo dnf install "./Mazzy VPN Desktop-0.1.0-1.x86_64.rpm"
```

AppImage:

```bash
chmod +x "Mazzy VPN Desktop_0.1.0_amd64.AppImage"
./"Mazzy VPN Desktop_0.1.0_amd64.AppImage"
```

Имена версий в примерах замените на фактические имена загруженных файлов.
Проверяйте SHA-256 рядом с релизом.

## Языки

Dashboard поддерживает русский, английский, немецкий, китайский, японский и
корейский. Выбор сохраняется локально в хранилище WebView и не отправляется в
сеть. Язык CLI можно независимо изменить командой:

```bash
mazzy-vpn language ru
mazzy-vpn language en
```

## Диагностика

Если Dashboard не получил данные:

```bash
sudo mazzy-vpn _refresh-dashboard-cache
mazzy-vpn status --json
systemctl status vpnctl-health.timer
```

Если действие не выполнилось, проверьте `pkexec`, затем CLI:

```bash
command -v pkexec
sudo mazzy-vpn doctor
sudo mazzy-vpn diagnose
```

Если tray не виден, убедитесь, что desktop environment поддерживает
StatusNotifier/AppIndicator. Сам VPN продолжает работать через systemd даже без
запущенного Desktop.

## Сборка из исходного кода

```bash
cd desktop
npm ci
npm run test
cargo clippy --manifest-path src-tauri/Cargo.toml --all-targets -- -D warnings
npm run build:release
```

Linux build создаёт AppImage, DEB и RPM в
`desktop/src-tauri/target/release/bundle/`. Зависимости npm и Cargo
зафиксированы lock-файлами. Release-команда удаляет локальный домашний путь
сборщика из диагностических строк Rust. CI собирает каждую ОС на
соответствующем GitHub runner; release workflow создаёт только preview-релиз,
пока артефакты не подписаны.
