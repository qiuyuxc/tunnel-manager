# DNS records

The "DNS management" page maintains records independently of domain binding — useful when you just want to read and edit resolution directly. (Iterated in v1.13 / v1.14.)

## Choosing a zone

Switch zones at the top of the page. The selector filters by domain name, so there is no Zone ID to copy.

## Supported record types

| Type | Notes |
| --- | --- |
| A / AAAA | IPv4 / IPv6 address targets |
| CNAME | Alias targets, common for optimized routes and CDNs |
| TXT | Text records: verification, SPF and friends |
| MX | Mail exchange records, with a priority |

## What you can edit

- TTL: a number of seconds, or automatic
- Proxy status: Cloudflare's orange cloud on or off (off by default for TXT and MX)
- MX priority: only shown for MX records

## Bulk operations

Select several rows for a **bulk edit** (v1.13) or a **bulk delete** (v1.14). Bulk actions summarise the pending change first and only submit to Cloudflare after you confirm.

::: warning
A bulk delete cannot be undone. On production domains, double-check your filter before ticking rows.
:::
