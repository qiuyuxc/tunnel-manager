# Examples

Every example assumes the service runs at `http://localhost:8080`.

## Authentication

| Method | Where it works | How |
| --- | --- | --- |
| User session | Every protected endpoint | Send the `X-Auth-Token` header after signing in |
| API key | Ordinary protected endpoints (not 2FA management); equivalent to administrator rights | The `X-API-Key` header, or an `?api_key=` URL parameter |

::: warning
With no `API_KEY` environment variable set, the API-key method is disabled entirely and only sessions work. The 2FA management endpoints never accept an API key.
:::

## Registering and signing in for a session

```bash
# 1. read the registration policy (it decides what the form needs)
curl http://localhost:8080/api/auth/config

# 2. register (when sign-up is open; invite and email codes follow the policy)
curl -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username": "demo", "email": "demo@example.com", "password": "secret123"}'

# 3. or sign in with an existing account (account accepts an email or a username)
curl -X POST http://localhost:8080/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"account": "admin", "password": "your-password"}'

# the token in the JSON response is the session token; send it as a header:
#   X-Auth-Token: <token>
# accounts with 2FA return a challenge_token first — exchange it at /api/admin/login/2fa
```

## Read-only calls with an API key

```bash
# health check (no auth)
curl http://localhost:8080/api/health

# read the current configuration
curl -H 'X-API-Key: YOUR_KEY' http://localhost:8080/api/config

# the same thing as a URL parameter
curl 'http://localhost:8080/api/config?api_key=YOUR_KEY'

# list tunnels
curl -H 'X-API-Key: YOUR_KEY' http://localhost:8080/api/tunnels
```

## Running one probe now

```bash
curl -X POST -H 'X-API-Key: YOUR_KEY' \
  http://localhost:8080/api/monitors/<monitorID>/check
```

## Reading alert history

```bash
curl -H 'X-Auth-Token: <token>' \
  http://localhost:8080/api/monitors/<monitorID>/alerts
```

## Configuring an optimized status-page domain

```bash
curl -X PUT http://localhost:8080/api/monitors/<monitorID> \
  -H 'X-Auth-Token: <token>' \
  -H 'Content-Type: application/json' \
  -d '{
    "public_domain": "status.example.com",
    "public_domain_mode": "preferred",
    "public_aux_domain": "status-origin.example.com",
    "public_preferred_cname": "preferred.example.net"
  }'
```

An empty `public_preferred_cname` falls back to the global default. A `domain_warning` in the response means the fields were saved but Cloudflare configuration did not complete.

## Fetching public status data

No authentication at all; the token may be the system token or your custom short path:

```bash
curl http://localhost:8080/api/public/status/<token>
```

::: tip
Request bodies for write operations match what the panel's frontend submits; when in doubt, `GET` the resource first to see its current shape. The full list lives in [Endpoints](/en/api/endpoints).
:::
