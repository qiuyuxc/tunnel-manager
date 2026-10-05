# DNS records

The Web and Android DNS management pages maintain records independently of domain binding. Every request uses the signed-in user's Cloudflare account, `dns` permission and per-record audit trail; it never borrows another user's credentials.

## Choosing a zone

Switch zones at the top of the page. The selector filters by domain name, so there is no Zone ID to copy.

## Supported record types

| Type | Notes |
| --- | --- |
| A / AAAA | IPv4 / IPv6 address targets |
| CNAME | Alias targets, common for optimized routes and CDNs |
| TXT | Text records: verification, SPF and friends |
| MX | Mail exchange records, with a priority |
| NS | Nameserver hostnames for subdomain delegation; does not change registrar nameservers |
| SRV | Full `_service._protocol.name`, priority, weight, port and target hostname |
| CAA | `flags`, `tag` (such as `issue`, `issuewild`, `iodef`) and `value` |
| PTR | Reverse-lookup target hostname; requires control of the relevant reverse zone |

Only these nine types support creation and editing, not every Cloudflare record type. Unknown types remain read-only, without edit, select or delete controls. Both clients also protect SRV/CAA records with missing, invalid or unsupported `data` fields from editing.

## What you can edit

- TTL: `1` means automatic; otherwise `60–86400` seconds. Omission defaults to `1`; explicit `0` or `null` is invalid.
- Proxy status: available only for A / AAAA / CNAME. Other types are forced to DNS-only. Omission defaults to `false`; `null` is invalid.
- MX priority: an integer from `0–65535`; zero survives editing. A null MX uses target `.` and priority `0`.
- SRV: priority, weight and port are integers from `0–65535`. Target is a hostname or `.`. Structured `data` is preserved on save rather than reconstructed from display text.
- CAA: flags is an integer from `0–255`, tag contains 1–15 ASCII letters/digits, and value is a single-line string of up to 4096 bytes (empty allowed).
- TXT: leading and trailing spaces are preserved. Use single-record editing for multiline text. Hostnames use ASCII/Punycode.

## Bulk creation

Click **批量新增** (bulk create) at the top; selecting existing records is not required.

1. Choose one zone and shared type, name, TTL and applicable proxy/MX-priority settings.
2. Paste one value per line. Supported bulk types are **A / AAAA / TXT / MX / NS / PTR**. CNAME does not support multiple targets at the same name; SRV/CAA require structured fields. These three types remain available through single-record creation.
3. Review original line numbers and validation messages. A batch accepts at most **100 nonempty lines, including duplicates**. Blank lines are ignored. Repeated values and loaded records with the same name/type/value are skipped. IPv6 duplicates are normalized, hostname comparison ignores case and trailing dots, and TXT comparison preserves spaces and case. Commas are not separators. Any invalid line prevents the entire batch from being sent.
4. Confirm to call the existing single-record creation endpoint serially, with per-line results. Shared settings lock after the first submission to keep retries on the original target.
5. Successful lines are removed, and definite failures remain for explicit retry. Successful lines are never automatically resent. This is not a transaction and does not roll back successful records.

A batch stays bound to the login session and target zone used when its dialog opened. Changing sessions stops subsequent requests, as does unloading the Web page, closing the Android dialog, or destroying its host activity. Unattempted rows are marked **not sent**, not unknown. An in-flight request may still complete and is not rolled back; check created records before opening a new batch under the current account. An old batch never continues with the new account's token.

::: warning Unknown does not mean not created
A timeout, disconnect, gateway error or unreadable provider response may occur after creation. Such a line is marked **result unknown; verify before retrying**, and the queue stops immediately. Remaining unattempted rows are marked **not sent**. Only failed, unknown and unattempted rows remain in the input. Check Cloudflare or refresh the records, remove rows that were actually created, then explicitly acknowledge verification before retrying. Closing the dialog clears the local batch input/results without undoing created records.
:::

Local deduplication cannot detect concurrent provider changes. Cloudflare still enforces conflicts, NS delegation counts and account limits. Multiline TXT belongs in the single-record editor.

## Bulk editing and deletion

Select existing records for bulk editing or deletion. SRV/CAA type and structured-field changes must use the single-record editor; bulk TTL changes retain the original `data`. Changing an MX record's TTL retains its priority.

::: warning
A bulk delete cannot be undone. On production domains, double-check your filter before ticking rows.
:::

Field references: [Cloudflare DNS record types](https://developers.cloudflare.com/dns/manage-dns-records/reference/dns-record-types/) and the [current official TypeScript SDK](https://github.com/cloudflare/cloudflare-typescript/blob/main/src/resources/dns/records.ts).
