#!/usr/bin/env bash
set -Eeuo pipefail

root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
clean_streak=0
for iteration in $(seq 1 15); do
  finding=0
  python3 "$root/tests/check-android-contract.py" >/dev/null || finding=1
  xmllint --noout "$root/android/app/src/main/AndroidManifest.xml" || finding=1
  grep -q 'startForeground' "$root/android/app/src/main/kotlin/com/mazzy/vpn/MazzyVpnService.kt" || finding=1
  grep -q 'establish()' "$root/android/app/src/main/kotlin/com/mazzy/vpn/MazzyVpnService.kt" && finding=1 || true
  if ((finding == 0)); then
    clean_streak=$((clean_streak + 1))
    printf 'iteration=%02d result=clean clean_streak=%d\n' "$iteration" "$clean_streak"
  else
    clean_streak=0
    printf 'iteration=%02d result=finding clean_streak=0\n' "$iteration"
  fi
done

if [[ ! -x "$root/android/gradlew" || -z "${ANDROID_HOME:-}${ANDROID_SDK_ROOT:-}" ]]; then
  printf '%s\n' 'BLOCKED: runtime/device gates not executed; Gradle wrapper or Android SDK is absent.' >&2
  exit 2
fi
((clean_streak >= 5))
