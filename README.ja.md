# Mazzy VPN — 日本語ガイド

作者・メンテナー:
[Nik m (@mazurovn)](https://github.com/mazurovn)

Mazzy VPN は AmneziaWG、WireGuard、OpenVPN、NetworkManager L2TP/IPsec
を一つにまとめる Linux VPN マネージャーです。対話型 TUI、自動化向け CLI、
安全な設定インポート、接続確認、自動ロールバック付きの実接続テストを提供します。

検証済みカタログは 13 プロトコルになりました。実際に接続できる Linux
バックエンドは上記 4 種類のままです。VLESS/REALITY、Hysteria 2、Mieru、
NaiveProxy、TUIC v5、Shadowsocks 2022、Trojan、AnyTLS、ShadowTLS v3 は
登録済みですが、接続状態は明示的に `planned` です。9 種類すべてに閉じた
managed profile schema とアトミックな Linux import が追加され、6 種類には
閉じた sing-box config renderer があります。ただし接続 lifecycle にはまだ
統合されていません。詳細:
[Protocol orchestration](docs/PROTOCOL_ORCHESTRATION.en.md)。

[英語のアーキテクチャ図](docs/ARCHITECTURE.en.md) ·
[ロシア語のアーキテクチャ図](docs/ARCHITECTURE.ru.md)

## インストールと言語

```bash
git clone https://github.com/mazurovn/mazzy-vpn.git
cd mazzy-vpn
sudo ./install.sh
mazzy-vpn
```

対話型インストールの最初に 6 言語から選択できます。自動インストール:

```bash
sudo ./install.sh --lang ja --yes
sudo ./install.sh --lang ja --config-dir ~/VPN-configs
```

インストール後はメニュー項目 16、または次のコマンドで即時変更できます。

```bash
mazzy-vpn language
mazzy-vpn language ja
```

対応コード: `ru`、`en`、`de`、`zh`、`ja`、`ko`。

## ダッシュボードとクイック接続

ダッシュボードは TUI メニューの上部に表示され、単独でも実行できます。

```bash
mazzy-vpn dashboard
mazzy-vpn quick
```

実際のトンネルとインターネット状態、選択ロケーション、プロトコル、既定の設定、
インターフェース、ハンドシェイク経過時間、パブリック IP、自動起動、
ヘルス監視、フォールバック、各プロファイル数を一画面で確認できます。

`mazzy-vpn quick` は保存済みの既定設定へ再選択なしで接続します。既定設定が
ない場合は選択画面を開き、その選択を新しい既定値として保存します。

### Desktop ダッシュボードとトレイ

Tauri Desktop は Quick Connect、Reconnect、Disconnect、Refresh、
Self-diagnostics をウィンドウとシステムトレイから提供します。Linux 版は
インストール済み CLI と連携して動作し、AppImage、DEB、RPM を用意します。
macOS と Windows はネイティブ VPN バックエンド実装前の UI プレビューです。
Desktop 0.4 は未署名 preview として公開済みです。Issue #31 はレビュー済みの
upstream `glib` backport、厳密な source provenance 検証、成功した RustSec、
Dependabot、CodeQL checks により解決済みです。GUI はプロファイルや鍵を読みません。詳細:
[Desktop guide (English)](docs/DESKTOP.en.md)。

## 主なコマンド

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

## 自動監視と修復

VPN プロセスが予期せず終了すると systemd が再起動します。独立したヘルス
タイマーは約 20 秒ごとに、希望状態、サービス、VPN インターフェース、および
そのインターフェース経由の実際の HTTPS 接続を確認します。
`DESIRED=up` なのにサービスが停止している場合は直ちに起動し、通信チェックが
2 回連続で失敗すると再接続します。`sudo mazzy-vpn doctor --fix` は監視を有効に
し、有効な既定プロファイルの自動起動を修復します。

## 設定フォルダー

```bash
mazzy-vpn init-config-dir ~/MazzyConfigs
mazzy-vpn import-dir ~/VPN-configs --dry-run
sudo mazzy-vpn import-dir ~/VPN-configs
```

`amneziawg/`、`wireguard/`、`openvpn/`、`l2tp/` が作成されます。プロトコルは
内容から判定され、コピー前に検証され、モード `600` で保存されます。実行可能な
hook、ネストした OpenVPN 設定、不完全なプロファイルは拒否されます。

## テスト、診断、ロールバック

`validate all` は接続せず全設定を検証します。`probe all` はロケーション一覧を
制限付き並列で確認し、到達性、遅延、現在のアクティブトンネルを表示します。
UDP で ICMP が遮断された場合は障害ではなく不明と表示します。`diagnose` はルート、DNS、サービス、
インターフェース、ハンドシェイク、VPN 経由のインターネットを確認します。

`test` と `test-all` は元の接続を保存して実際のトンネルを検証し、成功、失敗、
タイムアウト、シグナルの後に復元します。`--keep` を明示した場合のみ、成功した
接続を維持します。OpenVPN の `Too many connections` はサーバー側制限として
報告され、元の接続が直ちに復元されます。

## セキュリティ

実際の秘密鍵、PSK、パスワード、個人用 VPN 設定を Git に含めないでください。
公開リポジトリに運用設定はなく、ローカル設定は `/etc/vpnctl/profiles` に
モード `600` で保存されます。リリース前に回帰テスト、ShellCheck、公開監査、
Gitleaks を実行します。

## 作者とライセンス

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn)。
[GNU AGPL v3.0 以降](LICENSE) で公開されています。AGPL と原著作者表示を
維持する条件で変更・再配布できます。
