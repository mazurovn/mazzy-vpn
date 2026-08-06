#!/usr/bin/env bash
set -Eeuo pipefail
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
python3 "$root/tests/check-android-contract.py"
if [[ ! -x "$root/android/gradlew" || -z "${ANDROID_HOME:-}${ANDROID_SDK_ROOT:-}" ]]; then
  printf '%s\n' 'SKIP: Android SDK/Gradle wrapper unavailable; no APK build claimed.'
  exit 0
fi
cd "$root/android"
./gradlew --no-daemon testDebugUnitTest assembleDebug
