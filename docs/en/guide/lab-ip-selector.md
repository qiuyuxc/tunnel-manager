# IP optimizer lab (experimental)

The IP optimizer lab turns direct probing, latency ranking and DNS updates into one configurable experiment. It is not tied to one CDN: provide your service hostname, TLS SNI and authorized IP ranges, then keep the reachable addresses with the lowest latency.

## Enable the feature

1. Open **Admin console → System settings** as an administrator.
2. Turn on **Experimental features**.
3. Save; **IP Optimizer Lab** appears in the sidebar.

When the switch is off, the navigation entry is hidden and the lab APIs return `404`.

## Probe settings

| Setting | Description |
| --- | --- |
| Host | HTTP `Host` header, usually your service hosted on Cloudflare or another platform |
| SNI | TLS server name; empty means use Host |
| Probe path | Prefer a lightweight, cacheable health-check path |
| Accepted statuses | Comma-separated, for example `200,204` |
| Timeout | 1–15 seconds per connection |
| Workers | 1–256; choose according to your server and network capacity |
| Keep Top N | 1–100; retain only the N lowest-latency IPs |

Targets support three formats, separated by commas, spaces or new lines:

```text
1.2.3.4
1.2.4.0/24
1.2.5.10-1.2.5.20
```

Targets are expanded and de-duplicated. A run accepts at most 65,536 unique IPs.

## Execution and progress

After selecting **Run now**, the page shows:

- Current phase: preparing, probing, updating DNS or completed
- Scan percentage
- Scanned / total
- Matched count

Up to 20 execution records are retained. Each includes scanned and matched counts, top IPs, errors and whether DNS was updated.

## Segment statistics and removal

Results are grouped by your original target text, such as `1.2.4.0/24` or `1.2.5.10-1.2.5.20`.

The **Missed IP segments** card lists every segment with zero matches and totals how many future probes can be avoided. You can:

- **Copy missed segments** to compare or edit manually
- **Remove and save** to remove them from the current configuration automatically

If you edited the target list after the last run, removal only matches segments that are still exactly the same as the latest record, preventing accidental deletion of new entries.

## Automatic Huawei Cloud DNS updates

With **Update Huawei Cloud DNS** enabled, the task writes the lowest-latency Top N IPs to an A record.

| Setting | Description |
| --- | --- |
| Zone name / Zone ID | Either one; a Zone ID skips the lookup |
| A record | Record name to create or update |
| TTL | 60–86,400 seconds |
| API endpoint | Defaults to `https://dns.myhuaweicloud.com`; HTTPS only |
| Access Key / Secret Key | Huawei Cloud API credentials |

The Secret Key is encrypted at rest with `APP_ENCRYPTION_KEY` using AES-GCM. The API only reports whether it is configured and never returns the plaintext.

## Scheduled runs

Enable **Scheduled runs** to execute automatically every 5–1440 minutes. After a service restart, scheduling resumes from the most recent execution time.

::: warning
Only probe services and IP ranges you own or are explicitly authorized to test. For large ranges, use moderate concurrency and a lightweight path, and respect the policies of your origin, CDN and network providers.
:::
