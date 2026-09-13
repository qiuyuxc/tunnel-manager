# Android App

原生 Android 客户端。后端仍然是同一套 REST API，App 只是把控制台搬到手机上，网页端不受影响。

App 目前从源码构建（仓库内 `android/`），暂未随 GitHub Releases 附带发布包。

## 能做什么

- 控制面板、服务监控、隧道管理、域名绑定、DNS 管理、IP 优选实验室、Telegram 机器人、通知、管理后台、全局设置、账户与关于页均为原生实现
- 窄屏采用底部标签导航（概览 / 监控 / DNS / 更多）与半屏弹窗，长表单不再横向挤压
- 登录页原生，人机验证（Cloudflare Turnstile）内嵌在同一页面，全程不跳转浏览器
- 监控状态变化通过系统通知渠道推送到手机，不需要后台常驻网页
- 亮色 / 暗色跟随 App 内主题偏好，与网页端共用一套配色

## 登录

1. 打开 App，填写面板地址（例如 `https://tunnel.example.com`，也可带端口）。
2. 输入账号与密码；若面板开启了人机验证，验证组件会直接显示在登录表单下方。
3. 登录成功后进入控制台；如需切换账号，在顶栏头像处退出登录。

如果验证组件加载失败（例如面板所在网络无法访问 Cloudflare），可以点 **改用网页版登录**，退回内置 WebView 完成登录。

## 通知

首次进入 **通知** 页面时按提示授予系统通知权限。授权后 App 会在后台低频轮询 `/api/alerts`，把监控状态变化转成系统通知；关闭通知权限即回到静默状态，不影响网页端的邮件告警。

## 主题

App 内的主题偏好独立于系统深色模式：在侧边栏底部或顶栏头像菜单切换 **亮色 / 暗色**。启动闪屏也会按当前偏好选择日间或夜间插画。

## 构建

需要 JDK 17+、Android SDK（`platforms;android-34` 与 `build-tools`）以及 Gradle：

```bash
cd android
./build.sh            # debug APK -> app/build/outputs/apk/debug/app-debug.apk
./build.sh install    # 构建后 adb install -r 到已连接设备
```

签名沿用本机调试密钥（默认 `~/.android/debug.keystore`，可用环境变量 `ANDROID_DEBUG_KEYSTORE` 覆盖），因此重复安装不会因为签名不同而冲突。

最低支持 Android 8.0（API 26）。
