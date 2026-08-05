# Аудит systemd hotfix 1.4.1 / Desktop 0.4.1

Дата основной проверки: 2026-08-04. Повторный upgrade-аудит: 2026-08-05.

## Обнаруженный дефект

После обновления установленного Desktop с 0.3.2 до 0.4.0 основной VPN-туннель,
health timer и кэш 24 профилей работали, но локальный API socket отсутствовал.
Журнал systemd содержал ordering cycle:

```text
mazzy-vpn-api.socket -> mazzy-vpn-api-recovery.service -> basic.target
  -> sockets.target -> mazzy-vpn-api.socket
```

Socket unit запускается ранним `sockets.target`. Recovery oneshot был явно
заказан перед socket, но стандартные зависимости service неявно добавляли
`After=basic.target`. Systemd разрывал цикл удалением socket start job. Поэтому
Desktop не мог подключиться к `/run/mazzy-vpn/api-v1.sock` после загрузки,
хотя CLI и уже поднятый туннель продолжали работать.

## Upgrade-конфликт

Установка пакета поверх 1.3.2 сохраняет старые unit-файлы в
`/etc/systemd/system`. Они имеют приоритет над package units в `/usr/lib`.
Исполняемые пути уже корректировались drop-ins, но новые recovery dependencies,
start limits и permanent-authentication exit policy могли оставаться
перекрытыми старой базовой конфигурацией. Повторный аудит 2026-08-05 также
обнаружил, что старый `vpnctl-health.timer` сохранял 20-секундный интервал:
для накопительного `OnUnitActiveSec` ещё не было package drop-in с явным
сбросом.

## Исправление

- Recovery oneshot использует `DefaultDependencies=no`, явно требует и ждёт
  `local-fs.target`, конфликтует с `shutdown.target` и выполняется до shutdown,
  API socket и mutating consumers.
- API socket явно `Requires` и запускается `After` успешного recovery.
- Package drop-ins переносят критический recovery ordering на legacy units,
  ограничивают restart policy `10min/5`, сохраняют
  `RestartPreventExitStatus=77` и timeout test recovery `60s`.
- Отдельный `vpnctl-health.timer.d/10-package-interval.conf` очищает старые
  `OnBootSec`/`OnUnitActiveSec`, задаёт `15s`/`60s` и bounded jitter `5s`.
- Регрессионные проверки валидируют source units, staged installation и
  package drop-ins, включая условие отсутствия default `basic.target` ordering
  у раннего recovery.

## Проверка

- `systemd-analyze verify`: успешно;
- `bash -n` и ShellCheck: успешно;
- `tests/run.sh`: 103/103;
- AppImage/DEB/RPM payload, dependencies, lifecycle и Xvfb GUI launch: успешно;
- ранний socket hotfix DEB был установлен поверх 0.4.0; финальный кандидат с
  timer migration затем переустановлен как `0.4.1`, а `dpkg -V` не обнаружил
  изменённых package-owned файлов;
- effective recovery graph не содержит `basic.target`, socket прямо требует
  recovery и слушает `/run/mazzy-vpn/api-v1.sock` с `root:mazzy-vpn 0660`;
- `status.get`: `connected`, AmneziaWG, `vpnaw0`, `desired=up`;
- `profiles.list`: 24 профиля, из них 9 AmneziaWG и 15 OpenVPN;
- failed systemd units: 0.

После переустановки основной fragment намеренно остаётся старым ручным
`/etc/systemd/system/vpnctl-health.timer`, но systemd применяет package drop-in
из `/usr/lib`. Effective-проверка показала `OnUnitActiveUSec=1min`,
`RandomizedDelayUSec=5s`, `ActiveState=active`:

```bash
systemctl daemon-reload
systemctl restart vpnctl-health.timer
systemctl show vpnctl-health.timer -p FragmentPath -p DropInPaths
systemctl cat vpnctl-health.timer
```

## Граница релиза

Hotfix не добавляет новые VPN backends. Полностью реализованными Linux connect
backends остаются AmneziaWG, WireGuard, OpenVPN и L2TP/IPsec. Современные
протоколы остаются на документированном уровне import/validation/render
foundation до закрытия TUN, DNS/routing, rollback и leak integration gates.
