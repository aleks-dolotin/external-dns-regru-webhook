package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthzHandler_AlwaysOK(t *testing.T) {
	c := NewChecker()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	c.HealthzHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != StatusOK {
		t.Errorf("expected status ok, got %s", resp.Status)
	}
	if resp.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestReadyHandler_AllChecksPass(t *testing.T) {
	c := NewChecker(
		ReadyCheck{Name: "creds", Check: func() (bool, string) { return true, "" }},
		ReadyCheck{Name: "config", Check: func() (bool, string) { return true, "loaded" }},
	)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	c.ReadyHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp ReadyResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != StatusOK {
		t.Errorf("expected status ok, got %s", resp.Status)
	}
	if len(resp.Checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(resp.Checks))
	}
	for _, ch := range resp.Checks {
		if ch.Status != StatusOK {
			t.Errorf("check %s: expected ok, got %s", ch.Name, ch.Status)
		}
	}
}

func TestReadyHandler_OneCheckFails(t *testing.T) {
	c := NewChecker(
		ReadyCheck{Name: "creds", Check: func() (bool, string) { return true, "" }},
		ReadyCheck{Name: "config", Check: func() (bool, string) { return false, "config not loaded" }},
	)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	c.ReadyHandler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
	var resp ReadyResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != StatusFail {
		t.Errorf("expected status fail, got %s", resp.Status)
	}
	// creds should be ok, config should be fail
	if resp.Checks[0].Status != StatusOK {
		t.Errorf("creds check: expected ok, got %s", resp.Checks[0].Status)
	}
	if resp.Checks[1].Status != StatusFail {
		t.Errorf("config check: expected fail, got %s", resp.Checks[1].Status)
	}
	if resp.Checks[1].Message != "config not loaded" {
		t.Errorf("expected message 'config not loaded', got %q", resp.Checks[1].Message)
	}
}

func TestReadyHandler_AllChecksFail(t *testing.T) {
	c := NewChecker(
		ReadyCheck{Name: "creds", Check: func() (bool, string) { return false, "missing" }},
	)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	c.ReadyHandler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}

func TestReadyHandler_NoChecks(t *testing.T) {
	c := NewChecker()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	c.ReadyHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with no checks, got %d", rec.Code)
	}
	var resp ReadyResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != StatusOK {
		t.Errorf("expected ok with no checks, got %s", resp.Status)
	}
}

func TestAddCheck(t *testing.T) {
	c := NewChecker()
	c.AddCheck(ReadyCheck{Name: "dynamic", Check: func() (bool, string) { return true, "" }})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	c.ReadyHandler(rec, req)

	var resp ReadyResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Checks) != 1 {
		t.Fatalf("expected 1 check after AddCheck, got %d", len(resp.Checks))
	}
	if resp.Checks[0].Name != "dynamic" {
		t.Errorf("expected check name 'dynamic', got %q", resp.Checks[0].Name)
	}
}
