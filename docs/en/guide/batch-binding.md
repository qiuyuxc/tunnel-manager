# Batch domain binding

The single binding page handles one group of hostnames at a time; the "batch binding" page submits several groups at once, and each group picks its own mode, service address and optimized CNAME. (Since v1.14.)

## How it works

1. Press "add group" to get an empty binding row
2. For each group, choose:
   - **Mode**: direct or optimized, independently per group
   - **Primary hostname** (plus the auxiliary hostname in optimized mode)
   - **Service address**: where this group's traffic actually goes
   - **Optimized CNAME** (optimized mode only): empty uses the global default
3. Submit, and the system works through the groups and reports back

## Results

The batch endpoint reports success or failure per group: one failing group does not stop the others, and the result list spells out each group's error — hostname already in use, zone not found, and so on.

## Relationship to single binding

Both modes behave exactly as they do in [single binding](/en/guide/domain-binding): direct mode creates only the tunnel route plus a proxied CNAME, while optimized mode additionally creates a SaaS custom hostname. At the API level this is `POST /api/domain/bind-batch`.
