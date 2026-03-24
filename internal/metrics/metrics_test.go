package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func getCounterValue(counter *prometheus.CounterVec, labels ...string) float64 {
	m := &dto.Metric{}
	if err := counter.WithLabelValues(labels...).Write(m); err != nil {
		return 0
	}
	return m.GetCounter().GetValue()
}

func getHistogramCount(hist *prometheus.HistogramVec, labels ...string) uint64 {
	observer := hist.WithLabelValues(labels...)
	// prometheus.Observer doesn't expose Write; cast to Histogram which does.
	h, ok := observer.(prometheus.Histogram)
	if !ok {
		return 0
	}
	m := &dto.Metric{}
	if err := h.Write(m); err != nil {
		return 0
	}
	return m.GetHistogram().GetSampleCount()
}

func getGaugeValue(gauge prometheus.Gauge) float64 {
	m := &dto.Metric{}
	if err := gauge.Write(m); err != nil {
		return 0
	}
	return m.GetGauge().GetValue()
}

func TestRequestsTotalV2_IncrementsWithAllLabels(t *testing.T) {
	// Reset the metric by creating fresh label combinations.
	// Note: CounterVec cannot be reset, so we test increment delta.
	// Story 9.3: namespace label added as 5th label.
	before := getCounterValue(RequestsTotalV2, "example.com", "A", "create", "success", "prod")

	RequestsTotalV2.WithLabelValues("example.com", "A", "create", "success", "prod").Inc()
	RequestsTotalV2.WithLabelValues("example.com", "A", "create", "success", "prod").Inc()
	RequestsTotalV2.WithLabelValues("other.com", "CNAME", "delete", "failure", "staging").Inc()

	after := getCounterValue(RequestsTotalV2, "example.com", "A", "create", "success", "prod")
	delta := after - before
	if delta != 2 {
		t.Errorf("expected RequestsTotalV2{example.com,A,create,success,prod} delta=2, got %.0f", delta)
	}

	otherAfter := getCounterValue(RequestsTotalV2, "other.com", "CNAME", "delete", "failure", "staging")
	if otherAfter < 1 {
		t.Errorf("expected RequestsTotalV2{other.com,CNAME,delete,failure,staging} >= 1, got %.0f", otherAfter)
	}
}

func TestRequestDurationV2_ObservesWithCorrectLabels(t *testing.T) {
	before := getHistogramCount(RequestDurationV2, "example.com", "A", "create")

	RequestDurationV2.WithLabelValues("example.com", "A", "create").Observe(0.123)
	RequestDurationV2.WithLabelValues("example.com", "A", "create").Observe(0.456)

	after := getHistogramCount(RequestDurationV2, "example.com", "A", "create")
	delta := after - before
	if delta != 2 {
		t.Errorf("expected RequestDurationV2 observation count delta=2, got %d", delta)
	}
}

func TestQueueDepth_ReflectsSetValue(t *testing.T) {
	QueueDepth.Set(42)
	v := getGaugeValue(QueueDepth)
	if v != 42 {
		t.Errorf("expected QueueDepth=42, got %.0f", v)
	}

	QueueDepth.Set(0)
	v = getGaugeValue(QueueDepth)
	if v != 0 {
		t.Errorf("expected QueueDepth=0, got %.0f", v)
	}
}

func TestWorkerCountGauge_ReflectsSetValue(t *testing.T) {
	WorkerCountGauge.Set(5)
	v := getGaugeValue(WorkerCountGauge)
	if v != 5 {
		t.Errorf("expected WorkerCountGauge=5, got %.0f", v)
	}

	WorkerCountGauge.Set(0)
	v = getGaugeValue(WorkerCountGauge)
	if v != 0 {
		t.Errorf("expected WorkerCountGauge=0, got %.0f", v)
	}
}

// ===== Story 9.3: Tenant metrics breakdown tests =====

func TestRateLimitedTotal_HasNamespaceLabel(t *testing.T) {
	before := getCounterValue(RateLimitedTotal, "example.com", "team-a")
	RateLimitedTotal.WithLabelValues("example.com", "team-a").Inc()
	after := getCounterValue(RateLimitedTotal, "example.com", "team-a")
	if after-before != 1 {
		t.Errorf("expected RateLimitedTotal with namespace label delta=1, got %.0f", after-before)
	}
}

func TestFailedOpsTotal_HasNamespaceLabel(t *testing.T) {
	before := getCounterValue(FailedOpsTotal, "create", "team-b")
	FailedOpsTotal.WithLabelValues("create", "team-b").Inc()
	after := getCounterValue(FailedOpsTotal, "create", "team-b")
	if after-before != 1 {
		t.Errorf("expected FailedOpsTotal with namespace label delta=1, got %.0f", after-before)
	}
}

func TestRequestsTotalV2_NamespaceLabelPresent(t *testing.T) {
	// Verify that different namespaces produce independent counters.
	beforeA := getCounterValue(RequestsTotalV2, "z.com", "A", "create", "success", "ns-a")
	beforeB := getCounterValue(RequestsTotalV2, "z.com", "A", "create", "success", "ns-b")

	RequestsTotalV2.WithLabelValues("z.com", "A", "create", "success", "ns-a").Inc()
	RequestsTotalV2.WithLabelValues("z.com", "A", "create", "success", "ns-a").Inc()
	RequestsTotalV2.WithLabelValues("z.com", "A", "create", "success", "ns-b").Inc()

	afterA := getCounterValue(RequestsTotalV2, "z.com", "A", "create", "success", "ns-a")
	afterB := getCounterValue(RequestsTotalV2, "z.com", "A", "create", "success", "ns-b")

	if afterA-beforeA != 2 {
		t.Errorf("expected ns-a delta=2, got %.0f", afterA-beforeA)
	}
	if afterB-beforeB != 1 {
		t.Errorf("expected ns-b delta=1, got %.0f", afterB-beforeB)
	}
}

func TestRequestDurationV2_NoNamespaceLabel(t *testing.T) {
	// Story 9.3 AC#3: histograms should NOT have namespace label.
	// RequestDurationV2 should still accept exactly 3 labels: zone, record_type, operation.
	before := getHistogramCount(RequestDurationV2, "card.com", "AAAA", "update")
	RequestDurationV2.WithLabelValues("card.com", "AAAA", "update").Observe(0.5)
	after := getHistogramCount(RequestDurationV2, "card.com", "AAAA", "update")
	if after-before != 1 {
		t.Errorf("expected histogram count delta=1, got %d", after-before)
	}
}

func TestRegister_IncludesV2Metrics(t *testing.T) {
	// Verify that all expected metric collectors are non-nil and have correct descriptions.
	// We cannot rely on Register() + Gather() in unit tests because sync.Once
	// may already have fired with a different registry. Instead, verify the
	// exported variables are properly initialized.
	collectors := map[string]prometheus.Collector{
		"RequestsTotal":               RequestsTotal,
		"RequestDuration":             RequestDuration,
		"RetriesTotal":                RetriesTotal,
		"FailedOpsTotal":              FailedOpsTotal,
		"RateLimitedTotal":            RateLimitedTotal,
		"APIRetriesTotal":             APIRetriesTotal,
		"APIBackoffSeconds":           APIBackoffSeconds,
		"CircuitState":                CircuitState,
		"NamespaceRejectedTotal":      NamespaceRejectedTotal,
		"NamespaceQuotaUsed":          NamespaceQuotaUsed,
		"NamespaceQuotaLimit":         NamespaceQuotaLimit,
		"NamespaceQuotaRejectedTotal": NamespaceQuotaRejectedTotal,
		"RequestsTotalV2":             RequestsTotalV2,
		"RequestDurationV2":           RequestDurationV2,
		"QueueDepth":                  QueueDepth,
		"WorkerCountGauge":            WorkerCountGauge,
	}

	for name, c := range collectors {
		if c == nil {
			t.Errorf("expected %s to be non-nil", name)
			continue
		}
		// Verify Describe produces at least one descriptor.
		ch := make(chan *prometheus.Desc, 10)
		c.Describe(ch)
		close(ch)
		count := 0
		for range ch {
			count++
		}
		if count == 0 {
			t.Errorf("expected %s to produce at least one Desc, got 0", name)
		}
	}
}
