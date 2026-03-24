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
		prometheus.CounterOpts{Namespace: "regru", Name: "failed_ops_total", Help: "Operations that exhausted all retry attempts"},
		[]string{"action"},
	)

	// Story 5.1: per-zone rate limiter metrics.
	RateLimitedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: "regru", Name: "rate_limited_total", Help: "Total requests rejected by the per-zone rate limiter"},
		[]string{"zone"},
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
	})
}
