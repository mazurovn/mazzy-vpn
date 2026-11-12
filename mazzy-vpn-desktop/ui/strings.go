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
	Link, Handshake, Route, Unknown               string
	Chart, Output, NoData, ImportHint             string
	Filter, Commands, EgressIP, Clean, Service    string
	Journal, Search, LogEmpty, PathHint           string
	Ready, Busy, UpNote, Fail, InCatalog          string
	PanelTab, LogHint, Refresh, Exit              string
	DashDown, DashUp, Summary                     string
}

var phrases = map[i18n.Lang]Text{
	i18n.EN: {
		Connect: "Connect", Disconnect: "Disconnect", Status: "Status", Server: "Server", Protocol: "Protocol",
		Disconnected: "Disconnected", Connecting: "Connecting", Connected: "Connected", Failed: "Error",
		Profiles: "Profiles", Empty: "No profiles yet. Import one with mazzy-vpn import.",
		RootHint:  "Connect, reconnect, and disconnect use the local API socket as the current user.",
		Corporate: "AmneziaWG and WireGuard speak UDP. If the configured port does not handshake, Mazzy also tries UDP 443, 51820, 123, 53 and 1194. AdGuard VPN CLI can fall back to HTTP/2 on TCP/443; this app cannot.",
		Detail:    "Path",
		Link:      "Connection", Handshake: "Handshake", Route: "Path", Unknown: "unknown",
		Chart: "Probe", Output: "Output", NoData: "no measurements yet", ImportHint: "path for import or list",
		Filter: "Filter servers", Commands: "Commands", EgressIP: "Egress IP", Clean: "Cleanliness", Service: "Antigravity",
		Journal: "Log", Search: "search", LogEmpty: "log is empty", PathHint: "path",
		Ready: "Pick a server, then Connect.", Busy: "Connecting. The CLI answer opens the log.",
		UpNote: "Tunnel is up. Checks refresh on their own.", Fail: "Connect failed. The log has the CLI answer.",
		InCatalog: "in the catalog",
		PanelTab:  "Panel", LogHint: "The log fills in after a command.", Refresh: "Refresh", Exit: "Egress",
		DashDown: "The tunnel is down. No measurements.", DashUp: "The tunnel is up. No measurements.",
		Summary: "Summary",
	},
	i18n.RU: {
		Connect: "Подключить", Disconnect: "Отключить", Status: "Статус", Server: "Сервер", Protocol: "Протокол",
		Disconnected: "Отключено", Connecting: "Подключение", Connected: "Подключено", Failed: "Ошибка",
		Profiles: "Профили", Empty: "Профилей нет. Импорт: mazzy-vpn import.",
		RootHint:  "Подключение, переподключение и отключение идут в локальный сокет от текущего пользователя.",
		Corporate: "AmneziaWG и WireGuard работают по UDP. Если заданный порт молчит, Mazzy пробует UDP 443, 51820, 123, 53 и 1194. AdGuard VPN CLI умеет HTTP/2 на TCP/443; это приложение — нет.",
		Detail:    "Путь",
		Link:      "Связь", Handshake: "Рукопожатие", Route: "Путь", Unknown: "неизвестно",
		Chart: "Проверка", Output: "Вывод", NoData: "измерений ещё нет", ImportHint: "путь для импорта или списка",
		Filter: "Фильтр серверов", Commands: "Команды", EgressIP: "Внешний IP", Clean: "Чистота", Service: "Antigravity",
		Journal: "Журнал", Search: "поиск", LogEmpty: "журнал пуст", PathHint: "путь",
		Ready: "Выберите сервер слева и нажмите Подключить.", Busy: "Подключение. Ответ CLI откроет журнал.",
		UpNote: "Туннель поднят. Проверки обновляются сами.", Fail: "Подключение не удалось. Ответ CLI в журнале.",
		InCatalog: "в каталоге",
		PanelTab:  "Панель", LogHint: "Журнал появится после команды.", Refresh: "Обновить", Exit: "Выход",
		DashDown: "Туннель не поднят. Замеров нет.", DashUp: "Туннель поднят. Замеров нет.",
		Summary: "Сводка",
	},
	i18n.DE: {
		Connect: "Verbinden", Disconnect: "Trennen", Status: "Status", Server: "Server", Protocol: "Protokoll",
		Disconnected: "Getrennt", Connecting: "Verbinde", Connected: "Verbunden", Failed: "Fehler",
		Profiles: "Profile", Empty: "Keine Profile. Import mit mazzy-vpn import.",
		RootHint:  "Verbinden, erneut verbinden und trennen nutzen den lokalen API-Socket als aktueller Benutzer.",
		Corporate: "AmneziaWG und WireGuard sprechen UDP. Ohne Handshake probiert Mazzy UDP 443, 51820, 123, 53 und 1194. AdGuard VPN CLI kann auf HTTP/2 über TCP/443 ausweichen; diese App nicht.",
		Detail:    "Pfad",
		Link:      "Verbindung", Handshake: "Handshake", Route: "Pfad", Unknown: "unbekannt",
		Chart: "Messung", Output: "Ausgabe", NoData: "noch keine Messwerte", ImportHint: "Pfad für Import oder Liste",
		Filter: "Server filtern", Commands: "Befehle", EgressIP: "Ausgangs-IP", Clean: "Sauberkeit", Service: "Antigravity",
		Journal: "Journal", Search: "suchen", LogEmpty: "Journal ist leer", PathHint: "Pfad",
		Ready: "Server wählen, dann Verbinden.", Busy: "Verbindung. Die CLI-Antwort öffnet das Journal.",
		UpNote: "Tunnel steht. Prüfungen laufen von selbst.", Fail: "Verbindung fehlgeschlagen. Die CLI-Antwort steht im Journal.",
		InCatalog: "im Katalog",
		PanelTab:  "Panel", LogHint: "Das Journal füllt sich nach einem Befehl.", Refresh: "Aktualisieren", Exit: "Ausgang",
		DashDown: "Tunnel ist unten. Keine Messwerte.", DashUp: "Tunnel steht. Keine Messwerte.",
		Summary: "Überblick",
	},
	i18n.ZH: {
		Connect: "连接", Disconnect: "断开", Status: "状态", Server: "服务器", Protocol: "协议",
		Disconnected: "未连接", Connecting: "正在连接", Connected: "已连接", Failed: "错误",
		Profiles: "配置", Empty: "还没有配置。用 mazzy-vpn import 导入。",
		RootHint:  "连接、重连和断开使用本地 API 套接字，并以当前用户身份进行。",
		Corporate: "AmneziaWG 和 WireGuard 只走 UDP。配置端口没有握手时，Mazzy 会再试 UDP 443、51820、123、53 和 1194。AdGuard VPN CLI 可以退到 TCP/443 上的 HTTP/2；本程序不能。",
		Detail:    "路径",
		Link:      "连接", Handshake: "握手", Route: "路径", Unknown: "未知",
		Chart: "探测", Output: "输出", NoData: "还没有测量", ImportHint: "导入或列出的路径",
		Filter: "筛选服务器", Commands: "命令", EgressIP: "出口 IP", Clean: "纯净度", Service: "Antigravity",
		Journal: "日志", Search: "搜索", LogEmpty: "日志为空", PathHint: "路径",
		Ready: "先选服务器，再点连接。", Busy: "正在连接。CLI 的回复会打开日志。",
		UpNote: "隧道已建立。检查会自己更新。", Fail: "连接失败。CLI 的回复在日志里。",
		InCatalog: "在目录中",
		PanelTab:  "面板", LogHint: "日志会在命令之后出现。", Refresh: "刷新", Exit: "出口",
		DashDown: "隧道未建立。没有测量。", DashUp: "隧道已建立。没有测量。",
		Summary: "概览",
	},
	i18n.JA: {
		Connect: "接続", Disconnect: "切断", Status: "状態", Server: "サーバー", Protocol: "プロトコル",
		Disconnected: "未接続", Connecting: "接続中", Connected: "接続済み", Failed: "エラー",
		Profiles: "プロファイル", Empty: "プロファイルがありません。mazzy-vpn import で追加します。",
		RootHint:  "接続、再接続、切断はローカル API ソケットを現在のユーザーとして使います。",
		Corporate: "AmneziaWG と WireGuard は UDP です。設定ポートで握手できないとき、Mazzy は UDP 443、51820、123、53、1194 を試します。AdGuard VPN CLI は TCP/443 の HTTP/2 に戻れますが、このアプリはできません。",
		Detail:    "経路",
		Link:      "接続", Handshake: "ハンドシェイク", Route: "経路", Unknown: "不明",
		Chart: "計測", Output: "出力", NoData: "計測はまだありません", ImportHint: "インポートまたは一覧のパス",
		Filter: "サーバーを絞る", Commands: "コマンド", EgressIP: "出口 IP", Clean: "清潔さ", Service: "Antigravity",
		Journal: "ログ", Search: "検索", LogEmpty: "ログは空です", PathHint: "パス",
		Ready: "サーバーを選び、接続を押します。", Busy: "接続中。CLI の応答でログが開きます。",
		UpNote: "トンネルは上がっています。検査は自分で更新します。", Fail: "接続に失敗しました。CLI の応答はログにあります。",
		InCatalog: "カタログ内",
		PanelTab:  "パネル", LogHint: "ログはコマンドのあとに入ります。", Refresh: "更新", Exit: "出口",
		DashDown: "トンネルは下がっています。計測はありません。", DashUp: "トンネルは上がっています。計測はありません。",
		Summary: "概要",
	},
	i18n.KO: {
		Connect: "연결", Disconnect: "연결 해제", Status: "상태", Server: "서버", Protocol: "프로토콜",
		Disconnected: "연결 끊김", Connecting: "연결 중", Connected: "연결됨", Failed: "오류",
		Profiles: "프로필", Empty: "프로필이 없습니다. mazzy-vpn import 로 가져오세요.",
		RootHint:  "연결, 재연결, 끊기는 현재 사용자로 로컬 API 소켓을 사용합니다.",
		Corporate: "AmneziaWG 와 WireGuard 는 UDP 입니다. 설정한 포트가 응답하지 않으면 UDP 443, 51820, 123, 53, 1194 를 시도합니다. AdGuard VPN CLI 는 TCP/443 HTTP/2 로 바꿀 수 있지만 이 앱은 그럴 수 없습니다.",
		Detail:    "경로",
		Link:      "연결", Handshake: "핸드셰이크", Route: "경로", Unknown: "알 수 없음",
		Chart: "측정", Output: "출력", NoData: "측정값이 없습니다", ImportHint: "가져오기 또는 목록 경로",
		Filter: "서버 필터", Commands: "명령", EgressIP: "외부 IP", Clean: "청정", Service: "Antigravity",
		Journal: "로그", Search: "검색", LogEmpty: "로그가 비어 있습니다", PathHint: "경로",
		Ready: "서버를 고른 다음 연결을 누르세요.", Busy: "연결 중. CLI 응답이 로그를 엽니다.",
		UpNote: "터널이 올라왔습니다. 검사는 스스로 갱신됩니다.", Fail: "연결에 실패했습니다. CLI 응답은 로그에 있습니다.",
		InCatalog: "카탈로그",
		PanelTab:  "패널", LogHint: "로그는 명령 뒤에 채워집니다.", Refresh: "새로고침", Exit: "출구",
		DashDown: "터널이 내려가 있습니다. 측정값이 없습니다.", DashUp: "터널이 올라와 있습니다. 측정값이 없습니다.",
		Summary: "요약",
	},
}

// Phrases is the window copy. The command list is Russian, so the chrome is
// Russian too unless MAZZY_LANG names another catalog.
func Phrases() Text {
	if override := os.Getenv("MAZZY_LANG"); override != "" {
		if t, ok := phrases[i18n.Resolve(override)]; ok {
			return t
		}
	}
	return phrases[i18n.RU]
}
