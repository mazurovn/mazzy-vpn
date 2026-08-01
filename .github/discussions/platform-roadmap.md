# Platform roadmap / План платформ

| Surface / Интерфейс | Current status / Текущий статус | Tracking |
|---|---|---|
| CLI/TUI | Published Linux 1.3 release; service parity and versioned automation follow-up | [#12](https://github.com/mazurovn/mazzy-vpn/issues/12) |
| Desktop Linux | Published unsigned 0.3 preview; issue #31 closed and release checks green | [#4](https://github.com/mazurovn/mazzy-vpn/issues/4) |
| Desktop Windows | UI preview; native service/backend required | [#7](https://github.com/mazurovn/mazzy-vpn/issues/7) |
| Desktop macOS | UI preview; Network Extension/signing required | [#10](https://github.com/mazurovn/mazzy-vpn/issues/10) |
| Android | planned native `VpnService` client | [#13](https://github.com/mazurovn/mazzy-vpn/issues/13) |
| iOS | planned native Network Extension client | [#14](https://github.com/mazurovn/mazzy-vpn/issues/14) |

## Русский

Каждая платформа получает production-статус независимо и только после своего
машинно проверяемого release gate. UI preview, подпись или номер версии сами по
себе не подтверждают работу VPN backend. Android/iOS будут нативными клиентами,
а не WebView/Desktop wrapper.

Порядок: подтвердить исправленный RustSec gate #31 на `main` и выпустить Linux Desktop 0.3 → общий versioned API и
Linux Desktop 1.0 → независимые Windows/macOS previews → Android/iOS proof of
concept → отдельные mobile alpha/beta/production.

## English

Each platform receives production status independently and only after its
machine-validated release gate passes. A UI preview, signature or version
number does not prove that a VPN backend works. Android/iOS will be native
clients, not WebView/Desktop wrappers.

Order: confirm the resolved #31 RustSec gate on `main` and publish Linux Desktop 0.3 → common versioned API
and Linux Desktop 1.0 → independent Windows/macOS previews → Android/iOS proofs
of concept → independent mobile alpha/beta/production.

Full plan / Полный план:
https://github.com/mazurovn/mazzy-vpn/wiki/Platform-Roadmap

Community voting / Голосование сообщества:
https://github.com/mazurovn/mazzy-vpn/discussions/categories/polls
