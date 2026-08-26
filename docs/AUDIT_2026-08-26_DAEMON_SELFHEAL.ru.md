# Аудит 2026-08-26 — демон: зависания, самовосстановление, дашборд, лог, диагностика

Область: `mazzy-vpn-cli/cmd/mazzy-vpn/{daemon,background,recover,dashboard,diagnose_cmd,connect,desired}.go`,
`core/{livecheck,runstatus,measure,connect,netexec,engine/wireguard}`.

Жалобы пользователя: демон висит и не переподключается при проблемах провайдера; не ищет
другой хост; дашборд отстаёт; лог ущербный; диагностика не работает.
Диагноз подтверждён: все четыре жалобы имеют конкретные причины в коде.

---

## P0 — причины «висит и не лечится»

### P0-1. Весь цикл демона синхронный: `time.Sleep` и connect внутри обработчика тика
`daemon.go:195-302`. В ветке `case <-ticker.C` последовательно выполняются:
`time.Sleep(backoff(fails))` (до **30 с**, `daemon.go:236,283`), `connectZone`
(= `connect.Up` + `WaitProtected` до **20 с**, `daemon.go:363`), `pickBestLive`
(= `RankBest`: DNS + UDP + ICMP по всему каталогу). Одна итерация reconnect-цикла
занимает 50–90+ секунд, в течение которых:

- **heartbeat не пишется** → `Snapshot.Fresh(30s)` (`background.go:98`) считает демона
  мёртвым. Следствия каскадом — см. P0-2, P0-3;
- **SIGTERM не обрабатывается** — сигнал ждёт в канале до конца итерации;
- intent из TUI (пауза/смена зоны) читается с лагом в десятки секунд.

Это архитектурная ошибка: нужен конечный автомат на таймерах (reconnect-задержка =
`time.After`/следующий тик, а не Sleep), либо connect/rank в отдельной горутине,
а heartbeat — из независимой горутины с собственным тикером.

### P0-2. Мёртвая блокировка управления: stale heartbeat + вечный mutation.lock
Когда heartbeat протух (P0-1), а демон жив:

- `mazzy-vpn daemon <зона>` → `forwardToActiveDaemon` (`daemon.go:69,309`) видит «демон
  не запущен» → идёт в `lock.Acquire` → лок держит живой демон → **«another mazzy-vpn
  operation is in progress», exit 1**. Пользователь не может ни переподключить, ни
  переключить зону. Это и есть «висит демон и не подключается».
- `mazzy-vpn stop` → `signalDaemonPID` → `daemonRunning` → false → **exit 3 «нечего
  останавливать»**, при этом `recordDownIntent` уже записан → демон через ≤10 с
  запаузится (VPN упадёт), но процесс останется жить. Итог: VPN выключен, демон-зомби
  держит лок.
- `waitDaemonExit(pid, 35s)` (`background.go:181`) может не дождаться: демон обработает
  SIGTERM только после завершения текущей итерации (до ~90 с) → «daemon did not stop».

Фикс: правило «жив = PID жив», а свежесть heartbeat — только индикатор здоровья, не
существования; для forward/stop проверять PID из локфайла/heartbeat без окна Fresh(30s).

### P0-3. Единственный probe-URL `api.ipify.org` — ложный «egress lost» и вечный reconnect
`livecheck.go:46` — один жёстко зашитый URL, без fallback. Если провайдер режет/тормозит
ipify (реалистично), демон считает исправный туннель мёртвым и бесконечно рвёт/поднимает
его (reconnect-шторм), а `diagnose` тем же URL «измеряет» plain internet
(`diagnose_cmd.go:78`) → диагностика врёт в ту же сторону. Кроме того:

- `Snapshot.HandshakeAge` объявлен (`livecheck.go:25`), но **нигде не заполняется** —
  главный дешёвый сигнал WireGuard-здоровья (latest handshake через UAPI) не
  используется вовсе. Падение ipify при живом handshake должно означать «probe-сеть
  деградировала», а не «туннель мёртв».
- Реальная ошибка probe **проглатывается**: `livecheck.go:72-75` превращает любой err в
  «no traffic through tunnel yet» — в лог и дашборд не попадает ни DNS-ошибка, ни
  timeout, ни TLS reset. Отсюда «лог ущербный».

Фикс: цепочка из 2–3 независимых probe (ipify, `cloudflare.com/cdn-cgi/trace`,
`checkip.amazonaws.com`) + handshake-age как первичный сигнал; Reason = реальный err.

### P0-4. Failover зациклен на той же зоне и слеп к «пингуется, но не роутит»
`pickBestLive` (`daemon.go:384`) ранжирует по ICMP. Сценарий: провайдер зарезал
WireGuard на сервере X — сервер пингуется. Демон: connect X → link-up, egress fail →
reconnect → `pickBestLive` снова выбирает X (лучший ICMP) → бесконечный цикл без
прогресса. Никакого чёрного списка недавно провалившихся зон нет; `zonescore` существует,
но здесь не подключён; `BestAlive` не исключает текущую сбойную зону. Плюс при
`nz == zone` (`daemon.go:245,287`) `fails` не сбрасывается — backoff залипает на 30 с.

Фикс: failover должен исключать зону, провалившую egress N раз подряд (cooldown-список
с TTL), и учитывать `zonescore`.

### P0-5. `ctx = context.Background()` на весь CLI — нет отмены, «залипший» kill-switch
`main.go:31` — нет `signal.NotifyContext`. Ctrl+C во время `connect.Up`/`WaitProtected`/
`RankBest` убивает процесс дефолт-обработчиком **без teardown**: остаются интерфейс,
nft-таблицы и, хуже всего, армированный fail-closed kill-switch → у пользователя «нет
интернета вообще», пока он не догадается про `recover`. Внешне — «всё зависло и
заблокировалось».

---

## P1 — некорректное самовосстановление

### P1-1. Kill-switch снимается без подтверждения egress
`daemon.go:293-298`: комментарий «Egress re-confirmed inside connectZone» — неверен.
`connectZone` возвращает **non-nil conn при ok=false** (link-up, egress НЕ подтверждён,
`daemon.go:373-379`). Условие `conn != nil` снимает fail-closed guard при
неподтверждённом туннеле → окно утечки plaintext ровно в тот момент, ради которого
kill-switch армировался. Использовать второй возвращаемый флаг `ok`.

### P1-2. `rw.Reconnected()` инкрементится при неподтверждённом egress
`daemon.go:299-301` — та же причина: счётчик реконнектов в дашборде растёт, даже если
туннель не заработал. Метрика врёт.

### P1-3. Пауза-«зомби» после неудачного stop
`recordDownIntent` + intent TTL 2 мин (`desired.go:24`): демон запаузился, intent протух,
`paused` остаётся true навсегда (сбрасывается только явным «up»). Если stop вернул 3
(P0-2), пользователь думает «демона нет», демон думает «меня запаузили». Ни VPN, ни
ошибки. Нужно: paused-состояние отражать в heartbeat (`StatePaused`), меню — показывать.

### P1-4. `--session`-демон переживает падение меню
Session-демон должен умирать при выходе из меню, но это делает только явный quit-путь
меню. Крах/kill терминала оставляет session-демона жить как background (никто не
отличит: heartbeat помечает только `Background: false`). Наблюдаемый процесс
`daemon --best --session` (pid 30219) — живой пример.

---

## P1 — дашборд «отстаёт»

### P1-5. Пробелы в heartbeat во время реконнекта = дашборд исчезает
Данные пишутся только на тиках здорового цикла; в reconnect-цикле (>30 с без flush,
P0-1) `drawLiveDashboard`/`tuiHeader` через `daemonRunning` получают false и рисуют
«нет демона» вместо «RECONNECTING». Пользователь видит мигающий/пропадающий дашборд.
Фикс: отдельная heartbeat-горутина (flush каждые 2–5 с независимо от фазы) — тогда
Fresh(30s) честен, и состояние `reconnecting` видно непрерывно.

### P1-6. «Latency» в графике — это длительность HTTPS-probe, а не пинг
`daemon.go:253-258`: сэмпл = время полного HTTP-запроса к ipify (TCP+TLS+HTTP,
~240 мс на живом соединении из status.json). Метрика шумная и завышенная; во время
реконнекта сэмплов нет вообще (дыры в графике). Разделить: ICMP-пинг шлюза/сервера для
графика, HTTP-probe — только как булев признак egress.

### P1-7. `Protocol` в heartbeat всегда пуст
`daemon.go:129` — `NewWriter(zone, "", "", background)`; после connect протокол известен
(`proto.Title()`), но не записывается. В `status.json`: `"protocol": ""`. Дашборд не
может показать AmneziaWG/WireGuard.

---

## P1 — лог «ущербный»

### P1-8. Формат, полнота, ротация
`background.go:28-33,69`, `daemon.go:328-337`:
- метка времени только `15:04:05` — на многодневном демоне день неизвестен;
- нет уровней, нет структуры; в репо есть **`core/mlog` (175 строк) — демоном не
  используется вообще**;
- реальные причины сбоев не логируются (P0-3: err проглочен; `connect.Up`-ошибки — одна
  строка без контекста);
- `O_APPEND` навсегда: ни ротации, ни ограничения размера; `tailLog` читает весь файл в
  память (`background.go:190`);
- foreground-демон (без `--background`) пишет в терминал — `daemon.log` пуст, «l» в
  меню показывает старьё. Фактический лог за сессию — 4 строки (подтверждено на живой
  системе): connecting, protected и два «egress check failed (1)» без причины.

Фикс: перевести daemonState.logf на mlog (RFC3339, уровни, причина ошибки), ротация по
размеру, писать в файл и в stdout всегда.

---

## P1 — диагностика

### P1-9. `diagnose` не видит демона
`gatherSignal` (`diagnose_cmd.go:25-73`) не читает ни heartbeat, ни daemon.log, ни
desired.json. Диагност не может сказать главное: «демон 12 минут в reconnect-цикле по
зоне X, последние ошибки такие-то, intent = down». Добавить в Signal: состояние
heartbeat (state/age/fails/reconnects/последние ErrEvent), paused-факт, владельца лока.

### P1-10. Слепые зоны проверок
- «интернет есть» = один ipify (P0-3);
- `ConflictVPN` ловит только `tun0|tun1` (`diagnose_cmd.go:35`) — wg0, tailscale0,
  proton0 и пр. невидимы;
- `ServerAlive` сравнивается с `cur.Profile` из state-файла, который может быть протухшим
  (state пишется, но при крахе не чистится);
- handshake-age не проверяется нигде (P0-3);
- `trace`/`dns-check` не связаны с diagnose — нет единого «почему не работает» отчёта.

---

## P2

- `daemon.go:189` — сравнение stealth-скора `score < lastStealth-15` шумное: одна
  флап-проба ip-api даёт ложную тревогу; усреднить по 2–3 замерам.
- `stealthTicker` (5 мин) выполняет 3 HTTP-пробы до 6 с в том же цикле — ещё +6 с лага
  к обработке сигналов/intent (лечится общим фиксом P0-1).
- `runstatus.flush` пишет весь JSON с indent на каждый тик/ошибку — на tmpfs не больно,
  но при maxSamples=120 это ~3 КБ × каждые 10 с; можно не индентить.
- `backoff()` без джиттера: все клиенты одного сервера ломятся синхронно.
- `cmdStatus --json` собирает JSON `fmt.Printf`-ом (`connect.go:323`) — сломается на
  кавычках в reason; использовать encoding/json.
- `menu.go:111`/`tui.go:252` — quit-путь останавливает session-демона только при
  свежем heartbeat (та же Fresh(30s)-ловушка).

---

## Приоритетный план фикса

1. **P0-1/P1-5**: heartbeat в отдельной горутине; убрать `time.Sleep` из цикла —
   state machine на таймерах; обработка SIGTERM/intent мгновенная.
2. **P0-2**: существование демона = PID жив; Fresh — только health-индикатор.
3. **P0-3**: multi-probe egress + handshake-age по UAPI; Reason = реальная ошибка.
4. **P0-4**: cooldown-список сбойных зон в failover + zonescore.
5. **P0-5**: `signal.NotifyContext` в main + гарантированный teardown/disarm.
6. **P1-1/P1-2**: использовать флаг `ok` из connectZone.
7. **P1-8**: mlog + ротация. 8. **P1-9/10**: diagnose читает heartbeat/лог/handshake.

## Выполнено в этой итерации (2026-08-26, ветка feat/go-rewrite)

Реализовано и покрыто unit-тестами; сборка (включая vendor) и все тесты зелёные.
Живой демон НЕ перезапускался — изменения вступят в силу после пересборки/деплоя
бинарника и планового рестарта (см. процедуру ниже).

| # | Проблема | Фикс |
|---|----------|------|
| P0-1/P1-5 | Sleep в цикле, heartbeat голодает | Отдельная heartbeat-горутина (`rw.Touch()` каждые 5 с, потокобезопасный Writer + флаг closed); `time.Sleep(backoff)` заменён на гейт `nextAttempt` — цикл больше никогда не спит |
| P0-2 | stale heartbeat = «демона нет» → тупик лока | `daemonRunning`: существование = живой PID (+проверка `/proc/<pid>/cmdline` от переиспользования PID); свежесть — только health-сигнал; дашборд показывает «⚠ status N old» |
| P0-3 | один ipify; причины сбоёв скрыты | livecheck: цепочка независимых probe (ipify → checkip.amazonaws.com → icanhazip.com); Reason теперь несёт реальную ошибку |
| P0-3b | handshake-возраст не использовался | `Engine.IpcGet` + `Conn.HandshakeAge()`; демон различает «туннель мёртв» и «probe-endpoints недоступны»: при свежем handshake терпит до 6 тиков (softFailLimit), не рвёт рабочий туннель |
| P0-4 | failover выбирал ту же зону | cooldown-карта зон, проваливших egress (TTL 10 мин); `pickBestLive` их исключает (если остаются кандидаты) |
| P0-5 | Background-ctx, Ctrl+C без teardown | `signal.NotifyContext` в main; stop-пути демона и connect используют `context.WithoutCancel` для teardown — kill-switch и nft-гарды не «залипают» |
| P1-1/2 | kill-switch снимался без egress; Reconnected врал | оба действия только при `ok=true` из connectZone |
| P1-3 | пауза неотличима от смерти | новое состояние heartbeat `paused` + значок ⏸ PAUSED в дашборде |
| P1-7 | protocol в heartbeat пуст | `Writer.SetProtocol` из connectZone |
| P1-8 | лог без даты, без ротации | штамп `2006-01-02 15:04:05`; причины ошибок в каждой строке; ротация daemon.log → .1 при 5 МиБ |
| P1-9 | diagnose слеп к демону | Signal + Analyze знают: paused / reconnecting / stale-heartbeat демона, последнюю ошибку |
| P1-10 | ConflictVPN только tun0/tun1 | любые чужие VPN-интерфейсы (wg*, tailscale*, proton*, nordlynx*, ipsec*, ppp*, outline*) |

Дополнительно по итогам независимого ревью (GitHub Copilot CLI, 4 находки):

| # | Находка | Решение |
|---|---------|---------|
| C1 | kill-switch не армируется при самом первом коннекте | Осознанно: глушить весь интернет до первого рабочего туннеля нельзя; guard защищает только reconnect-гэп установленной сессии (задокументировано в коде) |
| C2 | `daemon --best` при живом демоне пре-резолвил зону без учёта cooldown владельца | Форвардится литерал `--best`; демон сам ре-ранжирует со своей cooldown-картой |
| C3 | softFails/reconnecting не сбрасывались при intent; эскалация soft→hard теряла тик | Сброс при resume/zone-switch; после softFailLimit — мгновенный переход к реконнекту |
| C4 | последовательные probe растягивали тик до 15 с | Общий бюджет на всю цепочку = 2× per-probe timeout (10 с) |

TUI-доработки: значок ⏸ PAUSED с подсказкой resume, предупреждение «⚠ status Ns old»
в шапке, протокол в заголовке, последняя ошибка одной строкой под графиком.
Версия CLI → 2.4.1, запись в CHANGELOG.

Легаси-остатки на хосте (вне кода, чистить руками при случае):
`/etc/modules-load.d/amneziawg.conf` (мёртвый модуль — ошибка в journalctl при каждой
загрузке), `/usr/bin/awg{,-quick}` (не используются), masked-юниты `vpnctl*`,
`mazzy-vpn-disabled.bak`. Активен charon (strongSwan) — следит за интерфейсами, конфликт
не создаёт, но держать в голове.

## Осталось на следующий этап (план)

1. `up_cmd`/`auto`/`test`: единый прогресс-индикатор RankBest (есть в TUI, нет в CLI).
2. Метрика графика: ICMP до шлюза/сервера вместо длительности HTTPS-probe (P1-6).
3. `doctor` → «спасатель»: сейчас doctor только читает; добавить режим
   `doctor --heal` (safe-последовательность: проверить heartbeat → resume paused →
   recover при мусорных таблицах → рестарт демона на best-зоне).
4. `cmdStatus --json` через encoding/json; jitter для intent-гонок; backoff-джиттер.
5. systemd-юнит: `Restart=always` + `WatchdogSec` c `sd_notify` от heartbeat-горутины —
   страховка 99.9 % от крэша самого демона.
6. Стресс-тест reconnect-цикла в netns (без риска для живого соединения).

## Как безопасно тестировать (важно: live-система)

Активный VPN этой машины управляется этим же демоном — интеграционные проверки рвут
соединение у самого тестировщика. Тестировать только через bash-обёртку с
автопереподключением по таймауту, например:

```bash
# разовый цикл: остановить, испытать сценарий, гарантированно поднять обратно
sudo timeout 120 ./test_scenario.sh; sudo mazzy-vpn daemon NetherlandsAmsterdamH4 --background
# или сторожевой цикл на время испытаний:
while true; do
  mazzy-vpn status --json | grep -q protected || sudo mazzy-vpn daemon --best --background
  sleep 30
done
```

Плюс unit-уровень: цикл демона уже тестируем (fake runner/livecheck) — правки P0-1..P0-4
покрываются без сети.
