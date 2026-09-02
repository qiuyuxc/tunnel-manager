# Domain binding modes

Single and batch domain binding both offer two modes; in batch binding, every group picks its own mode, service address and optimized CNAME.

## Direct mode

You only supply the hostname and the service address. The system does two things:

1. Adds a tunnel ingress rule for the hostname.
2. Creates a proxied CNAME pointing at `<tunnel-id>.cfargotunnel.com`.

No auxiliary hostname is needed and no SaaS custom hostname is created. This suits ordinary sites that don't need an optimized route.

## Optimized mode

Optimized mode uses two independent legs: the public hostname enters an optimized route, while an auxiliary hostname carries traffic back to the tunnel. The system configures:

1. A grey-cloud (DNS-only) CNAME on the public hostname pointing at the optimized CNAME you chose.
2. An orange-cloud (proxied) CNAME on the auxiliary hostname pointing at `<tunnel-id>.cfargotunnel.com`.
3. A SaaS custom hostname in the auxiliary hostname's zone, where the hostname is the public one and the custom origin is the auxiliary one.
4. Tunnel ingress rules for both the public and the auxiliary hostname.

The two hostnames must differ. Use a dedicated auxiliary hostname that carries nothing else, so saving does not overwrite existing DNS or ingress configuration.

The optimized CNAME can be picked from your saved CNAME presets or typed in. Left empty, the global default is used.

Reach for this when you need a custom optimized route.

| Resource | Direct mode | Optimized mode |
| --- | --- | --- |
| Public hostname | Proxied CNAME to the tunnel | DNS-only CNAME to the optimized CNAME |
| Auxiliary hostname | Not needed | Proxied CNAME to the tunnel |
| Custom hostname | Not created | Custom origin points at the auxiliary hostname |
| Ingress | Public hostname | Public and auxiliary hostname |

A public status page's custom domain reuses this same optimized setup, with additional host isolation and manual-check notes — see [Monitoring & status pages](/en/guide/monitors-status#custom-domains).

::: tip Compatibility
For older clients, an API request that omits the `mode` field is still treated as optimized mode.
:::

## Fallback origin

When SaaS custom hostnames are in play, the panel can point the fallback origin at a service address for you in one step.

## Cascading deletes

When deleting a tunnel route you can clean up the matching DNS record at the same time, leaving no dangling CNAME.
