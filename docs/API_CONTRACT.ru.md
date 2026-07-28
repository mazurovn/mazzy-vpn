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
status/profile outputs доступны, а CLI/TUI уже отправляют v1 envelopes для
`status.get`, `profiles.list` и `lifecycle.*` через единый dispatcher. Остальные
домены ещё используют совместимый прямой CLI-контур. Метаданные контракта
реализованы.
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
- Существующие schema-v1 документы `status --json` и `profiles --json` не
  меняют форму при появлении socket. Для явного получения API envelope
  используются `status --api-json` и `profiles --api-json`.

## Изменяющие операции

Каждая mutation требует:

- сгенерированный клиентом `action_id`;
- явный класс авторизации;
- ограниченный `deadline_ms`;
- очищенное audit event;
- объявленную rollback-семантику и итог rollback.

`deadline_ms` — монотонный бюджет mutation, а не обещание бросить обязательную
защитную работу при истечении времени ответа. Linux dispatcher запускает
бюджет после проверки mutation envelope, вычитает время lock/preflight и
передаёт executor оставшиеся миллисекунды без округления вверх. После истечения
бюджета executor не запускается. Таймауты executor и refresh завершают всю
process group, чтобы shell helper не продолжал старую операцию параллельно с
rollback. Обязательный rollback и crash reconciliation используют отдельные
ограниченные таймауты системного сервиса, поэтому ответ
может прийти позже `deadline_ms`, пока завершается rollback. Linux-клиенты
резервируют для итогового outcome ограниченный completion grace в 60 секунд.
Незавершённый rollback переводит API в recovery-only mode и не объявляется
успешным.

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
не завершает слишком длинную строку. До dispatch принимается ровно один
top-level JSON object; повторные envelope/payload keys, включая записанные
через JSON Unicode escapes, отклоняются. Обновление query cache ограничено
меньшим из необязательного query deadline и server refresh cap; при timeout
может быть возвращён уже существующий restricted cache. Прерванное обновление
удаляет свои временные файлы. Мутации
сериализованы, ограничены deadline и сохраняются по action ID в root-only
state. Audit содержит только operation ID и результат — без payload и raw
backend output. Mutation не запускается, пока её начальное audit event не
сохранено. Если terminal audit event нельзя записать уже после изменения
state, завершённый action сохраняет idempotency, но API переходит в
recovery-only mode для явной проверки администратором. По умолчанию журнал
ограничен 512 завершёнными outcomes, а
audit-файл ротируется при 2 МиБ с одним root-only архивом. На системах с
ограниченным диском лимиты можно уменьшить, но нельзя отключать.

Если crash reconciliation не может прочитать pre-action snapshot или
восстановить его, daemon сохраняет root-only recovery marker и отклоняет все
следующие API mutations. После ручной проверки и исправления текущего состояния
администратор должен явно подтвердить его командой
`sudo mazzy-vpn _api-clear-recovery --acknowledge-current-state`. Marker никогда
не снимается по таймеру или после постороннего успешного request.

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

Установленные CLI и TUI используют socket без `sudo` для status, списка
профилей, connect, quick, reconnect и disconnect. Клиент передаёт только
непрозрачный `profile_id`, задаёт ограниченный query refresh deadline,
ограничивает время и byte size ответа и принимает ровно один response document
с совпадающими `api_version`/`request_id`. При потере ответа тот же request
автоматически повторяется с тем же `action_id`, поэтому daemon возвращает
сохранённый outcome, не выполняя мутацию второй раз. Если transport остаётся
неопределённым, клиент не запускает ту же операцию через `sudo`. При ошибке
mutation печатается action ID для audit и recovery.

Desktop применяет то же правило одного идентичного retry к lifecycle requests.
Desktop также читает ответ до bounded EOF и принимает ровно один совпадающий
JSON document.
После неудачного первоначального подключения к socket разрешён typed
compatibility adapter, потому что request ещё не отправлялся. Любая
неопределённость после подключения возвращается пользователю и никогда не
переходит в `pkexec`.

Для Unix-socket клиента installer автоматически устанавливает `socat`.
Пользователь должен состоять в группе `mazzy-vpn`; после первого добавления в
группу может потребоваться новый login session.
