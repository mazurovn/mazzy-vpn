# Mazzy VPN — Deutsche Anleitung

Erstellt und gepflegt von
[Nik m (@mazurovn)](https://github.com/mazurovn).

Mazzy VPN ist ein Linux-Manager für AmneziaWG, WireGuard, OpenVPN und
NetworkManager L2TP/IPsec. Er bietet eine interaktive TUI, eine skriptfähige
CLI, sichere Profilimporte, Verbindungsprüfungen und transaktionale Live-Tests
mit automatischem Rollback.

[Architekturdiagramme auf Englisch](docs/ARCHITECTURE.en.md) ·
[Architektur auf Russisch](docs/ARCHITECTURE.ru.md)

## Installation und Sprache

```bash
git clone https://github.com/mazurovn/mazzy-vpn.git
cd mazzy-vpn
sudo ./install.sh
mazzy-vpn
```

Am Anfang der interaktiven Installation kann eine von sechs Sprachen gewählt
werden. Für eine automatische Installation:

```bash
sudo ./install.sh --lang de --yes
sudo ./install.sh --lang de --config-dir ~/VPN-Konfigurationen
```

Die Sprache kann später sofort über Menüpunkt 16 oder per CLI geändert werden:

```bash
mazzy-vpn language
mazzy-vpn language de
```

Unterstützte Codes: `ru`, `en`, `de`, `zh`, `ja`, `ko`.

## Dashboard und Schnellverbindung

Das Dashboard wird oberhalb des TUI-Menüs angezeigt:

```bash
mazzy-vpn dashboard
mazzy-vpn quick
```

Es zeigt den tatsächlichen Tunnel- und Internetstatus, den gewählten Standort,
das Protokoll, die Standardkonfiguration, die Schnittstelle, das Alter des
Handshakes, die öffentliche IP, Autostart, Zustandsüberwachung, Fallback und
Profilanzahlen.

`mazzy-vpn quick` verbindet die gespeicherte Standardkonfiguration ohne erneute
Auswahl. Ist noch kein Standard vorhanden, öffnet Mazzy VPN die Profilauswahl.

### Desktop Dashboard und Tray

Die Tauri-Desktop-App bietet Quick Connect, Reconnect, Disconnect, Refresh und
Selbstdiagnose in einem modernen Fenster und im System-Tray. Unter Linux ist sie
ein funktionsfähiger Begleiter des installierten CLI und wird als AppImage, DEB
und RPM gebaut. macOS und Windows sind UI-Vorschauen, bis native VPN-Backends
implementiert sind. Desktop 0.3 darf nicht als neuer Preview veröffentlicht
werden, bevor PR #32, der Default-Branch-Dependabot-Scan und die Release-Seiten
grün sind. Der Candidate enthält bereits den geprüften `glib`-Backport für
issue #31 und besteht den lokalen RustSec-Gate ohne Ausnahmen. Die GUI liest
keine Profile oder Schlüssel. Details:
[Desktop guide (English)](docs/DESKTOP.en.md).

## Wichtige Befehle

```bash
mazzy-vpn
mazzy-vpn list
mazzy-vpn quick
sudo mazzy-vpn connect amneziawg 1
sudo mazzy-vpn disconnect
mazzy-vpn diagnose
mazzy-vpn validate all
mazzy-vpn probe all --timeout 3 --jobs 4
sudo mazzy-vpn test openvpn 1 --timeout 60
sudo mazzy-vpn test-all all --timeout 30
sudo mazzy-vpn emergency --timeout 20
mazzy-vpn self-test
sudo mazzy-vpn doctor --fix
sudo mazzy-vpn autostart on
```

## Automatische Überwachung und Reparatur

Systemd startet den VPN-Prozess nach jedem unerwarteten Ende neu. Unabhängig
davon prüft der Health-Timer ungefähr alle 20 Sekunden den Sollzustand, den
Dienst, die VPN-Schnittstelle und echten HTTPS-Zugriff über diese Schnittstelle.
Ein bei `DESIRED=up` inaktiver Dienst wird sofort gestartet; zwei
aufeinanderfolgende Verkehrsfehler lösen eine Neuverbindung aus.
`sudo mazzy-vpn doctor --fix` aktiviert die Überwachung und repariert den
Autostart für ein gültiges Standardprofil.

## Profilordner

```bash
mazzy-vpn init-config-dir ~/MazzyConfigs
mazzy-vpn import-dir ~/VPN-Konfigurationen --dry-run
sudo mazzy-vpn import-dir ~/VPN-Konfigurationen
```

Die Struktur enthält `amneziawg/`, `wireguard/`, `openvpn/` und `l2tp/`.
Profile werden anhand ihres Inhalts erkannt, vor dem Kopieren validiert und mit
Modus `600` installiert. Ausführbare Hooks, verschachtelte OpenVPN-Konfigurationen
und unvollständige Profile werden abgelehnt.

## Tests, Diagnose und Rollback

`validate all` prüft alle Dateien ohne Verbindung. `probe all` prüft die ganze
Standortliste begrenzt parallel und meldet Erreichbarkeit, Latenz und den
aktiven Tunnel. Blockiertes ICMP bei UDP wird als unbekannt statt als Ausfall
gemeldet. `diagnose` prüft Route, DNS, Dienst,
Schnittstelle, Handshake und Internetzugriff über den VPN-Tunnel.

`test` und `test-all` speichern den vorherigen Zustand, testen einen echten
Tunnel und stellen ihn bei Erfolg, Fehler, Timeout oder Signal wieder her.
`--keep` behält nur eine erfolgreich getestete Verbindung. Meldet ein
OpenVPN-Anbieter `Too many connections`, wird dies als serverseitiges Limit
angezeigt und die vorherige Verbindung sofort wiederhergestellt.

## Sicherheit

Echte private Schlüssel, PSKs, Passwörter und persönliche Profile gehören
nicht in Git. Das öffentliche Repository enthält keine Betriebsprofile; diese
liegen lokal unter `/etc/vpnctl/profiles` mit Modus `600`. Das Release wird mit
Regressionstests, ShellCheck, einem öffentlichen Audit und Gitleaks geprüft.

## Autor und Lizenz

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).
Lizenz: [GNU AGPL v3.0 oder später](LICENSE). Änderungen und Weitergabe sind
unter Einhaltung der AGPL und unter Beibehaltung der Urheberhinweise erlaubt.
