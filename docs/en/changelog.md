# Changelog

Compiled from the repository's release commits; older details are on [GitHub Releases](https://github.com/qiuyuxc/tunnel-manager/releases).

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
