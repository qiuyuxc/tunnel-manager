# Quick start

Tunnel Manager ships as a single Docker container — the Vue 3 frontend and the Go backend live in the same image, and all state is persisted to a `data/` directory on the host.

## One-command deployment

```bash
tar xzf tunnel-manager.tar.gz
cd tunnel-manager
./install.sh
```

The script checks your Docker setup, walks you through a Cloudflare OAuth client (or a compatible API token), writes an `APP_ENCRYPTION_KEY`, then builds and starts the service; the administrator account is created by the install wizard on the first visit to the panel. It listens on port `8080` by default (`docker-compose.yml` maps it to host port 8080).

> Prefer not to build from source? Run the prebuilt image, or skip Docker entirely with the binary — see [Docker Compose deployment](/en/guide/docker-compose) and [Binary deployment](/en/guide/binary-deploy).

## Creating the administrator account

The first visit to the panel opens the install wizard. Until an account exists the panel serves only that page: pick the database (a single SQLite file or a PostgreSQL server, whose connection you can test first), then choose the administrator name and password. Installation restarts the service and drops you on the sign-in page. The password is the one you typed, it stays in the database and is never printed to the logs.

To skip it, set `ADMIN_PASSWORD` before the first start: the account is created at boot with that password and the panel goes straight to the sign-in page.

## Environment variables

| Variable | Required | Description |
| --- | --- | --- |
| `CF_OAUTH_CLIENT_ID` | for OAuth | Cloudflare OAuth client ID |
| `CF_OAUTH_CLIENT_SECRET` | for OAuth | Client secret; kept only in the server's environment |
| `CF_OAUTH_REDIRECT_URI` | recommended | OAuth callback URL; must match the value registered with Cloudflare exactly. Derived from the request when empty |
| `CF_OAUTH_SCOPES` | no | Space-separated scopes to request explicitly; empty uses whatever the OAuth client is configured with |
| `CF_API_TOKEN` | compatibility | Legacy static Cloudflare API token, used when OAuth is not connected |
| `CF_ACCOUNT_ID` | compatibility | Account ID for the static token; OAuth reads and stores the account itself |
| `API_KEY` | no | API key for automated calls |
| `ADMIN_PASSWORD` | no | Administrator password for the first start; leave it empty to choose one in the web install wizard |
| `APP_ENCRYPTION_KEY` | for 2FA | Base64-encoded 32-byte random key that encrypts TOTP secrets and OAuth tokens |
| `STORE_PATH` | no | Path to the SQLite database. The binary defaults to `data/tunnel-manager.db`; the Docker image sets `data/config.json`, a legacy filename kept so older installs upgrade in place |
| `DATABASE_URL` | no | PostgreSQL connection string such as `postgres://user:pass@host:5432/tunnel_manager?sslmode=disable`. Takes precedence over `STORE_PATH` |
| `DATA_DIR` | no | Directory for uploads and heartbeat logs. Defaults to the SQLite file's directory; set it explicitly when the store lives on PostgreSQL |
| `PORT` | no | HTTP port, `8080` by default |

::: warning Back up the encryption key
`APP_ENCRYPTION_KEY` must be backed up together with your data files. It protects both TOTP secrets and Cloudflare OAuth tokens: lose or replace it and an instance with 2FA enabled will refuse administrator logins, while OAuth connections have to be authorized again. See [Security & admin auth](/en/guide/security).
:::

## Persistence and upgrades

`docker-compose.yml` mounts the host's `./data` directory at `/app/data` inside the container. Your tunnel list, binding records, monitoring history and administrator credentials all live there, so keeping that directory is all an upgrade needs.

Once OAuth is connected, either pass `X-Forwarded-Proto` and `X-Forwarded-Host` through your reverse proxy or set `CF_OAUTH_REDIRECT_URI` explicitly, so the callback URL is never derived incorrectly.

## Local development

```bash
# Backend
cd backend
go run .

# Frontend
cd frontend
npm install
npm run dev
```

The frontend dev server proxies `/api` to `http://localhost:8080`.

Checks:

```bash
cd backend && go test ./... && go vet ./...
cd frontend && npm run build
```
