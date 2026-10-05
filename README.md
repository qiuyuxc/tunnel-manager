<div align="center"><img src="frontend/public/icon.webp" width="72" alt="Tunnel Manager" /></div>

# Tunnel Manager · Single-User Edition (slim)

English | [简体中文](README.zh-CN.md)

> **This is the slim, single-user edition** (branch `slim/single-user`). It is a trimmed build of Tunnel Manager for one self-hosted administrator — no registration, user groups, admin console, audit log, or Telegram remote control. For the full multi-user edition, see the [`main`](https://github.com/qiuyuxc/tunnel-manager/tree/main) branch.

A web control panel for Cloudflare Tunnel, plus a native Android client. Create and route tunnels, bind domains, manage DNS records and fallback origins, probe your services and publish a shareable status page — secured for a single administrator with password + optional passkey / two-factor sign-in.

> 📖 **Documentation**: **[https://docs.kukie.cn/en/](https://docs.kukie.cn/en/)** · slim edition: [Single-user edition](https://docs.kukie.cn/en/guide/slim-edition)

## What this edition changes vs. the full build

| Removed | Kept |
| --- | --- |
| Multi-user registration, invite codes, user groups & permissions | Single administrator (set in the install wizard) |
| Admin console (user / group / invite management) | System settings moved into the in-panel **Settings** page |
| Administrator audit log | Sign-in rate limiting, Turnstile, passkey / TOTP hardening |
| Telegram bot **remote control** | Telegram **notifications** (outbound alerts) |

Everything else is unchanged.

## Features

| Area | What it does | Docs |
| --- | --- | --- |
| Tunnels | Create and delete tunnels, add/edit/remove ingress rules, optionally clean up DNS on delete | [Tunnels & routing](https://docs.kukie.cn/en/guide/tunnels) |
| Domain binding | Two modes — direct CNAME and SaaS custom hostname — with per-group settings for batch binding | [Binding modes](https://docs.kukie.cn/en/guide/domain-binding) · [Batch binding](https://docs.kukie.cn/en/guide/batch-binding) |
| DNS | CRUD for A / AAAA / CNAME / TXT / MX / NS / SRV / CAA / PTR, structured record editing, batch-create previews and per-row outcomes | [DNS records](https://docs.kukie.cn/en/guide/dns-management) |
| AI assistant | Generate configuration tasks through a Chat Completions-compatible connection, then review and explicitly confirm execution; server-persisted conversations, tasks and encrypted credentials | [AI assistant](https://docs.kukie.cn/en/guide/ai-assistant) |
| IP optimizer lab | Independent TXT ownership verification, HTTP probes and diagnostics, configurable budgets and rate limits, optional scheduling and Huawei Cloud DNS updates | [IP optimizer lab](https://docs.kukie.cn/en/guide/lab-ip-selector) |
| Monitoring | HTTP / TCP / ICMP probes, several targets per monitor, seven-day latency bars | [Service monitoring](https://docs.kukie.cn/en/guide/monitors-status) |
| Alerts | Notifies on status transitions only, over email (SMTP) or a Telegram notification bot | [Email & alerts](https://docs.kukie.cn/en/guide/email-alerts) |
| Public status page | Share without login; short paths, custom domains, direct-tunnel or optimized CNAME access, and a custom domain only exposes its own page | [Public status page](https://docs.kukie.cn/en/guide/monitors-status#public-status-page) |
| Cloudflare connection | OAuth 2.0 with PKCE and automatic token refresh, multiple accounts; static API tokens still work | [OAuth connection](https://docs.kukie.cn/en/guide/cloudflare-oauth) |
| Android app | Native client for the same API: overview, monitoring, tunnels, DNS, the IP lab and notifications, with a bottom tab bar and system notifications | [Android app](https://docs.kukie.cn/en/guide/android-app) |
| Security | Argon2id password hashing, passkeys (WebAuthn), TOTP two-factor auth and one-time recovery codes | [Security](https://docs.kukie.cn/en/guide/security) |

## Using the synchronized features

- **Global settings → IP selector** exposes the experimental switch, daily request budget (1–10000000), requests per second (1–1000) and worker ceiling (1–256). Defaults remain 100000 / 10 / 32. Changes apply to subsequent runs; this is an instance budget, not cloud usage. Cancellation does not refund reservations, and the budget resets at UTC midnight.
- **Domain verification** is independent of probe Host/SNI and requires no saved probe settings. Successful verification hides TXT details; keep the DNS record for checks before scans and DNS updates. Previously scheduled jobs remain disabled without verification.
- **AI assistant** shows one administrator connection, with no multi-user sharing controls. Existing shared connection data remains readable and editable. Proposed tasks never execute automatically; check actual resources before retrying an uncertain result.
- **Batch DNS creation** supports A / AAAA / TXT / MX / NS / PTR and does not resubmit successful rows. Account changes or leaving the page stop unsent operations; manually check writes whose responses were lost.
- Back up the database and `APP_ENCRYPTION_KEY` before upgrading. This source sync does not change version numbers or publish a release. The documentation site remains on the main branch.

## Architecture

```text
┌──────────────┐
│  Vue 3 SPA   │──┐      ┌──────────────┐      ┌──────────────────┐
│  Naive UI    │  ├─────▶│  Go REST API │─────▶│ Cloudflare API   │
├──────────────┤  │      │  chi router  │      │ Tunnels / DNS    │
│ Android app  │──┘      └──────────────┘      └──────────────────┘
│  native UI   │
└──────────────┘
```

## Quick start

```bash
tar xzf tunnel-manager.tar.gz
cd tunnel-manager
./install.sh
```

The script walks you through a Cloudflare OAuth client (or a compatible API token), writes an `APP_ENCRYPTION_KEY`, then builds and starts the service on port `8080`. The first visit to the panel opens the install wizard: pick SQLite or PostgreSQL, test the connection, then choose the administrator name and password — that single account is the whole panel; there is no registration and no password printed to the logs.

**Shorter paths**:
- Run the prebuilt image — see [Docker Compose deployment](https://docs.kukie.cn/en/guide/docker-compose). The slim image is published as `:slim` (and `:<version>-slim`).
- No Docker, just the binary — see [Binary deployment](https://docs.kukie.cn/en/guide/binary-deploy)
- The phone client — build it from `android/`, see [Android app](https://docs.kukie.cn/en/guide/android-app)
- Published builds: [GitHub Releases](https://github.com/qiuyuxc/tunnel-manager/releases) — slim builds are tagged `vX.Y.Z-slim` and marked as pre-releases.

Environment variables, the OAuth setup walkthrough, enabling two-factor auth and resetting passwords are all covered in the [documentation](https://docs.kukie.cn/en/).

## Stack

| Layer | Technology |
| --- | --- |
| Frontend | Vue 3, TypeScript, Naive UI, Vite, Pinia |
| Backend | Go, chi, SQLite / PostgreSQL |
| Android app | Java, Gradle, Android SDK (API 26+) |
| Delivery | Multi-stage Docker build, GitHub Actions CI |

## Development

```bash
# Backend
cd backend && go run .

# Frontend
cd frontend && npm install && npm run dev   # /api is proxied to localhost:8080

# Android app
cd android && ./build.sh

# Checks
cd backend && go test ./... && go vet ./...
cd frontend && npm run build
```

Font assets: `frontend/public/fonts/` holds pre-split MiSans subsets generated by `frontend/scripts/subset-fonts.py` (needs `fonttools` and `brotli`). The full-coverage weights are not tracked in the repository — download them from Xiaomi if you need to regenerate.

### Keeping the slim branch in sync with `main`

`slim/single-user` is a long-lived, strict subset of `main`. Port fixes to shared code (tunnels, DNS, monitors, Cloudflare OAuth, passkeys) by **cherry-picking** the specific commits from `main` — never a full `git merge`, which would try to resurrect the removed code and conflict across every deleted file.

## License

[MIT](LICENSE)
