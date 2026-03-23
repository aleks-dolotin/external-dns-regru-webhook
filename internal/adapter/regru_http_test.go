package adapter

import (
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
