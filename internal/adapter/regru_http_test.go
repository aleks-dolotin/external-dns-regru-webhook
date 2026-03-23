package adapter

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateRecordSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"answer":{"domains":[]},"result":"success"}`))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	t.Setenv("REGRU_BASE_URL", srv.URL)
	adapter := NewHTTPAdapter(nil)

	rec := &Record{Name: "www", Type: "A", Content: "1.2.3.4", TTL: 300}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}
}

func TestCreateRecord_HTTP403_PermissionDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	t.Setenv("REGRU_BASE_URL", srv.URL)
	adapter := NewHTTPAdapter(nil)

	err := adapter.CreateRecord("example.com", &Record{Name: "www", Type: "A", Content: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

func TestCreateRecord_HTTP401_AuthFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	t.Setenv("REGRU_BASE_URL", srv.URL)
	adapter := NewHTTPAdapter(nil)

	err := adapter.CreateRecord("example.com", &Record{Name: "www", Type: "A", Content: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !errors.Is(err, ErrAuthenticationFailed) {
		t.Errorf("expected ErrAuthenticationFailed, got: %v", err)
	}
}

func TestCreateRecord_APIError_AccessDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"result":"error","error_code":"ACCESS_DENIED_TO_OBJECT","error_text":"No permission for this zone"}`))
	}))
	defer srv.Close()

	t.Setenv("REGRU_BASE_URL", srv.URL)
	adapter := NewHTTPAdapter(nil)

	err := adapter.CreateRecord("example.com", &Record{Name: "www", Type: "A", Content: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected error for ACCESS_DENIED_TO_OBJECT")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

func TestCreateRecord_APIError_InvalidAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"result":"error","error_code":"INVALID_AUTH","error_text":"Bad credentials"}`))
	}))
	defer srv.Close()

	t.Setenv("REGRU_BASE_URL", srv.URL)
	adapter := NewHTTPAdapter(nil)

	err := adapter.CreateRecord("example.com", &Record{Name: "www", Type: "A", Content: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected error for INVALID_AUTH")
	}
	if !errors.Is(err, ErrAuthenticationFailed) {
		t.Errorf("expected ErrAuthenticationFailed, got: %v", err)
	}
}

func TestFindRecord_HTTP403_PermissionDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	t.Setenv("REGRU_BASE_URL", srv.URL)
	adapter := NewHTTPAdapter(nil)

	_, err := adapter.FindRecord("example.com", "", "")
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

func TestCreateRecord_HTTP500_GenericError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	t.Setenv("REGRU_BASE_URL", srv.URL)
	adapter := NewHTTPAdapter(nil)

	err := adapter.CreateRecord("example.com", &Record{Name: "www", Type: "A", Content: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected error for 500")
	}
	// Should NOT be ErrPermissionDenied or ErrAuthenticationFailed
	if errors.Is(err, ErrPermissionDenied) || errors.Is(err, ErrAuthenticationFailed) {
		t.Errorf("500 should not map to permission/auth error, got: %v", err)
	}
}
