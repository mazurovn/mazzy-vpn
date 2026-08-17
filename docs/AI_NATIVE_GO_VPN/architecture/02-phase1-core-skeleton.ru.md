# Фаза 1 — реализованный скелет mazzy-core

Статус: в работе. Дата: 2026-08.

## Что уже сделано и проверено кодом

Модуль `core/` (Go 1.25, module `github.com/mazurovn/mazzy-vpn/core`).

| Пакет/файл | Назначение | Статус |
|---|---|---|
| `core/types.go` | Protocol, canonical/title/interface, DesiredState, Mode | ✅ |
| `core/profile/config.go` | парсер wg-quick `.conf` (Interface/Peer + Amnezia) | ✅ |
| `core/profile/validate.go` | валидация (паритет `validate_profile`) | ✅ |
| `core/profile/uapi.go` | рендер UAPI + **base64→hex** ключей | ✅ |
| `core/engine/wireguard/engine.go` | up/down через amneziawg-go как библиотеку | ✅ (код) |
| `core/cmd/engine-selftest` | доказательство автономного пути | ✅ |
| `core/build.sh` | pin Go toolchain 1.25 + static build | ✅ |
| `core/vendor/` | vendored amneziawg-go + x/crypto (12M) | ✅ |

## Доказательство автономности (C1-8, часть без root)

```
$ CGO_ENABLED=0 go build -o engine-selftest ./cmd/engine-selftest
$ file engine-selftest
ELF 64-bit ... statically linked ...   # НЕ динамический
$ ./engine-selftest
mazzy-core autonomous config path OK
protocol=amneziawg interface=vpnaw0 mtu=1420 peers=1
```

Путь `parse → validate → UAPI` работает **в процессе**, без `jq`, `awg`,
`awg-quick`, `wg`. Ключи конвертируются base64→hex (требование UAPI amneziawg-go,
подтверждено чтением `device/uapi.go`).

## Ключевые технические факты, зафиксированные при реализации

1. **UAPI ест hex, конфиги — base64.** `profile/uapi.go` конвертирует. Без
   этого движок молча не принял бы ключи.
2. **amneziawg-go не парсит `.conf`** — парсер целиком наш (`profile/config.go`),
   как и предсказал аудит R2.
3. **Address/DNS/MTU не входят в UAPI** — применяются нашими будущими
   `tun/routes/dns` (аудит R1). Пока распарсены и лежат в `Config`.
4. **Статическая линковка возможна** (`CGO_ENABLED=0`) — движок userspace,
   без cgo. Это и есть основа самодостаточного бинарника.

## Что дальше в Фазе 1

- C1-4a `routes` (таблицы/ip rule/fwmark), C1-4b `dns`, C1-5 `guard`
  (fail-closed nftables + IPv6). Это код, которого нет в amneziawg-go.
- C1-7 `state`+`lock` (намерение + single-flight).
- C1-8 privileged smoke: реально поднять `vpnaw0` под root на тестовом конфиге.

## Примечание про root

Создание TUN и настройка маршрутов требуют CAP_NET_ADMIN/root — как и текущий
bash. `engine-selftest` намеренно не трогает TUN, чтобы прогоняться без прав.
