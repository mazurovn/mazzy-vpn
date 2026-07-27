# Основа локального API Mazzy VPN v1

Язык-независимый контракт опубликован в:

- [`../api/v1/manifest.json`](../api/v1/manifest.json) — операции, правила
  авторизации, audit, deadline и rollback;
- [`../api/v1/schema.json`](../api/v1/schema.json) — безопасные для frontend
  request, response и event envelopes.

Опубликованная версия контракта — `1.0`. Это постепенная граница совместимости
для задачи
[#5](https://github.com/mazurovn/mazzy-vpn/issues/5). Текущий транспорт
`cli-json-adapter` явно имеет статус `partial`: существующие безопасные JSON
status/profile outputs доступны, но клиенты ещё не отправляют все v1 request
envelopes через единый dispatcher. Метаданные контракта реализованы.
Защищённый локальный service пока явно имеет статус `planned`: публикация схемы
не означает, что финальный daemon уже реализован.

## Совместимость

- `api_version` имеет формат `major.minor`.
- Изменение major может удалять или менять смысл полей и требует новой схемы.
- Изменение minor может добавлять необязательные операции, поля и enum values.
- Неизвестная major-версия возвращает `unsupported-version`.
- Клиенты используют operation/error codes, а не локализованный текст вывода.

## Изменяющие операции

Каждая mutation требует:

- сгенерированный клиентом `action_id`;
- явный класс авторизации;
- ограниченный `deadline_ms`;
- очищенное audit event;
- объявленную rollback-семантику и итог rollback.

Повторная доставка того же action ID не должна выполнять изменение дважды.
Это правило idempotency будет контролировать защищённый локальный service.

## Безопасность frontend

Frontend status, responses и events используют непрозрачные ID и message keys.
Запрещены private/preshared keys, пароли, credentials, secrets, endpoints,
полные конфигурации и неограниченные filesystem paths. До входа в общий API
пути импорта заменяются короткоживущими непрозрачными `import_token`.

Сырой вывод backend не является частью стабильного контракта. Doctor передаёт
finding codes, severity, локализуемые message keys и отдельно авторизуемые
`fix_id`.

## Текущий доступ

`mazzy-vpn api-info --json` без root возвращает manifest установленного
контракта. Desktop предоставляет те же метаданные webview через read-only
Tauri-команду. CI проверяет синхронность CLI, manifest и schema.
