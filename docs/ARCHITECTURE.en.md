# Mazzy VPN architecture

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).

[Русская версия](ARCHITECTURE.ru.md) · [Project README](../README.md)

Mazzy VPN is a Bash CLI/TUI and a small set of systemd units. It keeps protocol
profiles outside the source tree, uses one canonical desired-state file and
delegates tunnel lifetime to systemd. The interactive menu and automation
commands use the same functions, validation rules and state transitions.

## Component architecture

```mermaid
flowchart TB
    User["User or automation"]

    subgraph UX["CLI and terminal UI"]
        Entry["mazzy-vpn command dispatcher"]
        TUI["Interactive menu and dashboard"]
        Commands["connect, quick, test, doctor, import"]
    end

    subgraph Control["Validated control plane"]
        Validation["Profile parser and security validation"]
        ActionLock["Exclusive action lock"]
        State["Desired state and selected default<br/>/var/lib/vpnctl/active"]
        Runtime["Ephemeral counters, locks and test data<br/>/run/vpnctl"]
    end

    subgraph Supervisor["systemd supervision"]
        Service["vpnctl.service<br/>Restart=always"]
        Timer["vpnctl-health.timer<br/>about every 20 seconds"]
        Health["vpnctl-health.service"]
        BootRecovery["vpnctl-test-recovery.service"]
    end

    subgraph Data["Private local data"]
        Profiles["Protocol profiles<br/>/etc/vpnctl/profiles/*"]
        Locale["Installed language<br/>/etc/vpnctl/locale"]
    end

    subgraph Engines["Protocol adapters"]
        AWG["AmneziaWG<br/>awg-quick"]
        WG["WireGuard<br/>wg-quick"]
        OVPN["OpenVPN<br/>DNS hooks"]
        L2TP["L2TP/IPsec<br/>NetworkManager"]
    end

    Tunnel["One managed VPN interface"]
    Internet["Internet through the selected VPN"]
    External["Optional external fallback<br/>legacy or AdGuard"]

    User --> Entry
    Entry --> TUI
    Entry --> Commands
    TUI --> Commands
    Commands --> Validation
    Validation --> Profiles
    Commands --> ActionLock
    Commands --> State
    Commands --> Service
    Commands -. transactional fallback .-> External
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

The control plane never embeds a provider key in source code. The public
repository contains the manager, tests and documentation only. Operational
profiles stay root-readable with mode `600`.

## Normal connection flow

```mermaid
sequenceDiagram
    actor User
    participant CLI as mazzy-vpn
    participant Validator as Profile validator
    participant Lock as Action lock
    participant State as Desired state
    participant SD as systemd
    participant Runner as Protocol runner
    participant Net as VPN network

    User->>CLI: connect protocol profile
    CLI->>Validator: resolve, parse and validate
    alt invalid or unsafe profile
        Validator-->>CLI: reject with diagnostic
        CLI-->>User: no network change
    else valid profile
        Validator-->>CLI: accepted
        CLI->>Lock: acquire exclusive operation
        CLI->>SD: stop previous managed tunnel
        CLI->>CLI: stop a conflicting external VPN
        CLI->>State: save profile and DESIRED=up
        CLI->>SD: start vpnctl.service
        SD->>Runner: _service-run
        Runner->>Validator: validate again as root
        Runner->>Net: create protocol interface and routes
        SD-->>CLI: service started
        CLI->>Lock: release before returning to TUI
        CLI-->>User: connection started
    end
```

Validation happens before privilege-sensitive execution and again inside the
root service. Unsafe OpenVPN includes or executable hooks, unsafe WireGuard
hooks, incomplete profiles and open file permissions are rejected.

## Automatic monitoring and self-healing

```mermaid
stateDiagram-v2
    [*] --> TimerTick
    TimerTick --> Idle: DESIRED is not up
    TimerTick --> ValidateDefault: DESIRED is up
    ValidateDefault --> ReportBrokenState: default profile is missing
    ValidateDefault --> ServiceCheck: profile is valid
    ServiceCheck --> StartNow: service is inactive
    ServiceCheck --> InterfaceCheck: service is active
    InterfaceCheck --> FailureCounter: VPN interface is missing
    InterfaceCheck --> TrafficCheck: interface exists
    TrafficCheck --> Healthy: either HTTPS probe succeeds
    TrafficCheck --> FailureCounter: both HTTPS probes fail
    FailureCounter --> WaitForNextTick: first consecutive failure
    FailureCounter --> Reconnect: second consecutive failure
    StartNow --> Systemd
    Reconnect --> Systemd
    Systemd --> InterfaceCheck: start or restart accepted
    Healthy --> ResetCounter
    ResetCounter --> [*]
    Idle --> [*]
    WaitForNextTick --> [*]
    ReportBrokenState --> [*]
```

There are two independent recovery layers:

1. `vpnctl.service` uses `Restart=always` and a five-second delay. An unexpected
   process exit is recovered without waiting for the timer.
2. The health timer checks desired state, service state, the local VPN
   interface and two HTTPS paths. A desired but inactive service starts
   immediately. A locally active but unusable tunnel is restarted after two
   consecutive failed checks.

Manual `disconnect` writes `DESIRED=down` before stopping the service, so the
health monitor does not undo an intentional disconnect. An idle TUI does not
retain the action lock.

## Transactional live test and rollback

```mermaid
sequenceDiagram
    actor User
    participant CLI as mazzy-vpn test
    participant Snapshot as Transaction snapshot
    participant Guard as Independent timeout guard
    participant SD as vpnctl.service
    participant Probe as Interface and HTTPS checks
    participant Previous as Previous VPN or fallback

    User->>CLI: test profile with timeout
    CLI->>Snapshot: save state and previous connection
    CLI->>Guard: arm timeout and boot recovery
    CLI->>SD: start candidate in MODE=test
    SD->>Probe: create interface
    Probe-->>CLI: connection result
    alt candidate works and --keep
        CLI->>Snapshot: finalize candidate as MODE=normal
        CLI->>Guard: cancel guard
        CLI-->>User: candidate kept
    else candidate works without --keep
        CLI->>SD: stop candidate
        CLI->>Previous: restore previous connection
        CLI->>Guard: cancel guard
        CLI-->>User: test passed and rolled back
    else failure, signal, timeout or reboot
        Guard->>SD: stop incomplete candidate
        Guard->>Previous: restore saved connection
        Guard->>Snapshot: keep recovery data until success
        CLI-->>User: classified failure
    end
```

The transaction distinguishes a provider rejection, authentication failure,
timeout and local runtime error. Recovery state is removed only after the
previous connection has been restored successfully.

## State and security boundaries

| Path or unit | Purpose | Lifetime / access |
|---|---|---|
| `/etc/vpnctl/profiles/{amneziawg,wireguard,openvpn,l2tp}` | Private profiles | Persistent, directories `700`, files `600` |
| `/etc/vpnctl/locale` | Installed interface language | Persistent, non-secret |
| `/var/lib/vpnctl/active` | Selected protocol, default profile, `DESIRED`, test metadata | Persistent, root-owned |
| `/var/lib/vpnctl/test.*` | Transaction and rollback snapshot | Exists only while recovery may be needed |
| `/run/vpnctl` | Locks, health counter and sanitized runtime log | Cleared at boot |
| `vpnctl.service` | Owns the active managed tunnel | Long-running, systemd supervised |
| `vpnctl-health.timer` | Schedules independent health checks | Enabled for unattended recovery |
| `vpnctl-test-recovery.service` | Repairs interrupted tests after boot | Boot-time oneshot |

Security invariants:

- one managed tunnel and one serialized state-changing operation;
- profile type is detected from content, then validated before import;
- no private key, credential, personal path or operational profile belongs in
  Git;
- extended logs conceal private and AmneziaWG obfuscation parameters;
- test failure never silently replaces the last known working connection;
- external VPN fallback is optional and is not required for normal recovery.

## Verification map

| Concern | Command or automated check |
|---|---|
| Active route, DNS, interface, handshake and internet | `mazzy-vpn diagnose` |
| Dashboard and saved default | `mazzy-vpn dashboard` |
| All profile formats and permissions | `mazzy-vpn validate all` |
| Endpoint DNS and ping | `mazzy-vpn probe all --timeout 3` |
| Installation and systemd health | `mazzy-vpn doctor` |
| Safe automatic repairs | `sudo mazzy-vpn doctor --fix` |
| Offline full check | `mazzy-vpn self-test --offline` |
| Transactional live checks | `sudo mazzy-vpn test-all all` |
| Public-tree leak policy | `tests/audit-public.sh` and Gitleaks in CI |

