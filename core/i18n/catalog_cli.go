// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package i18n

// cliCatalog holds CLI command output strings in all six languages. It is
// merged into the main catalog at init so Translator.T/Tf resolve them too.
// Keys use the "cli.<command>.<what>" convention. Printf verbs inside a value
// must match across languages.
var cliCatalog = map[string]map[Lang]string{
	// --- generic ---
	"cli.err.lock": {
		EN: "another mazzy-vpn operation is in progress",
		RU: "другая операция mazzy-vpn уже выполняется",
		DE: "eine andere mazzy-vpn-Operation läuft bereits",
		ZH: "另一个 mazzy-vpn 操作正在进行中",
		JA: "別の mazzy-vpn 操作が実行中です",
		KO: "다른 mazzy-vpn 작업이 진행 중입니다",
	},
	"cli.err.need_root": {
		EN: "this command needs root; re-run with sudo",
		RU: "команде нужны права root; запустите через sudo",
		DE: "dieser Befehl benötigt root; mit sudo erneut ausführen",
		ZH: "此命令需要 root 权限；请使用 sudo 重新运行",
		JA: "このコマンドには root が必要です。sudo で再実行してください",
		KO: "이 명령은 root 권한이 필요합니다. sudo로 다시 실행하세요",
	},

	// --- connect ---
	"cli.connect.connecting": {
		EN: "Connecting %s (%s)...",
		RU: "Подключение %s (%s)...",
		DE: "Verbinde %s (%s)...",
		ZH: "正在连接 %s (%s)...",
		JA: "%s (%s) に接続中...",
		KO: "%s (%s) 연결 중...",
	},
	"cli.connect.pin_uplink": {
		EN: "Pinning egress to uplink: %s",
		RU: "Привязка выхода к аплинку: %s",
		DE: "Ausgang an Uplink binden: %s",
		ZH: "将出口绑定到上行接口：%s",
		JA: "エグレスをアップリンクに固定: %s",
		KO: "이그레스를 업링크에 고정: %s",
	},
	"cli.connect.failed": {
		EN: "connect failed:",
		RU: "не удалось подключиться:",
		DE: "Verbindung fehlgeschlagen:",
		ZH: "连接失败：",
		JA: "接続に失敗しました:",
		KO: "연결 실패:",
	},
	"cli.connect.verifying": {
		EN: "Verifying protected egress",
		RU: "Проверка защищённого выхода",
		DE: "Geschützten Ausgang prüfen",
		ZH: "正在验证受保护的出口",
		JA: "保護された出口を検証中",
		KO: "보호된 이그레스 확인 중",
	},
	"cli.connect.ok": {
		EN: "✔ CONNECTED and protected.",
		RU: "✔ ПОДКЛЮЧЕНО и защищено.",
		DE: "✔ VERBUNDEN und geschützt.",
		ZH: "✔ 已连接并受保护。",
		JA: "✔ 接続され保護されています。",
		KO: "✔ 연결됨 및 보호됨.",
	},
	"cli.connect.interface": {
		EN: "  interface : %s",
		RU: "  интерфейс : %s",
		DE: "  Schnittstelle : %s",
		ZH: "  接口 : %s",
		JA: "  インターフェース : %s",
		KO: "  인터페이스 : %s",
	},
	"cli.connect.egress": {
		EN: "  egress IP : %s",
		RU: "  выходной IP : %s",
		DE: "  Ausgangs-IP : %s",
		ZH: "  出口 IP : %s",
		JA: "  出口 IP : %s",
		KO: "  이그레스 IP : %s",
	},
	"cli.connect.protocol": {
		EN: "  protocol  : %s",
		RU: "  протокол  : %s",
		DE: "  Protokoll  : %s",
		ZH: "  协议  : %s",
		JA: "  プロトコル  : %s",
		KO: "  프로토콜  : %s",
	},
	"cli.connect.not_confirmed": {
		EN: "⚠ Interface %s is up, but egress is NOT confirmed: %s",
		RU: "⚠ Интерфейс %s поднят, но выход НЕ подтверждён: %s",
		DE: "⚠ Schnittstelle %s ist aktiv, aber der Ausgang ist NICHT bestätigt: %s",
		ZH: "⚠ 接口 %s 已启用，但出口未确认：%s",
		JA: "⚠ インターフェース %s は稼働中ですが、出口が未確認です: %s",
		KO: "⚠ 인터페이스 %s가 활성화되었지만 이그레스가 확인되지 않음: %s",
	},
	"cli.connect.dashboard_reconnect": {
		EN: "\nLive dashboard + auto-reconnect (Ctrl+C to disconnect):",
		RU: "\nЖивая панель + автопереподключение (Ctrl+C — отключить):",
		DE: "\nLive-Dashboard + automatische Wiederverbindung (Strg+C zum Trennen):",
		ZH: "\n实时面板 + 自动重连（Ctrl+C 断开）：",
		JA: "\nライブダッシュボード + 自動再接続 (Ctrl+C で切断):",
		KO: "\n실시간 대시보드 + 자동 재연결 (Ctrl+C로 연결 해제):",
	},
	"cli.connect.dashboard": {
		EN: "\nLive dashboard (Ctrl+C to disconnect):",
		RU: "\nЖивая панель (Ctrl+C — отключить):",
		DE: "\nLive-Dashboard (Strg+C zum Trennen):",
		ZH: "\n实时面板（Ctrl+C 断开）：",
		JA: "\nライブダッシュボード (Ctrl+C で切断):",
		KO: "\n실시간 대시보드 (Ctrl+C로 연결 해제):",
	},
	"cli.connect.egress_lost": {
		EN: "⟳ Egress lost (%d checks). Reconnecting %s...",
		RU: "⟳ Выход потерян (%d проверок). Переподключение %s...",
		DE: "⟳ Ausgang verloren (%d Prüfungen). Wiederverbindung %s...",
		ZH: "⟳ 出口丢失（%d 次检查）。正在重连 %s...",
		JA: "⟳ 出口が失われました (%d 回チェック)。%s を再接続中...",
		KO: "⟳ 이그레스 손실 (%d회 확인). %s 재연결 중...",
	},
	"cli.connect.reconnected": {
		EN: "✔ Reconnected. egress=%s",
		RU: "✔ Переподключено. выход=%s",
		DE: "✔ Wiederverbunden. Ausgang=%s",
		ZH: "✔ 已重连。出口=%s",
		JA: "✔ 再接続しました。出口=%s",
		KO: "✔ 재연결됨. 이그레스=%s",
	},
	"cli.connect.disconnecting": {
		EN: "\nDisconnecting...",
		RU: "\nОтключение...",
		DE: "\nTrenne...",
		ZH: "\n正在断开...",
		JA: "\n切断中...",
		KO: "\n연결 해제 중...",
	},
	"cli.connect.disconnected": {
		EN: "Disconnected.",
		RU: "Отключено.",
		DE: "Getrennt.",
		ZH: "已断开。",
		JA: "切断しました。",
		KO: "연결 해제됨.",
	},

	// --- disconnect / recover ---
	"cli.disconnect.none": {
		EN: "No active Mazzy VPN interface. Nothing to disconnect.",
		RU: "Нет активного интерфейса Mazzy VPN. Отключать нечего.",
		DE: "Keine aktive Mazzy-VPN-Schnittstelle. Nichts zu trennen.",
		ZH: "没有活动的 Mazzy VPN 接口。无需断开。",
		JA: "アクティブな Mazzy VPN インターフェースがありません。切断するものはありません。",
		KO: "활성 Mazzy VPN 인터페이스가 없습니다. 연결 해제할 항목이 없습니다.",
	},
	"cli.disconnect.ok": {
		EN: "✔ Disconnected (%s removed).",
		RU: "✔ Отключено (%s удалён).",
		DE: "✔ Getrennt (%s entfernt).",
		ZH: "✔ 已断开（已移除 %s）。",
		JA: "✔ 切断しました (%s を削除)。",
		KO: "✔ 연결 해제됨 (%s 제거).",
	},
	"cli.recover.restoring": {
		EN: "Restoring a clean network state...",
		RU: "Восстановление чистого состояния сети...",
		DE: "Sauberen Netzwerkzustand wiederherstellen...",
		ZH: "正在恢复干净的网络状态...",
		JA: "クリーンなネットワーク状態を復元中...",
		KO: "깨끗한 네트워크 상태 복원 중...",
	},
	"cli.recover.done": {
		EN: "\n✔ Clean state restored (%d actions). You are on plain Wi‑Fi/uplink now.",
		RU: "\n✔ Чистое состояние восстановлено (%d действий). Вы на обычном Wi‑Fi/аплинке.",
		DE: "\n✔ Sauberer Zustand wiederhergestellt (%d Aktionen). Sie sind jetzt auf normalem WLAN/Uplink.",
		ZH: "\n✔ 已恢复干净状态（%d 项操作）。您现在使用普通 Wi‑Fi/上行。",
		JA: "\n✔ クリーンな状態を復元しました (%d 操作)。通常の Wi‑Fi/アップリンクに戻りました。",
		KO: "\n✔ 깨끗한 상태 복원됨 (%d개 작업). 이제 일반 Wi‑Fi/업링크입니다.",
	},

	// --- status ---
	"cli.status.protected": {
		EN: "✔ PROTECTED",
		RU: "✔ ЗАЩИЩЕНО",
		DE: "✔ GESCHÜTZT",
		ZH: "✔ 受保护",
		JA: "✔ 保護済み",
		KO: "✔ 보호됨",
	},
	"cli.status.disconnected": {
		EN: "✖ DISCONNECTED",
		RU: "✖ ОТКЛЮЧЕНО",
		DE: "✖ GETRENNT",
		ZH: "✖ 已断开",
		JA: "✖ 切断済み",
		KO: "✖ 연결 해제됨",
	},
	"cli.status.state": {
		EN: "State:", RU: "Состояние:", DE: "Zustand:",
		ZH: "状态：", JA: "状態:", KO: "상태:",
	},
	"cli.status.interface": {
		EN: "Interface:", RU: "Интерфейс:", DE: "Schnittstelle:",
		ZH: "接口：", JA: "インターフェース:", KO: "인터페이스:",
	},
	"cli.status.egress": {
		EN: "Egress IP:", RU: "Выходной IP:", DE: "Ausgangs-IP:",
		ZH: "出口 IP：", JA: "出口 IP:", KO: "이그레스 IP:",
	},
	"cli.status.profile": {
		EN: "Profile:", RU: "Профиль:", DE: "Profil:",
		ZH: "配置：", JA: "プロファイル:", KO: "프로필:",
	},
	"cli.status.linkup": {
		EN: "⚠ LINK UP (egress not confirmed: %s)",
		RU: "⚠ ЛИНК ПОДНЯТ (выход не подтверждён: %s)",
		DE: "⚠ VERBINDUNG AKTIV (Ausgang nicht bestätigt: %s)",
		ZH: "⚠ 链路已启用（出口未确认：%s）",
		JA: "⚠ リンクアップ (出口未確認: %s)",
		KO: "⚠ 링크 활성 (이그레스 미확인: %s)",
	},
	"cli.status.down": {
		EN: "✖ down (no active VPN interface)",
		RU: "✖ выключено (нет активного VPN-интерфейса)",
		DE: "✖ aus (keine aktive VPN-Schnittstelle)",
		ZH: "✖ 关闭（无活动 VPN 接口）",
		JA: "✖ 停止 (アクティブな VPN インターフェースなし)",
		KO: "✖ 꺼짐 (활성 VPN 인터페이스 없음)",
	},
	"cli.status.last": {
		EN: "Last:", RU: "Последний:", DE: "Zuletzt:",
		ZH: "最近：", JA: "最後:", KO: "마지막:",
	},

	// --- up / best / auto ---
	"cli.up.no_profiles": {
		EN: "no managed profiles; import first: mazzy-vpn import <DIR>",
		RU: "нет управляемых профилей; сначала импорт: mazzy-vpn import <DIR>",
		DE: "keine verwalteten Profile; zuerst importieren: mazzy-vpn import <DIR>",
		ZH: "没有受管配置；请先导入：mazzy-vpn import <DIR>",
		JA: "管理プロファイルなし。先にインポート: mazzy-vpn import <DIR>",
		KO: "관리되는 프로필 없음; 먼저 가져오기: mazzy-vpn import <DIR>",
	},
	"cli.up.selecting_best": {
		EN: "No profile named; selecting the best reachable zone...",
		RU: "Имя профиля не указано; выбираю лучшую доступную зону...",
		DE: "Kein Profilname; wähle die beste erreichbare Zone...",
		ZH: "未指定配置；正在选择最佳可达区域...",
		JA: "プロファイル名なし。最適な到達可能ゾーンを選択中...",
		KO: "프로필 이름 없음; 가장 좋은 도달 가능 존 선택 중...",
	},
	"cli.up.no_reachable": {
		EN: "no reachable server found; check your connection (mazzy-vpn netdiag)",
		RU: "доступных серверов нет; проверьте соединение (mazzy-vpn netdiag)",
		DE: "kein erreichbarer Server; prüfen Sie die Verbindung (mazzy-vpn netdiag)",
		ZH: "未找到可达服务器；请检查连接（mazzy-vpn netdiag）",
		JA: "到達可能なサーバーなし。接続を確認 (mazzy-vpn netdiag)",
		KO: "도달 가능한 서버 없음; 연결을 확인하세요 (mazzy-vpn netdiag)",
	},
	"cli.up.cleanest": {
		EN: "Cleanest live zone: %s",
		RU: "Самая чистая живая зона: %s",
		DE: "Sauberste Live-Zone: %s",
		ZH: "最干净的活动区域：%s",
		JA: "最もクリーンなライブゾーン: %s",
		KO: "가장 깨끗한 라이브 존: %s",
	},
	"cli.up.best_alive": {
		EN: "Best zone: %s (%d ms, ✔ alive)",
		RU: "Лучшая зона: %s (%d мс, ✔ жива)",
		DE: "Beste Zone: %s (%d ms, ✔ aktiv)",
		ZH: "最佳区域：%s（%d 毫秒，✔ 在线）",
		JA: "最適ゾーン: %s (%d ms、✔ 稼働中)",
		KO: "최적 존: %s (%d ms, ✔ 활성)",
	},
	"cli.up.best_noicmp": {
		EN: "Best zone: %s (no ICMP reply; may still work)",
		RU: "Лучшая зона: %s (нет ICMP-ответа; может работать)",
		DE: "Beste Zone: %s (keine ICMP-Antwort; kann trotzdem funktionieren)",
		ZH: "最佳区域：%s（无 ICMP 回应；仍可能可用）",
		JA: "最適ゾーン: %s (ICMP 応答なし。動作する場合あり)",
		KO: "최적 존: %s (ICMP 응답 없음; 작동할 수 있음)",
	},
	"cli.up.ranking": {
		EN: "Ranking zones by reachability and latency...",
		RU: "Ранжирование зон по доступности и задержке...",
		DE: "Zonen nach Erreichbarkeit und Latenz sortieren...",
		ZH: "按可达性和延迟对区域排序...",
		JA: "到達性とレイテンシでゾーンをランキング中...",
		KO: "도달성과 지연 시간으로 존 순위 지정 중...",
	},
	"cli.up.auto_connecting": {
		EN: "Auto-connecting to best zone: %s",
		RU: "Автоподключение к лучшей зоне: %s",
		DE: "Automatische Verbindung zur besten Zone: %s",
		ZH: "正在自动连接到最佳区域：%s",
		JA: "最適ゾーンに自動接続中: %s",
		KO: "최적 존에 자동 연결 중: %s",
	},

	// --- usage / help (section headers; command syntax stays literal) ---
	"cli.usage.tagline": {
		EN: "autonomous AI-ready VPN client (Go)",
		RU: "автономный AI-ready VPN-клиент (Go)",
		DE: "autonomer, KI-tauglicher VPN-Client (Go)",
		ZH: "自主的 AI-ready VPN 客户端（Go）",
		JA: "自律的な AI-ready VPN クライアント (Go)",
		KO: "자율적인 AI-ready VPN 클라이언트 (Go)",
	},
	"cli.usage.sec.profiles": {
		EN: "Profiles (managed catalog):", RU: "Профили (управляемый каталог):",
		DE: "Profile (verwalteter Katalog):", ZH: "配置（管理目录）：",
		JA: "プロファイル（管理カタログ）:", KO: "프로필(관리 카탈로그):",
	},
	"cli.usage.sec.connect": {
		EN: "Connect:", RU: "Подключение:", DE: "Verbinden:",
		ZH: "连接：", JA: "接続:", KO: "연결:",
	},
	"cli.usage.sec.background": {
		EN: "Permanent (background) operation:", RU: "Постоянная (фоновая) работа:",
		DE: "Dauerhafter (Hintergrund-)Betrieb:", ZH: "持久（后台）运行：",
		JA: "常駐（バックグラウンド）動作:", KO: "상시(백그라운드) 작동:",
	},
	"cli.usage.sec.network": {
		EN: "Network:", RU: "Сеть:", DE: "Netzwerk:",
		ZH: "网络：", JA: "ネットワーク:", KO: "네트워크:",
	},
	"cli.usage.sec.recovery": {
		EN: "Recovery:", RU: "Восстановление:", DE: "Wiederherstellung:",
		ZH: "恢复：", JA: "復旧:", KO: "복구:",
	},
	"cli.usage.sec.diagnostics": {
		EN: "Diagnostics:", RU: "Диагностика:", DE: "Diagnose:",
		ZH: "诊断：", JA: "診断:", KO: "진단:",
	},
	"cli.usage.footer": {
		EN: "connect/up run in the foreground and hold the tunnel until Ctrl+C.",
		RU: "connect/up работают на переднем плане и держат туннель до Ctrl+C.",
		DE: "connect/up laufen im Vordergrund und halten den Tunnel bis Strg+C.",
		ZH: "connect/up 在前台运行，保持隧道直到 Ctrl+C。",
		JA: "connect/up はフォアグラウンドで実行され、Ctrl+C までトンネルを保持します。",
		KO: "connect/up는 포그라운드에서 실행되며 Ctrl+C까지 터널을 유지합니다.",
	},

	// --- providers ---
	"cli.providers.checking": {
		EN: "Checking %d AI provider(s) from the current egress...",
		RU: "Проверка %d AI-провайдеров через текущий выход...",
		DE: "Prüfe %d KI-Anbieter vom aktuellen Ausgang...",
		ZH: "正在从当前出口检查 %d 个 AI 提供商...",
		JA: "現在の出口から %d 個の AI プロバイダーを確認中...",
		KO: "현재 이그레스에서 %d개 AI 제공업체 확인 중...",
	},
	"cli.providers.summary": {
		EN: "%d/%d AI providers reachable from here.",
		RU: "%d/%d AI-провайдеров доступны отсюда.",
		DE: "%d/%d KI-Anbieter von hier erreichbar.",
		ZH: "从此处可达 %d/%d 个 AI 提供商。",
		JA: "ここから %d/%d の AI プロバイダーに到達可能。",
		KO: "여기서 %d/%d AI 제공업체에 도달 가능.",
	},

	// --- analysis command banners ---
	"cli.stealth.analyzing": {
		EN: "Analyzing detection vectors (IPv6/DNS/timezone/ASN/Cloudflare)...",
		RU: "Анализ векторов детектирования (IPv6/DNS/таймзона/ASN/Cloudflare)...",
		DE: "Analysiere Erkennungsvektoren (IPv6/DNS/Zeitzone/ASN/Cloudflare)...",
		ZH: "正在分析检测向量（IPv6/DNS/时区/ASN/Cloudflare）...",
		JA: "検出ベクトルを分析中 (IPv6/DNS/タイムゾーン/ASN/Cloudflare)...",
		KO: "탐지 벡터 분석 중 (IPv6/DNS/시간대/ASN/Cloudflare)...",
	},
	"cli.stealth.aligned": {
		EN: "Already aligned. Nothing to do.",
		RU: "Уже выровнено. Делать нечего.",
		DE: "Bereits ausgerichtet. Nichts zu tun.",
		ZH: "已对齐。无需操作。",
		JA: "すでに整合しています。何もすることはありません。",
		KO: "이미 정렬됨. 작업할 내용 없음.",
	},
	"cli.diagnose.analyzing": {
		EN: "Analyzing connection (uplink, tunnel, servers, DNS, egress)...",
		RU: "Анализ соединения (аплинк, туннель, серверы, DNS, выход)...",
		DE: "Analysiere Verbindung (Uplink, Tunnel, Server, DNS, Ausgang)...",
		ZH: "正在分析连接（上行、隧道、服务器、DNS、出口）...",
		JA: "接続を分析中（アップリンク、トンネル、サーバー、DNS、出口）...",
		KO: "연결 분석 중(업링크, 터널, 서버, DNS, 이그레스)...",
	},
	"cli.test.no_profiles": {
		EN: "No profiles with endpoints to test. Import some first.",
		RU: "Нет профилей с endpoint для теста. Сначала импортируйте.",
		DE: "Keine Profile mit Endpunkten zum Testen. Zuerst importieren.",
		ZH: "没有带 endpoint 的配置可测试。请先导入。",
		JA: "テストする endpoint 付きプロファイルなし。先にインポートしてください。",
		KO: "테스트할 endpoint 프로필 없음. 먼저 가져오세요.",
	},
	"cli.dnscheck.dot_needs_root": {
		EN: "enabling DoT needs root: sudo mazzy-vpn dns-check --dot",
		RU: "включение DoT требует root: sudo mazzy-vpn dns-check --dot",
		DE: "DoT aktivieren benötigt root: sudo mazzy-vpn dns-check --dot",
		ZH: "启用 DoT 需要 root：sudo mazzy-vpn dns-check --dot",
		JA: "DoT の有効化には root が必要: sudo mazzy-vpn dns-check --dot",
		KO: "DoT 활성화는 root가 필요: sudo mazzy-vpn dns-check --dot",
	},
}

func init() {
	for k, v := range cliCatalog {
		catalog[k] = v
	}
}
