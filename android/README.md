# 原生 App（Android）

控制台的原生实现。后端仍是同一套 API，网页端不受影响；功能与使用说明见 [Android App 文档](../docs/guide/android-app.md)。

## 构建

需要 JDK 17+、Android SDK（`platforms;android-34` 与 `build-tools`）以及 Gradle：

```bash
./build.sh          # debug APK -> app/build/outputs/apk/debug/app-debug.apk
./build.sh install  # 构建后 adb install -r
```

最低支持 Android 8.0（API 26）；通行密钥需要 Android 9（API 28）及以上，低版本仍保留密码登录。

## 通行密钥

App 使用 Android Credential Manager（`androidx.credentials`）读写通行密钥，后端下发的是未包裹的 WebAuthn 字典，因此 App 只做透传，不参与挑战的编解码。

Android 要求面板通过数字资产链接证明域名归属，才能为该域名创建 / 使用通行密钥：

- 面板需以 HTTPS 访问，且依赖方 ID 与访问域名一致；
- `https://<域名>/.well-known/assetlinks.json` 必须列出本 App 的包名（`com.tunnelmanager.app`）与签名证书 SHA-256 指纹。后端已提供该端点，包名与指纹在管理后台「系统设置 → 通行密钥」维护，默认预填仓库共用调试密钥的指纹。

更换签名密钥（例如改用 release 签名）后，用 `apksigner verify --print-certs app/build/outputs/apk/debug/app-debug.apk` 或 `keytool -list -v -keystore <keystore> | grep SHA256` 取得新指纹并填入面板，否则系统会拒绝通行密钥流程。打包完成后 APK 会镜像一份到 `~/tunnel/TunnelManager-debug.apk` 与 `~/storage/downloads/TunnelManager-debug.apk`（目录存在时）。Gradle daemon 常驻时增量构建约十几秒，冷构建约一分半。

## 签名

默认使用本机调试密钥 `~/.android/debug.keystore`，可用环境变量 `ANDROID_DEBUG_KEYSTORE` 覆盖。两条构建路径共用同一把密钥，因此互相覆盖安装不会因签名不一致失败。

## 目录

```text
app/src/main/java/com/tunnelmanager/app/
├── MainActivity.java      # 登录、通行密钥、Turnstile 与启动闪屏
├── ConsoleActivity.java   # 控制台外壳：侧边栏 / 底部标签与主题切换
├── Nav.java               # 导航表，对应 frontend/src/navigation.ts
├── Passkey.java           # 通行密钥：Credential Manager 封装
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
