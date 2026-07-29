# Mazzy VPN — 한국어 가이드

제작 및 유지관리:
[Nik m (@mazurovn)](https://github.com/mazurovn)

Mazzy VPN은 AmneziaWG, WireGuard, OpenVPN 및 NetworkManager L2TP/IPsec을
하나로 관리하는 Linux VPN 도구입니다. 대화형 TUI, 자동화용 CLI, 안전한 설정
가져오기, 연결 검사, 자동 롤백이 포함된 실제 터널 테스트를 제공합니다.

[영문 아키텍처 및 동작 다이어그램](docs/ARCHITECTURE.en.md) ·
[러시아어 아키텍처 다이어그램](docs/ARCHITECTURE.ru.md)

## 설치 및 언어

```bash
git clone https://github.com/mazurovn/mazzy-vpn.git
cd mazzy-vpn
sudo ./install.sh
mazzy-vpn
```

대화형 설치 시작 시 6개 언어 중 하나를 선택할 수 있습니다. 자동 설치:

```bash
sudo ./install.sh --lang ko --yes
sudo ./install.sh --lang ko --config-dir ~/VPN-configs
```

설치 후 메뉴 16번 또는 CLI에서 즉시 언어를 바꿀 수 있습니다.

```bash
mazzy-vpn language
mazzy-vpn language ko
```

지원 코드: `ru`, `en`, `de`, `zh`, `ja`, `ko`.

## 대시보드와 빠른 연결

대시보드는 TUI 메뉴 위에 표시되며 별도 명령으로도 사용할 수 있습니다.

```bash
mazzy-vpn dashboard
mazzy-vpn quick
```

실제 터널 및 인터넷 상태, 선택 위치, 프로토콜, 기본 설정, 인터페이스,
핸드셰이크 경과 시간, 공인 IP, 자동 시작, 상태 모니터, 대체 VPN 및 프로필
개수를 한 화면에서 보여 줍니다.

`mazzy-vpn quick`은 다시 선택하지 않고 저장된 기본 설정으로 연결합니다. 기본
설정이 없으면 프로필 선택 화면을 열고 선택한 값을 새 기본값으로 저장합니다.

### Desktop 대시보드와 트레이

Tauri Desktop은 Quick Connect, Reconnect, Disconnect, Refresh,
Self-diagnostics를 창과 시스템 트레이에서 제공합니다. Linux 버전은 설치된
CLI와 함께 실제로 동작하며 AppImage, DEB, RPM으로 제공됩니다. macOS와
Windows는 네이티브 VPN 백엔드가 구현되기 전까지 UI 미리보기입니다. GUI는
프로필이나 키를 읽지 않습니다. issue #31이 닫히기 전에는 Desktop 0.3을 새
preview로 게시하지 마십시오. 현재 Tauri/GTK `glib` 0.18은 RustSec gate를
통과하지 못합니다. 자세한 내용:
[Desktop guide (English)](docs/DESKTOP.en.md).

## 주요 명령

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

## 자동 감시 및 복구

VPN 프로세스가 예기치 않게 종료되면 systemd가 다시 시작합니다. 별도의 상태
타이머는 약 20초마다 원하는 상태, 서비스, VPN 인터페이스, 그리고 해당
인터페이스를 통한 실제 HTTPS 연결을 확인합니다. `DESIRED=up`인데 서비스가
중지되어 있으면 즉시 시작하고, 트래픽 검사가 두 번 연속 실패하면 다시
연결합니다. `sudo mazzy-vpn doctor --fix`는 감시를 활성화하고 유효한 기본
프로필의 자동 시작을 복구합니다.

## 설정 폴더

```bash
mazzy-vpn init-config-dir ~/MazzyConfigs
mazzy-vpn import-dir ~/VPN-configs --dry-run
sudo mazzy-vpn import-dir ~/VPN-configs
```

`amneziawg/`, `wireguard/`, `openvpn/`, `l2tp/` 구조가 생성됩니다. 프로토콜은
파일 내용으로 판별하며, 복사 전에 검증하고 권한 `600`으로 설치합니다. 실행 가능한
hook, 중첩 OpenVPN 설정, 필수 항목이 없는 프로필은 거부합니다.

## 테스트, 진단 및 롤백

`validate all`은 연결 없이 모든 설정을 검사합니다. `probe all`은 전체 위치 목록을
제한된 병렬 처리로 검사하고 도달 가능성, 지연 시간, 현재 활성 터널을 표시합니다.
UDP에서 ICMP가 차단되면 장애가 아니라 알 수 없음으로 표시합니다. `diagnose`는 경로, DNS, 서비스,
인터페이스, 핸드셰이크 및 VPN을 통한 인터넷 연결을 확인합니다.

`test`와 `test-all`은 이전 연결을 저장하고 실제 터널을 검사한 뒤 성공, 실패,
시간 초과 또는 신호 발생 시 복원합니다. `--keep`을 명시한 경우에만 성공한 연결을
유지합니다. OpenVPN의 `Too many connections`는 서버 측 제한으로 정확히 보고되고
기존 연결이 즉시 복구됩니다.

## 보안

실제 개인 키, PSK, 비밀번호 또는 개인 VPN 설정을 Git에 올리지 마십시오. 공개
저장소에는 운영 프로필이 포함되지 않으며 로컬 설정은 `/etc/vpnctl/profiles`에
권한 `600`으로 저장됩니다. 릴리스 전 회귀 테스트, ShellCheck, 공개 내용 감사 및
Gitleaks 검사를 실행합니다.

## 저자 및 라이선스

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn).
[GNU AGPL v3.0 이상](LICENSE)으로 배포됩니다. AGPL과 원저자 고지를 유지하는
조건으로 수정 및 재배포할 수 있습니다.
