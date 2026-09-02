# Multi-user & admin console

Since v2.0.0 Tunnel Manager supports user registration and an admin console. Everything is stored in SQLite (`data/tunnel-manager.db`); a legacy `config.json` is imported automatically on first start — the original administrator account, OAuth connection and monitors all carry over, and the old file is kept as a backup.

## Registration policy (Admin console → System settings)

| Switch | What it does |
| --- | --- |
| Open registration | Turn it off and the login page stops offering sign-up |
| Email verification | Requires SMTP first; when on, registration asks for a code sent by email |
| Invite codes | Off / optional / required, and a code can be tied to a user group |

How they combine:

- No mail service and invite codes off: an email address and a password are enough to register
- Mail service configured: registration requires the emailed code (valid for 10 minutes, no resend within 60 seconds)
- Invite codes required: a valid code is needed as well

## Users and groups

- Every user has their own Cloudflare authorization, tunnel selection and monitors; none of it is visible to anyone else
- A user who has not authorized their own Cloudflare account cannot see or touch any tunnel, domain, DNS record or monitor — they need to finish authorization under "Account settings → Cloudflare authorization" first. Unauthorized users never fall back to the administrator's static credentials
- Groups carry the permissions: tunnels, domain binding, DNS records, service monitoring, Cloudflare authorization
- The built-in "default group" decides what a newly registered user can do; the administrator is not restricted by permissions
- Administrators can disable or delete users (resetting a password invalidates all of that user's sessions)

## Invite codes

Generated in the admin console, optionally tied to a group, with a usage limit and an expiry. A code is only counted once a registration actually succeeds — failed attempts do not consume it.

## Personal profile

The "Profile" card on the "Account settings" page manages how the current account presents itself:

- **Display name**: a nickname for display only, never used to sign in
- **Avatar**: upload a local file (png / jpg / gif / webp, up to 4 MiB) or point at an image URL; uploads are saved to the account automatically
- **Account email**: bind one if there is none, or change an existing one. Both need the current password for confirmation, and no verification mail is sent to the new address — it takes effect as soon as the password check passes. The address receives registration and reset codes plus notifications

## User notifications

Each user configures their own alerts on the "Notifications" page, entirely separate from the panel administrator's Telegram bot:

- **Channels**: off / email only / Telegram only / email + Telegram
- **Email**: one address per line, several recipients allowed (sent through the global SMTP settings)
- **Telegram**: your own bot token and chat ID — the panel calls your bot directly
- **Events**: currently "login notification", sent after every successful sign-in with the time and IP, and switchable at any time

Save, then use "Send a test notification" to confirm it works. Telegram tokens are encrypted at rest with `APP_ENCRYPTION_KEY`.

## Telegram remote control

The "TG bot" page is open to every signed-in user, and each user runs **their own bot** in complete isolation:

- Create a bot with @BotFather in Telegram and paste its token; talk to @userinfobot to learn your numeric Telegram ID
- "Authorized TG IDs" lists the Telegram accounts allowed to command that bot (several, comma-separated)
- Once enabled the bot runs by long polling or webhook (see [TG bot settings](/en/guide/telegram-bot#operating-modes)) and can only touch **that user's own resources**: tunnel selection, forwarding address, DNS records, domain bindings, Cloudflare zones
- Notifications and remote control may use different bots; when only one side is configured, a "reuse" button appears to copy the token across
- **API endpoint**: a panel-wide setting (administrators configure it on the "TG bot" page) used by every user's bot and notification. Mainland China networks generally need a self-hosted reverse proxy, since the official `api.telegram.org` is not directly reachable there
- The commands match the old administrator bot (`/help` lists them), scoped to your own account
- An administrator's previous global bot configuration is migrated to their personal bot automatically

Each user's bot token is likewise encrypted with `APP_ENCRYPTION_KEY`.

## Forgotten passwords

- With SMTP configured: "Forgot password?" on the login page emails a reset code to the registered address for self-service recovery
- Without SMTP: ask an administrator to reset it in the admin console; the administrator themselves can use `./tunnel-manager -set-password NEW_PASSWORD` on the command line
