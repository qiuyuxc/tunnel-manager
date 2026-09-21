---
layout: home

hero:
  name: Tunnel Manager
  text: A web control panel for Cloudflare Tunnel
  tagline: Manage tunnels, bind domains, configure DNS with optimized CNAMEs and fallback origins, probe your services and publish a shareable status page — with Telegram bot remote control, administrator two-factor authentication and an experimental IP optimizer.
  actions:
    - theme: brand
      text: Quick start
      link: /en/guide/getting-started
    - theme: alt
      text: API reference
      link: /en/api/endpoints
    - theme: alt
      text: GitHub
      link: https://github.com/qiuyuxc/tunnel-manager

features:
  - title: Install wizard & two databases
    details: The first visit to the panel runs an install wizard — pick a single-file SQLite store or a PostgreSQL server, test the connection, then choose the administrator account; the password never reaches the logs. Environment variables can own the storage instead, so containers need no changes.
    link: /en/guide/getting-started
    linkText: Start deploying
  - title: Tunnel management
    details: Create, delete and list Cloudflare Tunnels, review and edit application routes; a freshly created tunnel shows its cloudflared token and run command right away.
    link: /en/guide/tunnels
    linkText: Manage tunnels & routes
  - title: Two domain-binding modes
    details: Direct mode only configures the tunnel route and a proxied CNAME; optimized mode runs the full optimized-CNAME and SaaS custom hostname flow, and supports batch binding.
    link: /en/guide/domain-binding
    linkText: Compare the modes
  - title: DNS management
    details: Query and manage A, AAAA, CNAME, TXT and MX records per zone, with TTL, proxy status, MX priority and multi-select bulk operations.
    link: /en/guide/dns-management
    linkText: Open DNS management
  - title: IP optimizer lab
    details: Experimentally probe authorized IP ranges with a custom Host and SNI, filter by accepted status codes, watch live progress and per-segment hit statistics, remove missed segments or update Huawei Cloud DNS.
    link: /en/guide/lab-ip-selector
    linkText: Open the lab
  - title: Service monitoring
    details: HTTP (GET/POST), TCP and ICMP probes, several targets per monitor, and a dashboard with seven days of latency bars.
    link: /en/guide/monitors-status
    linkText: Set up monitoring
  - title: Public status page
    details: Share results without a login — short paths, custom domains, direct-tunnel or optimized CNAME access; a custom domain only ever exposes its own status page.
    link: /en/guide/monitors-status
    linkText: Publish a status page
  - title: Security & remote control
    details: Argon2id password hashing, standard TOTP two-factor auth and one-time recovery codes; run tunnel, binding and DNS operations from a Telegram bot.
    link: /en/guide/security
    linkText: Security settings
  - title: Android app
    details: A native client for the same API — overview, monitoring, tunnels, DNS and the IP optimizer lab are native screens, the login page embeds human verification, and monitor changes arrive as system notifications.
    link: /en/guide/android-app
    linkText: Install & use
---
