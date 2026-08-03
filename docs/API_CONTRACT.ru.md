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
`status.get`, `profiles.list`, `protocols.list`, `planner.evaluate`,
`tests.probe`, `tests.verify-egress` и `lifecycle.*` через единый
dispatcher. Остальные
домены ещё используют совместимый прямой CLI-контур. Метаданные контракта
реализованы.
Socket-activated Linux transport имеет статус `partial`: он принимает
`status.get`, `profiles.list`, `protocols.list`, ограниченные queries `tests.probe` и
`tests.verify-egress`, read-only `planner.evaluate`, а также три мутации
`lifecycle.*`. Остальные операции и
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
top-level JSON object; повторные object keys на любой глубине, включая
записанные через JSON Unicode escapes, отклоняются. Обновление query cache ограничено
меньшим из необязательного query deadline и server refresh cap; при timeout
может быть возвращён уже существующий restricted cache. Прерванное обновление
удаляет свои временные файлы. Мутации
сериализованы, ограничены deadline и сохраняются по action ID в root-only
state. Audit содержит только operation ID и результат — без payload и raw
backend output. Mutation не запускается, пока её начальное audit event не
сохранено. Pre-action snapshot, running record и initial audit синхронизируются
вместе с parent directories до запуска lifecycle child. Completed record и
удаление snapshot также синхронизируются до terminal success. Если terminal
audit event нельзя записать уже после изменения
state, завершённый action сохраняет idempotency, но API переходит в
recovery-only mode для явной проверки администратором. По умолчанию журнал
ограничен 512 завершёнными outcomes, а
audit-файл ротируется при 2 МиБ с одним root-only архивом. На системах с
ограниченным диском лимиты можно уменьшить, но нельзя отключать.

Если crash reconciliation не может прочитать pre-action snapshot или
восстановить его, daemon сохраняет root-only recovery marker и отклоняет все
следующие API mutations. Hardened root boot oneshot согласует прерванные actions
под общим mutation lock до test recovery, `vpnctl.service`, health
remediation и API socket. Невозможность подготовить защищённые
каталоги или получить lock также сохраняет marker. Test recovery
требует успешного прохождения этого gate и запускается только после
него. Пока marker существует, managed-service start, test recovery и
health remediation fail closed. После ручной проверки и исправления текущего состояния
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

`protocols.list` возвращает очищенный каталог из 13 протоколов и policy
оркестрации. Detection/import/diagnostics отделены от connection readiness для
каждой платформы, поэтому запись каталога нельзя принять за готовый backend.
В ответе есть только публичные identifiers formats, engines и transports;
endpoint, credential, profile и backend config отсутствуют. Источник истины:
[`../protocols/v1/registry.json`](../protocols/v1/registry.json).

`planner.evaluate` — read-only оценка с обязательным ограниченным deadline.
Payload содержит workload и 1–128 уникальных opaque profile IDs с полным
ограниченным evidence object. Пять hard gates вычисляет server из текущего
локального состояния, а не caller: backend готов, профиль валиден, файл профиля
доступен только backend, защищённый rollback storage готов и Linux support имеет
статус `implemented`. Storage gate подтверждает только готовность места для
journal/snapshot; возможность rollback конкретного кандидата остаётся частью
будущего execution. Score и rank получает только кандидат, прошедший все gates.

Policy v1 выделяет 30 баллов recent outcome, 25 censorship fit, 20
reachability, 15 latency/loss и 10 workload fit. Наблюдаемое health evidence
(`recent_outcome`, reachability и latency/loss) старше 900 секунд даёт ноль;
`censorship-fit` backend выводит из protocol catalog, а `workload-fit` — из
workload, класса протокола и transports. Caller не может назначить эти факторы.
При равном score стабильным tie-breaker является opaque profile ID. Response
содержит reason codes и баллы факторов,
но не display name, endpoint, filename, path, configuration или credential.
Ответ всегда имеет `dry_run: true`: operation не подключает VPN и не выполняет
failover. Candidate validation получает тот же абсолютный monotonic deadline,
включая subprocess OpenVPN parser. CLI принимает один JSON payload до 64 КиБ
через stdin и ограничивает расширенный response 1 МиБ:

```bash
jq -n --arg profile_id "$PROFILE_ID" '{
  workload: "llm-streaming",
  candidates: [{
    profile_id: $profile_id,
    evidence: {
      recent_outcome: "success", consecutive_failures: 0,
      reachability: "reachable", latency_ms: 80, loss_percent: 0,
      evidence_age_seconds: 30
    }
  }]
}' | mazzy-vpn planner evaluate --stdin --json
```

Caller может влиять на dry-run rank только через ограниченное health evidence и
не может обойти backend-owned gate. Сбор доверенного history, authorized
execution и автоматический failover остаются за границами этой operation.

`tests.probe` проверяет все профили выбранного protocol scope с отдельным
таймаутом endpoint и ограниченной параллельностью 1–8 workers. Результат
содержит opaque profile ID, безопасное display name, protocol, флаги
default/active, transport, `reachability`, необязательный целочисленный
`latency_ms`, источник ICMP/TCP и message key. Endpoint в ответ не попадает.
`reachable` означает только ответ ICMP или настроенного TCP service и не
доказывает VPN credentials, handshake, routes или DNS через туннель. Для UDP
успешный DNS без ответа ICMP получает `unknown`, а не `unreachable`: рабочие
серверы часто блокируют ping, а безопасного универсального UDP handshake нет.
Полное доказательство остаётся за транзакционным live-test с rollback. Server
применяет request deadline ко всей worker group и сериализует batch probes
глобальным lock, чтобы параллельные socket clients не умножали сетевую нагрузку.

`tests.verify-egress` — read-only query с общим lock и ограниченным deadline.
Payload содержит только `timeout_seconds` и явный выбор `include_speed`.
Response передаёт:

- активные tunnel protocol/display name/interface;
- interface-bound и default IPv4 с признаком равенства;
- interface-bound/default IPv6 и флаг потенциальной утечки;
- ожидаемую/наблюдаемую страну, agreement providers и не более двух
  валидированных provider records;
- состояние настроенного DNS route;
- необязательный ограниченный speed sample;
- verdict, message key и уникальные finding codes.

Engine принимает geo record только для точного interface-bound IPv4. Для
`verified` нужны два разных providers, согласовавших страну, одинаковый
default/interface IPv4 egress, отсутствие потенциальной IPv6-утечки,
full-tunnel DNS и пустой список findings. Strict Desktop parser повторно
проверяет эти инварианты и отклоняет неизвестные поля, неверные семейства IP,
дубликат provider, несовпадение provider IP и необъяснённый non-verified
verdict. Response не содержит VPN endpoint, profile path, key или
configuration. По умолчанию `include_speed=false`; 5-МБ transfer никогда не
запускается неявно.

`tests.verify-service-egress` — отдельный read-only query. Его strict payload
содержит ровно `service` (`notebooklm`, `openai` или `all`) и целый
`timeout_seconds` от 3 до 15. Engine отправляет credential-free HEAD только на
встроенный HTTPS allowlist, привязывает запрос к выбранному VPN interface,
отключает redirects и proxy environment и ограничивает response headers.
Strict result содержит только schema/timestamp/scope, service ID,
reachability, egress eligibility, reason code и необязательный HTTP status.
NotebookLM доверяет только точным redirects unsupported-location и home.
OpenAI считает 401 или 405 с точным `Allow: POST` достижением auth boundary;
403 означает edge denial, а 429, 5xx и неизвестные ответы остаются
indeterminate. Network error даёт unreachable/indeterminate. URL, headers,
body, address, account и credentials не возвращаются и не сохраняются. Query
не используется health recovery или planner и не доказывает authentication,
subscription, organization или content access.

Установленные CLI и TUI используют socket без `sudo` для status, списка
профилей, batch endpoint probe, egress verification, connect, quick, reconnect
и disconnect. Клиент
передаёт только
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
Проверка списка локаций в Desktop использует структурированный `tests.probe`
и связывает очищенный результат с opaque profile ID, не разбирая raw CLI text.
Desktop egress verification принимает strict structured result
`tests.verify-egress`. После неудачного первоначального подключения к socket
разрешён typed
compatibility adapter, потому что request ещё не отправлялся. Любая
неопределённость после подключения возвращается пользователю и никогда не
переходит в `pkexec`.

Для Unix-socket клиента installer автоматически устанавливает `socat`.
Пользователь должен состоять в группе `mazzy-vpn`; после первого добавления в
группу может потребоваться новый login session.
