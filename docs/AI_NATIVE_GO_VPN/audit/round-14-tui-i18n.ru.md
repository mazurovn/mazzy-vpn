# Раунд 14 аудита — TUI + i18n (Фаза 2, C2-8) + качество кода

- Дата: 2026-08
- Область: `core/i18n`, `core/tui`.
- Метод: adversarial проверка полноты локализации + quality gate
  (staticcheck/vet/gofmt/-race), по запросу заказчика.

## Реализация

- `core/i18n` — 6 языков (ru/en/de/zh/ja/ko), нормализация локали
  (`de_DE.UTF-8`→`de`), data-driven каталог (map, не switch-спагетти),
  Translator с fallback EN→key.
- `core/tui` — модель меню, отделённая от рендеринга (unit-тестируется без TTY):
  Action↔Item↔i18n-key, `Select(n)`, смена языка.

## Найденные и исправленные дефекты

### I3 — P1 — рассинхрон каталога i18n и message_keys из core/verify
`core/verify` эмитит ~7 `verify.*` verdict-ключей, но каталог содержал только 2.
Остальные рендерились бы как сырые ключи (ущербность UX для агентов/UI).

**Исправление:** добавлены все verdict-ключи (verified, failed.inactive,
failed.no-egress, warning.route-mismatch/ipv6-leak/country-mismatch/dns-leak) с
полной локализацией на 6 языков. Введён `verifyMessageKeys` + regression-тест
`TestVerifyKeysLocalized`, который ловит будущий рассинхрон.

### I1 — документировано — DefaultLang EN vs bash RU
Bash по умолчанию использовал `ru`; Go — `en` (international-first). Это
осознанный продуктовый выбор; локаль пользователя всё равно имеет приоритет
через `Resolve`. Задокументировано в коде.

## Quality gate (запрос заказчика: dead code / hardcode / spaghetti / логика)

| Проверка | Результат |
|---|---|
| staticcheck (dead code) | чисто |
| go vet | чисто |
| gofmt | всё отформатировано |
| go test -race | 16 пакетов PASS |
| хардкод UI-строк в логике tui | НЕТ — все метки из i18n-ключей |
| спагетти | НЕТ — каталог data-driven (map), не гигантский switch |
| пустые функции | НЕТ |

### Анти-«ущербность» гейты (тестами)
- `TestCatalogCompleteness` — КАЖДЫЙ ключ имеет непустой перевод на ВСЕ 6
  языков. Пропуск перевода = падение теста.
- `TestVerifyKeysLocalized` — синхронизация с core/verify.
- `TestMenuSelectOutOfRange` — Select не паникует на 0/-1/999.

## Дизайн-решение

Меню отделено от рендеринга: `Menu.Lines()` возвращает готовые строки,
`Menu.Select(n)` мапит выбор в `Action`. Конкретный рендерер (построчный или
bubbletea) — тонкая обёртка Фазы 3, логика меню уже покрыта тестами.

## Проверки

- `core/i18n` (8) + `core/tui` (6) = 14 тестов PASS.
- Полный прогон: 16 пакетов, `-race` PASS, staticcheck/vet/gofmt чисто.
- Статическая сборка сохранена.
