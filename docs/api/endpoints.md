# API 参考

所有接口均以 `/api` 为前缀。普通受保护接口支持**管理员会话**或 `API_KEY`（Authorization 头）两种鉴权方式；2FA 管理接口只接受管理员会话，不能用 `API_KEY` 绕过。

## 健康检查

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/api/health` | 健康检查 | 无 |

## 管理员认证与会话

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| POST | `/api/admin/login` | 密码登录或返回 2FA challenge | 无 |
| POST | `/api/admin/login/2fa` | 使用 TOTP / 恢复码完成登录 | Challenge |
| POST | `/api/admin/logout` | 退出当前会话 | 无 |
| GET | `/api/admin/status` | 检查管理员会话 | 管理员会话 |
| PUT | `/api/admin/password` | 修改管理员密码 | 需要 |
| PUT | `/api/admin/username` | 修改管理员用户名 | 需要 |
| GET | `/api/admin/2fa/status` | 获取 2FA 状态 | 管理员会话 |
| POST | `/api/admin/2fa/setup` | 开始绑定验证器 | 管理员会话 |
| POST | `/api/admin/2fa/confirm` | 确认启用并生成恢复码 | 管理员会话 |
| POST | `/api/admin/2fa/disable` | 关闭 2FA | 管理员会话 |

## Cloudflare OAuth

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/api/cloudflare/oauth/status` | 获取 OAuth、凭据来源与账户状态 | 管理员会话 |
| POST | `/api/cloudflare/oauth/start` | 创建 OAuth 授权请求 | 管理员会话 |
| GET | `/api/cloudflare/oauth/callback` | Cloudflare OAuth 回调 | OAuth State |
| PUT | `/api/cloudflare/oauth/account` | 切换已授权账户 | 管理员会话 |
| DELETE | `/api/cloudflare/oauth` | 撤销并清除 OAuth 凭据 | 管理员会话 |

## 配置与站点品牌

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/api/config` | 获取配置 | 需要 |
| GET | `/api/site` | 获取公开站点品牌信息 | 无 |
| POST | `/api/config/tunnel` | 设置隧道 ID 与显示名称 | 需要 |
| POST | `/api/config/service` | 设置转发地址 | 需要 |
| POST | `/api/config/preferred-cname` | 设置优选 CNAME | 需要 |
| PUT | `/api/config/site` | 更新站点品牌信息 | 需要 |
| PUT | `/api/config/cname-presets` | 更新常用 CNAME 组 | 需要 |

## 隧道管理

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/api/tunnels` | 列出隧道 | 需要 |
| POST | `/api/tunnels` | 新建隧道并返回连接令牌 | 需要 |
| GET | `/api/tunnels/{tunnelID}` | 获取隧道详情与路由 | 需要 |
| DELETE | `/api/tunnels/{tunnelID}` | 删除隧道 | 需要 |
| POST | `/api/tunnels/{tunnelID}/ingress` | 新增应用程序路由 | 需要 |
| PUT | `/api/tunnels/{tunnelID}/ingress` | 更新应用程序路由 | 需要 |
| DELETE | `/api/tunnels/{tunnelID}/ingress` | 删除路由，可选连带删除 DNS 记录 | 需要 |

## DNS 记录

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/api/zones` | 列出 Zone | 需要 |
| GET | `/api/zones/{zoneID}/dns-records` | 查询 Zone 的 DNS 记录 | 需要 |
| POST | `/api/zones/{zoneID}/dns-records` | 新增 DNS 记录 | 需要 |
| PUT | `/api/zones/{zoneID}/dns-records/{recordID}` | 编辑 DNS 记录 | 需要 |
| DELETE | `/api/zones/{zoneID}/dns-records/{recordID}` | 删除 DNS 记录 | 需要 |

## 域名绑定

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| POST | `/api/domain/bind` | 绑定单组域名（支持 `mode` 区分简化/优选模式） | 需要 |
| POST | `/api/domain/bind-batch` | 批量绑定域名，每组独立选择模式与转发地址 | 需要 |
| POST | `/api/domain/fallback` | 设置回退源 | 需要 |

## 服务监控与状态页

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/api/monitors` | 列出监控与目标状态 | 需要 |
| POST | `/api/monitors` | 新建监控 | 需要 |
| PUT | `/api/monitors/{monitorID}` | 更新监控配置（公开标题 / 公告 / 主题 / 短路径等） | 需要 |
| DELETE | `/api/monitors/{monitorID}` | 删除监控 | 需要 |
| POST | `/api/monitors/{monitorID}/check` | 立即执行一次检测 | 需要 |
| POST | `/api/monitors/{monitorID}/targets` | 添加探测目标 | 需要 |
| PUT | `/api/monitors/{monitorID}/targets/{targetID}` | 编辑探测目标 | 需要 |
| DELETE | `/api/monitors/{monitorID}/targets/{targetID}` | 删除探测目标 | 需要 |
| GET | `/api/public/status/{token}` | 公开状态数据；token 可为系统令牌或短路径 | 无 |

## 上传与 Telegram

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| POST | `/api/uploads` | 上传公开状态页图片 | 需要 |
| GET | `/api/telegram/settings` | 获取 Bot 设置 | 需要 |
| PUT | `/api/telegram/settings` | 保存 Bot 设置 | 需要 |
| GET | `/api/telegram/status` | 获取 Bot 状态 | 需要 |
| POST | `/api/telegram/test` | 发送测试消息 | 需要 |
| POST | `/api/telegram/webhook` | Webhook 入口 | Secret Token |
