# Single-user edition (slim)

The single-user edition is a trimmed build of Tunnel Manager for a **single self-hosted administrator**, matching the repository's `slim/single-user` branch. It drops everything to do with multi-person collaboration and keeps only the tunnel / DNS / monitoring features one administrator needs — roughly half the code of the full build.

Teams that need multiple users, group permissions, an admin console or Telegram remote control should use the full edition (the `main` branch).

## Who it's for

- You are the only user and never need to hand out accounts
- You want a smaller attack surface and fewer moving parts (no sign-up, no audit table, no remote-control bot)
- You still want passkeys / two-factor, Cloudflare OAuth, service monitoring with alerts, and public status pages

## What changes vs. the full build

| Removed | Kept |
| --- | --- |
| Multi-user registration, email verification codes, invite codes | A single administrator (set in the install wizard, no sign-up) |
| User groups and the permission model | Password + passkey (WebAuthn) + TOTP two-factor sign-in |
| Admin console (user / group / invite management) | Sign-in rate limiting, Turnstile human verification |
| Administrator audit log | Cloudflare OAuth (PKCE, multi-account) and static tokens |
| Telegram bot **remote control** (commands that drive tunnels / DNS) | Telegram **notifications** (outbound alerts only — a separate token from remote control) |

The system-configuration cards that used to live in the admin console — SMTP, Cloudflare OAuth client, application encryption key, passkey relying party / origins, sign-in rate limiting, Turnstile, and the experimental-features toggle — have moved into the in-panel **Settings** page. The endpoints are unchanged; they simply no longer sit behind an admin-only entry.

Tunnels, domain binding (including batch), DNS management, the IP optimizer lab, service monitoring, public status pages and email alerts all behave exactly as in the full edition.

## Deploying

The same as the full edition, except the installer tracks the `slim/single-user` branch.

```bash
git clone -b slim/single-user https://github.com/qiuyuxc/tunnel-manager.git
cd tunnel-manager
./install.sh
```

`install.sh` clones and, on later updates, pulls the `slim/single-user` branch (never `main`). The first visit opens the install wizard: pick SQLite or PostgreSQL, then set the administrator username and password — that account is the whole panel.

Other options:

- **Docker image**: the slim image is tagged `:slim` (and `:<version>-slim`), e.g. `ghcr.io/qiuyuxc/tunnel-manager:slim`.
- **Binary**: download the pre-release asset for the `vX.Y.Z-slim` tag from [GitHub Releases](https://github.com/qiuyuxc/tunnel-manager/releases).

See [Docker Compose deployment](/en/guide/docker-compose) and [Binary deployment](/en/guide/binary-deploy) for the rest.

## Versioning & update checks

The slim edition releases independently of the full build:

- Versions carry a `-slim` suffix (e.g. `v2.6.0-slim`) and ship as GitHub **pre-releases**.
- The update check in the panel's About page and in the app compares **only against `-slim` releases** — so a full-edition release never shows up as an available update for slim, and vice versa.
- Because slim releases are always marked pre-release, they never become the repository's `releases/latest`, so they can't affect the full edition's update prompt either.

## Keeping in sync with the full edition

`slim/single-user` is a long-lived, strict subset of `main`. Port fixes to shared code (tunnels, DNS, monitors, Cloudflare OAuth, passkeys) by **cherry-picking** the relevant commits from `main`:

```bash
git checkout slim/single-user
git cherry-pick <commit-from-main>
```

Do not `git merge main` into the slim branch — that would try to reintroduce the removed multi-user / audit / remote-control code and conflict across every deleted file.
