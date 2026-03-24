package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "regru", Name: "api_requests_total", Help: "Total requests to reg.ru API"},
		[]string{"operation", "outcome"},
	)
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Namespace: "regru", Name: "api_request_duration_seconds", Help: "Duration of requests to reg.ru API"},
		[]string{"operation"},
	)
	RetriesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: "regru", Name: "api_retries_total", Help: "Total retry attempts for reg.ru API operations"},
	)
	FailedOpsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "regru", Name: "failed_ops_total", Help: "Operations that exhausted all retry attempts, broken down by namespace"},
		[]string{"action", "namespace"},
	)

	// Story 5.1: per-zone rate limiter metrics.
	// Story 9.3: added namespace label for per-tenant breakdown.
	RateLimitedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "regru", Name: "rate_limited_total", Help: "Total requests rejected by the per-zone rate limiter, broken down by namespace"},
		[]string{"zone", "namespace"},
	)

	// Story 5.2: retry/backoff metrics.
	APIRetriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "regru", Name: "api_retries_total_labeled", Help: "Total retry attempts labeled by zone and reason"},
		[]string{"zone", "reason"},
	)
	APIBackoffSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "regru",
			Name:      "api_backoff_seconds",
			Help:      "Backoff wait duration in seconds before retry",
			Buckets:   []float64{0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"zone"},
	)

	// Story 5.4: per-zone circuit breaker metrics.
	CircuitState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Namespace: "regru", Name: "circuit_state", Help: "Circuit breaker state per zone (0=closed, 1=half-open, 2=open)"},
		[]string{"zone"},
	)

	// Story 9.1: namespace isolation rejection metric.
	NamespaceRejectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "regru", Name: "namespace_rejected_total", Help: "Total requests rejected due to namespace isolation policy"},
		[]string{"zone", "namespace"},
	)

	// Story 9.2: per-namespace quota metrics.
	NamespaceQuotaUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Namespace: "regru", Name: "namespace_quota_used", Help: "Current quota usage per namespace within the hourly window"},
		[]string{"namespace"},
	)
	NamespaceQuotaLimit = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Namespace: "regru", Name: "namespace_quota_limit", Help: "Configured quota limit per namespace (requests/hour)"},
		[]string{"namespace"},
	)
	NamespaceQuotaRejectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "regru", Name: "namespace_quota_rejected_total", Help: "Total requests rejected due to per-namespace quota exceeded"},
		[]string{"namespace"},
	)

	// Story 6.1 + Story 9.3: extended metrics with zone, record_type, and namespace labels.
	// V2 metrics coexist with originals for backward compatibility during transition.
	RequestsTotalV2 = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "regru", Name: "api_requests_total_v2", Help: "Total requests to reg.ru API with zone, record_type, and namespace labels"},
		[]string{"zone", "record_type", "operation", "outcome", "namespace"},
	)
	// Story 9.3: RequestDurationV2 histogram intentionally does NOT have namespace label
	// to limit cardinality (zones × namespaces × record_types × operations × buckets).
	RequestDurationV2 = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Namespace: "regru", Name: "api_request_duration_seconds_v2", Help: "Duration of requests to reg.ru API with zone and record_type labels"},
		[]string{"zone", "record_type", "operation"},
	)

	// Story 6.1: operational gauges.
	QueueDepth = prometheus.NewGauge(
		prometheus.GaugeOpts{Namespace: "regru", Name: "queue_depth", Help: "Current number of operations in the work queue"},
	)
	WorkerCountGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{Namespace: "regru", Name: "worker_count", Help: "Current number of running worker goroutines"},
	)

	registerOnce sync.Once
)

// Register registers all metrics with the given registerer. Safe to call
// multiple times — registration happens only once.
func Register(r prometheus.Registerer) {
	registerOnce.Do(func() {
		r.MustRegister(RequestsTotal)
		r.MustRegister(RequestDuration)
		r.MustRegister(RetriesTotal)
		r.MustRegister(FailedOpsTotal)
		r.MustRegister(RateLimitedTotal)
		r.MustRegister(APIRetriesTotal)
		r.MustRegister(APIBackoffSeconds)
		r.MustRegister(CircuitState)
		r.MustRegister(NamespaceRejectedTotal)
		r.MustRegister(NamespaceQuotaUsed)
		r.MustRegister(NamespaceQuotaLimit)
		r.MustRegister(NamespaceQuotaRejectedTotal)
		r.MustRegister(RequestsTotalV2)
		r.MustRegister(RequestDurationV2)
		r.MustRegister(QueueDepth)
		r.MustRegister(WorkerCountGauge)
	})
}
