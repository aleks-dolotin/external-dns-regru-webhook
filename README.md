# external-dns-regru-webhook

Kubernetes sidecar adapter that manages DNS records in [Reg.ru](https://www.reg.ru) via their API v2. Designed to work alongside [ExternalDNS](https://github.com/kubernetes-sigs/external-dns) as a webhook provider.

## Features

- **CRUD** for A, AAAA, CNAME, TXT records via client-accessible Reg.ru API endpoints
- **Per-zone rate limiting** with token-bucket and Retry-After header support
- **Per-zone circuit breaker** to prevent cascading failures
- **Per-namespace isolation** — mapping rules scoped to namespaces
- **Per-namespace quotas** with hourly fixed-window counters
- **Multi-tenant metrics** — Prometheus counters with `namespace` label
- **Force-resync** API for drift repair
- **Replay failed operations** — dead-letter list with replay endpoint
- **Safe-mode** toggle to suppress writes during outages
- **Structured logging** with correlating IDs for end-to-end tracing
- **Audit trail** via structured log events
- **Hot-reload** — config and credentials update without restart

## Quick Start

### Prerequisites

- Go 1.25+
- Reg.ru account with API access enabled
- A domain managed via Reg.ru DNS

### Build

```bash
make build        # → bin/sidecar
```

### Run locally

```bash
export REGU_USERNAME='your-login'
export REGU_PASSWORD='your-api-password'
export REGADAPTER_MAPPINGS_PATH=./examples/mappings.yaml

./bin/sidecar
# Listening on :8080
# Endpoints: /healthz, /ready, /metrics, /adapter/v1/events
```

### Docker

```bash
# Build
docker build -t reg-adapter:latest --build-arg VERSION=$(git describe --tags --always) .

# Run
docker run -p 8080:8080 \
  -e REGU_USERNAME='your-login' \
  -e REGU_PASSWORD='your-api-password' \
  -v ./mappings.yaml:/etc/reg-adapter/mappings.yaml:ro \
  reg-adapter:latest
```

### Kubernetes (Helm)

```bash
# Install
helm install reg-adapter charts/reg-adapter \
  --namespace reg-adapter --create-namespace \
  --set image.repository=ghcr.io/aleks-dolotin/external-dns-regru-webhook \
  --set image.tag=v1.0.0

# Create credentials secret
kubectl create secret generic reg-credentials \
  --from-literal=username=YOUR_USER \
  --from-literal=password=YOUR_PASS \
  -n reg-adapter
```

See [k8s/deploy/README.md](k8s/deploy/README.md) for detailed deployment instructions.

## Configuration

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `REGU_USERNAME` | — | Reg.ru account login |
| `REGU_PASSWORD` | — | Reg.ru API password (or account password) |
| `AUTH_DRIVER` | `token` | Auth method: `token` or `rsa` |
| `REGADAPTER_MAPPINGS_PATH` | `/etc/reg-adapter/mappings.yaml` | Path to zone-namespace mapping config |
| `REGADAPTER_RATE_PER_HOUR` | `1200` | Per-zone rate limit (requests/hour) |
| `REGADAPTER_SAFE_MODE` | `false` | Enable safe-mode at startup |
| `WORKER_CONCURRENCY` | `2` | Number of concurrent worker goroutines |
| `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |

### Mappings Config (mappings.yaml)

```yaml
zones:
  - zone: example.com
    namespaces: ["prod", "staging"]
    template: "{{.Name}}.{{.Zone}}"
    ttl: 300
    quota_per_hour: 100    # per-namespace quota (0 = unlimited)
  - zone: internal.io
    namespaces: []          # empty = all namespaces allowed
    template: "{{.Name}}-{{.Namespace}}.internal.io"
```

## API Endpoints

| Method | Path | Description |
|---|---|---|
| POST | `/adapter/v1/events` | Submit DNS events from ExternalDNS |
| GET | `/adapter/v1/failed` | List failed (dead-letter) operations |
| POST | `/adapter/v1/replay/{id}` | Replay a specific failed operation |
| POST | `/adapter/v1/replay-all` | Replay all failed operations |
| POST | `/adapter/v1/resync?zone=X` | Force-resync a zone |
| POST | `/adapter/v1/resync?namespace=X` | Force-resync all zones for a namespace |
| POST | `/adapter/v1/safe-mode?enabled=true` | Enable/disable safe-mode |
| GET | `/adapter/v1/safe-mode` | Get safe-mode status |
| GET | `/adapter/v1/diagnostics` | Queue, worker, circuit breaker status |
| GET | `/healthz` | Liveness probe |
| GET | `/ready` | Readiness probe |
| GET | `/metrics` | Prometheus metrics (OpenMetrics) |

## Testing

```bash
make test               # Unit tests (19 packages)
make test-race          # Unit tests with race detector
make lint               # golangci-lint
make check              # vet + test + test-race

# Integration (mock Reg.ru server)
make test-integration

# Smoke tests (real Reg.ru API — requires credentials)
REGU_USERNAME='...' REGU_PASSWORD='...' SMOKE_TEST_ZONE='yourdomain.com' \
  make test-smoke
```

## Releasing

Releases are automated via [GoReleaser](https://goreleaser.com/) and GitHub Actions.

```bash
# Tag a release
git tag v1.0.0
git push origin v1.0.0
```

This triggers:
1. **GoReleaser** — builds binaries for linux/darwin (amd64/arm64), creates GitHub Release with changelog
2. **Docker** — builds and pushes multi-arch image to `ghcr.io/aleks-dolotin/external-dns-regru-webhook:v1.0.0`

### Manual Docker build

```bash
docker build -t reg-adapter:v1.0.0 --build-arg VERSION=v1.0.0 .
```

## Architecture

```
cmd/sidecar/          — HTTP sidecar (main entrypoint)
internal/
  adapter/            — Reg.ru API v2 HTTP client (CRUD, cache, retry)
  auth/               — Pluggable auth drivers (token, RSA)
  audit/              — Audit trail via structured logging
  circuitbreaker/     — Per-zone circuit breaker
  config/             — ConfigMap-based zone-namespace mappings with hot-reload
  desiredstate/       — In-memory desired DNS state cache for force-resync
  health/             — Healthz, readyz, diagnostics endpoints
  logging/            — Structured logging (zap)
  metrics/            — Prometheus metrics (V1 + V2 with namespace labels)
  normalizer/         — ExternalDNS event → adapter.Operation conversion
  queue/              — In-memory operation queue
  quota/              — Per-namespace quota enforcement (fixed-window)
  ratelimit/          — Per-zone token-bucket rate limiter
  reconciler/         — Drift detection and corrective action enqueue
  retry/              — Retry with exponential backoff and jitter
  safemode/           — Safe-mode (no-write) toggle
  worker/             — Concurrent worker pool with dead-letter
charts/reg-adapter/   — Helm chart
k8s/deploy/           — Raw Kubernetes manifests
docs/                 — OpenAPI spec, Grafana queries, runbooks
```

## Reg.ru API Compatibility

This adapter uses **client-accessible** Reg.ru API v2 endpoints (not reseller-only):

| Operation | Endpoint | Access |
|---|---|---|
| Create A | `zone/add_alias` | Clients |
| Create AAAA | `zone/add_aaaa` | Clients |
| Create CNAME | `zone/add_cname` | Clients |
| Create TXT | `zone/add_txt` | Clients |
| Read | `zone/get_resource_records` | Clients |
| Delete | `zone/remove_record` | Clients |
| Update | remove_record + add_* | Clients |
| Bulk Update | Sequential add_*/remove_record calls | Clients |

## License

MIT
