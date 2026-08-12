# Android Mazzy VPN: архитектура и 15 итераций аудита

Статус: embedded AmneziaWG/WireGuard `device-test candidate`, не production.
APK уже собирает pinned userspace `libwg-go.so`, использует один собственный
`VpnService`, импортирует `.conf` в единый зашифрованный Keystore-envelope,
создаёт TUN, маршруты/DNS и объявляет CONNECTED только после handshake. Emulator,
physical-device leak/routing и signed-release gates ещё не закрыты, поэтому
registry честно остаётся `android: planned`.

## Решения

- Native Android app, `minSdk 26`, `targetSdk 36`, foreground `VpnService`.
- UI не запускает shell, root или LLM-generated commands. Управление идёт через
  typed actions и системный `VpnService.prepare()` permission boundary.
- Native AWG/WG профили проходят строгий parser до сохранения, ограничены
  256 KiB и целиком хранятся одним Android Keystore AES-GCM envelope.
- Pinned engine сам владеет TUN/DNS/routing. Ошибка permission, parse, storage,
  establish, protect или handshake переводит сервис в `ERROR` и выполняет
  teardown, не создавая ложный connected state.
- Bootstrap DNS endpoint выполняется через underlying network до создания TUN.
  После создания TUN профиль обязан иметь DNS и IPv4 default route; неописанные
  IP families не разрешаются обходным `allowFamily`. Always-on отключён до
  device-теста durable reboot recovery выбранного профиля. Текущий candidate —
  IPv4 full-tunnel: IPv6 блокируется, а не утекает, до отдельного dual-stack
  gate. `PersistentKeepalive` опционален; если задан, он должен быть допустимым
  интервалом WireGuard. Начальный handshake проверяется независимо от keepalive.
- Первым runtime gate может быть только WireGuard/AmneziaWG с реальным Android
  backend. Proxy-протоколы из каталога требуют отдельной проверки embedded
  sing-box/native adapter, лицензии, размера ABI и leak behaviour.

## Что было критично найдено

1. Android source отсутствовал, а roadmap описывал только planned capability.
2. Нельзя переиспользовать Linux shell adapter внутри APK: нет root/systemd,
   другие lifecycle и permission semantics.
3. `VpnService.prepare`, `protect`, foreground promotion, `onRevoke`, sleep/wake,
   network change и reboot являются частью connection state machine.
4. Profile catalog не равен поддержанному connect backend. Для VLESS, Hysteria2,
   Mieru, NaiveProxy, TUIC, Shadowsocks 2022, Trojan, AnyTLS и ShadowTLS Android
   support остаётся planned.
5. Реальный debug APK и четыре ABI собраны локально; это ещё не заменяет
   emulator и physical-device доказательства data plane.

## 15 итераций

Каждая итерация: implement -> static review -> unit tests -> adversarial review ->
record findings. Любой дефект сбрасывает `clean_streak` в ноль. Остановка
допускается только при пяти последовательных чистых проверках и закрытых
runtime/device gates.

| # | Gate | Обязательный результат |
|---:|---|---|
| 1 | Gradle foundation | reproducible wrapper, locked deps, debug build |
| 2 | Contract parity | schema/registry/profile validator tests |
| 3 | Secret storage | Keystore, no plaintext logs, rotation test |
| 4 | Import/rollback | SAF/share import, atomic commit and recovery |
| 5 | Permission lifecycle | prepare, revoke, denial and re-approval |
| 6 | Foreground lifecycle | Android 8+ notification, stop, crash cleanup |
| 7 | First engine | pinned native backend, ABI and license inventory |
| 8 | TUN loop | packet ownership, cancellation, backpressure |
| 9 | DNS/routing | IPv4/IPv6 routes, private DNS policy, kill switch |
| 10 | Network changes | Wi-Fi/mobile/captive portal/sleep-wake reconnect |
| 11 | Fault rollback | timeout, handshake failure, engine crash, rollback |
| 12 | Diagnostics | redacted evidence, no credentials, typed errors |
| 13 | Agent API | allowlisted read/diagnose actions, no shell execution |
| 14 | Device security | emulator plus real-device leak/permission tests |
| 15 | Release | signed APK/AAB, mapping, Data safety, five clean audits |

## Exit criteria

`mobile-android-1.0` нельзя закрыть по наличию Kotlin-файлов. Нужны real Gradle
build, instrumented tests on supported API levels, connected/disconnected DNS and
IPv6 leak evidence, route rollback after crash, signed artifact verification and
five consecutive clean audit records. До этого Android releases are `foundation`
or `preview` only.
