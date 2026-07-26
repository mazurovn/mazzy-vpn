# Архитектура Mazzy VPN

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

[English version](ARCHITECTURE.en.md) · [Главная страница](../README.md)

Mazzy VPN состоит из Bash CLI/TUI и небольшого набора systemd units. Профили
протоколов хранятся вне исходного кода, выбранное состояние записывается в один
канонический файл, а временем жизни туннеля управляет systemd. Интерактивное
меню и команды автоматизации используют одинаковые функции, правила проверки и
переходы состояния.

## Компонентная архитектура

```mermaid
flowchart TB
    User["Пользователь или автоматизация"]

    subgraph UX["CLI и терминальный интерфейс"]
        Entry["Диспетчер команды mazzy-vpn"]
        TUI["Интерактивное меню и dashboard"]
        Commands["connect, quick, test, doctor, import"]
    end

    subgraph Control["Проверяемый контур управления"]
        Validation["Парсер и проверка безопасности профилей"]
        ActionLock["Эксклюзивная блокировка операции"]
        State["Желаемое состояние и default-профиль<br/>/var/lib/vpnctl/active"]
        Runtime["Временные счётчики, locks и данные теста<br/>/run/vpnctl"]
    end

    subgraph Supervisor["Контроль systemd"]
        Service["vpnctl.service<br/>Restart=always"]
        Timer["vpnctl-health.timer<br/>примерно каждые 20 секунд"]
        Health["vpnctl-health.service"]
        BootRecovery["vpnctl-test-recovery.service"]
    end

    subgraph Data["Приватные локальные данные"]
        Profiles["Профили протоколов<br/>/etc/vpnctl/profiles/*"]
        Locale["Язык интерфейса<br/>/etc/vpnctl/locale"]
    end

    subgraph Engines["Адаптеры протоколов"]
        AWG["AmneziaWG<br/>awg-quick"]
        WG["WireGuard<br/>wg-quick"]
        OVPN["OpenVPN<br/>DNS hooks"]
        L2TP["L2TP/IPsec<br/>NetworkManager"]
    end

    Tunnel["Один управляемый VPN-интерфейс"]
    Internet["Интернет через выбранный VPN"]
    External["Опциональный внешний fallback<br/>legacy или AdGuard"]

    User --> Entry
    Entry --> TUI
    Entry --> Commands
    TUI --> Commands
    Commands --> Validation
    Validation --> Profiles
    Commands --> ActionLock
    Commands --> State
    Commands --> Service
    Commands -. транзакционный fallback .-> External
    Timer --> Health
    Health --> State
    Health --> Runtime
    Health --> Service
    BootRecovery --> State
    BootRecovery --> Service
    Service --> Profiles
    Service --> AWG
    Service --> WG
    Service --> OVPN
    Service --> L2TP
    AWG --> Tunnel
    WG --> Tunnel
    OVPN --> Tunnel
    L2TP --> Tunnel
    Tunnel --> Internet
    Locale --> Entry
```

Контур управления не содержит ключей провайдера в исходном коде. В публичном
репозитории находятся только менеджер, тесты и документация. Рабочие профили
остаются доступными только root и имеют права `600`.

## Обычное подключение

```mermaid
sequenceDiagram
    actor User as Пользователь
    participant CLI as mazzy-vpn
    participant Validator as Валидатор профиля
    participant Lock as Action lock
    participant State as Желаемое состояние
    participant SD as systemd
    participant Runner as Адаптер протокола
    participant Net as VPN-сеть

    User->>CLI: connect протокол профиль
    CLI->>Validator: найти, разобрать и проверить
    alt профиль некорректен или небезопасен
        Validator-->>CLI: отказ с диагностикой
        CLI-->>User: сеть не изменяется
    else профиль корректен
        Validator-->>CLI: разрешено
        CLI->>Lock: занять эксклюзивную операцию
        CLI->>SD: остановить предыдущий managed-туннель
        CLI->>CLI: остановить конфликтующий внешний VPN
        CLI->>State: сохранить профиль и DESIRED=up
        CLI->>SD: запустить vpnctl.service
        SD->>Runner: _service-run
        Runner->>Validator: повторная root-проверка
        Runner->>Net: создать интерфейс и маршруты
        SD-->>CLI: сервис запущен
        CLI->>Lock: освободить перед возвратом в TUI
        CLI-->>User: подключение запущено
    end
```

Проверка выполняется до привилегированной операции и повторно внутри root
service. Запрещаются OpenVPN includes и executable hooks, небезопасные
WireGuard hooks, неполные профили и открытые права файлов.

## Автоматическое наблюдение и лечение

```mermaid
stateDiagram-v2
    [*] --> TimerTick
    TimerTick --> Idle: DESIRED не up
    TimerTick --> ValidateDefault: DESIRED равен up
    ValidateDefault --> ReportBrokenState: default-профиль отсутствует
    ValidateDefault --> ServiceCheck: профиль корректен
    ServiceCheck --> StartNow: сервис не активен
    ServiceCheck --> InterfaceCheck: сервис активен
    InterfaceCheck --> FailureCounter: VPN-интерфейс отсутствует
    InterfaceCheck --> TrafficCheck: интерфейс существует
    TrafficCheck --> Healthy: сработал любой HTTPS probe
    TrafficCheck --> FailureCounter: оба HTTPS probe не прошли
    FailureCounter --> WaitForNextTick: первая последовательная ошибка
    FailureCounter --> Reconnect: вторая последовательная ошибка
    StartNow --> Systemd
    Reconnect --> Systemd
    Systemd --> InterfaceCheck: start или restart принят
    Healthy --> ResetCounter
    ResetCounter --> [*]
    Idle --> [*]
    WaitForNextTick --> [*]
    ReportBrokenState --> [*]
```

Работают два независимых уровня восстановления:

1. `vpnctl.service` использует `Restart=always` и задержку пять секунд.
   Неожиданно завершившийся процесс восстанавливается без ожидания таймера.
2. Health timer проверяет желаемое состояние, systemd service, локальный
   VPN-интерфейс и два HTTPS-маршрута. Требуемый, но остановленный сервис
   запускается сразу. Формально активный, но не передающий трафик туннель
   перезапускается после двух последовательных неудачных проверок.

Ручной `disconnect` записывает `DESIRED=down` до остановки сервиса, поэтому
монитор не отменяет осознанное отключение. Ожидающее ввода TUI не удерживает
action lock.

## Транзакционный live-test и rollback

```mermaid
sequenceDiagram
    actor User as Пользователь
    participant CLI as mazzy-vpn test
    participant Snapshot as Снимок транзакции
    participant Guard as Независимый timeout guard
    participant SD as vpnctl.service
    participant Probe as Проверки интерфейса и HTTPS
    participant Previous as Предыдущий VPN или fallback

    User->>CLI: test профиль с timeout
    CLI->>Snapshot: сохранить состояние и прежнее соединение
    CLI->>Guard: включить timeout и boot recovery
    CLI->>SD: запустить кандидата в MODE=test
    SD->>Probe: создать интерфейс
    Probe-->>CLI: результат подключения
    alt кандидат работает и указан --keep
        CLI->>Snapshot: сделать кандидата MODE=normal
        CLI->>Guard: отменить защиту
        CLI-->>User: кандидат оставлен
    else кандидат работает без --keep
        CLI->>SD: остановить кандидата
        CLI->>Previous: вернуть прежнее соединение
        CLI->>Guard: отменить защиту
        CLI-->>User: тест успешен, rollback выполнен
    else ошибка, сигнал, timeout или перезагрузка
        Guard->>SD: остановить незавершённого кандидата
        Guard->>Previous: восстановить сохранённое соединение
        Guard->>Snapshot: хранить recovery до полного успеха
        CLI-->>User: классифицированная ошибка
    end
```

Транзакция различает отказ провайдера, ошибку авторизации, timeout и ошибку
локального runtime. Recovery-состояние удаляется только после успешного
возврата предыдущего соединения.

## Состояние и границы безопасности

| Путь или unit | Назначение | Время жизни / доступ |
|---|---|---|
| `/etc/vpnctl/profiles/{amneziawg,wireguard,openvpn,l2tp}` | Приватные профили | Постоянно, каталоги `700`, файлы `600` |
| `/etc/vpnctl/locale` | Выбранный язык | Постоянно, не является секретом |
| `/var/lib/vpnctl/active` | Протокол, default-профиль, `DESIRED`, метаданные теста | Постоянно, root |
| `/var/lib/vpnctl/test.*` | Снимок транзакции и rollback | Пока может понадобиться recovery |
| `/run/vpnctl` | Locks, health-счётчик и очищенный runtime log | Очищается при загрузке |
| `vpnctl.service` | Владеет активным managed-туннелем | Долгоживущий, под контролем systemd |
| `vpnctl-health.timer` | Планирует независимые health-проверки | Включён для автовосстановления |
| `vpnctl-test-recovery.service` | Исправляет прерванный тест после загрузки | Boot-time oneshot |

Инварианты безопасности:

- один managed-туннель и одна изменяющая состояние операция одновременно;
- тип профиля определяется по содержимому, затем проверяется до импорта;
- приватные ключи, credentials, личные пути и рабочие профили не попадают в
  Git;
- расширенный журнал скрывает приватные и AmneziaWG obfuscation-параметры;
- неудачный тест не заменяет молча последнее рабочее соединение;
- внешний VPN fallback опционален и не нужен обычному автовосстановлению.

## Карта проверок

| Что проверяется | Команда или автоматическая проверка |
|---|---|
| Маршрут, DNS, интерфейс, handshake и интернет | `mazzy-vpn diagnose` |
| Dashboard и сохранённый default | `mazzy-vpn dashboard` |
| Формат и права всех профилей | `mazzy-vpn validate all` |
| DNS и ping endpoint | `mazzy-vpn probe all --timeout 3` |
| Установка и systemd | `mazzy-vpn doctor` |
| Безопасное автоматическое исправление | `sudo mazzy-vpn doctor --fix` |
| Полная offline-проверка | `mazzy-vpn self-test --offline` |
| Транзакционные live-тесты | `sudo mazzy-vpn test-all all` |
| Утечки в публичном дереве | `tests/audit-public.sh` и Gitleaks в CI |

