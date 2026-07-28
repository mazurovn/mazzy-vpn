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
Clippy, npm audit and assembled-package inspection. Passing these gates reduces
risk but is not a proof that the software or third-party VPN service is
vulnerability-free.

The public repository enables secret scanning with push protection,
Dependabot vulnerability alerts and security updates, private vulnerability
reporting, and CodeQL default setup with extended remote-and-local queries.
Repository scans complement the local gates; they do not replace review,
clean-host testing, signing or platform-native security assessment.
