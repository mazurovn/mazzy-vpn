<!-- SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0 -->
<!-- Copyright © 2026 Nik m (@mazurovn). All rights reserved. -->

# Mazzy VPN — Go Native CLI (`mazzy-vpn`)

> Language: **English** · [Русский](Go-CLI#русский) · [Deutsch](Go-CLI#deutsch) · [中文](Go-CLI#中文) · [日本語](Go-CLI#日本語) · [한국어](Go-CLI#한국어)

The Go native CLI is a single, statically-linked binary that embeds the VPN
engine (AmneziaWG/WireGuard via a vendored `amneziawg-go`). It needs **no**
`awg`, `awg-quick`, `wg`, `jq`, or any external VPN backend — the engine is
built in. `CGO_ENABLED=0`, so the binary is fully portable across Linux hosts.

## Install

```sh
# From a release tarball:
tar xzf mazzy-vpn-go-*-linux-amd64.tar.gz
sudo ./install.sh          # installs /usr/local/bin/mazzy-vpn
```

The installer performs **no** package installation or compilation — it only
checks for base OS tools (`ip`, `nft`) and copies the static binary.

## First run & language

The UI defaults to **English**; your OS locale is honored automatically.

```sh
mazzy-vpn language --list      # show the 6 languages, current marked with *
mazzy-vpn language             # interactive selection menu
mazzy-vpn language de          # set directly (en/ru/de/zh/ja/ko)
```

Language resolution order (no hardcoding): saved setting → `MAZZY_LANG` →
`LC_ALL` → `LC_MESSAGES` → `LANG` → English.

## Core commands

| Command | What it does |
|---|---|
| `mazzy-vpn up [zone]` | connect to a profile (or the best one) |
| `mazzy-vpn best` | pick + connect the cleanest live server |
| `mazzy-vpn status` | show tunnel + real egress |
| `mazzy-vpn test` | rank live zones by ICMP + latency |
| `mazzy-vpn connect FILE.conf` | connect a specific profile, live dashboard |
| `mazzy-vpn disconnect` | graceful teardown |
| `mazzy-vpn recover` | panic button: restore a clean network state |
| `mazzy-vpn daemon <zone\|--best>` | supervised auto-reconnect + failover |
| `mazzy-vpn doctor` | full diagnostics |
| `mazzy-vpn verify` | prove protected egress (interface + egress + DNS) |

## AI-native commands

| Command | What it does |
|---|---|
| `mazzy-vpn providers` | reachability of 12 AI providers (OpenAI/Anthropic/Gemini…) |
| `mazzy-vpn stealth` | detection score (IPv6/DNS/timezone/ASN/Cloudflare) |
| `mazzy-vpn mimic` | align system timezone/locale to the egress country |
| `mazzy-vpn dns-check` | DNS privacy: in-country + encrypted (DoT) |
| `mazzy-vpn control id\|pair\|list` | control-plane identity & pairing |
| `mazzy-vpn netdiag` / `diagnose` / `trace` | network path analysis |

## Control plane (AI-native)

Every agent/harness gets a self-authenticating Ed25519 identity. Access is
**deny-by-default**: an agent can only reach an egress through an explicit,
signed, time-bounded, revocable grant. See [Architecture](Architecture).

```sh
mazzy-vpn control id                  # show this node's identity
mazzy-vpn control pair <ID> <PUBKEY>  # trust a peer (anti-impersonation enforced)
mazzy-vpn control list                # paired participants
```

## Safety model

- **Fail-closed**: an IPv6 leak guard is armed *before* the interface exists.
- **fwmark = routing table** (WireGuard policy-routing parity).
- **Real egress verification** before reporting "protected".
- **Atomic, durable state** (temp+rename, fsync).
- **Single-writer lock** for all mutations.

---

<a id="русский"></a>
## Русский

Нативный Go CLI — один статический бинарник со встроенным VPN-движком
(AmneziaWG/WireGuard). Не нужны `awg`/`wg`/`jq` — движок встроен.
`CGO_ENABLED=0`, бинарник переносим между Linux-хостами.

**Язык интерфейса** по умолчанию — английский; локаль ОС учитывается:

```sh
mazzy-vpn language --list      # 6 языков, текущий помечен *
mazzy-vpn language             # интерактивное меню
mazzy-vpn language ru          # задать напрямую
```

Порядок выбора языка (без хардкода): сохранённая настройка → `MAZZY_LANG` →
`LC_ALL` → `LC_MESSAGES` → `LANG` → английский.

Основные команды: `up`, `best`, `status`, `test`, `connect`, `disconnect`,
`recover`, `daemon`, `doctor`, `verify`. AI-команды: `providers`, `stealth`,
`mimic`, `dns-check`, `control`. Доступ агентов — deny-by-default через
подписанные гранты.

---

<a id="deutsch"></a>
## Deutsch

Die native Go-CLI ist eine statisch gelinkte Binärdatei mit eingebauter
VPN-Engine (AmneziaWG/WireGuard). Kein `awg`/`wg`/`jq` nötig — die Engine ist
eingebaut. `CGO_ENABLED=0`, voll portabel über Linux-Hosts.

**UI-Sprache** ist standardmäßig Englisch; die OS-Locale wird berücksichtigt:

```sh
mazzy-vpn language --list      # 6 Sprachen, aktuelle mit * markiert
mazzy-vpn language             # interaktives Menü
mazzy-vpn language de          # direkt setzen
```

Sprachauflösung (kein Hardcoding): gespeicherte Einstellung → `MAZZY_LANG` →
`LC_ALL` → `LC_MESSAGES` → `LANG` → Englisch.

Kernbefehle: `up`, `best`, `status`, `test`, `connect`, `disconnect`,
`recover`, `daemon`, `doctor`, `verify`. KI-Befehle: `providers`, `stealth`,
`mimic`, `dns-check`, `control`.

---

<a id="中文"></a>
## 中文

原生 Go 命令行是一个静态链接的二进制文件，内置 VPN 引擎（AmneziaWG/WireGuard）。
无需 `awg`/`wg`/`jq` — 引擎已内置。`CGO_ENABLED=0`，可在 Linux 主机间移植。

**界面语言**默认英语；自动识别操作系统区域设置：

```sh
mazzy-vpn language --list      # 6 种语言，当前以 * 标记
mazzy-vpn language             # 交互式选择菜单
mazzy-vpn language zh          # 直接设置
```

语言解析顺序（无硬编码）：已保存设置 → `MAZZY_LANG` → `LC_ALL` →
`LC_MESSAGES` → `LANG` → 英语。

核心命令：`up`、`best`、`status`、`test`、`connect`、`disconnect`、`recover`、
`daemon`、`doctor`、`verify`。AI 命令：`providers`、`stealth`、`mimic`、
`dns-check`、`control`。

---

<a id="日本語"></a>
## 日本語

ネイティブ Go CLI は VPN エンジン（AmneziaWG/WireGuard）を内蔵した単一の
静的リンクバイナリです。`awg`/`wg`/`jq` は不要 — エンジンは組み込み済み。
`CGO_ENABLED=0` で、Linux ホスト間で完全に移植可能です。

**UI 言語**は既定で英語。OS のロケールは自動的に尊重されます：

```sh
mazzy-vpn language --list      # 6 言語、現在の言語は * 印
mazzy-vpn language             # 対話式選択メニュー
mazzy-vpn language ja          # 直接設定
```

言語解決の順序（ハードコードなし）：保存された設定 → `MAZZY_LANG` →
`LC_ALL` → `LC_MESSAGES` → `LANG` → 英語。

主要コマンド：`up`、`best`、`status`、`test`、`connect`、`disconnect`、
`recover`、`daemon`、`doctor`、`verify`。AI コマンド：`providers`、`stealth`、
`mimic`、`dns-check`、`control`。

---

<a id="한국어"></a>
## 한국어

네이티브 Go CLI는 VPN 엔진(AmneziaWG/WireGuard)을 내장한 단일 정적 링크
바이너리입니다. `awg`/`wg`/`jq`가 필요 없습니다 — 엔진이 내장되어 있습니다.
`CGO_ENABLED=0`으로 Linux 호스트 간 완전히 이식 가능합니다.

**UI 언어**는 기본값이 영어이며 OS 로캘이 자동으로 적용됩니다:

```sh
mazzy-vpn language --list      # 6개 언어, 현재 언어는 * 표시
mazzy-vpn language             # 대화형 선택 메뉴
mazzy-vpn language ko          # 직접 설정
```

언어 결정 순서(하드코딩 없음): 저장된 설정 → `MAZZY_LANG` → `LC_ALL` →
`LC_MESSAGES` → `LANG` → 영어.

핵심 명령: `up`, `best`, `status`, `test`, `connect`, `disconnect`, `recover`,
`daemon`, `doctor`, `verify`. AI 명령: `providers`, `stealth`, `mimic`,
`dns-check`, `control`.
