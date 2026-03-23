package worker

import (
	"context"
	"testing"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/queue"
)

func TestWorkerPool_WorkerCount(t *testing.T) {
	q := queue.New()
	p := New(nil, q)

	if p.WorkerCount() != 0 {
		t.Errorf("expected 0 workers before start, got %d", p.WorkerCount())
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 3)

	// Give workers time to start
	time.Sleep(50 * time.Millisecond)

	if p.WorkerCount() != 3 {
		t.Errorf("expected 3 workers, got %d", p.WorkerCount())
	}

	cancel()
	p.wg.Wait()

	// Give time for atomic decrements
	time.Sleep(50 * time.Millisecond)

	if p.WorkerCount() != 0 {
		t.Errorf("expected 0 workers after stop, got %d", p.WorkerCount())
	}
}

func TestWorkerPool_LastHeartbeat(t *testing.T) {
	q := queue.New()
	p := New(nil, q)

	if !p.LastHeartbeat().IsZero() {
		t.Error("expected zero heartbeat before any work")
	}

	// Enqueue an operation and start a worker to handle it
	q.Enqueue(queue.Operation{ID: "test-1"})

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx, 1)

	// Wait for worker to process
	time.Sleep(200 * time.Millisecond)

	cancel()
	p.wg.Wait()

	hb := p.LastHeartbeat()
	if hb.IsZero() {
		t.Error("expected non-zero heartbeat after processing work")
	}
	if time.Since(hb) > 2*time.Second {
		t.Error("heartbeat too old")
	}
}

func TestWorkerPool_LastErrors(t *testing.T) {
	q := queue.New()
	p := New(nil, q)

	errs := p.LastErrors()
	if len(errs) != 0 {
		t.Errorf("expected empty errors map, got %d", len(errs))
	}

	// Simulate recording a zone error
	p.recordZoneError("example.com", &testError{msg: "connection refused"})

	errs = p.LastErrors()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	ze, ok := errs["example.com"]
	if !ok {
		t.Fatal("expected error for 'example.com'")
	}
	if ze.Message != "connection refused" {
		t.Errorf("expected 'connection refused', got %q", ze.Message)
	}
	if ze.Time.IsZero() {
		t.Error("expected non-zero error time")
	}
}

func TestWorkerPool_LastErrors_IsCopy(t *testing.T) {
	q := queue.New()
	p := New(nil, q)

	p.recordZoneError("a.com", &testError{msg: "err1"})

	errs := p.LastErrors()
	errs["b.com"] = ZoneError{Zone: "b.com", Message: "injected"}

	// The original map should not be affected
	original := p.LastErrors()
	if _, ok := original["b.com"]; ok {
		t.Error("LastErrors should return a copy, not the original map")
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
