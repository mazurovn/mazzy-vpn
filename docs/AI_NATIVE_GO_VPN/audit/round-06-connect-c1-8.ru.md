# Раунд 6 аудита — connect-оркестратор и C1-8 (privileged smoke)

- Дата: 2026-08
- Область: `core/connect`, `core/cmd/mazzy-core-smoke`, C1-8.
- Метод: adversarial проверка порядка fail-closed + попытка живого запуска.

## Оркестратор connect

Порядок Up (паритет service_run): `IPv6 guard → engine(crypto+TUN) →
addresses+routes → DNS`. Teardown/unwind — строго в обратном порядке.

### Найденные и исправленные дефекты

| ID | Severity | Суть | Статус |
|---|---|---|---|
| C4 | P1 | IPv6-guard ставился по детерминированному имени `vpnaw0` ДО подтверждения реального имени интерфейса от движка. При переименовании ядром guard разрешил бы не тот интерфейс. | ИСПРАВЛЕНО: re-affirm guard по реальному `eng.Interface`, если имя отличается. Fail-safe: рассинхрон дропает IPv6, не течёт. |

### Подтверждённые сильные свойства (self-audit)

- **Нет leak-окна:** guard армится ДО создания интерфейса (тест
  `TestUpArmsIPv6GuardFirst`: при сбое guard ни одна route/link-команда не
  выполняется).
- **Полный unwind:** все 3 точки сбоя (engine/routes/dns) вызывают `unwind`,
  который откатывает всё применённое в обратном порядке.
- **Единый fwmark (G1):** оркестратор вычисляет `EffectiveMark` один раз и
  передаёт И движку, И routes.

## C1-8 — статус живого запуска (честно)

### Что доказано локально (без root)

1. **Статический самодостаточный бинарник:** `mazzy-core-smoke` — `statically
   linked`, встраивает amneziawg-go; ноль внешних VPN-инструментов.
2. **Реальный путь к ядру:** `tun.CreateTUN("vpnaw0", ...)` доходит до
   настоящего syscall и падает ровно на границе прав:
   `CreateTUN err: operation not permitted`. Это доказывает, что crypto+TUN
   реально подключены к ядру, а не к внешнему `awg-quick`.
3. **Ключи генерируются в Go** (`scripts/genkey.go`, X25519) — без `wg genkey`.
4. **Все юнит-тесты** (7 пакетов) — PASS, `-race` чисто, vendor verified.

### Что осталось (требует вашего sudo)

Полный живой туннель (`CreateTUN` + addresses + policy routing + guards)
требует root/CAP_NET_ADMIN. В этой среде:
- `sudo` требует интерактивный пароль (non-interactive запрещён);
- unprivileged userns-mapping заблокирован (`/proc/self/uid_map: Операция не
  позволена`), поэтому netns-обход недоступен.

Готов скрипт для запуска вами:

```
sudo core/scripts/smoke-c1-8.sh          # авто-генерит тест-конфиг
sudo core/scripts/smoke-c1-8.sh my.conf  # ваш реальный AmneziaWG-профиль
```

Ожидаемый результат: `UP: interface=vpnaw0 ...` → (3s) → `DOWN: clean`,
и это без единого вызова `awg`/`awg-quick`/`wg`/`wg-quick`.

## Вердикт

Оркестратор и smoke-код готовы и проверены до границы прав. C1-8 помечается
**DONE (code + local proof)**, живой root-прогон — **PENDING USER SUDO**
(инфраструктурное ограничение среды, не код).
