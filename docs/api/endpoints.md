# API 参考

所有接口均以 `/api` 为前缀。鉴权方式：

- **用户会话**：登录后获得 `X-Auth-Token`，所有受保护接口接受
- **API Key**：`X-API-Key` 头（或 `?api_key=`），等同管理员权限的机器调用

普通业务接口按用户组权限校验（隧道管理 / 域名绑定 / DNS 记录 / 服务监控 / Cloudflare 授权），管理员不受限制；`/api/admin/*` 下的管理后台接口仅管理员可用。会话保存在数据库中，重启服务不会失效。

## 健康检查

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/api/health` | 健康检查，返回当前版本号 | 无 |

## 注册与身份

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/api/auth/config` | 注册策略：是否开放注册、邀请码模式、是否需要邮箱验证码，以及人机验证（Turnstile）开关与 Site Key | 无 |
| POST | `/api/auth/register` | 注册并自动登录；邀请码与验证码按服务端策略校验 | 无 |
| POST | `/api/auth/send-code` | 发送注册邮箱验证码（需已配置 SMTP，60 秒冷却） | 无 |
| POST | `/api/auth/forgot-password` | 发送密码重置验证码（需已配置 SMTP） | 无 |
| POST | `/api/auth/reset-password` | 使用重置验证码设置新密码，成功后踢掉所有会话 | 无 |
| GET | `/api/auth/me` | 当前登录身份：ID、用户名、昵称、头像、邮箱、角色与权限列表 | 用户会话 |

## 登录与会话

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| POST | `/api/admin/login` | 登录（`account` 支持邮箱或用户名），开启 2FA 时返回 challenge | 无 |
| POST | `/api/admin/login/2fa` | 使用 TOTP / 恢复码完成登录 | Challenge |
| POST | `/api/admin/logout` | 退出当前会话 | 用户会话 |
| GET | `/api/admin/status` | 检查会话有效性，返回用户名与角色 | 用户会话 |
| PUT | `/api/admin/password` | 修改当前用户密码（成功后撤销该用户全部会话） | 用户会话 |
| PUT | `/api/admin/username` | 修改当前用户用户名（需密码确认） | 用户会话 |
| PUT | `/api/admin/email` | 绑定或修改当前用户邮箱（需密码确认） | 用户会话 |
| PUT | `/api/admin/profile` | 更新当前用户自定义名称与头像地址 | 用户会话 |
| POST | `/api/account/avatar` | 上传头像图片（multipart `file`），自动保存到当前账户 | 用户会话 |
| GET | `/api/notify/settings` | 当前用户的通知设置（渠道、事件、邮箱、Telegram 配置状态） | 用户会话 |
| PUT | `/api/notify/settings` | 保存通知设置；`tg_bot_token` 留空表示保持不变，响应不返回 Token | 用户会话 |
| POST | `/api/notify/test` | 按当前配置发送测试通知 | 用户会话 |
| GET | `/api/admin/2fa/status` | 获取当前用户 2FA 状态 | 用户会话 |
| POST | `/api/admin/2fa/setup` | 开始绑定验证器 | 用户会话 |
| POST | `/api/admin/2fa/confirm` | 确认启用并生成恢复码 | 用户会话 |
| POST | `/api/admin/2fa/disable` | 关闭 2FA（需密码 + 动态码 / 恢复码） | 用户会话 |

## 管理后台（仅管理员）

### 用户管理

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/admin/users` | 列出全部用户（含用户组与最近登录） |
| POST | `/api/admin/users` | 创建用户（可指定角色与用户组） |
| PUT | `/api/admin/users/{id}/status` | 启用 / 禁用（禁用同时踢下线） |
| PUT | `/api/admin/users/{id}/group` | 调整用户组 |
| PUT | `/api/admin/users/{id}/password` | 重置密码（该用户被踢下线） |
| DELETE | `/api/admin/users/{id}` | 删除用户（不能删除自己或最后一名管理员） |

### 用户组

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/admin/groups` | 列出用户组与权限集 |
| POST | `/api/admin/groups` | 创建用户组 |
| PUT | `/api/admin/groups/{id}` | 更新名称与权限（内置组仅可改权限） |
| DELETE | `/api/admin/groups/{id}` | 删除用户组（内置组、有成员的组不可删） |

### 邀请码

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/admin/invites` | 列出邀请码 |
| POST | `/api/admin/invites` | 生成邀请码（绑定用户组、次数、有效期） |
| PUT | `/api/admin/invites/{code}` | 启用 / 停用 |
| DELETE | `/api/admin/invites/{code}` | 删除邀请码 |

### 系统设置

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET / PUT | `/api/admin/settings` | 注册开关、邀请码模式、默认用户组、邮箱验证开关、人机验证（Turnstile Site Key / Secret） |
| GET / PUT | `/api/admin/oauth` | Cloudflare OAuth 客户端（Client ID / Secret / 回调 / Scopes），优先于环境变量 |
| GET / PUT | `/api/admin/encryption-key` | 应用加密密钥（环境变量优先；更换后需重启） |
| GET / PUT | `/api/admin/smtp` | SMTP 邮件服务（加密 / 不加密两种模式） |
| POST | `/api/admin/smtp/test` | 发送测试邮件 |

## Cloudflare OAuth

每个用户可授权多个 Cloudflare 账户，连接之间随时切换。

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/api/cloudflare/oauth/status` | 连接列表、当前激活连接与可用账户 | 用户会话 |
| POST | `/api/cloudflare/oauth/start` | 创建 OAuth 授权请求（新增账户不会覆盖旧连接） | 需要 `oauth_connect` |
| GET | `/api/cloudflare/oauth/callback` | Cloudflare OAuth 回调，生成新连接并自动激活 | OAuth State |
| PUT | `/api/cloudflare/oauth/connection` | 切换当前使用的连接 | 需要 `oauth_connect` |
| PUT | `/api/cloudflare/oauth/account` | 切换当前连接下的 Cloudflare 账户 | 需要 `oauth_connect` |
| DELETE | `/api/cloudflare/oauth` | 撤销并删除连接（`?connection_id=` 指定，缺省为当前连接） | 需要 `oauth_connect` |

## 配置与站点品牌

隧道选择与转发地址为**每用户独立**；站点品牌、优选 CNAME 为全局配置（管理员）。

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/api/config` | 获取配置（全局品牌 + 当前用户选择） | 用户会话 |
| GET | `/api/site` | 获取公开站点品牌信息 | 无 |
| POST | `/api/config/tunnel` | 设置当前用户的隧道 | 用户会话 |
| POST | `/api/config/service` | 设置当前用户的转发地址 | 用户会话 |
| POST | `/api/config/preferred-cname` | 设置全局优选 CNAME | 管理员 |
| PUT | `/api/config/site` | 更新站点品牌信息 | 管理员 |
| PUT | `/api/config/cname-presets` | 更新常用 CNAME 组 | 管理员 |

## 隧道管理

需要用户组权限 `tunnels`；调用当前用户激活连接对应的 Cloudflare 账户。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/tunnels` | 列出隧道 |
| POST | `/api/tunnels` | 新建隧道并返回连接令牌 |
| GET | `/api/tunnels/{tunnelID}` | 获取隧道详情与路由 |
| DELETE | `/api/tunnels/{tunnelID}` | 删除隧道 |
| POST | `/api/tunnels/{tunnelID}/ingress` | 新增应用程序路由 |
| PUT | `/api/tunnels/{tunnelID}/ingress` | 更新应用程序路由 |
| DELETE | `/api/tunnels/{tunnelID}/ingress` | 删除路由，可选连带删除 DNS 记录 |

## DNS 记录

需要用户组权限 `dns`。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/zones` | 列出 Zone |
| GET | `/api/zones/{zoneID}/dns-records` | 查询 Zone 的 DNS 记录 |
| POST | `/api/zones/{zoneID}/dns-records` | 新增 DNS 记录 |
| PUT | `/api/zones/{zoneID}/dns-records/{recordID}` | 编辑 DNS 记录 |
| DELETE | `/api/zones/{zoneID}/dns-records/{recordID}` | 删除 DNS 记录 |

## 域名绑定

需要用户组权限 `domain_bind`。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/domain/bind` | 绑定单组域名（支持 `mode` 区分简化/优选模式） |
| POST | `/api/domain/bind-batch` | 批量绑定域名，每组独立选择模式与转发地址 |
| POST | `/api/domain/fallback` | 设置回退源 |

## 服务监控与状态页

需要用户组权限 `monitors`。监控项目按创建者隔离：普通用户仅见自己的项目，管理员可见全部。创建 / 更新接口支持 `alert_enabled` 与 `alert_emails`（逗号分隔）字段。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/monitors` | 列出监控与目标状态 |
| POST | `/api/monitors` | 新建监控 |
| GET | `/api/monitors/overview` | 全局概览统计 |
| PUT | `/api/monitors/{monitorID}` | 更新监控配置（公开页 / 告警开关 / 收件邮箱等） |
| DELETE | `/api/monitors/{monitorID}` | 删除监控 |
| POST | `/api/monitors/{monitorID}/check` | 立即执行一次检测 |
| GET | `/api/monitors/{monitorID}/alerts` | 最近 100 条告警记录（是否成功送达 / 失败原因） |
| POST | `/api/monitors/{monitorID}/targets` | 添加探测目标 |
| PUT | `/api/monitors/{monitorID}/targets/{targetID}` | 编辑探测目标 |
| DELETE | `/api/monitors/{monitorID}/targets/{targetID}` | 删除探测目标 |
| GET | `/api/public/status/{token}` | 公开状态数据；token 可为系统令牌或短路径 | 无 |

## 上传与 Telegram

Telegram 远程控制为每用户独立功能：每个账号配置自己的 Bot Token 与授权 TG ID，Bot 只操作该账号自己的资源。管理员历史全局 Bot 配置会在启动时自动迁移为管理员的个人配置。

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| POST | `/api/uploads` | 上传公开状态页图片 | 需要 `monitors` |
| GET | `/api/telegram/settings` | 获取当前用户的 Bot 设置 | 用户会话 |
| PUT | `/api/telegram/settings` | 保存当前用户的 Bot 设置并重启其 Bot | 用户会话 |
| GET | `/api/telegram/status` | 获取当前用户的 Bot 状态 | 用户会话 |
| POST | `/api/telegram/test` | 向当前用户的授权 TG ID 发送测试消息 | 用户会话 |
| PUT | `/api/telegram/endpoint` | 设置面板级 Telegram API 端点（自定义反代），所有用户 Bot 生效 | 管理员 |
| POST | `/api/telegram/webhook` | 旧全局 Bot 的 Webhook 入口（已停用） | Secret Token |

