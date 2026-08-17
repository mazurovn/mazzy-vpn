# ADR-0003: Автономный движок mazzy-core без внешних зависимостей

- Статус: **ПРИНЯТО** (зафиксировано заказчиком)
- Дата: 2026-08

## Проблема (текущее состояние)

Сейчас CLI **не автономен**:

- `install.sh` клонирует и собирает `amneziawg-tools` и `amneziawg-go` из
  GitHub (`git clone`, `make`, `go build`) либо ставит через PPA/COPR/apt.
- Рантайм зависит от системных бинарников: `awg`, `awg-quick`, `wg`,
  `wg-quick`, `openvpn`, `nmcli`, `jq`, `socat`, `python3`.

Это нарушает требование «CLI полностью автономен и самодостаточен».

## Решение

`mazzy-core` — встроенный движок, линкуемый в каждый продукт:

### WireGuard / AmneziaWG
- Встроить `amneziawg-go/v3` **как Go-библиотеку** (vendored в `core/`).
- Реализовать `up/down` полностью в коде: создание TUN, применение конфига,
  routes, DNS, nftables leak-guard, fwmark, ip rule.
- **Убирает зависимость** от `awg`, `awg-quick`, `wg`, `wg-quick` и от
  `git clone`/`make`/`go build` в install.

> **Уточнение по итогам Раунда 1 аудита (R1/R2, важно):**
> `amneziawg-go.IpcSet()` принимает только UAPI (ключи, endpoint,
> allowed_ip, Amnezia-обфускация). Он **НЕ** настраивает адрес интерфейса,
> маршруты, ip rule/fwmark, DNS, MTU и **не парсит** wg-quick `.conf`.
> Следовательно vendored-библиотека = крипто-движок + TUN-обмен.
> Вся обвязка wg-quick (~41 операция route/dns/rule/fwmark) и **собственный
> парсер `.conf`** — это наш код (задачи C1-4a/b/c, C1-6a).

### OpenVPN
- Этап 1: vendored **статический бинарник** внутри пакета продукта (автономно
  от системного `apt install openvpn`).
- Этап 2 (позже): оценка нативной реализации.

### L2TP/IPsec
- Сохранить через системный NetworkManager там, где это неизбежно, но не
  делать его обязательным для core-функций.

### Замена утилит
| Внешняя утилита | Замена в Go |
|---|---|
| `jq` (312 вызовов) | `encoding/json` |
| `timeout` (191) | `context.WithTimeout` |
| `awk`/`sed`/`grep` | обычный Go-код |
| `socat` | нативный net-listener локального API |
| `python3` (CLOCK_MONOTONIC) | `time` / `golang.org/x/sys/unix` |
| `awg-quick`/`wg-quick` | нативная конфигурация интерфейса |

## Критерий приёмки

- `doctor` больше **не проверяет и не требует** `awg`/`awg-quick`/`jq`/`socat`.
- `install` **не выполняет** `git clone` внешних VPN-репозиториев и не требует
  обязательного `apt install` VPN-бэкендов.
- Один статический бинарник поднимает AmneziaWG-туннель на чистой системе.

## Сборочное ограничение (R3)

`amneziawg-go/v3` требует `go 1.25.x`. На PATH по умолчанию может быть
старый `go` (у нас 1.22.10). Сборка обязана пинить toolchain 1.25.x
(задача C1-1a).

## Последствия

- Vendored `amneziawg-go` фиксируется по коммиту (провенанс), обновляется
  осознанно — как сейчас пинятся `AWG_*_COMMIT` в install.sh.
- Размер бинарника растёт (встроенный движок) — приемлемо ради автономности.
- Для TUN/nftables нужны CAP_NET_ADMIN/root — как и сейчас.
