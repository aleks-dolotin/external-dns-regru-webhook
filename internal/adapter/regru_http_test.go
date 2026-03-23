package adapter

import (
    "net/http"
    "net/http/httptest"
    "os"
    "testing"
)

func TestCreateRecordSuccess(t *testing.T) {
    // Start a local HTTP server to mock Reg.ru update_records
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(200)
        w.Write([]byte(`{"answer":{"domains":[]},"result":"success"}`))
    })
    srv := httptest.NewServer(handler)
    defer srv.Close()

    os.Setenv("REGRU_BASE_URL", srv.URL)
    adapter := NewHTTPAdapter()

    rec := &Record{Name: "www", Type: "A", Content: "1.2.3.4", TTL: 300}
    if err := adapter.CreateRecord("example.com", rec); err != nil {
        t.Fatalf("CreateRecord failed: %v", err)
    }
}

