# 架构与技术栈

## 总体架构

```text
┌──────────────┐     ┌──────────────┐     ┌──────────────────┐
│  Vue 3 前端   │────▶│  Go 后端 API  │────▶│ Cloudflare API   │
│  Naive UI    │     │  chi router  │     │ Tunnels / DNS    │
└──────────────┘     └──────────────┘     └──────────────────┘

单容器运行：前端静态文件由 Go 进程直接托管（`STATIC_DIR=frontend/dist`），没有独立的 Node 运行时。

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | Vue 3, TypeScript, Naive UI, Vite, Pinia |
| 后端 | Go, chi, JSON file store |
| 安全 | Argon2id、TOTP、AES-GCM 加密存储 |
| 部署 | Docker multi-stage, Docker Compose |

## 后端目录职责

```text
backend/
├── auth/              # TOTP、AES-GCM 与恢复码
├── handlers/          # HTTP handlers 与认证中间件
├── services/          # Cloudflare、域名绑定、Telegram 与监控业务逻辑
├── models/            # 数据类型定义
├── store/             # JSON 文件存储
└── main.go            # 入口与路由注册
```

关键机制：

- **凭据链**：优先 OAuth（PKCE + 自动刷新），回落静态 Token；OAuth 令牌经 `APP_ENCRYPTION_KEY` AES-GCM 加密落盘
- **监控**：Runner 协程按间隔调度探测，心跳写入独立 `heartbeats.json` 并定时刷盘
- **Telegram Bot**：支持长轮询与 Webhook 双模式，仅响应配置中的管理员 ID
