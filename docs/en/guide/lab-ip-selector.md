# IP selector lab (experimental)

The lab connects directly to candidate IPs over TLS, sends real HTTP requests with your Host and SNI, filters by status, and ranks results by latency. It supports history, per-segment statistics, scheduling, and optional Huawei Cloud DNS A-record updates. Regular DNS management, preferred domain binding, and the AI assistant are independent of this feature.

## Enable and migrate

1. Back up the database and `APP_ENCRYPTION_KEY` before upgrading. Keep the original key to decrypt existing Huawei credentials.
2. Sign in as an administrator and enable experimental features in the admin panel. The lab appears in Web navigation and the Android More menu.
3. Review restored settings, history, and credentials. Legacy schedules stay disabled without TXT authorization. Neither enabling the feature nor verifying a domain automatically enables scheduling.
4. Save the probe configuration, verify ownership, and start with a small manual scan.

New installations default to disabled. Disabling the feature hides navigation, makes lab APIs return `404`, and cancels running work before further DNS updates. Stop also disables scheduling. A DNS write already accepted by the provider cannot be recalled.

## Deploy a static probe

Use a lightweight probe you own or are explicitly authorized to use. `examples/ip-probe/` contains `ip-check.txt` and a Pages routing example. In an existing Vite project, copy the probe to `public/`, rebuild, and deploy the generated output.

Use `/ip-check.txt` and accepted status `200`. The scanner checks the status, not the response body. Do not use a third-party website or another installation's domain as a shared default.

On Cloudflare Pages, only static requests that **do not invoke Functions** avoid function usage. For a project with Functions, merge the probe into the output directory's `_routes.json` exclusions:

```json
{
  "version": 1,
  "include": ["/*"],
  "exclude": ["/ip-check.txt"]
}
```

Preserve existing routing rules. Also check Workers or proxies in front of the domain. Returning `200` inside a function still invokes it. See [invocation routes](https://developers.cloudflare.com/pages/functions/routing/#functions-invocation-routes) and [static asset pricing](https://developers.cloudflare.com/pages/functions/pricing/#static-asset-requests).

TXT authorization proves domain control, not zero function usage. A successful probe for one Host does not guarantee that the same IP works for another Workers, Pages, or Tunnel application.

## Verify DNS ownership

1. Enter a domain you control. Public suffixes cannot be verified. No probe settings need to be entered or saved first.
2. The verification domain is independent of the probe Host and SNI; they need not share a parent domain. Probe only self-hosted or explicitly authorized targets.
3. With an available Cloudflare account bound, Web and Android show **Fill and verify automatically**. This adds TXT through the current account and checks DNS without saving or changing the probe Host/SNI. The verification domain must belong to that account, and its credentials must permit DNS writes.
4. Without a bound account, a matching zone, or write permission, select **Generate TXT** and add the exact record name and random value at your DNS provider. For verification domain `example.com`, the name is `_tunnel-manager-verify.example.com`. If your DNS console appends the zone name, enter only the corresponding relative name. Manual verification requires no DNS API credentials.
5. If propagation is pending, wait and select **Check TXT** rather than generating another record. A successful API write does not prove ownership verification; the actual DNS check must pass.

The one-click action uses only the current user's selected Cloudflare connection, never another user's account. Administrators without a selected usable connection may use the project's existing static Cloudflare credentials; a failed selected authorization does not trigger an automatic switch to another account. Availability only indicates configured credentials; zone ownership and write permission are checked during the action.

Retries reuse the same valid challenge without duplicating its TXT record or replacing other TXT or A/CNAME records. The action disables scheduling and never starts a scan. Explicitly enable and save scheduling after verification if needed.

Authorization is bound to the instance and administrator. Unverified challenges expire after 24 hours. Generating a new challenge revokes the previous authorization and disables scheduling. Changing Host or SNI preserves verification; changing the operating administrator requires verification again. After successful verification, Web and Android hide the record name and value, showing only the verified domain.

Keep the TXT record. The backend rechecks it before every scan and before automatic DNS writes. Missing records, DNS errors, or an invalid administrator block execution. Resolver caching affects how quickly record removal becomes visible.

A manual TXT check through the API first requests cancellation of any running task; a failed check also disables scheduling. Saving configuration, rotating the challenge, and stopping likewise cancel without waiting for an in-flight DNS lookup to finish. Writes already accepted by the provider cannot be recalled.

## Probe settings and limits

| Setting | Behavior |
| --- | --- |
| Host / SNI | Valid domain names; blank SNI uses Host. TLS certificates are verified |
| Path | Site-local path, no full URL, spaces, or control characters. Redirects are not followed |
| Accepted statuses | For example `200,204`. Responses such as `403` or `418` are rejected unless explicitly allowed |
| Timeout | 1 to 15 seconds per connection |
| Workers | 1 to the administrator-configured ceiling (default 32); shared HTTP requests per second are also configurable (default 10) |
| Top N | 1 to 100, sorted by latency. Rate-limit waiting time is excluded from HTTP response latency |
| Targets | Public single IPs, CIDRs, or IPv4 start-end ranges; at most 65,536 unique candidates per run. Private, loopback, link-local, and reserved addresses are rejected |

Administrators can configure the daily request budget (1 to 10,000,000), shared requests per second (1 to 1,000), and worker ceiling (1 to 256) under **Admin → System settings → IP selector**. Defaults remain 100,000, 10, and 32. This is the panel's own budget, not a Cloudflare Functions quota or a measurement of cloud usage.

Manual and scheduled runs share a persistent instance-wide budget, reset at UTC midnight. Each run reserves its entire candidate count before starting. Failed or cancelled runs do not refund that reservation, and restarting does not reset it. Limit changes apply to the next run; an active run retains its initial limits. Lowering the budget preserves spent reservations, and existing worker settings above a new ceiling are capped during execution. Only one run can execute at a time, starts are at least 60 seconds apart, and each run ends within two hours or at the next UTC midnight.

Clients show authorization, budget, cooldown, progress, and errors. **Save and run** uses persisted settings. **Stop and disable scheduling** cancels work. Status polling preserves your draft fields.

Up to 20 runs are retained with counts, selected IPs, errors, and DNS-update state. Zero-match segments can be copied or removed from the current input; removal matches the original segment text rather than guessing overlapping address ranges.

## Huawei Cloud DNS and scheduling

Automatic DNS updates require a zone name or zone ID, a full A-record name, TTL, Access Key, and Secret Key. The default endpoint is `https://dns.myhuaweicloud.com`; endpoints must use HTTPS, and TTL is 60 to 86,400 seconds. A records require IPv4 candidates; disable DNS updates for IPv6-only scans.

The Secret Key is encrypted using `APP_ENCRYPTION_KEY` and is never returned through the API. Leave it blank to retain the existing secret. Changing the API endpoint requires re-entering the secret. DNS API redirects are not followed.

Updates require a successful nonempty scan, unchanged settings, and a fresh TXT check. Scan failures, cancellation, failed authorization, decryption errors, or DNS API errors are recorded; empty results never replace existing records.

New runs retain their Host, SNI, path, accepted statuses, and up to 100 rejected-IP diagnostics, including when a run fails. Web and Android show the returned HTTP status or connection, TLS, and timeout error. Older history has no diagnostics; run again or test one candidate separately if a large batch exceeds the diagnostic sample limit.

**A rejected candidate is not necessarily an unusable IP.** With `200` configured, a valid TLS connection returning `403` still does not qualify. A `200` response through ordinary DNS does not guarantee identical routing through a candidate IP. Check the intended Host, SNI, and path before removing a range; do not bypass certificate verification or blindly accept error statuses just to obtain a match.

After authorization, explicitly enable and save a schedule from 5 to 1440 minutes. On startup the next run is scheduled after the configured interval; missed runs are not replayed. Saving settings cancels current work. A new ownership challenge or Stop disables scheduling.
