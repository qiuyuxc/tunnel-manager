# Binary deployment

Don't want to run Docker? Download the prebuilt archive — one binary plus the frontend's static files is all it takes. **linux-amd64** builds are published; on machines with a Go toolchain you can also build from source (see the end of this page).

## Option 1: download the prebuilt archive (recommended)

Grab `tunnel-manager_<version>_linux_amd64.tar.gz` for the release you want from [GitHub Releases](https://github.com/qiuyuxc/tunnel-manager/releases) — every `v*` tag is built and attached automatically. Unpack it:

```bash
tar xzf tunnel-manager_v2.2.3_linux_amd64.tar.gz
cd tunnel-manager_v2.2.3_linux_amd64
```

What's inside:

```text
├── tunnel-manager       # the backend, a single statically linked binary
└── frontend/dist/       # frontend static files
```

## Running it

The binary reads its environment variables and keeps data under the working directory:

```bash
# example layout
# /opt/tunnel-manager/
# ├── tunnel-manager
# ├── frontend/dist/
# └── data/                # created on first run

export STATIC_DIR=frontend/dist
export STORE_PATH=data/tunnel-manager.db
export PORT=8080
export APP_ENCRYPTION_KEY=<Base64-encoded 32-byte key>
# Cloudflare credentials, pick one: the three OAuth variables, or CF_API_TOKEN + CF_ACCOUNT_ID
./tunnel-manager
```

Generate `APP_ENCRYPTION_KEY` with:

```bash
openssl rand -base64 32
```

The first start creates the administrator account and prints its password to standard output — keep that startup log.

::: tip
The frontend files that `STATIC_DIR` points at and the binary should come from the **same release archive**; mixing versions can mean the API and the UI disagree.
:::

## Keeping it alive with systemd

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

Give `.env` mode `600` — it holds your encryption key and Cloudflare credentials.

## Upgrading

Download the new archive, replace the binary and `frontend/dist`, restart the service. Your data stays in the SQLite database that `STORE_PATH` points at; the backup and restore procedure is shared with [Docker deployment](/en/guide/docker-compose) and covered in [Upgrade, backup & restore](/en/guide/upgrade-backup).

## Option 2: build from source (optional)

When you need to change the code, or run on a platform with no prebuilt archive (a phone under Termux, a Raspberry Pi or other ARM devices), build it on any machine with Node.js and Go:

```bash
# 1. build the frontend
cd frontend
npm install
npm run build          # output lands in frontend/dist

# 2. build the backend binary (add GOOS / GOARCH to cross-compile)
cd ../backend
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o tunnel-manager .
```

With the `tunnel-manager` executable in hand, use the same layout and environment variables as above.
