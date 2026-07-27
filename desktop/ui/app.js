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

Object.assign(translations.ru, {
  navDashboard: "Обзор", navProfiles: "Профили", navDiagnostics: "Диагностика", navSettings: "Настройки",
  profileLibrary: "БИБЛИОТЕКА ПРОФИЛЕЙ", profileManagement: "Профили и локации",
  profileManagementHint: "Импорт, проверка, выбор и безопасный тест конфигураций.",
  importFiles: "Загрузить файлы", scanFolder: "Проверить папку", importFolder: "Импортировать папку",
  searchProfiles: "Поиск профиля или локации", allProtocols: "Все протоколы",
  profilesLoading: "Загружаем профили…", profilesEmpty: "Профили не найдены",
  selectedProfile: "Выбран по умолчанию", connectProfile: "Подключить", testProfile: "Тест", removeProfile: "Удалить",
  testingTools: "ТЕСТИРОВАНИЕ", testingTitle: "Проверка конфигураций",
  testTimeout: "Таймаут живого теста", testProtocol: "Протокол",
  validateProfiles: "Проверить формат", probeEndpoints: "DNS и ping endpoint",
  testAllProfiles: "Живой test-all с rollback", emergencyMode: "Аварийный выбор рабочего VPN",
  rollbackHint: "Живые тесты временно меняют маршрут и всегда используют timeout + rollback.",
  diagnosticsCenter: "ДИАГНОСТИКА", diagnosticsTitle: "Doctor, тесты и журнал",
  diagnosticsHint: "Полный результат больше не скрывается: вывод сохраняется в этой панели.",
  runDoctor: "Запустить Doctor", repairSystem: "Установить/исправить",
  connectionDiagnose: "Диагностика соединения",
  connectionDiagnoseHint: "Маршрут, DNS, туннель, handshake и интернет",
  offlineSelfTest: "Offline self-test", offlineSelfTestHint: "Формат, зависимости, службы и права",
  liveSelfTest: "Live self-test", liveSelfTestHint: "Проверка туннелей с rollback",
  serviceLog: "Журнал сервиса", serviceLogHint: "Последние события без секретов", logLines: "Строк журнала",
  outputReady: "Готово к проверке", outputRunning: "Выполняется", outputSuccess: "Успешно",
  outputFailed: "Есть ошибки", diagnosticOutput: "Результат", clearOutput: "Очистить",
  outputPlaceholder: "Здесь появится полный вывод Doctor, тестов и журнала.",
  settingsCenter: "НАСТРОЙКИ", settingsTitle: "Установка и системные настройки",
  settingsHint: "Desktop включает совместимый движок и может установить недостающие зависимости.",
  installRepair: "Установить / обновить / исправить", installation: "УСТАНОВКА",
  engineReadiness: "Готовность движка", installedVersion: "Установлено",
  bundledVersion: "В комплекте Desktop", engineService: "Системная служба",
  recoveryMonitor: "Recovery monitor", services: "СЛУЖБЫ", serviceControl: "Управление службами",
  notifications: "Уведомления", notificationsHint: "Локально, без телеметрии",
  privacyMode: "Скрывать публичный IP", privacyModeHint: "Применяется к Dashboard",
  installed: "УСТАНОВЛЕНО", missing: "НЕТ", updateRequired: "НУЖНО ОБНОВИТЬ",
  ready: "ГОТОВО", confirmLiveTest: "Живой тест временно изменит VPN-маршрут и затем выполнит rollback. Продолжить?",
  confirmRemove: "Удалить выбранный VPN-профиль?", confirmRepair: "Запустить системную установку и исправление зависимостей?",
  profilesRefreshed: "Список профилей обновлён", noSelection: "Ничего не выбрано"
});

Object.assign(translations.en, {
  navDashboard: "Dashboard", navProfiles: "Profiles", navDiagnostics: "Diagnostics", navSettings: "Settings",
  profileLibrary: "PROFILE LIBRARY", profileManagement: "Profiles and locations",
  profileManagementHint: "Import, validate, select and safely test configurations.",
  importFiles: "Import files", scanFolder: "Scan folder", importFolder: "Import folder",
  searchProfiles: "Search profile or location", allProtocols: "All protocols",
  profilesLoading: "Loading profiles…", profilesEmpty: "No profiles found",
  selectedProfile: "Selected as default", connectProfile: "Connect", testProfile: "Test", removeProfile: "Remove",
  testingTools: "TESTING", testingTitle: "Configuration checks", testTimeout: "Live test timeout",
  testProtocol: "Protocol", validateProfiles: "Validate format", probeEndpoints: "DNS and endpoint ping",
  testAllProfiles: "Live test-all with rollback", emergencyMode: "Emergency working VPN selection",
  rollbackHint: "Live tests temporarily change routes and always use a timeout plus rollback.",
  diagnosticsCenter: "DIAGNOSTICS", diagnosticsTitle: "Doctor, tests and logs",
  diagnosticsHint: "Complete results are retained in this panel instead of being discarded.",
  runDoctor: "Run Doctor", repairSystem: "Install / repair",
  connectionDiagnose: "Connection diagnostics",
  connectionDiagnoseHint: "Route, DNS, tunnel, handshake and Internet",
  offlineSelfTest: "Offline self-test", offlineSelfTestHint: "Format, dependencies, services and permissions",
  liveSelfTest: "Live self-test", liveSelfTestHint: "Tunnel checks with rollback",
  serviceLog: "Service log", serviceLogHint: "Recent redacted events", logLines: "Log lines",
  outputReady: "Ready", outputRunning: "Running", outputSuccess: "Success", outputFailed: "Problems found",
  diagnosticOutput: "Result", clearOutput: "Clear",
  outputPlaceholder: "Complete Doctor, test and service log output will appear here.",
  settingsCenter: "SETTINGS", settingsTitle: "Installation and system settings",
  settingsHint: "Desktop bundles a compatible engine and can install missing dependencies.",
  installRepair: "Install / update / repair", installation: "INSTALLATION", engineReadiness: "Engine readiness",
  installedVersion: "Installed", bundledVersion: "Bundled with Desktop", engineService: "System service",
  recoveryMonitor: "Recovery monitor", services: "SERVICES", serviceControl: "Service control",
  notifications: "Notifications", notificationsHint: "Local only, no telemetry",
  privacyMode: "Hide public IP", privacyModeHint: "Applied to Dashboard",
  installed: "INSTALLED", missing: "MISSING", updateRequired: "UPDATE REQUIRED", ready: "READY",
  confirmLiveTest: "The live test temporarily changes VPN routes and then rolls back. Continue?",
  confirmRemove: "Remove the selected VPN profile?", confirmRepair: "Run system installation and dependency repair?",
  profilesRefreshed: "Profile list refreshed", noSelection: "Nothing selected"
});

const $ = (selector) => document.querySelector(selector);
const state = {
  lang: localStorage.getItem("mazzy-language") || "ru",
  hideIp: localStorage.getItem("mazzy-hide-ip") === "true",
  notifications: localStorage.getItem("mazzy-notifications") !== "false",
  page: "dashboard",
  status: null,
  profiles: [],
  installation: null,
  lastSignature: "",
  events: [],
  busy: false,
  lastOperation: null
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
  document.querySelectorAll("[data-i18n-placeholder]").forEach((node) => {
    node.placeholder = t(node.dataset.i18nPlaceholder);
  });
  if (state.status) renderStatus(state.status);
  renderProfiles();
  if (state.installation) renderInstallation(state.installation);
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

function showPage(page) {
  state.page = page;
  document.querySelectorAll("[data-page]").forEach((button) => {
    button.classList.toggle("active", button.dataset.page === page);
  });
  document.querySelectorAll("[data-page-panel]").forEach((panel) => {
    panel.classList.toggle("active", panel.dataset.pagePanel === page);
  });
}

function operationTimeout() {
  const value = Number($("#test-timeout")?.value || 45);
  return Math.min(600, Math.max(2, Number.isFinite(value) ? value : 45));
}

function setBusy(busy) {
  state.busy = busy;
  document.querySelectorAll("button").forEach((button) => {
    if (!button.matches("[data-page], #hide-button, #privacy-button")) button.disabled = busy;
  });
}

function showOperationResult(result, title = "") {
  state.lastOperation = result;
  const output = $("#operation-output");
  const status = $("#operation-state");
  $("#operation-title").textContent = title || result?.action || t("diagnosticOutput");
  status.textContent = result?.success ? t("outputSuccess") : t("outputFailed");
  status.className = `operation-state ${result?.success ? "success" : "error"}`;
  output.textContent = result?.output?.trim() || (result?.success ? t("actionDone") : t("actionFailed"));
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

function renderProfiles() {
  const list = $("#profile-list");
  if (!list) return;
  const query = ($("#profile-search")?.value || "").trim().toLocaleLowerCase();
  const protocol = $("#protocol-filter")?.value || "all";
  const profiles = state.profiles.filter((profile) => {
    const matchesProtocol = protocol === "all" || profile.protocol === protocol;
    const haystack = `${profile.name || ""} ${profile.location || ""} ${profile.protocol_name || ""}`.toLocaleLowerCase();
    return matchesProtocol && (!query || haystack.includes(query));
  });
  if (!profiles.length) {
    const empty = document.createElement("div");
    empty.className = "empty-state";
    empty.textContent = t("profilesEmpty");
    list.replaceChildren(empty);
    return;
  }
  list.replaceChildren(...profiles.map((profile) => {
    const row = document.createElement("div");
    row.className = `profile-item${profile.selected ? " selected" : ""}`;
    const marker = document.createElement("span");
    marker.className = "profile-marker";
    const copy = document.createElement("div");
    copy.className = "profile-copy";
    const name = document.createElement("strong");
    name.textContent = profile.name || profile.file_name;
    const detail = document.createElement("small");
    detail.textContent = `${profile.protocol_name || profile.protocol}${profile.selected ? ` · ${t("selectedProfile")}` : ""}`;
    copy.append(name, detail);
    const actions = document.createElement("div");
    actions.className = "profile-actions";
    const connect = document.createElement("button");
    connect.type = "button";
    connect.textContent = t("connectProfile");
    connect.addEventListener("click", () => runOperation({
      kind: "connect", protocol: profile.protocol, profile: profile.file_name
    }, t("connectProfile")));
    const test = document.createElement("button");
    test.type = "button";
    test.textContent = t("testProfile");
    test.addEventListener("click", () => {
      if (window.confirm(t("confirmLiveTest"))) {
        runOperation({
          kind: "test", protocol: profile.protocol, profile: profile.file_name,
          timeout: operationTimeout()
        }, `${t("testProfile")}: ${profile.name}`);
      }
    });
    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "remove";
    remove.textContent = t("removeProfile");
    remove.addEventListener("click", () => {
      if (window.confirm(t("confirmRemove"))) {
        runOperation({
          kind: "remove-profile", protocol: profile.protocol, profile: profile.file_name
        }, `${t("removeProfile")}: ${profile.name}`);
      }
    });
    actions.append(connect, test, remove);
    row.append(marker, copy, actions);
    return row;
  }));
}

async function refreshProfiles(manual = false) {
  try {
    if (!invoke) throw new Error("Tauri runtime is unavailable");
    const data = await invoke("get_profiles");
    state.profiles = Array.isArray(data?.profiles) ? data.profiles : [];
    renderProfiles();
    if (manual) showToast(t("profilesRefreshed"));
  } catch (error) {
    state.profiles = [];
    renderProfiles();
    if (manual) showToast(String(error), true);
  }
}

function renderInstallation(report) {
  state.installation = report;
  $("#installed-version").textContent = report?.installed_version || t("missing");
  $("#bundled-version").textContent = report?.bundled_version || t("missing");
  $("#service-installed").textContent = report?.service_installed ? t("installed") : t("missing");
  $("#monitor-installed").textContent = report?.monitor_installed ? t("installed") : t("missing");
  const badge = $("#installation-badge");
  badge.textContent = report?.needs_install ? t("updateRequired") : t("ready");
  badge.className = `health-badge ${report?.needs_install ? "bad" : "ok"}`;
  const dependencies = $("#dependency-list");
  dependencies.replaceChildren(...(report?.dependencies || []).map((dependency) => {
    const row = document.createElement("div");
    row.className = `dependency${dependency.installed ? " installed" : ""}`;
    const label = document.createElement("span");
    label.textContent = `${dependency.label} · ${dependency.required_for}`;
    const status = document.createElement("span");
    status.textContent = dependency.installed ? "OK" : t("missing");
    row.append(label, status);
    return row;
  }));
}

async function refreshInstallation() {
  if (!invoke) return;
  try {
    renderInstallation(await invoke("get_installation_report"));
  } catch (error) {
    showToast(String(error), true);
  }
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

async function runOperation(request, title = "", openOutput = true) {
  if (state.busy) return;
  setBusy(true);
  $("#operation-state").textContent = t("outputRunning");
  $("#operation-state").className = "operation-state running";
  $("#operation-title").textContent = title || request.kind;
  $("#operation-output").textContent = `${t("actionStarted")}: ${title || request.kind}…`;
  showToast(`${t("actionStarted")}: ${title || request.kind}`);
  try {
    if (!invoke) throw new Error("Tauri runtime is unavailable");
    const result = await invoke("run_operation", { request });
    const detail = result.output?.trim().split("\n").slice(-1)[0] || "";
    addEvent(result.success ? t("actionDone") : t("actionFailed"), detail,
      result.success ? "success" : "error");
    showOperationResult(result, title);
    showToast(result.success ? t("actionDone") : `${t("actionFailed")}: ${detail}`, !result.success);
    if (openOutput && (request.kind === "doctor" || request.kind === "self-test" ||
        request.kind === "logs" || !result.success)) {
      showPage("diagnostics");
    }
    return result;
  } catch (error) {
    addEvent(t("actionFailed"), String(error), "error");
    showToast(`${t("actionFailed")}: ${error}`, true);
    showOperationResult({ success: false, action: request.kind, output: String(error) }, title);
    if (openOutput) showPage("diagnostics");
    return null;
  } finally {
    setBusy(false);
    await refreshStatus(false);
    await refreshProfiles(false);
    await refreshInstallation();
  }
}

async function runAction(action) {
  const request = action === "doctor" ? { kind: "doctor", fix: false } : { kind: action };
  return runOperation(request, action, action === "doctor");
}

document.addEventListener("DOMContentLoaded", async () => {
  applyLanguage();
  $("#language-select").addEventListener("change", async (event) => {
    state.lang = event.target.value;
    localStorage.setItem("mazzy-language", state.lang);
    applyLanguage();
    await runOperation({ kind: "language", language: state.lang }, t("language"), false);
  });
  $("#privacy-button").addEventListener("click", () => {
    state.hideIp = !state.hideIp;
    localStorage.setItem("mazzy-hide-ip", String(state.hideIp));
    $("#privacy-toggle").checked = state.hideIp;
    if (state.status) renderStatus(state.status);
  });
  $("#privacy-toggle").checked = state.hideIp;
  $("#privacy-toggle").addEventListener("change", (event) => {
    state.hideIp = event.target.checked;
    localStorage.setItem("mazzy-hide-ip", String(state.hideIp));
    if (state.status) renderStatus(state.status);
  });
  $("#notifications-toggle").checked = state.notifications;
  $("#notifications-toggle").addEventListener("change", (event) => {
    state.notifications = event.target.checked;
    localStorage.setItem("mazzy-notifications", String(state.notifications));
  });
  $("#refresh-button").addEventListener("click", () => refreshStatus(true));
  $("#hide-button").addEventListener("click", async () => {
    if (invoke) await invoke("hide_main_window");
  });
  document.querySelectorAll("[data-page]").forEach((button) => {
    button.addEventListener("click", () => showPage(button.dataset.page));
  });
  document.querySelectorAll("[data-action]").forEach((button) => {
    button.addEventListener("click", () => runAction(button.dataset.action));
  });
  $("#profile-search").addEventListener("input", renderProfiles);
  $("#protocol-filter").addEventListener("change", renderProfiles);
  $("#profiles-refresh-button").addEventListener("click", async () => {
    await runOperation({ kind: "refresh" }, t("profilesRefreshed"), false);
    await refreshProfiles(true);
  });

  $("#import-files-button").addEventListener("click", async () => {
    if (!invoke || state.busy) return;
    const paths = await invoke("pick_profile_files");
    if (Array.isArray(paths) && paths.length) {
      await runOperation({ kind: "import-files", paths, force: false }, t("importFiles"), false);
    }
  });
  $("#scan-folder-button").addEventListener("click", async () => {
    if (!invoke || state.busy) return;
    const path = await invoke("pick_profile_folder");
    if (path) {
      await runOperation({
        kind: "import-folder", path, dry_run: true, force: false
      }, t("scanFolder"), true);
    }
  });
  $("#import-folder-button").addEventListener("click", async () => {
    if (!invoke || state.busy) return;
    const path = await invoke("pick_profile_folder");
    if (path) {
      await runOperation({
        kind: "import-folder", path, dry_run: false, force: false
      }, t("importFolder"), false);
    }
  });
  $("#validate-button").addEventListener("click", () => runOperation({
    kind: "validate", protocol: $("#test-protocol").value
  }, t("validateProfiles")));
  $("#probe-button").addEventListener("click", () => runOperation({
    kind: "probe", protocol: $("#test-protocol").value, timeout: Math.min(30, operationTimeout())
  }, t("probeEndpoints")));
  $("#test-all-button").addEventListener("click", () => {
    if (window.confirm(t("confirmLiveTest"))) {
      runOperation({
        kind: "test-all", protocol: $("#test-protocol").value, timeout: operationTimeout()
      }, t("testAllProfiles"));
    }
  });
  $("#emergency-button").addEventListener("click", () => {
    if (window.confirm(t("confirmLiveTest"))) {
      const selected = $("#test-protocol").value;
      runOperation({
        kind: "emergency", protocol: selected === "all" ? null : selected,
        timeout: operationTimeout()
      }, t("emergencyMode"));
    }
  });

  $("#doctor-button").addEventListener("click", () =>
    runOperation({ kind: "doctor", fix: false }, t("runDoctor")));
  $("#diagnose-button").addEventListener("click", () =>
    runOperation({ kind: "diagnose" }, t("connectionDiagnose")));
  $("#doctor-fix-button").addEventListener("click", () => {
    if (window.confirm(t("confirmRepair"))) {
      runOperation({ kind: "doctor", fix: true }, t("repairSystem"));
    }
  });
  $("#self-test-offline-button").addEventListener("click", () =>
    runOperation({ kind: "self-test", live: false, timeout: 3 }, t("offlineSelfTest")));
  $("#self-test-live-button").addEventListener("click", () => {
    if (window.confirm(t("confirmLiveTest"))) {
      runOperation({ kind: "self-test", live: true, timeout: 3 }, t("liveSelfTest"));
    }
  });
  $("#logs-button").addEventListener("click", () => {
    const lines = Math.min(1000, Math.max(20, Number($("#log-lines").value || 200)));
    runOperation({ kind: "logs", lines }, t("serviceLog"));
  });
  $("#clear-output-button").addEventListener("click", () => {
    state.lastOperation = null;
    $("#operation-state").textContent = t("outputReady");
    $("#operation-state").className = "operation-state";
    $("#operation-title").textContent = t("diagnosticOutput");
    $("#operation-output").textContent = t("outputPlaceholder");
  });

  $("#bootstrap-button").addEventListener("click", () => {
    if (window.confirm(t("confirmRepair"))) {
      runOperation({ kind: "bootstrap" }, t("installRepair"));
    }
  });
  document.querySelectorAll("[data-service-action]").forEach((button) => {
    button.addEventListener("click", () => {
      const [service, value] = button.dataset.serviceAction.split("-");
      runOperation({
        kind: service, enabled: value === "on"
      }, service === "autostart" ? t("autoConnect") : t("healthMonitor"), false);
    });
  });

  if (tauri?.event?.listen) {
    await tauri.event.listen("vpn-action-result", ({ payload }) => {
      const detail = payload?.output?.trim().split("\n").slice(-1)[0] || "";
      addEvent(payload?.success ? t("actionDone") : t("actionFailed"), detail,
        payload?.success ? "success" : "error");
      showOperationResult(payload, payload?.action || t("diagnosticOutput"));
      showToast(payload?.success ? t("actionDone") : `${t("actionFailed")}: ${detail}`, !payload?.success);
      if (payload?.action === "doctor" || !payload?.success) showPage("diagnostics");
      refreshStatus(false);
    });
  }

  if (invoke) {
    const platform = await invoke("get_platform_info");
    $("#platform-note").textContent = platform?.functional ? "" : t("preview");
  }
  await Promise.all([refreshStatus(false), refreshProfiles(false), refreshInstallation()]);
  setInterval(() => refreshStatus(false), 5000);
});
