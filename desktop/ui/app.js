// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: AGPL-3.0-or-later

const tauri = window.__TAURI__;
const invoke = tauri?.core?.invoke;

const translations = {
  ru: {
    language: "Язык", checking: "Проверяем соединение", waiting: "Ожидаем данные Mazzy VPN CLI",
    protocol: "Протокол", interface: "Интерфейс", handshake: "Handshake", publicIp: "Публичный IP",
    quickConnect: "Быстро подключить", useDefault: "Default-конфиг", reconnect: "Переподключить",
    safeRestart: "Безопасный restart", disconnect: "Отключить", stopTunnel: "Остановить туннель",
    doctor: "Самодиагностика", checkSystem: "Проверить систему", protection: "PROTECTION",
    systemHealth: "Состояние системы", checkingShort: "CHECK", autoConnect: "Автоподключение",
    startsOnBoot: "Запуск при загрузке", healthMonitor: "Health monitor",
    autoHealing: "Автопроверка и лечение", fallback: "Резервный VPN",
    conflictGuard: "Защита от конфликтов", configurations: "CONFIGURATIONS",
    vpnProfiles: "VPN-профили", activity: "ACTIVITY", eventLog: "События",
    noEvents: "Событий пока нет", neverUpdated: "Данные ещё не получены",
    connected: "VPN защищает соединение", connecting: "VPN подключается", disconnected: "VPN отключён",
    degraded: "Соединение требует внимания", internetOk: "Интернет доступен через защищённый туннель",
    internetDown: "Туннель без подтверждённого доступа в интернет", noProfile: "Профиль не выбран",
    enabled: "ON", disabled: "OFF", inactive: "НЕТ", active: "АКТИВЕН", healthy: "HEALTHY",
    attention: "ATTENTION", secondsAgo: "с назад", notReceived: "не получен", cacheStale: "Данные устарели",
    preview: "Preview: VPN backend этой ОС ещё не реализован", actionStarted: "Выполняется",
    actionDone: "Действие завершено", actionFailed: "Действие не выполнено", refreshed: "Статус обновлён",
    hidden: "скрыт", statusChanged: "Состояние VPN изменилось", update: "Обновлено"
  },
  en: {
    language: "Language", checking: "Checking connection", waiting: "Waiting for Mazzy VPN CLI data",
    protocol: "Protocol", interface: "Interface", handshake: "Handshake", publicIp: "Public IP",
    quickConnect: "Quick connect", useDefault: "Default config", reconnect: "Reconnect",
    safeRestart: "Safe restart", disconnect: "Disconnect", stopTunnel: "Stop tunnel",
    doctor: "Self-diagnostics", checkSystem: "Check the system", protection: "PROTECTION",
    systemHealth: "System health", checkingShort: "CHECK", autoConnect: "Auto-connect",
    startsOnBoot: "Starts on boot", healthMonitor: "Health monitor", autoHealing: "Automatic checks and healing",
    fallback: "Fallback VPN", conflictGuard: "Conflict protection", configurations: "CONFIGURATIONS",
    vpnProfiles: "VPN profiles", activity: "ACTIVITY", eventLog: "Events", noEvents: "No events yet",
    neverUpdated: "No data received yet", connected: "VPN is protecting your connection",
    connecting: "VPN is connecting", disconnected: "VPN is disconnected", degraded: "Connection needs attention",
    internetOk: "Internet is available through the secure tunnel", internetDown: "Tunnel has no verified internet access",
    noProfile: "No default profile selected", enabled: "ON", disabled: "OFF", inactive: "NONE",
    active: "ACTIVE", healthy: "HEALTHY", attention: "ATTENTION", secondsAgo: "s ago",
    notReceived: "not received", cacheStale: "Status data is stale",
    preview: "Preview: this OS VPN backend is not implemented yet", actionStarted: "Running",
    actionDone: "Action completed", actionFailed: "Action failed", refreshed: "Status refreshed",
    hidden: "hidden", statusChanged: "VPN status changed", update: "Updated"
  },
  de: {
    language: "Sprache", checking: "Verbindung wird geprüft", waiting: "Warte auf Mazzy VPN CLI",
    protocol: "Protokoll", interface: "Schnittstelle", handshake: "Handshake", publicIp: "Öffentliche IP",
    quickConnect: "Schnell verbinden", useDefault: "Standardkonfiguration", reconnect: "Neu verbinden",
    safeRestart: "Sicherer Neustart", disconnect: "Trennen", stopTunnel: "Tunnel stoppen",
    doctor: "Selbstdiagnose", checkSystem: "System prüfen", protection: "SCHUTZ",
    systemHealth: "Systemzustand", checkingShort: "PRÜFEN", autoConnect: "Autoverbindung",
    startsOnBoot: "Start beim Booten", healthMonitor: "Health Monitor", autoHealing: "Automatische Heilung",
    fallback: "Fallback-VPN", conflictGuard: "Konfliktschutz", configurations: "KONFIGURATIONEN",
    vpnProfiles: "VPN-Profile", activity: "AKTIVITÄT", eventLog: "Ereignisse", noEvents: "Noch keine Ereignisse",
    neverUpdated: "Noch keine Daten", connected: "VPN schützt die Verbindung", connecting: "VPN verbindet",
    disconnected: "VPN ist getrennt", degraded: "Verbindung benötigt Aufmerksamkeit",
    internetOk: "Internet über den geschützten Tunnel verfügbar", internetDown: "Kein Internet bestätigt",
    noProfile: "Kein Standardprofil gewählt", enabled: "AN", disabled: "AUS", inactive: "KEIN",
    active: "AKTIV", healthy: "GESUND", attention: "ACHTUNG", secondsAgo: "s zuvor",
    notReceived: "nicht empfangen", cacheStale: "Statusdaten sind veraltet",
    preview: "Vorschau: VPN-Backend für dieses OS fehlt noch", actionStarted: "Wird ausgeführt",
    actionDone: "Aktion abgeschlossen", actionFailed: "Aktion fehlgeschlagen", refreshed: "Status aktualisiert",
    hidden: "verborgen", statusChanged: "VPN-Status geändert", update: "Aktualisiert"
  },
  zh: {
    language: "语言", checking: "正在检查连接", waiting: "正在等待 Mazzy VPN CLI 数据",
    protocol: "协议", interface: "接口", handshake: "握手", publicIp: "公网 IP",
    quickConnect: "快速连接", useDefault: "默认配置", reconnect: "重新连接", safeRestart: "安全重启",
    disconnect: "断开连接", stopTunnel: "停止隧道", doctor: "自我诊断", checkSystem: "检查系统",
    protection: "保护", systemHealth: "系统状态", checkingShort: "检查", autoConnect: "自动连接",
    startsOnBoot: "开机启动", healthMonitor: "健康监控", autoHealing: "自动检查和修复",
    fallback: "备用 VPN", conflictGuard: "冲突保护", configurations: "配置", vpnProfiles: "VPN 配置",
    activity: "活动", eventLog: "事件", noEvents: "暂无事件", neverUpdated: "尚未收到数据",
    connected: "VPN 正在保护连接", connecting: "VPN 正在连接", disconnected: "VPN 已断开",
    degraded: "连接需要注意", internetOk: "网络已通过安全隧道连接",
    internetDown: "隧道尚未确认网络访问", noProfile: "尚未选择默认配置",
    enabled: "开启", disabled: "关闭", inactive: "无", active: "活动", healthy: "正常",
    attention: "注意", secondsAgo: "秒前", notReceived: "未收到", cacheStale: "状态数据已过期",
    preview: "预览：此操作系统的 VPN 后端尚未实现", actionStarted: "正在执行",
    actionDone: "操作完成", actionFailed: "操作失败", refreshed: "状态已刷新",
    hidden: "已隐藏", statusChanged: "VPN 状态已变化", update: "更新时间"
  },
  ja: {
    language: "言語", checking: "接続を確認中", waiting: "Mazzy VPN CLI のデータを待っています",
    protocol: "プロトコル", interface: "インターフェース", handshake: "ハンドシェイク", publicIp: "公開 IP",
    quickConnect: "クイック接続", useDefault: "既定の設定", reconnect: "再接続", safeRestart: "安全な再起動",
    disconnect: "切断", stopTunnel: "トンネルを停止", doctor: "自己診断", checkSystem: "システムを確認",
    protection: "保護", systemHealth: "システム状態", checkingShort: "確認", autoConnect: "自動接続",
    startsOnBoot: "起動時に開始", healthMonitor: "ヘルスモニター", autoHealing: "自動確認と修復",
    fallback: "予備 VPN", conflictGuard: "競合を防止", configurations: "設定", vpnProfiles: "VPN プロファイル",
    activity: "アクティビティ", eventLog: "イベント", noEvents: "イベントはありません",
    neverUpdated: "データ未受信", connected: "VPN が接続を保護しています", connecting: "VPN 接続中",
    disconnected: "VPN は切断されています", degraded: "接続を確認してください",
    internetOk: "安全なトンネル経由で接続済み", internetDown: "インターネット接続を確認できません",
    noProfile: "既定のプロファイルが未選択", enabled: "オン", disabled: "オフ", inactive: "なし",
    active: "有効", healthy: "正常", attention: "注意", secondsAgo: "秒前", notReceived: "未受信",
    cacheStale: "状態データが古いです", preview: "プレビュー：この OS の VPN バックエンドは未実装です",
    actionStarted: "実行中", actionDone: "操作完了", actionFailed: "操作失敗", refreshed: "状態を更新しました",
    hidden: "非表示", statusChanged: "VPN 状態が変わりました", update: "更新"
  },
  ko: {
    language: "언어", checking: "연결 확인 중", waiting: "Mazzy VPN CLI 데이터를 기다리는 중",
    protocol: "프로토콜", interface: "인터페이스", handshake: "핸드셰이크", publicIp: "공인 IP",
    quickConnect: "빠른 연결", useDefault: "기본 구성", reconnect: "다시 연결", safeRestart: "안전한 재시작",
    disconnect: "연결 해제", stopTunnel: "터널 중지", doctor: "자가 진단", checkSystem: "시스템 확인",
    protection: "보호", systemHealth: "시스템 상태", checkingShort: "확인", autoConnect: "자동 연결",
    startsOnBoot: "부팅 시 시작", healthMonitor: "상태 모니터", autoHealing: "자동 확인 및 복구",
    fallback: "대체 VPN", conflictGuard: "충돌 방지", configurations: "구성", vpnProfiles: "VPN 프로필",
    activity: "활동", eventLog: "이벤트", noEvents: "이벤트 없음", neverUpdated: "아직 데이터 없음",
    connected: "VPN이 연결을 보호합니다", connecting: "VPN 연결 중", disconnected: "VPN 연결 해제됨",
    degraded: "연결 확인이 필요합니다", internetOk: "보안 터널을 통해 인터넷 연결됨",
    internetDown: "인터넷 연결이 확인되지 않음", noProfile: "기본 프로필이 선택되지 않음",
    enabled: "켜짐", disabled: "꺼짐", inactive: "없음", active: "활성", healthy: "정상",
    attention: "주의", secondsAgo: "초 전", notReceived: "수신 안 됨", cacheStale: "상태 데이터가 오래됨",
    preview: "미리보기: 이 OS의 VPN 백엔드는 아직 구현되지 않음", actionStarted: "실행 중",
    actionDone: "작업 완료", actionFailed: "작업 실패", refreshed: "상태 새로 고침",
    hidden: "숨김", statusChanged: "VPN 상태 변경됨", update: "업데이트"
  }
};

const $ = (selector) => document.querySelector(selector);
const state = {
  lang: localStorage.getItem("mazzy-language") || "ru",
  hideIp: localStorage.getItem("mazzy-hide-ip") === "true",
  status: null,
  lastSignature: "",
  events: [],
  busy: false
};

function t(key) {
  return translations[state.lang]?.[key] || translations.en[key] || key;
}

function applyLanguage() {
  document.documentElement.lang = state.lang;
  $("#language-select").value = state.lang;
  document.querySelectorAll("[data-i18n]").forEach((node) => {
    node.textContent = t(node.dataset.i18n);
  });
  if (state.status) renderStatus(state.status);
}

function setMiniState(selector, enabled, activeLabel = "enabled", inactiveLabel = "disabled") {
  const node = $(selector);
  node.textContent = enabled ? t(activeLabel) : t(inactiveLabel);
  node.className = `mini-state ${enabled ? "on" : ""}`;
}

function addEvent(title, detail = "", type = "success") {
  state.events.unshift({ title, detail, type, time: new Date() });
  state.events = state.events.slice(0, 7);
  const list = $("#event-list");
  list.replaceChildren(...state.events.map((event) => {
    const row = document.createElement("div");
    row.className = `event ${event.type}`;
    const dot = document.createElement("span");
    dot.className = "event-dot";
    const copy = document.createElement("div");
    const strong = document.createElement("strong");
    strong.textContent = event.title;
    const small = document.createElement("small");
    small.textContent = event.detail;
    copy.append(strong, small);
    const time = document.createElement("time");
    time.textContent = event.time.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    row.append(dot, copy, time);
    return row;
  }));
}

let toastTimer;
function showToast(message, error = false) {
  const toast = $("#toast");
  toast.textContent = message;
  toast.className = `toast visible${error ? " error" : ""}`;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { toast.className = "toast"; }, 4200);
}

function renderStatus(data) {
  state.status = data;
  const available = data?.available !== false && Number(data?.generated_at || 0) > 0;
  const connected = Boolean(data?.tunnel_active && data?.internet === "up");
  const connecting = Boolean(data?.desired === "up" && !connected);
  const unhealthy = Boolean(available && (data?.health_failures > 0 || data?.internet === "down"));
  const visual = connected ? "connected" : connecting ? "connecting" : unhealthy ? "error" : "";

  $("#connection-orb").className = `connection-orb ${visual}`;
  $("#status-dot").className = `status-dot ${visual}`;
  $("#status-label").textContent = connected ? t("connected") : connecting ? t("connecting")
    : unhealthy ? t("degraded") : t("disconnected");
  $("#location-label").textContent = data?.location || (data?.selected ? "—" : t("noProfile"));
  $("#connection-summary").textContent = connected ? t("internetOk")
    : data?.selected ? t("internetDown") : t("noProfile");
  $("#protocol-value").textContent = data?.protocol_name || "—";
  $("#interface-value").textContent = data?.interface || "—";
  $("#handshake-value").textContent = Number.isFinite(data?.handshake_age)
    ? `${data.handshake_age} ${t("secondsAgo")}` : t("notReceived");
  const rawIp = data?.public_ip || "—";
  $("#ip-value").textContent = state.hideIp && rawIp !== "—" ? "•••.•••.•••.•••" : rawIp;
  $("#privacy-button").textContent = state.hideIp ? "◌" : "◉";

  setMiniState("#autostart-state", Boolean(data?.autostart));
  setMiniState("#monitor-state", Boolean(data?.health_monitor));
  setMiniState("#fallback-state", Boolean(data?.fallback?.active), "active", "inactive");
  if (data?.fallback?.active) $("#fallback-state").classList.add("warn");

  const badge = $("#health-badge");
  badge.textContent = connected && data?.healthy ? t("healthy") : t("attention");
  badge.className = `health-badge ${connected && data?.healthy ? "ok" : unhealthy ? "bad" : ""}`;

  const profiles = data?.profiles || {};
  const keys = ["amneziawg", "wireguard", "openvpn", "l2tp"];
  const counts = keys.map((key) => Number(profiles[key] || 0));
  const total = counts.reduce((sum, count) => sum + count, 0);
  const max = Math.max(...counts, 1);
  $("#profile-total").textContent = String(total);
  keys.forEach((key, index) => {
    $(`#count-${key}`).textContent = String(counts[index]);
    $(`#bar-${key}`).style.width = `${(counts[index] / max) * 100}%`;
  });

  $("#engine-version").textContent = `${data?.product || "Mazzy VPN"} CLI ${data?.version || ""}`.trim();
  const generated = Number(data?.generated_at || 0);
  const age = generated ? Math.max(0, Math.round(Date.now() / 1000 - generated)) : null;
  $("#freshness-label").textContent = age === null ? t("neverUpdated")
    : age > 60 ? `${t("cacheStale")} · ${age} ${t("secondsAgo")}`
    : `${t("update")} · ${age} ${t("secondsAgo")}`;

  const signature = `${data?.service_state}|${data?.desired}|${data?.internet}|${data?.profile}`;
  if (state.lastSignature && state.lastSignature !== signature) {
    addEvent(t("statusChanged"), data?.location || data?.service_state || "", connected ? "success" : "error");
  }
  state.lastSignature = signature;
}

async function refreshStatus(manual = false) {
  try {
    if (!invoke) throw new Error("Tauri runtime is unavailable");
    const data = await invoke("get_status");
    renderStatus(data);
    if (manual) showToast(t("refreshed"));
  } catch (error) {
    renderStatus({ available: false, generated_at: 0, desired: "unknown", profiles: {} });
    if (manual) showToast(String(error), true);
  }
}

async function runAction(action) {
  if (state.busy) return;
  state.busy = true;
  document.querySelectorAll("[data-action]").forEach((button) => { button.disabled = true; });
  showToast(`${t("actionStarted")}: ${action}`);
  try {
    if (!invoke) throw new Error("Tauri runtime is unavailable");
    const result = await invoke("run_action", { action });
    const detail = result.output?.trim().split("\n").slice(-1)[0] || "";
    addEvent(result.success ? t("actionDone") : t("actionFailed"), detail,
      result.success ? "success" : "error");
    showToast(result.success ? t("actionDone") : `${t("actionFailed")}: ${detail}`, !result.success);
  } catch (error) {
    addEvent(t("actionFailed"), String(error), "error");
    showToast(`${t("actionFailed")}: ${error}`, true);
  } finally {
    state.busy = false;
    document.querySelectorAll("[data-action]").forEach((button) => { button.disabled = false; });
    await refreshStatus(false);
  }
}

document.addEventListener("DOMContentLoaded", async () => {
  applyLanguage();
  $("#language-select").addEventListener("change", (event) => {
    state.lang = event.target.value;
    localStorage.setItem("mazzy-language", state.lang);
    applyLanguage();
  });
  $("#privacy-button").addEventListener("click", () => {
    state.hideIp = !state.hideIp;
    localStorage.setItem("mazzy-hide-ip", String(state.hideIp));
    if (state.status) renderStatus(state.status);
  });
  $("#refresh-button").addEventListener("click", () => refreshStatus(true));
  $("#hide-button").addEventListener("click", async () => {
    if (invoke) await invoke("hide_main_window");
  });
  document.querySelectorAll("[data-action]").forEach((button) => {
    button.addEventListener("click", () => runAction(button.dataset.action));
  });

  if (tauri?.event?.listen) {
    await tauri.event.listen("vpn-action-result", ({ payload }) => {
      const detail = payload?.output?.trim().split("\n").slice(-1)[0] || "";
      addEvent(payload?.success ? t("actionDone") : t("actionFailed"), detail,
        payload?.success ? "success" : "error");
      showToast(payload?.success ? t("actionDone") : `${t("actionFailed")}: ${detail}`, !payload?.success);
      refreshStatus(false);
    });
  }

  if (invoke) {
    const platform = await invoke("get_platform_info");
    $("#platform-note").textContent = platform?.functional ? "" : t("preview");
  }
  await refreshStatus(false);
  setInterval(() => refreshStatus(false), 5000);
});
