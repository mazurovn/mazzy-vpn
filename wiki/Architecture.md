# Архитектура

Полная версия: [docs/ARCHITECTURE.ru.md](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/ARCHITECTURE.ru.md).

```mermaid
flowchart TB
    User["Пользователь"]
    TUI["CLI / TUI"]
    Desktop["Tauri Dashboard / tray"]
    Cache["Sanitized status cache"]
    Validate["Profile validation"]
    State["Desired state"]
    Systemd["systemd service + health timer"]
    Engines["AWG / WG / OpenVPN / L2TP"]
    Profiles["root-only profiles"]
    Tunnel["Managed VPN tunnel"]

    User --> TUI
    User --> Desktop
    Desktop -->|read| Cache
    Desktop -->|fixed pkexec action| TUI
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
    participant PK as pkexec
    participant CLI as Root CLI
    participant Secret as Profiles mode 600

    CLI->>Cache: atomic safe snapshot
    UI->>Cache: read status
    UI->>PK: fixed enum action
    PK->>CLI: known argv only
    CLI->>Secret: validate and use
    Secret--xUI: never read
```

---

<a id="english"></a>

# Architecture

Full document: [docs/ARCHITECTURE.en.md](https://github.com/mazurovn/mazzy-vpn/blob/main/docs/ARCHITECTURE.en.md).

The CLI/TUI is the validated control plane. systemd owns tunnel lifetime and an
independent health timer. Desktop reads a sanitized cache and maps enum actions
to fixed CLI argument arrays through `pkexec`. Operational profiles remain
root-only and never cross the Desktop boundary.
