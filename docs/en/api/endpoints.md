# API reference

Every endpoint is prefixed with `/api`. Two ways to authenticate:

- **User session**: sign in to receive an `X-Auth-Token`, accepted by every protected endpoint
- **API key**: the `X-API-Key` header (or `?api_key=`), a machine-level equivalent of administrator rights

Ordinary endpoints are checked against group permissions (tunnels / domain binding / DNS records / service monitoring / Cloudflare authorization); administrators are unrestricted, and everything under `/api/admin/*` is administrator-only. Sessions live in the database, so restarting the service does not invalidate them.

## Health

| Method | Path | Description | Auth |
| --- | --- | --- | --- |
| GET | `/api/health` | Health check, returns the current version | none |

## Registration & identity

| Method | Path | Description | Auth |
| --- | --- | --- | --- |
| GET | `/api/auth/config` | Registration policy: whether sign-up is open, invite-code mode, whether an email code is required, plus the Turnstile switch and site key | none |
| POST | `/api/auth/register` | Register and sign in; invite codes and email codes are validated server-side | none |
| POST | `/api/auth/send-code` | Send a registration email code (needs SMTP, 60-second cooldown) | none |
| POST | `/api/auth/forgot-password` | Send a password-reset code (needs SMTP) | none |
| POST | `/api/auth/reset-password` | Set a new password with a reset code, dropping all sessions on success | none |
| GET | `/api/auth/me` | The signed-in identity: ID, username, display name, avatar, email, role and permissions | user session |

## Sign-in & sessions

| Method | Path | Description | Auth |
| --- | --- | --- | --- |
| POST | `/api/admin/login` | Sign in (`account` accepts an email or a username); returns a challenge when 2FA is on | none |
| POST | `/api/admin/login/2fa` | Complete sign-in with a TOTP or recovery code | challenge |
| POST | `/api/admin/logout` | End the current session | user session |
| GET | `/api/admin/status` | Check session validity, returns username and role | user session |
| PUT | `/api/admin/password` | Change the current user's password (revokes all of that user's sessions) | user session |
| PUT | `/api/admin/username` | Change the current user's username (password required) | user session |
| PUT | `/api/admin/email` | Bind or change the current user's email (password required) | user session |
| PUT | `/api/admin/profile` | Update the current user's display name and avatar URL | user session |
| POST | `/api/account/avatar` | Upload an avatar image (multipart `file`), saved to the account | user session |
| GET | `/api/notify/settings` | The current user's notification settings (channels, events, email, Telegram status) | user session |
| PUT | `/api/notify/settings` | Save notification settings; an empty `tg_bot_token` keeps the stored one, and responses never return it | user session |
| POST | `/api/notify/test` | Send a test notification with the current configuration | user session |
| GET | `/api/admin/2fa/status` | The current user's 2FA state | user session |
| POST | `/api/admin/2fa/setup` | Begin authenticator enrollment | user session |
| POST | `/api/admin/2fa/confirm` | Confirm enrollment and generate recovery codes | user session |
| POST | `/api/admin/2fa/disable` | Turn 2FA off (password plus a TOTP or recovery code) | user session |

## Admin console (administrators only)

### Users

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/admin/users` | List all users, with group and last sign-in |
| POST | `/api/admin/users` | Create a user (role and group optional) |
| PUT | `/api/admin/users/{id}/status` | Enable / disable (disabling also signs them out) |
| PUT | `/api/admin/users/{id}/group` | Change the group |
| PUT | `/api/admin/users/{id}/password` | Reset the password (signs that user out) |
| DELETE | `/api/admin/users/{id}` | Delete a user (never yourself or the last administrator) |

### Groups

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/admin/groups` | List groups and their permission sets |
| POST | `/api/admin/groups` | Create a group |
| PUT | `/api/admin/groups/{id}` | Update name and permissions (built-in groups allow permissions only) |
| DELETE | `/api/admin/groups/{id}` | Delete a group (not built-in ones, not ones with members) |

### Invite codes

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/admin/invites` | List invite codes |
| POST | `/api/admin/invites` | Generate one (group, use count, expiry) |
| PUT | `/api/admin/invites/{code}` | Enable / disable |
| DELETE | `/api/admin/invites/{code}` | Delete |

### System settings

| Method | Path | Description |
| --- | --- | --- |
| GET / PUT | `/api/admin/settings` | Registration switch, invite-code mode, default group, email verification, Turnstile site key / secret |
| GET / PUT | `/api/admin/oauth` | Cloudflare OAuth client (client ID / secret / callback / scopes), taking precedence over the environment |
| GET / PUT | `/api/admin/encryption-key` | Application encryption key (the environment wins; a change needs a restart) |
| GET / PUT | `/api/admin/smtp` | SMTP settings (encrypted or plain) |
| POST | `/api/admin/smtp/test` | Send a test email |

## Cloudflare OAuth

Each user may authorize several Cloudflare accounts and switch between connections at will.

| Method | Path | Description | Auth |
| --- | --- | --- | --- |
| GET | `/api/cloudflare/oauth/status` | Connection list, the active one and available accounts | user session |
| POST | `/api/cloudflare/oauth/start` | Begin an authorization request (adding an account never overwrites older connections) | `oauth_connect` |
| GET | `/api/cloudflare/oauth/callback` | Cloudflare's callback; creates a connection and activates it | OAuth state |
| PUT | `/api/cloudflare/oauth/connection` | Switch the active connection | `oauth_connect` |
| PUT | `/api/cloudflare/oauth/account` | Switch the Cloudflare account within the active connection | `oauth_connect` |
| DELETE | `/api/cloudflare/oauth` | Revoke and delete a connection (`?connection_id=`, defaults to the active one) | `oauth_connect` |

## Configuration & site branding

Tunnel selection and the service address are **per user**; site branding and the optimized CNAME are global, administrator-only settings.

| Method | Path | Description | Auth |
| --- | --- | --- | --- |
| GET | `/api/config` | Read configuration (global branding plus this user's selections) | user session |
| GET | `/api/site` | Public site branding | none |
| POST | `/api/config/tunnel` | Set this user's tunnel | user session |
| POST | `/api/config/service` | Set this user's service address | user session |
| POST | `/api/config/preferred-cname` | Set the global optimized CNAME | administrator |
| PUT | `/api/config/site` | Update site branding | administrator |
| PUT | `/api/config/cname-presets` | Update the CNAME presets | administrator |

## Tunnels

Requires the `tunnels` permission; calls run against the Cloudflare account of the user's active connection.

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/tunnels` | List tunnels |
| POST | `/api/tunnels` | Create a tunnel and return its connector token |
| GET | `/api/tunnels/{tunnelID}` | Tunnel detail and routes |
| DELETE | `/api/tunnels/{tunnelID}` | Delete a tunnel |
| POST | `/api/tunnels/{tunnelID}/ingress` | Add an application route |
| PUT | `/api/tunnels/{tunnelID}/ingress` | Update an application route |
| DELETE | `/api/tunnels/{tunnelID}/ingress` | Delete a route, optionally with its DNS record |

## DNS records

Requires the `dns` permission.

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/zones` | List zones |
| GET | `/api/zones/{zoneID}/dns-records` | List a zone's DNS records |
| POST | `/api/zones/{zoneID}/dns-records` | Create a record |
| PUT | `/api/zones/{zoneID}/dns-records/{recordID}` | Edit a record |
| DELETE | `/api/zones/{zoneID}/dns-records/{recordID}` | Delete a record |

## Domain binding

Requires the `domain_bind` permission.

| Method | Path | Description |
| --- | --- | --- |
| POST | `/api/domain/bind` | Bind one group (`mode` selects direct or optimized) |
| POST | `/api/domain/bind-batch` | Bind several groups, each with its own mode and service address |
| POST | `/api/domain/fallback` | Set the fallback origin |

## Monitoring & status pages

Requires the `monitors` permission. Monitors are isolated by creator: ordinary users see only their own, administrators see all. The create and update endpoints also carry the public-page, alert and recipient settings.

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/monitors` | List monitors and target states |
| POST | `/api/monitors` | Create a monitor |
| GET | `/api/monitors/overview` | Global summary statistics |
| PUT | `/api/monitors/{monitorID}` | Update configuration (public page, alert switch, recipients …) |
| DELETE | `/api/monitors/{monitorID}` | Delete a monitor |
| POST | `/api/monitors/{monitorID}/check` | Run one probe immediately |
| GET | `/api/monitors/{monitorID}/alerts` | The last 100 alert records (delivered or why not) |
| POST | `/api/monitors/{monitorID}/targets` | Add a probe target |
| PUT | `/api/monitors/{monitorID}/targets/{targetID}` | Edit a probe target |
| DELETE | `/api/monitors/{monitorID}/targets/{targetID}` | Delete a probe target |
| GET | `/api/public/status/{token}` | Public status data; the token may be the system token or the short path |

Custom-domain fields on `PUT /api/monitors/{monitorID}`:

| Field | Description |
| --- | --- |
| `public_domain` | The status page hostname; empty keeps only the link under the panel domain |
| `public_domain_mode` | `simple` for direct, `preferred` for optimized |
| `public_aux_domain` | Required in optimized mode: the proxied hostname that reaches the tunnel |
| `public_preferred_cname` | The optimized CNAME for this monitor; empty reads the global default |
| `domain_warning` | A response field meaning the settings saved but Cloudflare configuration failed |

In optimized mode the status hostname and the auxiliary hostname must differ, and the optimized CNAME may not equal the status hostname. Changing any of these fields re-runs the Cloudflare configuration. A failure still returns HTTP 200 and explains itself through `domain_warning`.

A custom domain only ever exposes that monitor's public page and the assets it needs. Other status pages, the admin console and protected APIs return 404.

## Uploads & Telegram

Telegram remote control is per user: each account configures its own bot token and authorized Telegram IDs, and the bot only touches that account's resources. An administrator's previous global bot configuration is migrated to their personal settings at startup.

| Method | Path | Description | Auth |
| --- | --- | --- | --- |
| POST | `/api/uploads` | Upload a status-page image | `monitors` |
| GET | `/api/telegram/settings` | This user's bot settings | user session |
| PUT | `/api/telegram/settings` | Save this user's bot settings and restart their bot | user session |
| GET | `/api/telegram/status` | This user's bot status | user session |
| POST | `/api/telegram/test` | Send a test message to this user's authorized Telegram IDs | user session |
| PUT | `/api/telegram/endpoint` | Set the panel-wide Telegram API endpoint (a custom proxy) for every user's bot | administrator |
| POST | `/api/telegram/webhook` | Webhook entry point for the legacy global bot (backwards compatibility) | secret token |
| POST | `/api/telegram/webhook/{userID}` | Per-user webhook entry point, dispatched only to the bot for that userID | secret token |
