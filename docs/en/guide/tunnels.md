# Tunnels & routing

Tunnels are the core Cloudflare Tunnel object: every domain binding eventually lands on some tunnel's ingress rules.

## Creating a tunnel

Press "new" on the "Tunnels" page and give it a name. Once created, the panel shows the **cloudflared token and run command** — copy it to the target machine and the tunnel is up.

::: tip
The connect command contains a sensitive token. Don't screenshot it or paste it anywhere public.
:::

## Choosing the active tunnel

Binding and configuration actions all revolve around the "currently selected tunnel". With several tunnels, select one in the list first, then go back to the binding page.

## Managing application routes (ingress)

Open a tunnel's detail page to review, edit and delete ingress rules:

- Each rule pairs a **request matcher** with a **service address**
- A new rule takes effect in the Cloudflare configuration immediately
- When deleting a rule you can **delete the matching DNS record too**, so no dangling CNAME is left behind (added in v1.14)

Path-only rules are preserved as they are — entries like `/api -> localhost:9000` keep working.

## Deleting a tunnel

Deleting removes the whole tunnel object from Cloudflare along with all of its ingress rules. DNS records are *not* cleaned up automatically, so if you are taking a hostname offline for good, remove its CNAME on the DNS management page by hand.
