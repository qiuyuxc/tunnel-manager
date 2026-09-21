# Binary deployment

Don't want to run Docker? Download the prebuilt archive — one binary plus the frontend's static files is all it takes. **linux-amd64** builds are published; on machines with a Go toolchain you can also build from source (see the end of this page).

## Option 1: download the prebuilt archive (recommended)

Grab `tunnel-manager_<version>_linux_amd64.tar.gz` for the release you want from [GitHub Releases](https://github.com/qiuyuxc/tunnel-manager/releases) — every `v*` tag is built and attached automatically. Unpack it:

```bash
tar xzf tunnel-manager_v2.4.0_linux_amd64.tar.gz
cd tunnel-manager_v2.4.0_linux_amd64
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

The first visit to the panel opens the install wizard: with no account yet, only the setup page is served. Pick the database (SQLite or PostgreSQL), test the connection, then choose the administrator name and password. You set the password yourself and it is never printed to the logs.

To let the wizard own the storage choice as well, leave both `STORE_PATH` and `DATABASE_URL` unset: the result is written to `data/setup.json` and reused on every later start.

## Using PostgreSQL

SQLite in a single file is enough for one instance. Switch the store to a PostgreSQL server when you need several instances, external backups, or already run one:

```bash
export DATABASE_URL="postgres://tunnel:tunnel@127.0.0.1:5432/tunnel_manager?sslmode=disable"
export DATA_DIR=data
```

- `DATABASE_URL` accepts both `postgres://` and `postgresql://`; every other connection parameter (`sslmode`, `connect_timeout`, multiple hosts) is passed to the driver
- The database itself must exist; tables and migrations are created on startup, so there is no SQL to run by hand
- `DATA_DIR` holds uploads and heartbeat logs. It defaults to `data/` in the working directory when the database is remote
- `DATABASE_URL` takes precedence over `STORE_PATH`; with neither set the store stays at `data/tunnel-manager.db`
- Existing SQLite data is not migrated: a new database means a fresh instance whose install wizard creates a new administrator account. Export the old data as described in [Upgrade, backup & restore](/en/guide/upgrade-backup)

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

Download the new archive, replace the binary and `frontend/dist`, restart the service. Your data stays in the SQLite database that `STORE_PATH` points at, or in the server `DATABASE_URL` names; the backup and restore procedure is shared with [Docker deployment](/en/guide/docker-compose) and covered in [Upgrade, backup & restore](/en/guide/upgrade-backup).

## Option 2: build from source (optional)

When you need to change the code, or run on a platform with no prebuilt archive (a Raspberry Pi or another ARM device), build it on any machine with Node.js and Go:

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
