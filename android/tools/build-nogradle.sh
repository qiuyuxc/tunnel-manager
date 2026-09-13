#!/data/data/com.termux/files/usr/bin/bash
#
# Builds the Android shell without Gradle.
#
# This app is one Activity and four drawables, so the Gradle/AGP stack buys
# nothing but several hundred megabytes of downloads. The four tools below are
# the same ones AGP drives under the hood: aapt2 links resources, javac compiles,
# d8 produces the dex, apksigner signs.
#
#   ./build.sh            debug build -> build/tunnel-manager-debug.apk
#   ./build.sh release    also runs the Release build (same key, minified off)
#   ./build.sh install    build then `adb install -r` onto a connected device
#
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
cd "$HERE"

SDK="${ANDROID_SDK_ROOT:-$HOME/.android-sdk}"
PLATFORM="$SDK/platforms/android-34/android.jar"
MIN_SDK=26
TARGET_SDK=34
APP=app/src/main
OUT=build

KEYSTORE="${ANDROID_DEBUG_KEYSTORE:-$HOME/.android/debug.keystore}"
KEY_ALIAS=androiddebugkey
KEY_PASS=android

for tool in aapt2 javac d8 apksigner; do
  command -v "$tool" >/dev/null 2>&1 || { echo "缺少 $tool，先跑：pkg install openjdk-17 aapt2 d8 apksigner" >&2; exit 1; }
done
[ -f "$PLATFORM" ] || { echo "找不到 android.jar：$PLATFORM" >&2; exit 1; }

echo "==> 清理"
rm -rf "$OUT"
mkdir -p "$OUT"/{res,gen,classes,dex,apk}

echo "==> 1/5 编译资源 (aapt2 compile)"
aapt2 compile --dir "$APP/res" -o "$OUT/res.zip"

echo "==> 2/5 链接资源与清单 (aapt2 link)"
aapt2 link \
  -o "$OUT/apk/base.apk" \
  -I "$PLATFORM" \
  --manifest "$APP/AndroidManifest.xml" \
  --java "$OUT/gen" \
  --min-sdk-version "$MIN_SDK" \
  --target-sdk-version "$TARGET_SDK" \
  --version-code 1 \
  --version-name 1.0 \
  --no-static-lib-packages \
  "$OUT/res.zip"

echo "==> 3/5 编译 Java (javac)"
find "$APP/java" "$OUT/gen" -name '*.java' > "$OUT/sources.txt"
# android.jar goes on the regular classpath rather than -bootclasspath: javac
# resolves the lambda bootstrap (java.lang.invoke.LambdaMetafactory) against the
# boot classpath, and android.jar's stub is not a drop-in for it. d8 does the
# actual lambda desugaring, and it is validated against --lib below, so the only
# cost is that javac would let a JDK-only API through unnoticed.
javac \
  -source 8 -target 8 \
  -Xlint:-options \
  -classpath "$PLATFORM" \
  -encoding UTF-8 \
  -d "$OUT/classes" \
  @"$OUT/sources.txt"

echo "==> 4/5 生成 dex (d8)"
find "$OUT/classes" -name '*.class' > "$OUT/classes.txt"
d8 --lib "$PLATFORM" --min-api "$MIN_SDK" --output "$OUT/dex" @"$OUT/classes.txt"

( cd "$OUT/dex" && zip -q -X "../apk/base.apk" classes.dex )

echo "==> 5/5 签名 (apksigner)"
if [ ! -f "$KEYSTORE" ]; then
  echo "    生成调试密钥：$KEYSTORE"
  mkdir -p "$(dirname "$KEYSTORE")"
  keytool -genkeypair -v \
    -keystore "$KEYSTORE" \
    -alias "$KEY_ALIAS" \
    -keyalg RSA -keysize 2048 -validity 10000 \
    -storepass "$KEY_PASS" -keypass "$KEY_PASS" \
    -dname "CN=Android Debug,O=Android,C=US" >/dev/null 2>&1
fi

APK="$OUT/tunnel-manager-debug.apk"
apksigner sign \
  --ks "$KEYSTORE" \
  --ks-key-alias "$KEY_ALIAS" \
  --ks-pass "pass:$KEY_PASS" \
  --key-pass "pass:$KEY_PASS" \
  --v1-signing-enabled true \
  --v2-signing-enabled true \
  --out "$APK" \
  "$OUT/apk/base.apk"

apksigner verify --print-certs "$APK" | head -3

SIZE=$(du -h "$APK" | cut -f1)
echo
echo "✅ 打包完成：$HERE/$APK  ($SIZE)"
echo "   装机：adb install -r $HERE/$APK"

if [ "${1:-}" = "install" ]; then
  echo "==> adb install"
  adb install -r "$HERE/$APK"
fi
