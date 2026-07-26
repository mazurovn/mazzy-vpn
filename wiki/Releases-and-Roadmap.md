# Релизы и roadmap

## 1.1.0

- CLI/TUI и двухуровневое автовосстановление;
- безопасный JSON status cache;
- Tauri Desktop Dashboard и tray;
- рабочие Linux AppImage, DEB и RPM;
- macOS app/DMG и Windows MSI/NSIS как UI preview;
- документация, схемы и Wiki на русском и английском;
- CI, Rust tests, ShellCheck, Clippy, npm audit и leak audit.

## Следующие этапы

1. нативный macOS backend: Network Extension, launchd, signing/notarization;
2. нативный Windows backend: Windows service, Wintun, code signing;
3. подписанные installers и автоматическая проверка обновлений;
4. расширенный выбор профилей из Desktop без раскрытия ключей;
5. уведомления о recovery и история health-событий;
6. accessibility и platform-specific integration tests.

Пока пункты 1–2 не завершены, macOS/Windows должны оставаться preview и не
называться рабочим VPN-клиентом.

---

<a id="english"></a>

# Releases and roadmap

Version 1.1.0 adds the sanitized status cache, Tauri Dashboard and tray,
functional Linux AppImage/DEB/RPM bundles, macOS/Windows UI previews, bilingual
architecture/Wiki and expanded CI/security checks.

Next milestones are a native macOS Network Extension/launchd backend, a Windows
service/Wintun backend, code signing and notarization, safe Desktop profile
selection, recovery notifications/history, accessibility and platform-specific
integration tests.

Until native backends are complete, macOS and Windows artifacts remain previews
and must not be called working VPN clients.
