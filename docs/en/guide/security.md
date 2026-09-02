# Security & admin auth

## Administrator authentication

- Passwords are hashed with Argon2id; older SHA-256 hashes are migrated automatically on sign-in
- Sessions last 12 hours
- Ordinary protected endpoints accept either an administrator session or `API_KEY`; the 2FA management endpoints accept **only an administrator session** and cannot be bypassed with `API_KEY`

## Two-factor authentication (2FA)

1. Sign in to the panel as an administrator.
2. Open **Account settings** and choose **Set up two-factor authentication**.
3. Scan the QR code with any TOTP-compatible authenticator, or type the setup key shown on the page.
4. Enter the 6-digit code to finish enrolling.
5. Copy or download the 10 one-time recovery codes shown once, and confirm you have stored them safely.

From then on, signing in needs the password plus a code — or one unused recovery code. Recovery codes are replay-protected. Turning 2FA off requires the current password and either a code or a recovery code.

The TOTP secret is stored AES-GCM encrypted with `APP_ENCRYPTION_KEY`.

## Password reset

If you forget the password, reset it from the CLI:

```bash
# generate a random password
docker compose exec tunnel-manager ./tunnel-manager --reset-password

# set a specific password
docker compose exec tunnel-manager ./tunnel-manager --set-password=NEW_PASSWORD
```

::: warning
A password reset does not bypass 2FA once it is enabled. If the TOTP encryption key is lost, the correct `APP_ENCRYPTION_KEY` has to be restored while the service is stopped.
:::

## Human verification (Cloudflare Turnstile)

Sign-in, registration and password recovery can be put behind Cloudflare Turnstile, optionally:

1. Create a widget in the [Cloudflare Turnstile](https://dash.cloudflare.com/?to=/:account/turnstile) dashboard and add this site's domain to its allowed hostnames (include `localhost` for local testing).
2. Open **Admin console → Settings** and enable the "Human verification" card with your site key and secret key. The secret is only submitted the first time or when you change it — leaving it blank keeps the stored one.
3. Once on, sign-in, registration, sending a verification code and resetting a password all require a successful challenge. Tokens are single-use and refresh after every submission.

The Turnstile secret is stored AES-GCM encrypted with `APP_ENCRYPTION_KEY`, and API responses never return the key itself.

## What `APP_ENCRYPTION_KEY` protects

The key covers all of the following and must be backed up together with the `data/` directory:

- TOTP secrets (your 2FA enrollments)
- Cloudflare OAuth access and refresh tokens

If it is lost or replaced, an instance with 2FA enabled will refuse administrator logins and OAuth connections have to be authorized again.
