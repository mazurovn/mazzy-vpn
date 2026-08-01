# Добро пожаловать в Mazzy VPN Discussions

Mazzy VPN — open-source менеджер VPN-профилей и подключений с CLI/TUI,
автоматическим восстановлением и Desktop control center. Проект не продаёт
VPN-доступ и не предоставляет серверы: вы используете конфигурации своего
провайдера или собственной инфраструктуры.

Автор и сопровождающий: [Nik m (@mazurovn)](https://github.com/mazurovn).
Лицензия: GNU AGPL-3.0-or-later.

## Текущий статус

- Опубликованная release line — CLI/TUI 1.3 для Linux и unsigned Desktop 0.3
  preview. Linux Desktop функционален; Windows и macOS остаются UI preview без
  native VPN backend. Issue #31 закрыт проверенным `glib` backport и чистыми
  default-branch security checks.
- Windows/macOS — UI preview без production VPN backend.
- Android/iOS — planned native clients; готовых мобильных пакетов пока нет.

Страница [Releases](https://github.com/mazurovn/mazzy-vpn/releases) — источник
истины: версия в `main`, changelog или PR ещё не выпущена без соответствующего
tag и страницы Release.

## Куда писать

- **Announcements** — релизы и важные изменения от сопровождающего.
- **Q&A** — вопросы об установке, профилях, Doctor и диагностике.
- **Ideas** — предложения функций и улучшений UX.
- **Polls** — голосования за порядок будущих пользовательских функций.
- **General** — архитектура, документация, локализация и сообщество.
- **Show and tell** — безопасные примеры интеграций без приватных конфигураций.

Ошибка с воспроизводимыми шагами относится в
[Issues](https://github.com/mazurovn/mazzy-vpn/issues). Уязвимость или секрет
нельзя публиковать: используйте процесс из
[SECURITY.md](https://github.com/mazurovn/mazzy-vpn/blob/main/SECURITY.md).

## Правила

1. Указывайте ОС, версию Mazzy VPN, протокол и точные шаги.
2. Удаляйте ключи, пароли, PSK, endpoint, IP и личные пути из логов.
3. Не публикуйте рабочие VPN-профили и чужие учётные данные.
4. Отделяйте подтверждённое поведение от предположений.
5. Уважайте участников и соблюдайте законы своей юрисдикции.
6. Не выдавайте изменённые сборки за официальные релизы и сохраняйте авторство.

Результат голосования помогает расставить приоритеты, но не отменяет
обязательные security, platform и release gates.

Начните с [Wiki](https://github.com/mazurovn/mazzy-vpn/wiki),
[FAQ](https://github.com/mazurovn/mazzy-vpn/wiki/FAQ) и
[плана платформ](https://github.com/mazurovn/mazzy-vpn/wiki/Platform-Roadmap).
