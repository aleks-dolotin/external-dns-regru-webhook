package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/metrics"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/queue"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/retry"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// mockAdapter implements adapter.Adapter for testing.
type mockAdapter struct {
	createErr error
	updateErr error
	deleteErr error
	calls     int32
}

func (m *mockAdapter) FindRecord(_, _, _ string) (*adapter.Record, error) { return nil, nil }
func (m *mockAdapter) CreateRecord(_ string, _ *adapter.Record) error {
	atomic.AddInt32(&m.calls, 1)
	return m.createErr
}
func (m *mockAdapter) UpdateRecord(_ string, _ *adapter.Record) error {
	atomic.AddInt32(&m.calls, 1)
	return m.updateErr
}
func (m *mockAdapter) DeleteRecord(_, _ string) error {
	atomic.AddInt32(&m.calls, 1)
	return m.deleteErr
}
func (m *mockAdapter) BulkUpdate(_ string, _ []adapter.BulkAction) error { return nil }

// fastRetry is a retry policy for tests (no real waits).
var fastRetry = retry.Policy{MaxAttempts: 3, InitialBackoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond}

func TestWorkerPool_WorkerCount(t *testing.T) {
	q := queue.New()
	p := New(nil, q)

	if p.WorkerCount() != 0 {
		t.Errorf("expected 0 workers before start, got %d", p.WorkerCount())
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 3)
	time.Sleep(50 * time.Millisecond)

	if p.WorkerCount() != 3 {
		t.Errorf("expected 3 workers, got %d", p.WorkerCount())
	}

	cancel()
	p.wg.Wait()
	time.Sleep(50 * time.Millisecond)

	if p.WorkerCount() != 0 {
		t.Errorf("expected 0 workers after stop, got %d", p.WorkerCount())
	}
}

func TestWorkerPool_HandleWithRetry_Success(t *testing.T) {
	q := queue.New()
	ma := &mockAdapter{}
	p := New(ma, q)
	p.SetRetryPolicy(fastRetry)

	q.Enqueue(queue.Operation{
		ID:   "op-1",
		Body: adapter.Operation{OpID: "op-1", Action: "create", Zone: "example.com", Name: "a.example.com", Type: "A", Content: "1.2.3.4"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 1)
	time.Sleep(300 * time.Millisecond)
	cancel()
	p.wg.Wait()

	if atomic.LoadInt32(&ma.calls) != 1 {
		t.Errorf("expected 1 adapter call, got %d", ma.calls)
	}
	if !p.LastHeartbeat().After(time.Now().Add(-5 * time.Second)) {
		t.Error("expected recent heartbeat after success")
	}
	if len(p.FailedOps()) != 0 {
		t.Errorf("expected no failed ops, got %d", len(p.FailedOps()))
	}
}

func TestWorkerPool_HandleWithRetry_ExhaustedRetries(t *testing.T) {
	q := queue.New()
	ma := &mockAdapter{createErr: errors.New("transient")}
	p := New(ma, q)
	p.SetRetryPolicy(fastRetry)

	q.Enqueue(queue.Operation{
		ID:   "op-fail",
		Body: adapter.Operation{OpID: "op-fail", Action: "create", Zone: "fail.com", Name: "x.fail.com", Type: "A"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 1)
	time.Sleep(300 * time.Millisecond)
	cancel()
	p.wg.Wait()

	if atomic.LoadInt32(&ma.calls) != 3 { // 3 attempts per fastRetry
		t.Errorf("expected 3 adapter calls, got %d", ma.calls)
	}
	failed := p.FailedOps()
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed op, got %d", len(failed))
	}
	if failed[0].Op.OpID != "op-fail" {
		t.Errorf("expected failed op 'op-fail', got %q", failed[0].Op.OpID)
	}
	if failed[0].Attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", failed[0].Attempts)
	}
	errs := p.LastErrors()
	if _, ok := errs["fail.com"]; !ok {
		t.Error("expected zone error for 'fail.com'")
	}
}

func TestWorkerPool_HandleWithRetry_SuccessAfterRetry(t *testing.T) {
	q := queue.New()
	callCount := int32(0)
	ma := &mockAdapter{}
	// Override createErr dynamically: fail first 2, succeed on 3rd
	p := New(nil, q)
	p.adapter = &flakyAdapter{failCount: 2, current: &callCount}
	p.SetRetryPolicy(fastRetry)

	q.Enqueue(queue.Operation{
		ID:   "op-retry",
		Body: adapter.Operation{OpID: "op-retry", Action: "create", Zone: "retry.com", Name: "r.retry.com", Type: "A"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 1)
	time.Sleep(300 * time.Millisecond)
	cancel()
	p.wg.Wait()
	_ = ma // unused, but keeps the pattern

	if len(p.FailedOps()) != 0 {
		t.Errorf("expected no failed ops after eventual success, got %d", len(p.FailedOps()))
	}
	if p.LastHeartbeat().IsZero() {
		t.Error("expected heartbeat after eventual success")
	}
}

func TestWorkerPool_Dispatch_Delete(t *testing.T) {
	q := queue.New()
	ma := &mockAdapter{}
	p := New(ma, q)
	p.SetRetryPolicy(fastRetry)

	q.Enqueue(queue.Operation{
		ID:   "op-del",
		Body: adapter.Operation{OpID: "op-del", Action: "delete", Zone: "del.com"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 1)
	time.Sleep(300 * time.Millisecond)
	cancel()
	p.wg.Wait()

	if atomic.LoadInt32(&ma.calls) != 1 {
		t.Errorf("expected 1 delete call, got %d", ma.calls)
	}
}

func TestWorkerPool_Dispatch_Update(t *testing.T) {
	q := queue.New()
	ma := &mockAdapter{}
	p := New(ma, q)
	p.SetRetryPolicy(fastRetry)

	q.Enqueue(queue.Operation{
		ID:   "op-upd",
		Body: adapter.Operation{OpID: "op-upd", Action: "update", Zone: "upd.com", Name: "u.upd.com", Type: "CNAME"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 1)
	time.Sleep(300 * time.Millisecond)
	cancel()
	p.wg.Wait()

	if atomic.LoadInt32(&ma.calls) != 1 {
		t.Errorf("expected 1 update call, got %d", ma.calls)
	}
}

func TestWorkerPool_SkipInvalidBody(t *testing.T) {
	q := queue.New()
	ma := &mockAdapter{}
	p := New(ma, q)
	p.SetRetryPolicy(fastRetry)

	// Body is a string, not adapter.Operation
	q.Enqueue(queue.Operation{ID: "bad-body", Body: "not an operation"})

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 1)
	time.Sleep(200 * time.Millisecond)
	cancel()
	p.wg.Wait()

	if atomic.LoadInt32(&ma.calls) != 0 {
		t.Errorf("expected 0 adapter calls for invalid body, got %d", ma.calls)
	}
}

func TestWorkerPool_LastErrors_IsCopy(t *testing.T) {
	q := queue.New()
	p := New(nil, q)

	p.recordZoneError("a.com", errors.New("err1"))

	errs := p.LastErrors()
	errs["b.com"] = ZoneError{Zone: "b.com", Message: "injected"}

	original := p.LastErrors()
	if _, ok := original["b.com"]; ok {
		t.Error("LastErrors should return a copy")
	}
}

func TestWorkerPool_FailedOps_IsCopy(t *testing.T) {
	q := queue.New()
	p := New(nil, q)

	p.mu.Lock()
	p.failedOps = append(p.failedOps, FailedOp{Op: adapter.Operation{OpID: "x"}, Attempts: 1})
	p.mu.Unlock()

	fops := p.FailedOps()
	if len(fops) != 1 {
		t.Fatalf("expected 1 failed op, got %d", len(fops))
	}
	_ = append(fops, FailedOp{Op: adapter.Operation{OpID: "y"}})
	if len(p.FailedOps()) != 1 {
		t.Error("FailedOps should return a copy")
	}
}

func TestWorkerPool_SetRetryPolicy(t *testing.T) {
	q := queue.New()
	p := New(nil, q)

	custom := retry.Policy{MaxAttempts: 10, InitialBackoff: 5 * time.Second, MaxBackoff: 120 * time.Second}
	p.SetRetryPolicy(custom)

	if p.retryPolicy.MaxAttempts != 10 {
		t.Errorf("expected MaxAttempts 10, got %d", p.retryPolicy.MaxAttempts)
	}
}

// flakyAdapter fails the first N calls then succeeds.
type flakyAdapter struct {
	failCount int
	current   *int32
}

func (f *flakyAdapter) FindRecord(_, _, _ string) (*adapter.Record, error) { return nil, nil }
func (f *flakyAdapter) CreateRecord(_ string, _ *adapter.Record) error {
	n := atomic.AddInt32(f.current, 1)
	if int(n) <= f.failCount {
		return errors.New("transient failure")
	}
	return nil
}
func (f *flakyAdapter) UpdateRecord(_ string, _ *adapter.Record) error    { return nil }
func (f *flakyAdapter) DeleteRecord(_, _ string) error                    { return nil }
func (f *flakyAdapter) BulkUpdate(_ string, _ []adapter.BulkAction) error { return nil }

// --- Story 6.1: V2 metrics tests ---

func TestWorkerPool_V2Metrics_Success(t *testing.T) {
	q := queue.New()
	ma := &mockAdapter{}
	p := New(ma, q)
	p.SetRetryPolicy(fastRetry)

	// Record baseline counter values before test.
	beforeCounter := getCounterValue(t, metrics.RequestsTotalV2, "v2test.com", "A", "create", "success")
	beforeHist := getHistogramCount(t, metrics.RequestDurationV2, "v2test.com", "A", "create")

	q.Enqueue(queue.Operation{
		ID:   "op-v2-ok",
		Body: adapter.Operation{OpID: "op-v2-ok", Action: "create", Zone: "v2test.com", Name: "a.v2test.com", Type: "A", Content: "1.2.3.4"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 1)
	time.Sleep(300 * time.Millisecond)
	cancel()
	p.wg.Wait()

	afterCounter := getCounterValue(t, metrics.RequestsTotalV2, "v2test.com", "A", "create", "success")
	if afterCounter-beforeCounter != 1 {
		t.Errorf("expected RequestsTotalV2{v2test.com,A,create,success} delta=1, got %.0f", afterCounter-beforeCounter)
	}

	afterHist := getHistogramCount(t, metrics.RequestDurationV2, "v2test.com", "A", "create")
	if afterHist-beforeHist != 1 {
		t.Errorf("expected RequestDurationV2 observation count delta=1, got %d", afterHist-beforeHist)
	}
}

func TestWorkerPool_V2Metrics_Failure(t *testing.T) {
	q := queue.New()
	ma := &mockAdapter{createErr: errors.New("permanent")}
	p := New(ma, q)
	p.SetRetryPolicy(fastRetry)

	beforeCounter := getCounterValue(t, metrics.RequestsTotalV2, "v2fail.com", "CNAME", "create", "failure")
	beforeHist := getHistogramCount(t, metrics.RequestDurationV2, "v2fail.com", "CNAME", "create")

	q.Enqueue(queue.Operation{
		ID:   "op-v2-fail",
		Body: adapter.Operation{OpID: "op-v2-fail", Action: "create", Zone: "v2fail.com", Name: "c.v2fail.com", Type: "CNAME"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 1)
	time.Sleep(300 * time.Millisecond)
	cancel()
	p.wg.Wait()

	afterCounter := getCounterValue(t, metrics.RequestsTotalV2, "v2fail.com", "CNAME", "create", "failure")
	if afterCounter-beforeCounter != 1 {
		t.Errorf("expected RequestsTotalV2{v2fail.com,CNAME,create,failure} delta=1, got %.0f", afterCounter-beforeCounter)
	}

	// Duration should be recorded for each attempt (3 attempts with fastRetry).
	afterHist := getHistogramCount(t, metrics.RequestDurationV2, "v2fail.com", "CNAME", "create")
	delta := afterHist - beforeHist
	if delta != 3 {
		t.Errorf("expected RequestDurationV2 observation count delta=3 (one per attempt), got %d", delta)
	}
}

func TestWorkerPool_QueueDepthGauge(t *testing.T) {
	q := queue.New()
	ma := &mockAdapter{}
	p := New(ma, q)
	p.SetRetryPolicy(fastRetry)

	// Enqueue 3 items.
	for i := 0; i < 3; i++ {
		q.Enqueue(queue.Operation{
			ID:   fmt.Sprintf("op-qd-%d", i),
			Body: adapter.Operation{OpID: fmt.Sprintf("op-qd-%d", i), Action: "create", Zone: "qd.com", Name: "a.qd.com", Type: "A", Content: "1.2.3.4"},
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 1)
	time.Sleep(500 * time.Millisecond) // wait for all ops to process
	cancel()
	p.wg.Wait()

	// After all ops processed, queue should be empty.
	v := getGaugeValue(t, metrics.QueueDepth)
	if v != 0 {
		t.Errorf("expected QueueDepth=0 after processing, got %.0f", v)
	}
}

func TestWorkerPool_WorkerCountGauge(t *testing.T) {
	q := queue.New()
	p := New(nil, q)

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 2)
	time.Sleep(100 * time.Millisecond)

	v := getGaugeValue(t, metrics.WorkerCountGauge)
	if v != 2 {
		t.Errorf("expected WorkerCountGauge=2, got %.0f", v)
	}

	cancel()
	p.wg.Wait()
	time.Sleep(50 * time.Millisecond)

	v = getGaugeValue(t, metrics.WorkerCountGauge)
	if v != 0 {
		t.Errorf("expected WorkerCountGauge=0 after stop, got %.0f", v)
	}
}

// --- Metric helper functions for worker tests ---

func getCounterValue(t *testing.T, counter *prometheus.CounterVec, labels ...string) float64 {
	t.Helper()
	m := &dto.Metric{}
	if err := counter.WithLabelValues(labels...).Write(m); err != nil {
		t.Fatalf("failed to read counter: %v", err)
	}
	return m.GetCounter().GetValue()
}

func getHistogramCount(t *testing.T, hist *prometheus.HistogramVec, labels ...string) uint64 {
	t.Helper()
	observer := hist.WithLabelValues(labels...)
	h, ok := observer.(prometheus.Histogram)
	if !ok {
		t.Fatal("WithLabelValues did not return a prometheus.Histogram")
	}
	m := &dto.Metric{}
	if err := h.Write(m); err != nil {
		t.Fatalf("failed to read histogram: %v", err)
	}
	return m.GetHistogram().GetSampleCount()
}

func getGaugeValue(t *testing.T, gauge prometheus.Gauge) float64 {
	t.Helper()
	m := &dto.Metric{}
	if err := gauge.Write(m); err != nil {
		t.Fatalf("failed to read gauge: %v", err)
	}
	return m.GetGauge().GetValue()
}

// --- Story 6.4: correlating_id tracing test ---

func TestWorkerPool_CorrelatingID_InLogs(t *testing.T) {
	q := queue.New()
	ma := &mockAdapter{}
	p := New(ma, q)
	p.SetRetryPolicy(fastRetry)

	// Capture log output.
	var buf bytes.Buffer
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:     "timestamp",
		LevelKey:    "level",
		MessageKey:  "msg",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
		EncodeTime:  zapcore.ISO8601TimeEncoder,
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(&buf),
		zapcore.DebugLevel,
	)
	p.SetLogger(zap.New(core))

	opID := "trace-uuid-12345"
	q.Enqueue(queue.Operation{
		ID: opID,
		Body: adapter.Operation{
			OpID:    opID,
			Action:  "create",
			Zone:    "trace.com",
			Name:    "a.trace.com",
			Type:    "A",
			Content: "1.2.3.4",
			ResourceRef: adapter.ResourceRef{
				Kind:      "Ingress",
				Namespace: "default",
				Name:      "my-svc",
			},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 1)
	time.Sleep(400 * time.Millisecond)
	cancel()
	p.wg.Wait()

	output := buf.String()

	// All log lines for this operation should contain the correlating_id.
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	foundDequeue := false
	foundDispatch := false
	foundSuccess := false

	for _, line := range lines {
		var entry map[string]interface{}
		if json.Unmarshal(line, &entry) != nil {
			continue
		}
		cid, _ := entry["correlating_id"].(string)
		if cid != opID {
			continue
		}
		msg, _ := entry["msg"].(string)
		switch msg {
		case "operation dequeued":
			foundDequeue = true
		case "dispatching to adapter":
			foundDispatch = true
		case "operation succeeded":
			foundSuccess = true
			// Verify structured fields.
			if entry["zone"] != "trace.com" {
				t.Errorf("expected zone='trace.com', got %v", entry["zone"])
			}
			if entry["operation"] != "create" {
				t.Errorf("expected operation='create', got %v", entry["operation"])
			}
			if entry["resource"] != "my-svc" {
				t.Errorf("expected resource='my-svc', got %v", entry["resource"])
			}
			if entry["namespace"] != "default" {
				t.Errorf("expected namespace='default', got %v", entry["namespace"])
			}
		}
	}

	if !foundDequeue {
		t.Error("expected 'operation dequeued' log with correlating_id, not found")
	}
	if !foundDispatch {
		t.Error("expected 'dispatching to adapter' log with correlating_id, not found")
	}
	if !foundSuccess {
		t.Error("expected 'operation succeeded' log with correlating_id, not found")
	}
	_ = output
}
