# FAQ

## Deployment & sign-in

### Where do I find the initial password?

It is generated on first start and printed to the container logs. The startup banner is in Chinese, so grep for the password label:

```bash
docker compose logs | grep 密
```

You can also set `ADMIN_PASSWORD` before the first start to choose it yourself.

### I forgot the password

Reset it from the host with the CLI:

```bash
docker compose exec tunnel-manager ./tunnel-manager --reset-password              # generate a random one
docker compose exec tunnel-manager ./tunnel-manager --set-password=NEW_PASSWORD   # set a specific one
```

Resetting the password **does not bypass 2FA**: an account with two-factor enabled still needs its one-time code to sign in.

### I lost my phone (or the TOTP secret) while 2FA was on

Use one of the one-time recovery codes on the login page first — ten of them are shown when you finish enrolling.

If the `APP_ENCRYPTION_KEY` itself is gone too, there are only two realistic ways out: **recover a backup of the original key**, or accept re-initializing with a brand-new key (wipe the data directory, follow [Quick start](/en/guide/getting-started) again, then set up 2FA and the Cloudflare connection from scratch).

That is a deliberate trade-off: the program refuses to unbind 2FA without the correct key precisely so that nobody who merely obtains your data files can silently switch off your second factor. The price is that a permanently lost key has no self-service recovery path.

## Cloudflare connection

### OAuth callback fails / redirect_uri mismatch

Cloudflare requires the callback URL to match the registered value **exactly**. Check that `CF_OAUTH_REDIRECT_URI` in `.env` is identical to what you configured under OAuth clients — scheme, port and the `/api/cloudflare/oauth/callback` path included. Behind a reverse proxy, make sure `X-Forwarded-Proto` and `X-Forwarded-Host` are passed through. See [OAuth connection](/en/guide/cloudflare-oauth).

### Can I skip OAuth entirely?

Yes. Supply `CF_API_TOKEN` and `CF_ACCOUNT_ID` and every feature works. The only differences are that a static token cannot refresh itself, and your permissions are whatever the token carries.

## Binding & DNS

### What is the `<tunnel-id>.cfargotunnel.com` CNAME that direct mode creates?

That is Cloudflare's own convention for reaching a tunnel: a proxied CNAME pointing at the tunnel ID is what lets traffic arrive at your `cloudflared`. Direct mode creates it for you and it needs no manual care.

### Why does deleting a route offer to delete DNS too?

Binding produces two pieces of configuration: an ingress rule and a DNS record. Removing only the route would leave the DNS record dangling, so the panel offers to clean both up (since v1.14). Tick it when you want it.

### An old script calls the binding API without `mode` — what happens?

It is treated as **optimized mode**, which keeps the historical behaviour. New integrations should pass `mode` explicitly to get deterministic results.

## Other

### The Telegram bot does not answer

Check that the bot token is correct, that the chat ID is in the administrator's Telegram ID allowlist, and look at "Bot status" in the panel. In group chats, commands accept an `@botname` suffix to disambiguate.

### Should API calls use a session or an API key?

For automation, prefer the `X-API-Key` header (or `?api_key=` in the URL). Two caveats: the whole mechanism is disabled when the `API_KEY` environment variable is unset, and the 2FA endpoints never accept an API key — those require an administrator session.
