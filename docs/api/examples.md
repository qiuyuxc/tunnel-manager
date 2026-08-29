# 调用示例

所有示例假设服务运行在 `http://localhost:8080`。

## 鉴权方式

| 方式 | 适用接口 | 用法 |
| --- | --- | --- |
| 用户会话 | 全部受保护接口 | 登录后携带 `X-Auth-Token` 请求头 |
| API Key | 普通受保护接口（不含 2FA 管理），等同管理员权限 | 请求头 `X-API-Key`，或 URL 参数 `?api_key=` |

::: warning
未设置环境变量 `API_KEY` 时，API Key 方式整体禁用，仅会话可用。2FA 管理接口从不接受 API Key。
:::

## 注册与登录获取会话

```bash
# ① 查询注册策略（决定注册表单要填什么）
curl http://localhost:8080/api/auth/config

# ② 注册（开放注册时可用；邀请码 / 邮箱验证码按策略提交）
curl -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username": "demo", "email": "demo@example.com", "password": "secret123"}'

# ③ 或使用已有账号登录（account 支持邮箱或用户名）
curl -X POST http://localhost:8080/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"account": "admin", "password": "你的密码"}'

# 返回 JSON 中的 token 即会话令牌，后续请求加请求头：
#   X-Auth-Token: <token>
# 开启 2FA 的账号会先返回 challenge_token，再调用 /api/admin/login/2fa 换取会话
```

## API Key 直调只读接口

```bash
# 健康检查（无需鉴权）
curl http://localhost:8080/api/health

# 读取当前配置
curl -H 'X-API-Key: 你的KEY' http://localhost:8080/api/config

# 参数形式的等价写法
curl 'http://localhost:8080/api/config?api_key=你的KEY'

# 列出隧道
curl -H 'X-API-Key: 你的KEY' http://localhost:8080/api/tunnels
```

## 触发一次监控检测

```bash
curl -X POST -H 'X-API-Key: 你的KEY' \
  http://localhost:8080/api/monitors/<monitorID>/check
```

## 查询告警记录

```bash
curl -H 'X-Auth-Token: <token>' \
  http://localhost:8080/api/monitors/<monitorID>/alerts
```

## 配置状态页优选域名

```bash
curl -X PUT http://localhost:8080/api/monitors/<monitorID> \
  -H 'X-Auth-Token: <token>' \
  -H 'Content-Type: application/json' \
  -d '{
    "public_domain": "status.example.com",
    "public_domain_mode": "preferred",
    "public_aux_domain": "status-origin.example.com",
    "public_preferred_cname": "preferred.example.net"
  }'
```

`public_preferred_cname` 留空时使用全局默认值。响应包含 `domain_warning` 时，表示字段已经保存，但 Cloudflare 自动配置没有完整执行。

## 获取公开状态页数据

无需任何鉴权，token 可为系统令牌或自定义短路径：

```bash
curl http://localhost:8080/api/public/status/<token>
```

::: tip
写操作的请求体字段与面板前端提交的一致；不确定时可先 `GET` 对应资源查看现有结构。完整接口清单见 [接口列表](/api/endpoints)。
:::

