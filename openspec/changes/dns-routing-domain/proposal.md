# Proposal

## Why

Инцидент 2026-09-21: в логе демона пробы egress уходили на подменённые адреса (`8.6.112.0`, `8.47.69.0` для `api.ipify.org`, реальные — Cloudflare `104.26.x.x`). Причина: `core/dns` при бэкенде systemd-resolved выполняет только `resolvectl dns <iface> <servers>`. Роутинг-домен не задаётся, поэтому физический аплинк (`wlp3s0f0` → `192.168.1.1`, DNS провайдера) остаётся равноправной default-route-областью, и resolved рассылает запросы на **оба** линка. Итог: утечка DNS мимо туннеля и ложные «egress lost» при подменённых ответах (половина проб таймаутит) — это вторая половина «VPN постоянно отваливается».

## What Changes

- Бэкенд `resolvectl` дополнительно выполняет `resolvectl domain <iface> ~.` и `resolvectl default-route <iface> yes` — туннельный линк становится предпочтительным резолвером для всех имён (так делают wg-quick, NM-плагины, AdGuard).
- `Manager.Available` — переопределяемая проверка бинарников для тестов; тест фиксирует точную последовательность команд и `revert` при `Down`.
- Vendor-копия core в `mazzy-vpn-cli` пересинхронизирована.

## Capabilities

### New Capabilities
- `tunnel-dns`: требования к настройке DNS туннеля на хосте с systemd-resolved.

### Modified Capabilities
(нет)

## Impact

- Код: `core/dns/dns.go`, `core/dns/dns_test.go`, vendor.
- Поведение: все DNS-запросы при поднятом туннеле идут через DNS из профиля; после `Down` — `resolvectl revert` возвращает прежнее состояние. Бэкенд `resolvconf` не меняется.
