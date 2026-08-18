// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package i18n

import "fmt"

// Catalog holds localized strings keyed by a stable message key. Missing
// translations fall back to English, then to the key itself, so the UI never
// shows an empty string.
//
// Keys mirror the machine-first API message_key namespace where relevant, so a
// single catalog serves CLI text, TUI menus and API-driven UIs.
var catalog = map[string]map[Lang]string{
	"menu.title": {
		RU: "Mazzy VPN — управление",
		EN: "Mazzy VPN — control",
		DE: "Mazzy VPN — Steuerung",
		ZH: "Mazzy VPN — 控制",
		JA: "Mazzy VPN — 制御",
		KO: "Mazzy VPN — 제어",
	},
	"menu.connect": {
		RU: "Подключить", EN: "Connect", DE: "Verbinden",
		ZH: "连接", JA: "接続", KO: "연결",
	},
	"menu.disconnect": {
		RU: "Отключить", EN: "Disconnect", DE: "Trennen",
		ZH: "断开", JA: "切断", KO: "연결 해제",
	},
	"menu.status": {
		RU: "Состояние", EN: "Status", DE: "Status",
		ZH: "状态", JA: "状態", KO: "상태",
	},
	"menu.verify": {
		RU: "Проверить выход", EN: "Verify egress", DE: "Ausgang prüfen",
		ZH: "验证出口", JA: "出口を検証", KO: "이그레스 확인",
	},
	"menu.doctor": {
		RU: "Диагностика", EN: "Doctor", DE: "Diagnose",
		ZH: "诊断", JA: "診断", KO: "진단",
	},
	"menu.language": {
		RU: "Язык", EN: "Language", DE: "Sprache",
		ZH: "语言", JA: "言語", KO: "언어",
	},
	"menu.quit": {
		RU: "Выход", EN: "Quit", DE: "Beenden",
		ZH: "退出", JA: "終了", KO: "종료",
	},
	"prompt.choose": {
		RU: "Выберите действие: ", EN: "Choose an action: ",
		DE: "Aktion wählen: ", ZH: "选择操作：",
		JA: "操作を選択: ", KO: "작업 선택: ",
	},
	// verify.* keys mirror core/verify message keys so verdicts localize.
	"verify.verified": {
		RU: "Защищённый выход подтверждён", EN: "Protected egress verified",
		DE: "Geschützter Ausgang bestätigt", ZH: "已验证受保护出口",
		JA: "保護された出口を確認", KO: "보호된 이그레스 확인됨",
	},
	"verify.failed.inactive": {
		RU: "Туннель не активен", EN: "Tunnel is not active",
		DE: "Tunnel ist nicht aktiv", ZH: "隧道未激活",
		JA: "トンネルが非アクティブ", KO: "터널이 비활성 상태",
	},
	"verify.failed.no-egress": {
		RU: "Нет выхода через туннель", EN: "No egress through the tunnel",
		DE: "Kein Ausgang durch den Tunnel", ZH: "隧道无出口",
		JA: "トンネル経由の出口なし", KO: "터널을 통한 이그레스 없음",
	},
	"verify.warning.route-mismatch": {
		RU: "Маршрут не через туннель", EN: "Route does not go through the tunnel",
		DE: "Route läuft nicht durch den Tunnel", ZH: "路由未经过隧道",
		JA: "ルートがトンネルを経由していない", KO: "경로가 터널을 통하지 않음",
	},
	"verify.warning.ipv6-leak": {
		RU: "Возможна утечка IPv6", EN: "Possible IPv6 leak",
		DE: "Möglicher IPv6-Leak", ZH: "可能存在 IPv6 泄漏",
		JA: "IPv6リークの可能性", KO: "IPv6 유출 가능성",
	},
	"verify.warning.country-mismatch": {
		RU: "Страна выхода не совпадает", EN: "Egress country does not match",
		DE: "Ausgangsland stimmt nicht überein", ZH: "出口国家/地区不匹配",
		JA: "出口の国が一致しない", KO: "이그레스 국가 불일치",
	},
	"verify.warning.dns-leak": {
		RU: "DNS идёт не через VPN", EN: "DNS does not go through the VPN",
		DE: "DNS läuft nicht über das VPN", ZH: "DNS 未经过 VPN",
		JA: "DNSがVPNを経由していない", KO: "DNS가 VPN을 통하지 않음",
	},
}

// verifyMessageKeys are the core/verify message keys that MUST be present in
// the catalog (enforced by TestVerifyKeysLocalized) so verdicts always render.
var verifyMessageKeys = []string{
	"verify.verified",
	"verify.failed.inactive",
	"verify.failed.no-egress",
	"verify.warning.route-mismatch",
	"verify.warning.ipv6-leak",
	"verify.warning.country-mismatch",
	"verify.warning.dns-leak",
}

// Translator renders messages for a fixed language.
type Translator struct{ Lang Lang }

// NewTranslator builds a Translator for lang (invalid falls back to default).
func NewTranslator(lang Lang) *Translator {
	if !lang.Valid() {
		lang = DefaultLang
	}
	return &Translator{Lang: lang}
}

// T returns the localized string for key, falling back to English then the key.
func (t *Translator) T(key string) string {
	entries, ok := catalog[key]
	if !ok {
		return key
	}
	if s, ok := entries[t.Lang]; ok && s != "" {
		return s
	}
	if s, ok := entries[EN]; ok && s != "" {
		return s
	}
	return key
}

// Tf is T with printf-style formatting.
func (t *Translator) Tf(key string, args ...any) string {
	return fmt.Sprintf(t.T(key), args...)
}

// HasKey reports whether the catalog contains key (for tests/tooling).
func HasKey(key string) bool {
	_, ok := catalog[key]
	return ok
}

// Keys returns all catalog keys (for completeness checks).
func Keys() []string {
	keys := make([]string, 0, len(catalog))
	for k := range catalog {
		keys = append(keys, k)
	}
	return keys
}
