# Docker Compose deployment

Pick either path: **option 1 pulls the official image** (recommended for most people — no Node or Go needed), **option 2 builds from source** (for developers, or when you want to modify things). Their `environment` and `volumes` are identical.

## The full compose file

### Option 1: pull the prebuilt image (recommended)

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

A single `docker-compose.yml` plus an `.env` file is enough — no need to clone the repository.

::: tip Faster pulls behind a slow ghcr.io
When ghcr.io is slow, swap the image prefix for a mirror, for example:

```yaml
    image: ghcr.nju.edu.cn/qiuyuxc/tunnel-manager:latest
```

The mirror proxies ghcr.io's contents, so tags and digests stay the same.
:::

### Option 2: build from source

From the root of a cloned repository, use the build-based configuration:

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
    # environment / volumes are identical to the block above
```

This is the path `install.sh` takes. Those three `build.args` mirrors only affect download speed during the build.

## Keep environment variables in `.env`

Compose reads an `.env` file from the same directory for variable substitution. Put the Cloudflare credentials, `API_KEY` and `APP_ENCRYPTION_KEY` there (template: [.env.example](https://github.com/qiuyuxc/tunnel-manager/blob/main/.env.example)) rather than committing them inside the yml. Each variable is described under [Quick start](/en/guide/getting-started#environment-variables).

## Common adjustments

| Goal | What to change |
| --- | --- |
| Different port | Set `ports` to `"9090:8080"` (host port first) — `"8081:8080"` puts the panel on 8081 |
| Reverse proxy only | Bind `127.0.0.1:8080:8080` so it listens locally and let Nginx/Caddy forward to it |
| Pin the version | Replace the `latest` tag with a specific one: released versions accept semantic tags like `v2.2.3` or `2.2`, and any commit can be pinned with `sha-<short-hash>` |
| Store data elsewhere | Change the left-hand side of the `volumes` entry to the directory you want |

## Everyday commands

```bash
# start / restart
docker compose up -d

# update the image and restart (image-based setup)
docker compose pull && docker compose up -d

# rebuild after changing code (build-based setup)
DOCKER_BUILDKIT=1 docker compose build --no-cache
docker compose up -d

# logs, including the initial password (the banner is in Chinese)
docker compose logs -f
docker compose logs | grep 密

# run a command inside the container, e.g. the password-reset CLI
docker compose exec tunnel-manager ./tunnel-manager --reset-password

# stop and remove the container (./data stays on the host)
docker compose down
```

`down` never deletes the mounted `./data`. For upgrades see [Upgrade, backup & restore](/en/guide/upgrade-backup).
