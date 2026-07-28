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
> языков и перенос оставшихся typed `pkexec`-операций в частично реализованный
> versioned local API.

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
   локации/default-профиля, удаление, массовая проверка endpoint с отдельными
   reachability/latency/active и точечный live-test.
3. **Диагностика** — validate, DNS/ping probe, транзакционные тесты, `test-all`,
   emergency recovery, полный результат Doctor/self-test и ограниченный журнал
   systemd.
4. **Настройки** — версии bundled/installed engine, состояние зависимостей,
   Install/Update/Repair, autostart, health monitor, privacy и уведомления.
5. **О программе** — версии Desktop/engine/platform, автор, лицензия,
   приватность и правила безопасной работы.

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
Файлы имеют права `0640 root:mazzy-vpn`, а каталог —
`0750 root:mazzy-vpn`. Они не содержат endpoint, пути профилей, приватные
ключи, логины, пароли или конфигурационные директивы.

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
| Validate / массовый probe локаций | `mazzy-vpn validate` / `probe all --jobs 4 --json` |
| Transactional tests | `mazzy-vpn test` / `test-all` / `emergency` |
| Install / repair | пакетный engine: `mazzy-vpn doctor --fix`; AppImage/manual: bundled `install.sh` |
| Service settings | `mazzy-vpn autostart` / `monitor` |
| Logs | `mazzy-vpn logs --lines N` |

Probe локаций намеренно отделён от живого VPN-теста. `reachable` подтверждает
DNS и доступность ICMP или TCP endpoint; `unknown` сохраняет частый случай,
когда UDP VPN-сервер блокирует ICMP. Это не заявление о работе credentials,
handshake или маршрутизации туннеля — для такого доказательства нужен
подтверждённый live-test.

GUI не строит shell-строку. Rust backend принимает enum и сопоставляет его
фиксированному request или массиву аргументов. Connect, reconnect и disconnect
используют `/run/mazzy-vpn/api-v1.sock`, если он предоставлен установленным
engine. Socket ограничен `root:mazzy-vpn`, а outcomes не содержат raw backend
output. Те же lifecycle-команды CLI/TUI используют этот socket без `sudo` и
выбирают профиль только по непрозрачному ID. Остальные изменяющие состояние
действия проходят через системный
`pkexec`, поэтому ОС может показать стандартный запрос прав администратора.
Закрытие окна скрывает приложение в tray; окончательный выход выполняется
пунктом **Quit Mazzy VPN**.

На Linux контекстное меню tray открывается правой кнопкой. Поддержка события
обычного клика зависит от desktop environment.

## Установка Linux

Установите один Desktop-пакет со страницы Releases. DEB и RPM теперь являются
package-managed установками: архив владеет engine в `/usr/bin`, публичным
runtime в `/usr/lib/mazzy-vpn`, systemd units/drop-ins в
`/usr/lib/systemd/system`, tmpfiles policy и Bash completion. Package manager
устанавливает базовые runtime-зависимости, а поддерживаемые VPN-протоколы
объявлены рекомендациями. Идемпотентный package script создаёт защищённую
структуру state, активирует socket/recovery monitor на запущенном systemd host
и проверяет engine/API manifest.

Существующие профили `/etc/vpnctl` и state `/var/lib/vpnctl` не входят в
package payload и намеренно сохраняются при upgrade и remove. Старый ручной
unit в `/etc/systemd/system` сохраняет свои настройки, а package drop-in
переключает только executable на package-owned `/usr/bin/mazzy-vpn`. Если
установка запущена через `sudo` или `pkexec`, вызывающий пользователь
добавляется в группу `mazzy-vpn`; для socket может понадобиться новый login.
Package-safe действие «Исправить» повторяет это добавление, если графический
package manager не сохранил сведения о вызвавшем пользователе.

DEB:

```bash
sudo apt install "./Mazzy VPN Desktop_0.2.0_amd64.deb"
```

RPM:

```bash
sudo dnf install "./Mazzy VPN Desktop-0.2.0-1.x86_64.rpm"
```

Для DEB/RPM действие **Установить / обновить / исправить** запускает
package-safe `mazzy-vpn doctor --fix`: оно исправляет поддерживаемые
недостающие protocol dependencies и service state, но не копирует package
files в `/usr/local`. Этот slice всё ещё preview: остаются clean-device
install/upgrade/remove tests для всех поддерживаемых дистрибутивов,
package rollback/fault injection, доставка AmneziaWG и подпись.

AppImage:

```bash
chmod +x "Mazzy VPN Desktop_0.2.0_amd64.AppImage"
./"Mazzy VPN Desktop_0.2.0_amd64.AppImage"
```

AppImage не может установить собственный privilege helper. Сначала проверьте
`command -v pkexec` и вручную установите package polkit/pkexec вашего
дистрибутива, если команды нет; иначе встроенный engine bootstrap не запустится.
AppImage по-прежнему использует явно разрешённый embedded installer и не
является package-managed установкой.

Имена версий в примерах замените на фактические имена загруженных файлов.
Preview-артефакты пока не подписаны, а workflow ещё не публикует подписанные
checksum или provenance. Проверяйте source commit и его GitHub Actions run, но
не считайте неподписанный hash доказательством издателя.

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

Если действие не выполнилось, проверьте API socket, членство в группе,
`pkexec` fallback, затем CLI:

```bash
systemctl status mazzy-vpn-api.socket
id -nG | tr ' ' '\n' | grep -x mazzy-vpn
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
