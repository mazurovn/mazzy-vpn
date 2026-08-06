# Android Mazzy VPN: architecture and 15 audit iterations

Status: `foundation`, not production. The repository now has a native Kotlin
foundation with `VpnService`, profile import and encrypted storage. A real
protocol engine, TUN packet loop, DNS forwarder, routing and device leak tests
remain open. No protocol is promoted from `android: planned`.

The app is native, uses Android's permission and foreground-service boundaries,
validates the shared v1 profile contract before storage, encrypts active/previous
slots with Android Keystore AES-GCM, and deliberately fails closed until a pinned
engine adapter exists. Linux shell/systemd adapters are not reusable inside APK.
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

Presence of Kotlin source is not a completed client. Until real engine, device,
leak and signed-release gates pass, Android remains `foundation`/`preview`.
