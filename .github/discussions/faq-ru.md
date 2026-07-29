# FAQ: релизы, установка, профили, безопасность и поддержка

## Это VPN-провайдер?

Нет. Mazzy VPN управляет вашими профилями и подключениями, но не выдаёт аккаунт
или VPN-сервер. Нужен профиль от вашего провайдера или собственной
инфраструктуры.

## Какая версия действительно выпущена?

Опубликованной считается только версия с tag и страницей в
[GitHub Releases](https://github.com/mazurovn/mazzy-vpn/releases). Версия в
`main`, changelog или draft PR ещё не является релизом.

Опубликованная release line — CLI/TUI 1.2.0 и Desktop 0.2.0 preview. Исходное
дерево 1.3.0/Desktop 0.3.0 остаётся release candidate, пока не существуют оба
tags и соответствующие Release pages. Публикация Desktop 0.3 дополнительно
заблокирована, пока issue #31 не уберёт Tauri/GTK `glib` 0.18 RustSec advisory
из Linux graph.

## Desktop требует отдельной установки CLI?

Linux Desktop 0.3 содержит совместимый engine и installer, поэтому
предварительная ручная установка CLI не требуется. Системные изменения
выполняются только
после стандартной авторизации ОС. Не устанавливайте Desktop 0.3 как новый
preview, пока RustSec gate issue #31 не закрыт.

## Как проверить фактические маршрут и локацию VPN?

Запустите `mazzy-vpn verify` или **Проверить VPN** в Desktop. Проверка сравнит
interface-bound/default IPv4, валидирует два geo providers именно для этого IP,
проверит настроенный DNS route и возможную IPv6-утечку. `--speed` отдельно
добавляет явный bounded 5-МБ sample. Это evidence сети в момент проверки;
регион аккаунта, cookies, WebRTC и геолокация устройства всё ещё могут влиять
на сайт.

## Означает ли успешная сборка рабочий VPN на любой ОС?

Нет. CI подтверждает, что UI собирается на Linux, macOS и Windows, но не
подменяет проверку VPN backend. Linux Desktop 0.3 — функциональный control-center
preview. Windows/macOS artifacts остаются UI preview до появления нативных
backend, подписанных installers и platform integration tests.

## Что можно импортировать?

AmneziaWG, WireGuard, OpenVPN и NetworkManager L2TP/IPsec. Поддержка backend
зависит от платформы; распознавание файла не означает готовность протокола на
Windows, macOS или mobile.

## Где находятся профили?

На Linux — в root-protected `/etc/vpnctl/profiles`. Frontend получает
очищенную метаинформацию без ключей, endpoint и приватных путей. Не публикуйте
рабочий профиль, private key, пароль, PSK, endpoint или полный журнал.

## Что проверяет Doctor?

Версии, зависимости, профили, systemd, desired state, VPN-интерфейс и
соединение. Desktop 0.3 показывает полный результат и ограниченный журнал.
Исправление запускается отдельно, объясняет системные изменения и требует
подтверждения.

## Безопасны ли live tests?

Тест временно меняет активный VPN-маршрут, проверяет профиль и выполняет
transactional rollback после успеха, ошибки, тайм-аута или сигнала. Перед
запуском сохраните важную сетевую работу и внимательно прочитайте подтверждение.

## Есть ли телеметрия?

Нет обязательного аккаунта, аналитики и телеметрии. Подробнее:
[PRIVACY.md](https://github.com/mazurovn/mazzy-vpn/blob/main/PRIVACY.md).

## Где скачать Windows, macOS, Android или iOS?

Windows/macOS artifacts пока UI preview и не должны использоваться для защиты
трафика. Android/iOS находятся на стадии планирования. Следите за
[release gates](https://github.com/mazurovn/mazzy-vpn/wiki/Releases-and-Roadmap).

## Почему `preview-release` в PR имеет статус `skipped`?

Это ожидаемое условие workflow, а не ошибка. Release job запускается только от
tag `desktop-v*`; в обычном push или PR выполняются тесты и сборка временных
artifacts. Публикация требует отдельного tag после принятия release PR.

## Как предложить и выбрать будущую функцию?

Используйте категории
[Ideas](https://github.com/mazurovn/mazzy-vpn/discussions/categories/ideas) и
[Polls](https://github.com/mazurovn/mazzy-vpn/discussions/categories/polls).
Голосование влияет на порядок пользовательских функций, но не отменяет
обязательные security, platform и release gates.

## Как получить помощь или сообщить об ошибке?

Запустите `mazzy-vpn version`, `mazzy-vpn doctor` и
`mazzy-vpn self-test --offline`. Для вопроса создайте Q&A с ОС, версией,
протоколом, шагами и очищенным выводом. Воспроизводимую ошибку перенесите в
[Issues](https://github.com/mazurovn/mazzy-vpn/issues). Уязвимости сообщайте
приватно по [SECURITY.md](https://github.com/mazurovn/mazzy-vpn/blob/main/SECURITY.md).

Полная версия FAQ:
https://github.com/mazurovn/mazzy-vpn/wiki/FAQ
