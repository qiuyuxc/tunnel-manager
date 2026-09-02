# Upgrade, backup & restore

## Upgrading

Tunnel Manager is delivered as a Docker image, so upgrading means rebuilding the image and restarting the container:

```bash
# after pulling new code or unpacking a new archive, from the project directory
DOCKER_BUILDKIT=1 docker compose build --no-cache
docker compose up -d
```

The build runs in the same two stages as the initial `install.sh` deployment — frontend, then backend. An upgrade never touches the host's `./data` directory.

## What lives in the data directory

| File / directory | Contents |
| --- | --- |
| `config.json` | The SQLite database, despite the legacy filename: administrator credentials (Argon2id), tunnel selection, binding records, monitor configuration, encrypted OAuth tokens, Telegram settings — all state |
| `heartbeats.json` | Probe history for service monitoring (what the 24-hour latency chart reads) |
| `uploads/` | Uploaded status-page images and site icons |

Inside the container the path is `/app/data`, mounted from `./data` on the host by `docker-compose.yml`.

## Backup

::: danger Back up both halves
The `data/` directory **and** the `APP_ENCRYPTION_KEY` from your environment are a pair. Back up the data without the key and you will not be able to decrypt TOTP secrets or OAuth tokens after a restore; back up the key without the data and there is nothing to restore.
:::

Minimum backup set:

```bash
tar czf tunnel-manager-backup.tar.gz data/ .env
```

`APP_ENCRYPTION_KEY` lives in `.env` (32 bytes, Base64). Keep the archive somewhere other than the machine you are running on.

## Restoring onto a new machine

1. Install Docker, then place the compose file and `.env` as described in [Quick start](/en/guide/getting-started) — using the original key
2. Unpack the old data: `tar xzf tunnel-manager-backup.tar.gz`
3. Start it: `docker compose up -d`
4. Sign in with the original administrator credentials; if 2FA was enabled, your authenticator keeps working

If the key is lost, an instance with 2FA enabled will refuse logins until the correct key is restored while the service is stopped, and OAuth connections need to be authorized once more.
