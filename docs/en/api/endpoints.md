# API reference

## IP selector lab

Lab endpoints require administrator access and return `404` while experimental features are disabled. Writes require a real administrator session, not a static API key. Domain authorization belongs to the administrator who generated the challenge.

| Method | Path | Behavior |
| --- | --- | --- |
| GET | `/api/lab/ip-selector` | Probe and DNS settings; returns `has_secret_key`, never the secret |
| PUT | `/api/lab/ip-selector` | Saves settings; Host/SNI changes preserve independent domain verification, and unverified schedules are disabled |
| GET | `/api/lab/ip-selector/status` | `status` includes progress, authorization, budget, limits, `next_allowed_at`, and errors; `runs` holds up to 20 runs |
| POST | `/api/lab/ip-selector/ownership/challenge` | Body `{"domain":"example.com"}`; returns TXT `record_name` and `token`, revoking previous authorization and scheduling |
| POST | `/api/lab/ip-selector/ownership/verify` | Checks TXT; returns authorization state on success or `403` on failure |
| GET | `/api/lab/ip-selector/ownership/dns` | Requires a real administrator session; returns the user's current DNS connection availability as `available`, optional `account_name`, and `message`, without credentials or a guarantee of zone write permission |
| POST | `/api/lab/ip-selector/ownership/dns` | Body `{"domain":"example.com"}`; adds verification TXT through the current connection and checks DNS, returning `verification`, `record_created`, `verified`, and `message` |
| POST | `/api/lab/ip-selector/run` | Uses persisted settings only; returns `202` on start, with no request-body target overrides |
| POST | `/api/lab/ip-selector/stop` | Cancels work and disables scheduling; `stopped: true` acknowledges the stop request |

Invalid configuration/challenge input returns `400`, authorization/budget/cooldown failures return `403`, and concurrent runs return `409`. Neither `202` nor `stopped: true` means the background task has finished; poll status. The full candidate count is reserved before starting, cancellation does not refund it, and the UTC daily budget survives restarts. See the [lab guide](../guide/lab-ip-selector.md).

New run records include `probe` (the run's `host`, `sni`, `path`, and `statuses`), `rejected` (the rejected count), and up to 100 `diagnostics` with `ip`, `status`, `latency_ms`, and `error`. Failed runs also retain diagnostics; `results` still contains only top-ranked matches. Older records may lack the new fields, and zero rejections may omit the count and diagnostics.

TXT challenge and one-click endpoints use only the requested verification domain, independent of probe Host/SNI. Clients need not save probe settings first. Missing accounts, missing zone permissions, or failed writes return `400`; manual entry remains available. A successful write with pending DNS propagation returns `200` with `verified: false`; wait and check again. `record_created: false` can mean the same record already exists. Retries reuse a valid challenge without replacing other records or enabling scheduling or scanning. The audit action is `lab_ownership_dns`.

Administrators read and update limits through `GET/PUT /api/admin/settings`: `lab_daily_request_limit` (1–10000000, default 100000), `lab_requests_per_second` (1–1000, default 10), and `lab_max_workers` (1–256, default 32). Values must be integers; omitted fields retain stored values. Invalid input returns `400` without saving. Changes apply to subsequent runs, and lowering the budget preserves spent reservations. These are instance limits, not cloud quotas. Status reports the current configured limits; active runs retain their startup snapshot. Reading probe settings caps the displayed worker count to the current ceiling; execution also enforces that ceiling.

## AI assistant

Assistant endpoints require a real user session through `X-Auth-Token`, not a static API key. Conversation ownership is enforced independently of shared provider settings.

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/api/assistant/settings` | Effective mode and configuration status; keys are never returned |
| PUT | `/api/assistant/settings` | Save `scope`, `endpoint`, `model`, `api_key`; `shared` scope and its `shared_enabled` flag are administrator-only |
| GET / POST | `/api/assistant/conversations` | List owned conversations / create a conversation |
| DELETE | `/api/assistant/conversations/{conversationID}` | Delete history, not business resources; unavailable while a task runs |
| POST | `/api/assistant/conversations/{conversationID}/messages` | Submit `message` and `include_resources`; generate replies and pending tasks without executing writes |
| POST | `/api/assistant/conversations/{conversationID}/tasks/{taskID}` | `action`: `execute`, `pause`, `resume`, `cancel` or `retry` |

Execution requires `confirmed: true`. Retry additionally requires `checked_resources: true` and an `unknown` task status. Arguments come from the stored task and cannot be replaced by the client. Completed or in-flight tasks cannot execute again. HTTP 200 means the task state was saved; check for `status: succeeded` to determine execution success.

See [AI assistant](../guide/ai-assistant.md) for privacy and operational limits.

## General authentication

Every endpoint is prefixed with `/api`. Two ways to authenticate:

- **User session**: sign in to receive an `X-Auth-Token`, accepted by every protected endpoint
- **API key**: the `X-API-Key` header (or `?api_key=`), a machine-level equivalent of administrator rights

Ordinary endpoints are checked against group permissions (tunnels / domain binding / DNS records / service monitoring / Cloudflare authorization); administrators are unrestricted, and everything under `/api/admin/*` is administrator-only. Sessions live in the database, so restarting the service does not invalidate them.

## Health

| Method | Path | Description | Auth |
| --- | --- | --- | --- |
| GET | `/api/health` | Health check, returns the current version | none |

## Install wizard

Until the panel has an administrator account the process mounts only these endpoints and the frontend bundle; after installation they all answer `404`.

| Method | Path | Description | Auth |
| --- | --- | --- | --- |
| GET | `/api/setup/status` | Setup state: whether the environment owns the storage, the current database configuration with credentials redacted, the data directory and the setup file path | none |
| POST | `/api/setup/test` | Try a connection with the submitted database form; nothing is stored | none |
| POST | `/api/setup/complete` | Save the storage choice, apply the schema and create the administrator account, then restart the process | none |

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

Every authentication entry point (sign-in, the second step, passkey sign-in, registration, email codes, password recovery and reset) sits behind the sign-in rate limiter: attempts are counted per account and per source IP, and an exhausted budget answers `429` with a `Retry-After` header (seconds) plus a `retry_after` field and a message in the body. Thresholds and the off switch live under "system settings → sign-in protection"; see [Security and administrator authentication](/en/guide/security#sign-in-rate-limiting) for the mechanics.

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
| GET / PUT | `/api/admin/settings` | Registration switch, invite-code mode, default group, email verification, Turnstile site key / secret, audit retention (`audit_retention_days`), the passkey relying party (`passkey_rp_id` / `passkey_origins`), the panel-wide password switch (`password_login_disabled`) and sign-in rate limiting (`rate_limit_enabled`, `rate_limit_per_account`, `rate_limit_per_ip`, `rate_limit_window_minutes`, `rate_limit_familiar_multiplier`, `rate_limit_notify`); omitted fields keep their stored value |
| GET / PUT | `/api/admin/oauth` | Cloudflare OAuth client (client ID / secret / callback / scopes), taking precedence over the environment |
| GET / PUT | `/api/admin/encryption-key` | Application encryption key (the environment wins; a change needs a restart) |
| GET / PUT | `/api/admin/smtp` | SMTP settings (encrypted or plain) |
| POST | `/api/admin/smtp/test` | Send a test email |

### Audit trail

Records administrator and configuration changes: sign-ins and sign-outs, users and groups, invite codes, system settings, tunnels and ingress rules, domain bindings, DNS records, monitors and their targets, and IP-selector lab runs. Read-only browsing is not recorded. Retention is configured under system settings, and expired rows are pruned hourly.

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/admin/audit-logs` | Page through the trail; filters are `actor` (fuzzy username match, also accepts an account ID), `category`, `action`, `from` / `to` (unix seconds) and `page` / `page_size` (default 50, max 200) |
| GET | `/api/admin/audit-logs/stats` | Totals for the `days` window (default 7, max 90): operations `total`, failures `failed`, active accounts `actors` and today's operations `today` |

Every row carries the time, the actor (account ID and username), category and action, the target object, the source IP and whether it succeeded.

## Passkeys (WebAuthn)

A passkey replaces the password with the fingerprint, face unlock or hardware key on the device. The relying party id is derived from the request host (falling back to the panel host) and can be overridden under "system settings → passkeys" for reverse proxies or multi-domain setups; HTTPS is required (localhost excepted). When the relying party cannot be resolved the account page says so and hides the binding entry point.

| Method | Path | Description | Auth |
| --- | --- | --- | --- |
| POST | `/api/auth/passkey/login/begin` | Start a passwordless sign-in; `account` is optional and only narrows what the browser offers — leave it out for a discoverable ceremony | none |
| POST | `/api/auth/passkey/login/finish` | Verify the assertion and finish; the passkey is a strong factor on its own, so no TOTP is layered on top | none |
| POST | `/api/admin/login/2fa/passkey/begin` | Switch the second step to a passkey after a password sign-in | challenge |
| POST | `/api/admin/login/2fa/passkey/finish` | Verify the assertion and complete the two-step sign-in | challenge |
| GET | `/api/account/passkeys` | List this account's passkeys, the relying party and the password switch | user session |
| POST | `/api/account/passkeys/begin` | Begin binding; requires the current password | user session |
| POST | `/api/account/passkeys/finish` | Finish binding and store the credential (`name` is a label, auto-numbered when blank) | user session |
| PUT | `/api/account/passkeys/{id}` | Rename | user session |
| DELETE | `/api/account/passkeys/{id}` | Delete, requires the current password; refuses to remove the last one while password sign-in is off | user session |
| PUT | `/api/account/password-login` | Toggle this account's password sign-in, requires the current password; turning it off needs a bound passkey first | user session |
| PUT | `/api/admin/users/{id}/password-login` | Force another account's password switch, used to recover an account that lost its passkey | administrator |

Every registration flow returns `{ceremony_token, public_key}`, where `public_key` is the browser's WebAuthn dictionary and `ceremony_token` must be echoed back verbatim by the matching finish call (single-use, valid for five minutes). The `-allow-password-login` command-line flag clears the panel-wide switch and every account switch, for when the install has locked itself out.

The native app additionally needs Digital Asset Links, which the backend serves at the site root (no authentication required):

| Method | Path | Description |
| --- | --- | --- |
| GET | `/.well-known/assetlinks.json` | Android Digital Asset Links: the package name and SHA-256 certificate fingerprints come from system settings (`passkey_android_package` / `passkey_android_fingerprints`, falling back to the shared debug keystore's hash); with no valid fingerprint it answers `404` rather than a broken document |

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

Creation and editing accept only A / AAAA / CNAME / TXT / MX / NS / SRV / CAA / PTR. Unknown types are read-only; update/delete first read the existing record through the current user's CF client to check its type. Credentials, the `dns` permission and per-record auditing remain scoped to the current user.

| Field | Rules |
| --- | --- |
| `type`, `name` | Required; type is uppercased, name is trimmed and loses its trailing dot. Supports `@`, wildcards and ASCII/Punycode |
| `ttl` | Omitted defaults to `1` (automatic); otherwise an integer `60–86400`. Explicit `0` / `null` is rejected |
| `proxied` | Omitted defaults to `false`; `null` is rejected. Only A / AAAA / CNAME can be proxied; other types are forced to `false` |
| `content` | Required for the seven non-structured types, up to 4096 bytes; TXT whitespace is preserved. A/AAAA validate the address family; CNAME/MX/NS/PTR require hostname targets |
| `priority` | Required for MX, integer `0–65535`; null MX (`content:"."`) requires `priority:0` |
| `data` | Required for SRV/CAA, rejected for other types. SRV/CAA do not use a `content` string |

SRV uses a full `_service._protocol.name` and `data.priority/weight/port/target`. The numeric fields are integers `0–65535`, and target is a hostname or `.`. Existing legacy `data.service/proto/name` fields are retained and synchronized with the top-level name on rename. CAA uses `data.flags` (integer `0–255`), `data.tag` (1–15 ASCII letters/digits), and `data.value` (single-line string, up to 4096 bytes, empty allowed). Missing/incorrectly typed fields, unknown fields, trailing JSON or a request body over 64 KiB return 400.

```json
{"type":"SRV","name":"_sip._tcp.example.com","ttl":300,"data":{"priority":0,"weight":5,"port":5060,"target":"sip.example.com"}}
```

```json
{"type":"CAA","name":"example.com","data":{"flags":128,"tag":"iodef","value":"mailto:security@example.com"}}
```

GET retains raw `data`, including fields on unknown record types; do not discard it before editing. Successful POST returns 201 and the record; PUT returns 200 and the record. Web/Android bulk creation adds no endpoint: only A / AAAA / TXT / MX / NS / PTR, up to 100 nonempty lines per client batch, validated/deduplicated locally and sent as serial POSTs. Every record is audited separately; the operation is not atomic. Definite failures remain for retry. Uncertain network/provider responses stop the queue and require checking the actual records before explicit retry; a transport error must not be interpreted as proof that no record was created.

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
| GET | `/api/monitors/overview` | Global summary statistics; each bucket carries an `issues` array (`omitempty`, listing only the targets that saw a warn or a down sample that day) with the target identity, `warn` / `down` counts, `incident_count` and the `incidents` outage windows (start and end time, state, run length, status code and error) that the chart drills into |
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
