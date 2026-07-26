# Mazzy VPN 1.0.0

Copyright (C) 2026 Nik m ([@mazurovn](https://github.com/mazurovn)).

## English

The first stable release of Mazzy VPN, a transactional Linux CLI/TUI for
AmneziaWG, WireGuard, OpenVPN and L2TP/IPsec profiles.

Highlights:

- interactive terminal menu and the `mazzy-vpn` command;
- live dashboard, selected location and saved default-config quick connection;
- Russian, English, German, Chinese, Japanese and Korean interface selection;
- automatic profile discovery and folder import with safe permissions;
- validation, connectivity probes, batch tests and self-diagnostics;
- transactional switching with automatic rollback to the last working profile;
- systemd process restart plus independent 20-second health checks, immediate
  recovery of an inactive desired service and reconnect after two confirmed
  traffic failures;
- optional AdGuard VPN fallback without publishing personal credentials;
- installation and usage documentation in six languages.

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

Первый стабильный выпуск Mazzy VPN — транзакционного CLI/TUI-клиента Linux для
профилей AmneziaWG, WireGuard, OpenVPN и L2TP/IPsec.

Основные возможности:

- интерактивное терминальное меню и команда `mazzy-vpn`;
- единый dashboard, выбранная локация и быстрое подключение default-конфига;
- выбор русского, английского, немецкого, китайского, японского или корейского;
- автоматическое распознавание профилей и импорт папок с безопасными правами;
- валидация, проверка доступности, пакетные тесты и самодиагностика;
- безопасное переключение с автоматическим возвратом к последнему рабочему профилю;
- автозапуск systemd и независимая проверка примерно каждые 20 секунд:
  остановленный сервис запускается сразу, а две подтверждённые ошибки трафика
  вызывают переподключение;
- необязательный резерв через AdGuard VPN без публикации личных данных;
- инструкции по установке и использованию на шести языках.

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
