# Monitoring & status pages

Add probe targets to a monitor and the system checks them on your interval and summarises the results on the dashboard; switch on "public page" to publish those results as a status page that needs no login. (Added in v1.16.)

## Probe types

| Type | Notes |
| --- | --- |
| HTTP | GET or POST |
| TCP | Port reachability |
| ICMP | Ping |

- A monitor can carry several targets, each editable later, each with its own "link out" switch.
- Changing a target's **address**, **type** or **method** clears its history and starts the statistics over.
- The dashboard shows a 24-hour latency bar chart.

## Public status page

### Where it lives

- **Custom short path**: enter 1–32 characters (lowercase letters, digits, `-` or `_`) under "public page settings" and it is served at `/status/your-name`.
- **System token**: with no short path, the generated token is the address.

Both links keep working, and visitors need no account to read the results.

### Custom domains

Enter a hostname such as `status.example.com` under "public page settings". Afterwards that hostname's root path shows this monitor's public page, and the address bar keeps your domain.

One hostname binds to exactly one monitor. Through that hostname only this status page, the public data endpoint and the static assets the page needs are reachable; the admin console, the login endpoint, every other API and every other status page return 404. Turn the public page off and the hostname returns 404 as well.

Hostnames are globally unique and cannot equal the panel's own domain, which administrators maintain under "Global settings".

#### What automatic configuration needs

Automatic configuration uses the monitor owner's active Cloudflare connection and selected tunnel, so all of the following must hold:

1. An administrator has filled in the panel domain.
2. The current user has connected Cloudflare and selected the tunnel the panel runs behind.
3. The tunnel's ingress contains a rule matching the panel domain exactly, with no `path` set.
4. The status hostname — and, in optimized mode, the auxiliary origin hostname — belong to zones the current Cloudflare connection can manage.

The panel copies the service address and origin parameters from the panel domain's ingress rule, but drops `httpHostHeader`. Status pages identify their monitor from the visitor's original `Host`, so a fixed origin host would stop requests from matching the right page.

#### Direct vs optimized

| Cloudflare resource | Direct | Optimized |
| --- | --- | --- |
| Status hostname DNS | Proxied `CNAME` to `<tunnel-id>.cfargotunnel.com` | DNS-only `CNAME` to the optimized CNAME you chose |
| Auxiliary origin hostname DNS | Not needed | Proxied `CNAME` to `<tunnel-id>.cfargotunnel.com` |
| Custom hostname | Not created; a stale record of the same name is cleaned up on save | Hostname is the status hostname, custom origin is the auxiliary hostname |
| Tunnel ingress | Status hostname added | Status and auxiliary hostname added |

Saving creates or updates the matching DNS record, custom hostname and ingress rule. Don't use these hostnames for anything else, or the existing configuration will be overwritten.

**Direct mode** suits status pages that need no optimized route: the hostname goes through Cloudflare's proxy straight into the tunnel, with no auxiliary hostname.

**Optimized mode** splits the visitor path from the origin path:

```text
status hostname   --DNS-only CNAME--> optimized CNAME
status hostname   --custom hostname--> custom origin (auxiliary hostname)
auxiliary hostname --proxied CNAME--> <tunnel-id>.cfargotunnel.com
```

The status hostname is where visitors arrive; the auxiliary hostname carries Cloudflare SaaS origin requests into the tunnel. They must differ, the auxiliary one should be a dedicated hostname carrying nothing else, and it must not be the panel domain.

"Optimized CNAME for this page" can come from your presets or be typed in. Left empty, the global default from global settings is used; with neither configured the status page settings still save, but automatic configuration returns a warning.

#### When automatic configuration fails

A failed Cloudflare call never rolls back status page settings you already saved. The response carries `domain_warning`, and the page shows the reason plus three copyable checklists: the DNS CNAME, the SaaS custom hostname and the cloudflared ingress rule.

Common causes:

- No panel domain in global settings.
- The current user has no active Cloudflare connection, or no tunnel selected.
- No ingress rule in the tunnel matches the panel domain.
- The status or auxiliary hostname is not in a zone this connection can manage.
- Both the per-page and the global optimized CNAME are empty.

Fix the cause and save the public page settings again — the panel retries automatic configuration.

::: warning Preserve the original Host when configuring by hand
cloudflared matches ingress by `Host`. When adding a status page rule manually, keep the service address the same as the panel domain's rule but do not set `httpHostHeader`. Put the new rule before the catch-all at the end, or requests will fall through to the default `http_status:404`.

If the panel is exposed through an A record or nginx instead, automatic configuration does not apply: configure DNS, HTTPS and the reverse proxy yourself, and make sure the original Host survives forwarding.
:::

::: tip HTTP redirects and certificates
Whether HTTP redirects to HTTPS is decided by Cloudflare's zone and custom hostname settings. A Cloudflare `301 Moved Permanently` to HTTPS usually means an edge-side HTTPS redirect is on. After a custom hostname is created for the first time in optimized mode, give the certificate and hostname status time to become active.
:::

::: warning Clean up external resources
Clearing a custom domain or deleting a monitor does not delete the Cloudflare DNS record, custom hostname or tunnel ingress rule. After retiring a hostname, remove those resources in Cloudflare so the old name stops pointing at your panel.
:::

### Page customization

| Item | Notes |
| --- | --- |
| Title | Custom status page title |
| Announcement bar | For maintenance notices and the like |
| Brand icon | Upload PNG / JPG / GIF / WebP (up to 4 MB), or give a URL |
| Theme | Two palettes: tech blue and warm gold |

A visitor's light/dark choice applies to the status page only and never changes the admin panel's colours.

### Linking out

Targets with "link" ticked are clickable on the public page.
