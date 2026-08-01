# Глубокий аудит профилей и protocol orchestration — 2026-08-01

## Область проверки

Проверены установленная система 0.2/1.2, текущий Desktop/CLI source, profile
cache, package lifecycle, API boundary, protocol detection, архитектура
собственных серверов, AI/agent contract, Windows/Linux/Android roadmap,
документация и release claims.

## Исправленные блокеры

### 1. High: Dashboard видел 24 профиля, а Profiles показывал пустой список

В `/run/mazzy-vpn/profiles.json` действительно находились 24 записи: 9
AmneziaWG и 15 OpenVPN. Старый CLI 1.2 публиковал schema без `profile_id`, а
Desktop 0.3.0 требовал это поле и отбрасывал весь cache. UI не различал
недоступный cache и корректный пустой список.

Исправление: Rust принимает только это контролируемое legacy-отклонение,
вычисляет opaque `profile_id` тем же SHA-256 алгоритмом, что CLI, затем снова
применяет строгие invariants. UI хранит отдельный availability state и выводит
ошибку cache вместо ложного empty state. Regression test использует точный
старый формат.

### 2. High: старый ручной CLI перекрывал новый Desktop package

Пакет Desktop 0.2 был установлен, но shell запускал root-owned regular copy
`/usr/local/bin/mazzy-vpn` версии 1.2.0. Package engine под `/usr/bin` отсутствовал
в этой старой установке, local API socket не был активен. Простая установка
нового DEB оставила бы приоритет `/usr/local/bin` и продолжила запускать старый
engine.

Исправление: package post-install распознаёт только собственные root-owned,
не доступные группе/всем на запись CLI-файлы с ожидаемой license/product/version
signature. Он перемещает их в private migration directory и создаёт ссылки на
`/usr/bin/mazzy-vpn`. Uninstall восстанавливает содержимое и исходные права.
Сторонние, user-owned, unsafe или изменённые после установки файлы не
перезаписываются.

### 3. High: риск ложного заявления «поддерживается 13 протоколов»

VLESS, Hysteria 2, Mieru, NaiveProxy, TUIC, Shadowsocks 2022, Trojan, AnyTLS и
ShadowTLS не являются девятью готовыми drop-in VPN backends. Большинство —
proxy/transport protocols, которым нужен TUN adapter и platform lifecycle.
VLESS также требует внешнего security/transport слоя; Naive требует
совместимого Cronet build. Наличие `sing-box` в PATH не доказывает готовность
конкретного backend.

Исправление: versioned registry разделяет `vpn`, `proxy`, `transport` и статусы
`implemented`, `partial`, `planned` по detection/import/diagnostics/platform.
Только AmneziaWG, WireGuard, OpenVPN и L2TP/IPsec имеют Linux connection
`implemented`. CI запрещает повышать status без изменения проверяемой матрицы.

### 4. High: произвольный engine JSON нельзя запускать от root

Прямая передача пользовательского `sing-box`/Xray JSON привилегированному
процессу расширила бы command/config injection surface, позволила бы создавать
непредусмотренные listeners/routes и разрушила sanitization contract.

Архитектурное решение: каждый protocol adapter должен принимать закрытую typed
schema, хранить secrets только в root-only store, генерировать engine config
самостоятельно, ограничивать outbound/TUN/DNS policy и иметь transaction,
deadline, audit и rollback. Custom server не равен arbitrary config execution.

### 5. Medium: share URI мог содержать управляющий байт

Первый detector не отражал URI, но Bash command substitution способен
отбрасывать NUL. Исправление валидирует размер и исходные байты в base64
представлении до помещения decoded value в shell variable. Допускается ровно
один конечный `LF` или `CRLF`; embedded/repeated newline, NUL, другие C0 controls
и DEL отклоняются. Результат содержит только protocol ID, kind и readiness;
host, UUID, username, password, query и fragment отсутствуют.

### 6. High: fake transport test не обнаружил закрытие API response-half

После установки DEB 0.3.1 socket activation запускал worker, но клиентский
`socat` завершался при EOF request stdin. Worker получал `broken pipe`, а
Desktop не мог получить typed API response. Исправление использует
`STDIO,ignoreeof`: write-half завершается после запроса, read-half остаётся до
ответа и закрытия worker. Новый regression запускает настоящий Unix socket и
system `socat` с задержанным responder; fake dispatcher больше не является
единственным transport gate.

## Архитектура orchestration

Registry содержит 13 записей. Безопасный порядок выбора:

1. Hard gates: platform/backend ready, typed profile valid, secrets доступны
   только backend, transaction и rollback подготовлены.
2. Evidence: recent success, censorship fit, endpoint/tunnel reachability,
   latency/loss и workload fit.
3. Deterministic planner формирует объяснимый dry-run rank; bounded fallback
   остаётся отдельной authorized mutation.
4. Привилегированный executor повторно проверяет plan и выполняет только
   allowlisted typed operation с action ID.

LLM не участвует в hard gates. Агент получает opaque profile IDs, redacted
health evidence и catalog status. Он может запросить или объяснить plan, но его
текст не становится shell command, engine JSON или credential store input.

## Открытые production-блокеры

- Реализовать отдельные адаптеры: sing-box family, Mieru и Naive/Cronet;
- определить typed custom-server/import schema и secret lifecycle;
- реализовать Linux TUN/DNS/route lifecycle с leak, rollback и fault tests;
- добавить history, authorized planner execution/failover и human approval;
- завершить Windows Service/Wintun, signed installer и native integration tests;
- создать Android `VpnService` source, embedded reproducible libraries,
  Keystore/import и real-device lifecycle tests;
- заменить оставшиеся Desktop `pkexec` operations общим service API;
- подписывать artifacts, публиковать provenance/SBOM и тестировать clean
  install/upgrade/remove/rollback на поддерживаемых дистрибутивах.

## Вывод

Patch release может безопасно выпускать исправление профилей, package migration,
catalog/detection/API foundation и обновлённый backlog. Он не может честно
объявить девять новых протоколов рабочими подключениями, Windows полноценным
VPN-клиентом или Android готовым приложением до закрытия перечисленных gates.

## Проверки patch source и artifacts

- `./tests/run.sh`: 80/80;
- Rust unit tests: 24/24;
- Clippy `-D warnings`: проходит; два известных warning принадлежат
  byte-verified upstream `glib` 0.18 source;
- `npm audit --audit-level=high`: 0 vulnerabilities;
- cargo-deny 0.20.2 с обновлённой RustSec database: `advisories ok`;
- Desktop contract: v0.3.2, 90 DOM IDs и 135 localized labels;
- capability registry: 17 capabilities и 6 release gates;
- API contract: v1.0, 28 operations и 14 errors;
- protocol registry: 13 entries и 9 однозначных share URI schemes;
- AppImage/DEB/RPM package audit: payload byte identity, dependencies,
  lifecycle, corresponding source и Xvfb GUI launch подтверждены;
- все 12 screenshots пересняты из localhost-only fixture на 1680×951 с RFC
  5737 data и визуально проверены.

Post-audit update: read-only `planner.evaluate` и CLI route реализованы после
релиза 1.3.2. Они применяют пять backend-owned hard gates, policy v1 scoring,
stale-evidence rules и stable opaque-ID tie-break. Operation не меняет сеть;
issue #39 остаётся открытым для history, authorized execution/failover и
Desktop/mobile integration. Визуальные surfaces не изменялись, поэтому
повторная генерация screenshots для этого среза не требуется.
