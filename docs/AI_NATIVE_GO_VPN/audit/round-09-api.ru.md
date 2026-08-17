# Раунд 9 аудита — machine-first API (Фаза 2, C2-7)

- Дата: 2026-08
- Область: `core/api` (envelope + router).
- Метод: adversarial проверка безопасности agent-facing API.

## Назначение

`core/api` — AI-native поверхность: агенты и харнессы управляют VPN
программно через стабильный версионированный JSON-конверт, а не парся
человеческий вывод. Замена bash socat+jq на нативный Go (encoding/json).

## Проверенные угрозы (agent-facing API — высокий риск)

### A1 — request smuggling через дублирующиеся ключи — ЗАКРЫТО
`encoding/json` при дубле ключа молча берёт последний → можно протащить другой
`operation` мимо валидатора. `rejectDuplicateKeys` отвергает дубли на верхнем
уровне (вложенные корректно пропускаются). Проверено эмпирически + тест
`TestDuplicateKeyRejected`.

### A2 — утечка внутренних ошибок (info disclosure) — ЗАКРЫТО
Обработчики возвращают либо контролируемый `*HandlerError` (стабильные
message_key), либо приводятся к generic `api.internal`. Сырой `err.Error()`
клиенту НИКОГДА не отправляется. Пути/креды не раскрываются.

### A3 — подмена request_id клиентом — НЕВОЗМОЖНА
`request_id` генерируется сервером (`r.reqID()`). В структуре `Request` поля
`request_id` нет вовсе (`grep` = 0) — клиент не может его задать.

### A4 — неизвестные поля / мусор — ОТВЕРГАЕТСЯ
`DisallowUnknownFields` → любой лишний ключ (`"evil":...`) отклоняется
(`TestUnknownFieldRejected`).

### A5 — авторизация мутаций — ЗАФИКСИРОВАНА
- read-only (`status.get/profiles.list/protocols.list/doctor.run/api.capabilities`)
  НЕ должны нести креды → иначе reject (`TestReadOnlyRejectsCredentials`);
- мутации (`connect/disconnect/reconnect/quick`) ОБЯЗАНЫ нести
  `action_id`+`authorization` → иначе `permission-denied`
  (`TestMutationRequiresAuthorization`).

Это паритет с bash allowlist (`api.capabilities|status.get|profiles.list|
protocols.list`).

### A6 — версия схемы — ПРОВЕРЯЕТСЯ
Неверный `api_version` → `unsupported-version` + `user_action_required`
(`TestUnsupportedVersion`).

## Конверт (паритет с v1)

```
{ "api_version":"1.0", "request_id":"...", "status":"ok",    "result":{...} }
{ "api_version":"1.0", "request_id":"...", "status":"error", "error":{
    "code":..., "message_key":..., "retryable":..., "user_action_required":... } }
```

## Дизайн

Router разделяет read-only/mutation политику до вызова обработчика; обработчики
регистрируются через `Handle(op, fn)`. Транспорт (unix-сокет с правами группы)
— отдельная задача Фазы 3 (пока Router транспортно-независим и тестируется
напрямую байтами).

## Проверки

- `core/api` — 9 тестов PASS; `-race` в общем прогоне чист.
- Дефектов не найдено; безопасность подтверждена по 6 векторам.
