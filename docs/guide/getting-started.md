# 快速开始

Tunnel Manager 通过 Docker 单容器部署，Vue 3 前端与 Go 后端打包在同一个镜像内，所有数据持久化在宿主机的 `data/` 目录。

## 一键部署

```bash
tar xzf tunnel-manager.tar.gz
cd tunnel-manager
./install.sh
```

脚本会检测 Docker 环境，引导填写 Cloudflare OAuth 客户端或兼容的 API Token，生成 `APP_ENCRYPTION_KEY`，然后构建并启动服务；管理员账户在首次打开面板时由安装引导创建。服务默认监听 `8080` 端口（`docker-compose.yml` 映射为宿主机 8080）。

> 不想从源码构建？也可以直接拉取预编译镜像运行，或下载二进制免 Docker 部署：见 [Docker Compose 部署详解](/guide/docker-compose) 与 [二进制部署](/guide/binary-deploy)。

## 创建管理员账户

首次打开面板会进入安装引导：面板还没有任何账户时只提供安装页面，先选择数据库（SQLite 单文件或 PostgreSQL，PostgreSQL 可先测试连接），再自己填写管理员用户名与密码即可完成安装，随后服务自动重启进入登录页。密码由你设置、只保存在数据库里，不会打印到日志。

想跳过这一步，就在首次启动前设置 `ADMIN_PASSWORD`：启动时直接用它创建管理员账户，之后再打开面板就是登录页。

## 环境变量

| 变量 | 必需 | 说明 |
| --- | --- | --- |
| `CF_OAUTH_CLIENT_ID` | OAuth 必需 | Cloudflare OAuth Client ID |
| `CF_OAUTH_CLIENT_SECRET` | OAuth 必需 | Client Secret，仅保存在服务端环境变量中 |
| `CF_OAUTH_REDIRECT_URI` | 推荐 | OAuth 回调地址，必须与 Cloudflare 客户端登记值完全一致；留空时根据请求地址推导 |
| `CF_OAUTH_SCOPES` | 否 | 显式请求的空格分隔 scope；留空使用 OAuth 客户端配置的 scopes |
| `CF_API_TOKEN` | 兼容 | 旧版静态 Cloudflare API Token；未连接 OAuth 时使用 |
| `CF_ACCOUNT_ID` | 兼容 | 静态 Token 对应的 Account ID；OAuth 会自动读取并保存账户 |
| `API_KEY` | 否 | 自动化调用使用的 API Key |
| `ADMIN_PASSWORD` | 否 | 首次启动的管理员密码；留空时改为在网页安装引导里设置 |
| `APP_ENCRYPTION_KEY` | 2FA 必需 | Base64 编码的 32 字节随机密钥，用于加密 TOTP Secret 与 OAuth Token |
| `STORE_PATH` | 否 | SQLite 数据库路径，默认 `data/tunnel-manager.db`；Docker 镜像沿用旧名 `data/config.json` 以就地升级 |
| `DATABASE_URL` | 否 | PostgreSQL 连接串，如 `postgres://user:pass@host:5432/tunnel_manager?sslmode=disable`；设置后优先于 `STORE_PATH` |
| `DATA_DIR` | 否 | 上传文件与心跳日志目录；默认取 SQLite 文件所在目录，使用 PostgreSQL 时建议显式指定 |
| `PORT` | 否 | HTTP 端口，默认 `8080` |

::: warning 加密密钥必须备份
`APP_ENCRYPTION_KEY` 必须与数据文件一起备份。它同时保护 TOTP Secret 与 Cloudflare OAuth Token：密钥丢失或被替换后，已启用 2FA 的服务会拒绝管理员登录，OAuth 连接也需要重新授权。详见[安全与管理员认证](/guide/security)。
:::

## 数据持久化与升级

`docker-compose.yml` 将宿主机 `./data` 目录挂载到容器内 `/app/data`，隧道列表、绑定记录、监控历史与管理员凭据都保存在这里。升级版本时保留该目录即可平滑迁移。

OAuth 连接后推荐反向代理传递 `X-Forwarded-Proto` 与 `X-Forwarded-Host` 头，或显式设置 `CF_OAUTH_REDIRECT_URI`，避免回调地址推导错误。

## 本地开发

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
