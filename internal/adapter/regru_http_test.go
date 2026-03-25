package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

// --- helpers ---

// parseFormRequest reads the form-encoded request body and returns the parsed input_data JSON.
func parseFormRequest(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	vals, err := url.ParseQuery(string(body))
	if err != nil {
		t.Fatalf("parse form: %v", err)
	}
	if vals.Get("input_format") != "json" {
		t.Fatalf("expected input_format=json, got %q", vals.Get("input_format"))
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(vals.Get("input_data")), &data); err != nil {
		t.Fatalf("unmarshal input_data: %v", err)
	}
	return data
}

// successJSON returns a standard Reg.ru success response with optional rrs.
func successJSON(serviceID string, rrs string) string {
	rrsField := ""
	if rrs != "" {
		rrsField = `,"rrs":` + rrs
	}
	sid := `"` + serviceID + `"`
	if serviceID == "" {
		sid = `"0"`
	}
	return `{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","service_id":` + sid + rrsField + `}]}}`
}

// emptyZoneJSON returns a success response with empty rrs.
func emptyZoneJSON() string {
	return successJSON("0", "[]")
}

// routingHandler creates a test HTTP handler that routes by URL path suffix.
// Keys should be path suffixes like "/zone/add_alias", "/zone/remove_record", etc.
func routingHandler(t *testing.T, routes map[string]http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		for suffix, handler := range routes {
			if strings.HasSuffix(r.URL.Path, suffix) {
				handler(w, r)
				return
			}
		}
		t.Errorf("unexpected request path: %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func newTestAdapter(t *testing.T, handler http.HandlerFunc) *HTTPAdapter {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("REGRU_BASE_URL", srv.URL)
	return NewHTTPAdapter(nil)
}

// --- FindRecord tests ---

func TestFindRecord_RecordFound(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		parseFormRequest(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(successJSON("12345", `[
			{"subname":"www","rectype":"A","content":"1.2.3.4","prio":"0"},
			{"subname":"mail","rectype":"CNAME","content":"mail.example.com","prio":"0"}
		]`)))
	})

	rec, err := adapter.FindRecord("example.com", "www", "A")
	if err != nil {
		t.Fatalf("FindRecord failed: %v", err)
	}
	if rec == nil {
		t.Fatal("expected record, got nil")
	}
	if rec.Name != "www" || rec.Type != "A" || rec.Content != "1.2.3.4" {
		t.Errorf("unexpected record: %+v", rec)
	}
	if rec.ID != "12345" {
		t.Errorf("expected ID=12345, got %q", rec.ID)
	}
}

func TestFindRecord_RecordNotFound(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(successJSON("12345", `[{"subname":"other","rectype":"A","content":"5.6.7.8","prio":"0"}]`)))
	})
	rec, err := adapter.FindRecord("example.com", "www", "A")
	if err != nil {
		t.Fatalf("FindRecord failed: %v", err)
	}
	if rec != nil {
		t.Fatalf("expected nil, got %+v", rec)
	}
}

func TestFindRecord_APIError(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"error","error_code":"ACCESS_DENIED_TO_OBJECT","error_text":"No permission"}`))
	})
	_, err := adapter.FindRecord("example.com", "www", "A")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

func TestFindRecord_HTTP403_PermissionDenied(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	_, err := adapter.FindRecord("example.com", "www", "A")
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

func TestFindRecord_EmptyRrs(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(emptyZoneJSON()))
	})
	rec, err := adapter.FindRecord("example.com", "www", "A")
	if err != nil {
		t.Fatalf("FindRecord failed: %v", err)
	}
	if rec != nil {
		t.Fatalf("expected nil for empty rrs, got %+v", rec)
	}
}

// --- CreateRecord tests (now uses zone/add_* client endpoints) ---

func TestCreateRecord_Success_A(t *testing.T) {
	var callCount int32
	adapter := newTestAdapter(t, routingHandler(t, map[string]http.HandlerFunc{
		"/zone/get_resource_records": func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&callCount, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyZoneJSON()))
		},
		"/zone/add_alias": func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&callCount, 1)
			data := parseFormRequest(t, r)
			if data["subdomain"] != "www" {
				t.Errorf("expected subdomain=www, got %v", data["subdomain"])
			}
			if data["ipaddr"] != "1.2.3.4" {
				t.Errorf("expected ipaddr=1.2.3.4, got %v", data["ipaddr"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(successJSON("99999", "")))
		},
	}))

	rec := &Record{Name: "www", Type: "A", Content: "1.2.3.4", TTL: 300}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}
	if rec.ID != "99999" {
		t.Errorf("expected ID=99999, got %q", rec.ID)
	}
	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("expected 2 calls (find + add), got %d", callCount)
	}
}

func TestCreateRecord_Success_AAAA(t *testing.T) {
	adapter := newTestAdapter(t, routingHandler(t, map[string]http.HandlerFunc{
		"/zone/get_resource_records": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyZoneJSON()))
		},
		"/zone/add_aaaa": func(w http.ResponseWriter, r *http.Request) {
			data := parseFormRequest(t, r)
			if data["ipaddr"] != "::1" {
				t.Errorf("expected ipaddr=::1, got %v", data["ipaddr"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(successJSON("10001", "")))
		},
	}))

	rec := &Record{Name: "ipv6", Type: "AAAA", Content: "::1"}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord AAAA failed: %v", err)
	}
	if rec.ID != "10001" {
		t.Errorf("expected ID=10001, got %q", rec.ID)
	}
}

func TestCreateRecord_Success_CNAME(t *testing.T) {
	adapter := newTestAdapter(t, routingHandler(t, map[string]http.HandlerFunc{
		"/zone/get_resource_records": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyZoneJSON()))
		},
		"/zone/add_cname": func(w http.ResponseWriter, r *http.Request) {
			data := parseFormRequest(t, r)
			if data["canonical_name"] != "target.example.com" {
				t.Errorf("expected canonical_name=target.example.com, got %v", data["canonical_name"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(successJSON("10002", "")))
		},
	}))

	rec := &Record{Name: "alias", Type: "CNAME", Content: "target.example.com"}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord CNAME failed: %v", err)
	}
	if rec.ID != "10002" {
		t.Errorf("expected ID=10002, got %q", rec.ID)
	}
}

func TestCreateRecord_Success_TXT(t *testing.T) {
	adapter := newTestAdapter(t, routingHandler(t, map[string]http.HandlerFunc{
		"/zone/get_resource_records": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyZoneJSON()))
		},
		"/zone/add_txt": func(w http.ResponseWriter, r *http.Request) {
			data := parseFormRequest(t, r)
			if data["text"] != "v=spf1 include:example.com ~all" {
				t.Errorf("expected text value, got %v", data["text"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(successJSON("10003", "")))
		},
	}))

	rec := &Record{Name: "@", Type: "TXT", Content: "v=spf1 include:example.com ~all"}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord TXT failed: %v", err)
	}
	if rec.ID != "10003" {
		t.Errorf("expected ID=10003, got %q", rec.ID)
	}
}

func TestCreateRecord_Idempotency_SkipsWhenExists(t *testing.T) {
	var callCount int32
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(successJSON("77777", `[{"subname":"www","rectype":"A","content":"1.2.3.4","prio":"0"}]`)))
	})

	rec := &Record{Name: "www", Type: "A", Content: "1.2.3.4", TTL: 300}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord (idempotent) failed: %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 API call (FindRecord only), got %d", callCount)
	}
	if rec.ID != "77777" {
		t.Errorf("expected ID=77777, got %q", rec.ID)
	}
}

func TestCreateRecord_APIError(t *testing.T) {
	adapter := newTestAdapter(t, routingHandler(t, map[string]http.HandlerFunc{
		"/zone/get_resource_records": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyZoneJSON()))
		},
		"/zone/add_alias": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":"error","error_code":"ACCESS_DENIED_TO_OBJECT","error_text":"No permission for this zone"}`))
		},
	}))

	err := adapter.CreateRecord("example.com", &Record{Name: "www", Type: "A", Content: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

func TestCreateRecord_DomainLevelError(t *testing.T) {
	adapter := newTestAdapter(t, routingHandler(t, map[string]http.HandlerFunc{
		"/zone/get_resource_records": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyZoneJSON()))
		},
		"/zone/add_alias": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"error","error_code":"INVALID_ACTION","error_text":"Invalid action"}]}}`))
		},
	}))

	err := adapter.CreateRecord("example.com", &Record{Name: "www", Type: "A", Content: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected domain-level error")
	}
}

func TestCreateRecord_HTTP403_PermissionDenied(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	err := adapter.CreateRecord("example.com", &Record{Name: "www", Type: "A", Content: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

func TestCreateRecord_HTTP401_AuthFailed(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	err := adapter.CreateRecord("example.com", &Record{Name: "www", Type: "A", Content: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !errors.Is(err, ErrAuthenticationFailed) {
		t.Errorf("expected ErrAuthenticationFailed, got: %v", err)
	}
}

func TestCreateRecord_HTTP500_GenericError(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	})
	err := adapter.CreateRecord("example.com", &Record{Name: "www", Type: "A", Content: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestCreateRecord_APIError_InvalidAuth(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"error","error_code":"INVALID_AUTH","error_text":"Bad credentials"}`))
	})
	err := adapter.CreateRecord("example.com", &Record{Name: "www", Type: "A", Content: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected error for INVALID_AUTH")
	}
	if !errors.Is(err, ErrAuthenticationFailed) {
		t.Errorf("expected ErrAuthenticationFailed, got: %v", err)
	}
}

func TestCreateRecord_UnsupportedRecordType(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(emptyZoneJSON()))
	})
	err := adapter.CreateRecord("example.com", &Record{Name: "mx", Type: "MX", Content: "mail.example.com"})
	if err == nil {
		t.Fatal("expected error for unsupported record type MX")
	}
	if !errors.Is(err, ErrUnsupportedRecordType) {
		t.Errorf("expected ErrUnsupportedRecordType, got: %v", err)
	}
}

func TestCreateRecord_UsesFormEncoding(t *testing.T) {
	adapter := newTestAdapter(t, routingHandler(t, map[string]http.HandlerFunc{
		"/zone/get_resource_records": func(w http.ResponseWriter, r *http.Request) {
			ct := r.Header.Get("Content-Type")
			if ct != "application/x-www-form-urlencoded" {
				t.Errorf("expected form-urlencoded, got %q", ct)
			}
			parseFormRequest(t, r)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyZoneJSON()))
		},
		"/zone/add_alias": func(w http.ResponseWriter, r *http.Request) {
			ct := r.Header.Get("Content-Type")
			if ct != "application/x-www-form-urlencoded" {
				t.Errorf("expected form-urlencoded, got %q", ct)
			}
			parseFormRequest(t, r)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(successJSON("1", "")))
		},
	}))

	rec := &Record{Name: "www", Type: "A", Content: "1.2.3.4"}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}
}

func TestCreateRecord_UsesCorrectEndpoint(t *testing.T) {
	tests := []struct {
		recType  string
		endpoint string
	}{
		{"A", "/zone/add_alias"},
		{"AAAA", "/zone/add_aaaa"},
		{"CNAME", "/zone/add_cname"},
		{"TXT", "/zone/add_txt"},
	}
	for _, tt := range tests {
		t.Run(tt.recType, func(t *testing.T) {
			var hitEndpoint string
			adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
				hitEndpoint = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/zone/get_resource_records") {
					_, _ = w.Write([]byte(emptyZoneJSON()))
				} else {
					_, _ = w.Write([]byte(successJSON("1", "")))
				}
			})
			rec := &Record{Name: "test", Type: tt.recType, Content: "value"}
			_ = adapter.CreateRecord("example.com", rec)
			if !strings.HasSuffix(hitEndpoint, tt.endpoint) {
				t.Errorf("expected endpoint %s, got %s", tt.endpoint, hitEndpoint)
			}
		})
	}
}

// --- BulkUpdate tests (now uses sequential client endpoints) ---

func TestBulkUpdate_Success_MultipleActions(t *testing.T) {
	var paths []string
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/zone/get_resource_records") {
			_, _ = w.Write([]byte(emptyZoneJSON()))
		} else {
			_, _ = w.Write([]byte(successJSON("123", "")))
		}
	})

	actions := []BulkAction{
		{Action: "add_alias", Subdomain: "www", Content: "1.2.3.4", RecType: "A"},
		{Action: "add_cname", Subdomain: "cdn", Content: "cdn.example.com", RecType: "CNAME"},
		{Action: "remove_record", Subdomain: "old", Content: "5.6.7.8", RecType: "A"},
	}
	if err := adapter.BulkUpdate("example.com", actions); err != nil {
		t.Fatalf("BulkUpdate failed: %v", err)
	}
	// Expect: get_resource_records (idempotency for www) + add_alias + get_resource_records (idempotency for cdn) + add_cname + remove_record
	if len(paths) < 3 {
		t.Errorf("expected at least 3 API calls, got %d: %v", len(paths), paths)
	}
}

func TestBulkUpdate_EmptyActions(t *testing.T) {
	var callCount int32
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(200)
	})

	if err := adapter.BulkUpdate("example.com", nil); err != nil {
		t.Fatalf("BulkUpdate with nil should not error: %v", err)
	}
	if err := adapter.BulkUpdate("example.com", []BulkAction{}); err != nil {
		t.Fatalf("BulkUpdate with empty should not error: %v", err)
	}
	if atomic.LoadInt32(&callCount) != 0 {
		t.Errorf("expected 0 API calls for empty batch, got %d", callCount)
	}
}

func TestBulkUpdate_PartialFailure(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/zone/get_resource_records") {
			_, _ = w.Write([]byte(emptyZoneJSON()))
		} else {
			_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"error","error_code":"INVALID_ACTION","error_text":"Invalid"}]}}`))
		}
	})

	actions := []BulkAction{
		{Action: "add_alias", Subdomain: "www", Content: "1.2.3.4", RecType: "A"},
	}
	err := adapter.BulkUpdate("example.com", actions)
	if err == nil {
		t.Fatal("expected error for partial failure")
	}
}

func TestBulkUpdate_APIError(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"error","error_code":"ACCESS_DENIED_TO_OBJECT","error_text":"No access"}`))
	})

	actions := []BulkAction{
		{Action: "remove_record", Subdomain: "www", Content: "1.2.3.4", RecType: "A"},
	}
	err := adapter.BulkUpdate("example.com", actions)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

// --- DeleteRecord tests (Story 1.3) ---

func TestDeleteRecord_Success(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		data := parseFormRequest(t, r)
		if data["subdomain"] != "www" {
			t.Errorf("expected subdomain=www, got %v", data["subdomain"])
		}
		if data["record_type"] != "A" {
			t.Errorf("expected record_type=A, got %v", data["record_type"])
		}
		if data["content"] != "1.2.3.4" {
			t.Errorf("expected content=1.2.3.4, got %v", data["content"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(successJSON("12345", "")))
	})

	if err := adapter.DeleteRecord("example.com", "www:A:1.2.3.4"); err != nil {
		t.Fatalf("DeleteRecord failed: %v", err)
	}
}

func TestDeleteRecord_WithoutContent(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		data := parseFormRequest(t, r)
		if data["subdomain"] != "old" {
			t.Errorf("expected subdomain=old, got %v", data["subdomain"])
		}
		if data["record_type"] != "CNAME" {
			t.Errorf("expected record_type=CNAME, got %v", data["record_type"])
		}
		if _, ok := data["content"]; ok {
			t.Error("expected no content field")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(successJSON("0", "")))
	})

	if err := adapter.DeleteRecord("example.com", "old:CNAME"); err != nil {
		t.Fatalf("DeleteRecord without content failed: %v", err)
	}
}

func TestDeleteRecord_InvalidID(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	err := adapter.DeleteRecord("example.com", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid id format")
	}
}

func TestDeleteRecord_APIError(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"error","error_code":"ACCESS_DENIED_TO_OBJECT","error_text":"No permission"}`))
	})
	err := adapter.DeleteRecord("example.com", "www:A:1.2.3.4")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

func TestDeleteRecord_UsesRemoveRecordEndpoint(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/zone/remove_record") {
			t.Errorf("expected /zone/remove_record, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(successJSON("0", "")))
	})
	if err := adapter.DeleteRecord("example.com", "www:A:1.2.3.4"); err != nil {
		t.Fatalf("DeleteRecord failed: %v", err)
	}
}

// --- UpdateRecord tests (now uses remove_record + add_*) ---

func TestUpdateRecord_Success_ContentChanged(t *testing.T) {
	var callPaths []string
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasSuffix(r.URL.Path, "/zone/get_resource_records"):
			// Return existing record with old IP (for initial find and idempotency check in CreateRecord)
			if len(callPaths) <= 1 {
				_, _ = w.Write([]byte(successJSON("12345", `[{"subname":"www","rectype":"A","content":"1.1.1.1","prio":"0"}]`)))
			} else {
				// After delete, return empty (for CreateRecord's idempotency check)
				_, _ = w.Write([]byte(emptyZoneJSON()))
			}
		case strings.HasSuffix(r.URL.Path, "/zone/remove_record"):
			_, _ = w.Write([]byte(successJSON("12345", "")))
		case strings.HasSuffix(r.URL.Path, "/zone/add_alias"):
			_, _ = w.Write([]byte(successJSON("12345", "")))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	rec := &Record{Name: "www", Type: "A", Content: "2.2.2.2"}
	if err := adapter.UpdateRecord("example.com", rec); err != nil {
		t.Fatalf("UpdateRecord failed: %v", err)
	}
	if rec.ID != "12345" {
		t.Errorf("expected ID=12345, got %q", rec.ID)
	}
	// Should have: get_resource_records, remove_record, get_resource_records (idempotency), add_alias
	if len(callPaths) < 3 {
		t.Errorf("expected at least 3 calls, got %d: %v", len(callPaths), callPaths)
	}
}

func TestUpdateRecord_NoChange(t *testing.T) {
	var callCount int32
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(successJSON("12345", `[{"subname":"www","rectype":"A","content":"1.2.3.4","prio":"0"}]`)))
	})

	rec := &Record{Name: "www", Type: "A", Content: "1.2.3.4"}
	if err := adapter.UpdateRecord("example.com", rec); err != nil {
		t.Fatalf("UpdateRecord (no-op) failed: %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call (find only), got %d", callCount)
	}
	if rec.ID != "12345" {
		t.Errorf("expected ID=12345, got %q", rec.ID)
	}
}

func TestUpdateRecord_NotExists_FallsBackToCreate(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/zone/get_resource_records") {
			_, _ = w.Write([]byte(emptyZoneJSON()))
		} else {
			_, _ = w.Write([]byte(successJSON("99999", "")))
		}
	})

	rec := &Record{Name: "new", Type: "A", Content: "3.3.3.3"}
	if err := adapter.UpdateRecord("example.com", rec); err != nil {
		t.Fatalf("UpdateRecord (fallback to create) failed: %v", err)
	}
	if rec.ID != "99999" {
		t.Errorf("expected ID=99999, got %q", rec.ID)
	}
}

func TestUpdateRecord_NoDuplicates(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/zone/get_resource_records"):
			_, _ = w.Write([]byte(successJSON("100", `[{"subname":"app","rectype":"CNAME","content":"old.example.com","prio":"0"}]`)))
		case strings.HasSuffix(r.URL.Path, "/zone/remove_record"):
			_, _ = w.Write([]byte(successJSON("100", "")))
		case strings.HasSuffix(r.URL.Path, "/zone/add_cname"):
			_, _ = w.Write([]byte(successJSON("100", "")))
		}
	})

	rec := &Record{Name: "app", Type: "CNAME", Content: "new.example.com"}
	if err := adapter.UpdateRecord("example.com", rec); err != nil {
		t.Fatalf("UpdateRecord failed: %v", err)
	}
}

// --- Cache integration tests (Story 1.5) ---

func newTestAdapterWithCache(t *testing.T, handler http.HandlerFunc) (*HTTPAdapter, *ExternalIDCache) {
	t.Helper()
	cache := NewExternalIDCache()
	adapter := newTestAdapter(t, handler)
	adapter.SetCache(cache)
	return adapter, cache
}

func TestCacheIntegration_FindRecord_PopulatesCache(t *testing.T) {
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(successJSON("12345", `[{"subname":"www","rectype":"A","content":"1.2.3.4","prio":"0"}]`)))
	})

	rec, err := adapter.FindRecord("example.com", "www", "A")
	if err != nil {
		t.Fatalf("FindRecord: %v", err)
	}
	if rec == nil {
		t.Fatal("expected record")
	}
	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}
	if got := cache.Get(key); got != "12345" {
		t.Errorf("expected cache hit 12345, got %q", got)
	}
}

func TestCacheIntegration_CreateRecord_PopulatesCache(t *testing.T) {
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/zone/get_resource_records") {
			_, _ = w.Write([]byte(emptyZoneJSON()))
		} else {
			_, _ = w.Write([]byte(successJSON("99999", "")))
		}
	})

	rec := &Record{Name: "www", Type: "A", Content: "1.2.3.4"}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}
	if got := cache.Get(key); got != "99999" {
		t.Errorf("expected cache hit 99999, got %q", got)
	}
}

func TestCacheIntegration_DeleteRecord_EvictsCache(t *testing.T) {
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(successJSON("0", "")))
	})
	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}
	cache.Set(key, "12345")

	if err := adapter.DeleteRecord("example.com", "www:A:1.2.3.4"); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}
	if got := cache.Get(key); got != "" {
		t.Errorf("expected cache eviction, got %q", got)
	}
}

func TestCacheIntegration_LookupExternalID_CacheHit(t *testing.T) {
	var callCount int32
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(200)
	})
	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}
	cache.Set(key, "cached-id")

	id, err := adapter.LookupExternalID("example.com", "www", "A")
	if err != nil {
		t.Fatalf("LookupExternalID: %v", err)
	}
	if id != "cached-id" {
		t.Errorf("expected cached-id, got %q", id)
	}
	if atomic.LoadInt32(&callCount) != 0 {
		t.Errorf("expected 0 API calls (cache hit), got %d", callCount)
	}
}

func TestCacheIntegration_LookupExternalID_CacheMiss_FallsBackToAPI(t *testing.T) {
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(successJSON("55555", `[{"subname":"www","rectype":"A","content":"1.2.3.4","prio":"0"}]`)))
	})

	id, err := adapter.LookupExternalID("example.com", "www", "A")
	if err != nil {
		t.Fatalf("LookupExternalID: %v", err)
	}
	if id != "55555" {
		t.Errorf("expected 55555, got %q", id)
	}
	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}
	if got := cache.Get(key); got != "55555" {
		t.Errorf("expected cache populated, got %q", got)
	}
}

func TestCacheIntegration_ReconcileCache_EvictsStale(t *testing.T) {
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(emptyZoneJSON()))
	})
	key := CacheKey{Zone: "example.com", Name: "gone", RecType: "A"}
	cache.Set(key, "stale-id")

	adapter.ReconcileCache(context.Background())
	if got := cache.Get(key); got != "" {
		t.Errorf("expected stale entry evicted, got %q", got)
	}
}

func TestCacheIntegration_ReconcileCache_OneRequestPerZone(t *testing.T) {
	var callCount int32
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		data := parseFormRequest(t, r)
		domains := data["domains"].([]interface{})
		zone := domains[0].(map[string]interface{})["dname"].(string)
		w.Header().Set("Content-Type", "application/json")
		if zone == "example.com" {
			_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","rrs":[
				{"subname":"www","rectype":"A","content":"1.2.3.4","prio":"0"},
				{"subname":"mail","rectype":"CNAME","content":"mail.example.com","prio":"0"}
			]}]}}`))
		} else {
			_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"` + zone + `","result":"success","rrs":[
				{"subname":"app","rectype":"A","content":"5.6.7.8","prio":"0"}
			]}]}}`))
		}
	})

	cache.Set(CacheKey{Zone: "example.com", Name: "www", RecType: "A"}, "id1")
	cache.Set(CacheKey{Zone: "example.com", Name: "mail", RecType: "CNAME"}, "id2")
	cache.Set(CacheKey{Zone: "example.com", Name: "gone", RecType: "TXT"}, "id3")
	cache.Set(CacheKey{Zone: "other.com", Name: "app", RecType: "A"}, "id4")

	adapter.ReconcileCache(context.Background())

	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("expected 2 API calls (1 per zone), got %d", callCount)
	}
	if got := cache.Get(CacheKey{Zone: "example.com", Name: "www", RecType: "A"}); got != "id1" {
		t.Errorf("live entry should remain, got %q", got)
	}
	if got := cache.Get(CacheKey{Zone: "example.com", Name: "gone", RecType: "TXT"}); got != "" {
		t.Errorf("stale entry should be evicted, got %q", got)
	}
}
