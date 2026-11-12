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
	PanelTab, LogHint                             string
}

var phrases = map[i18n.Lang]Text{
	i18n.EN: {
		Connect: "Connect", Disconnect: "Disconnect", Status: "Status", Server: "Server", Protocol: "Protocol",
		Disconnected: "Disconnected", Connecting: "Connecting", Connected: "Connected", Failed: "Error",
		Profiles: "Profiles", Empty: "No profiles yet. Import one with mazzy-vpn import.",
		RootHint:  "This window stays unprivileged. Connect runs mazzy-vpn up through pkexec, or through sudo when there is no PolicyKit agent.",
		Corporate: "AmneziaWG and WireGuard speak UDP. If the configured port does not handshake, Mazzy also tries UDP 443, 51820, 123, 53 and 1194. AdGuard VPN CLI can fall back to HTTP/2 on TCP/443; this app cannot.",
		Detail:    "Path",
		Link:      "Connection", Handshake: "Handshake", Route: "Path", Unknown: "unknown",
		Chart: "Probe", Output: "Output", NoData: "no measurements yet", ImportHint: "path for import or list",
		Filter: "Filter servers", Commands: "Commands", EgressIP: "Egress IP", Clean: "Cleanliness", Service: "Antigravity",
		Journal: "Log", Search: "search", LogEmpty: "log is empty", PathHint: "path",
		Ready: "Pick a server, then Connect.", Busy: "Connecting. The CLI answer opens the log.",
		UpNote: "Tunnel is up. Checks refresh on their own.", Fail: "Connect failed. The log has the CLI answer.",
		InCatalog: "in the catalog",
		PanelTab:  "Panel", LogHint: "The log fills in after a command or a connection.",
	},
	i18n.RU: {
		Connect: "Подключить", Disconnect: "Отключить", Status: "Статус", Server: "Сервер", Protocol: "Протокол",
		Disconnected: "Отключено", Connecting: "Подключение", Connected: "Подключено", Failed: "Ошибка",
		Profiles: "Профили", Empty: "Профилей нет. Импорт: mazzy-vpn import.",
		RootHint:  "Окно не требует root. Подключение запускает mazzy-vpn up через pkexec, а без агента PolicyKit — через sudo.",
		Corporate: "AmneziaWG и WireGuard работают по UDP. Если заданный порт молчит, Mazzy пробует UDP 443, 51820, 123, 53 и 1194. AdGuard VPN CLI умеет HTTP/2 на TCP/443; это приложение — нет.",
		Detail:    "Путь",
		Link:      "Связь", Handshake: "Рукопожатие", Route: "Путь", Unknown: "неизвестно",
		Chart: "Проверка", Output: "Вывод", NoData: "измерений ещё нет", ImportHint: "путь для импорта или списка",
		Filter: "Фильтр серверов", Commands: "Команды", EgressIP: "Внешний IP", Clean: "Чистота", Service: "Antigravity",
		Journal: "Журнал", Search: "поиск", LogEmpty: "журнал пуст", PathHint: "путь",
		Ready: "Выберите сервер слева и нажмите Подключить.", Busy: "Подключение. Ответ CLI откроет журнал.",
		UpNote: "Туннель поднят. Проверки обновляются сами.", Fail: "Подключение не удалось. Ответ CLI в журнале.",
		InCatalog: "в каталоге",
		PanelTab:  "Панель", LogHint: "Журнал появится после команды или подключения.",
	},
	i18n.DE: {
		Connect: "Verbinden", Disconnect: "Trennen", Status: "Status", Server: "Server", Protocol: "Protokoll",
		Disconnected: "Getrennt", Connecting: "Verbinde", Connected: "Verbunden", Failed: "Fehler",
		Profiles: "Profile", Empty: "Keine Profile. Import mit mazzy-vpn import.",
		RootHint:  "Dieses Fenster bleibt ohne root. Verbinden startet mazzy-vpn up über pkexec, ohne PolicyKit-Agent über sudo.",
		Corporate: "AmneziaWG und WireGuard sprechen UDP. Ohne Handshake probiert Mazzy UDP 443, 51820, 123, 53 und 1194. AdGuard VPN CLI kann auf HTTP/2 über TCP/443 ausweichen; diese App nicht.",
		Detail:    "Pfad",
		Link:      "Verbindung", Handshake: "Handshake", Route: "Pfad", Unknown: "unbekannt",
		Chart: "Messung", Output: "Ausgabe", NoData: "noch keine Messwerte", ImportHint: "Pfad für Import oder Liste",
		Filter: "Server filtern", Commands: "Befehle", EgressIP: "Ausgangs-IP", Clean: "Sauberkeit", Service: "Antigravity",
		Journal: "Journal", Search: "suchen", LogEmpty: "Journal ist leer", PathHint: "Pfad",
		Ready: "Server wählen, dann Verbinden.", Busy: "Verbindung. Die CLI-Antwort öffnet das Journal.",
		UpNote: "Tunnel steht. Prüfungen laufen von selbst.", Fail: "Verbindung fehlgeschlagen. Die CLI-Antwort steht im Journal.",
		InCatalog: "im Katalog",
		PanelTab:  "Panel", LogHint: "Das Journal füllt sich nach einem Befehl oder einer Verbindung.",
	},
	i18n.ZH: {
		Connect: "连接", Disconnect: "断开", Status: "状态", Server: "服务器", Protocol: "协议",
		Disconnected: "未连接", Connecting: "正在连接", Connected: "已连接", Failed: "错误",
		Profiles: "配置", Empty: "还没有配置。用 mazzy-vpn import 导入。",
		RootHint:  "此窗口不需要 root。连接通过 pkexec 运行 mazzy-vpn up；没有 PolicyKit 时改用 sudo。",
		Corporate: "AmneziaWG 和 WireGuard 只走 UDP。配置端口没有握手时，Mazzy 会再试 UDP 443、51820、123、53 和 1194。AdGuard VPN CLI 可以退到 TCP/443 上的 HTTP/2；本程序不能。",
		Detail:    "路径",
		Link:      "连接", Handshake: "握手", Route: "路径", Unknown: "未知",
		Chart: "探测", Output: "输出", NoData: "还没有测量", ImportHint: "导入或列出的路径",
		Filter: "筛选服务器", Commands: "命令", EgressIP: "出口 IP", Clean: "纯净度", Service: "Antigravity",
		Journal: "日志", Search: "搜索", LogEmpty: "日志为空", PathHint: "路径",
		Ready: "先选服务器，再点连接。", Busy: "正在连接。CLI 的回复会打开日志。",
		UpNote: "隧道已建立。检查会自己更新。", Fail: "连接失败。CLI 的回复在日志里。",
		InCatalog: "在目录中",
		PanelTab:  "面板", LogHint: "日志会在命令或连接之后出现。",
	},
	i18n.JA: {
		Connect: "接続", Disconnect: "切断", Status: "状態", Server: "サーバー", Protocol: "プロトコル",
		Disconnected: "未接続", Connecting: "接続中", Connected: "接続済み", Failed: "エラー",
		Profiles: "プロファイル", Empty: "プロファイルがありません。mazzy-vpn import で追加します。",
		RootHint:  "このウィンドウ自体は root 不要です。接続は pkexec で mazzy-vpn up を実行し、PolicyKit が無いときは sudo を使います。",
		Corporate: "AmneziaWG と WireGuard は UDP です。設定ポートで握手できないとき、Mazzy は UDP 443、51820、123、53、1194 を試します。AdGuard VPN CLI は TCP/443 の HTTP/2 に戻れますが、このアプリはできません。",
		Detail:    "経路",
		Link:      "接続", Handshake: "ハンドシェイク", Route: "経路", Unknown: "不明",
		Chart: "計測", Output: "出力", NoData: "計測はまだありません", ImportHint: "インポートまたは一覧のパス",
		Filter: "サーバーを絞る", Commands: "コマンド", EgressIP: "出口 IP", Clean: "清潔さ", Service: "Antigravity",
		Journal: "ログ", Search: "検索", LogEmpty: "ログは空です", PathHint: "パス",
		Ready: "サーバーを選び、接続を押します。", Busy: "接続中。CLI の応答でログが開きます。",
		UpNote: "トンネルは上がっています。検査は自分で更新します。", Fail: "接続に失敗しました。CLI の応答はログにあります。",
		InCatalog: "カタログ内",
		PanelTab:  "パネル", LogHint: "ログはコマンドか接続のあとに入ります。",
	},
	i18n.KO: {
		Connect: "연결", Disconnect: "연결 해제", Status: "상태", Server: "서버", Protocol: "프로토콜",
		Disconnected: "연결 끊김", Connecting: "연결 중", Connected: "연결됨", Failed: "오류",
		Profiles: "프로필", Empty: "프로필이 없습니다. mazzy-vpn import 로 가져오세요.",
		RootHint:  "이 창은 root 로 실행하지 않습니다. 연결은 pkexec 로 mazzy-vpn up 을 실행하고, PolicyKit 이 없으면 sudo 를 사용합니다.",
		Corporate: "AmneziaWG 와 WireGuard 는 UDP 입니다. 설정한 포트가 응답하지 않으면 UDP 443, 51820, 123, 53, 1194 를 시도합니다. AdGuard VPN CLI 는 TCP/443 HTTP/2 로 바꿀 수 있지만 이 앱은 그럴 수 없습니다.",
		Detail:    "경로",
		Link:      "연결", Handshake: "핸드셰이크", Route: "경로", Unknown: "알 수 없음",
		Chart: "측정", Output: "출력", NoData: "측정값이 없습니다", ImportHint: "가져오기 또는 목록 경로",
		Filter: "서버 필터", Commands: "명령", EgressIP: "외부 IP", Clean: "청정", Service: "Antigravity",
		Journal: "로그", Search: "검색", LogEmpty: "로그가 비어 있습니다", PathHint: "경로",
		Ready: "서버를 고른 다음 연결을 누르세요.", Busy: "연결 중. CLI 응답이 로그를 엽니다.",
		UpNote: "터널이 올라왔습니다. 검사는 스스로 갱신됩니다.", Fail: "연결에 실패했습니다. CLI 응답은 로그에 있습니다.",
		InCatalog: "카탈로그",
		PanelTab:  "패널", LogHint: "로그는 명령이나 연결 뒤에 채워집니다.",
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
