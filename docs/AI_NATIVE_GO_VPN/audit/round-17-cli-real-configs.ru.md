# Раунд 17 аудита — CLI-бинарник + реальные конфиги + лицензия/авторство

- Дата: 2026-08
- Область: `mazzy-vpn-cli/`, парсер профилей, лицензия, защита секретов.
- Метод: проверка на РЕАЛЬНЫХ конфигах заказчика (AmneziaWG + OpenVPN).

## НАЙДЕННЫЙ ДЕФЕКТ (пойман только реальными конфигами)

### P1-1 — P0 — парсер отвергал реальные AmneziaWG-конфиги
Реальный `.conf` заказчика содержит поля, которых не было в парсере:
- `S3`, `S4` (доп. junk-размеры);
- `i1`..`i5` (obfuscation-chain строки, `<b 0x...>`);
- **H1..H4 как ДИАПАЗОНЫ** (`127765534-127831069`), а не одиночные uint32.

Старый парсер падал: `unknown [Interface] key "s3"` и не принял бы H-диапазоны.

**Исправление:** `AmneziaParams` переписан на opaque-строки (map key→raw value).
amneziawg-go.IpcSet — авторитетный парсер (H-ranges через `UintRange.FromString`,
i-chains через `newObfChain`), поэтому значения форвардятся ВЕРБАТИМ. Убран
мёртвый `atou32Field`. Regression: `TestParseRealWorldAmneziaFields` (S3/S4,
H-range, i1-chain → parse+validate+UAPI).

**Проверка на всех реальных конфигах:** 6/6 AmneziaWG + OpenVPN — распознаны и
валидны.

## CLI-бинарник (P3-1)

`mazzy-vpn-cli/cmd/mazzy-vpn` — автономный Go CLI (линкует core, ноль внешних
awg/jq). Команды: `doctor`, `list DIR`, `validate FILE`, `version`, каждая с
`--json` (machine-first для агентов).

Живая проверка:
- `doctor` → OK=5 WARN=1 FAIL=0 (без awg/awg-quick);
- `list VPN_KEY` → 6 AmneziaWG OK;
- `list "other os"` → OpenVPN распознаны;
- `validate --json` → `{file, protocol, valid}`.

## Безопасность (запрос заказчика: не утекли личные данные/секреты)

| Проверка | Результат |
|---|---|
| конфиги внутри репо? | НЕТ (внешние, `/home/.../Документы`) |
| `conf` в .gitignore | да |
| секреты в выводе CLI (--json) | НЕТ (только file/protocol/valid) |
| личные пути в коде | НЕТ |
| public audit | OK |

## Лицензия и авторство (запрос заказчика)

- SPDX-заголовок `AGPL-3.0-or-later` + `Copyright © 2026 Nik m (@mazurovn)`
  добавлен в **53** core-файла и все CLI-файлы.
- Соответствует существующим LICENSE (AGPL-3.0) и AUTHORS.md (Nik m @mazurovn).
- go.mod модулей помечены автором.

## Quality gate

| Проверка | Результат |
|---|---|
| go vet | чисто |
| staticcheck | чисто |
| gofmt | чисто |
| go test | 19 пакетов PASS |
| dead code (atou32Field) | удалён |

## Итог

CLI готов к началу перехода: реальные конфиги парсятся, вывод машиночитаем и
безопасен, авторство и лицензия проставлены, секреты не утекают. Дальше —
привилегированные connect/disconnect в CLI (следующий шаг P3-1).
