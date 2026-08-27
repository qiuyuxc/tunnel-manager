# 常见问题 FAQ

## 部署与登录

### 初始密码在哪里？

首次启动自动生成，从容器日志获取：

```bash
docker compose logs | grep 密
```

也可以在首次启动前用环境变量 `ADMIN_PASSWORD` 直接指定。

### 忘记密码了怎么办？

在宿主机执行 CLI 重置：

```bash
docker compose exec tunnel-manager ./tunnel-manager --reset-password        # 生成随机密码
docker compose exec tunnel-manager ./tunnel-manager --set-password=新密码   # 指定密码
```

重置密码**不会绕过 2FA**：已启用双重验证的账户重置后仍需动态口令登录。

### 开启 2FA 时手机丢了 / 密钥没了？

优先用登录页的一次性恢复码（共 10 枚，绑定完成时展示）。

如果连加密密钥 `APP_ENCRYPTION_KEY` 也遗失了，现实中的出路只有两条：**找回原密钥的备份**，或接受用全新密钥重新初始化（清空数据目录后按[快速开始](/guide/getting-started)重来一次，并重新配置 2FA 与 Cloudflare 连接）。

这是有意的安全取舍：程序拒绝在「拿不到正确密钥」的情况下解绑 2FA，正是为了避免任何拿到数据文件的人都能静默关掉你的双因素验证。代价就是密钥彻底遗失时没有自助找回通道。

## Cloudflare 连接

### OAuth 授权回调失败 / redirect_uri mismatch？

Cloudflare 要求回调地址**精确匹配**登记值。检查 `.env` 里的 `CF_OAUTH_REDIRECT_URI` 是否与 OAuth clients 配置完全一致（含协议、端口、路径 `/api/cloudflare/oauth/callback`）；反向代理场景需传递 `X-Forwarded-Proto` 与 `X-Forwarded-Host`。详见 [OAuth 连接](/guide/cloudflare-oauth)。

### 不想配 OAuth，能直接用吗？

可以。提供 `CF_API_TOKEN` 与 `CF_ACCOUNT_ID` 即可运行全部功能，区别只是静态 Token 不具备自动刷新能力，且权限以 Token 本身为准。

## 绑定与 DNS

### 简化直连创建的 `<tunnel-id>.cfargotunnel.com` CNAME 是什么？

这是 Cloudflare 官方约定的隧道接入记录：代理开启的 CNAME 指向隧道 ID 即可让流量进入你的 cloudflared。它是简化模式自动创建的标准产物，不需要手动干预。

### 删除路由时提示可连带删除 DNS？

绑定会同时产生 Ingress 路由和 DNS 记录两份配置。只删路由会让 DNS 变成悬空记录，因此面板提供联动清理选项（v1.14 起），按需勾选即可。

### 旧脚本调用绑定接口没传 mode，会怎样？

按**优选模式**处理，与历史行为保持兼容。新集成建议显式传 `mode` 字段以获得确定行为。

## 其他

### Telegram Bot 收不到响应？

确认机器人 Token 正确、对话者 ID 在管理员的 Telegram ID 白名单内，并检查面板「Bot 状态」。群组中使用时命令支持 `@botname` 后缀消除歧义。

### API 调用该用会话还是 API Key？

自动化脚本推荐 `X-API-Key` 请求头（或在 URL 加 `?api_key=`）。注意两点：未设置 `API_KEY` 环境变量时该方式整体禁用；2FA 相关接口从不接受 API Key，必须走管理员会话。
