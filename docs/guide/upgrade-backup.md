# 升级、备份与恢复

## 升级版本

Tunnel Manager 以 Docker 镜像交付，升级即重建镜像并重启容器：

```bash
# 拉取新代码或新包后，在项目目录执行
DOCKER_BUILDKIT=1 docker compose build --no-cache
docker compose up -d
```

与 `install.sh` 首次部署一致，构建分前端、后端两阶段。升级不会触碰宿主机 `./data` 目录。

## 数据目录里有什么

| 文件 / 目录 | 内容 |
| --- | --- |
| `config.json` | 管理员凭据（Argon2id）、隧道选择、绑定记录、监控配置、OAuth 加密令牌、Telegram 设置等全部状态 |
| `heartbeats.json` | 服务监控的历史心跳数据（近 24 小时延迟图表的来源） |
| `uploads/` | 公开状态页图片与站点图标上传文件 |

容器内的路径为 `/app/data`，由 `docker-compose.yml` 从宿主机 `./data` 挂载。

## 备份

::: danger 必须一起备份
`data/` 目录 **和** 环境变量中的 `APP_ENCRYPTION_KEY` 是一对：只备数据不备密钥，恢复后将无法解密 TOTP Secret 与 OAuth Token；只备密钥不备数据则无从恢复。
:::

最小备份集：

```bash
tar czf tunnel-manager-backup.tar.gz data/ .env
```

`APP_ENCRYPTION_KEY` 保存在 `.env` 中（32 字节 Base64）。请把备份文件存到运行机之外的位置。

## 恢复到新机器

1. 安装 Docker 后按[快速开始](/guide/getting-started)放置 compose 与 `.env`（使用原密钥）
2. 解包旧数据：`tar xzf tunnel-manager-backup.tar.gz`
3. `docker compose up -d` 启动
4. 用原管理员凭据登录；如启用了 2FA，验证器照常工作

若密钥遗失，已启用 2FA 的服务会拒绝登录，需在服务停止时还原正确密钥；OAuth 连接则需要重新授权一次。
