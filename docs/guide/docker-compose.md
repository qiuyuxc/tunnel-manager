# Docker Compose 部署详解

两种启动方式任选其一：**方式一拉取官方镜像**（推荐普通用户，无需 Node/Go 环境），**方式二本地源码构建**（适合开发者或需要魔改的场景）。两者的 `environment` 与 `volumes` 完全一致。

## 完整 compose 文件

### 方式一：拉取预编译镜像（推荐）

```yaml
services:
  tunnel-manager:
    image: ghcr.io/qiuyuxc/tunnel-manager:latest
    container_name: tunnel-manager
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - CF_API_TOKEN=${CF_API_TOKEN}
      - CF_ACCOUNT_ID=${CF_ACCOUNT_ID}
      - CF_OAUTH_CLIENT_ID=${CF_OAUTH_CLIENT_ID}
      - CF_OAUTH_CLIENT_SECRET=${CF_OAUTH_CLIENT_SECRET}
      - CF_OAUTH_REDIRECT_URI=${CF_OAUTH_REDIRECT_URI}
      - CF_OAUTH_SCOPES=${CF_OAUTH_SCOPES}
      - API_KEY=${API_KEY}
      - ADMIN_PASSWORD=${ADMIN_PASSWORD}
      - APP_ENCRYPTION_KEY=${APP_ENCRYPTION_KEY}
    volumes:
      - ./data:/app/data
```

只需一个 `docker-compose.yml` 和 `.env` 文件即可运行，不需要克隆整个仓库。

::: tip 国内镜像加速
ghcr.io 拉取慢时，把镜像前缀换成高校镜像站即可，例如：

```yaml
    image: ghcr.nju.edu.cn/qiuyuxc/tunnel-manager:latest
```

原理是镜像站代理了 ghcr.io 的仓库内容，标签与摘要不变。
:::

### 方式二：源码构建

在克隆好的仓库根目录使用构建型配置：

```yaml
services:
  tunnel-manager:
    build:
      context: .
      args:
        NPM_REGISTRY: ${NPM_REGISTRY:-https://registry.npmjs.org}
        GOPROXY: ${GOPROXY:-https://proxy.golang.org,direct}
        ALPINE_MIRROR: ${ALPINE_MIRROR:-https://dl-cdn.alpinelinux.org/alpine}
    container_name: tunnel-manager
    restart: unless-stopped
    ports:
      - "8080:8080"
    # environment / volumes 与上方完全一致，略
```

`install.sh` 执行的就是这条路径。`build.args` 的三个镜像源只影响构建期下载速度。

## 环境变量放 `.env`

compose 会自动读取同目录下的 `.env` 文件做变量替换。把 Cloudflare 凭据、`API_KEY`、`APP_ENCRYPTION_KEY` 都写在这里（模板见 [.env.example](https://github.com/qiuyuxc/tunnel-manager/blob/main/.env.example)），不要写进 yml 提交到仓库。变量含义见[快速开始](/guide/getting-started#环境变量)。

## 常见修改

| 想做什么 | 改哪里 |
| --- | --- |
| 换端口 | `ports` 改为 `"9090:8080"`（宿主机端口在前），例如改成 `"8081:8080"` 让面板走 8081 |
| 只走反向代理 | 绑定到 `127.0.0.1:8080:8080` 仅本机监听，由 Nginx/Caddy 转发 |
| 锁定版本 | 镜像 tag 由 `latest` 换成具体版本：发布过的版本直接用 `v1.16.1`、`1.16` 这样的语义化版本号；任意提交可用 `sha-<短哈希>` 回溯 |
| 数据另存路径 | `volumes` 左侧改为你想持久化的目录 |

## 日常操作速查

```bash
# 启动 / 重启
docker compose up -d

# 更新镜像并重启（拉镜像方式）
docker compose pull && docker compose up -d

# 改了代码后重建（构建方式）
DOCKER_BUILDKIT=1 docker compose build --no-cache
docker compose up -d

# 看日志（含初始密码）
docker compose logs -f
docker compose logs | grep 密

# 容器内执行命令（如密码重置 CLI）
docker compose exec tunnel-manager ./tunnel-manager --reset-password

# 停止并移除容器（data/ 目录保留在宿主机）
docker compose down
```

`down` 不会删除挂载出来的 `./data`；升级、备份与恢复见[升级、备份与恢复](/guide/upgrade-backup)。
