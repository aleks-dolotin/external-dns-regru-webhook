# Reg Adapter Runbook — Operational Playbook (Detailed)

This runbook contains step-by-step operator actions for the Reg.ru adapter sidecar and worker system.
Keep this file concise and actionable — include actual commands (macOS `zsh`) with placeholders.

Owner & Contacts
- Primary oncall: <team-oncall>@example.com (PagerDuty: <pagerduty-service-id>)
- Product Owner: <product-owner-email>
- Dev lead / Architect: <dev-lead-email>

Prerequisites
- kubectl with context to target cluster
- `jq` and `curl` available locally
- Admin access to patch ConfigMap/Secrets in the target namespace

Common namespaces and resources (replace placeholders)
- Namespace: <NAMESPACE>
- Sidecar deployment: reg-adapter-sidecar
- Worker deployment: reg-adapter-worker
- ConfigMap: reg-adapter-config
- Secret: reg-credentials
- Admin API (internal): http://reg-adapter-admin.<NAMESPACE>.svc.cluster.local:8080
- Metrics endpoint: http://reg-adapter-metrics.<NAMESPACE>.svc.cluster.local:9090/metrics

Quick diagnostic commands
```bash
# list pods
kubectl -n <NAMESPACE> get pods -l app=reg-adapter -o wide

# view sidecar logs (last 10m)
kubectl -n <NAMESPACE> logs -l app=reg-adapter --since=10m | tail -n 200

# query metrics endpoint
curl -s http://reg-adapter-metrics.<NAMESPACE>.svc.cluster.local:9090/metrics | grep requests_total

# diagnostics via admin API
curl -s http://reg-adapter-admin.<NAMESPACE>.svc.cluster.local:8080/v1/diagnostics | jq .
```

Incident A — Rate-limit surge / many 429 responses

Detection
- Alerts: `rate_limited_total` spike; sudden error increase for adapter to Reg.ru; circuit_state metric OPEN for zone
- Logs: repeated 429 responses in adapter logs

Immediate mitigation (0–10 minutes)
1. Inspect metrics and logs:
```bash
kubectl -n <NAMESPACE> logs -l app=reg-adapter --since=10m | grep "429" | tail -n 50
curl -s http://reg-adapter-metrics.<NAMESPACE>.svc.cluster.local:9090/metrics | egrep "rate_limited_total|requests_total|retries_total"
```
2. Reduce outgoing throughput by lowering worker concurrency or applying temporary rate-limit config:
```bash
# scale down workers
kubectl -n <NAMESPACE> scale deployment/reg-adapter-worker --replicas=1

# or patch configmap with conservative rate values
kubectl -n <NAMESPACE> patch configmap reg-adapter-config --type merge -p '{"data":{"rate_limit_per_zone":"100"}}'
```
3. Pause non-critical CI pipelines that produce heavy event load (coordinate with teams).

Verification (10–30 minutes)
- Confirm `rate_limited_total` stabilizes and `requests_total` outbound rate is under threshold.
- Check logs for decreasing 429 counts.

Recovery (30–120 minutes)
- Gradually restore worker replicas and rate_limit settings while monitoring metrics.
- When stable, resume paused pipelines.

Escalation
- If high error rate persists >30 minutes and SLA breached, page oncall and open an incident with timeline and logs.

Incident B — Authentication failure (invalid credentials / signature errors)

Detection
- Adapter logs show 4xx auth failures or signature error messages; `/ready` returns false for auth checks

Immediate mitigation
1. Inspect current Secret and last rotate timestamp:
```bash
kubectl -n <NAMESPACE> get secret reg-credentials -o yaml
kubectl -n <NAMESPACE> describe secret reg-credentials
```
2. Run a direct test call using the Secret values (do locally; mask outputs):
```bash
# example test (replace placeholders)
USERNAME="$(kubectl -n <NAMESPACE> get secret reg-credentials -o jsonpath='{.data.username}' | base64 --decode)"
PASSWORD="$(kubectl -n <NAMESPACE> get secret reg-credentials -o jsonpath='{.data.password}' | base64 --decode)"
curl -s -X POST 'https://api.reg.ru/api/regru2/zone/get_resource_records' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d "input_format=json&input_data={\"username\":\"${USERNAME}\",\"password\":\"${PASSWORD}\",\"domains\":[{\"dname\":\"example.com\"}]}" | jq .
```
3. If credentials invalid: rotate to known-good credentials and apply secret update (use sealed-secret/KMS if supported):
```bash
kubectl -n <NAMESPACE> create secret generic reg-credentials --from-literal=username=NEWUSER --from-literal=password=NEWPASS --dry-run=client -o yaml | kubectl apply -f -
```
4. For sig-based auth: ensure private key accessible in KMS or mounted secret and run signature test per docs.

Verification
- `/ready` returns true; integration test call to `zone/get_resource_records` succeeds.

Escalation
- If credentials cannot be validated, contact Reg.ru account admin and Security.

Incident C — Reg.ru outage / large-scale 5xx

Detection
- High 5xx rate; retries_total increasing; circuit breakers open; `/ready` may be false

Immediate mitigation
1. Place adapter into safe-mode (no-write) if supported by admin API:
```bash
curl -X POST http://reg-adapter-admin.<NAMESPACE>.svc.cluster.local:8080/v1/admin/safe-mode -d '{"mode":"on","reason":"reg-ru-unavailable"}'
```
2. Scale down workers to reduce retries and avoid amplifying load:
```bash
kubectl -n <NAMESPACE> scale deployment/reg-adapter-worker --replicas=0
```
3. Notify stakeholders and open incident; capture logs and metrics for postmortem.

Recovery
- After Reg.ru recovers: gradually scale workers back and disable safe-mode:
```bash
curl -X POST http://reg-adapter-admin.<NAMESPACE>.svc.cluster.local:8080/v1/admin/safe-mode -d '{"mode":"off"}'
kubectl -n <NAMESPACE> scale deployment/reg-adapter-worker --replicas=3
```

Incident D — Circuit-breaker triggered (per zone)

Detection
- `circuit_state` metric shows OPEN for a zone; error rate high for specific zone

Immediate mitigation
1. Inspect error details for the zone in logs and diagnostics:
```bash
kubectl -n <NAMESPACE> logs -l app=reg-adapter --since=1h | grep '"zone":"example.com"' | tail -n 200
curl -s http://reg-adapter-admin.<NAMESPACE>.svc.cluster.local:8080/v1/diagnostics | jq .zones
```
2. Temporarily disable writes for that zone in ConfigMap if root cause is external or misconfiguration:
```bash
kubectl -n <NAMESPACE> patch configmap reg-adapter-config --type merge -p '{"data":{"zones_to_disable":"example.com"}}'
```
3. If a configuration regression caused the failures, revert recent config or mapping changes.

Verification
- circuit_state transitions to HALF_OPEN and then CLOSED once error rate subsides.

Incident E — Queue backlog and worker lag

Detection
- `queue_depth` high; worker lag metric high; backlog not decreasing

Immediate mitigation
1. Inspect diagnostics:
```bash
curl -s http://reg-adapter-admin.<NAMESPACE>.svc.cluster.local:8080/v1/diagnostics | jq .queue_depth
kubectl -n <NAMESPACE> get deployment reg-adapter-worker -o wide
```
2. Determine cause (transient downstream errors vs sustained load). If transient, reduce worker concurrency to avoid retries storms; if capacity, scale workers up cautiously.
```bash
kubectl -n <NAMESPACE> scale deployment/reg-adapter-worker --replicas=5
# or reduce concurrency via config map
kubectl -n <NAMESPACE> patch configmap reg-adapter-config --type merge -p '{"data":{"worker_concurrency":"2"}}'
```
3. Consider pausing non-critical event producers.

Verification
- queue_depth decreases and worker processing rate normalizes.

Post-incident
- Create an incident report (include correlating_ids, metrics, timeline)
- Add a short postmortem in the repo under `docs/incidents/` with root cause and action items

Appendix: quick commands reference
```bash
# show metrics for a specific metric name
curl -s http://reg-adapter-metrics.<NAMESPACE>.svc.cluster.local:9090/metrics | grep requests_total

# get last 500 log lines for sidecar
kubectl -n <NAMESPACE> logs -l app=reg-adapter --tail=500
```

Notes
- Replace placeholders with real values from your cluster and infra.
- This is a short operational playbook — expand incident-specific runbooks in `docs/incidents/` with postmortems and checklists.

---

## Credential Setup & Minimum Permissions (Least-Privilege)

### Minimum Required Reg.ru API Permissions

The adapter requires **only zone-management permissions**. Do **not** use admin-level credentials.

| Permission / API Function | Required | Purpose |
|---|---|---|
| `zone/get_resource_records` | ✅ Yes | Read DNS records for a zone |
| `zone/update_records` | ✅ Yes | Create, update, delete DNS records (via action_list) |
| `zone/add_txt`, `zone/add_aaaa`, etc. | ❌ No | Not used — adapter uses `update_records` for all mutations |
| `service/get_list` | ❌ No | Not required for DNS operations |
| Domain transfer / registration | ❌ No | Not required |
| Billing / payment operations | ❌ No | Not required |

### Credential Types

| Auth Mode | Env Variables | Notes |
|---|---|---|
| **Token** (default) | `REGU_USERNAME`, `REGU_PASSWORD` | Simple username + API password |
| **RSA Signature** | `REGU_USERNAME`, `REGU_RSA_PRIVATE_KEY` or `REGU_RSA_PRIVATE_KEY_PATH` | Set `AUTH_DRIVER=rsa` |

### Setting Up Credentials in Kubernetes

1. Create a dedicated Reg.ru API user with **only** `zone/get_resource_records` and `zone/update_records` permissions.

2. Create the Kubernetes Secret:
```bash
kubectl -n <NAMESPACE> create secret generic reg-credentials \
  --from-literal=username=<REGU_USERNAME> \
  --from-literal=password=<REGU_PASSWORD>
```

3. For RSA auth, mount the private key:
```bash
kubectl -n <NAMESPACE> create secret generic reg-credentials \
  --from-literal=username=<REGU_USERNAME> \
  --from-file=rsa-key=<PATH_TO_PRIVATE_KEY_PEM>
```

### Credential Rotation

The adapter supports **live rotation without pod restart** via `ReloadableDriver`:
- Credentials are mounted as files from the K8s Secret volume at `REGU_CREDENTIALS_PATH`
- The kubelet automatically updates mounted secret files when the Secret object changes (sync period ~60s by default)
- The adapter's `ReloadableDriver` re-reads files on every tick (default: **30 seconds**)
- During rotation, old credentials continue serving requests until new ones are verified
- **Zero downtime** — no requests fail during rotation

> **Important:** This works because credentials are mounted as volume files, NOT via `secretKeyRef` env vars. Env vars are immutable after pod start — only file mounts support live rotation.

Configurable via environment variable:
- `REGU_ROTATION_INTERVAL_SEC` — interval in seconds between re-reads (default: 30)

Rotation procedure:
```bash
# 1. Update the secret with new credentials
kubectl -n <NAMESPACE> create secret generic reg-credentials \
  --from-literal=username=<NEW_USERNAME> \
  --from-literal=password=<NEW_PASSWORD> \
  --dry-run=client -o yaml | kubectl apply -f -

# 2. Wait for kubelet sync (~60s) + adapter reload (~30s) = ~90s max
# 3. Verify /ready returns 200
curl -s http://reg-adapter.<NAMESPACE>.svc.cluster.local:8080/ready

# 4. (Optional) Check adapter logs for "credentials reloaded successfully"
kubectl -n <NAMESPACE> logs -l app.kubernetes.io/name=reg-adapter --since=2m | grep "reloaded"
```

Fallback (if file mount is not configured):
- Set `REGU_USERNAME` and `REGU_PASSWORD` as env vars (requires pod restart for rotation)
- The adapter reads env vars as a fallback when `REGU_CREDENTIALS_PATH` is not set

### Permission Error Handling

The adapter classifies permission errors clearly:
- **HTTP 403** → `ErrPermissionDenied` — credentials lack required zone-management permissions
- **HTTP 401** → `ErrAuthenticationFailed` — credentials are invalid or expired
- **API error `ACCESS_DENIED_TO_OBJECT`** → `ErrPermissionDenied` — user lacks access to the specific zone
- **API error `INVALID_AUTH`** → `ErrAuthenticationFailed` — authentication rejected by Reg.ru

If you see permission errors in logs, verify the API user has the minimum permissions listed above.

---

## Operation Tracing (Story 6.4)

Every DNS operation carries a unique `correlating_id` (UUID v4) from intake to completion. Use it to reconstruct the full timeline of any operation.

### Pipeline stages traced

1. **Event received** → `handleEvents` logs `event enqueued` with `correlating_id`
2. **Queue dequeue** → worker logs `operation dequeued` with `correlating_id`
3. **Rate-limit check** → worker logs `rate-limited` if throttled
4. **Circuit-breaker check** → worker logs `rejected by circuit breaker` if open
5. **Dispatch** → worker logs `dispatching to adapter` before API call
6. **Retries** → worker logs each retry attempt with `correlating_id`
7. **Result** → worker logs `operation succeeded` or `operation FAILED`
8. **Audit event** → audit logger emits `audit_event` with `correlating_id`

### kubectl + jq: trace a single operation

```bash
# Find all log entries for a specific correlating_id
kubectl -n <NAMESPACE> logs -l app.kubernetes.io/name=reg-adapter --since=1h \
  | jq -s 'sort_by(.timestamp) | .[] | select(.correlating_id == "<CORRELATING_ID>")'
```

### kubectl + jq: find operations for a zone

```bash
kubectl -n <NAMESPACE> logs -l app.kubernetes.io/name=reg-adapter --since=1h \
  | jq 'select(.zone == "example.com")'
```

### Loki / Grafana query

```logql
{app="reg-adapter"} | json | correlating_id = "<CORRELATING_ID>"
```

Or search by zone:
```logql
{app="reg-adapter"} | json | zone = "example.com" | line_format "{{.timestamp}} {{.level}} {{.msg}} cid={{.correlating_id}}"
```

### Prometheus exemplar query

Exemplars on `regru_api_request_duration_seconds_v2` carry `correlating_id`. In Grafana:

1. Open a panel with `regru_api_request_duration_seconds_v2` histogram.
2. Enable **Exemplars** toggle in the query options.
3. Click any exemplar dot to see the `correlating_id` attached.

### Timeline reconstruction steps

1. Obtain the `correlating_id` from an alert, audit log, or metrics exemplar.
2. Run the kubectl+jq query above to get all log entries for that ID.
3. Sort by `timestamp` to form a timeline.
4. Check `result` field for success/failure outcome.
5. If failed, check `error_detail` in audit events and `error` in log entries.

### Audit log queries

Audit events are written to stdout as structured JSON with `audit=true` marker.
They are collected by the cluster log aggregation system (Loki/ELK/CloudWatch).

```bash
# Loki query: all audit events for a specific correlating_id
{app="reg-adapter"} | json | audit = "true" | correlating_id = "<ID>"

# Loki query: all failures in the last 24h
{app="reg-adapter"} | json | audit = "true" | result = "failure"

# kubectl fallback: search audit events by correlating_id
kubectl -n <NAMESPACE> logs -l app=reg-adapter --since=24h | \
  jq -r 'select(.audit == true and .correlating_id == "<ID>")'
```

### Configuration: Audit

| Env var | Default | Description |
|---------|---------|-------------|
| `LOG_LEVEL` | info | Structured log level (debug, info, warn, error) |

> **Note:** Audit retention (90 days) is managed by the cluster log aggregation platform (Loki/ELK retention policies), not by the application.

