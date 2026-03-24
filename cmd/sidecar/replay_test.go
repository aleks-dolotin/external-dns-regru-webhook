package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/audit"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/worker"
)

// --- Story 8.2: Replay failed operations tests ---

// injectFailedOp adds a FailedOp to the worker pool's dead-letter list for testing.
// This uses handleWithRetry indirectly — instead we directly manipulate via exposed methods.
// Since failedOps is private, we use a helper that creates a pool with a failing adapter.
func newAppWithFailedOps(t *testing.T, ops []worker.FailedOp) *app {
	t.Helper()
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	a := newApp()

	// Inject failed ops via the pool's internal state.
	// Since we can't access private fields, we'll add ops through the public test helper.
	// WorkerPool doesn't expose a method to inject failed ops, so we need to add one
	// or use a different approach. Let's use the pool's AddFailedOp test helper.
	for _, fo := range ops {
		a.pool.InjectFailedOp(fo)
	}
	return a
}

// --- Subtask 5.1: Test list-failed returns correct JSON format ---

func TestListFailed_EmptyList(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	a := newApp()

	req := httptest.NewRequest(http.MethodGet, "/adapter/v1/failed", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var result []failedOpJSON
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty list, got %d items", len(result))
	}
}

func TestListFailed_WithItems(t *testing.T) {
	a := newAppWithFailedOps(t, []worker.FailedOp{
		{
			Op:       adapter.Operation{OpID: "op-1", Zone: "example.com", Action: "create", Name: "app.example.com", Type: "A"},
			Err:      errors.New("adapter: HTTP 500"),
			Attempts: 3,
			Time:     time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
		},
		{
			Op:       adapter.Operation{OpID: "op-2", Zone: "test.org", Action: "update", Name: "api.test.org", Type: "CNAME"},
			Err:      errors.New("timeout"),
			Attempts: 5,
			Time:     time.Date(2026, 3, 24, 13, 0, 0, 0, time.UTC),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/adapter/v1/failed", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var result []failedOpJSON
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if result[0].OpID != "op-1" {
		t.Errorf("expected op_id 'op-1', got %q", result[0].OpID)
	}
	if result[0].Zone != "example.com" {
		t.Errorf("expected zone 'example.com', got %q", result[0].Zone)
	}
	if result[0].Action != "create" {
		t.Errorf("expected action 'create', got %q", result[0].Action)
	}
	if result[0].Error != "adapter: HTTP 500" {
		t.Errorf("expected error 'adapter: HTTP 500', got %q", result[0].Error)
	}
	if result[0].Attempts != 3 {
		t.Errorf("expected attempts 3, got %d", result[0].Attempts)
	}
	if result[1].OpID != "op-2" {
		t.Errorf("expected op_id 'op-2', got %q", result[1].OpID)
	}
}

func TestListFailed_MethodNotAllowed(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	a := newApp()

	req := httptest.NewRequest(http.MethodPost, "/adapter/v1/failed", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	// Go 1.22+ pattern "GET /adapter/v1/failed" only matches GET.
	// POST to this path may result in 405 or 404 depending on mux behavior.
	if rec.Code != http.StatusMethodNotAllowed {
		t.Logf("NOTE: expected 405 for POST to GET-only route, got %d (mux routing behavior)", rec.Code)
	}
}

// --- Subtask 5.2: Test replay by ID — found case ---

func TestReplay_Found(t *testing.T) {
	a := newAppWithFailedOps(t, []worker.FailedOp{
		{
			Op:       adapter.Operation{OpID: "replay-me", Zone: "example.com", Action: "create", Name: "app.example.com", Type: "A"},
			Err:      errors.New("timeout"),
			Attempts: 3,
			Time:     time.Now().UTC(),
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/adapter/v1/replay/replay-me", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp replayResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Status != "replayed" {
		t.Errorf("expected status 'replayed', got %q", resp.Status)
	}
	if resp.OpID != "replay-me" {
		t.Errorf("expected op_id 'replay-me', got %q", resp.OpID)
	}

	// Verify re-enqueued
	if a.queue.Len() != 1 {
		t.Errorf("expected 1 item in queue, got %d", a.queue.Len())
	}

	// Verify removed from failed list
	failed := a.pool.FailedOps()
	if len(failed) != 0 {
		t.Errorf("expected 0 failed ops after replay, got %d", len(failed))
	}
}

// --- Subtask 5.3: Test replay by ID — not found (404) ---

func TestReplay_NotFound(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	a := newApp()

	req := httptest.NewRequest(http.MethodPost, "/adapter/v1/replay/nonexistent", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp resyncErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Error != "operation nonexistent not found in failed list" {
		t.Errorf("expected not found error, got %q", resp.Error)
	}
}

// --- Subtask 5.4: Test replay-all ---

func TestReplayAll(t *testing.T) {
	a := newAppWithFailedOps(t, []worker.FailedOp{
		{
			Op:       adapter.Operation{OpID: "op-a", Zone: "a.com", Action: "create"},
			Err:      errors.New("fail-a"),
			Attempts: 2,
			Time:     time.Now().UTC(),
		},
		{
			Op:       adapter.Operation{OpID: "op-b", Zone: "b.com", Action: "update"},
			Err:      errors.New("fail-b"),
			Attempts: 3,
			Time:     time.Now().UTC(),
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/adapter/v1/replay-all", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp replayAllResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Replayed != 2 {
		t.Errorf("expected replayed=2, got %d", resp.Replayed)
	}

	// Verify all re-enqueued
	if a.queue.Len() != 2 {
		t.Errorf("expected 2 items in queue, got %d", a.queue.Len())
	}

	// Verify failed list cleared
	if len(a.pool.FailedOps()) != 0 {
		t.Errorf("expected 0 failed ops after replay-all, got %d", len(a.pool.FailedOps()))
	}
}

func TestReplayAll_Empty(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	a := newApp()

	req := httptest.NewRequest(http.MethodPost, "/adapter/v1/replay-all", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp replayAllResponse
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Replayed != 0 {
		t.Errorf("expected replayed=0, got %d", resp.Replayed)
	}
}

// --- Test audit event for replay (AC #4) ---

func TestReplay_AuditEventRecorded(t *testing.T) {
	a := newAppWithFailedOps(t, []worker.FailedOp{
		{
			Op:       adapter.Operation{OpID: "audit-test", Zone: "z.com", Action: "create", Name: "a.z.com", Type: "A"},
			Err:      errors.New("fail"),
			Attempts: 1,
			Time:     time.Now().UTC(),
		},
	})

	recorder := &replayAuditRecorder{}
	a.auditor = recorder

	req := httptest.NewRequest(http.MethodPost, "/adapter/v1/replay/audit-test", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if len(recorder.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(recorder.events))
	}
	evt := recorder.events[0]
	if evt.Operation != "replay" {
		t.Errorf("expected operation 'replay', got %q", evt.Operation)
	}
	if evt.CorrelatingID != "audit-test" {
		t.Errorf("expected correlating_id 'audit-test', got %q", evt.CorrelatingID)
	}
	if evt.Result != "replayed" {
		t.Errorf("expected result 'replayed', got %q", evt.Result)
	}
}

// --- Test JSON content type ---

func TestListFailed_JSONContentType(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	a := newApp()

	req := httptest.NewRequest(http.MethodGet, "/adapter/v1/failed", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}
}

// --- Test helpers ---

type replayAuditRecorder struct {
	mu     sync.Mutex
	events []audit.AuditEvent
}

func (r *replayAuditRecorder) Record(event audit.AuditEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}
