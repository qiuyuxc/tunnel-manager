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
- 域名绑定：可选择简化直连或优选模式；简化模式只配置主域名 Tunnel 路由和代理 CNAME，优选模式继续配置辅助域名与 SaaS Custom Hostname
- 批量绑定：一次提交多个绑定组，每组可独立选择绑定模式、转发地址与优选 CNAME
- DNS 管理：独立页面按 Zone 查询、新增、编辑和删除 A、AAAA、CNAME、TXT、MX 记录，支持 TTL、代理状态与 MX 优先级
- 站点品牌：自定义站点名称、描述、导航/登录页图标与浏览器标题
- 优选 CNAME：自定义全局默认值并维护常用 CNAME 组，绑定时可直接选择
- 回退源设置：一键配置 fallback origin
- Cloudflare OAuth 2.0：管理员授权后自动获取并刷新访问令牌，可选择授权账户，无需手动复制 API Token
- Telegram Bot：远程管理隧道、简化/优选域名绑定与 DNS 记录，支持长轮询、Webhook、自定义 API 端点和 DNS 删除二次确认
- 关于页面：展示版本号与 GitHub 仓库地址，自动检查线上最新 Release 并展示更新内容
- 管理员认证：Argon2id 密码哈希、12 小时会话、旧 SHA-256 哈希自动迁移
- 双重身份验证：标准 TOTP、加密密钥存储、一次性恢复码与防重放
- 密码重置：忘记密码时通过 CLI 命令重置

## 域名绑定模式

Web 端的单个与批量域名绑定支持两种模式，批量操作时每组可以独立选择：

- **简化直连**：只填写主域名与转发地址。系统为主域名添加 Tunnel Ingress，并创建代理开启的 CNAME 指向 `<tunnel-id>.cfargotunnel.com`；不需要辅助域名，也不会创建 SaaS Custom Hostname。
- **优选模式**：沿用优选 CNAME、辅助域名与 Cloudflare for SaaS Custom Hostname 的完整流程。优选 CNAME 留空时使用全局默认值。

为兼容旧客户端，未提交 `mode` 的 API 请求仍按优选模式处理。Telegram Bot 当前也继续使用优选模式。

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

脚本会检测 Docker 环境，引导填写 Cloudflare OAuth 客户端或兼容的 API Token，生成管理员配置和 `APP_ENCRYPTION_KEY`，然后构建并启动服务。首次启动后查看日志获取初始密码：

```bash
docker compose logs | grep 密
```

环境变量示例见 `.env.example`：

| 变量 | 必需 | 说明 |
|---|---|---|
| `CF_OAUTH_CLIENT_ID` | OAuth 必需 | Cloudflare OAuth Client ID |
| `CF_OAUTH_CLIENT_SECRET` | OAuth 必需 | Cloudflare OAuth Client Secret，仅保存在服务端环境变量中 |
| `CF_OAUTH_REDIRECT_URI` | 推荐 | OAuth 回调地址，必须与 Cloudflare 客户端登记值完全一致；留空时根据请求地址推导 |
| `CF_OAUTH_SCOPES` | 否 | 显式请求的空格分隔 scope；默认留空并使用 OAuth 客户端配置的 scopes |
| `CF_API_TOKEN` | 兼容 | 旧版静态 Cloudflare API Token；未连接 OAuth 时使用 |
| `CF_ACCOUNT_ID` | 兼容 | 静态 Token 对应的 Account ID；OAuth 会自动读取并保存账户 |
| `API_KEY` | 否 | 自动化调用使用的 API Key |
| `ADMIN_PASSWORD` | 否 | 首次启动的管理员密码，留空时自动生成 |
| `APP_ENCRYPTION_KEY` | 2FA 必需 | Base64 编码的 32 字节随机密钥，用于加密 TOTP secret |
| `STORE_PATH` | 否 | JSON 配置路径，默认 `data/config.json` |
| `PORT` | 否 | HTTP 端口，默认 `8080` |

> `APP_ENCRYPTION_KEY` 必须与数据文件一起备份。它同时保护 TOTP Secret 与 Cloudflare OAuth Token；密钥丢失或被替换后，已启用 2FA 的服务会拒绝管理员登录，OAuth 连接也需要重新授权。

### 配置 Cloudflare OAuth 2.0

1. 在 Cloudflare 控制台进入 **Manage Account → OAuth clients** 并创建客户端。
2. 客户端类型选择服务端 Web 应用，启用 `authorization_code` 与 `refresh_token`，Token Authentication Method 选择 `client_secret_basic`，Response Type 选择 `code`。
3. 登记回调地址：`https://你的面板域名/api/cloudflare/oauth/callback`。生产环境建议通过 HTTPS 访问并显式设置 `CF_OAUTH_REDIRECT_URI`。
4. 为客户端选择本项目需要的权限：Account Settings Read、Cloudflare Tunnel Edit、Zone Read、DNS Edit、SSL and Certificates Edit。Cloudflare 会根据 Grant Type 自动处理 `offline_access`。
5. 将 Client ID、Client Secret 和回调地址写入 `.env`，重启服务后登录面板，在“全局设置 → Cloudflare 连接”点击“连接 Cloudflare”。
6. 授权页可以选择允许本应用访问的账户；授权完成后，面板会自动读取账户列表，多账户时可以随时切换。

OAuth 使用授权码流程并附加 S256 PKCE。访问令牌和刷新令牌通过 `APP_ENCRYPTION_KEY` 使用 AES-GCM 加密后写入配置文件，Client Secret 不会发送到浏览器。若暂时未配置 OAuth，服务仍可使用 `CF_API_TOKEN` 与 `CF_ACCOUNT_ID`。

> Cloudflare OAuth 客户端的回调地址要求精确匹配。通过反向代理部署时，应传递 `X-Forwarded-Proto` 和 `X-Forwarded-Host`，或直接配置 `CF_OAUTH_REDIRECT_URI`。

### 启用双重身份验证

1. 使用管理员账户登录。
2. 打开“账户设置”，选择“设置双重验证”。
3. 用任意兼容 TOTP 的验证器扫描二维码，或手动输入页面中的设置密钥。
4. 输入 6 位动态口令完成绑定。
5. 复制或下载页面一次性显示的 10 枚恢复码，并确认已安全保存。

启用后，登录需要密码加动态口令，也可以使用一枚未使用的恢复码。关闭 2FA 需要当前密码和动态口令或恢复码。

### Telegram Bot 命令

Bot 仅接受配置中的管理员 Telegram ID，支持群组命令中的 `@botname` 后缀。

- `/直连域名 [主域名]`：使用简化模式直接绑定到当前 Tunnel
- `/优选绑定 [主域名] [辅助域名] [可选优选CNAME]`：执行完整优选绑定；旧命令 `/绑定域名` 继续兼容
- `/列出区域`：列出可用 Zone（仅显示域名，无需复制 Zone ID）
- `/DNS列表 [域名或ZoneID] [可选类型] [可选名称]`：简洁列表（每行一条记录，长内容自动截断），区域参数可直接填写域名
- `/DNS详情 [完整主机名]`：如 `/DNS详情 bbs.kukie.cn`，查看单条记录完整信息（含记录 ID、TTL、代理状态、MX 优先级）；也支持 `/DNS详情 [区域] [记录名]` 旧写法
- `/选择区域 [域名或ZoneID]`：选定 DNS 区域（类似选定隧道），后续 DNS 命令可省略区域参数
- `/DNS添加 [区域可选] [类型] [名称] [内容] [TTL可选] [代理可选] [MX优先级]`：管理 A、AAAA、CNAME、TXT、MX；TTL 缺省 auto，代理缺省 on（TXT/MX 自动关闭代理）
- `/DNS修改 [区域可选] [记录名或ID] [类型] [新内容] [TTL可选] [代理可选]`：按记录名直接修改（如 `/DNS修改 bbs CNAME saas.com`），也兼容按 RecordID 的旧格式
- `/DNS删除 [区域可选] [记录名或RecordID]`：生成五分钟有效的一次性确认码，必须再发送 `/确认删除 [确认码]` 才会删除

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
| GET | `/api/cloudflare/oauth/status` | 获取 OAuth、凭据来源与账户状态 | 管理员会话 |
| POST | `/api/cloudflare/oauth/start` | 创建 OAuth 授权请求 | 管理员会话 |
| GET | `/api/cloudflare/oauth/callback` | Cloudflare OAuth 回调 | OAuth State |
| PUT | `/api/cloudflare/oauth/account` | 切换已授权账户 | 管理员会话 |
| DELETE | `/api/cloudflare/oauth` | 撤销并清除 OAuth 凭据 | 管理员会话 |
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
| GET | `/api/zones/{zoneID}/dns-records` | 查询 Zone DNS 记录 | 需要 |
| POST | `/api/zones/{zoneID}/dns-records` | 新增 DNS 记录 | 需要 |
| PUT | `/api/zones/{zoneID}/dns-records/{recordID}` | 编辑 DNS 记录 | 需要 |
| DELETE | `/api/zones/{zoneID}/dns-records/{recordID}` | 删除 DNS 记录 | 需要 |
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
