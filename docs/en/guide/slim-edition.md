# Slim (single-user) edition

The slim edition is a trimmed build of Tunnel Manager for **single-administrator, self-hosted** setups, tracked on the repository's `slim/single-user` branch. It drops every collaboration-oriented capability and keeps only the tunnel / DNS / monitoring features a single administrator uses — roughly half the code of the full edition.

Teams that need multiple users, group permissions, an admin console, or Telegram remote control should use the full edition (the `main` branch).

## Who it's for

- You are the only person using it and don't need to create accounts for others
- You want a smaller attack surface and fewer moving parts (no registration entry, no audit tables, no remote-control bot)
- You still want passkeys / two-factor, Cloudflare OAuth, service monitoring and alerts, and public status pages

## Differences from the full edition

| Removed | Kept |
| --- | --- |
| Multi-user registration, email codes, invite codes | A single administrator (set during install, no registration entry) |
| User groups and the permission system | Password + passkey (WebAuthn) + TOTP two-factor login |
| Admin console (users / groups / invite management) | Login rate limiting, Turnstile bot check |
| Administrator audit log | Cloudflare OAuth (PKCE, multi-account) and static token |
| Telegram bot **remote control** (operating tunnels / DNS via commands) | Telegram **notifications** (outbound alerts only — a separate token from remote control) |

The system-configuration cards that used to live in the admin console — SMTP, the Cloudflare OAuth client, the app encryption key, passkey relying party / origins, login rate limiting, Turnstile, and the experimental-features switch — all move into the in-panel **Global settings** page. The endpoints are unchanged; they simply no longer need an admin-only entry.

Tunnel management, domain binding (including batch), DNS management, the IP optimizer lab, service monitoring and public status pages, and email alerts are all identical to the full edition.

## Deployment

Same as the full edition, except the install script tracks the `slim/single-user` branch.

```bash
git clone -b slim/single-user https://github.com/qiuyuxc/tunnel-manager.git
cd tunnel-manager
./install.sh
```

`install.sh` clones and, on later updates, pulls the `slim/single-user` branch (never `main`). Open the panel for the first time to enter the install wizard: choose SQLite or PostgreSQL, then set the administrator username and password — that account *is* the whole panel.

Other options:

- **Docker image**: the slim image is tagged `:slim` (and `:<version>-slim`), e.g. `ghcr.io/qiuyuxc/tunnel-manager:slim`.
- **Binary**: download the pre-release assets attached to the `vX.Y.Z-slim` tags on [GitHub Releases](https://github.com/qiuyuxc/tunnel-manager/releases).

For the rest of the deployment details see [Docker Compose deployment](/en/guide/docker-compose) and [Binary deployment](/en/guide/binary-deploy).

## Versioning and update checks

The slim edition is released independently of the full edition:

- Version numbers carry a `-slim` suffix (e.g. `v2.7.0-slim`) and are published as **pre-releases** on GitHub.
- The panel's "About" page and the App's update check **compare only against `-slim` releases** — so a new full-edition release never makes the slim edition report "update available", and vice versa.
- Because slim releases are always marked as pre-releases, none of them becomes the repository's `releases/latest`, so they never affect the full edition's update prompt.

## Staying in sync with the full edition

`slim/single-user` is a long-lived trimmed subset of `main`. For fixes to shared code (tunnels, DNS, monitoring, Cloudflare OAuth, passkeys), **cherry-pick** the corresponding commits from `main` onto the slim branch:

```bash
git checkout slim/single-user
git cherry-pick <commit-on-main>
```

Do not `git merge main` wholesale into slim — that tries to reintroduce the deleted multi-user / audit / remote-control code and produces conflicts on every removed file.
