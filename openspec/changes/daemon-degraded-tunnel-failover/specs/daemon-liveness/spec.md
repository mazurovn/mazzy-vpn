# Spec Delta

## Purpose
Определяет, как фоновый демон `mazzy-vpn daemon` распознаёт деградировавший туннель (крипто-слой жив, data-path не работает) и когда обязан выполнить реконнект или failover на другую зону.

## ADDED Requirements

### Requirement: Rolling-window egress loss detection
The daemon SHALL keep a rolling window of the last 12 egress-probe outcomes for the active tunnel and SHALL treat the tunnel as lost when at least 5 outcomes in that window are failures, regardless of whether the failures are consecutive.

#### Scenario: Alternating probe results
- **WHEN** egress probes alternate between failure and success (e.g. F,S,F,S,F,S,F,S,F,S) while the WireGuard handshake stays fresh
- **THEN** after the 5th failure inside the window the daemon logs the loss and enters the reconnect path instead of staying in "protected"

#### Scenario: Healthy tunnel with an isolated blip
- **WHEN** at most 4 probes fail within any 12-probe window
- **THEN** the daemon keeps the tunnel and does not reconnect

### Requirement: Soft-failure counter decays instead of resetting
The soft-failure counter (probe failed while handshake is fresh) SHALL decrement by one on a successful probe and SHALL NOT be reset to zero by a single success.

#### Scenario: Two failures per success
- **WHEN** probe outcomes follow the pattern F,F,S,F,F,S,…
- **THEN** the soft-failure counter grows by one per cycle and reaches its limit within 6 cycles

### Requirement: Handshake churn signals a dead data path
The daemon SHALL count consecutive WireGuard re-handshakes that occur while the previous handshake is younger than 90 seconds. When this count reaches 4 the daemon SHALL treat the tunnel as lost even if the latest probe succeeded.

#### Scenario: Peer re-keys every 30 seconds
- **WHEN** `last_handshake_time_sec` advances on 4 consecutive health ticks and each previous handshake was < 90 s old
- **THEN** the daemon logs "handshake churn" and enters the reconnect path

#### Scenario: Normal rekey interval
- **WHEN** handshakes are ≥ 90 s old at the time of a tick
- **THEN** the churn counter resets to zero

### Requirement: Degradation detected on a successful tick still reconnects
When a health tick reports egress as confirmed but either the rolling-window loss limit or the handshake-churn limit is reached, the daemon SHALL follow the same reconnect/failover path as a confirmed egress loss.

#### Scenario: Good probe on a flapping tunnel
- **WHEN** the current probe succeeds but the window already holds 5 failures
- **THEN** the daemon logs "tunnel degraded despite a good probe" and tears the tunnel down for reconnect

### Requirement: Detection state resets on teardown
All degradation accumulators (window, soft-failure counter, churn counter) SHALL be cleared whenever the tunnel is torn down, so a new zone starts with a clean slate.

#### Scenario: Failover to another zone
- **WHEN** the daemon fails over from zone A to zone B
- **THEN** zone B is judged only by probes and handshakes observed after its own connect
