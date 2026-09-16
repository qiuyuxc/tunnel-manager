# Changelog

Compiled from the repository's release commits; older details are on [GitHub Releases](https://github.com/qiuyuxc/tunnel-manager/releases).

## v2.5.0

- Passkeys (WebAuthn): the account page binds fingerprints, face unlock or a hardware security key; the sign-in page accepts a passkey on its own, and an account with two-factor enabled can use one instead of a TOTP code
- "Disable password sign-in" comes in two sizes: an account can turn off its own password (a passkey must be bound first), and an administrator can require passkeys for the whole panel; the panel-wide switch only turns on once an active administrator has a passkey, so the install cannot lock itself out
- There is always a way back in: `-allow-password-login` clears the panel switch and every account switch, and the account list has a one-click "restore password sign-in" action
- The relying party is derived from the request host (falling back to the panel host); reverse proxies and multi-domain setups can set the relying party id and allowed origins under "system settings → passkeys", where each origin must share the relying party's domain
- Binding, removing and renaming a passkey are audited, as are passkey sign-in successes and failures (a new "passkey" category); failed attempts keep the source IP
- Operations that would break the sign-in path are refused: removing the last passkey, disabling password sign-in without one, deleting an account
- The Android app gains the same passkey support: sign in with a passkey, use one for the second step, and bind / rename / remove credentials or toggle password sign-in from the account page; the admin console gains an audit tab (account, operation type and time-range filters, paging and totals) plus the passkey settings
- The backend serves `/.well-known/assetlinks.json`: Android checks the domain through Digital Asset Links, and the package name plus SHA-256 certificate fingerprints are maintained under "system settings → passkeys", prefilled with the shared debug keystore's hash
- New dependencies: `github.com/go-webauthn/webauthn` on the backend, `@simplewebauthn/browser` on the frontend, `androidx.credentials` in the app
- New sign-in rate limiting: repeated failures hit a sliding window counted separately per account and per source IP, answering `429` with `Retry-After`; it covers password sign-in, the second step, passkey sign-in, registration, email codes and password resets
- Familiar subnets earn extra room: a successful sign-in remembers the source subnet (IPv4 grouped by `/24`, IPv6 by `/64`, valid for 90 days, up to 32 per account) and failures from it are multiplied against the base budget, while unknown IPs stay on the default
- Thresholds and the kill switch live under "system settings → sign-in protection" (account / IP budget, window, familiar multiplier, and whether to email the administrator on a lockout)
- The overview chart drills down: tapping a column lists that day's incidents grouped by target, with start time, run length, state and failure reason, and jumps to the monitor; the Android app supports the same tap
- New audit trail: the admin panel gains an audit tab that records sign-ins and sign-outs, user and group changes, invite codes, system settings (SMTP, OAuth and the encryption key included), tunnels and ingress rules, domain bindings, DNS records, monitors and their targets, and IP-selector lab runs
- The trail searches by account name, filters by operation type and time range, paginates, and summarizes the last seven days: operations, failures, today's activity and active accounts
- Successful logins, failed logins (two-factor included) and sign-outs are recorded; failed attempts keep the source IP, which is what makes brute-force attempts visible
- Retention defaults to 90 days and is configurable per install (30 / 90 / 180 / 365 days or keep forever); expired rows are pruned hourly
- Audit rows live in their own append-only `audit_logs` table (migration v13) rather than in the configuration document, so the trail never slows down configuration saves
- Read-only browsing is not recorded, and `/api/admin/audit-logs` plus `/api/admin/audit-logs/stats` are administrator-only

## v2.4.1

- Fixed the Android app looping on the launch screen after a session expired: the stale credential was never cleared, so the login page relaunched the console immediately, took another 401 and bounced back — over and over. The session is now cleared, the app returns to the sign-in form and says "登录状态已失效，请重新登录"
- Any `401` from the API is now handed to the console shell as a whole, and handled exactly once; `403` still means "not allowed" and never signs anyone out
- Session lifetime extended from 12 hours to seven days
- The dashboard chart window grew from 12 hours to seven days: one column per day (that day's peak latency and worst state), empty days included, instead of compressing a day into one oversized bar
- Heartbeat history is retained for eight days (up to 10080 samples per target) to feed that window
- The overview endpoint returns `bucket_sec` and `uptime` (`uptime_24h` is kept for older clients); clients fall back to hourly buckets when the field is missing, so an old server with a new client never mislabels the axis

## v2.4.0

- New native Android app: overview, monitoring, tunnels & bindings, DNS, the IP optimizer lab, notifications and About are all native, with a bottom tab bar plus a More page, hitting the same backend API — see [Using the Android app](/en/guide/android-app)
- App notifications ride the system notification channel: a low-frequency background poll of `/api/alerts` brings status changes to the phone without a browser open
- Cloudflare Turnstile is embedded in the native login page, so enabling human verification no longer bounces you to the web console and is immune to leftover browser sessions
- The login page and console follow the theme preference between light and dark (Vercel light / Vercel dark), and the launch splash picks the day or night artwork to match
- New narrow-screen bottom tab navigation and half-sheet panels; monitoring, DNS and the IP optimizer lab no longer squeeze sideways on a phone
- Two landing-page looks: Vercel (enterprise blue, light/dark) and Claude (warm, light/dark)
- Hardened error handling: messages are redacted so a Telegram bot token never reaches a log line, status card or screenshot
- New `/api/alerts` cursor endpoint so a client can pull monitor state changes incrementally with `since`
- The overview chart now always returns twelve hourly buckets, empty ones included, instead of one oversized bar for the hour that happened to get a heartbeat

## v2.3.0

- Added an experimental-features switch: administrators control whether lab navigation is exposed, and the related APIs return `404` while disabled
- Added the IP optimizer lab with configurable Host, TLS SNI, probe path, accepted status codes, IP ranges, workers, timeout and Top N
- Added live progress, execution history and per-segment hit statistics, plus one-click copy and removal of zero-hit IP ranges
- Added scheduled runs and automatic Huawei Cloud DNS A-record updates; the Secret Key is encrypted at rest with the application key using AES-GCM
- Added bilingual lab documentation, API references and About-page copy

## v2.2.3

- Much smaller first paint: MiSans now loads as a core subset plus unicode-range shards, where the core subset already covers every character in the interface. First-paint font weight drops from roughly 19 MB to 373 KB, and only uncommon characters in user content pull an extra shard
- The backend gzips static assets: the main JS bundle goes from 450 KB to 149 KB and the font stylesheet from 116 KB to 9 KB
- Font files carry long-lived cache headers, so repeat visits no longer revalidate them one by one; a request for a missing font no longer falls back to the frontend index page
- Responsive breakpoints unified to 480 / 640 / 768 / 1024 (there were 11 different values), so pages no longer reflow at their own widths while a window resizes
- Page-header spacing unified, removing nine pages' duplicate overrides of the shared style
- Fixed the dashboard's quick actions showing the wrong number of columns between 640 and 768 px

## v2.2.2

- Fixed domain binding always using the global configuration: whichever tunnel or port you picked, the ingress rule and CNAME landed on the globally configured ones. It now prefers the tunnel and service address saved by the current user, falling back to the global configuration when unset

## v2.2.1

- Fixed human verification never passing on registration and password recovery: the Turnstile widget's action did not match what the endpoints validated (sending an email code, recovering and resetting a password). Only the login page sends an action now
- Security fix: users who have not connected their own Cloudflare OAuth no longer fall back to the administrator's static credentials, so unauthorized users cannot see or operate the administrator's tunnels, domains, DNS records or monitors

## v2.2.0

- Status pages can bind their own custom domain, and requests are routed to the matching public page by Host (status-page host isolation)
- Direct and optimized access for status pages: optimized mode separates the status hostname from the auxiliary origin hostname, the custom hostname's custom origin points at the auxiliary one, and ingress is configured for both
- Saving a custom domain creates the proxied Cloudflare CNAME automatically and copies the service and origin parameters from the panel's tunnel ingress; if that fails the status-page settings are kept and a warning is returned, while the UI keeps offering manual DNS and ingress guidance
- An optimized CNAME can be chosen or typed per monitor, falling back to the global default when empty
- Fixed cross-user access caused by monitor ownership not being persisted, and backfilled ownership on existing monitors
- Restored per-user Telegram long polling and webhook modes: webhook secrets are generated automatically and kept in the backend only, with per-user route verification and account isolation

## v2.1.0

- User profile: a custom display name and avatar (with image upload), binding an email address when there is none and changing it when there is
- Optional human verification with Cloudflare Turnstile, switchable from the admin console along with the site key and secret, covering sign-in, registration and password recovery
- User notification centre: every user configures their own channels (email / their own Telegram bot / both) and event switches, with multiple recipients and login notifications
- Telegram remote control opened to all users: one isolated bot each (long polling) operating only their own tunnels, DNS records and domains; an administrator's old global bot migrates to their personal bot
- Notification and remote-control bot tokens are stored separately, with a one-click reuse button to copy a configured token to the other side (each keeps its own when both are set)
- The Telegram API endpoint became a panel-level setting (a custom reverse proxy), used by both remote control and notifications, which solves networks that cannot reach the official API directly

## v2.0.0

- Multi-user support: email registration (with optional invite and email codes), group permissions, and sessions in the database that survive a restart
- Admin console: users, groups, invite codes and registration policy
- Storage moved to SQLite, importing a legacy `config.json` automatically and keeping it as a backup
- Multiple Cloudflare accounts over OAuth: one account may authorize several Cloudflare accounts and switch at will
- Monitor alerts: email on status transitions only, with several recipients and an alert log
- SMTP configuration with a test send

## v1.16.0

- Service monitoring: HTTP / TCP / ICMP probes, several targets per monitor and latency bars
- Public status pages: shareable without a login, with a custom short path, announcement, brand icon and theme
- Image uploads and a custom favicon

## v1.15.0

- Redesigned frontend with a visual theme system

## v1.14.0

- Live tunnel creation and deletion, plus better route management
- Deleting a route can delete its DNS record too
- Bulk DNS deletion

## v1.13.0

- Bulk DNS edits
- A better zone selector
- Fixed Cloudflare response parsing
