# Mazzy VPN 1.2.0 / Desktop 0.2.0

Copyright (C) 2026 Nik m
([@mazurovn](https://github.com/mazurovn)).

## English

Mazzy VPN 1.2.0 expands the safe Linux CLI/TUI and turns Desktop 0.2 into a
self-contained Linux control-center preview for AmneziaWG, WireGuard, OpenVPN
and NetworkManager L2TP/IPsec profiles.

### Highlights

- Desktop screens for Dashboard, Profiles, Diagnostics, Settings and About,
  plus the system tray;
- bundled compatible engine and installer in Linux Desktop packages, with
  version/dependency checks and explicitly authorized install, update and
  repair;
- safe file/folder import, sanitized profile discovery, search, default/selected
  connections and profile removal;
- validation, endpoint probes, transactional single/batch live tests,
  emergency recovery and connection diagnosis;
- complete retained Doctor and self-test output, bounded service logs,
  autostart and independent recovery-monitor controls;
- typed Desktop operations with a fixed allowlist, argument/path validation and
  output sanitization instead of UI-generated shell commands;
- CLI additions for sanitized `profiles --json`, multi-file import, independent
  monitor control and bounded logs;
- About, privacy and security guidance with product/engine/platform versions,
  author and AGPL license information;
- machine-validated capability gates and bilingual roadmaps for Linux, Windows,
  macOS, Android and iOS;
- synchronized Wiki, Discussions FAQ/support topics and community feature
  polls.

### Platform status

- **Linux CLI/TUI 1.2.0:** functional release.
- **Linux Desktop 0.2.0:** functional control-center preview with bundled
  engine bootstrap; it remains preview until the Desktop 1.0 release gate
  passes.
- **Windows and macOS Desktop artifacts:** unsigned UI previews only. They do
  not provide traffic protection until native services/backends, signing and
  platform integration tests are complete.
- **Android and iOS:** planned native clients; no working mobile packages are
  included.

Install the CLI/TUI from source:

```bash
git clone https://github.com/mazurovn/mazzy-vpn.git
cd mazzy-vpn
sudo ./install.sh
mazzy-vpn
```

Linux Desktop packages can bootstrap the compatible engine after explicit OS
authorization. Private VPN profiles and credentials are intentionally absent
from all release artifacts.

## Русский

Mazzy VPN 1.2.0 расширяет безопасный Linux CLI/TUI, а Desktop 0.2 становится
самодостаточным Linux control-center preview для профилей AmneziaWG, WireGuard,
OpenVPN и NetworkManager L2TP/IPsec.

### Основные изменения

- экраны Desktop Dashboard, Profiles, Diagnostics, Settings и «О программе»,
  а также системный tray;
- совместимый engine и installer внутри Linux Desktop packages, проверка версий
  и зависимостей, явно подтверждаемая установка, обновление и repair;
- безопасный импорт файлов/папок, очищенный список профилей, поиск, подключение
  default/выбранного профиля и удаление;
- validation, endpoint probe, транзакционные одиночные/пакетные live tests,
  emergency recovery и диагностика соединения;
- полный сохранённый вывод Doctor и self-test, ограниченный журнал, управление
  autostart и независимым recovery monitor;
- типизированные Desktop operations с фиксированным allowlist, проверкой
  аргументов/путей и очисткой вывода вместо shell-команд из UI;
- CLI-команды для очищенного `profiles --json`, импорта нескольких файлов,
  отдельного управления monitor и ограниченного журнала;
- экран About, правила приватности и безопасности с версиями
  product/engine/platform, автором и лицензией AGPL;
- машинно проверяемые capability gates и двуязычный roadmap для Linux, Windows,
  macOS, Android и iOS;
- синхронизированные Wiki, Discussions FAQ/support и голосования за будущие
  функции.

### Статус платформ

- **Linux CLI/TUI 1.2.0:** функциональный релиз.
- **Linux Desktop 0.2.0:** функциональный control-center preview со встроенным
  bootstrap engine; статус preview сохраняется до прохождения release gate
  Desktop 1.0.
- **Windows и macOS Desktop artifacts:** только неподписанные UI preview. Они не
  защищают трафик до появления нативных services/backends, подписи и platform
  integration tests.
- **Android и iOS:** запланированы нативные клиенты; рабочих mobile packages в
  этом релизе нет.

Установка CLI/TUI из исходников:

```bash
git clone https://github.com/mazurovn/mazzy-vpn.git
cd mazzy-vpn
sudo ./install.sh
mazzy-vpn
```

Linux Desktop packages могут установить совместимый engine после явного
системного разрешения. Личные VPN-профили и учётные данные намеренно не входят
ни в один release artifact.

## Safety / Безопасность

Live tests temporarily change the active VPN route and use transactional
rollback after success, failure, timeout or termination. Review the prompt and
save important network work before running them.

Live tests временно меняют активный VPN-маршрут и выполняют transactional
rollback после успеха, ошибки, тайм-аута или завершения. Перед запуском
прочитайте подтверждение и сохраните важную сетевую работу.

Licensed under the GNU Affero General Public License v3.0 or later.
