# 原生 App（Android）

控制台的原生实现。后端仍是同一套 API，网页端不受影响；功能与使用说明见 [Android App 文档](https://docs.kukie.cn/guide/android-app)。

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

## WebView 认证隔离

面板会话只注入配置服务器的 HTTP(S) 来源，协议、主机和端口必须一致。外部认证页在独立 WebView 窗口中打开，共享一次性认证 cookie，但没有面板令牌注入或原生桥；回到可信面板来源时再交回面板窗口。其他外部页面也不会收到面板会话。

## 本轮控制台更新（2026-11-14）

原生 App 使用黑白灰深浅主题、实体内容面板和中性导航。
所有原生页面使用公共组件的新间距与圆角；顶栏提供当前页面位置、独立绘制的刷新按钮、主题与账户入口，主体不再重复页面名称。
刷新复用各页面原有读取逻辑，保留权限过滤与编辑校验；隧道页移除重复刷新入口。

控制面板与监控列表分别并行读取概览和服务列表，先完成的区域立即展示，刷新失败时保留已有内容。
隧道、详情、账户、管理后台、TG、IP 优选及助手的独立读取通过有界线程池并行执行。
保存会话的启动直接进入控制台，取消进入控制台前的装饰等待。

原生 API 请求与回调绑定提交时的服务器和会话；切换账户或服务器后，旧请求不会借用新令牌或刷新新账户。
认证请求禁止自动重定向；公开图片不携带面板令牌。上传文件名与 MIME 去除头注入风险，响应限制为 8 MiB。
权限不足（403）保留会话，只有认证失败（401）触发退出。
登录密码不进入 Activity 保存状态；系统云备份和设备迁移排除应用数据，避免迁移持久会话。

运行 `node --test android/tools/*.test.mjs` 可验证 API 来源、并行读取、旧会话回调、重定向、上传边界和既有业务逻辑。
本轮按用户要求只进行代码与构建检查，视觉验收由用户完成。

## 标题与弹层调整（2026-11-15）

顶栏只显示实际页面位置、刷新、主题和账户操作，去掉常驻站点名。
控制面板、监控、隧道和登录页删除宣传口号，页面使用「控制面板」「服务监控」等功能名称；
保留绑定条件、会话隐私、授权范围和编辑校验等操作必要说明。

公共 `Sheet` 使用 Material `BottomSheetDialog`，所有调用入口按场景明确指定单个固定高度：

- 30%：新建快捷导航、删除 / 重试 / 执行确认、简短代理与审计筛选。
- 60%：更多导航、隧道操作、历史会话、CNAME / 区域 / 类型 / TTL / 用户组选择、普通创建与回源表单。
- 80%：监控异常明细、公开页 / 服务配置、DNS 编辑与批量操作、助手任务与模型设置。

比例基于完整活动窗口高度；每个场景只使用其指定的 30% / 60% / 80% 高度，没有拖动把手或切档操作。
内部使用 `NestedScrollView`，内容视口与场景高度一致，底部操作保持在可见范围。
键盘由窗口 `adjustResize` 避让，只给导航栏安全边距；剩余空间不足时限制高度，关闭键盘后恢复同一场景高度。
旋转时重新计算同一场景高度和窗口宽度；嵌套弹层关闭后恢复上一层的返回键处理。

`node --test android/tools/sheet-sizing.test.mjs` 验证各场景固定比例、键盘空间限制与旋转重算。
本轮仍只验证代码、逻辑与 APK 构建，视觉和手势体验由用户验收。

## 黑白主题与控件统一（2026-11-15）

默认动作、成功状态、导航、玻璃、文字、边线与 ripple 均使用中性灰；仅异常和警告保留红 / 橙语义。
配色继续以 `palette-tokens.json` 为唯一来源，并重新生成 Java 与 XML，保留已保存的深浅主题偏好。
移除彩色启动覆盖与装饰渐变，启动背景和启动器标记也使用中性颜色。

所有控制台页主体删除与顶栏重复的通用页面标题；真实隧道名、监控名、卡片与章节标题保留。
登录与账户菜单共用 App 样式的语言选择列表，固定 30% 高度并显示当前选择，替代系统选择对话框。
语言偏好沿用原来的保存机制，登录 XML 也应用当前语言。

弹层禁止拖动、半展开及无障碍展开 / 收起动作。普通小确认弹窗继续按内容适应高度。
新建隧道的填写场景固定 60%，成功后凭据详情固定 80%，这是任务阶段变更。
