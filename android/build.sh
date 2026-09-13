#!/data/data/com.termux/files/usr/bin/bash
#
# Builds the native Android app with Gradle.
#
# This project runs AGP on aarch64 Termux, which Google does not support: the
# build-tools Google publishes are x86_64-only. The SDK tree at ~/.android-sdk
# therefore carries aarch64 replacements for aapt2, aidl and zipalign, and
# gradle.properties points AGP at that aapt2 via android.aapt2FromMavenOverride.
# Setup notes live in android/README.md.
#
#   ./build.sh            debug APK -> app/build/outputs/apk/debug/app-debug.apk
#   ./build.sh install    build, then adb install -r onto a connected device
#
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
cd "$HERE"

# Gradle needs a real JDK; Termux keeps it out of PATH by default.
export JAVA_HOME="${JAVA_HOME:-/data/data/com.termux/files/usr/lib/jvm/java-21-openjdk}"
export PATH="$JAVA_HOME/bin:$PATH"

APK="$HERE/app/build/outputs/apk/debug/app-debug.apk"

# The daemon is what makes an incremental rebuild ~15s instead of ~90s, so it
# stays on by default. Pass --no-daemon through for one-shot runs.
gradle "${@:2}" :app:assembleDebug

SIZE=$(du -h "$APK" | cut -f1)
echo
echo "✅ 打包完成：$APK  ($SIZE)"

# Gradle writes into app/build/, which is not where anyone looks. Mirror the
# APK to the two spots the app has always been picked up from.
for dest in "$HOME/tunnel/TunnelManager-debug.apk" "$HOME/storage/downloads/TunnelManager-debug.apk"; do
  dir="$(dirname "$dest")"
  [ -d "$dir" ] || continue
  cp "$APK" "$dest" 2>/dev/null && echo "   → $dest" || echo "   → 跳过 $dest（无权限）"
done

if [ "${1:-}" = "install" ]; then
  echo "==> adb install"
  adb install -r "$APK"
fi
