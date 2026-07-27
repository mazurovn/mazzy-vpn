# Mazzy VPN Desktop: Linux Control Center и tray

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

[English](DESKTOP.en.md) · [Архитектура](ARCHITECTURE.ru.md) ·
[План самостоятельного Desktop 1.0](DESKTOP_ROADMAP.ru.md) ·
[Паритет функций](FEATURE_PARITY.md) · [Главная страница](../README.md)

![Mazzy VPN Desktop Dashboard — preview с тестовыми данными](images/dashboard-connected-preview.png)

Mazzy VPN Desktop — приложение на Tauri 2 с общим движком `mazzy-vpn`.
Linux-пакет 0.2 включает совместимый engine installer: отдельная предварительная
установка CLI больше не требуется. CLI и TUI остаются самостоятельными
клиентами того же движка и состояния.

> **Статус 0.2:** Linux control-center уже покрывает основной рабочий цикл, но
> остаётся preview до закрытия [release gates](FEATURE_PARITY.md). Для Desktop
> 1.0 ещё нужны полный fallback-policy UI, перевод новых экранов на все шесть
> языков и versioned daemon API вместо промежуточного typed `pkexec`-адаптера.

## Статус платформ

| Платформа | Статус | Пакеты |
|---|---|---|
| Linux x86_64 | Control Center с встроенным bootstrap общего движка | AppImage, DEB, RPM |
| macOS | Preview интерфейса, VPN backend ещё не реализован | app, DMG |
| Windows | Preview интерфейса, VPN backend ещё не реализован | MSI, NSIS EXE |

macOS и Windows preview не нужно использовать как средство защиты трафика.
Для полноценной поддержки нужны нативные Network Extension/launchd и Windows
service/Wintun backends, подпись кода и platform-specific тесты.

## Экраны Desktop 0.2

1. **Обзор** — туннель, интернет, IP, handshake, health, recovery и tray.
2. **Профили** — безопасный импорт файлов/папок, поиск, протоколы, выбор
   локации/default-профиля, удаление и точечный live-test.
3. **Тестирование** — validate, DNS/ping probe, `test-all` и emergency с
   timeout/rollback.
4. **Диагностика** — полный, не обрезанный результат Doctor/self-test и
   ограниченный журнал systemd.
5. **Настройки** — версии bundled/installed engine, состояние зависимостей,
   Install/Update/Repair, autostart, health monitor, privacy и уведомления.

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
Санитизированная библиотека профилей читается из
`/run/mazzy-vpn/profiles.json`. Root engine пересоздаёт оба файла атомарно.
Они не содержат endpoint, пути профилей, приватные ключи, логины, пароли или
конфигурационные директивы.

## Действия в окне и tray

| Действие | CLI-команда |
|---|---|
| Quick Connect | `mazzy-vpn quick` |
| Reconnect | `mazzy-vpn reconnect` |
| Disconnect | `mazzy-vpn disconnect` |
| Self-diagnostics | `mazzy-vpn doctor` |
| Диагностика активного соединения | `mazzy-vpn diagnose` |
| Refresh Status | `mazzy-vpn _refresh-dashboard-cache` |
| Connect profile | `mazzy-vpn connect PROTOCOL PROFILE` |
| Import files/folder | `mazzy-vpn import-files` / `import-dir` |
| Validate / probe | `mazzy-vpn validate` / `probe` |
| Transactional tests | `mazzy-vpn test` / `test-all` / `emergency` |
| Install / repair | bundled `install.sh --yes --skip-tests` |
| Service settings | `mazzy-vpn autostart` / `monitor` |
| Logs | `mazzy-vpn logs --lines N` |

GUI не строит shell-строку. Rust backend принимает enum и сопоставляет его
фиксированному массиву аргументов. На Linux изменяющие состояние действия
проходят через системный `pkexec`, поэтому ОС может показать стандартный запрос
прав администратора. Закрытие окна скрывает приложение в tray; окончательный
выход выполняется пунктом **Quit Mazzy VPN**.

На Linux контекстное меню tray открывается правой кнопкой. Поддержка события
обычного клика зависит от desktop environment.

## Установка Linux

Установите один Desktop-пакет со страницы Releases. Затем откройте
**Настройки → Установить / обновить / исправить**. Встроенный проверенный
installer определит ОС, сохранит существующие профили, установит недостающие
зависимости, systemd engine/recovery и выполнит offline self-test.

DEB:

```bash
sudo apt install "./Mazzy VPN Desktop_0.2.0_amd64.deb"
```

RPM:

```bash
sudo dnf install "./Mazzy VPN Desktop-0.2.0-1.x86_64.rpm"
```

AppImage:

```bash
chmod +x "Mazzy VPN Desktop_0.2.0_amd64.AppImage"
./"Mazzy VPN Desktop_0.2.0_amd64.AppImage"
```

Имена версий в примерах замените на фактические имена загруженных файлов.
Проверяйте SHA-256 рядом с релизом.

## Языки

Dashboard 0.1 полностью поддерживал русский, английский, немецкий, китайский,
японский и корейский. Новые экраны 0.2 полностью переведены на русский и
английский; для остальных языков пока используется английский fallback, поэтому
release gate локализации честно остаётся `partial`. Выбор сохраняется локально
и синхронизируется с общим engine.

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
