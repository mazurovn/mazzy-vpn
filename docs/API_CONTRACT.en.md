# Mazzy VPN local API v1 foundation

The language-neutral contract is published in:

- [`../api/v1/manifest.json`](../api/v1/manifest.json) — operations,
  authorization, audit, deadline and rollback requirements;
- [`../api/v1/schema.json`](../api/v1/schema.json) — frontend-safe request,
  response and event envelopes.

The published contract version is `1.0`. This is an incremental compatibility
boundary for issue
[#5](https://github.com/mazurovn/mazzy-vpn/issues/5). The current
`cli-json-adapter` is explicitly `partial`: existing safe JSON status/profile
outputs remain available, but clients do not yet submit every v1 request
envelope through one dispatcher. Contract metadata is implemented. The
socket-activated Linux transport is `partial`: it accepts `status.get`,
`profiles.list` and the three `lifecycle.*` mutations. Other operations and
non-Linux transports remain `planned`, so this still does not claim that the
complete cross-platform daemon exists.

## Compatibility

- `api_version` uses `major.minor`.
- A major change may remove or reinterpret fields and requires a new schema.
- A minor change may add optional operations, enum values or fields.
- Unknown major versions must return `unsupported-version`.
- Clients must use operation and error codes, not localized output text.

## Mutations

Every mutation requires:

- a caller-generated `action_id`;
- an explicit authorization class;
- a bounded `deadline_ms`;
- a sanitized audit event;
- declared rollback semantics and a final rollback outcome.

`deadline_ms` is a monotonic mutation budget, not a promise to abandon safety
work when the response clock expires. The Linux dispatcher starts the budget
after validating the mutation envelope, subtracts lock/preflight time and
passes the remaining milliseconds to the executor without rounding up. It
never starts the executor after the budget has expired. Executor and refresh
timeouts terminate their process groups, so a timed-out shell helper is not
left running concurrently with rollback. Required rollback and crash
reconciliation use separate bounded system-service timeouts; a response
may therefore arrive after `deadline_ms` while rollback is being completed.
Linux clients reserve a bounded 60-second completion grace for that outcome.
An incomplete rollback enters recovery-only mode instead of being reported as
successful.

Repeated delivery of the same action ID during the documented retention window
must never apply the mutation twice. The Linux lifecycle dispatcher enforces
this rule with a persistent, root-readable action journal. It keeps the newest
512 completed outcomes by default. A client must not reuse an evicted action ID
as a new operation. Remaining mutation domains must adopt the same rule and
publish their retention policy as they move behind the service.

The dispatcher stores the rollback snapshot before marking an action
`running`. After a service crash, the next mutation reconciles every orphaned
running record under the global mutation lock. It restores the snapshot and
stores a terminal `rolled-back` or `rollback-failed` outcome instead of leaving
the action permanently busy or executing it again.

## Frontend safety

Frontend status, responses and events use opaque IDs and message keys. Private
or preshared keys, passwords, credentials, secrets, endpoints, full
configurations and unrestricted filesystem paths are forbidden. Import paths
are exchanged for short-lived opaque `import_token` values before they enter
the shared API.

Raw backend output is not part of the stable contract. Doctor exposes finding
codes, severity, localized message keys and individually authorizable fix IDs.

## Current access

`mazzy-vpn api-info --json` returns the installed contract manifest without
root privileges. Desktop exposes the same metadata to its webview through a
read-only Tauri command. CI checks that the CLI result, manifest and schema stay
synchronized.

On Linux, `mazzy-vpn-api.socket` exposes `/run/mazzy-vpn/api-v1.sock` as
`root:mazzy-vpn` mode `0660`. Every connection carries one newline-terminated
request and receives one newline-terminated response. The service stops reading
at the configured byte limit before JSON parsing, including for a client that
never terminates an oversized line. It accepts exactly one top-level JSON
object and rejects duplicate envelope or payload keys, including keys encoded
with JSON Unicode escapes, before dispatch. Query cache refreshes are bounded
by the smaller of the optional query deadline and the server refresh cap; an
existing restricted cache may be returned when live refresh times out.
Interrupted refreshes clean their private temporary files. Mutations are serialized,
deadline-bounded and recorded by action ID under root-only state. The audit log
contains operation IDs and outcomes only, never request payloads or backend
output. A mutation is not started unless its initial audit event is durable.
If a terminal audit event cannot be stored after state changed, the completed
action remains idempotent but the API enters recovery-only mode for explicit
administrator inspection. Completed outcomes are bounded to 512 records by default. The audit
file rotates at 2 MiB and keeps one root-only archive; these limits can be
reduced for constrained systems but must not be disabled.

If crash reconciliation cannot read the pre-action snapshot or cannot restore
it, the daemon persists a root-only recovery marker and rejects every later API
mutation. After manually inspecting and repairing the current VPN state, an
administrator must explicitly acknowledge it with
`sudo mazzy-vpn _api-clear-recovery --acknowledge-current-state`. The marker is
never cleared by a timer or an unrelated successful request.

Desktop retries a lifecycle request exactly once, using the identical request
and action ID, only after a post-connect transport failure makes the outcome
indeterminate. A failed initial socket connection may use the typed
compatibility adapter because no request was sent; post-connect uncertainty
never falls through to `pkexec`.

`status.get` may add safe runtime detail for terminal/dashboard parity:
desired mode, interface, handshake age, current public IP, autostart, health
monitor, failure count and external-fallback state. These fields are optional
for minor-version compatibility. The VPN endpoint, profile filename/path and
configuration remain forbidden.
