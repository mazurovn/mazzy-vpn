# Android аудит 2026-08-06

## Результат

Создан native Android foundation, но Android 1.0 не готов и намеренно не
помечен готовым. Реальный Gradle/APK/device gate не выполнен: в рабочем
окружении отсутствуют Android SDK, `adb`, Gradle и Gradle Wrapper.

Проверено локально:

- контракт Android: PASS;
- XML manifest: PASS (`xmllint`);
- 15 статических итераций: 15 clean подряд;
- общий regression suite: `1..103`, PASS;
- build script: честный `SKIP`, без маскировки сборки.

Это не равно пяти чистым production-аудитам: пять-clean правило применяется
только после прохождения runtime, emulator и real-device gates.

## Найденные и исправленные проблемы

1. Android source отсутствовал, но документация уже описывала будущие gates.
   Добавлен native foundation с явным статусом `foundation`.
2. Параллельные агенты создали два валидатора и два storage-пути. Дубли
   удалены; оставлена одна строгая модель `ManagedProfileContractValidator`,
   один `ProfileImportService`, один Keystore secret store.
3. Первичная проверка не закрывала protocol-specific credentials и неизвестные
   root keys. Закрытый root и credentials rules теперь проверяются до импорта.
4. Импорт мог бы привести к plaintext credentials в document store. Импорт
   сохраняет document без credentials, а secrets пишет через AES-GCM/Keystore.
5. VpnService foundation мог быть ошибочно воспринят как working tunnel.
   Сервис не вызывает system tunnel API без engine и переходит в `ERROR`.
6. Android SDK отсутствует. Build gate теперь явно печатает `SKIP` и не
   выдаёт отсутствие сборки за успешный APK.

## Открытые блокеры

- pinned native engine и ABI/licence inventory;
- реальный packet loop для TUN и `VpnService.protect()` для bootstrap socket;
- DNS proxy, IPv4/IPv6 route policy, kill switch и leak evidence;
- reconnect при смене Wi-Fi/mobile, Doze, sleep/wake и reboot;
- rollback после handshake timeout, engine crash и revoke;
- Gradle Wrapper, SDK matrix, emulator tests и physical Android device;
- signed APK/AAB, reproducible metadata, Data safety and Play policy review.

До закрытия этих пунктов все 13 протокольных записей сохраняют
`support.android: planned`.
