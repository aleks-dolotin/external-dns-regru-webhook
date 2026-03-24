# curl Examples for ExternalDNS RegRu Adapter

Practical curl commands for every sidecar endpoint. Default sidecar address: `http://localhost:8080`.

## Health & Readiness

### Liveness probe

```bash
curl -s http://localhost:8080/healthz | jq .
```

Expected response (always 200):

```json
{
  "status": "ok",
  "timestamp": "2026-03-24T12:00:00Z"
}
```

### Readiness probe

```bash
curl -s http://localhost:8080/ready | jq .
```

Expected response (200 when ready, 503 when not):

```json
{
  "status": "ok",
  "timestamp": "2026-03-24T12:00:00Z",
  "checks": [
    { "name": "credentials", "status": "ok" }
  ]
}
```

## Prometheus Metrics

```bash
curl -s http://localhost:8080/metrics | head -30
```

Key metrics to look for:

- `regadapter_operations_total{zone,action}` — operation counter
- `regadapter_rate_limited_total{zone}` — rate limit hits
- `regadapter_circuit_state{zone}` — circuit breaker gauge
- `regadapter_queue_depth` — current queue depth

## Event Intake

### Submit a single DNS event

```bash
curl -s -X POST http://localhost:8080/adapter/v1/events \
  -H 'Content-Type: application/json' \
  -d '[
    {
      "kind": "DNSEndpoint",
      "namespace": "default",
      "name": "my-service",
      "dnsName": "app.example.com",
      "recordType": "A",
      "targets": ["10.0.0.1"],
      "recordTTL": 300
    }
  ]' | jq .
```

Expected response (200 — all accepted):

```json
{
  "accepted": 1,
  "errors": 0
}
```

### Submit multiple events (batch)

```bash
curl -s -X POST http://localhost:8080/adapter/v1/events \
  -H 'Content-Type: application/json' \
  -d '[
    {
      "kind": "DNSEndpoint",
      "namespace": "prod",
      "name": "web",
      "dnsName": "web.example.com",
      "recordType": "A",
      "targets": ["10.0.1.1"]
    },
    {
      "kind": "DNSEndpoint",
      "namespace": "prod",
      "name": "api",
      "dnsName": "api.example.com",
      "recordType": "CNAME",
      "targets": ["lb.example.com"]
    }
  ]' | jq .
```

### Partial failure (206)

When some events fail validation, the response uses HTTP 206:

```json
{
  "accepted": 1,
  "errors": 1,
  "error_details": ["op abc123: namespace \"staging\" not allowed for zone \"example.com\""]
}
```

## Diagnostics

```bash
curl -s http://localhost:8080/adapter/v1/diagnostics | jq .
```

Expected response:

```json
{
  "queue_depth": 3,
  "worker_count": 2,
  "last_heartbeat": "2026-03-24T12:05:00Z",
  "backpressure": false,
  "throttled_zones": [],
  "zones": {
    "example.com": {
      "circuit_state": "closed"
    }
  },
  "timestamp": "2026-03-24T12:05:01Z"
}
```

## Admin: Safe-Mode

### Enable safe-mode (no-write)

```bash
curl -s -X POST http://localhost:8080/adapter/v1/admin/safe-mode \
  -H 'Content-Type: application/json' \
  -d '{"enabled": true}' | jq .
```

### Disable safe-mode

```bash
curl -s -X POST http://localhost:8080/adapter/v1/admin/safe-mode \
  -H 'Content-Type: application/json' \
  -d '{"enabled": false}' | jq .
```

## Port-Forwarding from Kubernetes

If the sidecar is running in a pod:

```bash
kubectl port-forward -n reg-adapter deploy/reg-adapter 8080:8080

# Then use any of the above curl commands against localhost:8080
```

## Tips

- Pipe to `jq .` for pretty-printed JSON.
- Use `curl -v` for verbose output including headers.
- Use `curl -w "\n%{http_code}\n"` to print the HTTP status code.
- Use `-o /dev/null -s -w "%{http_code}"` for status-only checks in scripts.
