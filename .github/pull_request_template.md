## What changed

Describe the user-visible outcome and the capability IDs from
`docs/capabilities.json`.

## Surface parity

- [ ] Shared core/API behavior is implemented or explicitly not applicable.
- [ ] CLI behavior and completion are updated.
- [ ] TUI menu/dashboard behavior is updated.
- [ ] Desktop Linux behavior is updated.
- [ ] Desktop macOS behavior is updated or remains explicitly marked preview.
- [ ] Desktop Windows behavior is updated or remains explicitly marked preview.
- [ ] `docs/capabilities.json` statuses and release gates are accurate.
- [ ] Russian and English documentation/Wiki sources are synchronized.

## Verification

- [ ] `./tests/run.sh`
- [ ] `python3 tests/check-capabilities.py`
- [ ] `./tests/audit-public.sh`
- [ ] Rust tests and Clippy when Desktop/core code changed
- [ ] npm audit when Desktop dependencies changed
- [ ] No real profiles, credentials, personal paths or machine state are present

Copyright © 2026 Nik m ([@mazurovn](https://github.com/mazurovn)).
