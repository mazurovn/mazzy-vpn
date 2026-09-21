# Spec Delta

## Purpose
Определяет, как клиент настраивает DNS туннельного интерфейса, чтобы при поднятом туннеле ни один запрос не уходил на резолвер физического аплинка.

## ADDED Requirements

### Requirement: Tunnel link owns the catch-all routing domain
When the DNS backend is systemd-resolved, bringing DNS up for the tunnel interface SHALL set the interface's DNS servers, SHALL set its routing domain to `~.` and SHALL mark it as a default-route link, in that order.

#### Scenario: Up with resolvectl
- **WHEN** `Manager.Up(["1.1.1.1"])` runs with `resolvectl` available
- **THEN** exactly `resolvectl dns <iface> 1.1.1.1`, `resolvectl domain <iface> ~.`, `resolvectl default-route <iface> yes` are executed

#### Scenario: Uplink resolver no longer consulted
- **WHEN** the tunnel is up and a name is resolved
- **THEN** systemd-resolved sends the query to the tunnel link's servers only

### Requirement: Down restores the previous state
Tearing DNS down SHALL revert every per-link setting made by Up.

#### Scenario: Down with resolvectl
- **WHEN** `Manager.Down()` runs after a successful Up
- **THEN** `resolvectl revert <iface>` is executed and no other resolvectl command
