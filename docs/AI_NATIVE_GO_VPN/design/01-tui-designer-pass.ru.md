# Interface-Designer Pass — Mazzy VPN TUI (redesign spec)

Роль: interface-designer. Задача: спроектировать полноэкранный, неблокирующий,
информативный TUI лучше текущего построчного меню.

## 1. Проблемы текущего TUI (что чинить)

| # | Проблема | Влияние |
|---|---|---|
| P1 | Построчный вывод — при каждом действии экран «прокручивается», история засоряется | низкая читаемость |
| P2 | Статус сэмплируется только при отрисовке меню — не live | нет мониторинга |
| P3 | Меню блокируется на время команды (test/providers) | «зависает» |
| P4 | 14 плоских пунктов — cognitive load, нет фокуса на главном действии | медленный выбор |
| P5 | Нет обратной связи о прогрессе (спиннеры/проценты) | непонятно, работает ли |
| P6 | Рамка статуса ломается по ширине (эмодзи + кириллица) | визуальный дефект |

## 2. Целевая модель — full-screen bubbletea

Экран делится на зоны (Elm-архитектура: Model/Update/View):

```
╭─ Mazzy VPN ───────────────────────────── wlp3s0 (wifi) ─╮
│  ● PROTECTED   NL · 203.0.113.10 · 68 ms · ↑4MB ↓7MB   │  ← live header (тикер 2s)
╰──────────────────────────────────────────────────────────╯
  [c] Connect best   [z] Zones   [d] Disconnect   [r] Recover
  [t] Test   [a] Adapters   [n] Netdiag   [p] AI providers
  [i] Import   [s] Settings   [?] Help   [q] Quit
╭─ Activity ───────────────────────────────────────────────╮
│ 19:40:12  ✔ connected NetherlandsAmsterdamH4 (68 ms)      │  ← scrolling log pane
│ 19:40:30  ⟳ egress check ok                               │
╰──────────────────────────────────────────────────────────╯
```

### Зоны
1. **Header (live)** — состояние тоннеля, зона, egress, пинг, трафик; тикер 2s,
   не блокирует ввод. Цвет: зелёный PROTECTED / жёлтый LINK-UP / серый DOWN.
2. **Action bar** — hotkeys (одна буква), сгруппированы: соединение /
   диагностика / профили. Главное действие `[c] Connect best` — первое.
3. **Activity log** — прокручиваемая лента событий (connect/reconnect/notify),
   последние N строк. Долгие операции (test/providers) пишут прогресс сюда,
   **не блокируя** header и hotkeys.
4. **Modal/overlay** — для списков (зоны с пингом), настроек (тоглы), выбора.

## 3. Взаимодействие (non-blocking)

- Всё на bubbletea `tea.Cmd` — длинные операции (ping-ranking, provider-check,
  connect) выполняются как асинхронные команды, шлют `tea.Msg` в Update.
- Header-тикер (`tea.Tick`) обновляет статус каждые 2s независимо.
- Пользователь может нажать `q`/сменить экран, пока идёт `test`.

## 4. Информационная иерархия

1. **Самое важное всегда видно**: подключён/нет, egress, зона (header).
2. **Главное действие первым**: Connect best (одна клавиша `c`).
3. **Опасное — с подтверждением**: Recover (`r`) и Disconnect (`d`) требуют
   `y`-подтверждения в overlay.
4. **Диагностика — на второй линии** hotkeys.

## 5. Состояния (failure states) — обязательно показать

| Состояние | Header | Действие-подсказка |
|---|---|---|
| DOWN | серый `● DISCONNECTED` | `[c] Connect best` |
| CONNECTING | жёлтый `◐ connecting… NL` спиннер | — |
| LINK-UP (no egress) | жёлтый `▲ no egress` | `[r] try another zone` |
| PROTECTED | зелёный `● PROTECTED` | `[d] disconnect` |
| RECONNECTING | жёлтый `⟳ reconnecting…` | — |
| NO SERVERS | красный `✖ no live zone` | `[n] netdiag` |

## 6. Настройки (overlay, тоглы)

```
╭─ Settings ─────────────────────────╮
│ [1] Auto-connect on start    ✖ off │
│ [2] Auto-diagnostics         ✔ on  │
│ [3] Notifications            ✔ on  │
│ [4] Auto-reconnect           ✔ on  │
│ [5] Kill-switch              ✔ on  │
│ [6] Preferred zone      best(auto) │
│ [esc] back                         │
╰────────────────────────────────────╯
```

## 7. Доступность / робастность

- Работает без цвета (`NO_COLOR`) — статус дублируется символом (●/▲/✖).
- Fallback на построчный режим, если не TTY (`--plain` или пайп).
- Ширина адаптивная (lipgloss измеряет строки, не ломается на эмодзи+кириллице).
- Все hotkeys также доступны цифрами (совместимость с текущим меню).

## 8. Технологии

- `charmbracelet/bubbletea` (Model/Update/View), `lipgloss` (стили/рамки),
  `bubbles` (list, viewport, spinner). Всё — чистый Go, статическая сборка.
- Header-данные из `livecheck.Check`, зоны из `measure.RankBest`, провайдеры из
  `cmdProviders` — логика уже есть, TUI её только рендерит.

## 9. План внедрения

1. `internal/tui` (bubbletea Model) — header + action bar + log pane.
2. Async-команды: connectCmd, testCmd, providersCmd → tea.Msg.
3. Overlays: zone-list, settings.
4. Fallback `--plain` → текущий построчный (сохранить).
5. Аудит топ-моделью (следующий артефакт).
