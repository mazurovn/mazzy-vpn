# План платформ

Источник истины:
[PLATFORM_ROADMAP.ru.md](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/PLATFORM_ROADMAP.ru.md)
и [матрица функций](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/FEATURE_PARITY.md).

| Поверхность | Сейчас | Следующий production gate |
|---|---|---|
| CLI/TUI | Linux 1.2 | стабильные schema/error codes, полный TUI service parity — [#12](https://github.com/mazurovn/mazzy-vpn/issues/12) |
| Desktop Linux | функциональный 0.2 preview | versioned service API, policy/localization/accessibility, signed lifecycle — [#4](https://github.com/mazurovn/mazzy-vpn/issues/4) |
| Desktop Windows | UI preview | Windows Service, WireGuard/Wintun, signed installer — [#7](https://github.com/mazurovn/mazzy-vpn/issues/7) |
| Desktop macOS | UI preview | Network Extension, signing/notarization — [#10](https://github.com/mazurovn/mazzy-vpn/issues/10) |
| Android | planned | native `VpnService`, device tests, signed AAB — [#13](https://github.com/mazurovn/mazzy-vpn/issues/13) |
| iOS | planned | Network Extension, real-device tests, TestFlight/App Store — [#14](https://github.com/mazurovn/mazzy-vpn/issues/14) |

Каждая платформа продвигается независимо. UI preview, подпись или номер версии
не означают готовый VPN backend. Android/iOS создаются как нативные клиенты, а
не WebView/Desktop wrapper.

## Порядок

1. CLI/TUI 1.3 и Linux Desktop 0.3.
2. Versioned core/API и Linux Desktop 1.0.
3. Независимые Windows/macOS previews и platform gates.
4. Android/iOS proof-of-concept на общем профильном контракте.
5. Отдельное продвижение mobile alpha → beta → production.

---

<a id="english"></a>

# Platform roadmap

Sources of truth:
[PLATFORM_ROADMAP.en.md](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/PLATFORM_ROADMAP.en.md)
and the [capability matrix](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/FEATURE_PARITY.md).

| Surface | Current state | Next production gate |
|---|---|---|
| CLI/TUI | Linux 1.2 | stable schemas/error codes and complete TUI service parity — [#12](https://github.com/mazurovn/mazzy-vpn/issues/12) |
| Desktop Linux | functional 0.2 preview | versioned service API, policy/localization/accessibility and signed lifecycle — [#4](https://github.com/mazurovn/mazzy-vpn/issues/4) |
| Desktop Windows | UI preview | Windows Service, WireGuard/Wintun and signed installer — [#7](https://github.com/mazurovn/mazzy-vpn/issues/7) |
| Desktop macOS | UI preview | Network Extension, signing and notarization — [#10](https://github.com/mazurovn/mazzy-vpn/issues/10) |
| Android | planned | native `VpnService`, device tests and signed AAB — [#13](https://github.com/mazurovn/mazzy-vpn/issues/13) |
| iOS | planned | Network Extension, real-device tests and TestFlight/App Store — [#14](https://github.com/mazurovn/mazzy-vpn/issues/14) |

Each platform is promoted independently. A UI preview, signature or version
number does not imply a working VPN backend. Android/iOS are native clients,
not WebView/Desktop wrappers.

## Order

1. CLI/TUI 1.3 and Linux Desktop 0.3.
2. Versioned core/API and Linux Desktop 1.0.
3. Independent Windows/macOS previews and platform gates.
4. Android/iOS proofs of concept on the common profile contract.
5. Independent mobile alpha → beta → production promotion.
