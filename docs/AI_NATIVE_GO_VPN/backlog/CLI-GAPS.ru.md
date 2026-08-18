# Анализ: что не проработано по CLI (по состоянию на v2.1.0)

Обзор реального кода `mazzy-vpn-cli/` против заявленных возможностей и бэклога.
CLI функционально полон (~48 команд, работает на реальных конфигах), но есть
несколько незакрытых зон.

## Статус бэклога по CLI

| ID | Задача | Бэклог | Реально |
|---|---|---|---|
| U1–U7 | catalog/measure/up/menu/netadapter/netdiag/uplink | DONE | ✅ работает |
| C2-8 | TUI-меню + i18n (6 языков) | DONE (модель) | ⚠️ модель есть, вывод НЕ локализован |
| I18N-CLI | команда выбора языка | DONE | ✅ меню/‑‑list/резолв |
| P3-1 | автономный статический бинарник | **«В РАБОТЕ»** | ✅ по факту готово — статус устарел |
| P3-2 | install без apt/git | DONE | ✅ + CI-гейт |

## Незакрытые зоны CLI

### 1. Локализация вывода (P1) — главный разрыв
- i18n-инфраструктура есть (`core/i18n`, 6 языков, резолв без хардкода), но
  **каталог содержит всего 11 ключей**, а `translator().T()` вызывается только
  в `language.go` и `menu.go`.
- **278 вызовов `fmt.Print*`** по всему CLI — весь пользовательский вывод
  (`up`/`status`/`connect`/`doctor`/`providers`/`stealth`/…) захардкожен на
  английском.
- `printUsage()` (help) — тоже английский хардкод, не идёт через каталог.
- **Итог:** выбор языка меняет только меню и пару строк; остальной UX всегда
  английский. Это расхождение с обещанием «6 языков интерфейса».

### 2. OpenVPN в connect (P1, связано с P3-5)
- `loadProfile()` явно возвращает ошибку: `OpenVPN (.ovpn) is not yet supported
  by the embedded engine`. Движок OpenVPN не встроен (P3-5 TODO).
- `providers`/каталог упоминают OpenVPN, но `connect`/`up` работают только с
  AmneziaWG/WireGuard.

### 3. Отсутствуют UX-удобства
- **Shell-completion** (bash/zsh/fish) — нет генератора автодополнения команд.
- **man-страница** — нет `mazzy-vpn.1`.
- **`settings`/`config` команда** — нет способа посмотреть/править настройки
  (auto_connect, kill_switch, notifications…) из CLI; они меняются только
  косвенно (`favorite`, `language`) или правкой JSON.

### 4. Мелочи консистентности
- `--version`/`version` печатает версию, но не локализован.
- `help` не имеет per-command справки (`mazzy-vpn help connect`).

## Что НЕ требует работы (проверено — готово)
- Автономный статический бинарник (CGO_ENABLED=0), 0 внешних зависимостей.
- Полноэкранный TUI (`tui.go`, 408 строк, реальный bubbletea) + line-menu
  fallback.
- connect/daemon/recover/status/test/best/providers/stealth/mimic/dns-check/
  netdiag/diagnose/trace/control/language — все реализованы и протестированы.
- Единый источник имён интерфейсов, fail-closed, atomic state, single lock.

## Рекомендованный порядок доработки CLI

1. **CLI-L10N (P1)** — расширить каталог i18n на весь пользовательский вывод и
   прогнать `fmt.Print*` через `translator()`; локализовать `printUsage()`.
   Самый заметный разрыв с обещанием продукта.
2. **CLI-SETTINGS (P2)** — команда `settings`/`config` (list/get/set) для всех
   полей `core/settings`.
3. **CLI-COMPLETION (P2)** — генератор bash/zsh/fish completion + man-страница.
4. **OpenVPN (P3-5, P1)** — встроить движок, снять ограничение в `loadProfile`.
