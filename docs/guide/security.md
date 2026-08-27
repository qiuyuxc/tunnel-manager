# 安全与管理员认证

## 管理员认证

- 密码使用 Argon2id 哈希存储，旧的 SHA-256 哈希会在登录时自动迁移
- 会话有效期 12 小时
- 普通受保护接口支持管理员会话或 `API_KEY`；2FA 管理接口**只接受管理员会话**，不能用 `API_KEY` 绕过

## 双重身份验证（2FA）

1. 使用管理员账户登录面板。
2. 打开 **账户设置**，选择 **设置双重验证**。
3. 用任意兼容 TOTP 的验证器扫描二维码，或手动输入页面中的设置密钥。
4. 输入 6 位动态口令完成绑定。
5. 复制或下载页面一次性显示的 10 枚恢复码，并确认已安全保存。

启用后登录需要密码加动态口令，也可以使用一枚未使用的恢复码。恢复码具备防重放保护。关闭 2FA 需要当前密码和动态口令或恢复码。

TOTP Secret 通过 `APP_ENCRYPTION_KEY` 以 AES-GCM 加密存储。

## 密码重置

忘记密码时通过 CLI 命令重置：

```bash
# 随机生成新密码
docker compose exec tunnel-manager ./tunnel-manager --reset-password

# 设置为指定密码
docker compose exec tunnel-manager ./tunnel-manager --set-password=新密码
```

::: warning
重置密码不会绕过已启用的 2FA。若 TOTP 加密密钥遗失，需要在服务停止时恢复正确的 `APP_ENCRYPTION_KEY`。
:::

## `APP_ENCRYPTION_KEY` 备份清单

该密钥同时保护以下数据，必须与 `data/` 目录一起备份：

- TOTP Secret（2FA 绑定关系）
- Cloudflare OAuth 访问令牌与刷新令牌

密钥丢失或被替换后，已启用 2FA 的服务会拒绝管理员登录，OAuth 连接也需要重新授权。
