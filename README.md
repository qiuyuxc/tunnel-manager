<div align="center"><img src="frontend/public/icon.webp" width="72" alt="Tunnel Manager" /></div>

# Tunnel Manager

Cloudflare Tunnel 可视化管理面板。通过 Web UI 管理隧道、绑定域名、配置 DNS 优选与回退源，提供服务可用性监控与可分享的公开状态页，支持多用户注册与管理后台、状态变化邮件告警，并支持 Telegram Bot 远程管理和管理员双重身份验证。

> 📖 **在线文档**：**[https://docs.kukie.cn](https://docs.kukie.cn)**

> 部署指南、Cloudflare OAuth 配置、域名绑定双模式、DNS 管理、服务监控与公开状态页、Telegram Bot 命令全集与 API 参考均在文档站内，本页仅保留快速上手所需的最少信息。

## 功能一览

| 模块 | 能力 | 文档 |
| --- | --- | --- |
| 隧道管理 | 新建 / 删除隧道，Ingress 路由增删改，删除可联动清理 DNS | [隧道管理与路由](https://docs.kukie.cn/guide/tunnels) |
| 域名绑定 | 简化直连与 SaaS 优选双模式，批量绑定逐组独立配置 | [域名绑定模式](https://docs.kukie.cn/guide/domain-binding) · [批量绑定](https://docs.kukie.cn/guide/batch-binding) |
| DNS 管理 | A / AAAA / CNAME / TXT / MX 增删改查，TTL、代理状态与优先级，支持批量操作 | [DNS 记录管理](https://docs.kukie.cn/guide/dns-management) |
| 服务监控 | HTTP / TCP / ICMP 探测，多目标挂载，24 小时延迟柱图 | [服务监控](https://docs.kukie.cn/guide/monitors-status) |
| 邮件告警 | 服务状态变化时自动发送邮件通知，SMTP 可视化配置 | [邮件服务与告警](https://docs.kukie.cn/guide/email-alerts) |
| 多用户 | 邮箱注册、用户组权限、管理后台统一管理用户与邀请码 | [多用户与管理后台](https://docs.kukie.cn/guide/multi-user) |
| 公开状态页 | 免登录分享，支持短路径、自定义域名、直连 Tunnel 与优选 CNAME，自定义域名仅开放对应状态页 | [公开状态页](https://docs.kukie.cn/guide/monitors-status#公开状态页) |
| Cloudflare 连接 | OAuth 2.0（PKCE）自动刷新令牌，多账户切换；兼容静态 Token | [OAuth 连接](https://docs.kukie.cn/guide/cloudflare-oauth) |
| Telegram Bot | 远程管理隧道、绑定与 DNS 记录，删除二次确认 | [Telegram Bot](https://docs.kukie.cn/guide/telegram-bot) |
| 安全 | Argon2id 密码哈希、TOTP 双因素验证与一次性恢复码 | [安全与管理员认证](https://docs.kukie.cn/guide/security) |

## 架构

```text
┌──────────────┐     ┌──────────────┐     ┌──────────────────┐
│  Vue 3 前端   │────▶│  Go 后端 API  │────▶│ Cloudflare API   │
│  Naive UI    │     │  chi router  │     │ Tunnels / DNS    │
└──────────────┘     └──────────────┘     └──────────────────┘
```

## 快速部署

```bash
tar xzf tunnel-manager.tar.gz
cd tunnel-manager
./install.sh
```

脚本会引导填写 Cloudflare OAuth 客户端或兼容的 API Token，生成管理员配置和 `APP_ENCRYPTION_KEY`，然后构建并启动服务（默认 `8080` 端口）。首次启动后获取初始密码：

```bash
docker compose logs | grep 密
```

**更简单的姿势**：
- 拉取预编译镜像运行：见[「Docker Compose 部署详解」](https://docs.kukie.cn/guide/docker-compose)
- 免 Docker 二进制部署：见[「二进制部署」](https://docs.kukie.cn/guide/binary-deploy)
- 已发布版本：[GitHub Releases](https://github.com/qiuyuxc/tunnel-manager/releases)

环境变量、OAuth 配置步骤、双因素验证启用与密码重置等详细说明均见[文档站](https://docs.kukie.cn)。

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | Vue 3, TypeScript, Naive UI, Vite, Pinia |
| 后端 | Go, chi, SQLite |
| 部署 | Docker multi-stage, GitHub Actions CI |

## 开发

```bash
# 后端
cd backend && go run .

# 前端
cd frontend && npm install && npm run dev   # /api 代理到 localhost:8080

# 验证
cd backend && go test ./... && go vet ./...
cd frontend && npm run build
```

## License

[MIT](LICENSE)
