# AI assistant

Open **AI assistant** from the desktop sidebar or the center plus menu on mobile Web and the native Android app. The production route is `/assistant`; the separate design prototype remains a reference, not a backend implementation.

## Connection settings

Use the settings button in the assistant header to enter an API endpoint, model ID and API key. Choose **Chat Completions (compatible)** or **OpenAI Responses (native)**. Enter a base URL, such as `https://api.openai.com/v1`, or a full `/chat/completions` or `/responses` endpoint; the backend uses the path for the selected format. Existing configurations keep Chat Completions unless an unspecified format has an endpoint ending in `/responses`. Tool support is required to generate executable plans. There is no default paid model.

- Personal mode is the default. Each user, including administrators, has an independent connection.
- An administrator can save a dedicated shared connection and enable site-wide sharing. Other users see availability only, not the shared endpoint or key.
- Disabling sharing restores personal connections. Shared failures never fall back to a personal key.
- Saved keys are not returned. Leave the key blank to retain it; provide a replacement to rotate it. Changing the endpoint requires re-entering the key.
- Keys are encrypted with the application's encryption key and owner-bound encryption contexts. Preserve `APP_ENCRYPTION_KEY` or the database's generated key during backup and restore.

HTTP and HTTPS endpoints support local/private networks, IPv4/IPv6 and custom ports from 1 to 65535. Examples include a local proxy at `http://127.0.0.1:8787/v1`, Ollama's compatible API at `http://localhost:11434/v1`, or a private service at `https://192.168.1.10:8443/v1`. The service and model must still support the selected API format and tool calls, and a key accepted by the service is required.

Requests originate from the backend: `localhost`, `127.0.0.1` and `::1` refer to the computer or container running Tunnel Manager, not the browser or Android device. To reach a host service from a container, use the host address provided by your deployment (such as a configured `host.docker.internal`) or a reachable private address.

HTTP sends API keys and conversations in plaintext; use it only on the local machine or trusted networks, and prefer HTTPS over the public Internet. HTTPS certificate verification remains enabled; self-signed certificates must be trusted by the backend system. URL credentials, query strings, fragments, unspecified, link-local (including `169.254.169.254`), multicast and reserved addresses are rejected, as is the `fd00:ec2::254` metadata address. All resolved DNS addresses are checked, connections use the validated IPs directly, redirects are not followed, and environment proxies are not used.

Allowing local/private endpoints lets signed-in users direct AI requests into the backend's networks; this is not a network isolation boundary. In multi-user deployments, allow personal connection settings only for trusted users or enable an administrator-managed shared endpoint, and use backend firewall rules to restrict destinations that should not be reachable.

## Privacy and scope

Messages and recent conversation history are sent to the configured provider. Do not paste passwords or connector tokens. Resource sharing is unchecked by default; enabling it sends authorized tunnel/zone names and IDs plus names and IDs of the user's own monitors and targets, not Cloudflare credentials or another user's conversations. Resource lists are capped at 100 items each.

History and tasks are stored per user. A shared provider does not share conversations. Limits are 20 conversations per user, 48 messages and 48 tasks per conversation, and 8 proposed tasks per reply. Unsent drafts survive navigation in client memory, not necessarily reloads or application restarts.

## Streaming and reasoning summaries

On Web and Android, Send immediately shows the user message and clears the composer so the next draft can be written. Replies arrive incrementally over SSE. The “AI 思考中” (Thinking) indicator has an arrow to expand progress or a reasoning summary actually returned by the provider. Compatible services can return `reasoning_content`; for Responses, enable the optional reasoning-summary setting to request `reasoning.summary: auto`. It defaults to off and requires provider/model support. Without a summary, only progress is shown.

Summaries are saved with completed replies in the user's history but are excluded from subsequent model input. Responses requests use `store: false`; provider retention policies still apply. See the [OpenAI streaming guide](https://developers.openai.com/api/docs/guides/streaming-responses). Providers that ignore streaming and return ordinary JSON are also supported, with the reply displayed all at once.

Replies and pending tasks are saved only after a complete, validated response. Interrupted replies keep the text already displayed and offer a retry without overwriting a fresh draft. Incomplete turns exist only in client memory and may be lost on reload or app restart. If the connection fails after the server has saved a reply, refresh history before retrying to avoid duplicate proposals. Reverse proxies must allow SSE without response buffering.

## Supported tasks

| Task | Effect after confirmation | Permission |
| --- | --- | --- |
| Create tunnel | Creates a tunnel, without installing or starting a connector | `tunnels` |
| New direct domain binding | Adds an ingress rule and proxied CNAME with automatic TTL, exposing the service publicly | `domain_bind` |
| Create DNS record | Adds A, AAAA, CNAME, TXT, MX, NS, SRV, CAA or PTR records | `dns` |
| Update / delete DNS | Resolves an existing name and type within a zone, shows original values; ambiguous matches require the record ID | `dns` |
| Delete tunnel | Deletes a tunnel without cleaning up DNS; active connections remain subject to Cloudflare restrictions | `tunnels` |
| Delete monitor / target | Deletes an owned project, its targets and history, or one target | `monitors` |
| Update ordinary settings | Site name, description, icon, landing page, registration, invite mode and experimental features | Administrator |
| Update preferred CNAME | Changes the global default or replaces presets; does not rewrite existing DNS | Administrator |
| Create monitor project | Creates an empty private project, without publishing or enabling alerts | `monitors` |
| Add HTTP target | Adds a GET target to an existing project and starts scheduled checks | `monitors` |

Only fixed tools are accepted. The model cannot execute commands, choose arbitrary API routes, read secrets or modify permissions. Execution rechecks authorization. Cloudflare tasks also recheck the connection; changing connections invalidates those plans. Local settings do not require a working Cloudflare connection.

Once required information is clear, the assistant proposes pending tasks without asking for optional parameters that have safe product defaults: monitor checks default to **60 seconds**, with publishing and alerts disabled; new DNS records default to **TTL 1 (automatic)** and **proxy off**. The server fills in and validates these values before saving tasks, so Web and Android confirmation cards show the effective defaults. Explicit valid values are preserved; invalid `0`, `null` or empty strings are not treated as omission (`false` is a valid proxy value, and `0` is a valid MX priority). Missing or ambiguous required information, such as real resource IDs, names, domains, record types and values, target URLs or MX priority, still requires clarification. Resources are never invented, and execution still requires confirmation.

Optimized/SaaS bindings, connector installation, same-batch task dependencies and automatic rollback are not supported. Create the prerequisite resource first, then use its actual ID in a subsequent request. Existing DNS or matching ingress rules block a new direct binding.

SRV uses structured `data.priority`, `weight`, `port`, and `target`, with a `_service._protocol.name` record name; CAA uses `data.flags`, `tag`, and `value`. Other types use `content`. These record-specific fields have no implicit defaults and must be provided. An explicitly empty CAA value is valid, unlike omission. Confirmation cards retain structured data for review before execution.

The assistant uses known information and product defaults to prepare editable proposals instead of asking about every optional detail. Ambiguous resource identities and required values without a valid basis still need clarification. DNS edits keep the zone and record IDs fixed. Delete tasks have no editable configuration; cancel and regenerate to choose another target.

DNS updates preserve omitted fields, including TTL, proxy, MX priority and structured SRV/CAA data, without applying creation defaults. The server resolves the record and saves its original values alongside the proposed changes. Ordinary settings use an explicit field allowlist. Resolved snapshots and merged DNS arguments stay in the user’s tasks on the panel; subsequent model requests include only task titles, IDs and status summaries.

Updates and deletions recheck the snapshot immediately before execution. A changed resource or setting requires a fresh plan and confirmation. Assistant changes to the same record, monitor or settings are serialized, but manual edits and external Cloudflare operations do not share this lock. Checking and writing are not a cross-system atomic transaction; avoid concurrent edits to the same object.

## Confirmation and recovery

Wide web layouts show tasks in the right-hand panel. On phones, narrow tablets and the native Android app, selectable task cards appear directly in the conversation, with pending tasks brought into view after a reply or when restoring a conversation. The pending-task button returns to those cards; typing "confirm" in chat does not execute them. Completed and cancelled tasks retain their status but no longer have checkboxes. Native refreshes preserve selections only for tasks still pending; switching conversations clears the selection. The final confirmation executes only its captured selection and task revisions. If another window edits a task, the old confirmation is rejected until the latest values are reviewed.

Mobile web and native composers start with one line and scroll internally after reaching their height limit, with Send alongside the input. Native input supports newlines, displays up to four lines and accepts up to 8000 characters. Privacy details open on demand, and resource-sharing consent remains off by default. The assistant uses compact bottom navigation; the native app hides it and removes its reserved space while the keyboard is open, restoring the appropriate layout when the keyboard closes or the user leaves the assistant.

Review exact arguments. For creation and update tasks, choose **修改参数** (Edit parameters), adjust the form, then **保存修改** (Save changes). Saving validates and updates the pending task without executing it. Select the saved tasks and confirm execution. Unselected tasks remain pending. Web runs at most two independent requests together; Android processes the selection sequentially. Pending tasks may be paused, resumed or cancelled. In-flight requests cannot be withdrawn.

Completed tasks cannot run again, and duplicate concurrent execution is rejected. A timeout, restart or error may leave an unknown outcome. Inspect the real resources before explicitly confirming a retry; failure does not prove that a change had no effect. Retry only if the change is still needed and has not taken effect; changed original values require a new plan.

Domain binding is not atomic: DNS may succeed before ingress configuration fails. The assistant preserves the complete configuration and rechecks it before the ingress write, but external Cloudflare edits do not share its lock. Avoid editing the same tunnel concurrently. Check both DNS and ingress after a failure; no automatic rollback occurs.

Writes use existing business audit categories. Provider billing, quotas and data retention remain the provider's responsibility. Set provider-side budgets before enabling shared access.

## Verification

Run `go test ./...` in `backend`, `npm test` and `npm run build` in `frontend`, and the Android debug build. `frontend/tests/assistant-browser.mjs` uses an existing Vite server and Chromium CDP, defaulting to ports 8084 and 9223. Override these with `ASSISTANT_TEST_ORIGIN` and `ASSISTANT_TEST_CDP`; `ASSISTANT_TEST_OUTPUT` controls screenshots.

The browser test mocks business APIs and blocks external requests. It checks responsive layouts, themes, immediate sending, incremental replies, expandable summaries, interrupted-reply retries, edited approvals, pause/resume, read-only shared settings, session isolation and key non-persistence. These checks do not call paid models or write real Cloudflare resources. A successful APK build is not a device UI or live integration test.
