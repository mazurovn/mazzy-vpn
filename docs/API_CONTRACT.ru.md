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
Socket-activated Linux transport имеет статус `partial`: он принимает
`status.get`, `profiles.list` и три мутации `lifecycle.*`. Остальные операции и
не-Linux transports пока `planned`, поэтому полный кроссплатформенный daemon ещё
не заявлен готовым.

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

Повторная доставка того же action ID в пределах опубликованного окна хранения
не должна выполнять изменение дважды. Linux lifecycle dispatcher контролирует
это правило через постоянный root-readable action journal и по умолчанию
хранит 512 последних завершённых outcomes. Клиент не должен повторно
использовать вытесненный action ID как новую операцию. Остальные mutation
domains должны принять то же правило и опубликовать свою retention policy при
переносе за service boundary.

Dispatcher сохраняет rollback snapshot до перевода action в состояние
`running`. После аварии service следующая mutation под глобальной блокировкой
согласует все оставшиеся running records: восстанавливает snapshot и сохраняет
терминальный outcome `rolled-back` или `rollback-failed`, не оставляя action
навсегда busy и не выполняя его повторно.

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

В Linux `mazzy-vpn-api.socket` публикует `/run/mazzy-vpn/api-v1.sock` с
правами `0660 root:mazzy-vpn`. Одно соединение принимает один JSON request,
завершённый переводом строки, и возвращает один response. Service прекращает
чтение на настроенном byte limit ещё до JSON parsing, в том числе если клиент
не завершает слишком длинную строку. Мутации
сериализованы, ограничены deadline и сохраняются по action ID в root-only
state. Audit содержит только operation ID и результат — без payload и raw
backend output. По умолчанию журнал ограничен 512 завершёнными outcomes, а
audit-файл ротируется при 2 МиБ с одним root-only архивом. На системах с
ограниченным диском лимиты можно уменьшить, но нельзя отключать.

Desktop повторяет lifecycle request ровно один раз с тем же request/action ID
только после post-connect ошибки транспорта, когда outcome неопределён.
Неудачное первоначальное подключение может использовать typed compatibility
adapter, потому что request ещё не отправлен; post-connect неопределённость
никогда не переходит в `pkexec`.

`status.get` может передавать безопасные runtime details для parity терминала и
dashboard: desired mode, interface, возраст handshake, текущий public IP,
autostart, health monitor, число сбоев и состояние внешнего fallback. Для
совместимости minor-версии поля необязательны. VPN endpoint, имя/путь файла
профиля и конфигурация остаются запрещены.
