# ADR-0006: TUI-фреймворк (bubbletea) и модель привилегий

- Статус: **ПРИНЯТО** (после design-audit D1/D2)
- Дата: 2026-08
- Связано: ADR-0003 (автономность), design-audit `design/02-tui-design-audit.ru.md`.

## Контекст

Design-audit выявил два архитектурных блокера до реализации full-screen TUI:
- **D1**: не нарушает ли bubbletea автономность/static build?
- **D2**: как TUI выполняет привилегированные действия (connect/disconnect)?

## Решение D1 — bubbletea, vendored, static

Принять `charmbracelet/bubbletea` + `lipgloss` + `bubbles`.

Проверено:
- **Static build без cgo**: `CGO_ENABLED=0 go build` → statically linked. ✅
- 25 транзитивных модулей, все charmbracelet/x/sys/x/text — **MIT/BSD**. ✅
- Прирост бинарника ~+2 МБ (приемлемо). ✅
- Всё vendored в `mazzy-vpn-cli/vendor` → сборка без `go get`. ✅

Автономность сохраняется: TUI — чистый Go, один статический бинарник.

## Решение D2 — TUI как контроллер daemon'а (не sudo-обёртка)

**TUI работает БЕЗ root.** Привилегированные действия делегируются фоновому
`daemon`, который держит root (через systemd или разовый `sudo mazzy-vpn daemon`).

### Разделение ответственности
```
┌─ mazzy-vpn (TUI, без root) ──────────────┐
│  читает: livecheck, measure, status      │  ← мониторинг, безопасно без root
│  пишет:  control-file (желаемая зона)     │  ← намерение
└──────────────────┬───────────────────────┘
                   │ control-file / signal
┌──────────────────▼───────────────────────┐
│  mazzy-vpn daemon (root, systemd)         │  ← реально поднимает тоннель
│  читает control-file → connect/switch     │
│  auto-reconnect + zone-failover           │
└───────────────────────────────────────────┘
```

- **TUI без root**: показывает live-статус, список зон с пингом, диагностику,
  провайдеров, настройки. Всё read-only + запись намерения.
- **Действия**: `[c] Connect` пишет в control-file желаемую зону; daemon
  подхватывает и подключает. `[d] Disconnect` пишет desired=down.
- **Если daemon не запущен**: TUI показывает точную команду
  `sudo systemctl start mazzy-vpn@<zone>` или `sudo mazzy-vpn up <zone>`.

### Почему так
- Файлы настроек остаются user-owned (не root).
- Нет «весь UI под sudo» (небезопасно).
- Чистое разделение: UI vs привилегированный executor.
- Переиспользует уже готовый `daemon` (zone-failover, auto-reconnect).

## Прочие правки аудита (в реализацию)

- D3: кольцевой буфер лога (200 строк).
- D4: контексты ввода — глобальные hotkeys только на главном экране,
  в overlay-списке навигация стрелками.
- D5: таймауты async-команд (connect 20s, test 90s) через context.
- D6: ASCII-символы статуса (●▲✖◐⟳) для layout; эмодзи — только в подписях.

## Control-file протокол (TUI ↔ daemon)

`/run/mazzy-vpn/control.json` (root-writable dir, TUI пишет через
`mazzy-vpn request <zone>` — маленькая привилегированная команда, ИЛИ daemon
опрашивает user-файл `~/.config/mazzy-vpn/desired.json`).

Решение: **daemon опрашивает `~/.config/mazzy-vpn/desired.json`** (user-owned,
TUI пишет без root). Поля: `{ "zone": "…", "desired": "up|down", "ts": … }`.
Daemon применяет изменения на каждом тике.

## Последствия

- Реализация TUI не требует root — безопаснее и проще.
- Нужен минимальный протокол desired.json + опрос в daemon.
- `--plain` fallback сохраняется для не-TTY.
