package main

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/yourorg/externaldns-regru-sidecar/internal/auth"
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
