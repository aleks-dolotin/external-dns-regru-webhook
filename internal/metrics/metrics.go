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
	})
}
