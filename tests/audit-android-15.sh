#!/usr/bin/env bash
set -Eeuo pipefail

root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
export ANDROID_HOME="${ANDROID_HOME:-$root/.mazzy/android-sdk}"
export ANDROID_SDK_ROOT="${ANDROID_SDK_ROOT:-$ANDROID_HOME}"
export ANDROID_USER_HOME="${ANDROID_USER_HOME:-$root/.mazzy/android-user}"
export GRADLE_USER_HOME="${GRADLE_USER_HOME:-$root/.mazzy/gradle}"
export TMPDIR="${TMPDIR:-$root/.mazzy/work/sdk-tmp}"
mkdir -p "$ANDROID_USER_HOME" "$GRADLE_USER_HOME" "$TMPDIR"
clean_streak=0
for iteration in $(seq 1 15); do
  finding=0
  python3 "$root/tests/check-android-contract.py" >/dev/null || finding=1
  xmllint --noout "$root/android/app/src/main/AndroidManifest.xml" || finding=1
  grep -q 'startForeground' "$root/android/app/src/main/kotlin/com/mazzy/vpn/MazzyVpnService.kt" || finding=1
  grep -q 'builder.establish()' "$root/android/app/src/main/kotlin/com/mazzy/vpn/MazzyVpnService.kt" || finding=1
  grep -q 'GoBackend.awgTurnOn' "$root/android/app/src/main/kotlin/com/mazzy/vpn/MazzyVpnService.kt" || finding=1
  grep -q 'requireCurrentGeneration' "$root/android/app/src/main/kotlin/com/mazzy/vpn/MazzyVpnService.kt" || finding=1
  if ((finding == 0)); then
    clean_streak=$((clean_streak + 1))
    printf 'iteration=%02d result=clean clean_streak=%d\n' "$iteration" "$clean_streak"
  else
    clean_streak=0
    printf 'iteration=%02d result=finding clean_streak=0\n' "$iteration"
  fi
done

if [[ ! -x "$root/android/gradlew" || ( -z "${ANDROID_HOME:-}${ANDROID_SDK_ROOT:-}" && ! -r "$root/android/local.properties" ) ]]; then
  printf '%s\n' 'BLOCKED: runtime/device gates not executed; Gradle wrapper or Android SDK is absent.' >&2
  exit 2
fi
((clean_streak >= 5))
"$root/tests/check-android-build.sh"
