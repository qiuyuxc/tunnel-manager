# Telegram bot

Since v2.1.0 the Telegram bot is open to **every user**: each account configures its own bot, that bot can only touch that account's tunnels, DNS records and domain bindings, and users are fully isolated from one another. An administrator's old global bot is migrated to their personal bot at startup, with nothing to reconfigure.

## Getting started

1. Talk to [@BotFather](https://t.me/BotFather) in Telegram, send `/newbot` to create your bot and take note of the token
2. Talk to [@userinfobot](https://t.me/userinfobot) to learn your numeric Telegram ID
3. Open the "TG bot" page in the panel, enter the bot token and the "authorized TG IDs" (the Telegram accounts allowed to command this bot; several, comma-separated), then enable it

Send `/help` to list every command.

## Operating modes

Each account's bot runs in one of two modes:

- **Long polling (default)**: the bot pulls messages from Telegram, so no public callback URL is needed — right for panels without a public entry point.
- **Webhook**: Telegram pushes messages to the panel for faster responses, but the panel needs a publicly reachable HTTPS address.

Choosing webhook, enter the panel's public **HTTPS base URL** (for example `https://panel.example.com`) on the "TG bot" page; the system appends the per-user callback path `/api/telegram/webhook/{userID}` and registers it. The webhook secret is generated automatically and kept in the backend only — it is never displayed and cannot be changed. Switching modes or changing the URL or secret restarts the bot. If the panel is not reachable from the internet, stay on long polling.

## API endpoint (administrators)

A panel-wide setting on the "TG bot" page: every user's remote-control bot and notification delivery goes through it. It defaults to the official `api.telegram.org`; where that is not directly reachable, point it at your own reverse proxy.

## Tunnel and domain commands

Commands are in Chinese, as sent to the bot:

| Command | What it does |
| --- | --- |
| `/直连域名 [hostname]` | Bind directly to the current tunnel in direct mode |
| `/优选绑定 [hostname] [auxiliary hostname] [optimized CNAME, optional]` | Run the full optimized binding; the older `/绑定域名` still works |

## Zone and tunnel selection

| Command | What it does |
| --- | --- |
| `/列出区域` | List available zones (domain names only — no Zone ID to copy) |
| `/选择区域 [domain or zone ID]` | Pick the DNS zone, after which DNS commands may omit it |

The current tunnel can be selected the same way, after which binding commands may omit it.

## DNS commands

| Command | What it does |
| --- | --- |
| `/DNS列表 [zone] [type, optional] [name, optional]` | Compact list, one record per line, long values truncated |
| `/DNS详情 [full hostname]` | Everything about one record (record ID, TTL, proxy status, MX priority), e.g. `/DNS详情 bbs.kukie.cn`; `/DNS详情 [zone] [name]` also works |
| `/DNS添加 [zone, optional] [type] [name] [content] [TTL, optional] [proxy, optional] [MX priority]` | Manages A, AAAA, CNAME, TXT and MX; TTL defaults to auto and proxy to on (TXT and MX turn the proxy off automatically) |
| `/DNS修改 [zone, optional] [name or ID] [type] [new content] [TTL, optional] [proxy, optional]` | Edit by record name (e.g. `/DNS修改 bbs CNAME saas.com`); the older record-ID form still works |
| `/DNS删除 [zone, optional] [name or record ID]` | Returns a one-time confirmation code valid for five minutes |
| `/确认删除 [code]` | Completes the deletion with that code |

## Notes

- DNS deletions need that second confirmation step, to make accidents unlikely
- In group chats, commands accept an `@botname` suffix
- The [notification centre](/en/guide/multi-user#user-notifications) can use a separate bot for alerts; when only one side is configured, a "reuse" button copies the configured token across
- Bot tokens are encrypted at rest with `APP_ENCRYPTION_KEY`
