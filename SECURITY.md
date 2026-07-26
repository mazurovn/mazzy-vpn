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

Mazzy VPN Desktop does not read VPN profile files. It consumes the sanitized
`/run/mazzy-vpn/status.json` cache, which excludes endpoints, profile paths,
keys and configuration directives. Privileged operations accept a closed enum
and map it to fixed `mazzy-vpn` argument arrays through `pkexec`; arbitrary
shell input is not supported.

Linux is the only platform with a functional VPN backend in the current
Desktop release. macOS and Windows artifacts are unsigned UI previews and must
not be treated as traffic-protection tools.

Dependency updates are tracked for GitHub Actions, npm and Cargo. Release
candidates must pass the Bash regression suite, ShellCheck, Gitleaks, public
repository audit, Rust tests, Clippy and npm audit.
