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
}

func init() {
	for k, v := range cliCatalog {
		catalog[k] = v
	}
}
