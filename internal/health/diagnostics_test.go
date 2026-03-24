package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockDiagSource is a test implementation of DiagnosticsSource.
type mockDiagSource struct {
	queueDepth     int
	workerCount    int
	heartbeat      time.Time
	zoneErrors     map[string]ZoneErrorInfo
	backpressured  bool
	throttledZones []string
	circuitStates  map[string]string
}

func (m *mockDiagSource) QueueDepth() int                      { return m.queueDepth }
func (m *mockDiagSource) WorkerCount() int                     { return m.workerCount }
func (m *mockDiagSource) LastHeartbeat() time.Time             { return m.heartbeat }
func (m *mockDiagSource) ZoneErrors() map[string]ZoneErrorInfo { return m.zoneErrors }
func (m *mockDiagSource) IsBackpressured() bool                { return m.backpressured }
func (m *mockDiagSource) ThrottledZones() []string             { return m.throttledZones }
func (m *mockDiagSource) CircuitStates() map[string]string     { return m.circuitStates }

func TestDiagnosticsHandler_Empty(t *testing.T) {
	src := &mockDiagSource{}
	handler := DiagnosticsHandler(src)

	req := httptest.NewRequest(http.MethodGet, "/adapter/v1/diagnostics", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var resp DiagnosticsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.QueueDepth != 0 {
		t.Errorf("expected queue_depth 0, got %d", resp.QueueDepth)
	}
	if resp.WorkerCount != 0 {
		t.Errorf("expected worker_count 0, got %d", resp.WorkerCount)
	}
	if resp.LastHeartbeat != nil {
		t.Error("expected nil last_heartbeat")
	}
	if resp.Zones != nil {
		t.Error("expected nil zones")
	}
}

func TestDiagnosticsHandler_WithData(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	errTime := now.Add(-5 * time.Minute)

	src := &mockDiagSource{
		queueDepth:  42,
		workerCount: 3,
		heartbeat:   now,
		zoneErrors: map[string]ZoneErrorInfo{
			"example.com": {Message: "timeout", Time: errTime},
		},
	}
	handler := DiagnosticsHandler(src)

	req := httptest.NewRequest(http.MethodGet, "/adapter/v1/diagnostics", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp DiagnosticsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.QueueDepth != 42 {
		t.Errorf("expected queue_depth 42, got %d", resp.QueueDepth)
	}
	if resp.WorkerCount != 3 {
		t.Errorf("expected worker_count 3, got %d", resp.WorkerCount)
	}
	if resp.LastHeartbeat == nil {
		t.Fatal("expected non-nil last_heartbeat")
	}
	if resp.Zones == nil {
		t.Fatal("expected non-nil zones")
	}
	zs, ok := resp.Zones["example.com"]
	if !ok {
		t.Fatal("expected zone 'example.com' in zones")
	}
	if zs.LastError != "timeout" {
		t.Errorf("expected last_error 'timeout', got %q", zs.LastError)
	}
}

func TestDiagnosticsHandler_NoErrors(t *testing.T) {
	now := time.Now().UTC()
	src := &mockDiagSource{
		queueDepth:  10,
		workerCount: 2,
		heartbeat:   now,
		zoneErrors:  nil,
	}
	handler := DiagnosticsHandler(src)

	req := httptest.NewRequest(http.MethodGet, "/adapter/v1/diagnostics", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	var resp DiagnosticsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Zones != nil {
		t.Errorf("expected nil zones when no errors, got %v", resp.Zones)
	}
}

func TestDiagnosticsHandler_MultipleZoneErrors(t *testing.T) {
	now := time.Now().UTC()
	src := &mockDiagSource{
		queueDepth:  100,
		workerCount: 5,
		heartbeat:   now,
		zoneErrors: map[string]ZoneErrorInfo{
			"a.com": {Message: "rate limited", Time: now},
			"b.com": {Message: "connection refused", Time: now},
		},
	}
	handler := DiagnosticsHandler(src)

	req := httptest.NewRequest(http.MethodGet, "/adapter/v1/diagnostics", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	var resp DiagnosticsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp.Zones) != 2 {
		t.Errorf("expected 2 zones, got %d", len(resp.Zones))
	}
}
