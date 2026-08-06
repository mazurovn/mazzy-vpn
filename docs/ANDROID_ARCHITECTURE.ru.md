# Android Mazzy VPN: архитектура и 15 итераций аудита

Статус: `foundation`, не production. В репозитории появился нативный Kotlin
foundation с `VpnService`, импортом и зашифрованным хранилищем профилей. Реальный
protocol engine, TUN packet loop, DNS forwarder, routing и device leak tests ещё
не закрыты. Поэтому ни один протокол не переводится из `android: planned`.

## Решения

- Native Android app, `minSdk 26`, `targetSdk 35`, foreground `VpnService`.
- UI не запускает shell, root или LLM-generated commands. Управление идёт через
  typed actions и системный `VpnService.prepare()` permission boundary.
- Профили проверяются до сохранения по v1 contract, ограничены 256 KiB, секреты
  хранятся в Android Keystore AES-GCM; активный и предыдущий слоты обеспечивают
  атомарный rollback.
- Engine adapter должен быть pinned/reproducible и владеть TUN/DNS/routing.
  До его появления сервис намеренно заканчивает запуск в `ERROR`, не создавая
  ложный connected state и не направляя трафик в несуществующий туннель.
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
5. Gradle/SDK/device отсутствуют в текущем workspace, поэтому APK не объявляется
   собранным: CI обязан печатать `SKIP`, а не ложный `PASS`.

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
