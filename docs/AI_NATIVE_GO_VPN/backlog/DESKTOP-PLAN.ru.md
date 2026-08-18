# План: Desktop (gio) + оставшиеся задачи CLI

Статус на v2.1.1. Готовим переход к Desktop и фиксируем, что осталось по CLI.

## Часть 1 — что осталось по CLI

| ID | Задача | Приор | Оценка |
|---|---|---|---|
| CLI-L10N | локализация вывода | P1 | **DONE** (98 ключей; остаются только подписи пунктов меню) |
| CLI-SETTINGS | команда `settings` (list/get/set) для всех полей `core/settings` | P2 | S (полдня) |
| CLI-COMPLETION | генератор bash/zsh/fish + `mazzy-vpn.1` man | P2 | M |
| CLI-HELP | per-command help (`mazzy-vpn help connect`) | P2 | S |
| OpenVPN (P3-5) | встроить движок, снять ограничение в `loadProfile` | P1 | L (отдельный движок) |
| C1-4d | native-netlink вместо exec `ip` | P2 | L |

**Вывод по CLI:** основной путь готов и локализован. Оставшееся — удобства
(settings/completion/help) уровня P2 и крупные движковые задачи (OpenVPN,
netlink). CLI можно считать функционально завершённым для релиза.

## Часть 2 — Desktop на gio (P3-3)

ADR-0004 уже выбрал **gio**: единственный тулкит без webkit/GTK-рантайма,
чистый Go, линкует общий `core/`, работает без CLI.

### Ключевой архитектурный нюанс: cgo

- CLI: `CGO_ENABLED=0` (полностью статичный, ноль зависимостей).
- Desktop (gio): **требует cgo** для X11/Wayland/GL/EGL/xkbcommon — это
  системные `.so`, которые есть на любом Linux-десктопе (не webkit!).
- Это НЕ нарушает автономность: пользователь ничего не доустанавливает; линкуются
  библиотеки, которые уже стоят в любой графической сессии.
- **Разделение**: `core/` остаётся `CGO_ENABLED=0` и переиспользуется обоими;
  cgo только в GUI-слое Desktop.

### Структура (предлагаемая)

```
mazzy-vpn-desktop/           # новый Go-модуль (не трогает Tauri desktop/)
  cmd/mazzy-vpn-desktop/
    main.go                  # gio app entry
  internal/
    ui/                      # gio-виджеты (dashboard, zone list, settings)
    controller/              # мост UI → core (connect/status/measure/settings)
    tray/                    # трей-иконка (P3-3a)
  go.mod                     # линкует core/ (replace ../core)
```

### Модель взаимодействия с core (ADR-0006)

Desktop работает **непривилегированно** и пишет намерение в
`~/.config/mazzy-vpn/desired.json`; привилегированный daemon применяет (тот же
контроллер-паттерн, что уже в CLI daemon). Так Desktop:
- не требует root у GUI;
- переиспользует `state`/`settings`/`livecheck`/`measure` из core;
- показывает реальный egress (тот же `livecheck`, что в CLI).

### Экраны MVP

1. **Dashboard** — статус (PROTECTED/LINK-UP/down), egress IP, страна, кнопка
   Connect/Disconnect. Данные из `livecheck` + `detectLiveInterface`.
2. **Zones** — список профилей из `catalog`, пинг/ранжирование из `measure`,
   выбор → desired.json.
3. **Settings** — те же поля `core/settings` (auto-connect, kill-switch,
   notifications, язык). Общий i18n-каталog (6 языков) переиспользуется.
4. **Diagnostics** — вывод `doctor`/`diagnose`/`stealth` в GUI.

### Задачи (разбивка P3-3)

- **P3-3-0**: SDD + скелет модуля `mazzy-vpn-desktop`, gio "hello" окно,
  линковка core, статус из livecheck (read-only).
- **P3-3-1**: Dashboard + connect/disconnect через desired.json + daemon.
- **P3-3-2**: Zones (catalog + measure) и выбор.
- **P3-3-3**: Settings (core/settings + i18n, 6 языков).
- **P3-3a**: трей-иконка (чистый-Go пакет) + автозапуск (`.desktop` autostart).
- **P3-3b**: рендер-проверка Wayland + X11 на целевых DE.

### Риски

- gio immediate-mode UX пишем с нуля (нельзя переиспользовать web-фронт Tauri) —
  осознанная цена (ADR-0004).
- Трей: gio сам не даёт — отдельный пакет (`fyne.io/systray` или аналог,
  чистый Go, без GTK).
- Сборочная машина: нужны dev-заголовки X11/Wayland/GL (есть; это зависимость
  сборки, не пользователя).

### Критерий приёмки (ADR-0004)

Desktop-бинарник запускается на чистом Linux-десктопе **без установки**
webkit/GTK/Electron/.NET — линкует только уже присутствующие системные графические `.so`.
