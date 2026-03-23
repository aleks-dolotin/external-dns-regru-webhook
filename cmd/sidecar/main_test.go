package main

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// setCredEnv uses t.Setenv for automatic cleanup after test.
func setCredEnv(t *testing.T, username, password string) {
	t.Helper()
	t.Setenv("REGU_USERNAME", username)
	t.Setenv("REGU_PASSWORD", password)
}

func TestReady_ValidCredentials(t *testing.T) {
	setCredEnv(t, "user", "pass")

	a := newApp() // exercises real EnvSecretProvider + atomic wiring

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
