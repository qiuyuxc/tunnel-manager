# Tunnel Manager

Cloudflare Tunnel 可视化管理面板。通过 Web UI 管理隧道、绑定域名、配置 DNS 优选与回退源，并提供 Telegram Bot 和管理员双重身份验证。

## 架构

```text
┌──────────────┐     ┌──────────────┐     ┌──────────────────┐
│  Vue 3 前端   │────▶│  Go 后端 API  │────▶│ Cloudflare API   │
│  Naive UI    │     │  chi router  │     │ Tunnels / DNS    │
└──────────────┘     └──────────────┘     └──────────────────┘
```

## 功能

- 隧道管理：列出、选择 Cloudflare Tunnel，查看和编辑应用程序路由
- 域名绑定：自动配置 Tunnel 路由、DNS CNAME 与 SaaS Custom Hostname
- 批量绑定：一次提交多个绑定组，每组可独立配置转发地址与优选 CNAME
- 站点品牌：自定义站点名称、描述、导航/登录页图标与浏览器标题
- 优选 CNAME：自定义全局默认值并维护常用 CNAME 组，绑定时可直接选择
- 回退源设置：一键配置 fallback origin
- Telegram Bot：远程管理隧道，支持长轮询、Webhook 与自定义 API 端点
- 管理员认证：Argon2id 密码哈希、12 小时会话、旧 SHA-256 哈希自动迁移
- 双重身份验证：标准 TOTP、加密密钥存储、一次性恢复码与防重放
- 密码重置：忘记密码时通过 CLI 命令重置

## 技术栈

| 层 | 技术 |
|---|---|
| 前端 | Vue 3, TypeScript, Naive UI, Vite, Pinia |
| 后端 | Go, chi, JSON file store |
| 部署 | Docker multi-stage, Docker Compose |

## 部署

```bash
tar xzf tunnel-manager.tar.gz
cd tunnel-manager
./install.sh
```

脚本会检测 Docker 环境，引导填写 Cloudflare 凭据，生成管理员配置和 `APP_ENCRYPTION_KEY`，然后构建并启动服务。首次启动后查看日志获取初始密码：

```bash
docker compose logs | grep 密
```

环境变量示例见 `.env.example`：

| 变量 | 必需 | 说明 |
|---|---|---|
| `CF_API_TOKEN` | 是 | Cloudflare API Token |
| `CF_ACCOUNT_ID` | 是 | Cloudflare Account ID |
| `API_KEY` | 否 | 自动化调用使用的 API Key |
| `ADMIN_PASSWORD` | 否 | 首次启动的管理员密码，留空时自动生成 |
| `APP_ENCRYPTION_KEY` | 2FA 必需 | Base64 编码的 32 字节随机密钥，用于加密 TOTP secret |
| `STORE_PATH` | 否 | JSON 配置路径，默认 `data/config.json` |
| `PORT` | 否 | HTTP 端口，默认 `8080` |

> `APP_ENCRYPTION_KEY` 必须与数据文件一起备份。启用 2FA 后如果密钥丢失或被替换，服务会拒绝管理员登录，不会降级绕过第二重验证。

### 启用双重身份验证

1. 使用管理员账户登录。
2. 打开“账户设置”，选择“设置双重验证”。
3. 用任意兼容 TOTP 的验证器扫描二维码，或手动输入页面中的设置密钥。
4. 输入 6 位动态口令完成绑定。
5. 复制或下载页面一次性显示的 10 枚恢复码，并确认已安全保存。

启用后，登录需要密码加动态口令，也可以使用一枚未使用的恢复码。关闭 2FA 需要当前密码和动态口令或恢复码。

### 重置密码

```bash
# 随机生成新密码
docker compose exec tunnel-manager ./tunnel-manager --reset-password

# 设置为指定密码
docker compose exec tunnel-manager ./tunnel-manager --set-password=新密码
```

重置密码不会绕过已启用的 2FA。若 TOTP 加密密钥遗失，需要在服务停止时恢复正确的 `APP_ENCRYPTION_KEY`。

## API

普通受保护接口支持管理员会话或 `API_KEY`。2FA 管理接口只接受管理员会话，不能使用 `API_KEY` 绕过。

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | `/api/health` | 健康检查 | 无 |
| POST | `/api/admin/login` | 密码登录或返回 2FA challenge | 无 |
| POST | `/api/admin/login/2fa` | 使用 TOTP/恢复码完成登录 | Challenge |
| POST | `/api/admin/logout` | 退出当前会话 | 无 |
| GET | `/api/admin/status` | 检查管理员会话 | 管理员会话 |
| PUT | `/api/admin/password` | 修改管理员密码 | 需要 |
| PUT | `/api/admin/username` | 修改管理员用户名 | 需要 |
| GET | `/api/admin/2fa/status` | 获取 2FA 状态 | 管理员会话 |
| POST | `/api/admin/2fa/setup` | 开始绑定验证器 | 管理员会话 |
| POST | `/api/admin/2fa/confirm` | 确认启用并生成恢复码 | 管理员会话 |
| POST | `/api/admin/2fa/disable` | 关闭 2FA | 管理员会话 |
| GET | `/api/config` | 获取配置 | 需要 |
| GET | `/api/site` | 获取公开站点品牌信息 | 无 |
| POST | `/api/config/tunnel` | 设置隧道 ID 与显示名称 | 需要 |
| POST | `/api/config/service` | 设置转发地址 | 需要 |
| POST | `/api/config/preferred-cname` | 设置优选 CNAME | 需要 |
| PUT | `/api/config/site` | 更新站点品牌信息 | 需要 |
| PUT | `/api/config/cname-presets` | 更新常用 CNAME 组 | 需要 |
| GET | `/api/tunnels` | 列出隧道 | 需要 |
| GET | `/api/tunnels/{tunnelID}` | 获取隧道详情与路由 | 需要 |
| POST | `/api/tunnels/{tunnelID}/ingress` | 新增应用程序路由 | 需要 |
| PUT | `/api/tunnels/{tunnelID}/ingress` | 更新应用程序路由 | 需要 |
| GET | `/api/zones` | 列出 Zone | 需要 |
| POST | `/api/domain/bind` | 绑定单组域名 | 需要 |
| POST | `/api/domain/bind-batch` | 批量绑定域名 | 需要 |
| POST | `/api/domain/fallback` | 设置回退源 | 需要 |
| GET | `/api/telegram/settings` | 获取 Bot 设置 | 需要 |
| PUT | `/api/telegram/settings` | 保存 Bot 设置 | 需要 |
| GET | `/api/telegram/status` | 获取 Bot 状态 | 需要 |
| POST | `/api/telegram/test` | 发送测试消息 | 需要 |
| POST | `/api/telegram/webhook` | Webhook 入口 | Secret Token |

## 开发

```bash
# 后端
cd backend
go run .

# 前端
cd frontend
npm install
npm run dev
```

前端开发服务器会将 `/api` 代理到 `http://localhost:8080`。

验证命令：

```bash
cd backend && go test ./... && go vet ./...
cd frontend && npm run build
```

## 项目结构

```text
├── backend/
│   ├── auth/              # TOTP、AES-GCM 与恢复码
│   ├── handlers/          # HTTP handlers 与认证中间件
│   ├── services/          # Cloudflare、域名与 Telegram 业务逻辑
│   ├── models/            # 数据类型
│   ├── store/             # JSON 文件存储
│   └── main.go            # 入口与路由注册
├── frontend/
│   ├── src/
│   │   ├── views/         # 页面
│   │   ├── components/    # 组件
│   │   ├── stores/        # Pinia stores
│   │   ├── api/           # API 封装
│   │   └── router/        # 路由
│   └── vite.config.ts
├── install.sh
├── Dockerfile
└── docker-compose.yml
```
