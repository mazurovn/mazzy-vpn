# Архитектура

Полная версия: [docs/ARCHITECTURE.ru.md](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/ARCHITECTURE.ru.md).

Эта страница описывает действующую архитектуру CLI/TUI и Desktop 0.3.
Целевая архитектура самостоятельного Desktop 1.0, общий core/API и правила
синхронизации интерфейсов описаны на странице
[[Desktop Full Application Plan]].

```mermaid
flowchart TB
    User["Пользователь"]
    TUI["CLI / TUI"]
    Desktop["Tauri control center / tray"]
    API["Protected local API v1"]
    Registry["13-protocol registry + safe detector"]
    Cache["Sanitized status + profiles cache"]
    Verify["Endpoint probe + actual egress verification"]
    Validate["Profile validation"]
    State["Desired state"]
    Systemd["systemd service + health timer"]
    Engines["AWG / WG / OpenVPN / L2TP"]
    Profiles["root-only profiles"]
    Tunnel["Managed VPN tunnel"]

    User --> TUI
    User --> Desktop
    Desktop -->|read| Cache
    Desktop -->|typed query/lifecycle| API
    TUI --> Registry
    API --> Registry
    API --> TUI
    Desktop -. remaining fixed pkexec actions .-> TUI
    TUI --> Verify
    TUI --> Validate
    Validate --> Profiles
    TUI --> State
    TUI --> Systemd
    Systemd --> State
    Systemd --> Cache
    Systemd --> Engines
    Engines --> Profiles
    Engines --> Tunnel
```

## Подключение

```mermaid
sequenceDiagram
    actor User
    participant CLI
    participant Validator
    participant Lock
    participant State
    participant Systemd
    participant Engine

    User->>CLI: connect protocol profile
    CLI->>Validator: parse + security check
    Validator-->>CLI: accepted
    CLI->>Lock: serialize mutation
    CLI->>State: profile + DESIRED=up
    CLI->>Systemd: start service
    Systemd->>Validator: validate again as root
    Systemd->>Engine: create tunnel
    CLI-->>User: connection started
```

## Desktop boundary

```mermaid
sequenceDiagram
    participant UI as Unprivileged Desktop
    participant Cache as Readable sanitized JSON
    participant API as Protected local API
    participant PK as pkexec
    participant CLI as Root CLI
    participant Secret as Profiles mode 600

    CLI->>Cache: atomic safe snapshot
    UI->>Cache: read status + profiles
    UI->>API: lifecycle / probe / verify typed envelope
    API->>CLI: validated operation
    CLI-->>API: sanitized structured result
    UI->>PK: typed fixed operation
    PK->>CLI: known argv only
    CLI->>Secret: validate and use
    Secret--xUI: never read
```

---

<a id="english"></a>

# Architecture

Full document: [docs/ARCHITECTURE.en.md](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/ARCHITECTURE.en.md).

The CLI/TUI is the current validated control plane. systemd owns tunnel
lifetime and an independent health timer. Desktop reads sanitized caches and
uses the protected local API for lifecycle, whole-list probes and actual egress
verification. Remaining typed operations map to fixed CLI argument arrays
through `pkexec`. The bundle contains compatible installer/engine resources.
Operational profiles remain root-only and never cross the Desktop boundary.
The shared 13-entry registry and redacted detector are described in
[[Protocol Orchestration]]. Catalog presence never bypasses platform/backend,
profile-validation or rollback gates. LLM clients receive opaque IDs and
evidence only; credentials and generated shell commands are outside the API.
