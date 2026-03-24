package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/safemode"
)

// --- Story 8.3: Safe-mode HTTP endpoint tests ---

// --- Subtask 5.1: Test enable/disable toggle ---

func TestSafeMode_Enable(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	a := newApp()

	req := httptest.NewRequest(http.MethodPost, "/adapter/v1/safe-mode?enabled=true", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var st safemode.Status
	if err := json.NewDecoder(rec.Body).Decode(&st); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !st.Enabled {
		t.Error("expected enabled=true")
	}
	if st.Since == nil {
		t.Error("expected non-nil since")
	}
}

func TestSafeMode_Disable(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	a := newApp()

	// Enable first.
	a.safeMode.Enable()

	req := httptest.NewRequest(http.MethodPost, "/adapter/v1/safe-mode?enabled=false", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var st safemode.Status
	_ = json.NewDecoder(rec.Body).Decode(&st)
	if st.Enabled {
		t.Error("expected enabled=false after disable")
	}
}

func TestSafeMode_InvalidParam(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	a := newApp()

	req := httptest.NewRequest(http.MethodPost, "/adapter/v1/safe-mode?enabled=maybe", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestSafeMode_MissingParam(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	a := newApp()

	req := httptest.NewRequest(http.MethodPost, "/adapter/v1/safe-mode", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// --- Subtask 5.4: Test status endpoint returns correct counts ---

func TestSafeMode_StatusEndpoint(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	a := newApp()

	// Enable and increment suppressed count.
	a.safeMode.Enable()
	a.safeMode.IncrementSuppressed()
	a.safeMode.IncrementSuppressed()

	req := httptest.NewRequest(http.MethodGet, "/adapter/v1/safe-mode", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var st safemode.Status
	if err := json.NewDecoder(rec.Body).Decode(&st); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !st.Enabled {
		t.Error("expected enabled=true")
	}
	if st.SuppressedCount != 2 {
		t.Errorf("expected suppressed_count=2, got %d", st.SuppressedCount)
	}
}

func TestSafeMode_StatusDisabled(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	a := newApp()

	req := httptest.NewRequest(http.MethodGet, "/adapter/v1/safe-mode", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var st safemode.Status
	_ = json.NewDecoder(rec.Body).Decode(&st)
	if st.Enabled {
		t.Error("expected enabled=false by default")
	}
	if st.SuppressedCount != 0 {
		t.Errorf("expected suppressed_count=0, got %d", st.SuppressedCount)
	}
}

// --- Subtask 5.5: Test env var startup activation (AC #5) ---

func TestSafeMode_EnvVarStartup(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	t.Setenv("REGADAPTER_SAFE_MODE", "true")

	a := newApp()

	if !a.safeMode.IsEnabled() {
		t.Error("expected safe-mode enabled from REGADAPTER_SAFE_MODE=true")
	}
}

func TestSafeMode_EnvVarNotSet(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	t.Setenv("REGADAPTER_SAFE_MODE", "")

	a := newApp()

	if a.safeMode.IsEnabled() {
		t.Error("expected safe-mode disabled when REGADAPTER_SAFE_MODE is not set")
	}
}

// --- Test JSON content type ---

func TestSafeMode_JSONContentType(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	a := newApp()

	req := httptest.NewRequest(http.MethodGet, "/adapter/v1/safe-mode", nil)
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}
}

// --- Test safe-mode wired to pool ---

func TestSafeMode_WiredToPool(t *testing.T) {
	setCredEnv(t, "user", "pass")
	t.Setenv("REGADAPTER_MAPPINGS_PATH", "/nonexistent/path.yaml")
	a := newApp()

	if a.safeMode == nil {
		t.Fatal("expected non-nil safeMode in app")
	}
}
