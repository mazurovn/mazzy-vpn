#!/usr/bin/env bash
set -Eeuo pipefail
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
python3 "$root/tests/check-android-contract.py"
sdk_root="${ANDROID_SDK_ROOT:-${ANDROID_HOME:-}}"
if [[ -z "$sdk_root" && -r "$root/android/local.properties" ]]; then
  sdk_root="$(sed -n 's/^sdk\.dir=//p' "$root/android/local.properties" | head -n1)"
fi
if [[ ! -x "$root/android/gradlew" ||
      ! -d "$sdk_root/platforms/android-36" ||
      ! -d "$sdk_root/build-tools/36.0.0" ||
      ! -d "$sdk_root/ndk/26.1.10909125" ||
      ! -d "$sdk_root/cmake/3.22.1" ]]; then
  printf '%s\n' 'SKIP: Android SDK/Gradle wrapper unavailable; no APK build claimed.'
  exit 0
fi
cd "$root/android"
./gradlew --no-daemon --offline testDebugUnitTest lintDebug assembleDebug

build_tools="$sdk_root/build-tools/36.0.0"
apk="$root/android/app/build/outputs/apk/debug/app-debug.apk"
test -s "$apk"
"$build_tools/zipalign" -c -P 16 4 "$apk"
"$build_tools/apksigner" verify "$apk"
service_count=$("$build_tools/aapt" dump xmltree "$apk" AndroidManifest.xml |
  grep -c 'E: service')
[[ "$service_count" == 1 ]]
for abi in arm64-v8a armeabi-v7a x86 x86_64; do
  unzip -l "$apk" "lib/$abi/libwg-go.so" | grep -F "lib/$abi/libwg-go.so" >/dev/null
done
unzip -l "$apk" | grep -F 'assets/amneziawg-android-APACHE-2.0.txt' >/dev/null
unzip -l "$apk" | grep -F 'assets/amneziawg-go-MIT.txt' >/dev/null
if unzip -p "$apk" classes.dex | strings | grep -E 'RootShell|AwgQuickBackend|ToolsInstaller' >/dev/null; then
  printf '%s\n' 'FAIL: root/shell backend leaked into Android APK' >&2
  exit 1
fi
printf 'PASS: Android APK artifact gate: %s\n' "$(sha256sum "$apk" | cut -d' ' -f1)"
