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
protected local service transport is explicitly `planned`; publishing the
schema does not claim that the final daemon exists.

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

Repeated delivery of the same action ID must never apply the mutation twice.
This idempotency rule will be enforced by the protected local service.

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
