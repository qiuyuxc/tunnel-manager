# Architecture & stack

## Overview

```text
┌──────────────┐     ┌──────────────┐     ┌──────────────────┐
│  Vue 3 SPA   │────▶│  Go REST API │────▶│ Cloudflare API   │
│  Naive UI    │     │  chi router  │     │ Tunnels / DNS    │
└──────────────┘     └──────────────┘     └──────────────────┘
```

Single container: the Go process serves the frontend's static files directly (`STATIC_DIR=frontend/dist`), so there is no separate Node runtime.

## Stack

| Layer | Technology |
| --- | --- |
| Frontend | Vue 3, TypeScript, Naive UI, Vite, Pinia |
| Backend | Go, chi, SQLite (`modernc.org/sqlite`, pure-Go driver) |
| Security | Argon2id, TOTP, AES-GCM encryption at rest |
| Delivery | Multi-stage Docker build, Docker Compose |

## Backend layout

```text
backend/
├── auth/              # TOTP, AES-GCM and recovery codes
├── db/                # SQLite connection and versioned schema migrations
├── handlers/          # HTTP handlers and auth middleware
├── services/          # Cloudflare, domain binding, Telegram and monitoring logic
├── models/            # Data types
├── store/             # SQLite storage layer (with automatic legacy JSON import)
└── main.go            # Entry point and route registration
```

Key mechanisms:

- **Credential chain**: OAuth first (PKCE with automatic refresh), falling back to a static token; OAuth tokens are encrypted at rest with AES-GCM using `APP_ENCRYPTION_KEY`
- **Storage**: one SQLite database (`data/tunnel-manager.db` by default, WAL mode); a legacy `config.json` is imported on first start and kept as a backup
- **Monitoring**: a runner goroutine schedules probes per interval, writing heartbeats to a separate `heartbeats.json` that is flushed periodically
- **Telegram bot**: long polling or webhook, and it only answers the administrator IDs in its configuration
