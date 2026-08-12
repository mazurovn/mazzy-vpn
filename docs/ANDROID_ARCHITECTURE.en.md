# Android Mazzy VPN: architecture and 15 audit iterations

Status: embedded AmneziaWG/WireGuard `device-test candidate`, not production.
The native Kotlin app builds a pinned userspace engine, owns one `VpnService`,
imports native profiles into a Keystore-encrypted envelope, establishes TUN,
installs profile routes/DNS and waits for a real handshake. Emulator,
physical-device routing/leak and signed-release gates remain open, so the
published registry honestly remains `android: planned`.

The app is native, uses Android's permission and foreground-service boundaries,
uses a separate strict native-profile parser, encrypts the complete profile
envelope with Android Keystore AES-GCM, and fails closed on parse, storage,
permission, TUN, socket-protection or handshake errors. Linux shell/systemd
adapters are not packaged or invoked inside the APK.
Endpoint bootstrap DNS is deliberately resolved on the underlying network
before TUN activation; after activation, imported profiles must provide DNS and
an IPv4 default route, and Android's unconfigured-family blocking remains in
force. The current candidate is explicitly IPv4 full-tunnel: IPv6 is blocked,
not leaked, until dual-stack device gates are implemented. A bounded
PersistentKeepalive is required so handshake readiness is observable. Always-on
is disabled until durable selected-profile reboot recovery is device-tested.
The nine modern proxy/transport protocols remain catalog entries, not Android
connect support.

## 15-iteration audit loop

Each iteration is implement -> static review -> unit tests -> adversarial review ->
record findings. Any defect resets `clean_streak` to zero. Work can stop only after
five consecutive clean audits and all runtime/device gates are closed.

1. Gradle wrapper, locked dependencies and reproducible debug build.
2. Schema/registry parity and profile validator tests.
3. Keystore-backed secrets, rotation and redacted logs.
4. SAF/share import with atomic commit and rollback.
5. `VpnService.prepare`, denial, revoke and re-approval.
6. Foreground lifecycle, notification, stop and crash cleanup.
7. Pinned native engine, ABI and license inventory.
8. TUN packet ownership, cancellation and backpressure.
9. IPv4/IPv6 routes, DNS policy and kill switch.
10. Network changes, captive portal, Doze and sleep/wake.
11. Handshake timeout, crash recovery and rollback.
12. Redacted diagnostics and typed errors.
13. Allowlisted agent API without shell execution.
14. Emulator plus real-device permission and leak tests.
15. Signed APK/AAB, Data safety, release verification and five clean audits.

An APK build is not device proof. Until emulator, physical-device, leak and
signed-release gates pass, Android remains a preview candidate.
