# FAQ: установка, профили, Doctor и релизы

## Это VPN-провайдер?

Нет. Mazzy VPN управляет вашими профилями и подключениями, но не выдаёт аккаунт
или VPN-сервер.

## Desktop требует отдельной установки CLI?

Linux Desktop 0.2 содержит совместимый engine и installer. Предварительно
ставить CLI вручную не нужно. Системные изменения выполняются после стандартной
авторизации ОС.

## Что можно импортировать?

AmneziaWG, WireGuard, OpenVPN и NetworkManager L2TP/IPsec. Поддержка backend
зависит от платформы; распознавание файла не означает готовность протокола на
Windows, macOS или mobile.

## Где находятся профили?

На Linux — в root-protected `/etc/vpnctl/profiles`. Frontend получает
очищенную метаинформацию, а не приватные ключи.

## Что проверяет Doctor?

Версии, зависимости, профили, systemd, desired state, VPN-интерфейс и
соединение. Desktop 0.2 показывает полный результат и ограниченный журнал.
Исправление запускается отдельно и требует подтверждения.

## Есть ли телеметрия?

Нет обязательного аккаунта, аналитики и телеметрии. Подробнее:
[PRIVACY.md](https://github.com/mazurovn/mazzy-vpn/blob/main/PRIVACY.md).

## Где скачать Windows, macOS, Android или iOS?

Windows/macOS artifacts пока UI preview и не должны использоваться для защиты
трафика. Android/iOS находятся на стадии планирования. Следите за
[release gates](https://github.com/mazurovn/mazzy-vpn/wiki/Releases-and-Roadmap).

Полная версия FAQ:
https://github.com/mazurovn/mazzy-vpn/wiki/FAQ
