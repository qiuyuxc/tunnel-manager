# 原生 App（Android）

控制台的原生实现。后端仍是同一套 API，网页端不受影响；功能与使用说明见 [Android App 文档](../docs/guide/android-app.md)。

## 构建

需要 JDK 17+、Android SDK（`platforms;android-34` 与 `build-tools`）以及 Gradle：

```bash
./build.sh          # debug APK -> app/build/outputs/apk/debug/app-debug.apk
./build.sh install  # 构建后 adb install -r
```

最低支持 Android 8.0（API 26）；通行密钥需要 Android 9（API 28）及以上，低版本仍保留密码登录。

## 原生设计

原生配色与 Web 控制台独立，源文件是 `android/palette-tokens.json`。
修改后从仓库根目录运行 `node android/tools/gen-tokens.mjs`；不要手改生成的 `Palette.java`
或 `native_palette.xml`。构建会调用 Node.js 检查生成文件是否同步。
运行 `node --test android/tools/palette-tokens.test.mjs` 可检查深浅主题的完整性、文字对比度和生成结果。

运行 `node --test android/tools/*.test.mjs`
可检查配色、隧道筛选、监控历史和项目版本一致性；Java 测试需要本机 JDK，可用 `JAVA_HOME` 指定。
监控摘要使用真实七天统计，历史图按单个服务的抽样时间绘制，不等同于连续检查记录。

关于页从已安装包读取 App 版本及构建号，服务端版本单独展示，不要求两端同时升级。
项目发布时同步 Android、后端、前端和文档的版本字段，并更新中英文版本历史。
打包前运行版本一致性测试；发布附件使用校验过签名和版本的 APK，不使用旧的本地镜像副本。

底栏采用静态透光材质，不执行实时截图、模糊或折射。账户菜单中的「轻量效果」可关闭非必要动画，
系统关闭动画时也会同步降级。深浅主题在现有视图上重新着色，不重建 Activity。

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
├── Palette.java           # 由 palette-tokens.json 生成，勿手改
└── *Fragment.java         # 各原生页面

app/src/main/res/          # 布局、drawable 与主题（values / values-night）
```

原生主题色以 `android/palette-tokens.json` 为唯一来源，改色后重新生成 Java 与 XML 资源：

```bash
node android/tools/gen-tokens.mjs
```

## 备用构建路径

`tools/build-nogradle.sh` 不依赖 Gradle/AGP，直接用 `aapt2 + javac + d8 + apksigner` 打包，留作 Gradle 或 SDK 出问题时的兜底，日常不用。
