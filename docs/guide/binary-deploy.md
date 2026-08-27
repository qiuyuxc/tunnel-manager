# 二进制部署

不想跑 Docker？下载预编译包，一个二进制加前端静态文件即可运行。当前提供 **linux-amd64** 包；有 Go 环境的设备也可从源码自行编译（见文末）。

## 方式一：下载预编译包（推荐）

到 [GitHub Releases](https://github.com/qiuyuxc/tunnel-manager/releases) 下载对应版本的 `tunnel-manager_<版本>_linux_amd64.tar.gz`（每个打了 `v*` 标签的版本都会自动构建并附加）。解包：

```bash
tar xzf tunnel-manager_v1.16.1_linux_amd64.tar.gz
cd tunnel-manager_v1.16.1_linux_amd64
```

目录内容：

```text
├── tunnel-manager       # 后端单二进制（静态编译，无依赖）
└── frontend/dist/       # 前端静态文件
```

## 运行方式

二进制直接读取环境变量与工作目录下的数据：

```bash
# 目录布局（示例）
# /opt/tunnel-manager/
# ├── tunnel-manager
# ├── frontend/dist/
# └── data/                # 首次运行自动生成

export STATIC_DIR=frontend/dist
export STORE_PATH=data/config.json
export PORT=8080
export APP_ENCRYPTION_KEY=<Base64 的 32 字节密钥>
# Cloudflare 凭据二选一：OAuth 三件套 或 CF_API_TOKEN + CF_ACCOUNT_ID
./tunnel-manager
```

`APP_ENCRYPTION_KEY` 生成方式：

```bash
openssl rand -base64 32
```

首次启动会自动生成管理员账户，密码打印在标准输出中——记得保留这份启动日志。

::: tip
`STATIC_DIR` 指向的前端文件与二进制应来自**同一个发布包**，前后端版本混搭可能出现接口不匹配。
:::

## 建议配合 systemd 常驻

```ini
# /etc/systemd/system/tunnel-manager.service
[Unit]
After=network-online.target
Wants=network-online.target

[Service]
WorkingDirectory=/opt/tunnel-manager
EnvironmentFile=/opt/tunnel-manager/.env
ExecStart=/opt/tunnel-manager/tunnel-manager
Restart=unless-stopped

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload && systemctl enable --now tunnel-manager
```

注意 `.env` 权限建议 `600`，里面有加密密钥与 Cloudflare 凭据。

## 升级

下载新版本包替换二进制与 `frontend/dist` 并重启服务即可；数据仍在 `STORE_PATH` 指向的 JSON 中，备份恢复流程与 [Docker 部署](/guide/docker-compose)通用，详见[升级、备份与恢复](/guide/upgrade-backup)。

## 方式二：从源码自行编译（可选）

需要改动代码、或在不提供预编译包的平台（如 Termux 手机、树莓派等 ARM 设备）上运行时，可在装有 Node.js 与 Go 的机器上自行构建：

```bash
# 1. 构建前端静态文件
cd frontend
npm install
npm run build          # 产物在 frontend/dist

# 2. 编译后端单二进制（交叉编译时追加 GOOS / GOARCH）
cd ../backend
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o tunnel-manager .
```

得到 `tunnel-manager` 可执行文件后，按上文相同的目录布局与环境变量运行即可。
