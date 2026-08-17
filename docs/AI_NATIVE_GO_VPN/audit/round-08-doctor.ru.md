# Раунд 8 аудита — doctor (Фаза 2, C2-5)

- Дата: 2026-08
- Область: `core/doctor`.
- Метод: adversarial проверка автономности (что doctor больше НЕ требует).

## Главный результат: автономность стала измеримой

Легендарная боль текущего CLI — doctor проверял **20 внешних инструментов**:

```
awg  awg-quick  wg  wg-quick  jq  socat  python3  curl  flock  getent
sha256sum  setfacl  openvpn  timeout  systemctl  systemd-run  ping  bash  ip  nft
```

Новый `core/doctor` ссылается **только** на базовые ОС-инструменты:

```
ip   nft   resolvectl (или resolvconf)
```

Всё остальное:
- `awg/awg-quick/wg/wg-quick` → встроенный движок (amneziawg-go);
- `jq` → `encoding/json`; `socat` → нативный net; `python3` → `time`;
- `curl` → `net/http`; `timeout` → `context`; `flock` → `x/sys/unix`;
- `getent/sha256sum/setfacl` → Go stdlib.

Это прямое доказательство ADR-0003/R1: обвязка исчезла из требований.

## Проверенные утверждения

### D1 — doctor не переизобретает legacy-зависимости — ЗАФИКСИРОВАНО ТЕСТОМ
`TestDoctorNeverRequiresLegacyVPNTools`: окружение БЕЗ единого legacy-tool, но
с базовыми `ip/nft/resolvectl` + `/dev/net/tun` + root — **полностью healthy**.
Если кто-то вернёт зависимость от `awg`/`jq`, тест упадёт.

### D2 — явный положительный чек движка
Doctor выводит `AmneziaWG engine: embedded (no awg/awg-quick required)` —
делает автономность видимой, а не молчаливой.

### D3 — уровни severity корректны
- нет `ip`/`nft`/`/dev/net/tun` → **FAIL** (реально нельзя поднять тоннель);
- нет DNS-бэкенда → **WARN** (VPN DNS может не примениться);
- не root → **WARN** (не FAIL: диагностика работает и без прав).

Живой прогон на хосте: `OK=5 WARN=1 FAIL=0 healthy=true` (WARN — только
privilege, т.к. запуск без root).

## Структурный дизайн

`doctor.Run(ctx, Environment)` — `Environment` абстрагирует хостовые запросы
(`LookPath/FileExists/IsRoot`), поэтому диагностика unit-тестируема без хоста.
`Report`/`Check` сериализуются в JSON (для CLI/Desktop/API — одна модель).
Пути/креды не раскрываются (portable).

## Отложено (осознанно)

- Проверки systemd-юнитов — вернутся в Фазе 3, когда появятся сервисные юниты
  Go-CLI (сейчас юниты — из старого стека).
- fallback/transition-marker чеки — вернутся вместе с `bootrecovery` (C2-3).

## Проверки

- `core/doctor` — 6 тестов PASS; `-race` в общем прогоне чист.
- Статическая сборка сохранена.
