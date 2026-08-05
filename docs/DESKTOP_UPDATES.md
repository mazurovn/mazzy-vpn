# Desktop signed updates / Подписанные обновления Desktop

Copyright (C) 2026 Nik m ([@mazurovn](https://github.com/mazurovn)).

## English

### User workflow

Desktop checks the fixed Mazzy VPN GitHub update feed once at startup by
default. The check can be disabled in **Settings -> Automatic update checks**,
and a manual check remains available. Finding a version never starts a download:
the application first shows the current and available versions in a modal.

- **AppImage, Windows, macOS:** **Download and install** uses the pending
  update, verifies its Tauri signature and installs it. A failed download keeps
  the same signed pending update available for an explicit retry. Restart
  remains an explicit second action where the platform installer does not exit itself.
  AppImage updates the Desktop bundle; after restart the existing installation
  report directs the user to explicit **Install / update / repair** when the
  system engine version differs from the newly bundled engine.
- **DEB/RPM:** **Open release** opens the exact `desktop-v<version>` page. The
  application does not overwrite a package-owned executable with an AppImage.
- **Later:** closes the dialog without downloading anything.

### Trust and publication

The endpoint and project release prefix are compile-time constants. The WebView
has neither raw `updater` nor `opener` permission and cannot provide a URL. A
pending `Update` exists only in Rust memory. Tauri verifies every installable
artifact with the public key embedded in `tauri.conf.json`; the encrypted
private key and its password are GitHub Actions secrets and are never committed.

Normal pull-request builds set `createUpdaterArtifacts=false`. The release
wrapper enables it only when `TAURI_SIGNING_PRIVATE_KEY` exists. A tagged
release remains a draft until Linux, Windows and macOS jobs have produced valid
non-empty signatures and same-tag asset URLs. Only then does CI publish the
versioned preview and replace `latest.json` on the fixed `desktop-updater`
release; both publication jobs use the protected `desktop-release` environment.

The Tauri updater signature authenticates an artifact to existing Mazzy VPN
Desktop installations. It is not Authenticode, Apple code signing/notarization,
an RPM signature or an APT repository signature. Windows/macOS also remain
UI-only previews until their native VPN backends pass separate release gates.

### Failure and rollback boundaries

- A network error is silent during the automatic check and visible during a
  manual check. It never changes VPN state.
- Missing, malformed, wrong-platform or invalidly signed metadata/artifacts fail
  before installation.
- A failed download leaves the consent dialog open and can be retried; it does
  not silently accept different metadata or a different version.
- Closing the dialog leaves the running version unchanged.
- No automatic post-install rollback is claimed. Keep the previous installer
  until the new version has started and passed diagnostics; DEB/RPM rollback is
  performed with the package manager.
- Loss of the private updater key prevents future in-app updates. Key rotation
  requires a transition release trusted by the key embedded in existing apps.

## Русский

### Сценарий пользователя

Desktop по умолчанию один раз при запуске проверяет фиксированный GitHub feed
Mazzy VPN. Проверку можно отключить в **Настройки -> Автопроверка обновлений**;
ручная проверка остаётся доступной. Найденная версия не запускает загрузку:
сначала приложение показывает modal dialog с текущей и новой версиями.

- **AppImage, Windows, macOS:** кнопка **Скачать и установить** использует
  pending update, проверяет Tauri-подпись и устанавливает его. При ошибке
  загрузки тот же подписанный pending update остаётся доступен для явного
  повтора. Restart остаётся отдельным явным действием, если platform installer
  не завершил приложение.
  AppImage обновляет Desktop bundle; после restart существующий installation
  report предлагает явный **Установить / обновить / исправить**, если версия
  system engine отличается от нового bundled engine.
- **DEB/RPM:** кнопка **Открыть релиз** открывает точную страницу
  `desktop-v<version>`. Приложение не заменяет package-owned executable на
  AppImage.
- **Позже:** закрывает dialog без загрузки.

### Trust boundary и публикация

Endpoint и project release prefix являются compile-time constants. У WebView
нет raw permission к `updater` или `opener`, и он не может передать URL.
Pending `Update` хранится только в памяти Rust. Tauri проверяет installable
artifact публичным ключом из `tauri.conf.json`; зашифрованный private key и его
пароль находятся только в GitHub Actions Secrets и не коммитятся.

Обычная PR-сборка использует `createUpdaterArtifacts=false`. Release wrapper
включает artifacts только при наличии `TAURI_SIGNING_PRIVATE_KEY`. Tagged
release остаётся draft, пока Linux, Windows и macOS jobs не создадут непустые
подписи и URL того же tag. Только после этого CI публикует versioned preview и
заменяет `latest.json` в фиксированном release `desktop-updater`; обе операции
публикации защищены environment `desktop-release`.

Tauri updater signature подтверждает artifact для уже установленного Mazzy VPN
Desktop. Она не является Authenticode, Apple code signing/notarization,
подписью RPM или APT repository. Windows/macOS также остаются UI-only preview
до прохождения отдельных gates нативных VPN backends.

### Ошибки и rollback

- Network error не показывается при автопроверке и показывается при ручной; VPN
  state не меняется.
- Missing, malformed, wrong-platform или invalidly signed metadata/artifact
  блокируется до установки.
- Ошибка загрузки оставляет consent dialog открытым и допускает повтор той же
  подписанной версии; другие metadata или версия молча не принимаются.
- Закрытие dialog сохраняет текущую работающую версию.
- Автоматический post-install rollback пока не заявлен. Предыдущий installer
  следует сохранять до запуска новой версии и прохождения диагностики;
  DEB/RPM rollback выполняется package manager.
- Потеря private updater key блокирует дальнейшие in-app updates. Rotation
  требует transition release, которому доверяет ключ в уже установленных apps.
