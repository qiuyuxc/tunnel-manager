# Static IP probe

Deploy `ip-check.txt` as a static file on a domain you control. No JavaScript, function, secret, or backend is required. Configure the IP selector with that domain as Host and SNI, `/ip-check.txt` as the path, and `200` as the accepted status. The selector validates the HTTP status, not the response body.

For a new static Pages deployment, upload this directory as the site output. For an existing Vite project such as `tunnel-web`, copy the probe into `public/` and rebuild. Do not deploy without checking the generated output and the project's Worker/Functions configuration.

The included `_routes.json` excludes the probe from Pages Functions. **Do not overwrite an existing routing file:** add `/ip-check.txt` to its `exclude` list while preserving all other rules. Put the resulting file in the deployment output directory. External Workers or reverse proxies can still run before Pages, so check those routes too.

Add the instance-generated DNS TXT record before starting a scan. Keep the record in place and start with one candidate IP. A successful probe only demonstrates reachability for this Host and path. It does not authorize scanning someone else's infrastructure or guarantee another application's behavior.

Reference: [Pages invocation routes](https://developers.cloudflare.com/pages/functions/routing/#functions-invocation-routes) and [static asset pricing](https://developers.cloudflare.com/pages/functions/pricing/#static-asset-requests).
