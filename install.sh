#!/usr/bin/env bash
# Copyright (C) 2026 Nik m (@mazurovn)
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PRODUCT_NAME="Mazzy VPN"
DESTDIR=""
DRY_RUN=0
ASSUME_YES=0
NO_DEPS=0
DEPS_ONLY=0
SKIP_TESTS=0
SKIP_CHECKS=0
LIVE_TEST=0
CONFIG_SOURCE=""
FORCE_CONFIGS=0
INSTALL_LANG="${VPNCTL_INSTALL_LANG:-${VPNCTL_LANG:-}}"
LANG_EXPLICIT=0
AWG_TOOLS_VERSION="1.0.20260618-2"
AWG_GO_VERSION="3.0.1"
AMNEZIA_PPA_BASE="https://ppa.launchpadcontent.net/amnezia/ppa/ubuntu"

usage() {
    cat <<'EOF'
Установка Mazzy VPN:
  sudo ./install.sh [--yes] [--no-deps] [--dry-run]

Параметры:
  --yes          не задавать вопросы при установке пакетов
  --no-deps      не устанавливать отсутствующие зависимости
  --deps-only    только проверить/установить пакеты
  --skip-tests   пропустить встроенные preflight-тесты
  --skip-checks  пропустить post-install self-test
  --live-test    после установки живьём проверить все VPN-профили
  --config-dir DIR
                 рекурсивно найти и импортировать VPN-конфиги из DIR
  --force-configs
                 заменять одноимённые системные профили при импорте DIR
  --lang CODE    язык интерфейса: ru, en, de, zh, ja или ko
  --dry-run      показать действия без изменений
  --destdir DIR  установить в staging-каталог (для сборки/тестов)

Устанавливаемые команды:
  mazzy-vpn      основной CLI/TUI
  vpnctl         совместимый alias
  mazzyvpn       короткий alias без дефиса
EOF
}

while (($#)); do
    case "$1" in
        --yes|-y) ASSUME_YES=1 ;;
        --no-deps) NO_DEPS=1 ;;
        --deps-only) DEPS_ONLY=1 ;;
        --skip-tests) SKIP_TESTS=1 ;;
        --skip-checks) SKIP_CHECKS=1 ;;
        --live-test) LIVE_TEST=1 ;;
        --config-dir)
            shift
            [[ $# -gt 0 ]] || { echo "--config-dir требует путь" >&2; exit 2; }
            CONFIG_SOURCE="$1"
            ;;
        --force-configs) FORCE_CONFIGS=1 ;;
        --lang)
            shift
            [[ $# -gt 0 ]] || { echo "--lang требует код языка" >&2; exit 2; }
            INSTALL_LANG="$1"
            LANG_EXPLICIT=1
            ;;
        --dry-run) DRY_RUN=1 ;;
        --destdir)
            shift
            [[ $# -gt 0 ]] || { echo "--destdir требует путь" >&2; exit 2; }
            DESTDIR="$1"
            ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Неизвестный параметр: $1" >&2; usage >&2; exit 2 ;;
    esac
    shift
done

run() {
    if ((DRY_RUN)); then
        printf '+'
        printf ' %q' "$@"
        printf '\n'
    else
        "$@"
    fi
}

normalize_install_language() {
    local language="${1:-}"
    language="${language,,}"
    language="${language%%.*}"
    language="${language%%_*}"
    language="${language%%-*}"
    case "$language" in
        ru|rus|russian) printf 'ru\n' ;;
        en|eng|english) printf 'en\n' ;;
        de|deu|ger|german|germany|deutsch) printf 'de\n' ;;
        zh|zho|chi|chinese|china) printf 'zh\n' ;;
        ja|jpn|japanese|japan) printf 'ja\n' ;;
        ko|kor|korean|korea) printf 'ko\n' ;;
        *) return 1 ;;
    esac
}

choose_install_language() {
    local normalized="" answer
    normalized="$(normalize_install_language "$INSTALL_LANG" 2>/dev/null || true)"
    if ((LANG_EXPLICIT)) && [[ -z "$normalized" ]]; then
        echo "Unsupported language / Неподдерживаемый язык: $INSTALL_LANG" >&2
        return 2
    fi
    if [[ -z "$normalized" && $ASSUME_YES -eq 0 && -z "$DESTDIR" ]] &&
       { [[ -t 0 ]] || [[ "${VPNCTL_INSTALL_FORCE_INTERACTIVE:-0}" == "1" ]]; }; then
        cat <<'EOF'

╭────────────────────────────────────────╮
│        M A Z Z Y   V P N               │
│  Select language / Выберите язык       │
╰────────────────────────────────────────╯
  1. ru — Русский
  2. en — English
  3. de — Deutsch
  4. zh — 中文
  5. ja — 日本語
  6. ko — 한국어
EOF
        read -r -p "Language / Язык [1-6]: " answer
        case "$answer" in
            1|ru) normalized=ru ;; 2|en) normalized=en ;; 3|de) normalized=de ;;
            4|zh) normalized=zh ;; 5|ja) normalized=ja ;; 6|ko) normalized=ko ;;
            *) echo "Invalid language / Неверный язык" >&2; return 2 ;;
        esac
    fi
    INSTALL_LANG="${normalized:-ru}"
    case "$INSTALL_LANG" in
        ru) echo "Язык установки: Русский" ;;
        en) echo "Installation language: English" ;;
        de) echo "Installationssprache: Deutsch" ;;
        zh) echo "安装语言：中文" ;;
        ja) echo "インストール言語: 日本語" ;;
        ko) echo "설치 언어: 한국어" ;;
    esac
}

write_installed_language() {
    local target="$1"
    if ((DRY_RUN)); then
        printf '+ write-language %q %q\n' "$INSTALL_LANG" "$target"
        return 0
    fi
    printf '%s\n' "$INSTALL_LANG" >"$target"
    chmod 644 "$target"
}

validate_source_tree() {
    local required
    for required in mazzy-vpn install.sh LICENSE AUTHORS.md CHANGELOG.md SECURITY.md \
        PRIVACY.md \
        README.md README.ru.md README.en.md README.de.md README.zh.md \
        README.ja.md README.ko.md docs/ARCHITECTURE.en.md \
        docs/ARCHITECTURE.ru.md docs/DESKTOP.en.md docs/DESKTOP.ru.md \
        docs/DESKTOP_ROADMAP.en.md docs/DESKTOP_ROADMAP.ru.md \
        docs/PLATFORM_ROADMAP.en.md docs/PLATFORM_ROADMAP.ru.md \
        docs/FEATURE_PARITY.md docs/capabilities.json \
        desktop/README.md \
        tests/run.sh tests/audit-public.sh tests/check-capabilities.py \
        completions/mazzy-vpn systemd/vpnctl.service \
        systemd/vpnctl-health.service systemd/vpnctl-health.timer \
        systemd/vpnctl-test-recovery.service; do
        [[ -e "$SCRIPT_DIR/$required" ]] || {
            echo "Неполный дистрибутив: отсутствует $required" >&2
            return 1
        }
    done
}

choose_config_source() {
    local answer
    [[ -z "$CONFIG_SOURCE" && -t 0 && $ASSUME_YES -eq 0 &&
       $DEPS_ONLY -eq 0 && -z "$DESTDIR" ]] || return 0
    read -r -e -p "Дополнительная папка VPN-конфигов [Enter = пропустить]: " answer
    CONFIG_SOURCE="$answer"
}

validate_config_source() {
    [[ -z "$CONFIG_SOURCE" ]] && return 0
    [[ -d "$CONFIG_SOURCE" ]] || {
        echo "Папка конфигов не найдена: $CONFIG_SOURCE" >&2
        return 1
    }
    CONFIG_SOURCE="$(readlink -f -- "$CONFIG_SOURCE")"
}

run_preflight_tests() {
    ((SKIP_TESTS)) && {
        echo "Preflight-тесты пропущены (--skip-tests)."
        return 0
    }
    echo "==> $PRODUCT_NAME: syntax и встроенные regression-тесты"
    bash -n "$SCRIPT_DIR/mazzy-vpn" "$SCRIPT_DIR/vpnctl" \
        "$SCRIPT_DIR/install.sh" "$SCRIPT_DIR/tests/run.sh" \
        "$SCRIPT_DIR/setup_amnezia_vpn.sh" "$SCRIPT_DIR/stop_amnezia_vpn.sh"
    bash -n "$SCRIPT_DIR/completions/mazzy-vpn" "$SCRIPT_DIR/completions/vpnctl"
    "$SCRIPT_DIR/tests/run.sh"
}

import_config_source() {
    local target_root="$DESTDIR/etc/vpnctl/profiles"
    local -a args=(import-dir "$CONFIG_SOURCE")
    [[ -n "$CONFIG_SOURCE" ]] || return 0
    ((FORCE_CONFIGS)) && args+=(--force)
    if ((DRY_RUN)); then
        args+=(--dry-run)
    fi
    echo "==> Автоимпорт VPN-конфигов: $CONFIG_SOURCE"
    VPNCTL_ALLOW_UNPRIVILEGED=1 \
        VPNCTL_CONFIG_DIR="$target_root" \
        "$SCRIPT_DIR/mazzy-vpn" "${args[@]}"
}

post_install_checks() {
    local bin="/usr/local/bin/mazzy-vpn" failed=0
    ((SKIP_CHECKS)) && {
        echo "Post-install проверки пропущены (--skip-checks)."
        return 0
    }
    echo "==> $PRODUCT_NAME: post-install проверки"
    "$bin" version || failed=1
    [[ -x "$bin" && -L /usr/local/bin/vpnctl && -L /usr/local/bin/mazzyvpn ]] ||
        {
            echo "Не установлены CLI aliases." >&2
            failed=1
        }
    [[ -r /usr/local/lib/mazzy-vpn/README.ru.md &&
       -r /usr/local/lib/mazzy-vpn/README.en.md &&
       -r /usr/local/lib/mazzy-vpn/README.de.md &&
       -r /usr/local/lib/mazzy-vpn/README.zh.md &&
       -r /usr/local/lib/mazzy-vpn/README.ja.md &&
       -r /usr/local/lib/mazzy-vpn/README.ko.md &&
       -r /usr/local/lib/mazzy-vpn/docs/DESKTOP.en.md &&
       -r /usr/local/lib/mazzy-vpn/docs/DESKTOP.ru.md &&
       -r /usr/local/lib/mazzy-vpn/docs/DESKTOP_ROADMAP.en.md &&
       -r /usr/local/lib/mazzy-vpn/docs/DESKTOP_ROADMAP.ru.md &&
       -r /usr/local/lib/mazzy-vpn/docs/FEATURE_PARITY.md &&
       -r /usr/local/lib/mazzy-vpn/docs/capabilities.json &&
       -r /usr/local/lib/mazzy-vpn/docs/ARCHITECTURE.en.md &&
       -r /usr/local/lib/mazzy-vpn/docs/ARCHITECTURE.ru.md &&
       -r /usr/local/lib/mazzy-vpn/docs/PLATFORM_ROADMAP.en.md &&
       -r /usr/local/lib/mazzy-vpn/docs/PLATFORM_ROADMAP.ru.md &&
       -r /usr/local/lib/mazzy-vpn/LICENSE &&
       -r /usr/local/lib/mazzy-vpn/AUTHORS.md &&
       -r /usr/local/lib/mazzy-vpn/PRIVACY.md &&
       -r /usr/local/share/bash-completion/completions/mazzy-vpn ]] ||
        {
            echo "Не установлены документация или Bash completion." >&2
            failed=1
        }
    "$bin" status --json | grep -q '"schema_version":1' || {
        echo "Безопасный cache Dashboard не создан." >&2
        failed=1
    }
    systemd-analyze verify \
        /etc/systemd/system/vpnctl.service \
        /etc/systemd/system/vpnctl-health.service \
        /etc/systemd/system/vpnctl-health.timer \
        /etc/systemd/system/vpnctl-test-recovery.service || failed=1
    NO_COLOR=1 "$bin" self-test --offline || failed=1
    if ((LIVE_TEST)); then
        NO_COLOR=1 "$bin" self-test --live || failed=1
    fi
    ((failed == 0)) || {
        echo "$PRODUCT_NAME установлен, но post-install проверки нашли проблему." >&2
        echo "Диагностика: sudo mazzy-vpn doctor --fix" >&2
        return 1
    }
    echo "$PRODUCT_NAME: post-install проверки пройдены."
}

amnezia_kernel_ready() {
    command -v modprobe >/dev/null 2>&1 &&
        (modprobe -n amneziawg >/dev/null 2>&1 || [[ -d /sys/module/amneziawg ]])
}

amnezia_userspace_ready() {
    command -v amneziawg-go >/dev/null 2>&1
}

amnezia_ready() {
    command -v awg >/dev/null 2>&1 &&
        command -v awg-quick >/dev/null 2>&1 &&
        { amnezia_kernel_ready || amnezia_userspace_ready; }
}

confirm() {
    local prompt="$1" answer
    ((ASSUME_YES)) && return 0
    [[ -t 0 ]] || return 1
    read -r -p "$prompt [y/N]: " answer
    [[ "$answer" == "y" || "$answer" == "Y" ]]
}

os_value() {
    local key="$1"
    awk -F= -v wanted="$key" '
        $1 == wanted {
            value=substr($0, index($0, "=") + 1)
            gsub(/^"|"$/, "", value)
            print value
            exit
        }
    ' /etc/os-release
}

amnezia_ppa_supports_current_ubuntu() {
    local codename
    case "${VPNCTL_AMNEZIA_PPA_AVAILABLE:-auto}" in
        1|yes|true) return 0 ;;
        0|no|false) return 1 ;;
        auto) ;;
        *) echo "VPNCTL_AMNEZIA_PPA_AVAILABLE должен быть auto, 0 или 1" >&2; return 1 ;;
    esac
    codename="$(os_value VERSION_CODENAME)"
    [[ -n "$codename" ]] || return 1
    command -v curl >/dev/null 2>&1 || return 1
    curl -fsSI --connect-timeout 5 --max-time 10 \
        "$AMNEZIA_PPA_BASE/dists/$codename/Release" >/dev/null 2>&1
}

install_amnezia_userspace() {
    local build_dir tools_dir go_dir
    echo "Установка официального AmneziaWG userspace backend (без модуля ядра)."
    run apt-get install -y build-essential git golang-go || return 1
    if ((DRY_RUN)); then
        build_dir="/tmp/mazzy-vpn-amnezia-build"
    else
        build_dir="$(mktemp -d)"
    fi
    tools_dir="$build_dir/amneziawg-tools"
    go_dir="$build_dir/amneziawg-go"

    if ! command -v awg >/dev/null 2>&1 || ! command -v awg-quick >/dev/null 2>&1; then
        run git clone --quiet --depth 1 --branch "v$AWG_TOOLS_VERSION" \
            https://github.com/amnezia-vpn/amneziawg-tools.git "$tools_dir" || return 1
        run make -C "$tools_dir/src" || return 1
        run install -m 755 "$tools_dir/src/wg" /usr/local/bin/awg || return 1
        run install -m 755 "$tools_dir/src/wg-quick/linux.bash" \
            /usr/local/bin/awg-quick || return 1
    fi
    if ! amnezia_userspace_ready; then
        run git clone --quiet --depth 1 --branch "v$AWG_GO_VERSION" \
            https://github.com/amnezia-vpn/amneziawg-go.git "$go_dir" || return 1
        run go -C "$go_dir" build -trimpath -o "$build_dir/amneziawg-go" . ||
            return 1
        run install -m 755 "$build_dir/amneziawg-go" /usr/local/bin/amneziawg-go ||
            return 1
    fi
    if ((DRY_RUN == 0)); then
        rm -rf -- "$build_dir"
    fi
}

install_debian_dependencies() {
    local -a packages=(iproute2 curl ca-certificates openvpn wireguard-tools
        network-manager network-manager-l2tp strongswan xl2tpd iputils-ping
        netcat-openbsd)
    if ! run apt-get update; then
        echo "Предупреждение: один из APT-репозиториев не обновился." >&2
        echo "Продолжаю с успешно обновлёнными индексами; doctor покажет остаточные проблемы." >&2
    fi
    run env DEBIAN_FRONTEND=noninteractive apt-get install -y "${packages[@]}"

    if ! amnezia_ready; then
        if [[ "$(os_value ID)" == "ubuntu" ]]; then
            if amnezia_ppa_supports_current_ubuntu; then
                echo "AmneziaWG backend не готов; PPA поддерживает $(os_value VERSION_CODENAME)."
            elif confirm "PPA не поддерживает $(os_value VERSION_CODENAME). Установить официальный userspace backend?"; then
                install_amnezia_userspace ||
                    echo "AmneziaWG userspace установить не удалось; остальные протоколы установлены."
                return 0
            else
                echo "AmneziaWG пропущен; doctor продолжит сообщать об этой зависимости."
                return 0
            fi
            if confirm "Добавить PPA и установить amneziawg?"; then
                run apt-get install -y software-properties-common python3-launchpadlib gnupg2 \
                    "linux-headers-$(uname -r)"
                run add-apt-repository -y ppa:amnezia/ppa
                run apt-get update ||
                    echo "Не все APT-репозитории обновились; проверяю доступный пакет."
                run apt-get install -y amneziawg ||
                    echo "Kernel backend AmneziaWG не установился; можно повторить с userspace backend."
            else
                echo "AmneziaWG пропущен; doctor продолжит сообщать об этой зависимости."
            fi
        else
            echo "Автоустановка AmneziaWG для Debian отключена: Ubuntu PPA нельзя добавлять без явного решения."
            echo "Инструкция: https://github.com/amnezia-vpn/amneziawg-linux-kernel-module"
        fi
    fi
}

install_fedora_dependencies() {
    run dnf install -y iproute curl ca-certificates openvpn wireguard-tools \
        NetworkManager NetworkManager-l2tp strongswan xl2tpd iputils
    if ! amnezia_ready && confirm "Включить COPR amneziavpn/amneziawg?"; then
        run dnf install -y dnf-plugins-core
        run dnf copr enable -y amneziavpn/amneziawg
        run dnf install -y amneziawg-dkms amneziawg-tools
    fi
}

install_arch_dependencies() {
    run pacman -S --needed --noconfirm iproute2 curl ca-certificates openvpn \
        wireguard-tools networkmanager-l2tp strongswan xl2tpd iputils
    if ! amnezia_ready; then
        echo "AmneziaWG отсутствует. Установите amneziawg-tools и совместимый модуль из AUR вручную."
    fi
}

install_suse_dependencies() {
    run zypper --non-interactive install iproute2 curl ca-certificates openvpn \
        wireguard-tools NetworkManager-l2tp strongswan xl2tpd iputils
    if ! amnezia_ready; then
        echo "AmneziaWG отсутствует; автоматическая установка для openSUSE не поддерживается."
    fi
}

install_dependencies() {
    local id id_like
    id="$(os_value ID)"
    id_like="$(os_value ID_LIKE)"
    case " $id $id_like " in
        *" debian "*|*" ubuntu "*) install_debian_dependencies ;;
        *" fedora "*|*" rhel "*) install_fedora_dependencies ;;
        *" arch "*) install_arch_dependencies ;;
        *" suse "*) install_suse_dependencies ;;
        *)
            echo "Неизвестный дистрибутив '$id'. Установите openvpn, wireguard-tools," >&2
            echo "AmneziaWG tools/module и NetworkManager-l2tp вручную." >&2
            return 1
            ;;
    esac
}

copy_profiles() {
    local source_dir="$1" target_dir="$2" pattern="$3" source target
    [[ -d "$source_dir" ]] || return 0
    run install -d -m 700 "$target_dir"
    while IFS= read -r -d '' source; do
        target="$target_dir/$(basename -- "$source")"
        if [[ -e "$target" ]] && ! cmp -s "$source" "$target"; then
            echo "Сохранён существующий профиль (отличается): $target"
            run chmod 600 "$target"
            continue
        fi
        run install -m 600 -- "$source" "$target"
    done < <(find "$source_dir" -maxdepth 1 -type f -name "$pattern" -print0 | sort -z)
}

install_files() {
    local bin_dir="$DESTDIR/usr/local/bin"
    local lib_dir="$DESTDIR/usr/local/lib/mazzy-vpn"
    local docs_dir="$lib_dir/docs"
    local config_dir="$DESTDIR/etc/vpnctl/profiles"
    local unit_dir="$DESTDIR/etc/systemd/system"
    local completion_dir="$DESTDIR/usr/local/share/bash-completion/completions"

    run install -d -m 755 "$bin_dir" "$lib_dir" "$docs_dir" \
        "$unit_dir" "$completion_dir"
    run install -d -m 700 "$DESTDIR/etc/vpnctl" "$config_dir" \
        "$config_dir/amneziawg" "$config_dir/wireguard" \
        "$config_dir/openvpn" "$config_dir/l2tp"
    write_installed_language "$DESTDIR/etc/vpnctl/locale"
    run install -m 755 "$SCRIPT_DIR/mazzy-vpn" "$bin_dir/mazzy-vpn"
    run ln -sfn mazzy-vpn "$bin_dir/vpnctl"
    run ln -sfn mazzy-vpn "$bin_dir/mazzyvpn"
    run install -m 755 "$SCRIPT_DIR/install.sh" "$lib_dir/install.sh"
    run install -m 755 "$SCRIPT_DIR/setup_amnezia_vpn.sh" \
        "$lib_dir/setup_amnezia_vpn.sh"
    run install -m 755 "$SCRIPT_DIR/stop_amnezia_vpn.sh" \
        "$lib_dir/stop_amnezia_vpn.sh"
    run install -m 644 "$SCRIPT_DIR/README.md" "$lib_dir/README.md"
    run install -m 644 "$SCRIPT_DIR/README.ru.md" "$lib_dir/README.ru.md"
    run install -m 644 "$SCRIPT_DIR/README.en.md" "$lib_dir/README.en.md"
    run install -m 644 "$SCRIPT_DIR/README.de.md" "$lib_dir/README.de.md"
    run install -m 644 "$SCRIPT_DIR/README.zh.md" "$lib_dir/README.zh.md"
    run install -m 644 "$SCRIPT_DIR/README.ja.md" "$lib_dir/README.ja.md"
    run install -m 644 "$SCRIPT_DIR/README.ko.md" "$lib_dir/README.ko.md"
    run install -m 644 "$SCRIPT_DIR/docs/ARCHITECTURE.en.md" \
        "$docs_dir/ARCHITECTURE.en.md"
    run install -m 644 "$SCRIPT_DIR/docs/ARCHITECTURE.ru.md" \
        "$docs_dir/ARCHITECTURE.ru.md"
    run install -m 644 "$SCRIPT_DIR/docs/DESKTOP.en.md" \
        "$docs_dir/DESKTOP.en.md"
    run install -m 644 "$SCRIPT_DIR/docs/DESKTOP.ru.md" \
        "$docs_dir/DESKTOP.ru.md"
    run install -m 644 "$SCRIPT_DIR/docs/DESKTOP_ROADMAP.en.md" \
        "$docs_dir/DESKTOP_ROADMAP.en.md"
    run install -m 644 "$SCRIPT_DIR/docs/DESKTOP_ROADMAP.ru.md" \
        "$docs_dir/DESKTOP_ROADMAP.ru.md"
    run install -m 644 "$SCRIPT_DIR/docs/PLATFORM_ROADMAP.en.md" \
        "$docs_dir/PLATFORM_ROADMAP.en.md"
    run install -m 644 "$SCRIPT_DIR/docs/PLATFORM_ROADMAP.ru.md" \
        "$docs_dir/PLATFORM_ROADMAP.ru.md"
    run install -m 644 "$SCRIPT_DIR/docs/FEATURE_PARITY.md" \
        "$docs_dir/FEATURE_PARITY.md"
    run install -m 644 "$SCRIPT_DIR/docs/capabilities.json" \
        "$docs_dir/capabilities.json"
    run install -m 644 "$SCRIPT_DIR/LICENSE" "$lib_dir/LICENSE"
    run install -m 644 "$SCRIPT_DIR/AUTHORS.md" "$lib_dir/AUTHORS.md"
    run install -m 644 "$SCRIPT_DIR/CHANGELOG.md" "$lib_dir/CHANGELOG.md"
    run install -m 644 "$SCRIPT_DIR/SECURITY.md" "$lib_dir/SECURITY.md"
    run install -m 644 "$SCRIPT_DIR/PRIVACY.md" "$lib_dir/PRIVACY.md"
    run install -m 644 "$SCRIPT_DIR/systemd/vpnctl.service" "$unit_dir/vpnctl.service"
    run install -m 644 "$SCRIPT_DIR/systemd/vpnctl-health.service" "$unit_dir/vpnctl-health.service"
    run install -m 644 "$SCRIPT_DIR/systemd/vpnctl-health.timer" "$unit_dir/vpnctl-health.timer"
    run install -m 644 "$SCRIPT_DIR/systemd/vpnctl-test-recovery.service" \
        "$unit_dir/vpnctl-test-recovery.service"
    run install -m 644 "$SCRIPT_DIR/completions/mazzy-vpn" \
        "$completion_dir/mazzy-vpn"
    run ln -sfn mazzy-vpn "$completion_dir/vpnctl"
    run ln -sfn mazzy-vpn "$completion_dir/mazzyvpn"

    copy_profiles "$SCRIPT_DIR/conf/AMNEZIA" "$config_dir/amneziawg" '*.conf'
    copy_profiles "$SCRIPT_DIR/conf/amneziawg" "$config_dir/amneziawg" '*.conf'
    copy_profiles "$SCRIPT_DIR/conf/wireguard" "$config_dir/wireguard" '*.conf'
    copy_profiles "$SCRIPT_DIR/conf/openvpn" "$config_dir/openvpn" '*.ovpn'
    copy_profiles "$SCRIPT_DIR/conf/openvpn" "$config_dir/openvpn" '*.conf'
    copy_profiles "$SCRIPT_DIR/conf/l2tp" "$config_dir/l2tp" '*.nmconnection'

    if [[ -z "$DESTDIR" ]]; then
        run chown root:root "$bin_dir/mazzy-vpn" "$lib_dir/install.sh" \
            "$lib_dir/setup_amnezia_vpn.sh" "$lib_dir/stop_amnezia_vpn.sh" \
            "$lib_dir/README.md" "$lib_dir/README.ru.md" "$lib_dir/README.en.md" \
            "$lib_dir/README.de.md" "$lib_dir/README.zh.md" \
            "$lib_dir/README.ja.md" "$lib_dir/README.ko.md" \
            "$docs_dir/ARCHITECTURE.en.md" "$docs_dir/ARCHITECTURE.ru.md" \
            "$docs_dir/DESKTOP.en.md" "$docs_dir/DESKTOP.ru.md" \
            "$docs_dir/DESKTOP_ROADMAP.en.md" \
            "$docs_dir/DESKTOP_ROADMAP.ru.md" \
            "$docs_dir/PLATFORM_ROADMAP.en.md" \
            "$docs_dir/PLATFORM_ROADMAP.ru.md" \
            "$docs_dir/FEATURE_PARITY.md" "$docs_dir/capabilities.json" \
            "$lib_dir/LICENSE" "$lib_dir/AUTHORS.md" "$lib_dir/CHANGELOG.md" \
            "$lib_dir/SECURITY.md" "$lib_dir/PRIVACY.md" \
            "$unit_dir/vpnctl.service" \
            "$unit_dir/vpnctl-health.service" "$unit_dir/vpnctl-health.timer" \
            "$unit_dir/vpnctl-test-recovery.service" "$completion_dir/mazzy-vpn"
        run chown -R root:root "$DESTDIR/etc/vpnctl"
    fi
}

validate_source_tree
choose_install_language
choose_config_source
validate_config_source

if [[ -z "$DESTDIR" && $EUID -ne 0 && $DRY_RUN -eq 0 ]]; then
    echo "Запустите установщик через sudo: sudo ./install.sh" >&2
    exit 1
fi

if ((DEPS_ONLY == 0)); then
    if [[ -z "$DESTDIR" ]]; then
        run_preflight_tests
    fi
    install_files
    import_config_source
fi

if ((NO_DEPS == 0)) && [[ -z "$DESTDIR" ]]; then
    install_dependencies
fi

if [[ -z "$DESTDIR" && $DEPS_ONLY -eq 0 ]]; then
    run systemctl daemon-reload
    run systemctl enable vpnctl-test-recovery.service
    run systemctl enable --now vpnctl-health.timer
    run systemctl restart vpnctl-health.timer
    run /usr/local/bin/mazzy-vpn _refresh-dashboard-cache
    if ((DRY_RUN)); then
        echo
        echo "Dry-run завершён: изменения не применялись."
    else
        post_install_checks
        echo
        echo "$PRODUCT_NAME установлен и проверен."
        echo "  mazzy-vpn                 # интерактивное TUI"
        echo "  mazzy-vpn self-test       # полная самодиагностика"
        echo "  mazzy-vpn list"
        echo "  sudo mazzy-vpn doctor --fix"
        echo "Aliases: vpnctl, mazzyvpn"
    fi
elif ((DEPS_ONLY == 0)); then
    echo "$PRODUCT_NAME установлен в staging-каталог: $DESTDIR"
fi
