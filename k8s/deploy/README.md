# Kubernetes deployment recommendations — Reg.ru Adapter

This folder contains recommended Kubernetes manifests for deploying the
ExternalDNS → Reg.ru adapter (sidecar listener + worker) in a cluster.

Files included
- `namespace.yaml` — namespace `reg-adapter` for isolation
- `configmap-reg-adapter.yaml` — example operator-config (mappings, rate limits, retry)
- `secret-reg-credentials.yaml` — secret template for Reg.ru credentials
- `deployment-reg-adapter.yaml` — adapter (sidecar listener) deployment
- `deployment-reg-worker.yaml` — worker deployment (background queue processors)
- `hpa-reg-worker.yaml` — HPA for workers (CPU based)
- `service-and-monitor.yaml` — Service and optional ServiceMonitor for Prometheus
- `pdb-reg-adapter.yaml` — PodDisruptionBudget to maintain availability

Quick apply
```bash
# from repo root
kubectl apply -f k8s/deploy/namespace.yaml
kubectl apply -f k8s/deploy/configmap-reg-adapter.yaml
# create secret with real values
kubectl create secret generic reg-credentials --from-literal=username=YOUR_USER --from-literal=password=YOUR_PASS -n reg-adapter
kubectl apply -f k8s/deploy/deployment-reg-adapter.yaml
kubectl apply -f k8s/deploy/deployment-reg-worker.yaml
kubectl apply -f k8s/deploy/service-and-monitor.yaml
kubectl apply -f k8s/deploy/hpa-reg-worker.yaml
kubectl apply -f k8s/deploy/pdb-reg-adapter.yaml
```

Resource sizing guidance
- Start with the provided conservative defaults:
  - adapter: requests: cpu=200m, memory=256Mi; limits: cpu=500m, memory=512Mi
  - worker: requests: cpu=300m, memory=512Mi; limits: cpu=1, memory=1Gi
- Tune using load tests and Prometheus metrics. Increase replicas/HPA thresholds if queue depth grows.

Probes and readiness
- The `readiness` probe should return false if config or secrets are invalid to avoid traffic.
- The `liveness` probe should be light-weight and fast.

Rate-limiting and backpressure
- The ConfigMap includes `rate_limiter` defaults. Implement local per-zone limiters that apply before contacting Reg.ru.

Security
- Use Kubernetes `Secret` for credentials. Consider sealed-secrets or external KMS for production.
- Ensure logs mask secrets and do not leak credentials.

Observability
- Expose `/metrics` for Prometheus with labels by zone and record_type.
- Configure alerts for `rate_limited_total` and `retries_total` thresholds.

Notes
- These manifests are a starting point for MVP. Perform performance and chaos testing to refine resource requests/limits and HPA settings.

