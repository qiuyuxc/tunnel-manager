<div align="center"><img src="frontend/public/icon.webp" width="72" alt="Tunnel Manager" /></div>

# Tunnel Manager · 单用户精简版（slim）

[English](README.md) | 简体中文

> **这是单用户精简版**（分支 `slim/single-user`）。它是 Tunnel Manager 面向单管理员自托管场景的裁剪版本——去掉了注册、用户组、管理后台、审计日志与 Telegram 远程控制。需要完整的多用户版本请看 [`main`](https://github.com/qiuyuxc/tunnel-manager/tree/main) 分支。

采用黑白主题的 Cloudflare Tunnel 可视化管理面板，附带原生 Android 客户端。通过 Web UI 管理隧道、绑定域名、配置 DNS 优选与回退源，提供服务可用性监控与可分享的公开状态页——面向单个管理员，支持密码 + 通行密钥 / 双因素登录。

> 📖 **在线文档**：**[https://docs.kukie.cn](https://docs.kukie.cn)** · 精简版：[单用户精简版](https://docs.kukie.cn/guide/slim-edition)

## 相比完整版的差异

| 移除 | 保留 |
| --- | --- |
| 多用户注册、邀请码、用户组与权限 | 单个管理员（安装引导时设置） |
| 管理后台（用户 / 组 / 邀请码管理） | 系统设置迁入面板内的**全局设置**页 |
| 管理员操作审计日志 | 登录限流、Turnstile、通行密钥 / TOTP 加固 |
| Telegram Bot **远程控制** | Telegram **通知**（对外告警推送） |

隧道、DNS、监控与认证的共用能力随完整版同步，保留单管理员系统设置。

## 功能一览

| 模块 | 能力 | 文档 |
| --- | --- | --- |
| 隧道管理 | 新建 / 删除隧道，Ingress 路由增删改，删除可联动清理 DNS | [隧道管理与路由](https://docs.kukie.cn/guide/tunnels) |
| 域名绑定 | 简化直连与 SaaS 优选双模式，批量绑定逐组独立配置 | [域名绑定模式](https://docs.kukie.cn/guide/domain-binding) · [批量绑定](https://docs.kukie.cn/guide/batch-binding) |
| DNS 管理 | A / AAAA / CNAME / TXT / MX / NS / SRV / CAA / PTR 增删改查，结构化字段编辑、批量新增预览与逐条结果 | [DNS 记录管理](https://docs.kukie.cn/guide/dns-management) |
| AI 助手 | 支持 Chat Completions 与 OpenAI Responses、流式输出、推理摘要及可编辑任务参数，人工核对、明确确认后执行；会话、任务和加密密钥由后端保存 | [AI 助手](https://docs.kukie.cn/guide/ai-assistant) |
| IP 优选实验室 | 独立 TXT 域名验证、HTTP 探测与诊断、动态预算和限速，可选定时执行与华为云 DNS 更新 | [IP 优选实验室](https://docs.kukie.cn/guide/lab-ip-selector) |
| 服务监控 | HTTP / TCP / ICMP 探测，多目标挂载，真实近 24 小时时间轴（48 个半小时桶），近期逐次检查单独展示 | [服务监控](https://docs.kukie.cn/guide/monitors-status) |
| 告警 | 仅在服务状态变化时通知，支持邮件（SMTP）或 Telegram 通知 Bot | [邮件服务与告警](https://docs.kukie.cn/guide/email-alerts) |
| 公开状态页 | 免登录分享，支持短路径、自定义域名、直连 Tunnel 与优选 CNAME，自定义域名仅开放对应状态页 | [公开状态页](https://docs.kukie.cn/guide/monitors-status#公开状态页) |
| Cloudflare 连接 | OAuth 2.0（PKCE）自动刷新令牌，多账户切换；兼容静态 Token | [OAuth 连接](https://docs.kukie.cn/guide/cloudflare-oauth) |
| Android App | 同一套 API 的原生客户端：概览、监控、隧道绑定、DNS、IP 优选实验室与通知，底部标签导航 + 系统通知 | [Android App](https://docs.kukie.cn/guide/android-app) |
| 安全 | Argon2id 密码哈希、通行密钥（WebAuthn）、TOTP 双因素验证与一次性恢复码 | [安全](https://docs.kukie.cn/guide/security) |

## 本次同步的使用说明

- **全局设置 → IP 优选**：开启实验功能，并设置每日请求预算（1–10000000）、每秒请求数（1–1000）和并发上限（1–256）。默认仍为 100000 / 10 / 32，修改从下一轮生效；这是实例预算，不是云平台用量，取消不退还预留量，UTC 次日重置。
- **域名验证**：验证域名与探测 Host / SNI 独立，无需先保存探测配置。验证成功后收起 TXT 信息，但不要删除 DNS 中的记录，任务启动及更新 DNS 前仍会复查。旧定时任务在未完成验证时保持停用。
- **AI 助手**：slim 只显示管理员的连接配置，不提供多用户共享开关。已有共享连接数据仍兼容读取和编辑；模型生成任务不会自动执行，未知结果需先核对资源再重试。
- **批量 DNS**：支持 A / AAAA / TXT / MX / NS / PTR；成功行不重复提交。切换账户或离开页面会停止尚未发送的操作，已发出但响应不明的写入需人工核对。
- 升级前备份数据库及 `APP_ENCRYPTION_KEY`。服务端与 App 版本已对齐为 `3.1.0-slim`（Android 构建号 310），发布产物使用 `v3.1.0-slim` 标签；文档站继续由主分支维护。

## 3.1.0 更新

- Web 与 Android 统一黑白主题，使用应用内语言选择器；底部弹层依场景固定为 30%、60% 或 80%。
- 控制台按页面加载，取消过期请求，独立数据并行读取。公开状态页直接加载，不再先闪现控制台底栏与背景。
- 状态图按真实近 24 小时分成 48 个半小时桶，近期检查条数及对应时间跨度单独显示。
- 同步密码与通行密钥挑战、邮箱修改、图片上传、探测地址及 Cloudflare 凭据隔离的安全修复。`TRUSTED_PROXY_CIDRS` 只应填写你控制的反向代理，确保转发来源地址可信。
- Web、Android 与后端响应支持简体中文和英语。slim 保持既有 Android 包名，兼容升级与已绑定通行密钥。

## 架构

```text
┌──────────────┐
│  Vue 3 前端   │──┐      ┌──────────────┐      ┌──────────────────┐
│  Naive UI    │  ├─────▶│  Go 后端 API  │─────▶│ Cloudflare API   │
├──────────────┤  │      │  chi router  │      │ Tunnels / DNS    │
│ Android App  │──┘      └──────────────┘      └──────────────────┘
│  原生界面     │
└──────────────┘
```

## 快速部署

```bash
tar xzf tunnel-manager.tar.gz
cd tunnel-manager
./install.sh
```

脚本会引导填写 Cloudflare OAuth 客户端或兼容的 API Token，生成 `APP_ENCRYPTION_KEY`，然后构建并启动服务（默认 `8080` 端口）。首次打开面板会进入安装引导：选择 SQLite 或 PostgreSQL、测试连接，再自己设置管理员用户名与密码——这个账户就是整个面板，没有注册入口，密码也不会打印到日志。

**更简单的姿势**：
- 拉取预编译镜像运行：见[「Docker Compose 部署详解」](https://docs.kukie.cn/guide/docker-compose)。精简版镜像标签为 `:slim`（以及 `:<版本>-slim`）。
- 免 Docker 二进制部署：见[「二进制部署」](https://docs.kukie.cn/guide/binary-deploy)
- 手机客户端：从 `android/` 本地构建，见[「Android App」](https://docs.kukie.cn/guide/android-app)
- 已发布版本：[GitHub Releases](https://github.com/qiuyuxc/tunnel-manager/releases)——精简版以 `vX.Y.Z-slim` 标签发布，并标记为 pre-release。

环境变量、OAuth 配置步骤、双因素验证启用与密码重置等详细说明均见[文档站](https://docs.kukie.cn)。

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | Vue 3, TypeScript, Naive UI, Vite, Pinia |
| 后端 | Go, chi, SQLite / PostgreSQL |
| Android App | Java, Gradle, Android SDK（API 26+） |
| 部署 | Docker multi-stage, GitHub Actions CI |

## 开发

```bash
# 后端
cd backend && go run .

# 前端
cd frontend && npm install && npm run dev   # /api 代理到 localhost:8080

# Android App
cd android && ./build.sh

# 验证
(cd backend && go test ./... && go vet ./...)
(cd frontend && npm test && npm run build)
node --test android/tools/*.test.mjs tools/slim-edition.test.mjs
node tools/i18n/generate.mjs --check
```

字体资产：仓库内 `frontend/public/fonts/` 是已切分好的 MiSans 子集，由 `frontend/scripts/subset-fonts.py` 从官方全量字重生成（需要 `fonttools` 与 `brotli`）。全量字重不入库，重新生成时请自行下载。

### 与 `main` 保持同步

`slim/single-user` 是 `main` 的长期精简子集。针对性的共用修复可 cherry-pick；大范围更新请在隔离 worktree 合并，保留 slim 路由和系统设置，移除完整版专属模块与文档站，并在发布前通过 `tools/slim-edition.test.mjs` 及前后端、Android 检查。

## License

[MIT](LICENSE)
