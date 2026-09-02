# Cloudflare OAuth connection

Tunnel Manager supports Cloudflare's OAuth 2.0 authorization-code flow with S256 PKCE: once an administrator authorizes it, access tokens are obtained and refreshed automatically, the account can be chosen from a list, and no API token has to be copied by hand.

## Setup

1. In the Cloudflare dashboard, go to **Manage Account → OAuth clients** and create a client.
2. Choose a server-side web application, enable `authorization_code` and `refresh_token`, set the token authentication method to `client_secret_basic` and the response type to `code`.
3. Register the callback URL: `https://your-panel-domain/api/cloudflare/oauth/callback`. In production, serve the panel over HTTPS and set `CF_OAUTH_REDIRECT_URI` explicitly.
4. Grant the client the permissions this project needs: Account Settings Read, Cloudflare Tunnel Edit, Zone Read, DNS Edit, SSL and Certificates Edit. Cloudflare handles `offline_access` itself based on the grant types.
5. Put the client ID, client secret and callback URL into `.env`, restart the service, then press "Connect Cloudflare" under **Global settings → Cloudflare connection**.
6. The authorization page lets you pick which accounts this app may access. Afterwards the panel reads the account list, and with several accounts you can switch between them at any time.

## How credentials are stored

- Access and refresh tokens are AES-GCM encrypted with `APP_ENCRYPTION_KEY` before being written to storage.
- The client secret stays in the server's environment and is never sent to the browser.
- Without OAuth you can still use the static `CF_API_TOKEN` and `CF_ACCOUNT_ID`, but they will not refresh themselves.

## The callback URL must match exactly

Cloudflare requires an exact match for the OAuth client's callback URL. Behind a reverse proxy, pass `X-Forwarded-Proto` and `X-Forwarded-Host` through, or set `CF_OAUTH_REDIRECT_URI` in `.env` directly.

To revoke access, delete the OAuth connection in global settings; the service falls back to static-token mode.

## Multiple accounts (v2.0.0+)

One panel can be authorized against several Cloudflare accounts: each "add account authorization" on the account page creates its own connection, earlier connections keep working, and you can switch the active account between connection cards whenever you like.

## Configuring the client from the console (no environment variables)

Environment variables are optional: under Admin console → Accounts & security → Cloudflare authorization you can enter the client ID, client secret, callback URL and scopes directly. Saved values take precedence over the environment (a blank secret keeps the stored one; saving with every field blank resets it).

## Permissions the client needs

When creating or editing the OAuth client in the Cloudflare dashboard, these permissions are mandatory — without them authorization yields a zero-permission token that cannot call anything:

- Account Settings : Read
- Cloudflare Tunnel : Edit
- Zone : Read
- DNS : Edit
- SSL and Certificates : Edit
