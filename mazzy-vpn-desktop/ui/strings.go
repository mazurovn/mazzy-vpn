// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package ui

import (
	"os"

	"github.com/mazurovn/mazzy-vpn/core/i18n"
)

// Text is one desktop string in the six product languages.
type Text struct {
	Connect, Disconnect, Status, Server, Protocol string
	Disconnected, Connecting, Connected, Failed   string
	Profiles, Empty, RootHint, Corporate          string
	Detail                                        string
}

var phrases = map[i18n.Lang]Text{
	i18n.EN: {
		Connect: "Connect", Disconnect: "Disconnect", Status: "Status", Server: "Server", Protocol: "Protocol",
		Disconnected: "Disconnected", Connecting: "Connecting", Connected: "Connected", Failed: "Error",
		Profiles: "Profiles", Empty: "No profiles yet. Import one with mazzy-vpn import.",
		RootHint:  "Connect and disconnect need root. Start this window with sudo.",
		Corporate: "AmneziaWG and WireGuard speak UDP. If the configured port does not handshake, Mazzy also tries UDP 443, 51820, 123, 53 and 1194. AdGuard VPN CLI can fall back to HTTP/2 on TCP/443; this app cannot.",
		Detail:    "Path",
	},
	i18n.RU: {
		Connect: "Подключить", Disconnect: "Отключить", Status: "Статус", Server: "Сервер", Protocol: "Протокол",
		Disconnected: "Отключено", Connecting: "Подключение", Connected: "Подключено", Failed: "Ошибка",
		Profiles: "Профили", Empty: "Профилей нет. Импорт: mazzy-vpn import.",
		RootHint:  "Подключение требует root. Запустите окно через sudo.",
		Corporate: "AmneziaWG и WireGuard работают по UDP. Если заданный порт молчит, Mazzy пробует UDP 443, 51820, 123, 53 и 1194. AdGuard VPN CLI умеет HTTP/2 на TCP/443; это приложение — нет.",
		Detail:    "Путь",
	},
	i18n.DE: {
		Connect: "Verbinden", Disconnect: "Trennen", Status: "Status", Server: "Server", Protocol: "Protokoll",
		Disconnected: "Getrennt", Connecting: "Verbinde", Connected: "Verbunden", Failed: "Fehler",
		Profiles: "Profile", Empty: "Keine Profile. Import mit mazzy-vpn import.",
		RootHint:  "Verbinden braucht root. Fenster mit sudo starten.",
		Corporate: "AmneziaWG und WireGuard sprechen UDP. Ohne Handshake probiert Mazzy UDP 443, 51820, 123, 53 und 1194. AdGuard VPN CLI kann auf HTTP/2 über TCP/443 ausweichen; diese App nicht.",
		Detail:    "Pfad",
	},
	i18n.ZH: {
		Connect: "连接", Disconnect: "断开", Status: "状态", Server: "服务器", Protocol: "协议",
		Disconnected: "未连接", Connecting: "正在连接", Connected: "已连接", Failed: "错误",
		Profiles: "配置", Empty: "还没有配置。用 mazzy-vpn import 导入。",
		RootHint:  "连接需要 root。请用 sudo 启动此窗口。",
		Corporate: "AmneziaWG 和 WireGuard 只走 UDP。配置端口没有握手时，Mazzy 会再试 UDP 443、51820、123、53 和 1194。AdGuard VPN CLI 可以退到 TCP/443 上的 HTTP/2；本程序不能。",
		Detail:    "路径",
	},
	i18n.JA: {
		Connect: "接続", Disconnect: "切断", Status: "状態", Server: "サーバー", Protocol: "プロトコル",
		Disconnected: "未接続", Connecting: "接続中", Connected: "接続済み", Failed: "エラー",
		Profiles: "プロファイル", Empty: "プロファイルがありません。mazzy-vpn import で追加します。",
		RootHint:  "接続には root が必要です。sudo で起動してください。",
		Corporate: "AmneziaWG と WireGuard は UDP です。設定ポートで握手できないとき、Mazzy は UDP 443、51820、123、53、1194 を試します。AdGuard VPN CLI は TCP/443 の HTTP/2 に戻れますが、このアプリはできません。",
		Detail:    "経路",
	},
	i18n.KO: {
		Connect: "연결", Disconnect: "연결 해제", Status: "상태", Server: "서버", Protocol: "프로토콜",
		Disconnected: "연결 끊김", Connecting: "연결 중", Connected: "연결됨", Failed: "오류",
		Profiles: "프로필", Empty: "프로필이 없습니다. mazzy-vpn import 로 가져오세요.",
		RootHint:  "연결에는 root 가 필요합니다. sudo 로 이 창을 실행하세요.",
		Corporate: "AmneziaWG 와 WireGuard 는 UDP 입니다. 설정한 포트가 응답하지 않으면 UDP 443, 51820, 123, 53, 1194 를 시도합니다. AdGuard VPN CLI 는 TCP/443 HTTP/2 로 바꿀 수 있지만 이 앱은 그럴 수 없습니다.",
		Detail:    "경로",
	},
}

// Phrases resolves the desktop copy from MAZZY_LANG and the process locale.
func Phrases() Text {
	lang := i18n.Resolve(os.Getenv("MAZZY_LANG"), os.Getenv("LC_ALL"), os.Getenv("LC_MESSAGES"), os.Getenv("LANG"))
	if t, ok := phrases[lang]; ok {
		return t
	}
	return phrases[i18n.EN]
}
