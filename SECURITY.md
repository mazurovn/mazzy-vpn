# Security policy

Mazzy VPN is maintained by
[Nik m (@mazurovn)](https://github.com/mazurovn).

## Reporting a vulnerability

Please use the repository's private **Security → Report a vulnerability**
workflow on GitHub. Do not open a public issue for an unpatched vulnerability
and do not include real VPN keys, credentials, profiles, public IP history or
machine logs in a report.

Include the affected version, protocol, minimal reproduction and expected
impact. Redact all secrets. The maintainer will acknowledge a valid report as
soon as practical and coordinate disclosure after a fix is available.

Only the latest release is supported with security fixes.

## Desktop security boundary

Mazzy VPN Desktop does not read VPN profile files. It consumes strictly typed
status and profile caches that exclude endpoints, full profile paths, keys and
configuration directives. Cache, local API and egress-verification responses
are deserialized into strict `deny_unknown_fields` structures and checked for
cross-field consistency before they reach the webview. Active profiles are
matched by opaque ID or exact config basename, not a hard-coded location or an
ambiguous display name.
Privileged operations accept a closed enum and map it to fixed `mazzy-vpn`
argument arrays through the protected local API or `pkexec`; arbitrary shell
input is not supported.

Public IP is personal runtime data. Desktop hides it by default, screenshots
must use RFC 5737 documentation addresses, and public bug reports must redact
it. Actual-location verification sends only the VPN egress request to the
documented providers described in [PRIVACY.md](PRIVACY.md); profiles, endpoints
and keys are not sent.

Linux is the only platform with a functional VPN backend in the current
Desktop release. macOS and Windows artifacts are unsigned UI previews and must
not be treated as traffic-protection tools.

Dependency updates are tracked for GitHub Actions, npm and Cargo. Release
candidates must pass the Bash regression suite, ShellCheck, full-history
Gitleaks, the public repository and Desktop UI contract audits, Rust tests,
Clippy, npm audit, RustSec advisory checks through `cargo-deny` and
assembled-package inspection. The RustSec policy fails on vulnerability,
unsound and yanked advisories; unmaintained advisories are tracked separately so
they do not hide memory-safety defects. Passing these gates reduces risk but is
not a proof that the software or third-party VPN service is vulnerability-free.

Tauri 2.11.5 still uses the end-of-life GTK3 Rust bindings on Linux. For
`RUSTSEC-2024-0429`, the repository therefore carries the crates.io `glib`
0.18.5 source with the exact upstream `VariantStrIter` fix from gtk-rs commit
`b5a4071e439bef2b5eea76c3aa25e5ae84839e34`. Before `cargo-deny` runs,
`tests/check-glib-backport.py` verifies the crates.io archive checksum, compares
all upstream files, proves that the two reviewed mutability changes are the only
source delta and confirms the Cargo path override. The advisory ignore list
remains empty. This temporary backport must be removed when Tauri migrates to a
maintained GTK/glib line.

The public repository enables secret scanning with push protection,
Dependabot vulnerability alerts and security updates, private vulnerability
reporting, and CodeQL default setup with extended remote-and-local queries.
Repository scans complement the local gates; they do not replace review,
clean-host testing, signing or platform-native security assessment.
