# Runtime-инцидент Desktop и health recovery, 2026-08-05

## Область проверки

Проверены установленный Mazzy VPN CLI `1.4.1`, Desktop `0.4.1`, systemd units,
локальный API v1, runtime-кэши, маршруты, DNS, журналы текущей загрузки и
фактический IPv4 egress. Содержимое конфигураций, ключи и публичный IP в этот
отчёт не включены.

## Подтверждённая временная линия

1. В `13:10:28` и `13:10:49 MSK` health-monitor дважды не подтвердил HTTPS
   через активный VPN-интерфейс.
2. В `13:10:50` `systemctl restart vpnctl.service` успешно остановил и снова
   запустил сервис. Новый AmneziaWG-интерфейс был создан через userspace
   fallback, но защищённый egress не был подтверждён за 20 секунд.
3. В `13:11:11` monitor остановил непроверенный туннель и сохранил
   fail-closed transition marker. Последующие автоматические и ручные действия
   блокировались до отдельного восстановления, как требует модель rollback.
4. В `13:12` и `13:15` пользователь запустил два независимых процесса
   `mazzy-vpn-desktop`. В D-Bus одновременно были зарегистрированы две tray
   icons, а каждый процесс держал отдельный WebKit renderer примерно по
   180 MiB RSS.
5. После ручного восстановления в `13:20` сервис, API socket и health timer
   стали активны. Локальный API стабильно вернул 24 профиля: 9 AmneziaWG и
   15 OpenVPN. Два независимых геопровайдера подтвердили австрийский IPv4
   egress; IPv6 egress отсутствовал, потенциальная IPv6-утечка не обнаружена.
6. В период отказа сторонние пользовательские процессы также фиксировали
   TCP timeout к внешним сервисам. Это подтверждает общий data-plane сбой, а
   не падение локального API, но по журналу всё ещё нельзя доказать, находилась
   причина на VPN endpoint, промежуточном пути или внешнем probe.
7. Повторная проверка в `13:44` подтвердила маршрут, DNS и отсутствие IPv6 leak,
   но один из двух GeoIP providers временно не ответил. UI снова показал
   `ЕСТЬ РИСКИ`, что выявило второй informational finding в той же ошибочной
   ветке классификации.

## Причины

- Первичный сетевой триггер: два последовательных отказа data-plane проверки
  выбранного туннеля. Журнал не позволяет доказать, был ли недоступен VPN
  endpoint или внешний HTTPS probe; локальный API и systemd socket не падали.
- Усиливающий фактор: timer выполнял внешний probe примерно каждые 20 секунд.
  Это создаёт лишнюю нагрузку, повышает вероятность реакции на короткий сетевой
  сбой и даёт меньше минуты на восстановление до restart.
- Ошибка наблюдаемости: сообщение `systemd отклонил команду` было неверным.
  systemd выполнил restart; не прошла следующая проверка защищённого egress.
- Desktop не ограничивал число экземпляров. Повторный запуск создавал второй
  tray, WebView и независимое состояние дашборда вместо показа существующего
  окна.
- UI переводил любое `warning` проверки egress как `ЕСТЬ РИСКИ`. Для профилей
  без необязательного `country_code` engine возвращает informational finding
  `verify.geo.expected-country-unavailable`; временный отказ одного GeoIP
  provider добавляет `verify.geo.single-provider`. Даже когда маршрут, DNS и
  leak checks прошли, исправный австрийский egress выглядел как проблема.
- Запуск из desktop shell направлял stdout/stderr в `/dev/null`; поэтому у
  Desktop не было отдельного crash-журнала. Coredump не найден, а сами процессы
  не падали.
- Автопроверка обновлений получает `404 Not Found`, пока release-feed
  `desktop-updater/latest.json` не опубликован. Это не влияет на туннель и API,
  но создаёт отдельную ошибку updater при каждом старте и подтверждает, что
  release pipeline ещё не завершён.

## Исправления в patch source

- `tauri-plugin-single-instance` регистрируется первым Tauri plugin. Повторный
  запуск завершает новый процесс и показывает/фокусирует существующее окно.
- `vpnctl-health.timer` переведён с 20 на 60 секунд с небольшим jitter. Два
  последовательных отказа по-прежнему запускают bounded recovery, но короткий
  сетевой сбой не приводит к restart через 40 секунд.
- Package drop-in очищает накопительный `OnUnitActiveSec` и задаёт 60 секунд,
  поэтому исправление действует и при upgrade поверх старого ручного
  `/etc/systemd/system/vpnctl-health.timer`, который имеет приоритет над unit
  из пакета.
- Health recovery теперь отдельно сообщает отказ команды systemd и случай,
  когда restart выполнен, но защищённый egress не подтверждён.
- Комбинации только `expected-country-unavailable` и `single-provider`
  показываются как `ПРОВЕРЕНО ЧАСТИЧНО`, но лишь при активном туннеле,
  совпадающем IPv4 egress, полном VPN DNS и отсутствии IPv6 leak. Реальные
  route/location mismatch, DNS/IPv6 leak, GeoIP disagreement/unavailable и
  другие failures по-прежнему отображаются как риски.
- Добавлен постоянный Desktop log с ротацией `KeepOne` и пределом
  `1 000 000` байт. Он
  фиксирует lifecycle, single-instance, updater, тип/результат операции, число
  профилей и агрегаты probe/verify без имён профилей, endpoint, IP, ключей и
  содержимого конфигураций.
- Добавлены contract-тест порядка single-instance plugin, проверки минутного
  timer, legacy-unit upgrade и регрессия классификации post-restart egress
  failure.

## Что не являлось причиной

- Профили не потеряны: runtime cache и API содержат все 24 профиля.
- API boot-ordering hotfix работает: `mazzy-vpn-api.socket` слушает,
  `mazzy-vpn-api-recovery.service` успешно завершён, все просмотренные API
  request units завершились без ошибки.
- Текущий туннель не выдаёт российский egress: обе независимые проверки
  согласились по стране и городу выхода.
- Отдельного падения WebKit/Desktop не зафиксировано. Видимая проблема была
  сочетанием fail-closed recovery state и двух экземпляров UI.

## Повторная проверка

```bash
systemctl --failed --no-pager
systemctl status vpnctl.service mazzy-vpn-api.socket vpnctl-health.timer --no-pager
mazzy-vpn status --api-json
mazzy-vpn profiles --api-json
mazzy-vpn verify --timeout 10 --json
journalctl -b -u vpnctl.service -u vpnctl-health.service --no-pager
gdbus call --session --dest org.kde.StatusNotifierWatcher \
  --object-path /StatusNotifierWatcher \
  --method org.freedesktop.DBus.Properties.Get \
  org.kde.StatusNotifierWatcher RegisteredStatusNotifierItems
```

Нормальный результат: один `tray_icon_tray_app_mazzy_main`, один Desktop
процесс, API status `ok`, 24 профиля и отсутствие unresolved recovery marker.

Desktop log на Linux:

```bash
tail -n 100 "$HOME/.local/share/com.mazurovn.mazzy-vpn/logs/Mazzy VPN Desktop.log"
```

Финальный DEB с timer drop-in переустановлен на текущей машине как `0.4.1`.
`dpkg -V` не обнаружил изменённых package-owned файлов. Старый ручной fragment
с `OnUnitActiveSec=20s` остаётся виден в `systemctl cat`, но package drop-in
сначала очищает накопительное значение и задаёт 60 секунд; effective state
подтверждает `OnUnitActiveUSec=1min`, jitter `5s` и активный timer.

## Повторный аудит release gate

До публикации patch release PR-проверки нашли и устранили четыре независимых
проблемы pipeline:

- Windows UI audit передавал большой сгенерированный JavaScript через
  `node -e` и превышал системный предел длины команды (`WinError 206`). Script
  теперь передаётся через stdin, при этом тот же parser и contract остаются
  обязательными на Linux, macOS и Windows.
- CodeQL обнаружил один command-injection flow из `CARGO` и восемь
  path-injection flow из аргументов helper проверки updater signatures.
  Release builder вызывает фиксированный `cargo`; helper не принимает пути от
  caller, использует фиксированный workspace, bounded inventory скачанных
  regular files и локальные порядковые имена signatures.
- systemd regression локально зависел от уже установленного
  `/usr/bin/mazzy-vpn` и host unit path. На чистом runner это корректно
  приводило к отказу. Тест перенесён в отдельный root с package units,
  drop-ins, API template, executable и inert standard targets.
- Signed Desktop workflow проверяет все updater signatures до публикации и
  после этого создаёт и загружает
  `Mazzy.VPN.Desktop_0.4.1_SHA256SUMS`. Fixed update feed продвигается только
  после успешной Linux/macOS/Windows matrix и этой проверки.
