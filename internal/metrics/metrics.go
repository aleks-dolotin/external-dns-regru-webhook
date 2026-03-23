package metrics

import (
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
)

func Register(r prometheus.Registerer) {
    r.MustRegister(RequestsTotal)
    r.MustRegister(RequestDuration)
}

