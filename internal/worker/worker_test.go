package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/queue"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/retry"
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
