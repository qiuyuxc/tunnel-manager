# Android app

A native Android client for the same panel. The backend is still the same REST API — the app just puts the console on your phone, and the web UI is unaffected.

The app is currently built from source (`android/` in the repository); no APK ships with GitHub Releases yet.

## What it does

- Overview, monitoring, tunnels, domain binding, DNS, the IP optimizer lab, the Telegram bot, notifications, the admin console, global settings, account and About are all native screens
- On narrow screens the app uses a bottom tab bar (Overview / Monitoring / DNS / More) plus half-sheet panels, so long forms no longer squeeze sideways
- The login screen is native, with Cloudflare Turnstile embedded in the same page — no bounce to a browser at any point
- Monitor state changes reach the phone through the system notification channel, with no web page kept open
- Passkeys: sign in without a password, use one for the second factor, and bind / rename / remove credentials or turn password sign-in off from the account page
- The admin console gains an audit tab that filters by account, operation type and time range, with paging; the filters use a bottom sheet rather than a long row of pickers
- The overview chart drills down: tapping a column lists that day's outage windows and failure reasons, and a row jumps straight to the monitor
- The admin console's system settings gain sign-in protection: rate-limit budgets, the window, the familiar-network multiplier and the lockout notification switch
- Light and dark follow the app's own theme preference, sharing one palette with the web UI

## Signing in

The panel has to be installed first: run the install wizard in a browser and create the administrator account, and only then can the app sign in.

1. Open the app and enter your panel address (for example `https://tunnel.example.com`, with a port if you use one).
2. Enter your account and password; if the panel enables human verification, the widget appears right below the login form.
3. After login you land in the console; sign out from the avatar menu in the top bar to switch accounts.

If the verification widget cannot load — say the panel's network cannot reach Cloudflare — tap **改用网页版登录** to fall back to the built-in WebView.

## Passkeys

Available on Android 9 and newer. **使用通行密钥登录** on the sign-in screen hands the panel's WebAuthn options straight to the system credential manager, so a fingerprint, face unlock or screen lock completes the sign-in; leaving the account field empty offers every passkey bound to the device. Accounts with two-factor enabled also get **使用通行密钥验证** after the password step.

The passkey card on the account page binds new credentials (the current password is required), renames and removes them, and switches password sign-in off for that account. Binding, removal and the switch are all written to the audit trail.

The app goes through Android's Digital Asset Links check, which asks two things of the panel:

1. It is reached over HTTPS and its relying party id matches the domain.
2. It serves `https://<domain>/.well-known/assetlinks.json` containing the app's package name and the SHA-256 fingerprint of its signing certificate.

The backend serves that document; maintain the package name and fingerprints under "system settings → passkeys" in the admin console (the shared debug keystore's hash is prefilled — after switching to your own release signing key, read the new hash with `keytool -list -v -keystore <keystore> | grep SHA256` or `apksigner verify --print-certs <apk>`). A device without Google Play services cannot use passkeys; the app says so and keeps the password form.

## Notifications

The first time you open **Notifications**, grant the system notification permission. After that the app polls `/api/alerts` at a low frequency in the background and turns monitor state changes into system notifications. Revoking the permission simply returns the app to silent mode; email alerts on the web side are unaffected.

## Theme

The app's theme preference is independent of the system dark mode: switch **light / dark** from the bottom of the sidebar or the avatar menu. The launch splash follows the same preference, choosing the day or night artwork.

## Building

You need JDK 17+, the Android SDK (`platforms;android-34` and `build-tools`) and Gradle:

```bash
cd android
./build.sh            # debug APK -> app/build/outputs/apk/debug/app-debug.apk
./build.sh install    # build, then adb install -r onto a connected device
```

Signing reuses the local debug keystore (default `~/.android/debug.keystore`, overridable with `ANDROID_DEBUG_KEYSTORE`), so reinstalling never conflicts over a different signature.

Android 8.0 (API 26) and newer is supported.
