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


