package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/auth"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/health"
	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/queue"
)

// setCredEnv uses t.Setenv for automatic cleanup after test.
func setCredEnv(t *testing.T, username, password string) {
	t.Helper()
	t.Setenv("REGU_USERNAME", username)
	t.Setenv("REGU_PASSWORD", password)
}

func TestReady_ValidCredentials(t *testing.T) {
	setCredEnv(t, "user", "pass")

	a := newApp()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/ready with valid creds: expected 200, got %d", rec.Code)
	}
	if atomic.LoadInt32(a.credsValid) != 1 {
		t.Error("expected credsValid == 1 after loading valid credentials")
	}
	if a.driver == nil {
		t.Error("expected non-nil AuthDriver when credentials are valid")
	}

	// Verify JSON response
	var resp health.ReadyResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if resp.Status != health.StatusOK {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
}

func TestReady_ValidCredentials_ReloadableDriver(t *testing.T) {
	setCredEnv(t, "user", "pass")

	a := newApp()

	if a.reloader == nil {
		t.Fatal("expected non-nil reloader when credentials are valid")
	}
	// driver should be the reloadable wrapper
	if _, ok := a.driver.(*auth.ReloadableDriver); !ok {
		t.Errorf("expected driver to be *auth.ReloadableDriver, got %T", a.driver)
	}
}

func TestReady_MissingCredentials(t *testing.T) {
	setCredEnv(t, "", "")

	a := newApp()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/ready without creds: expected 503, got %d", rec.Code)
	}
	if atomic.LoadInt32(a.credsValid) != 0 {
		t.Error("expected credsValid == 0 when credentials are missing")
	}
	if a.driver != nil {
		t.Error("expected nil AuthDriver when credentials are missing")
	}
	if a.reloader != nil {
		t.Error("expected nil reloader when credentials are missing")
	}

	// Verify JSON response with fail status
	var resp health.ReadyResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if resp.Status != health.StatusFail {
		t.Errorf("expected status 'fail', got %q", resp.Status)
	}
	// credentials check should be failed
	found := false
	for _, ch := range resp.Checks {
		if ch.Name == "credentials" && ch.Status == health.StatusFail {
			found = true
		}
	}
	if !found {
		t.Error("expected 'credentials' check with status 'fail'")
	}
}

func TestReady_PartialCredentials_MissingPassword(t *testing.T) {
	setCredEnv(t, "user", "")

	a := newApp()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/ready with partial creds: expected 503, got %d", rec.Code)
	}
}

func TestReady_PartialCredentials_MissingUsername(t *testing.T) {
	setCredEnv(t, "", "pass")

	a := newApp()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/ready with partial creds (no username): expected 503, got %d", rec.Code)
	}
}

func TestHealthz_AlwaysOK_WithCreds(t *testing.T) {
	setCredEnv(t, "user", "pass")

	a := newApp()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/healthz with creds: expected 200, got %d", rec.Code)
	}
	// Verify JSON response
	var resp health.HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if resp.Status != health.StatusOK {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
}

func TestHealthz_AlwaysOK_WithoutCreds(t *testing.T) {
	setCredEnv(t, "", "")

	a := newApp()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/healthz without creds: expected 200, got %d", rec.Code)
	}
}

func TestHealthz_JSONContentType(t *testing.T) {
	setCredEnv(t, "user", "pass")

	a := newApp()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}
}

func TestReady_JSONContentType(t *testing.T) {
	setCredEnv(t, "user", "pass")

	a := newApp()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}
}

func TestReady_CheckerPresent(t *testing.T) {
	setCredEnv(t, "user", "pass")

	a := newApp()

	if a.checker == nil {
		t.Fatal("expected non-nil health.Checker in app")
	}
}

func TestRotationInterval_Default(t *testing.T) {
	t.Setenv("REGU_ROTATION_INTERVAL_SEC", "")
	d := rotationInterval()
	if d != auth.DefaultRotationInterval {
		t.Errorf("expected %v, got %v", auth.DefaultRotationInterval, d)
	}
}

func TestRotationInterval_Custom(t *testing.T) {
	t.Setenv("REGU_ROTATION_INTERVAL_SEC", "60")
	d := rotationInterval()
	if d.Seconds() != 60 {
		t.Errorf("expected 60s, got %v", d)
	}
}

func TestRotationInterval_Invalid(t *testing.T) {
	t.Setenv("REGU_ROTATION_INTERVAL_SEC", "notanumber")
	d := rotationInterval()
	if d != auth.DefaultRotationInterval {
		t.Errorf("expected default for invalid input, got %v", d)
	}
}

// --- Diagnostics endpoint tests (Story 10.2) ---

func TestDiagnostics_EndpointRegistered(t *testing.T) {
	setCredEnv(t, "user", "pass")
	a := newApp()

	req := httptest.NewRequest(http.MethodGet, "/adapter/v1/diagnostics", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/adapter/v1/diagnostics: expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}

	var resp health.DiagnosticsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode diagnostics response: %v", err)
	}
	// With no queue/worker wired, expect zero values
	if resp.QueueDepth != 0 {
		t.Errorf("expected queue_depth 0, got %d", resp.QueueDepth)
	}
	if resp.WorkerCount != 0 {
		t.Errorf("expected worker_count 0, got %d", resp.WorkerCount)
	}
	if resp.LastHeartbeat != nil {
		t.Error("expected nil last_heartbeat with no worker pool")
	}
	if resp.Zones != nil {
		t.Error("expected nil zones with no worker pool")
	}
	if resp.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestDiagnostics_EndpointRegistered_NoCreds(t *testing.T) {
	setCredEnv(t, "", "")
	a := newApp()

	req := httptest.NewRequest(http.MethodGet, "/adapter/v1/diagnostics", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	// Diagnostics should work regardless of credential state
	if rec.Code != http.StatusOK {
		t.Errorf("/adapter/v1/diagnostics without creds: expected 200, got %d", rec.Code)
	}
}

func TestDiagnostics_DiagSrcPresent(t *testing.T) {
	setCredEnv(t, "user", "pass")
	a := newApp()

	if a.diagSrc == nil {
		t.Fatal("expected non-nil diagSrc in app")
	}
}

func TestDiagnosticsSource_NilSafe(t *testing.T) {
	src := &diagnosticsSource{}

	if src.QueueDepth() != 0 {
		t.Errorf("expected QueueDepth 0 with nil queue, got %d", src.QueueDepth())
	}
	if src.WorkerCount() != 0 {
		t.Errorf("expected WorkerCount 0 with nil pool, got %d", src.WorkerCount())
	}
	if !src.LastHeartbeat().IsZero() {
		t.Error("expected zero LastHeartbeat with nil pool")
	}
	if src.ZoneErrors() != nil {
		t.Error("expected nil ZoneErrors with nil pool")
	}
}

func TestDiagnosticsSource_WithQueue(t *testing.T) {
	q := queue.New()
	q.Enqueue(queue.Operation{ID: "op1"})
	q.Enqueue(queue.Operation{ID: "op2"})

	src := &diagnosticsSource{q: q}

	if depth := src.QueueDepth(); depth != 2 {
		t.Errorf("expected QueueDepth 2, got %d", depth)
	}
	// worker-related methods still nil-safe
	if src.WorkerCount() != 0 {
		t.Errorf("expected WorkerCount 0 with nil pool")
	}
	if !src.LastHeartbeat().IsZero() {
		t.Error("expected zero LastHeartbeat with nil pool")
	}
}

func TestDiagnosticsSource_ImplementsInterface(t *testing.T) {
	// Compile-time check that diagnosticsSource implements health.DiagnosticsSource
	var _ health.DiagnosticsSource = (*diagnosticsSource)(nil)
}

func TestDiagnostics_ResponseTimestamp(t *testing.T) {
	setCredEnv(t, "user", "pass")
	a := newApp()

	before := time.Now().UTC().Add(-time.Second)
	req := httptest.NewRequest(http.MethodGet, "/adapter/v1/diagnostics", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)
	after := time.Now().UTC().Add(time.Second)

	var resp health.DiagnosticsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Timestamp.Before(before) || resp.Timestamp.After(after) {
		t.Errorf("timestamp %v not in expected range [%v, %v]", resp.Timestamp, before, after)
	}
}
