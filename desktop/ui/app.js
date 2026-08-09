// Copyright (C) 2026 Nik m (@mazurovn)
// SPDX-License-Identifier: AGPL-3.0-or-later

const tauri = window.__TAURI__;
const invoke = tauri?.core?.invoke;

const translations = {
  ru: {
    language: "Язык", brandNetwork: "СЕТЬ ДЛЯ ЛЮДЕЙ И ИИ", brandEngine: "АВТОЛЕЧЕНИЕ",
    liveStatus: "РАБОТАЕТ", checking: "Проверяем соединение", waiting: "Ожидаем данные Mazzy VPN CLI",
    protocol: "Протокол", interface: "Интерфейс", handshake: "Рукопожатие", publicIp: "Публичный IP",
    quickConnect: "Быстро подключить", useDefault: "Профиль по умолчанию", reconnect: "Переподключить",
    safeRestart: "Безопасный перезапуск", disconnect: "Отключить", stopTunnel: "Остановить туннель",
    doctor: "Самодиагностика", checkSystem: "Проверить систему", protection: "ЗАЩИТА",
    systemHealth: "Состояние системы", checkingShort: "ПРОВЕРКА", autoConnect: "Автоподключение",
    startsOnBoot: "Запуск при загрузке", healthMonitor: "Монитор состояния",
    autoHealing: "Автопроверка и лечение", fallback: "Резервный VPN",
    conflictGuard: "Защита от конфликтов", configurations: "КОНФИГУРАЦИИ",
    vpnProfiles: "VPN-профили", activity: "АКТИВНОСТЬ", eventLog: "События",
    noEvents: "Событий пока нет", neverUpdated: "Данные ещё не получены",
    connected: "VPN защищает соединение", connecting: "VPN подключается", disconnected: "VPN отключён",
    degraded: "Соединение требует внимания", internetOk: "Интернет доступен через защищённый туннель",
    internetDown: "Туннель без подтверждённого доступа в интернет", noProfile: "Профиль не выбран",
    enabled: "ВКЛ", disabled: "ВЫКЛ", inactive: "НЕТ", active: "АКТИВЕН", healthy: "ИСПРАВНО",
    attention: "ATTENTION", secondsAgo: "с назад", notReceived: "не получен", cacheStale: "Данные устарели",
    preview: "Preview: VPN backend этой ОС ещё не реализован", actionStarted: "Выполняется",
    actionDone: "Действие завершено", actionFailed: "Действие не выполнено", refreshed: "Статус обновлён",
    hidden: "скрыт", statusChanged: "Состояние VPN изменилось", update: "Обновлено"
  },
  en: {
    language: "Language", brandNetwork: "AI-READY NETWORK", brandEngine: "SELF-HEALING CORE",
    liveStatus: "LIVE", checking: "Checking connection", waiting: "Waiting for Mazzy VPN CLI data",
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
    language: "Sprache", brandNetwork: "KI-BEREITES NETZ", brandEngine: "SELBSTHEILUNG",
    liveStatus: "AKTIV", checking: "Verbindung wird geprüft", waiting: "Warte auf Mazzy VPN CLI",
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
    language: "语言", brandNetwork: "面向 AI 的网络", brandEngine: "自恢复核心",
    liveStatus: "运行中", checking: "正在检查连接", waiting: "正在等待 Mazzy VPN CLI 数据",
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
    language: "言語", brandNetwork: "AI 対応ネットワーク", brandEngine: "自己修復コア",
    liveStatus: "稼働中", checking: "接続を確認中", waiting: "Mazzy VPN CLI のデータを待っています",
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
    language: "언어", brandNetwork: "AI 지원 네트워크", brandEngine: "자가 복구 코어",
    liveStatus: "작동 중", checking: "연결 확인 중", waiting: "Mazzy VPN CLI 데이터를 기다리는 중",
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
  navDashboard: "Обзор", navProfiles: "Профили", navDiagnostics: "Диагностика",
  navAgents: "Агенты", navSettings: "Настройки", navAbout: "О программе",
  agentsCenter: "УПРАВЛЕНИЕ АГЕНТАМИ", agentsTitle: "Доступ к AI-агентам",
  agentsHint: "Диагностика локальных кандидатов и готовности будущих каналов управления.",
  refreshAgents: "Обновить", providerAdapters: "АДАПТЕРЫ ПРОВАЙДЕРОВ",
  localAgents: "Локальные агенты", agentsLoading: "Проверяем агенты…",
  remoteAccess: "СТАТУС УПРАВЛЕНИЯ", diagnosticsOnlyTitle: "Только диагностика",
  embeddedAgentClient: "Встроенный клиент",
  firstPartyGateway: "Mazzy gateway", telegramIngress: "Telegram",
  vendorRemoteBoundary: "Запуск, сопряжение и остановка отключены до native approval, trusted executable resolution и process-tree containment.",
  transportReadiness: "ГОТОВНОСТЬ ТРАНСПОРТОВ", mazzyControlPaths: "Каналы Mazzy Agent Control",
  runtimeReady: "ГОТОВ", runtimeUnavailable: "НЕТ RUNTIME", gatewayPlanned: "ЗАПЛАНИРОВАНО",
  providerInstalled: "установлен", providerMissing: "не установлен", daemonRunning: "работает",
  daemonStopped: "остановлен", remoteSupported: "remote control доступен",
  remoteUnsupported: "remote control недоступен", discoveryOnly: "ТОЛЬКО ОБНАРУЖЕНИЕ",
  profileLibrary: "БИБЛИОТЕКА ПРОФИЛЕЙ", profileManagement: "Профили и локации",
  profileManagementHint: "Импорт, проверка, выбор и безопасный тест конфигураций.",
  importFiles: "Загрузить файлы", scanFolder: "Проверить папку", importFolder: "Импортировать папку",
  searchProfiles: "Поиск профиля или локации", allProtocols: "Все протоколы",
  profilesLoading: "Загружаем профили…", profilesEmpty: "Профили не найдены",
  profilesUnavailable: "Кэш профилей недоступен. Обновите движок или запустите диагностику.",
  selectedProfile: "Выбран по умолчанию", connectProfile: "Подключить", testProfile: "Проверить", removeProfile: "Удалить",
  testingTools: "ТЕСТИРОВАНИЕ", testingTitle: "Проверка конфигураций",
  testTimeout: "Лимит времени живого теста", testProtocol: "Протокол",
  validateProfiles: "Проверить формат", probeEndpoints: "DNS и ping сервера",
  testAllProfiles: "Проверить все профили с откатом", emergencyMode: "Аварийный выбор рабочего VPN",
  rollbackHint: "Живые тесты временно меняют маршрут и всегда используют лимит времени и откат.",
  diagnosticsCenter: "ДИАГНОСТИКА", diagnosticsTitle: "Самодиагностика, тесты и журнал",
  diagnosticsHint: "Полный результат больше не скрывается: вывод сохраняется в этой панели.",
  runDoctor: "Запустить самодиагностику", repairSystem: "Установить/исправить",
  connectionDiagnose: "Диагностика соединения",
  connectionDiagnoseHint: "Маршрут, DNS, туннель, рукопожатие и интернет",
  offlineSelfTest: "Проверка без подключения", offlineSelfTestHint: "Формат, зависимости, службы и права",
  liveSelfTest: "Проверка реального туннеля", liveSelfTestHint: "Проверка туннелей с откатом",
  serviceLog: "Журнал сервиса", serviceLogHint: "Последние события без секретов", logLines: "Строк журнала",
  outputReady: "Готово к проверке", outputRunning: "Выполняется", outputSuccess: "Успешно",
  outputFailed: "Есть ошибки", diagnosticOutput: "Результат", clearOutput: "Очистить",
  outputPlaceholder: "Здесь появится полный вывод Doctor, тестов и журнала.",
  settingsCenter: "НАСТРОЙКИ", settingsTitle: "Установка и системные настройки",
  settingsHint: "DEB/RPM управляют движком через менеджер пакетов; AppImage использует явную установку.",
  installRepair: "Установить / обновить / исправить", installation: "УСТАНОВКА",
  engineReadiness: "Готовность движка", installedVersion: "Установлено",
  bundledVersion: "В комплекте Desktop", installationType: "Тип установки",
  packageManaged: "ЧЕРЕЗ ПАКЕТНЫЙ МЕНЕДЖЕР", embeddedInstall: "ВСТРОЕННАЯ / РУЧНАЯ",
  engineService: "Системная служба",
  recoveryMonitor: "Монитор восстановления", localApi: "Защищённый локальный API",
  apiReady: "АКТИВЕН", services: "СЛУЖБЫ", serviceControl: "Управление службами",
  notifications: "Уведомления", notificationsHint: "Запланировано; в этой версии недоступно",
  privacyMode: "Скрывать публичный IP", privacyModeHint: "Применяется к экрану обзора",
  checkUpdates: "Проверить обновления", automaticUpdateChecks: "Автопроверка обновлений",
  automaticUpdateChecksHint: "Показывает диалог; ничего не устанавливает без подтверждения",
  updateAvailable: "Доступно обновление", currentDesktopVersion: "Текущая версия",
  availableDesktopVersion: "Новая версия", signedUpdate: "ПОДПИСАННОЕ ОБНОВЛЕНИЕ",
  signedUpdateHint: "Артефакт проверяется встроенным публичным ключом. AppImage обновляет Desktop; системный engine при необходимости обновляется отдельно через Install/Repair.",
  updateDialogInstallHint: "Версия {version} готова к безопасной загрузке. Установка начнётся только после вашего подтверждения.",
  updateDialogPackageHint: "Версия {version} доступна. Эта установка управляется DEB/RPM, поэтому пакет нужно выбрать на странице релиза.",
  updateLater: "Позже", installUpdate: "Скачать и установить", openRelease: "Открыть релиз",
  checkingForUpdates: "Проверяем обновления", upToDate: "Установлена последняя версия",
  updateDownloading: "Загрузка подписанного обновления", updateInstalling: "Установка обновления",
  updateInstalled: "Обновление установлено",
  restartNow: "Перезапустить", updateCheckFailed: "Не удалось проверить обновления",
  installed: "УСТАНОВЛЕНО", missing: "НЕТ", updateRequired: "НУЖНО ОБНОВИТЬ",
  ready: "ГОТОВО", confirmLiveTest: "Живой тест временно изменит VPN-маршрут и затем выполнит rollback. Продолжить?",
  confirmRemove: "Удалить выбранный VPN-профиль?", confirmRepair: "Запустить системную установку и исправление зависимостей?",
  profilesRefreshed: "Список профилей обновлён", noSelection: "Ничего не выбрано",
  checkAllLocations: "Проверить все локации", checkingLocations: "Проверяем локации",
  locationReachable: "Сервер доступен", locationUnknown: "Доступность не подтверждена",
  locationUnreachable: "Сервер недоступен", locationInvalid: "Некорректный адрес сервера",
  locationNotChecked: "Доступность ещё не проверялась", activeTunnel: "АКТИВЕН",
  lastChecked: "Последняя проверка",
  endpointProbeHint: "Проверка измеряет DNS/ICMP/TCP доступность сервера, но не подтверждает VPN-авторизацию и маршрутизацию.",
  connectFastest: "Подключить самую быструю", fastestUnavailable: "Сначала проверьте локации; доступный ping не найден",
  sortRecommended: "Рекомендуемые", sortLatency: "По ping", sortStatus: "По доступности", sortName: "По имени",
  realVerification: "ПРОВЕРКА СЕТЕВОГО EGRESS", realVerificationTitle: "Фактический маршрут и локация",
  realVerificationHint: "Сравнивает исходящий маршрут с туннелем, два источника геолокации, DNS и IPv6.",
  notVerified: "НЕ ПРОВЕРЕНО", verifyNow: "Проверить сетевой egress", verifyWithSpeed: "Проверить egress + скорость",
  observedLocation: "Фактическая локация", profileLocationMatch: "Совпадение с профилем",
  systemEgress: "Системный IPv4 egress", ipv6Leak: "IPv6-утечка", dnsRouting: "DNS-маршрут",
  speedSample: "Контрольная проверка скорости", verified: "СЕТЕВОЙ EGRESS ПОДТВЕРЖДЁН", warning: "ЕСТЬ РИСКИ",
  partiallyVerified: "ПРОВЕРЕНО ЧАСТИЧНО", profileCountryMissing: "СТРАНА ПРОФИЛЯ НЕ УКАЗАНА",
  failed: "НЕ РАБОТАЕТ", matches: "СОВПАДАЕТ", mismatch: "НЕ СОВПАДАЕТ", unknown: "НЕИЗВЕСТНО",
  sameEgress: "Через VPN", differentEgress: "Маршрут отличается", noLeak: "Не обнаружена",
  potentialLeak: "ВОЗМОЖНА", fullTunnelDns: "Через VPN", partialDns: "Не подтверждён полностью",
  speedNotRun: "Не запускался", verificationDisclaimer: "Сайты также могут учитывать регион аккаунта, cookies, язык браузера, WebRTC и геолокацию устройства.",
  startService: "Запустить", restartService: "Перезапустить", stopService: "Остановить",
  currentState: "Сейчас",
  aboutProduct: "О ПРОДУКТЕ", aboutLead: "AI-ready VPN для устойчивой работы с ИИ, открытым вебом, видео и корпоративными ресурсами.",
  aboutPurpose: "НАЗНАЧЕНИЕ", aboutPurposeTitle: "Открытый и самовосстанавливающийся VPN-контур",
  aboutPurposeText: "Mazzy VPN управляет AmneziaWG, WireGuard, OpenVPN и L2TP/IPsec, проверяет реальный egress и локации, восстанавливает соединение и помогает людям и AI-агентам сохранять стабильный сетевой доступ.",
  aboutPlatform: "Платформа", aboutAuthor: "АВТОР И СОПРОВОЖДЕНИЕ",
  aboutAuthorText: "Проект создан и сопровождается Nik m. Оригинальное авторство и уведомления об авторских правах должны сохраняться.",
  aboutCopyright: "Авторское право", aboutLicense: "Лицензия",
  aboutRules: "ПРАВИЛА И БЕЗОПАСНОСТЬ", aboutRulesTitle: "Прозрачная работа без скрытых действий",
  aboutRuleLocal: "Профили, ключи и журналы остаются на устройстве; телеметрии нет.",
  aboutRulePrivilege: "Системные изменения выполняются только после стандартного разрешения ОС.",
  aboutRuleTests: "Live-тесты временно меняют маршруты и обязаны завершаться rollback.",
  aboutRuleLicense: "Изменённые сетевые версии должны соблюдать условия AGPL и предоставлять исходный код.",
  aboutRuleIdentity: "Нельзя выдавать изменённую сборку за оригинальный релиз или искажать авторство.",
  aboutRuleProvider: "Пользователь отвечает за законность VPN-провайдера и импортируемых конфигураций.",
  aboutDocuments: "Документы:"
});

Object.assign(translations.en, {
  navDashboard: "Dashboard", navProfiles: "Profiles", navDiagnostics: "Diagnostics",
  navAgents: "Agents", navSettings: "Settings", navAbout: "About",
  agentsCenter: "AGENT CONTROL", agentsTitle: "AI agent access",
  agentsHint: "Diagnostics for local candidates and future control-channel readiness.",
  refreshAgents: "Refresh", providerAdapters: "PROVIDER ADAPTERS", localAgents: "Local agents",
  agentsLoading: "Checking agents…", remoteAccess: "CONTROL STATUS",
  diagnosticsOnlyTitle: "Diagnostics only",
  embeddedAgentClient: "Embedded client", firstPartyGateway: "Mazzy gateway",
  telegramIngress: "Telegram", vendorRemoteBoundary: "Start, pair and stop are disabled until native approval, trusted executable resolution and process-tree containment exist.",
  transportReadiness: "TRANSPORT READINESS", mazzyControlPaths: "Mazzy Agent Control paths",
  runtimeReady: "READY", runtimeUnavailable: "NO RUNTIME", gatewayPlanned: "PLANNED",
  providerInstalled: "installed", providerMissing: "not installed", daemonRunning: "running",
  daemonStopped: "stopped", remoteSupported: "remote control available",
  remoteUnsupported: "remote control unavailable", discoveryOnly: "DISCOVERY ONLY",
  profileLibrary: "PROFILE LIBRARY", profileManagement: "Profiles and locations",
  profileManagementHint: "Import, validate, select and safely test configurations.",
  importFiles: "Import files", scanFolder: "Scan folder", importFolder: "Import folder",
  searchProfiles: "Search profile or location", allProtocols: "All protocols",
  profilesLoading: "Loading profiles…", profilesEmpty: "No profiles found",
  profilesUnavailable: "The profile cache is unavailable. Update the engine or run diagnostics.",
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
  settingsHint: "DEB/RPM manage the engine as a package; AppImage uses explicit bootstrap.",
  installRepair: "Install / update / repair", installation: "INSTALLATION", engineReadiness: "Engine readiness",
  installedVersion: "Installed", bundledVersion: "Bundled with Desktop",
  installationType: "Installation type", packageManaged: "PACKAGE-MANAGED",
  embeddedInstall: "EMBEDDED / MANUAL", engineService: "System service",
  recoveryMonitor: "Recovery monitor", localApi: "Protected local API", apiReady: "ACTIVE",
  services: "SERVICES", serviceControl: "Service control",
  notifications: "Notifications", notificationsHint: "Planned; unavailable in this preview",
  privacyMode: "Hide public IP", privacyModeHint: "Applied to Dashboard",
  checkUpdates: "Check for updates", automaticUpdateChecks: "Automatic update checks",
  automaticUpdateChecksHint: "Shows a dialog; nothing is installed without confirmation",
  updateAvailable: "Update available", currentDesktopVersion: "Current version",
  availableDesktopVersion: "New version", signedUpdate: "SIGNED UPDATE",
  signedUpdateHint: "The artifact is verified with the embedded public key. AppImage updates Desktop; update the system engine separately through Install/Repair when requested.",
  updateDialogInstallHint: "Version {version} is ready for a secure download. Installation starts only after you confirm.",
  updateDialogPackageHint: "Version {version} is available. This installation is managed by DEB/RPM, so choose the package on the release page.",
  updateLater: "Later", installUpdate: "Download and install", openRelease: "Open release",
  checkingForUpdates: "Checking for updates", upToDate: "You have the latest version",
  updateDownloading: "Downloading signed update", updateInstalling: "Installing update",
  updateInstalled: "Update installed",
  restartNow: "Restart", updateCheckFailed: "Unable to check for updates",
  installed: "INSTALLED", missing: "MISSING", updateRequired: "UPDATE REQUIRED", ready: "READY",
  confirmLiveTest: "The live test temporarily changes VPN routes and then rolls back. Continue?",
  confirmRemove: "Remove the selected VPN profile?", confirmRepair: "Run system installation and dependency repair?",
  profilesRefreshed: "Profile list refreshed", noSelection: "Nothing selected",
  checkAllLocations: "Check all locations", checkingLocations: "Checking locations",
  locationReachable: "Endpoint reachable", locationUnknown: "Reachability unconfirmed",
  locationUnreachable: "Endpoint unreachable", locationInvalid: "Invalid endpoint",
  locationNotChecked: "Reachability has not been checked", activeTunnel: "ACTIVE",
  lastChecked: "Last checked",
  endpointProbeHint: "The probe checks DNS/ICMP/TCP reachability; it does not prove VPN authentication or routing.",
  connectFastest: "Connect fastest", fastestUnavailable: "Check locations first; no reachable latency result is available",
  sortRecommended: "Recommended", sortLatency: "By ping", sortStatus: "By availability", sortName: "By name",
  realVerification: "NETWORK EGRESS VERIFICATION", realVerificationTitle: "Actual route and location",
  realVerificationHint: "Compares system egress with the tunnel, two geolocation sources, DNS and IPv6.",
  notVerified: "NOT VERIFIED", verifyNow: "Verify network egress", verifyWithSpeed: "Verify egress + speed",
  observedLocation: "Observed location", profileLocationMatch: "Profile location match",
  systemEgress: "System IPv4 egress", ipv6Leak: "IPv6 leak", dnsRouting: "DNS routing",
  speedSample: "Bounded speed sample", verified: "NETWORK EGRESS VERIFIED", warning: "RISKS FOUND",
  partiallyVerified: "PARTIALLY VERIFIED", profileCountryMissing: "PROFILE COUNTRY NOT SET",
  failed: "FAILED", matches: "MATCH", mismatch: "MISMATCH", unknown: "UNKNOWN",
  sameEgress: "Through VPN", differentEgress: "Route differs", noLeak: "Not detected",
  potentialLeak: "POTENTIAL", fullTunnelDns: "Through VPN", partialDns: "Not fully confirmed",
  speedNotRun: "Not run", verificationDisclaimer: "Sites may also use account region, cookies, browser language, WebRTC and device location.",
  startService: "Start", restartService: "Restart", stopService: "Stop", currentState: "Current",
  aboutProduct: "ABOUT THE PRODUCT", aboutLead: "An AI-ready VPN for resilient access to AI, the open web, video and corporate resources.",
  aboutPurpose: "PURPOSE", aboutPurposeTitle: "An open, self-healing VPN control plane",
  aboutPurposeText: "Mazzy VPN manages AmneziaWG, WireGuard, OpenVPN and L2TP/IPsec, verifies actual egress and locations, recovers connections, and helps people and AI agents keep stable network access.",
  aboutPlatform: "Platform", aboutAuthor: "AUTHOR AND MAINTAINER",
  aboutAuthorText: "The project was created and is maintained by Nik m. Original authorship and copyright notices must be preserved.",
  aboutCopyright: "Copyright", aboutLicense: "License",
  aboutRules: "RULES AND SECURITY", aboutRulesTitle: "Transparent operation without hidden actions",
  aboutRuleLocal: "Profiles, keys and logs stay on the device; no telemetry is collected.",
  aboutRulePrivilege: "System changes run only after the operating system's standard authorization.",
  aboutRuleTests: "Live tests temporarily change routes and must finish with rollback.",
  aboutRuleLicense: "Modified network versions must follow the AGPL and provide corresponding source code.",
  aboutRuleIdentity: "A modified build must not be presented as an original release or misrepresent authorship.",
  aboutRuleProvider: "The user is responsible for the legality of their VPN provider and imported configurations.",
  aboutDocuments: "Documents:"
});

Object.assign(translations.de, {
  navDashboard: "Übersicht", navProfiles: "Profile", navDiagnostics: "Diagnose",
  navSettings: "Einstellungen", navAbout: "Über",
  checkAllLocations: "Alle Standorte prüfen", lastChecked: "Zuletzt geprüft",
  connectFastest: "Schnellsten verbinden", fastestUnavailable: "Zuerst Standorte prüfen; kein erreichbarer Ping verfügbar",
  sortRecommended: "Empfohlen", sortLatency: "Nach Ping", sortStatus: "Nach Erreichbarkeit", sortName: "Nach Name",
  realVerification: "ECHTE VPN-PRÜFUNG", realVerificationTitle: "Tatsächliche Route und Standort",
  realVerificationHint: "Vergleicht System-Egress und Tunnel, zwei Geoquellen, DNS und IPv6.",
  notVerified: "NICHT GEPRÜFT", verifyNow: "VPN prüfen", verifyWithSpeed: "Prüfen + Geschwindigkeit",
  observedLocation: "Ermittelter Standort", profileLocationMatch: "Profilstandort stimmt",
  systemEgress: "System-IPv4-Egress", ipv6Leak: "IPv6-Leck", dnsRouting: "DNS-Route",
  speedSample: "Begrenzter Geschwindigkeitstest", verified: "NETZWERK-EGRESS BESTÄTIGT", warning: "RISIKEN GEFUNDEN",
  partiallyVerified: "TEILWEISE BESTÄTIGT", profileCountryMissing: "PROFILLAND FEHLT",
  failed: "FEHLGESCHLAGEN", matches: "STIMMT", mismatch: "ABWEICHUNG", unknown: "UNBEKANNT",
  sameEgress: "Über VPN", differentEgress: "Route weicht ab", noLeak: "Nicht erkannt",
  potentialLeak: "MÖGLICH", fullTunnelDns: "Über VPN", partialDns: "Nicht vollständig bestätigt",
  speedNotRun: "Nicht gestartet", verificationDisclaimer: "Websites können auch Kontoregion, Cookies, Browsersprache, WebRTC und Gerätestandort verwenden.",
  startService: "Starten", restartService: "Neu starten", stopService: "Stoppen", currentState: "Aktuell",
  checkUpdates: "Nach Updates suchen", automaticUpdateChecks: "Automatische Update-Prüfung",
  automaticUpdateChecksHint: "Zeigt einen Dialog; ohne Bestätigung wird nichts installiert",
  updateAvailable: "Update verfügbar", currentDesktopVersion: "Aktuelle Version",
  availableDesktopVersion: "Neue Version", signedUpdate: "SIGNIERTES UPDATE",
  signedUpdateHint: "Das Artefakt wird mit dem eingebetteten öffentlichen Schlüssel geprüft. AppImage aktualisiert Desktop; die System-Engine wird bei Bedarf separat über Install/Repair aktualisiert.",
  updateDialogInstallHint: "Version {version} kann sicher geladen werden. Die Installation beginnt erst nach Ihrer Bestätigung.",
  updateDialogPackageHint: "Version {version} ist verfügbar. Diese Installation wird von DEB/RPM verwaltet; wählen Sie das Paket auf der Release-Seite.",
  updateLater: "Später", installUpdate: "Herunterladen und installieren", openRelease: "Release öffnen",
  checkingForUpdates: "Updates werden geprüft", upToDate: "Die neueste Version ist installiert",
  updateDownloading: "Signiertes Update wird geladen", updateInstalling: "Update wird installiert",
  updateInstalled: "Update installiert",
  restartNow: "Neu starten", updateCheckFailed: "Update-Prüfung fehlgeschlagen"
});

Object.assign(translations.zh, {
  navDashboard: "概览", navProfiles: "配置", navDiagnostics: "诊断",
  navSettings: "设置", navAbout: "关于",
  checkAllLocations: "检查所有位置", lastChecked: "上次检查",
  connectFastest: "连接最快节点", fastestUnavailable: "请先检查位置；没有可用的延迟结果",
  sortRecommended: "推荐", sortLatency: "按延迟", sortStatus: "按可用性", sortName: "按名称",
  realVerification: "真实 VPN 验证", realVerificationTitle: "实际出口与位置",
  realVerificationHint: "比较系统出口和隧道、两个地理位置来源、DNS 与 IPv6。",
  notVerified: "未验证", verifyNow: "验证 VPN", verifyWithSpeed: "验证并测速",
  observedLocation: "实际位置", profileLocationMatch: "与配置位置匹配",
  systemEgress: "系统 IPv4 出口", ipv6Leak: "IPv6 泄漏", dnsRouting: "DNS 路由",
  speedSample: "有限测速样本", verified: "网络出口已验证", warning: "发现风险",
  partiallyVerified: "部分验证", profileCountryMissing: "配置未设置国家",
  failed: "失败", matches: "匹配", mismatch: "不匹配", unknown: "未知",
  sameEgress: "通过 VPN", differentEgress: "路由不同", noLeak: "未发现",
  potentialLeak: "可能泄漏", fullTunnelDns: "通过 VPN", partialDns: "未完全确认",
  speedNotRun: "未运行", verificationDisclaimer: "网站还可能使用账户地区、Cookie、浏览器语言、WebRTC 和设备位置。",
  startService: "启动", restartService: "重启", stopService: "停止", currentState: "当前",
  checkUpdates: "检查更新", automaticUpdateChecks: "自动检查更新",
  automaticUpdateChecksHint: "仅显示确认对话框；未经确认不会安装",
  updateAvailable: "有可用更新", currentDesktopVersion: "当前版本",
  availableDesktopVersion: "新版本", signedUpdate: "已签名更新",
  signedUpdateHint: "更新包会使用内置公钥验证。AppImage 更新 Desktop；如有提示，请通过 Install/Repair 单独更新系统引擎。",
  updateDialogInstallHint: "版本 {version} 可安全下载。只有确认后才会开始安装。",
  updateDialogPackageHint: "版本 {version} 可用。此安装由 DEB/RPM 管理，请在发布页面选择软件包。",
  updateLater: "稍后", installUpdate: "下载并安装", openRelease: "打开发布页面",
  checkingForUpdates: "正在检查更新", upToDate: "已安装最新版本",
  updateDownloading: "正在下载已签名更新", updateInstalling: "正在安装更新",
  updateInstalled: "更新已安装",
  restartNow: "重新启动", updateCheckFailed: "无法检查更新"
});

Object.assign(translations.ja, {
  navDashboard: "概要", navProfiles: "プロファイル", navDiagnostics: "診断",
  navSettings: "設定", navAbout: "このアプリについて",
  checkAllLocations: "すべての場所を確認", lastChecked: "最終確認",
  connectFastest: "最速へ接続", fastestUnavailable: "先に場所を確認してください。利用可能な ping がありません",
  sortRecommended: "おすすめ", sortLatency: "Ping 順", sortStatus: "到達性順", sortName: "名前順",
  realVerification: "実際の VPN 検証", realVerificationTitle: "実際の出口と場所",
  realVerificationHint: "システム出口とトンネル、2つの位置情報、DNS、IPv6 を比較します。",
  notVerified: "未検証", verifyNow: "VPN を検証", verifyWithSpeed: "検証 + 速度",
  observedLocation: "検出された場所", profileLocationMatch: "プロファイル位置との一致",
  systemEgress: "システム IPv4 出口", ipv6Leak: "IPv6 リーク", dnsRouting: "DNS ルート",
  speedSample: "制限付き速度サンプル", verified: "ネットワーク出口を確認済み", warning: "リスクあり",
  partiallyVerified: "一部確認済み", profileCountryMissing: "プロファイルの国が未設定",
  failed: "失敗", matches: "一致", mismatch: "不一致", unknown: "不明",
  sameEgress: "VPN 経由", differentEgress: "ルートが異なる", noLeak: "検出なし",
  potentialLeak: "可能性あり", fullTunnelDns: "VPN 経由", partialDns: "完全には未確認",
  speedNotRun: "未実行", verificationDisclaimer: "サイトはアカウント地域、Cookie、ブラウザー言語、WebRTC、端末位置も使用できます。",
  startService: "開始", restartService: "再起動", stopService: "停止", currentState: "現在",
  checkUpdates: "更新を確認", automaticUpdateChecks: "更新の自動確認",
  automaticUpdateChecksHint: "確認ダイアログを表示し、許可なくインストールしません",
  updateAvailable: "更新があります", currentDesktopVersion: "現在のバージョン",
  availableDesktopVersion: "新しいバージョン", signedUpdate: "署名済み更新",
  signedUpdateHint: "更新ファイルは内蔵の公開鍵で検証します。AppImage は Desktop を更新し、必要な system engine は Install/Repair で別途更新します。",
  updateDialogInstallHint: "バージョン {version} を安全にダウンロードできます。確認後にのみインストールします。",
  updateDialogPackageHint: "バージョン {version} があります。このインストールは DEB/RPM 管理のため、リリースページでパッケージを選択してください。",
  updateLater: "後で", installUpdate: "ダウンロードしてインストール", openRelease: "リリースを開く",
  checkingForUpdates: "更新を確認中", upToDate: "最新バージョンです",
  updateDownloading: "署名済み更新をダウンロード中", updateInstalling: "更新をインストール中",
  updateInstalled: "更新をインストールしました",
  restartNow: "再起動", updateCheckFailed: "更新を確認できませんでした"
});

Object.assign(translations.ko, {
  navDashboard: "개요", navProfiles: "프로필", navDiagnostics: "진단",
  navSettings: "설정", navAbout: "정보",
  checkAllLocations: "모든 위치 확인", lastChecked: "마지막 확인",
  connectFastest: "가장 빠른 위치 연결", fastestUnavailable: "먼저 위치를 확인하세요. 사용 가능한 ping 결과가 없습니다",
  sortRecommended: "추천", sortLatency: "Ping 순", sortStatus: "도달성 순", sortName: "이름 순",
  realVerification: "실제 VPN 검증", realVerificationTitle: "실제 출구와 위치",
  realVerificationHint: "시스템 출구와 터널, 두 위치 정보 제공자, DNS 및 IPv6를 비교합니다.",
  notVerified: "검증 안 됨", verifyNow: "VPN 검증", verifyWithSpeed: "검증 + 속도",
  observedLocation: "관측 위치", profileLocationMatch: "프로필 위치 일치",
  systemEgress: "시스템 IPv4 출구", ipv6Leak: "IPv6 유출", dnsRouting: "DNS 경로",
  speedSample: "제한된 속도 샘플", verified: "네트워크 이그레스 확인됨", warning: "위험 발견",
  partiallyVerified: "부분 확인됨", profileCountryMissing: "프로필 국가 미설정",
  failed: "실패", matches: "일치", mismatch: "불일치", unknown: "알 수 없음",
  sameEgress: "VPN 경유", differentEgress: "경로 다름", noLeak: "감지 안 됨",
  potentialLeak: "가능성 있음", fullTunnelDns: "VPN 경유", partialDns: "완전히 확인되지 않음",
  speedNotRun: "실행 안 함", verificationDisclaimer: "사이트는 계정 지역, 쿠키, 브라우저 언어, WebRTC 및 기기 위치도 사용할 수 있습니다.",
  startService: "시작", restartService: "다시 시작", stopService: "중지", currentState: "현재",
  checkUpdates: "업데이트 확인", automaticUpdateChecks: "자동 업데이트 확인",
  automaticUpdateChecksHint: "확인 대화상자만 표시하며 승인 없이 설치하지 않습니다",
  updateAvailable: "업데이트 사용 가능", currentDesktopVersion: "현재 버전",
  availableDesktopVersion: "새 버전", signedUpdate: "서명된 업데이트",
  signedUpdateHint: "업데이트 파일은 내장 공개 키로 검증합니다. AppImage는 Desktop을 업데이트하며 필요한 시스템 엔진은 Install/Repair에서 별도로 업데이트합니다.",
  updateDialogInstallHint: "버전 {version}을 안전하게 다운로드할 수 있습니다. 확인한 후에만 설치합니다.",
  updateDialogPackageHint: "버전 {version}을 사용할 수 있습니다. 이 설치는 DEB/RPM으로 관리되므로 릴리스 페이지에서 패키지를 선택하세요.",
  updateLater: "나중에", installUpdate: "다운로드 및 설치", openRelease: "릴리스 열기",
  checkingForUpdates: "업데이트 확인 중", upToDate: "최신 버전이 설치되어 있습니다",
  updateDownloading: "서명된 업데이트 다운로드 중", updateInstalling: "업데이트 설치 중",
  updateInstalled: "업데이트 설치됨",
  restartNow: "다시 시작", updateCheckFailed: "업데이트를 확인할 수 없습니다"
});

const $ = (selector) => document.querySelector(selector);
const supportedLanguages = new Set(["ru", "en", "de", "zh", "ja", "ko"]);
// Keep startup recovery finite: the engine normally becomes ready within this window.
const profileRetryDelays = [500, 1000, 2000, 4000, 8000, 12000];
const previewParameters = new URLSearchParams(window.location.search);
const localDocumentationHosts = new Set(["127.0.0.1", "localhost", "[::1]"]);
const documentationPreview =
  !window.__TAURI_INTERNALS__ &&
  window.location.protocol === "http:" &&
  localDocumentationHosts.has(window.location.hostname) &&
  previewParameters.get("preview") === "docs";
const previewLanguage = previewParameters.get("lang");
const state = {
  lang: documentationPreview && supportedLanguages.has(previewLanguage)
    ? previewLanguage
    : localStorage.getItem("mazzy-language") || "ru",
  hideIp: documentationPreview ? false : localStorage.getItem("mazzy-hide-ip") !== "false",
  autoUpdateChecks: localStorage.getItem("mazzy-auto-update-check") !== "false",
  page: "dashboard",
  status: null,
  profiles: [],
  profileCacheAvailable: true,
  profileHealth: new Map(),
  probeCheckedAt: null,
  probeRunningScope: null,
  verification: null,
  installation: null,
  agentIntegrations: null,
  platformInfo: null,
  updateInfo: null,
  updateChecking: false,
  updateInstalling: false,
  updateDownloadFinished: false,
  updateInstalled: false,
  updateDownloaded: 0,
  updateTotal: null,
  lastSignature: "",
  lastActiveProfileSignature: "",
  engineStartupAttempted: false,
  events: [],
  busy: false,
  lastOperation: null
};

let profileRefreshPromise = null;
let profileRetryTimer = null;
let profileRetryAttempt = 0;
let profileRetryExhausted = false;

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
  renderUpdateDialog();
  renderProbeCheckedAt();
  renderVerification();
  if (state.installation) renderInstallation(state.installation);
  if (state.agentIntegrations) renderAgentIntegrations(state.agentIntegrations);
  renderAbout();
}

function updateMessage(key, version) {
  return t(key).replace("{version}", version || "—");
}

function renderUpdateDialog() {
  const dialog = $("#update-dialog");
  const info = state.updateInfo;
  if (!dialog || !info) return;
  $("#update-current-version").textContent = info.current_version || "—";
  $("#update-available-version").textContent = info.version || "—";
  $("#update-dialog-hint").textContent = updateMessage(
    info.install_supported ? "updateDialogInstallHint" : "updateDialogPackageHint",
    info.version
  );
  const action = $("#update-action-button");
  action.disabled = state.updateInstalling;
  action.textContent = state.updateInstalled ? t("restartNow")
    : info.install_supported ? t("installUpdate") : t("openRelease");
  $("#update-later-button").disabled = state.updateInstalling;

  const progress = $("#update-progress");
  progress.hidden = !state.updateInstalling && !state.updateInstalled;
  const bar = $("#update-progress-bar");
  if (state.updateTotal && state.updateTotal > 0) {
    bar.max = state.updateTotal;
    bar.value = Math.min(state.updateDownloaded, state.updateTotal);
  } else if (state.updateInstalling) {
    bar.removeAttribute("value");
  } else {
    bar.max = 1;
    bar.value = 1;
  }
  $("#update-progress-label").textContent = state.updateInstalled
    ? t("updateInstalled")
    : state.updateDownloadFinished ? t("updateInstalling") : t("updateDownloading");
}

function showUpdateDialog(info) {
  state.updateInfo = info;
  state.updateInstalling = false;
  state.updateDownloadFinished = false;
  state.updateInstalled = false;
  state.updateDownloaded = 0;
  state.updateTotal = null;
  renderUpdateDialog();
  const dialog = $("#update-dialog");
  if (!dialog.open) dialog.showModal();
}

async function checkForUpdates(manual = false) {
  if (!invoke || state.updateChecking || state.updateInstalling) return;
  state.updateChecking = true;
  const button = $("#check-update-button");
  button.disabled = true;
  button.textContent = t("checkingForUpdates");
  try {
    const info = await invoke("check_for_update");
    if (info) showUpdateDialog(info);
    else if (manual) showToast(t("upToDate"));
  } catch (error) {
    if (manual) showToast(`${t("updateCheckFailed")}: ${error}`, true);
  } finally {
    state.updateChecking = false;
    button.disabled = false;
    button.textContent = t("checkUpdates");
  }
}

async function runPendingUpdateAction() {
  if (!invoke || !state.updateInfo || state.updateInstalling) return;
  if (state.updateInstalled) {
    await invoke("restart_application");
    return;
  }
  if (!state.updateInfo.install_supported) {
    try {
      await invoke("open_update_release");
      $("#update-dialog").close();
    } catch (error) {
      showToast(String(error), true);
    }
    return;
  }
  state.updateInstalling = true;
  state.updateDownloadFinished = false;
  state.updateDownloaded = 0;
  state.updateTotal = null;
  renderUpdateDialog();
  try {
    await invoke("install_update");
    state.updateInstalled = true;
    showToast(t("updateInstalled"));
  } catch (error) {
    showToast(String(error), true);
    state.updateDownloadFinished = false;
    state.updateDownloaded = 0;
    state.updateTotal = null;
  } finally {
    state.updateInstalling = false;
    renderUpdateDialog();
  }
}

function handleUpdateProgress(payload) {
  if (!payload || typeof payload.event !== "string") return;
  if (payload.event === "started") {
    state.updateTotal = Number(payload.content_length) || null;
  } else if (payload.event === "progress") {
    state.updateDownloaded += Math.max(0, Number(payload.chunk_length) || 0);
  } else if (payload.event === "finished") {
    state.updateDownloadFinished = true;
  } else if (payload.event === "installed") {
    state.updateInstalled = true;
  }
  renderUpdateDialog();
}

function setMiniState(selector, enabled, activeLabel = "enabled", inactiveLabel = "disabled") {
  const node = $(selector);
  node.textContent = enabled ? t(activeLabel) : t(inactiveLabel);
  node.className = `mini-state ${enabled ? "on" : ""}`;
}

function addEvent(title, detail = "", type = "success", output = detail) {
  state.events.unshift({ title, detail, type, output, time: new Date() });
  state.events = state.events.slice(0, 7);
  const list = $("#event-list");
  list.replaceChildren(...state.events.map((event) => {
    const row = document.createElement("button");
    row.type = "button";
    row.className = `event ${event.type}`;
    row.title = t("diagnosticOutput");
    row.addEventListener("click", () => {
      showOperationResult({
        success: event.type !== "error",
        action: event.title,
        output: event.output || event.detail
      }, event.title);
      showPage("diagnostics");
      $("#operation-output")?.focus();
    });
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
  if (!["dashboard", "profiles", "agents", "diagnostics", "settings", "about"].includes(page)) return;
  state.page = page;
  document.querySelectorAll("[data-page]").forEach((button) => {
    button.classList.toggle("active", button.dataset.page === page);
  });
  document.querySelectorAll("[data-page-panel]").forEach((panel) => {
    panel.classList.toggle("active", panel.dataset.pagePanel === page);
  });
  if (page === "agents" && invoke && !state.agentIntegrations) refreshAgents(false);
}

// Documentation-only browser fixture. Runtime profiles always come from get_profiles/API.
function documentationPreviewData() {
  if (!documentationPreview) throw new Error("Documentation fixture is disabled");
  const localizedLocation = {
    ru: ["Бельгия", "Брюссель"],
    en: ["Belgium", "Brussels"],
    de: ["Belgien", "Brüssel"],
    zh: ["比利时", "布鲁塞尔"],
    ja: ["ベルギー", "ブリュッセル"],
    ko: ["벨기에", "브뤼셀"]
  }[state.lang] || ["Belgium", "Brussels"];
  const [country, city] = localizedLocation;
  const checkedAt = new Date().toISOString();
  const profileNames = {
    ru: [
      "Бельгия — Брюссель", "Германия — Берлин", "Нидерланды — Амстердам",
      "Франция — Париж", "Великобритания — Лондон"
    ],
    en: [
      "Belgium — Brussels", "Germany — Berlin", "Netherlands — Amsterdam",
      "France — Paris", "United Kingdom — London"
    ],
    de: [
      "Belgien — Brüssel", "Deutschland — Berlin", "Niederlande — Amsterdam",
      "Frankreich — Paris", "Vereinigtes Königreich — London"
    ],
    zh: ["比利时 — 布鲁塞尔", "德国 — 柏林", "荷兰 — 阿姆斯特丹", "法国 — 巴黎", "英国 — 伦敦"],
    ja: [
      "ベルギー — ブリュッセル", "ドイツ — ベルリン", "オランダ — アムステルダム",
      "フランス — パリ", "イギリス — ロンドン"
    ],
    ko: [
      "벨기에 — 브뤼셀", "독일 — 베를린", "네덜란드 — 암스테르담",
      "프랑스 — 파리", "영국 — 런던"
    ]
  }[state.lang] || [
    "Belgium — Brussels", "Germany — Berlin", "Netherlands — Amsterdam",
    "France — Paris", "United Kingdom — London"
  ];
  const protocols = ["openvpn", "amneziawg", "wireguard", "openvpn", "l2tp"];
  const protocolNames = ["OpenVPN", "AmneziaWG", "WireGuard", "OpenVPN", "L2TP/IPsec"];
  const profileIds = ["be-brussels", "de-berlin", "nl-amsterdam", "fr-paris", "gb-london"]
    .map((name) => `profile-docs-${name}`);
  const profiles = profileNames.map((name, index) => ({
    profile_id: profileIds[index],
    name,
    location: name,
    protocol: protocols[index],
    protocol_name: protocolNames[index],
    file_name: `documentation-${profileIds[index]}.conf`,
    selected: index === 0
  }));
  const latencies = [24, 31, 37, 46, null];
  const reachability = ["reachable", "reachable", "reachable", "reachable", "unknown"];
  const probeResults = profiles.map((profile, index) => ({
    profile_id: profile.profile_id,
    display_name: profile.name,
    protocol: profile.protocol,
    selected: profile.selected,
    active: index === 0,
    transport: index === 4 ? "udp" : "tcp",
    reachability: reachability[index],
    latency_ms: latencies[index],
    latency_source: latencies[index] === null ? "none" : index === 1 ? "tcp" : "icmp",
    message_key: latencies[index] === null ? "probe.unknown" : "probe.reachable"
  }));
  return {
    checkedAt,
    status: {
      available: true,
      generated_at: Math.floor(Date.now() / 1000),
      product: "Mazzy VPN",
      version: "1.4.6",
      service_state: "active",
      desired: "up",
      internet: "up",
      tunnel_active: true,
      healthy: true,
      health_failures: 0,
      protocol: "openvpn",
      protocol_name: "OpenVPN",
      profile: profiles[0].name,
      selected: profiles[0].file_name,
      location: profiles[0].name,
      interface: "vpnovpn0",
      handshake_age: 18,
      public_ip: "203.0.113.7",
      autostart: true,
      health_monitor: true,
      fallback: { active: false },
      profiles: { amneziawg: 1, wireguard: 1, openvpn: 2, l2tp: 1 }
    },
    profiles,
    probe: {
      schema_version: 1,
      checked_at: checkedAt,
      scope: "all",
      timeout_seconds: 3,
      concurrency: 4,
      duration_ms: 842,
      summary: {
        total: 5, reachable: 4, unknown: 1, unreachable: 0, invalid: 0, active: 1
      },
      results: probeResults
    },
    verification: {
      schema_version: 1,
      checked_at: checkedAt,
      verdict: "verified",
      message_key: "verify.verified",
      tunnel: {
        active: true,
        protocol: "openvpn",
        profile: profiles[0].name,
        interface: "vpnovpn0"
      },
      ipv4: {
        interface_ip: "203.0.113.7",
        default_ip: "203.0.113.7",
        same_egress: true
      },
      ipv6: {
        interface_ip: null,
        default_ip: null,
        potential_leak: false
      },
      geo: {
        expected_country_code: "BE",
        observed_country_code: "BE",
        country_match: "match",
        providers_agree: true,
        providers: [
          {
            provider: "ipapi",
            ip: "203.0.113.7",
            country_code: "BE",
            country,
            region: city,
            city
          },
          {
            provider: "ipwho",
            ip: "203.0.113.7",
            country_code: "BE",
            country,
            region: city,
            city
          }
        ]
      },
      dns: { state: "vpn-full-tunnel" },
      speed: { requested: true, measured: true, mbps: 84.6, connect_ms: 164 },
      findings: []
    },
    installation: {
      engine_installed: true,
      package_managed: true,
      installed_version: "1.4.6",
      bundled_version: "1.4.6",
      bundled_installer: true,
      needs_install: false,
      service_installed: true,
      monitor_installed: true,
      api_installed: true,
      api_socket_available: true,
      api_socket_accessible: true,
      status_cache_available: true,
      profile_cache_available: true,
      dependencies_ready: true,
      missing_dependencies: 0,
      dependencies: [
        {
          id: "curl", label: "curl", installed: true,
          required_for: state.lang === "ru" ? "проверка маршрута" : "route verification"
        },
        {
          id: "openvpn", label: "OpenVPN", installed: true,
          required_for: state.lang === "ru" ? "профили OpenVPN" : "OpenVPN profiles"
        },
        {
          id: "wireguard", label: "WireGuard tools", installed: true,
          required_for: state.lang === "ru" ? "профили WireGuard" : "WireGuard profiles"
        }
      ]
    },
    agents: {
      schema_version: 1,
      generated_at: Math.floor(Date.now() / 1000),
      embedded_client_ready: false,
      first_party_gateway_ready: false,
      telegram_ready: false,
      providers: [
        {
          id: "codex", display_name: "Codex", installed: true,
          version: null, adapter_status: "discovery-only",
          connection_model: "diagnostics-only", remote_control_supported: false,
          running: null, actions: []
        },
        {
          id: "claude-code", display_name: "Claude Code", installed: true,
          version: "2.1.197", adapter_status: "discovery-only",
          connection_model: "interactive-vendor-native", remote_control_supported: true,
          running: null, actions: []
        }
      ],
      transports: [
        "LAN WSS", "iroh QUIC", "libp2p QUIC", "WebRTC DataChannel",
        "WebTransport", "Tailscale / Headscale", "Reverse WSS broker"
      ].map((display_name, index) => ({
        id: `docs-transport-${index}`, display_name,
        candidate_runtime_available: false, runtime_ready: false
      }))
    },
    platform: {
      functional: true,
      os: "linux",
      desktop_version: "0.4.7",
      author: "Nik m (@mazurovn)",
      license: "AGPL-3.0-or-later"
    }
  };
}

function renderDocumentationPreview() {
  const fixture = documentationPreviewData();
  const bannerText = {
    ru: "ПРЕДПРОСМОТР ДОКУМЕНТАЦИИ · ТЕСТОВЫЕ ДАННЫЕ RFC 5737",
    en: "DOCUMENTATION PREVIEW · RFC 5737 TEST DATA",
    de: "DOKUMENTATIONSVORSCHAU · RFC-5737-TESTDATEN",
    zh: "文档预览 · RFC 5737 测试数据",
    ja: "ドキュメントプレビュー · RFC 5737 テストデータ",
    ko: "문서 미리보기 · RFC 5737 테스트 데이터"
  };
  document.body.classList.add("documentation-preview");
  const banner = document.createElement("div");
  banner.className = "preview-banner";
  banner.setAttribute("role", "status");
  banner.textContent = bannerText[state.lang] || bannerText.en;
  document.body.prepend(banner);
  state.status = fixture.status;
  state.profiles = fixture.profiles;
  state.profileHealth.clear();
  fixture.probe.results.forEach((entry) => state.profileHealth.set(entry.profile_id, entry));
  state.probeCheckedAt = fixture.probe.checked_at;
  state.installation = fixture.installation;
  state.agentIntegrations = fixture.agents;
  state.platformInfo = fixture.platform;
  renderStatus(fixture.status);
  state.verification = fixture.verification;
  renderProfiles();
  renderProbeCheckedAt();
  renderVerification();
  renderInstallation(fixture.installation);
  renderAgentIntegrations(fixture.agents);
  renderAbout();
  const verification = verificationOutput(fixture.verification);
  showOperationResult({
    success: true,
    action: "verify",
    output: verification
  }, t("verifyNow"));
  addEvent(t("verified"), `${fixture.status.location} · 24 ms`, "success", verification);
  addEvent(t("checkAllLocations"), "4/5 · 24–46 ms", "success",
    formatProbeOutput(fixture.probe));
  showPage(previewParameters.get("page") || "dashboard");
  if (previewParameters.get("update") === "available") {
    showUpdateDialog({
      current_version: "0.4.7",
      version: "0.4.8",
      install_supported: true,
      installation_method: "signed-in-app"
    });
  }
}

function setServiceControlState(service, enabled) {
  document.querySelectorAll(`[data-service-action^="${service}-"]`).forEach((button) => {
    const active = button.dataset.serviceAction === `${service}-${enabled ? "on" : "off"}`;
    button.classList.toggle("active", active);
    button.setAttribute("aria-pressed", String(active));
  });
  const node = $(`#${service}-control-state`);
  if (node) {
    node.textContent = `${t("currentState")}: ${enabled ? t("enabled") : t("disabled")}`;
    node.className = `setting-current ${enabled ? "on" : ""}`;
  }
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
  if (!busy && state.agentIntegrations) renderAgentIntegrations(state.agentIntegrations);
}

function showOperationResult(result, title = "") {
  state.lastOperation = result;
  const output = $("#operation-output");
  const status = $("#operation-state");
  $("#operation-title").textContent = title || result?.action || t("diagnosticOutput");
  const warning = result?.severity === "warning";
  status.textContent = warning ? t("warning")
    : result?.success ? t("outputSuccess") : t("outputFailed");
  status.className = `operation-state ${warning ? "warning" : result?.success ? "success" : "error"}`;
  output.textContent = result?.output?.trim() || (result?.success ? t("actionDone") : t("actionFailed"));
}

function verificationOutput(result) {
  const provider = result?.geo?.providers?.[0] || {};
  const maskIp = (value) => !value ? "—" : state.hideIp ? "•••.•••.•••.•••" : value;
  return [
    `verdict=${result?.verdict || "unknown"}`,
    `checked_at=${result?.checked_at || "—"}`,
    `tunnel.active=${Boolean(result?.tunnel?.active)}`,
    `tunnel.protocol=${result?.tunnel?.protocol || "—"}`,
    `tunnel.interface=${result?.tunnel?.interface || "—"}`,
    `ipv4.interface=${maskIp(result?.ipv4?.interface_ip)}`,
    `ipv4.default=${maskIp(result?.ipv4?.default_ip)}`,
    `ipv4.same_egress=${Boolean(result?.ipv4?.same_egress)}`,
    `ipv6.potential_leak=${Boolean(result?.ipv6?.potential_leak)}`,
    `geo.observed=${result?.geo?.observed_country_code || "—"} ${provider.country || ""} ${provider.city || ""}`.trim(),
    `geo.expected=${result?.geo?.expected_country_code || "—"}`,
    `geo.match=${result?.geo?.country_match || "unknown"}`,
    `geo.providers_agree=${result?.geo?.providers_agree ?? "unknown"}`,
    `dns.state=${result?.dns?.state || "unknown"}`,
    `speed=${result?.speed?.measured ? `${result.speed.mbps} Mbps` : t("speedNotRun")}`,
    "",
    ...(result?.findings || [])
  ].join("\n");
}

function limitedConfidenceVerification(result) {
  const nonRiskFindings = new Set([
    "verify.geo.expected-country-unavailable",
    "verify.geo.single-provider"
  ]);
  return result?.verdict === "warning"
    && result?.tunnel?.active === true
    && result?.ipv4?.same_egress === true
    && result?.ipv6?.potential_leak === false
    && result?.dns?.state === "vpn-full-tunnel"
    && Array.isArray(result?.findings)
    && result.findings.length > 0
    && result.findings.every((finding) => nonRiskFindings.has(finding));
}

function verificationPresentation(result) {
  if (limitedConfidenceVerification(result)) {
    return { key: "partiallyVerified", css: "partial", severity: "warning", eventType: "warning" };
  }
  const verdict = result?.verdict || "failed";
  return {
    key: verdict,
    css: verdict,
    severity: verdict === "warning" ? "warning" : verdict,
    eventType: verdict === "verified" ? "success" : verdict === "warning" ? "warning" : "error"
  };
}

function renderVerification() {
  const card = $("#verification-card");
  if (!card) return;
  const result = state.verification;
  const badge = $("#verification-badge");
  if (!result) {
    card.className = "glass verification-card";
    badge.textContent = t("notVerified");
    badge.className = "health-badge";
    $("#verified-location").textContent = "—";
    $("#verified-profile-match").textContent = "—";
    $("#verified-route").textContent = "—";
    $("#verified-ipv6").textContent = "—";
    $("#verified-dns").textContent = "—";
    $("#verified-speed").textContent = t("speedNotRun");
    $("#verification-note").textContent = t("verificationDisclaimer");
    return;
  }
  const presentation = verificationPresentation(result);
  card.className = `glass verification-card ${presentation.css}`;
  badge.textContent = t(presentation.key);
  const badgeState = presentation.css === "verified" ? "ok"
    : presentation.css === "failed" ? "bad"
      : presentation.css === "partial" || presentation.css === "warning" ? "warn" : "";
  badge.className = `health-badge ${badgeState}`.trim();
  const provider = result?.geo?.providers?.[0] || {};
  const location = [provider.country, provider.city].filter(Boolean).join(" · ");
  $("#verified-location").textContent = location
    ? `${result?.geo?.observed_country_code || "—"} · ${location}`
    : result?.geo?.observed_country_code || t("unknown");
  const matches = {
    match: t("matches"),
    mismatch: t("mismatch"),
    unknown: t("unknown")
  };
  $("#verified-profile-match").textContent = result?.findings?.includes("verify.geo.expected-country-unavailable")
    ? t("profileCountryMissing")
    : matches[result?.geo?.country_match] || t("unknown");
  $("#verified-route").textContent = result?.ipv4?.same_egress ? t("sameEgress") : t("differentEgress");
  $("#verified-ipv6").textContent = result?.ipv6?.potential_leak ? t("potentialLeak") : t("noLeak");
  $("#verified-dns").textContent = result?.dns?.state === "vpn-full-tunnel"
    ? t("fullTunnelDns") : t("partialDns");
  $("#verified-speed").textContent = result?.speed?.measured
    ? `${result.speed.mbps} Mbps · ${result.speed.connect_ms} ms`
    : t("speedNotRun");
  const checked = new Date(result.checked_at);
  const time = Number.isNaN(checked.getTime()) ? "" : checked.toLocaleTimeString([], {
    hour: "2-digit", minute: "2-digit", second: "2-digit"
  });
  $("#verification-note").textContent = `${time ? `${t("lastChecked")}: ${time}. ` : ""}${t("verificationDisclaimer")}`;
}

async function verifyConnection(includeSpeed = false) {
  if (state.busy) return null;
  setBusy(true);
  const title = includeSpeed ? t("verifyWithSpeed") : t("verifyNow");
  $("#operation-state").textContent = t("outputRunning");
  $("#operation-state").className = "operation-state running";
  $("#operation-title").textContent = title;
  $("#operation-output").textContent = `${t("actionStarted")}: ${title}…`;
  showToast(`${t("actionStarted")}: ${title}`);
  try {
    if (!invoke) throw new Error("Tauri runtime is unavailable");
    const result = await invoke("verify_connection", {
      timeout: 10,
      includeSpeed
    });
    state.verification = result;
    renderVerification();
    const success = result?.verdict !== "failed";
    const presentation = verificationPresentation(result);
    const output = verificationOutput(result);
    showOperationResult({
      success,
      severity: presentation.severity,
      action: "verify",
      output
    }, title);
    addEvent(
      t(presentation.key),
      `${result?.geo?.observed_country_code || t("unknown")} · ${result?.dns?.state || "unknown"}`,
      presentation.eventType,
      output
    );
    showToast(t(presentation.key), result?.verdict === "failed");
    if (result?.verdict === "failed") showPage("diagnostics");
    return result;
  } catch (error) {
    const message = String(error);
    state.verification = null;
    renderVerification();
    showOperationResult({ success: false, action: "verify", output: message }, title);
    addEvent(t("actionFailed"), message, "error", message);
    showToast(`${t("actionFailed")}: ${message}`, true);
    showPage("diagnostics");
    return null;
  } finally {
    setBusy(false);
    await refreshStatus(false);
  }
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
  setServiceControlState("autostart", Boolean(data?.autostart));
  setServiceControlState("monitor", Boolean(data?.health_monitor));
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
  const activeProfileSignature = data?.tunnel_active
    ? `${data?.protocol || ""}|${data?.profile || ""}`
    : "";
  if (state.lastActiveProfileSignature !== activeProfileSignature) {
    state.lastActiveProfileSignature = activeProfileSignature;
    state.verification = null;
    renderVerification();
    if (state.profiles.length) renderProfiles();
  }
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
  const sortMode = $("#profile-sort")?.value || "recommended";
  const reachabilityRank = { reachable: 0, unknown: 1, unreachable: 3, invalid: 4 };
  const activeProfile = (profile) => statusMatchesProfile(profile);
  profiles.sort((left, right) => {
    const leftHealth = state.profileHealth.get(left.profile_id);
    const rightHealth = state.profileHealth.get(right.profile_id);
    const leftLatency = Number.isFinite(leftHealth?.latency_ms) ? leftHealth.latency_ms : Infinity;
    const rightLatency = Number.isFinite(rightHealth?.latency_ms) ? rightHealth.latency_ms : Infinity;
    const leftStatus = reachabilityRank[leftHealth?.reachability] ?? 2;
    const rightStatus = reachabilityRank[rightHealth?.reachability] ?? 2;
    const nameOrder = String(left.name || left.file_name).localeCompare(
      String(right.name || right.file_name), state.lang, { sensitivity: "base" }
    );
    if (sortMode === "name") return nameOrder;
    if (sortMode === "latency") return leftLatency - rightLatency || leftStatus - rightStatus || nameOrder;
    if (sortMode === "status") return leftStatus - rightStatus || leftLatency - rightLatency || nameOrder;
    return Number(activeProfile(right)) - Number(activeProfile(left))
      || leftStatus - rightStatus
      || leftLatency - rightLatency
      || Number(right.selected) - Number(left.selected)
      || nameOrder;
  });
  if (!profiles.length) {
    const empty = document.createElement("div");
    empty.className = "empty-state";
    empty.textContent = state.profileCacheAvailable ? t("profilesEmpty") : t("profilesUnavailable");
    list.replaceChildren(empty);
    return;
  }
  list.replaceChildren(...profiles.map((profile) => {
    const health = state.profileHealth.get(profile.profile_id);
    const checking = state.probeRunningScope === "all"
      || state.probeRunningScope === profile.protocol;
    const liveStatusAvailable = state.status?.available !== false
      && Number(state.status?.generated_at || 0) > 0;
    const active = liveStatusAvailable
      ? statusMatchesProfile(profile)
      : Boolean(health?.active);
    const row = document.createElement("div");
    row.className = `profile-item${profile.selected ? " selected" : ""}`
      + `${health?.reachability ? ` health-${health.reachability}` : ""}`
      + `${active ? " active-profile" : ""}`;
    const marker = document.createElement("span");
    marker.className = "profile-marker";
    const copy = document.createElement("div");
    copy.className = "profile-copy";
    const name = document.createElement("strong");
    name.textContent = profile.name || profile.file_name;
    const detail = document.createElement("small");
    const location = profile.location && profile.location !== profile.name
      ? `${profile.location} · `
      : "";
    detail.textContent = `${location}${profile.protocol_name || profile.protocol}`
      + `${profile.selected ? ` · ${t("selectedProfile")}` : ""}`;
    const healthLine = document.createElement("small");
    healthLine.className = `profile-health${health?.reachability ? ` ${health.reachability}` : ""}`;
    const healthLabels = {
      reachable: t("locationReachable"),
      unknown: t("locationUnknown"),
      unreachable: t("locationUnreachable"),
      invalid: t("locationInvalid")
    };
    const latency = Number.isFinite(health?.latency_ms)
      ? ` · ${health.latency_ms} ms ${String(health.latency_source || "").toUpperCase()}`
      : "";
    healthLine.textContent = `${checking
      ? `${t("checkingLocations")}…`
      : (healthLabels[health?.reachability] || t("locationNotChecked"))}`
      + latency + `${active ? ` · ${t("activeTunnel")}` : ""}`;
    copy.append(name, detail, healthLine);
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

function statusMatchesProfile(profile) {
  if (
    state.status?.available === false
    || Number(state.status?.generated_at || 0) <= 0
    || !state.status?.tunnel_active
    || state.status?.protocol !== profile.protocol
  ) {
    return false;
  }
  if (typeof state.status.profile_id === "string" && state.status.profile_id) {
    return state.status.profile_id === profile.profile_id;
  }
  if (
    typeof state.status.profile_file_name === "string"
    && state.status.profile_file_name
  ) {
    return state.status.profile_file_name === profile.file_name;
  }
  const legacyMatches = state.profiles.filter((candidate) =>
    candidate.protocol === state.status.protocol
    && (candidate.name || candidate.file_name) === state.status.profile);
  return legacyMatches.length === 1
    && legacyMatches[0].profile_id === profile.profile_id;
}

function fastestReachableProfile() {
  return state.profiles
    .map((profile) => ({ profile, health: state.profileHealth.get(profile.profile_id) }))
    .filter(({ health }) => health?.reachability === "reachable"
      && Number.isFinite(health?.latency_ms))
    .sort((left, right) => left.health.latency_ms - right.health.latency_ms)[0]?.profile || null;
}

async function connectFastestProfile() {
  const profile = fastestReachableProfile();
  if (!profile) {
    showToast(t("fastestUnavailable"), true);
    showPage("profiles");
    return;
  }
  await runOperation({
    kind: "connect",
    protocol: profile.protocol,
    profile: profile.file_name
  }, `${t("connectFastest")}: ${profile.name || profile.file_name}`);
}

function scheduleProfileRefresh() {
  if (state.profileCacheAvailable || profileRetryTimer !== null || profileRetryExhausted) return;
  if (profileRetryAttempt >= profileRetryDelays.length) {
    profileRetryExhausted = true;
    return;
  }
  const delay = profileRetryDelays[profileRetryAttempt];
  profileRetryAttempt += 1;
  profileRetryTimer = window.setTimeout(async () => {
    profileRetryTimer = null;
    await refreshProfiles(false);
    if (!state.profileCacheAvailable) scheduleProfileRefresh();
  }, delay);
}

async function refreshProfiles(manual = false) {
  if (manual) {
    profileRetryAttempt = 0;
    profileRetryExhausted = false;
    if (profileRetryTimer !== null) {
      window.clearTimeout(profileRetryTimer);
      profileRetryTimer = null;
    }
  }
  if (profileRefreshPromise) {
    await profileRefreshPromise;
    return;
  }
  profileRefreshPromise = (async () => {
    try {
      if (!invoke) throw new Error("Tauri runtime is unavailable");
      const data = await invoke("get_profiles");
      state.profiles = Array.isArray(data?.profiles) ? data.profiles : [];
      state.profileCacheAvailable = data?.available !== false;
      if (state.profileCacheAvailable) {
        profileRetryAttempt = 0;
        profileRetryExhausted = false;
        if (profileRetryTimer !== null) {
          window.clearTimeout(profileRetryTimer);
          profileRetryTimer = null;
        }
      }
      renderProfiles();
      if (!state.profileCacheAvailable) scheduleProfileRefresh();
    } catch (error) {
      state.profiles = [];
      state.profileCacheAvailable = false;
      renderProfiles();
      scheduleProfileRefresh();
      if (manual) showToast(String(error), true);
      return;
    } finally {
      profileRefreshPromise = null;
    }
    if (manual) showToast(
      state.profileCacheAvailable ? t("profilesRefreshed") : t("profilesUnavailable"),
      !state.profileCacheAvailable
    );
  })();
  await profileRefreshPromise;
}

function renderProbeCheckedAt() {
  const node = $("#location-health-meta");
  if (!node) return;
  if (!state.probeCheckedAt) {
    node.textContent = t("locationNotChecked");
    return;
  }
  const checked = new Date(state.probeCheckedAt);
  node.textContent = Number.isNaN(checked.getTime())
    ? t("lastChecked")
    : `${t("lastChecked")}: ${checked.toLocaleTimeString([], {
      hour: "2-digit", minute: "2-digit", second: "2-digit"
    })}`;
}

function formatProbeOutput(result) {
  const summary = result?.summary || {};
  const lines = (result?.results || []).map((entry) => {
    const labels = {
      reachable: t("locationReachable"),
      unknown: t("locationUnknown"),
      unreachable: t("locationUnreachable"),
      invalid: t("locationInvalid")
    };
    const latency = Number.isFinite(entry.latency_ms)
      ? ` · ${entry.latency_ms} ms ${String(entry.latency_source || "").toUpperCase()}`
      : "";
    const active = entry.active ? ` · ${t("activeTunnel")}` : "";
    return `${entry.display_name} · ${labels[entry.reachability] || entry.reachability}${latency}${active}`;
  });
  return [
    `total=${summary.total || 0} reachable=${summary.reachable || 0}`
      + ` unknown=${summary.unknown || 0} unreachable=${summary.unreachable || 0}`
      + ` invalid=${summary.invalid || 0} active=${summary.active || 0}`,
    t("endpointProbeHint"),
    "",
    ...lines
  ].join("\n");
}

async function checkLocations(protocol = "all", title = "") {
  if (state.busy) return null;
  setBusy(true);
  const operationTitle = title || t("checkAllLocations");
  state.probeRunningScope = protocol;
  renderProfiles();
  $("#operation-state").textContent = t("outputRunning");
  $("#operation-state").className = "operation-state running";
  $("#operation-title").textContent = operationTitle;
  $("#operation-output").textContent = `${t("checkingLocations")}…`;
  showToast(`${t("checkingLocations")}…`);
  try {
    if (!invoke) throw new Error("Tauri runtime is unavailable");
    const result = await invoke("probe_profiles", {
      protocol,
      timeout: Math.min(30, operationTimeout()),
      concurrency: 4
    });
    if (protocol === "all") {
      state.profileHealth.clear();
    } else {
      state.profiles
        .filter((profile) => profile.protocol === protocol)
        .forEach((profile) => state.profileHealth.delete(profile.profile_id));
    }
    (result?.results || []).forEach((entry) => {
      state.profileHealth.set(entry.profile_id, entry);
    });
    state.probeCheckedAt = result?.checked_at || null;
    renderProfiles();
    renderProbeCheckedAt();
    const hardFailures = Number(result?.summary?.unreachable || 0)
      + Number(result?.summary?.invalid || 0);
    const success = Number(result?.summary?.total || 0) > 0 && hardFailures === 0;
    const output = formatProbeOutput(result);
    const operation = {
      success,
      action: "probe",
      output
    };
    showOperationResult(operation, operationTitle);
    addEvent(
      success ? t("actionDone") : t("actionFailed"),
      `reachable=${result?.summary?.reachable || 0}, unknown=${result?.summary?.unknown || 0}`,
      success ? "success" : "error",
      output
    );
    showToast(success ? t("actionDone") : t("actionFailed"), !success);
    return result;
  } catch (error) {
    const message = String(error);
    showOperationResult({ success: false, action: "probe", output: message }, operationTitle);
    addEvent(t("actionFailed"), message, "error");
    showToast(`${t("actionFailed")}: ${message}`, true);
    return null;
  } finally {
    state.probeRunningScope = null;
    renderProfiles();
    setBusy(false);
    await refreshStatus(false);
  }
}

function renderInstallation(report) {
  state.installation = report;
  $("#installed-version").textContent = report?.installed_version || t("missing");
  $("#bundled-version").textContent = report?.bundled_version || t("missing");
  $("#installation-type").textContent = report?.package_managed
    ? t("packageManaged")
    : t("embeddedInstall");
  $("#service-installed").textContent = report?.service_installed ? t("installed") : t("missing");
  $("#monitor-installed").textContent = report?.monitor_installed ? t("installed") : t("missing");
  $("#api-installed").textContent = report?.api_socket_accessible
    ? t("apiReady")
    : (report?.api_installed ? t("installed") : t("missing"));
  const badge = $("#installation-badge");
  badge.textContent = report?.needs_install ? t("updateRequired") : t("ready");
  badge.className = `health-badge ${report?.needs_install ? "bad" : "ok"}`;
  const dependencies = $("#dependency-list");
  dependencies.replaceChildren(...(report?.dependencies || []).map((dependency) => {
    const row = document.createElement("div");
    row.className = `dependency${dependency.installed ? " installed" : ""}`;
    const label = document.createElement("span");
    const russianLabels = {
      ping: "ICMP ping", getent: "DNS-резолвер",
      systemd: "systemd и временные службы", journalctl: "системный журнал",
      pkexec: "авторизация PolicyKit", jq: "среда JSON API",
      acl: "доступ Desktop к локальному API",
      socat: "клиент локального API", "dns-integration": "интеграция DNS",
      "wireguard-tools": "инструменты WireGuard",
      "amneziawg-tools": "инструменты AmneziaWG",
      "amneziawg-backend": "модуль или userspace-движок AmneziaWG",
      "l2tp-transport": "транспорт L2TP"
    };
    const russianPurposes = {
      core: "ядро", Desktop: "Desktop", "local API": "локальный API",
      "CLI / TUI local API": "локальный API CLI/TUI", "VPN DNS": "DNS VPN",
      "route verification": "проверка маршрута",
      "OpenVPN profiles": "профили OpenVPN",
      "WireGuard profiles": "профили WireGuard"
    };
    const dependencyLabel = state.lang === "ru"
      ? russianLabels[dependency.id] || dependency.label
      : dependency.label;
    const dependencyPurpose = state.lang === "ru"
      ? russianPurposes[dependency.required_for] || dependency.required_for
      : dependency.required_for;
    label.textContent = `${dependencyLabel} · ${dependencyPurpose}`;
    const status = document.createElement("span");
    status.textContent = dependency.installed ? "OK" : t("missing");
    row.append(label, status);
    return row;
  }));
  renderAbout();
}

function renderAgentIntegrations(report) {
  state.agentIntegrations = report;
  const providers = report?.providers || [];
  const providerList = $("#agent-provider-list");
  if (!providerList) return;
  providerList.replaceChildren(...providers.map((provider) => {
    const row = document.createElement("div");
    row.className = "agent-provider-row";
    const copy = document.createElement("div");
    copy.className = "agent-provider-copy";
    const name = document.createElement("strong");
    name.textContent = provider.display_name;
    const details = document.createElement("small");
    const installation = provider.installed ? t("providerInstalled") : t("providerMissing");
    const remote = provider.remote_control_supported ? t("remoteSupported") : t("remoteUnsupported");
    const running = provider.running === true ? t("daemonRunning")
      : provider.running === false ? t("daemonStopped") : "";
    details.textContent = [provider.version || installation, remote, running].filter(Boolean).join(" · ");
    copy.append(name, details);
    const status = document.createElement("span");
    const discovery = provider.adapter_status === "discovery-only";
    status.className = `agent-provider-state${discovery ? " partial" : ""}`;
    status.textContent = discovery ? t("discoveryOnly") : t("runtimeUnavailable");
    row.append(copy, status);
    return row;
  }));

  const readyProviders = providers.filter((provider) =>
    provider.installed
      && provider.remote_control_supported
      && provider.adapter_status === "ready"
      && provider.connection_model !== "diagnostics-only"
  ).length;
  const providerBadge = $("#agent-provider-badge");
  providerBadge.textContent = `${readyProviders}/${providers.length}`;
  providerBadge.className = `health-badge ${readyProviders ? "ok" : "bad"}`;

  $("#embedded-agent-client-state").textContent = report?.embedded_client_ready
    ? t("runtimeReady") : t("runtimeUnavailable");
  $("#first-party-gateway-state").textContent = report?.first_party_gateway_ready
    ? t("runtimeReady") : t("gatewayPlanned");
  $("#telegram-agent-state").textContent = report?.telegram_ready
    ? t("runtimeReady") : t("gatewayPlanned");

  const transports = report?.transports || [];
  $("#agent-transport-list").replaceChildren(...transports.map((transport) => {
    const row = document.createElement("div");
    row.className = `agent-transport-row${transport.runtime_ready ? " ready" : ""}`;
    const label = document.createElement("span");
    label.textContent = transport.display_name;
    const status = document.createElement("strong");
    status.textContent = transport.runtime_ready ? t("runtimeReady") : t("runtimeUnavailable");
    row.append(label, status);
    return row;
  }));
  const readyTransports = transports.filter((transport) => transport.runtime_ready).length;
  const transportBadge = $("#agent-transport-badge");
  transportBadge.textContent = `${readyTransports}/${transports.length}`;
  transportBadge.className = `health-badge ${readyTransports ? "ok" : "bad"}`;
}

async function refreshAgents(manual = false) {
  if (!invoke) return;
  try {
    const report = await invoke("get_agent_integrations");
    renderAgentIntegrations(report);
    if (manual) showToast(t("refreshed"));
  } catch (error) {
    if (manual) showToast(String(error), true);
  }
}

function renderAbout() {
  const info = state.platformInfo;
  if (!$("#about-version")) return;
  const desktopVersion = info?.desktop_version || "—";
  const engineVersion = state.installation?.installed_version ||
    state.installation?.bundled_version || state.status?.version || "—";
  $("#about-version").textContent = `Desktop ${desktopVersion}`;
  $("#about-desktop-version").textContent = desktopVersion;
  $("#about-engine-version").textContent = engineVersion;
  $("#about-platform").textContent = info?.os ? info.os.toUpperCase() : "—";
  $("#about-author").textContent = info?.author || "Nik m (@mazurovn)";
  $("#about-license").textContent = info?.license || "AGPL-3.0-or-later";
}

async function refreshInstallation(startEmbeddedEngine = false) {
  if (!invoke) return;
  try {
    const report = await invoke("get_installation_report");
    renderInstallation(report);
    // Start the bundled engine before the first status/profile reads. PolicyKit
    // remains the native authorization boundary for privileged host changes.
    if (startEmbeddedEngine
        && state.platformInfo?.functional
        && report?.startup_repair_needed
        && report?.bundled_cli
        && report?.bundled_installer
        && !state.engineStartupAttempted
        && !state.busy) {
      state.engineStartupAttempted = true;
      return runOperation({ kind: "bootstrap" }, t("installRepair"));
    }
    return report;
  } catch (error) {
    showToast(String(error), true);
    return null;
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
  $("#auto-update-toggle").checked = state.autoUpdateChecks;
  $("#auto-update-toggle").addEventListener("change", (event) => {
    state.autoUpdateChecks = event.target.checked;
    localStorage.setItem("mazzy-auto-update-check", String(state.autoUpdateChecks));
  });
  $("#check-update-button").addEventListener("click", () => checkForUpdates(true));
  $("#update-later-button").addEventListener("click", () => {
    if (!state.updateInstalling) $("#update-dialog").close();
  });
  $("#update-action-button").addEventListener("click", runPendingUpdateAction);
  $("#update-dialog").addEventListener("cancel", (event) => {
    if (state.updateInstalling) event.preventDefault();
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
  $("#profile-sort").addEventListener("change", renderProfiles);
  $("#profiles-refresh-button").addEventListener("click", async () => {
    await runOperation({ kind: "refresh" }, t("profilesRefreshed"), false);
    await refreshProfiles(true);
  });
  $("#location-health-button").addEventListener("click", () =>
    checkLocations("all", t("checkAllLocations")));
  $("#connect-fastest-button").addEventListener("click", connectFastestProfile);
  $("#verify-egress-button").addEventListener("click", () => verifyConnection(false));
  $("#verify-speed-button").addEventListener("click", () => verifyConnection(true));
  $("#agents-refresh-button").addEventListener("click", () => refreshAgents(true));

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
  $("#probe-button").addEventListener("click", () =>
    checkLocations($("#test-protocol").value, t("probeEndpoints")));
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
  $("#diagnostic-verify-button").addEventListener("click", () => verifyConnection(false));
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
    await tauri.event.listen("update-progress", ({ payload }) => {
      handleUpdateProgress(payload);
    });
    await tauri.event.listen("navigate-page", ({ payload }) => {
      showPage(String(payload || "dashboard"));
    });
    await tauri.event.listen("vpn-action-result", ({ payload }) => {
      let displayPayload = payload;
      let eventType = payload?.success ? "success" : "error";
      if (payload?.action === "verify" && payload?.data) {
        state.verification = payload.data;
        renderVerification();
        displayPayload = {
          ...payload,
          severity: payload.data.verdict === "warning" ? "warning" : payload.data.verdict,
          output: verificationOutput(payload.data)
        };
        eventType = payload.data.verdict === "verified" ? "success"
          : payload.data.verdict === "warning" ? "warning" : "error";
      } else if (payload?.action === "probe-all" && payload?.data) {
        state.profileHealth.clear();
        (payload.data.results || []).forEach((entry) => {
          state.profileHealth.set(entry.profile_id, entry);
        });
        state.probeCheckedAt = payload.data.checked_at || null;
        renderProfiles();
        renderProbeCheckedAt();
        displayPayload = { ...payload, output: formatProbeOutput(payload.data) };
      }
      const detail = displayPayload?.output?.trim().split("\n").slice(-1)[0] || "";
      addEvent(displayPayload?.success ? t("actionDone") : t("actionFailed"), detail,
        eventType, displayPayload?.output);
      showOperationResult(displayPayload, displayPayload?.action || t("diagnosticOutput"));
      showToast(displayPayload?.success ? t("actionDone")
        : `${t("actionFailed")}: ${detail}`, !displayPayload?.success);
      if (displayPayload?.action === "doctor" || !displayPayload?.success) showPage("diagnostics");
      refreshStatus(false);
    });
  }

  if (documentationPreview) {
    renderDocumentationPreview();
    return;
  }

  if (invoke) {
    const platform = await invoke("get_platform_info");
    state.platformInfo = platform;
    $("#platform-note").textContent = platform?.functional ? "" : t("preview");
    renderAbout();
  }
  await refreshInstallation(true);
  await Promise.all([
    refreshStatus(false), refreshProfiles(false), refreshAgents(false)
  ]);
  if (state.autoUpdateChecks) void checkForUpdates(false);
  setInterval(() => {
    refreshStatus(false);
    if (!state.profileCacheAvailable && !profileRetryExhausted) refreshProfiles(false);
  }, 5000);
  setInterval(() => {
    if (state.page === "agents" && !state.busy) refreshAgents(false);
  }, 15000);
});
