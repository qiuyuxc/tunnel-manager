# Cloudflare OAuth 连接

Tunnel Manager 支持 Cloudflare OAuth 2.0 授权码流程（附加 S256 PKCE）：管理员授权后自动获取并刷新访问令牌，可选择授权账户，无需手动复制 API Token。

## 配置步骤

1. 在 Cloudflare 控制台进入 **Manage Account → OAuth clients** 并创建客户端。
2. 客户端类型选择服务端 Web 应用，启用 `authorization_code` 与 `refresh_token`，Token Authentication Method 选择 `client_secret_basic`，Response Type 选择 `code`。
3. 登记回调地址：`https://你的面板域名/api/cloudflare/oauth/callback`。生产环境建议通过 HTTPS 访问并显式设置 `CF_OAUTH_REDIRECT_URI`。
4. 为客户端选择本项目需要的权限：Account Settings Read、Cloudflare Tunnel Edit、Zone Read、DNS Edit、SSL and Certificates Edit。Cloudflare 会根据 Grant Type 自动处理 `offline_access`。
5. 将 Client ID、Client Secret 和回调地址写入 `.env`，重启服务后在面板 **全局设置 → Cloudflare 连接** 点击「连接 Cloudflare」。
6. 授权页可以选择允许本应用访问的账户；授权完成后面板会自动读取账户列表，多账户时可以随时切换。

## 凭据存储

- 访问令牌和刷新令牌通过 `APP_ENCRYPTION_KEY` 使用 AES-GCM 加密后写入配置文件。
- Client Secret 仅保存在服务端环境变量中，不会发送到浏览器。
- 未配置 OAuth 时仍可使用 `CF_API_TOKEN` 与 `CF_ACCOUNT_ID` 静态凭据，但不会自动刷新。

## 回调地址精确匹配

Cloudflare OAuth 客户端的回调地址要求**精确匹配**。通过反向代理部署时应传递 `X-Forwarded-Proto` 和 `X-Forwarded-Host`，或直接在 `.env` 中配置 `CF_OAUTH_REDIRECT_URI`。

如需撤销授权，可在全局设置中删除 OAuth 连接，服务将回退到静态 Token 模式。

## 多账户授权（v2.0.0+）

一个面板可授权多个 Cloudflare 账户：账户页每次「新增账户授权」都会生成一条独立连接，历史连接不会失效，可在连接卡片间随时切换当前使用的账户。

## 后台配置客户端（免环境变量）

无需环境变量：管理后台 → 账号与安全 → Cloudflare 授权，可直接填写 Client ID / Client Secret / 回调地址 / Scopes，保存后优先于环境变量（Secret 留空表示保持不变；全部字段留空保存即重置）。

## 客户端所需权限

在 Cloudflare 控制台创建 / 编辑 OAuth 客户端时，必须勾选以下权限，否则授权会得到零权限令牌，无法调用任何 API：

- Account Settings : Read
- Cloudflare Tunnel : Edit
- Zone : Read
- DNS : Edit
- SSL and Certificates : Edit
