# API 参考

所有接口均以 `/api` 为前缀。鉴权方式：

- **用户会话**：登录后获得 `X-Auth-Token`，所有受保护接口接受
- **API Key**：`X-API-Key` 头（或 `?api_key=`），等同管理员权限的机器调用

普通业务接口按用户组权限校验（隧道管理 / 域名绑定 / DNS 记录 / 服务监控 / Cloudflare 授权），管理员不受限制；`/api/admin/*` 下的管理后台接口仅管理员可用。会话保存在数据库中，重启服务不会失效。

## 健康检查

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/api/health` | 健康检查，返回当前版本号 | 无 |

## 安装引导

面板还没有管理员账户时，进程只挂载这一组接口与前端页面；安装完成后它们一律返回 `404`。

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/api/setup/status` | 安装状态：存储是否由环境变量接管、当前的数据库配置（凭据已脱敏）、数据目录与配置文件路径 | 无 |
| POST | `/api/setup/test` | 用表单里的数据库配置试连一次，不落盘 | 无 |
| POST | `/api/setup/complete` | 保存存储配置、建表并创建管理员账户，成功后进程自行重启 | 无 |

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

所有认证入口（登录、两步验证、通行密钥登录、注册、邮箱验证码、找回与重置密码）都受登录限流保护：账号与来源 IP 分别计数，额度耗尽时返回 `429`，响应头带 `Retry-After`（秒），响应体带 `retry_after` 与提示文案；阈值与开关见「系统设置 → 登录保护」，机制详见[安全与管理员认证](/guide/security#登录限流)。

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
| GET / PUT | `/api/admin/settings` | 注册开关、邀请码模式、默认用户组、邮箱验证开关、人机验证（Turnstile Site Key / Secret）、实验性功能开关、审计日志保留天数（`audit_retention_days`）、通行密钥依赖方（`passkey_rp_id` / `passkey_origins`）、全局禁用密码登录（`password_login_disabled`）与登录限流（`rate_limit_enabled`、`rate_limit_per_account`、`rate_limit_per_ip`、`rate_limit_window_minutes`、`rate_limit_familiar_multiplier`、`rate_limit_notify`）；省略字段保持原值 |
| GET / PUT | `/api/admin/oauth` | Cloudflare OAuth 客户端（Client ID / Secret / 回调 / Scopes），优先于环境变量 |
| GET / PUT | `/api/admin/encryption-key` | 应用加密密钥（环境变量优先；更换后需重启） |
| GET / PUT | `/api/admin/smtp` | SMTP 邮件服务（加密 / 不加密两种模式） |
| POST | `/api/admin/smtp/test` | 发送测试邮件 |

### 审计日志

记录管理后台与业务变更操作：登录与登出、用户与用户组、邀请码、系统设置、隧道与隧道规则、域名绑定、DNS 记录、监控项目与目标、IP 优选实验室。只读浏览不计入。保留时长在系统设置中配置，超期日志每小时自动清理。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/admin/audit-logs` | 分页查询审计日志；支持 `actor`（用户名模糊匹配，也接受账号 ID）、`category`、`action`、`from` / `to`（Unix 秒）与 `page` / `page_size`（默认 50，上限 200） |
| GET | `/api/admin/audit-logs/stats` | 统计概览：`days`（默认 7，上限 90）窗口内的操作总数 `total`、失败数 `failed`、活跃账号数 `actors` 与今日操作数 `today` |

每条日志包含时间、操作人（账号 ID 与用户名）、分类与操作类型、目标对象、来源 IP 与成功 / 失败结果。

## 通行密钥（WebAuthn）

通行密钥用设备上的指纹、面容或硬件安全密钥替代密码。依赖方 ID 默认按访问域名自动推导（沿用面板域名），反向代理或多域名场景可在「系统设置 → 通行密钥」覆盖；要求 HTTPS（localhost 例外）。依赖方解析失败时账户页会提示，且不会展示绑定入口。

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| POST | `/api/auth/passkey/login/begin` | 开始免密登录；`account` 可选，填了只缩小浏览器提供的凭据范围，留空为无用户名（discoverable）流程 | 无 |
| POST | `/api/auth/passkey/login/finish` | 校验断言并完成登录；通行密钥本身即强因子，不再叠加 TOTP | 无 |
| POST | `/api/admin/login/2fa/passkey/begin` | 密码登录进入 2FA 后，改用通行密钥完成第二步 | Challenge |
| POST | `/api/admin/login/2fa/passkey/finish` | 校验断言并完成 2FA 登录 | Challenge |
| GET | `/api/account/passkeys` | 列出本账户的通行密钥、依赖方信息与密码登录开关状态 | 用户会话 |
| POST | `/api/account/passkeys/begin` | 开始绑定，需提交当前密码 | 用户会话 |
| POST | `/api/account/passkeys/finish` | 完成绑定并保存凭据（`name` 为备注名，留空自动编号） | 用户会话 |
| PUT | `/api/account/passkeys/{id}` | 重命名 | 用户会话 |
| DELETE | `/api/account/passkeys/{id}` | 删除，需当前密码；已禁用密码登录时不允许删除最后一个 | 用户会话 |
| PUT | `/api/account/password-login` | 开关本账户的密码登录，需当前密码；开启（禁用密码登录）前必须已绑定通行密钥 | 用户会话 |
| PUT | `/api/admin/users/{id}/password-login` | 管理员强制开关某账户的密码登录，用于账户丢失通行密钥后恢复 | 管理员 |

所有注册流程都返回 `{ceremony_token, public_key}`，`public_key` 为浏览器 WebAuthn 字典，`ceremony_token` 需在对应的 finish 请求中原样回传（一次性、5 分钟有效）。命令行 `-allow-password-login` 可关闭全局开关并清除所有账户开关，用于面板被锁死时恢复。

原生 App 还需要数字资产链接，后端在站点根路径提供（无需鉴权）：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/.well-known/assetlinks.json` | Android 数字资产链接：包名与 SHA-256 签名指纹取自系统设置（`passkey_android_package` / `passkey_android_fingerprints`，留空时用仓库共用调试密钥的指纹）；没有有效指纹时返回 404 而不是错误内容 |

## IP 优选实验室（实验性）

默认关闭；关闭时以下接口返回 `404`。Secret Key 仅写入加密存储，响应中只返回是否已配置。

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/api/lab/ip-selector` | 获取探测、定时任务与华为云 DNS 配置 | 管理员 |
| PUT | `/api/lab/ip-selector` | 保存配置；`secret_key` 留空表示保持不变 | 管理员 |
| GET | `/api/lab/ip-selector/status` | 获取运行状态、实时进度与执行历史 | 管理员 |
| POST | `/api/lab/ip-selector/run` | 触发一次异步执行；已有任务运行时返回 `409` | 管理员 |

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

需要用户组权限 `monitors`。监控项目按创建者隔离：普通用户仅见自己的项目，管理员可见全部。创建 / 更新接口支持公开页、告警开关与收件邮箱配置。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/monitors` | 列出监控与目标状态 |
| POST | `/api/monitors` | 新建监控 |
| GET | `/api/monitors/overview` | 全局概览统计；每个时间桶带 `issues` 数组（`omitempty`，只列出当天出现过异常的目标），内含目标标识、`warn` / `down` 计数、`incident_count` 与 `incidents` 异常时间段（起止时间、状态、连续次数、状态码与错误信息），供图表下钻 |
| PUT | `/api/monitors/{monitorID}` | 更新监控配置（公开页 / 告警开关 / 收件邮箱等） |
| DELETE | `/api/monitors/{monitorID}` | 删除监控 |
| POST | `/api/monitors/{monitorID}/check` | 立即执行一次检测 |
| GET | `/api/monitors/{monitorID}/alerts` | 最近 100 条告警记录（是否成功送达 / 失败原因） |
| POST | `/api/monitors/{monitorID}/targets` | 添加探测目标 |
| PUT | `/api/monitors/{monitorID}/targets/{targetID}` | 编辑探测目标 |
| DELETE | `/api/monitors/{monitorID}/targets/{targetID}` | 删除探测目标 |
| GET | `/api/public/status/{token}` | 公开状态数据，token 可为系统令牌或短路径 |
| GET | `/api/alerts?since=<unix秒>` | 告警游标流，供手机 App 轮询；`since=0` 只回游标不重放历史 |

`PUT /api/monitors/{monitorID}` 的自定义域名字段：

| 字段 | 说明 |
| --- | --- |
| `public_domain` | 状态页访问域名，留空则只保留面板域名下的公开链接 |
| `public_domain_mode` | `simple` 为直连，`preferred` 为优选 |
| `public_aux_domain` | 优选模式必填，橙云回源到 Tunnel 的辅助域名 |
| `public_preferred_cname` | 本监控使用的优选 CNAME，留空时读取全局默认值 |
| `domain_warning` | 响应字段，表示设置已保存，但 Cloudflare 自动配置失败 |

优选模式下，访问域名、辅助回源域名不能相同，优选 CNAME 也不能与访问域名相同。修改上述任一字段都会重新执行 Cloudflare 自动配置。自动配置失败返回 HTTP 200，并通过 `domain_warning` 说明原因。

自定义域名只允许访问当前监控的公开页和相关静态资源。其他状态页、管理后台和受保护 API 会返回 404。

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
| POST | `/api/telegram/webhook` | 旧全局 Bot 的 Webhook 入口（向后兼容） | Secret Token |
| POST | `/api/telegram/webhook/{userID}` | 每用户 Bot 的 Webhook 入口，仅分发到 URL 中 userID 对应的 Bot | Secret Token |
