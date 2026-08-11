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
outputs remain available, and CLI/TUI now submit v1 envelopes for `status.get`,
`profiles.list`, `protocols.list`, `planner.evaluate`, `tests.probe`,
`tests.verify-egress` and `lifecycle.*` through one dispatcher. Remaining domains
still use the compatible direct CLI control plane. Contract metadata is
implemented. The
socket-activated Linux transport is `partial`: it accepts `status.get`,
`profiles.list`, `protocols.list`, `planner.evaluate`, the bounded
`tests.probe`/`tests.verify-egress` queries and the three `lifecycle.*`
mutations. Other operations and
non-Linux transports remain `planned`, so this still does not claim that the
complete cross-platform daemon exists.

## Compatibility

- `api_version` uses `major.minor`.
- A major change may remove or reinterpret fields and requires a new schema.
- A minor change may add optional operations, enum values or fields.
- Unknown major versions must return `unsupported-version`.
- Clients must use operation and error codes, not localized output text.
- Existing `status --json` and `profiles --json` schema-v1 documents do not
  change shape when the socket appears. Callers that need the API envelope use
  `status --api-json` or `profiles --api-json` explicitly.

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
object and rejects duplicate object keys at every depth, including keys encoded
with JSON Unicode escapes, before dispatch. Query cache refreshes are bounded
by the smaller of the optional query deadline and the server refresh cap; an
existing restricted cache may be returned when live refresh times out.
Interrupted refreshes clean their private temporary files. Mutations are serialized,
deadline-bounded and recorded by action ID under root-only state. The audit log
contains operation IDs and outcomes only, never request payloads or backend
output. A mutation is not started unless its initial audit event is durable.
The snapshot, running action record and initial audit are synchronized together
with their parent directories before the lifecycle child starts. Completed
records and snapshot deletion are synchronized before terminal success.
If a terminal audit event cannot be stored after state changed, the completed
action remains idempotent but the API enters recovery-only mode for explicit
administrator inspection. Completed outcomes are bounded to 512 records by default. The audit
file rotates at 2 MiB and keeps one root-only archive; these limits can be
reduced for constrained systems but must not be disabled.

If crash reconciliation cannot read the pre-action snapshot or cannot restore
it, the daemon persists a root-only recovery marker and rejects every later API
mutation. A hardened root boot oneshot reconciles interrupted actions under the
shared mutation lock before test recovery, `vpnctl.service`, health remediation
and the API socket. Failure to prepare its protected directories or acquire the
lock also persists the marker. Test recovery requires this gate and starts only
after it succeeds. While the marker exists, managed-service start, test recovery
and health remediation fail closed. After manually inspecting and repairing the current VPN state, an
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

`protocols.list` returns the sanitized 13-entry protocol catalog and
orchestration policy. It separates detection/import/diagnostics from
per-platform connection readiness, so a catalog entry cannot be mistaken for
an implemented backend. The response contains public format, engine and
transport identifiers only; it contains no endpoint, credential, profile or
backend configuration. Its source of truth is
[`../protocols/v1/registry.json`](../protocols/v1/registry.json).

`planner.evaluate` is a read-only, deadline-bounded evaluation. The payload
contains a workload and 1–128 unique opaque profile IDs, each with a complete
bounded evidence object. The server, not the caller, computes these five hard
gates from current local state: backend ready, profile valid, backend-only
profile storage, protected rollback storage ready and Linux support implemented.
The storage gate proves only that a secure journal/snapshot location can be
used; candidate-specific rollback remains an execution concern. Only candidates
that pass every gate receive a score and rank.

The policy-v1 score is 30 points for recent outcome, 25 for censorship fit, 20
for reachability, 15 for latency/loss and 10 for workload fit. Observed health
evidence (`recent_outcome`, reachability and latency/loss) older than 900
seconds scores zero. The backend derives `censorship-fit` from the protocol
catalog and `workload-fit` from workload, protocol class and transports; the
caller cannot self-assign either factor. Equal scores use the opaque profile ID
as the stable tie-breaker. The response includes reason codes
and factor points, but no display name, endpoint, filename, path, configuration
or credential. It is always `dry_run: true`; it cannot connect or fail over.
Candidate validation receives the same absolute monotonic deadline as the
evaluator, including an OpenVPN parser subprocess. The CLI accepts one JSON
payload up to 64 KiB on stdin and bounds the expanded explanation response to
1 MiB:

```bash
jq -n --arg profile_id "$PROFILE_ID" '{
  workload: "llm-streaming",
  candidates: [{
    profile_id: $profile_id,
    evidence: {
      recent_outcome: "success", consecutive_failures: 0,
      reachability: "reachable", latency_ms: 80, loss_percent: 0,
      evidence_age_seconds: 30
    }
  }]
}' | mazzy-vpn planner evaluate --stdin --json
```

The caller can influence dry-run rank only through bounded health evidence and
cannot bypass a backend-owned gate. Trusted history collection, authorized
execution and automatic failover remain outside this operation.

`tests.probe` checks every profile in the requested protocol scope with a
per-endpoint timeout and bounded concurrency of 1–8 workers. It returns an
opaque profile ID, safe display name, protocol, current active/default flags,
transport, `reachability`, optional integer `latency_ms`, its ICMP/TCP source
and a message key. It never returns the endpoint. `reachable` means that ICMP
or the configured TCP service answered; it does not prove VPN credentials,
handshake, routes or DNS through a tunnel. For UDP, DNS success without an
ICMP response is `unknown`, not `unreachable`, because many valid servers block
ping and UDP has no safe generic connection handshake. A full proof remains a
transactional live test with rollback. The server applies the request deadline
to the entire worker group and serializes batch probes with a global lock so
concurrent socket clients cannot multiply network load.

`tests.verify-egress` is a read-only, globally serialized query with a bounded
deadline. Its payload contains only `timeout_seconds` and the explicit
`include_speed` choice. The response reports:

- active tunnel protocol/display name/interface;
- interface-bound and default IPv4 plus their equality;
- interface-bound/default IPv6 and a potential-leak flag;
- expected/observed country, provider agreement and up to two validated
  provider records;
- configured DNS route state;
- an optional bounded speed sample;
- a verdict, message key and unique finding codes.

The engine accepts location data only when the provider reports the exact
interface-bound IPv4. A `verified` result requires two distinct providers that
agree on country, matching default/interface IPv4 egress, no potential IPv6
leak, full-tunnel DNS state and no findings. The strict Desktop parser
recomputes these invariants and rejects unknown fields, invalid IP families,
provider duplication, provider-IP mismatch and unexplained non-verified
verdicts. The response contains no VPN endpoint, profile path, key or
configuration. `include_speed=false` is the default; the five-megabyte transfer
is never implicit.

`tests.verify-service-egress` is a separate read-only query. Its strict payload
is exactly `service` (`notebooklm`, `openai`, `google`, `antigravity` or `all`)
plus integer
`timeout_seconds` from 3 through 15. The engine sends credential-free HEAD
requests only to the built-in HTTPS allowlist, binds them to the selected VPN
interface, disables redirects and proxy inheritance, and caps response headers.
The strict result contains only schema/timestamp/scope plus service ID,
reachability, egress eligibility, reason code and an optional HTTP status.
NotebookLM trusts only the exact unsupported-location and home redirects.
OpenAI treats 401, or 405 with exact `Allow: POST`, as the authentication
boundary; 403 is an edge denial, while 429, 5xx and all unrecognized responses
remain indeterminate. Google (`generativelanguage.googleapis.com`) and
Antigravity (`daily-cloudcode-pa.googleapis.com`) treat 401, 403 or 404 as the
reached boundary (eligible), 429 as rate-limited and 5xx as unavailable, with all
other responses indeterminate. Network errors are unreachable/indeterminate. No URL,
header, body, address, account or credential is returned or persisted. This
query does not feed health recovery or planner evidence and does not prove
authentication, subscription, organization or content access.

Provider identity, display name, HTTPS probe endpoints, supported ISO 3166-1
alpha-2 countries, reason-code prefix and probe strategy come from the embedded
schema-version-1 provider registry. Google availability follows the official
Gemini API region list; Antigravity follows its official geography FAQ. The
registry is the only provider-selection source used by the CLI probe framework.

`region-check --provider ID [--target-country CC] --json` is a read-only
region-readiness check. It resolves the default IPv4 egress country only from
sanitized geo records that report that exact public IP, maps the system IANA
timezone to an ISO 3166-1 alpha-2 country through the installed timezone table,
and checks the provider registry. `country_consistent` is true exactly when the
egress and timezone countries match and that country is supported by the
provider. An optional target country adds strict target/provider, target/egress
and target/timezone gates. `ready` requires an empty `mismatches` array.

The result is schema version 1 and contains only the provider ID, country codes,
system timezone, booleans, verdict, stable `region.*` mismatch reason codes and
an `account_region_hint`. The hint explicitly describes the manual account
country association at layer L2; it does not claim that account, cookie,
subscription or browser region was inspected or changed. This command never
connects, disconnects or selects a VPN profile.

Installed CLI and TUI clients use the socket without `sudo` for status, profile
listing, batch endpoint probes, egress verification, connect, quick, reconnect
and disconnect. The client sends only an
opaque `profile_id`, sends a bounded query refresh deadline, bounds response
time and byte size, and accepts exactly one response document with matching
`api_version`/`request_id`. If a response is lost, it automatically retries the
same request with the same `action_id`, so the daemon returns the stored outcome
without applying the mutation twice. If the transport remains indeterminate,
the client does not run the same operation through `sudo`. Mutation failures
print the action ID needed for audit and recovery.

Desktop applies the same one-identical-retry rule to lifecycle requests. A
Desktop response is also read to bounded EOF and must contain exactly one
matching JSON document. A
Desktop location check consumes the structured `tests.probe` result and binds
each sanitized result to its opaque profile ID; it does not parse raw CLI text.
Desktop egress verification consumes the strict structured
`tests.verify-egress` result. A failed initial socket connection may use the
typed compatibility adapter
because no request was sent; any post-connect uncertainty is returned to the
user and never falls through to `pkexec`.

The installer adds `socat` for the Unix-socket client. The user must belong to
the `mazzy-vpn` group; a new login session may be required after initial group
enrollment.
