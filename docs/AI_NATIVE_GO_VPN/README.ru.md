# AI-NATIVE-GO-VPN — рабочая документация

Это корневой индекс SDD-документации полного переписывания MAZZY_VPN на
нативный Go: единый движок `mazzy-core` и три самодостаточных продукта — CLI,
Desktop и Android.

## Структура

| Раздел | Содержание |
|---|---|
| [`sdd/`](sdd/) | Software Design Documents: концепция, требования, дизайн |
| [`adr/`](adr/) | Architecture Decision Records (принятые решения) |
| [`architecture/`](architecture/) | Архитектурные артефакты: слои, диаграммы, контракты |
| [`backlog/`](backlog/) | Бэклог задач рефакторинга по фазам |
| [`audit/`](audit/) | Отчёты консилиума и циклов проверок |

## Ключевые решения (кратко)

- **Язык: Go (нативно).** Обоснование — [`adr/ADR-0001-go-native.ru.md`](adr/ADR-0001-go-native.ru.md).
- **Три самодостаточных продукта.** Каждый работает независимо: Desktop и
  Android НЕ зависят от CLI. — [`adr/ADR-0002-three-autonomous-products.ru.md`](adr/ADR-0002-three-autonomous-products.ru.md).
- **Автономность от внешних библиотек.** `mazzy-core` встраивает
  `amneziawg-go` как библиотеку (не FFI, не внешний бинарник). Ноль
  `git clone`/`apt install` в рантайме. — [`adr/ADR-0003-autonomous-engine.ru.md`](adr/ADR-0003-autonomous-engine.ru.md).
- **Двухслойная AI-NATIVE архитектура** (control plane + secure data plane) —
  [`sdd/01-concept.ru.md`](sdd/01-concept.ru.md).

## Правила проекта

См. [`sdd/00-project-rules.ru.md`](sdd/00-project-rules.ru.md) — правила
публикации, локального форка Control Panel, git-исключений и аудита.
