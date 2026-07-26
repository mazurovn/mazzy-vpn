# Mazzy VPN 1.1.0

Copyright (C) 2026 Nik m ([@mazurovn](https://github.com/mazurovn)).

## English

Mazzy VPN 1.1.0 adds a secure Desktop Dashboard and system tray to the
transactional Linux CLI/TUI for AmneziaWG, WireGuard, OpenVPN and L2TP/IPsec.

Highlights:

- interactive terminal menu and the `mazzy-vpn` command;
- Tauri 2 Desktop Dashboard and tray with functional Linux AppImage, DEB and
  RPM bundles;
- sanitized JSON status cache with no endpoint, profile path or VPN secret;
- live dashboard, selected location and saved default-config quick connection;
- Russian, English, German, Chinese, Japanese and Korean interface selection;
- automatic profile discovery and folder import with safe permissions;
- validation, connectivity probes, batch tests and self-diagnostics;
- transactional switching with automatic rollback to the last working profile;
- systemd process restart plus independent 20-second health checks, immediate
  recovery of an inactive desired service and reconnect after two confirmed
  traffic failures;
- optional AdGuard VPN fallback without publishing personal credentials;
- installation and usage documentation in six languages;
- macOS and Windows UI preview builds, clearly marked as non-functional until
  native VPN backends and signing are implemented.

Install:

```bash
git clone https://github.com/mazurovn/mazzy-vpn.git
cd mazzy-vpn
sudo ./install.sh
mazzy-vpn
```

Private VPN profiles and credentials are intentionally not included in the
release. Import your own folder with
`mazzy-vpn import-dir /path/to/profiles`.

## Русский

Mazzy VPN 1.1.0 добавляет безопасный Desktop Dashboard и системный tray к
транзакционному Linux CLI/TUI для AmneziaWG, WireGuard, OpenVPN и L2TP/IPsec.

Основные возможности:

- интерактивное терминальное меню и команда `mazzy-vpn`;
- Tauri 2 Desktop Dashboard и tray с рабочими Linux AppImage, DEB и RPM;
- очищенный JSON status cache без endpoint, пути профиля и VPN-секретов;
- единый dashboard, выбранная локация и быстрое подключение default-конфига;
- выбор русского, английского, немецкого, китайского, японского или корейского;
- автоматическое распознавание профилей и импорт папок с безопасными правами;
- валидация, проверка доступности, пакетные тесты и самодиагностика;
- безопасное переключение с автоматическим возвратом к последнему рабочему профилю;
- автозапуск systemd и независимая проверка примерно каждые 20 секунд:
  остановленный сервис запускается сразу, а две подтверждённые ошибки трафика
  вызывают переподключение;
- необязательный резерв через AdGuard VPN без публикации личных данных;
- инструкции по установке и использованию на шести языках;
- preview-сборки macOS и Windows, явно отмеченные как нерабочие до реализации
  нативных VPN backends и подписи.

Установка:

```bash
git clone https://github.com/mazurovn/mazzy-vpn.git
cd mazzy-vpn
sudo ./install.sh
mazzy-vpn
```

Личные VPN-профили и учётные данные намеренно не входят в релиз. Импортируйте
свою папку командой `mazzy-vpn import-dir /путь/к/профилям`.

## Known limitation / Известное ограничение

An OpenVPN provider may reject a valid profile when its account or server
connection limit is exhausted. Mazzy VPN reports this as a server-side failure
and restores the previous connection instead of leaving the host offline.

Провайдер OpenVPN может отклонить исправный профиль, если исчерпан лимит
соединений учётной записи или сервера. Mazzy VPN определяет такую ошибку как
серверную и восстанавливает предыдущее подключение, не оставляя компьютер без
сети.

Licensed under the GNU Affero General Public License v3.0 or later.
