# Email service & alerts

## SMTP settings (Admin console → Email service)

| Field | Notes |
| --- | --- |
| Host | For example `smtp.qq.com` or `smtp.gmail.com` |
| Port | `465` (implicit TLS) or `587` (STARTTLS) |
| Encryption | Just tick "encrypted" — the program adapts to 465 implicit TLS and 587 STARTTLS on its own |
| Username / password | The mailbox and its app password (QQ, 163 and similar providers require an app password, not the login password) |
| From address | The same address you authenticate with is fine |

Fill it in, then press "Send a test email" to confirm the path works. SMTP is used for registration codes, password resets and monitor alerts, and it is also the sending channel for the email option in the [user notification centre](/en/guide/multi-user#user-notifications).

## Configuring monitor alerts

Enable it under "Alert settings" on a monitor's detail page:

1. Turn on "Email alerts on status change"
2. Enter the recipients (comma-separated; leave blank to use your registered address)

Alerts are only sent when the status **changes**: one message when a service goes from healthy to failing, another when it recovers. A service that stays down does not keep mailing you. Every notification is recorded in the alert list, and failures note the reason.

Alert emails are HTML cards with a plain-text alternative attached.

## Encryption key

SMTP passwords, Cloudflare tokens and 2FA secrets are encrypted with `APP_ENCRYPTION_KEY`. It is resolved in this order: environment variable → the key stored from the admin console → generated on first start and saved to the database (the admin console shows which source is in use and lets you replace it).

Changing the key requires a restart and invalidates everything already encrypted — you will need to authorize Cloudflare again, re-enroll 2FA and re-enter the SMTP password.
