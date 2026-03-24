# Grafana PromQL Queries for Per-Tenant Monitoring

Story 9.3: Tenant metrics breakdown — example queries for namespace-level observability.

## Request Rate per Namespace

```promql
# Top 5 namespaces by request rate (5m window)
topk(5, sum by (namespace) (rate(regru_api_requests_total_v2[5m])))

# Request rate per namespace and zone
sum by (namespace, zone) (rate(regru_api_requests_total_v2[5m]))

# Request rate per namespace, operation, and outcome
sum by (namespace, operation, outcome) (rate(regru_api_requests_total_v2[5m]))
```

## Rate-Limited Requests per Namespace

```promql
# Rate-limited requests per namespace (5m rate)
sum by (namespace) (rate(regru_rate_limited_total[5m]))

# Rate-limited requests per namespace and zone
sum by (namespace, zone) (rate(regru_rate_limited_total[5m]))
```

## Failed Operations per Namespace

```promql
# Failed operations per namespace (hourly increase)
sum by (namespace) (increase(regru_failed_ops_total[1h]))

# Failed operations per namespace and action
sum by (namespace, action) (increase(regru_failed_ops_total[1h]))
```

## Namespace Quota Utilization

```promql
# Quota utilization ratio per namespace
regru_namespace_quota_used / regru_namespace_quota_limit

# Namespaces approaching quota limit (> 80%)
regru_namespace_quota_used / regru_namespace_quota_limit > 0.8

# Quota rejections per namespace (5m rate)
sum by (namespace) (rate(regru_namespace_quota_rejected_total[5m]))
```

## Namespace Isolation Rejections

```promql
# Cross-namespace rejections per zone
sum by (zone, namespace) (rate(regru_namespace_rejected_total[5m]))
```

## Alert Rules (PrometheusRule examples)

```yaml
groups:
  - name: regru-namespace-alerts
    rules:
      - alert: NamespaceQuotaNearLimit
        expr: regru_namespace_quota_used / regru_namespace_quota_limit > 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Namespace {{ $labels.namespace }} approaching quota limit"
          description: "Quota usage is {{ $value | humanizePercentage }} of limit."

      - alert: NamespaceQuotaExceeded
        expr: rate(regru_namespace_quota_rejected_total[5m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Namespace {{ $labels.namespace }} quota exceeded"
          description: "Requests are being rejected due to quota limits."

      - alert: NamespaceHighFailureRate
        expr: sum by (namespace) (rate(regru_failed_ops_total[5m])) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High failure rate for namespace {{ $labels.namespace }}"
```
