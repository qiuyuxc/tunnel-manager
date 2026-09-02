# Site branding & global settings

The "Global settings" page manages the panel's own appearance and your everyday presets.

## Site branding

These publicly visible details are customizable:

- Site name and description
- Navigation and login-page icons
- Browser title
- Favicon, uploadable

Uploads go through `POST /api/uploads` and images are stored under `uploads/` in the data directory.

## Landing page

The "enable landing page" switch decides what the site's front door is:

- **On**: the site root shows a public portal first — site name and description, a feature overview and a link to the GitHub repository. The top bar offers light/dark switching, and signed-in visitors can jump straight to the console
- **Off** (default): the previous behaviour — visitors who are not signed in land on the login page

The landing page needs no login, which makes it useful as a public face.

## CNAME presets

Keep a list of optimized CNAME values you use often, so binding is a dropdown choice instead of retyping. Order in the list is the order in the dropdown.

## Global default optimized CNAME

The CNAME used when optimized mode does not name a route of its own. Individual bindings and status pages may override it temporarily without changing the global default.

## Cloudflare connection status

The same page shows which credentials are in use (OAuth or a static token), which account is authorized, and lets you switch — see [Cloudflare OAuth connection](/en/guide/cloudflare-oauth).
