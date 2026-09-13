# 原生 App（Android）

控制台的原生实现。后端仍是同一套 API，网页端不受影响；功能与使用说明见 [Android App 文档](../docs/guide/android-app.md)。

## 构建

需要 JDK 17+、Android SDK（`platforms;android-34` 与 `build-tools`）以及 Gradle：

```bash
./build.sh          # debug APK -> app/build/outputs/apk/debug/app-debug.apk
./build.sh install  # 构建后 adb install -r
```

最低支持 Android 8.0（API 26）。打包完成后 APK 会镜像一份到 `~/tunnel/TunnelManager-debug.apk` 与 `~/storage/downloads/TunnelManager-debug.apk`（目录存在时）。Gradle daemon 常驻时增量构建约十几秒，冷构建约一分半。

## 签名

默认使用本机调试密钥 `~/.android/debug.keystore`，可用环境变量 `ANDROID_DEBUG_KEYSTORE` 覆盖。两条构建路径共用同一把密钥，因此互相覆盖安装不会因签名不一致失败。

## 目录

```text
app/src/main/java/com/tunnelmanager/app/
├── MainActivity.java      # 登录、Turnstile 与启动闪屏
├── ConsoleActivity.java   # 控制台外壳：侧边栏 / 底部标签与主题切换
├── Nav.java               # 导航表，对应 frontend/src/navigation.ts
├── Palette.java           # 由 styles.css 生成，勿手改
└── *Fragment.java         # 各原生页面

app/src/main/res/          # 布局、drawable 与主题（values / values-night）
```

主题色以 `frontend/src/styles.css` 为唯一来源，改色后重新生成 Java 侧调色板：

```bash
node android/tools/gen-tokens.mjs
```

## 备用构建路径

`tools/build-nogradle.sh` 不依赖 Gradle/AGP，直接用 `aapt2 + javac + d8 + apksigner` 打包，留作 Gradle 或 SDK 出问题时的兜底，日常不用。
