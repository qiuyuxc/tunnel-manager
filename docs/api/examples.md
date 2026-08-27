# 调用示例

所有示例假设服务运行在 `http://localhost:8080`。

## 两种鉴权方式

| 方式 | 适用接口 | 用法 |
| --- | --- | --- |
| 管理员会话 | 全部受保护接口 | 登录后携带会话 Cookie |
| API Key | 普通受保护接口（不含 2FA 管理） | 请求头 `X-API-Key`，或 URL 参数 `?api_key=` |

::: warning
未设置环境变量 `API_KEY` 时，API Key 方式整体禁用，仅会话可用。2FA 相关接口从不接受 API Key。
:::

## 登录获取会话

```bash
# 第一步：密码登录
curl -c cookies.txt -X POST http://localhost:8080/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"username": "admin", "password": "你的密码"}'

# 未启用 2FA 时：返回成功并写入会话 Cookie，后续请求加 -b cookies.txt 即可
# 已启用 2FA 时：返回 challenge_token，继续第二步：
curl -c cookies.txt -X POST http://localhost:8080/api/admin/login/2fa \
  -H 'Content-Type: application/json' \
  -d '{"challenge_token": "上一步返回的值", "code": "123456"}'
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

## 获取公开状态页数据

无需任何鉴权，token 可为系统令牌或自定义短路径：

```bash
curl http://localhost:8080/api/public/status/<token>
```

::: tip
写操作的请求体字段与面板前端提交的一致；不确定时可先 `GET` 对应资源查看现有结构。完整接口清单见 [接口列表](/api/endpoints)。
:::
