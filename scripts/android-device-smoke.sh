#!/usr/bin/env bash
set -Eeuo pipefail

root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
package="com.mazzy.vpn"
apk="$root/android/app/build/outputs/apk/debug/app-debug.apk"
profile=""
mode="install"

while (($#)); do
  case "$1" in
    --apk) apk="${2:?missing APK path}"; shift 2 ;;
    --profile) profile="${2:?missing profile path}"; shift 2 ;;
    --uninstall) mode="uninstall"; shift ;;
    *) printf 'Unknown argument: %s\n' "$1" >&2; exit 64 ;;
  esac
done

sdk_root="${ANDROID_SDK_ROOT:-${ANDROID_HOME:-}}"
if [[ -z "$sdk_root" && -r "$root/android/local.properties" ]]; then
  sdk_root="$(sed -n 's/^sdk\.dir=//p' "$root/android/local.properties" | head -n1)"
fi
adb="$sdk_root/platform-tools/adb"
[[ -x "$adb" ]] || { printf '%s\n' 'adb not found; configure ANDROID_SDK_ROOT.' >&2; exit 69; }

mapfile -t devices < <("$adb" devices | awk 'NR>1 && $2 == "device" {print $1}')
[[ ${#devices[@]} == 1 ]] || {
  printf 'Expected exactly one authorized Android device, found %d.\n' "${#devices[@]}" >&2
  "$adb" devices -l >&2
  exit 69
}

if [[ "$mode" == uninstall ]]; then
  "$adb" uninstall "$package"
  printf '%s\n' 'Mazzy VPN Android removed; Android restores the previous network owner.'
  exit 0
fi

[[ -f "$apk" && ! -L "$apk" ]] || { printf 'APK not found: %s\n' "$apk" >&2; exit 66; }
"$adb" install -r "$apk"
"$adb" shell am start -W -n "$package/.MainActivity"
"$adb" shell dumpsys package "$package" |
  grep -E 'versionCode=|versionName=|targetSdk=' | head -n 6

if [[ -n "$profile" ]]; then
  [[ -f "$profile" && ! -L "$profile" ]] || { printf 'Profile not found: %s\n' "$profile" >&2; exit 66; }
  size=$(stat -c '%s' "$profile")
  ((size > 0 && size <= 262144)) || { printf '%s\n' 'Profile must be 1..262144 bytes.' >&2; exit 65; }
  remote='/sdcard/Download/mazzy-device-test.conf'
  "$adb" push "$profile" "$remote" >/dev/null
  printf '%s\n' 'On the phone: Import profile -> Download -> mazzy-device-test.conf.'
  printf '%s\n' 'After import, delete the plaintext staging copy with:'
  printf '  %q shell rm -f -- %q\n' "$adb" "$remote"
fi

printf '%s\n' 'Next: grant VPN permission, Start VPN, then verify handshake/IP/DNS and Stop VPN.'
