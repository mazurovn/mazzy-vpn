# Глубокий аудит установки и Desktop-релиза 2026-08-07

## Итог

До этого цикла рабочий код не доходил до пользователя в виде рабочего
релиза. На проверенной машине были установлены `mazzy-vpn-desktop 0.4.1` и
CLI `1.4.1`. Git tag `v1.4.4` существовал, но опубликованного CLI release не
было, а `desktop-v0.4.4` оставался draft/prerelease.

Существующий draft DEB содержал CLI и systemd units, но был собран с commit
до последних исправлений. Поэтому проверка исходников ветки не доказывала,
что установленный пакет содержит эти исправления.

## Найденные дефекты

1. Desktop при отсутствии или несовпадении engine только показывал кнопку
   Install / Repair. Пользователь должен был сам найти её, поэтому свежая
   установка выглядела как Desktop без CLI.
2. `get_installation_report()` считал `needs_install` по всем отсутствующим
   зависимостям. Необязательные альтернативные tunnel backends и L2TP/IPsec
   блокировали готовую установку, хотя bootstrap использовал более мягкое и
   корректное правило: core/Desktop зависимости плюс хотя бы один backend.
3. CI не имел `workflow_dispatch` для основного regression workflow, поэтому
   release-аудит нельзя было воспроизводимо запустить вручную на exact commit.
4. Headless package smoke на Xvfb мог падать внутри accessibility bridge и
   libayatana-appindicator с `SIGSEGV`. Это было свойством тестового окружения,
   а не установленного приложения; smoke теперь запускается с
   `NO_AT_BRIDGE=1`.

## Исправления

- Добавлен одноразовый consent-gated startup prompt: Desktop спрашивает,
  запускать ли системный Install / Repair, и использует тот же privileged
  bootstrap path, что и ручная кнопка.
- Installation report теперь использует тот же `dependencies_ready()`, что и
  bootstrap, поэтому optional protocol packages не объявляются блокером.
- Добавлен ручной запуск `.github/workflows/ci.yml`.
- Package audit усилен headless-настройками и проверяет payload DEB/RPM/AppImage
  на byte-level совпадение с текущим source tree.

## Проверки exact source tree

- `./tests/run.sh`: `1..103`, все тесты passed.
- `python3 tests/check-desktop-ui.py`: passed, 110 IDs и 160 localized labels.
- `cargo check --manifest-path desktop/src-tauri/Cargo.toml --all-targets`:
  passed; vendor glib emits only existing lifetime warnings.
- `npm run build:release`: DEB, RPM и AppImage built.
- `./tests/check-linux-packages.sh`: payload, dependencies, lifecycle, systemd
  и GUI launch verified.

## Что ещё не является production release

- Linux Desktop is a functional preview and includes the shared CLI/engine.
- macOS and Windows remain UI previews: Network Extension, Windows
  Service/Wintun, native privilege helpers, code signing and notarization are
  not implemented.
- Android has a contract/foundation audit, but no signed production APK/AAB
  and no device-level VPN/TUN acceptance gate.
- Modern protocols remain catalog/import/planner foundations where the runtime
  adapter and TUN/DNS/routing/rollback gates are not green. They must not be
  advertised as working connect support.

## Release rule

A release is publishable only after the exact tagged commit has green CI,
Desktop Linux package audit, signed artifact checks and a release page with
non-draft assets. A draft or a locally built package is not a release and must
not be used as proof of an install on another computer.
