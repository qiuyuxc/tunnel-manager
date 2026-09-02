# 架构与技术栈

## 总体架构

```text
┌──────────────┐     ┌──────────────┐     ┌──────────────────┐
│  Vue 3 前端   │────▶│  Go 后端 API  │────▶│ Cloudflare API   │
│  Naive UI    │     │  chi router  │     │ Tunnels / DNS    │
└──────────────┘     └──────────────┘     └──────────────────┘
```

单容器运行：前端静态文件由 Go 进程直接托管（`STATIC_DIR=frontend/dist`），没有独立的 Node 运行时。

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | Vue 3, TypeScript, Naive UI, Vite, Pinia |
| 后端 | Go, chi, SQLite（modernc.org/sqlite 纯 Go 驱动） |
| 安全 | Argon2id、TOTP、AES-GCM 加密存储 |
| 部署 | Docker multi-stage, Docker Compose |

## 后端目录职责

```text
backend/
├── auth/              # TOTP、AES-GCM 与恢复码
├── db/                # SQLite 连接与版本化 schema 迁移
├── handlers/          # HTTP handlers 与认证中间件
├── services/          # Cloudflare、域名绑定、Telegram 与监控业务逻辑
├── models/            # 数据类型定义
├── store/             # SQLite 存储层（含旧版 JSON 自动迁移）
└── main.go            # 入口与路由注册
```

关键机制：

- **凭据链**：优先 OAuth（PKCE + 自动刷新），回落静态 Token；OAuth 令牌经 `APP_ENCRYPTION_KEY` AES-GCM 加密落盘
- **存储**：单 SQLite 库（默认 `data/tunnel-manager.db`，WAL 模式）；首次启动自动导入旧版 `config.json` 并保留原文件作备份；探测心跳仍写入独立 `heartbeats.json` 定时刷盘
- **监控**：Runner 协程按间隔调度探测，心跳写入独立 `heartbeats.json` 并定时刷盘
- **Telegram Bot**：支持长轮询与 Webhook 双模式，仅响应配置中的管理员 ID
