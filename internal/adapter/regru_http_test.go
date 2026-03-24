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
		parseFormRequest(t, r) // validate form encoding
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"result": "success",
			"answer": {
				"domains": [{
					"dname": "example.com",
					"result": "success",
					"service_id": "12345",
					"rrs": [
						{"subname": "www", "rectype": "A", "content": "1.2.3.4", "prio": "0"},
						{"subname": "mail", "rectype": "CNAME", "content": "mail.example.com", "prio": "0"}
					]
				}]
			}
		}`))
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
		_, _ = w.Write([]byte(`{
			"result": "success",
			"answer": {
				"domains": [{
					"dname": "example.com",
					"result": "success",
					"service_id": "12345",
					"rrs": [
						{"subname": "other", "rectype": "A", "content": "5.6.7.8", "prio": "0"}
					]
				}]
			}
		}`))
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
		_, _ = w.Write([]byte(`{
			"result": "success",
			"answer": {"domains": [{"dname": "example.com", "result": "success", "rrs": []}]}
		}`))
	})

	rec, err := adapter.FindRecord("example.com", "www", "A")
	if err != nil {
		t.Fatalf("FindRecord failed: %v", err)
	}
	if rec != nil {
		t.Fatalf("expected nil for empty rrs, got %+v", rec)
	}
}

// --- CreateRecord tests ---

func TestCreateRecord_Success_A(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		data := parseFormRequest(t, r)
		w.Header().Set("Content-Type", "application/json")

		// First call is FindRecord (idempotency check) → return empty rrs.
		if callCount == 1 {
			_, _ = w.Write([]byte(`{
				"result": "success",
				"answer": {"domains": [{"dname": "example.com", "result": "success", "rrs": []}]}
			}`))
			return
		}

		// Second call is CreateRecord → validate action_list payload.
		domains, ok := data["domains"].([]interface{})
		if !ok || len(domains) == 0 {
			t.Fatal("missing domains in create request")
		}
		dom := domains[0].(map[string]interface{})
		actions := dom["action_list"].([]interface{})
		act := actions[0].(map[string]interface{})
		if act["action"] != "add_alias" {
			t.Errorf("expected action=add_alias, got %v", act["action"])
		}
		if act["ipaddr"] != "1.2.3.4" {
			t.Errorf("expected ipaddr=1.2.3.4, got %v", act["ipaddr"])
		}

		_, _ = w.Write([]byte(`{
			"result": "success",
			"answer": {"domains": [{"dname": "example.com", "result": "success", "service_id": "99999"}]}
		}`))
	})

	rec := &Record{Name: "www", Type: "A", Content: "1.2.3.4", TTL: 300}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}
	if rec.ID != "99999" {
		t.Errorf("expected ID=99999 (service_id), got %q", rec.ID)
	}
}

func TestCreateRecord_Success_AAAA(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		data := parseFormRequest(t, r)
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","rrs":[]}]}}`))
			return
		}

		domains := data["domains"].([]interface{})
		act := domains[0].(map[string]interface{})["action_list"].([]interface{})[0].(map[string]interface{})
		if act["action"] != "add_aaaa" {
			t.Errorf("expected action=add_aaaa, got %v", act["action"])
		}
		if act["ipaddr"] != "::1" {
			t.Errorf("expected ipaddr=::1, got %v", act["ipaddr"])
		}

		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","service_id":"10001"}]}}`))
	})

	rec := &Record{Name: "ipv6", Type: "AAAA", Content: "::1"}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord AAAA failed: %v", err)
	}
	if rec.ID != "10001" {
		t.Errorf("expected ID=10001, got %q", rec.ID)
	}
}

func TestCreateRecord_Success_CNAME(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		data := parseFormRequest(t, r)
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","rrs":[]}]}}`))
			return
		}

		domains := data["domains"].([]interface{})
		act := domains[0].(map[string]interface{})["action_list"].([]interface{})[0].(map[string]interface{})
		if act["action"] != "add_cname" {
			t.Errorf("expected action=add_cname, got %v", act["action"])
		}
		if act["canonical_name"] != "target.example.com" {
			t.Errorf("expected canonical_name=target.example.com, got %v", act["canonical_name"])
		}

		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","service_id":"10002"}]}}`))
	})

	rec := &Record{Name: "alias", Type: "CNAME", Content: "target.example.com"}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord CNAME failed: %v", err)
	}
	if rec.ID != "10002" {
		t.Errorf("expected ID=10002, got %q", rec.ID)
	}
}

func TestCreateRecord_Success_TXT(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		data := parseFormRequest(t, r)
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","rrs":[]}]}}`))
			return
		}

		domains := data["domains"].([]interface{})
		act := domains[0].(map[string]interface{})["action_list"].([]interface{})[0].(map[string]interface{})
		if act["action"] != "add_txt" {
			t.Errorf("expected action=add_txt, got %v", act["action"])
		}
		if act["text"] != "v=spf1 include:example.com ~all" {
			t.Errorf("expected text value, got %v", act["text"])
		}

		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","service_id":"10003"}]}}`))
	})

	rec := &Record{Name: "@", Type: "TXT", Content: "v=spf1 include:example.com ~all"}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord TXT failed: %v", err)
	}
	if rec.ID != "10003" {
		t.Errorf("expected ID=10003, got %q", rec.ID)
	}
}

func TestCreateRecord_Idempotency_SkipsWhenExists(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		// FindRecord returns existing record — CreateRecord should NOT send a second request.
		_, _ = w.Write([]byte(`{
			"result": "success",
			"answer": {"domains": [{"dname": "example.com", "result": "success", "service_id": "77777",
				"rrs": [{"subname": "www", "rectype": "A", "content": "1.2.3.4", "prio": "0"}]
			}]}
		}`))
	})

	rec := &Record{Name: "www", Type: "A", Content: "1.2.3.4", TTL: 300}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord (idempotent) failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected exactly 1 API call (FindRecord only), got %d", callCount)
	}
	if rec.ID != "77777" {
		t.Errorf("expected ID=77777 from existing record, got %q", rec.ID)
	}
}

func TestCreateRecord_APIError(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			// FindRecord → no records
			_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","rrs":[]}]}}`))
			return
		}
		// CreateRecord → API error
		_, _ = w.Write([]byte(`{"result":"error","error_code":"ACCESS_DENIED_TO_OBJECT","error_text":"No permission for this zone"}`))
	})

	err := adapter.CreateRecord("example.com", &Record{Name: "www", Type: "A", Content: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected error for ACCESS_DENIED_TO_OBJECT")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

func TestCreateRecord_DomainLevelError(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","rrs":[]}]}}`))
			return
		}
		// Domain-level error in update_records response
		_, _ = w.Write([]byte(`{
			"result": "success",
			"answer": {"domains": [{"dname": "example.com", "result": "error", "error_code": "INVALID_ACTION", "error_text": "Invalid action"}]}
		}`))
	})

	err := adapter.CreateRecord("example.com", &Record{Name: "www", Type: "A", Content: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected domain-level error")
	}
}

// --- HTTP error classification tests (preserved from original) ---

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
	if errors.Is(err, ErrPermissionDenied) || errors.Is(err, ErrAuthenticationFailed) {
		t.Errorf("500 should not map to permission/auth error, got: %v", err)
	}
}

func TestCreateRecord_APIError_InvalidAuth(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			// FindRecord also fails with INVALID_AUTH
			_, _ = w.Write([]byte(`{"result":"error","error_code":"INVALID_AUTH","error_text":"Bad credentials"}`))
			return
		}
	})

	err := adapter.CreateRecord("example.com", &Record{Name: "www", Type: "A", Content: "1.2.3.4"})
	if err == nil {
		t.Fatal("expected error for INVALID_AUTH")
	}
	if !errors.Is(err, ErrAuthenticationFailed) {
		t.Errorf("expected ErrAuthenticationFailed, got: %v", err)
	}
}

// --- Form encoding validation ---

func TestCreateRecord_UnsupportedRecordType(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		// FindRecord → no records
		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","rrs":[]}]}}`))
	})

	err := adapter.CreateRecord("example.com", &Record{Name: "mx", Type: "MX", Content: "mail.example.com"})
	if err == nil {
		t.Fatal("expected error for unsupported record type MX")
	}
	if !errors.Is(err, ErrUnsupportedRecordType) {
		t.Errorf("expected ErrUnsupportedRecordType, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 API call (FindRecord only), got %d", callCount)
	}
}

func TestCreateRecord_UsesFormEncoding(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded" {
			t.Errorf("call %d: expected form-urlencoded content type, got %q", callCount, ct)
		}
		parseFormRequest(t, r) // will fail if not proper form encoding
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","rrs":[],"service_id":"1"}]}}`))
	})

	rec := &Record{Name: "www", Type: "A", Content: "1.2.3.4"}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}
}

// --- BulkUpdate tests (Story 1.2) ---

func TestBulkUpdate_Success_MultipleActions(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		data := parseFormRequest(t, r)
		domains := data["domains"].([]interface{})
		dom := domains[0].(map[string]interface{})
		actions := dom["action_list"].([]interface{})
		if len(actions) != 3 {
			t.Fatalf("expected 3 actions in batch, got %d", len(actions))
		}
		// Validate action types
		a0 := actions[0].(map[string]interface{})
		if a0["action"] != "add_alias" {
			t.Errorf("action[0]: expected add_alias, got %v", a0["action"])
		}
		a1 := actions[1].(map[string]interface{})
		if a1["action"] != "add_cname" {
			t.Errorf("action[1]: expected add_cname, got %v", a1["action"])
		}
		a2 := actions[2].(map[string]interface{})
		if a2["action"] != "remove_record" {
			t.Errorf("action[2]: expected remove_record, got %v", a2["action"])
		}
		if a2["record_type"] != "A" {
			t.Errorf("action[2]: expected record_type=A, got %v", a2["record_type"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","service_id":"123"}]}}`))
	})

	actions := []BulkAction{
		{Action: "add_alias", Subdomain: "www", Content: "1.2.3.4", RecType: "A"},
		{Action: "add_cname", Subdomain: "cdn", Content: "cdn.example.com", RecType: "CNAME"},
		{Action: "remove_record", Subdomain: "old", Content: "5.6.7.8", RecType: "A"},
	}
	if err := adapter.BulkUpdate("example.com", actions); err != nil {
		t.Fatalf("BulkUpdate failed: %v", err)
	}
}

func TestBulkUpdate_EmptyActions(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(200)
	})

	if err := adapter.BulkUpdate("example.com", nil); err != nil {
		t.Fatalf("BulkUpdate with nil should not error: %v", err)
	}
	if err := adapter.BulkUpdate("example.com", []BulkAction{}); err != nil {
		t.Fatalf("BulkUpdate with empty should not error: %v", err)
	}
	if callCount != 0 {
		t.Errorf("expected 0 API calls for empty batch, got %d", callCount)
	}
}

func TestBulkUpdate_PartialFailure(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"result": "success",
			"answer": {"domains": [{"dname": "example.com", "result": "error", "error_code": "INVALID_ACTION", "error_text": "Invalid action: bad_action"}]}
		}`))
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
		{Action: "add_alias", Subdomain: "www", Content: "1.2.3.4", RecType: "A"},
	}
	err := adapter.BulkUpdate("example.com", actions)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

func TestBulkUpdate_SingleRequest(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success"}]}}`))
	})

	actions := []BulkAction{
		{Action: "add_alias", Subdomain: "a", Content: "1.1.1.1", RecType: "A"},
		{Action: "add_alias", Subdomain: "b", Content: "2.2.2.2", RecType: "A"},
		{Action: "add_txt", Subdomain: "c", Content: "hello", RecType: "TXT"},
	}
	if err := adapter.BulkUpdate("example.com", actions); err != nil {
		t.Fatalf("BulkUpdate failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected exactly 1 API call for batch, got %d", callCount)
	}
}

// --- DeleteRecord tests (Story 1.3) ---

func TestDeleteRecord_Success(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		data := parseFormRequest(t, r)
		// Verify remove_record fields
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
		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","service_id":"12345"}]}}`))
	})

	// id format: subdomain:rectype:content
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
			t.Error("expected no content field when not provided")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success"}]}}`))
	})

	if err := adapter.DeleteRecord("example.com", "old:CNAME"); err != nil {
		t.Fatalf("DeleteRecord without content failed: %v", err)
	}
}

func TestDeleteRecord_InvalidID(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})

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
			t.Errorf("expected /zone/remove_record endpoint, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success"}]}}`))
	})

	if err := adapter.DeleteRecord("example.com", "www:A:1.2.3.4"); err != nil {
		t.Fatalf("DeleteRecord failed: %v", err)
	}
}

// --- UpdateRecord tests (Story 1.4) ---

func TestUpdateRecord_Success_ContentChanged(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		data := parseFormRequest(t, r)
		w.Header().Set("Content-Type", "application/json")

		// Call 1: FindRecord returns existing record with old IP
		if callCount == 1 {
			_, _ = w.Write([]byte(`{
				"result": "success",
				"answer": {"domains": [{"dname": "example.com", "result": "success", "service_id": "12345",
					"rrs": [{"subname": "www", "rectype": "A", "content": "1.1.1.1", "prio": "0"}]
				}]}
			}`))
			return
		}

		// Call 2: update_records with remove + add action_list
		domains := data["domains"].([]interface{})
		dom := domains[0].(map[string]interface{})
		actions := dom["action_list"].([]interface{})
		if len(actions) != 2 {
			t.Fatalf("expected 2 actions (remove+add), got %d", len(actions))
		}
		remove := actions[0].(map[string]interface{})
		if remove["action"] != "remove_record" {
			t.Errorf("expected first action=remove_record, got %v", remove["action"])
		}
		if remove["content"] != "1.1.1.1" {
			t.Errorf("expected remove content=1.1.1.1 (old), got %v", remove["content"])
		}
		add := actions[1].(map[string]interface{})
		if add["action"] != "add_alias" {
			t.Errorf("expected second action=add_alias, got %v", add["action"])
		}
		if add["ipaddr"] != "2.2.2.2" {
			t.Errorf("expected add ipaddr=2.2.2.2 (new), got %v", add["ipaddr"])
		}

		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","service_id":"12345"}]}}`))
	})

	rec := &Record{Name: "www", Type: "A", Content: "2.2.2.2"}
	if err := adapter.UpdateRecord("example.com", rec); err != nil {
		t.Fatalf("UpdateRecord failed: %v", err)
	}
	if rec.ID != "12345" {
		t.Errorf("expected ID=12345, got %q", rec.ID)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls (find + update), got %d", callCount)
	}
}

func TestUpdateRecord_NoChange(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"result": "success",
			"answer": {"domains": [{"dname": "example.com", "result": "success", "service_id": "12345",
				"rrs": [{"subname": "www", "rectype": "A", "content": "1.2.3.4", "prio": "0"}]
			}]}
		}`))
	})

	rec := &Record{Name: "www", Type: "A", Content: "1.2.3.4"}
	if err := adapter.UpdateRecord("example.com", rec); err != nil {
		t.Fatalf("UpdateRecord (no-op) failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 API call (find only — no change), got %d", callCount)
	}
	if rec.ID != "12345" {
		t.Errorf("expected ID=12345, got %q", rec.ID)
	}
}

func TestUpdateRecord_NotExists_FallsBackToCreate(t *testing.T) {
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount <= 2 {
			// Both FindRecord calls (from UpdateRecord and from CreateRecord) return empty
			_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","rrs":[]}]}}`))
			return
		}
		// CreateRecord succeeds
		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","service_id":"99999"}]}}`))
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
	callCount := 0
	adapter := newTestAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		data := parseFormRequest(t, r)
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			_, _ = w.Write([]byte(`{
				"result": "success",
				"answer": {"domains": [{"dname": "example.com", "result": "success", "service_id": "100",
					"rrs": [{"subname": "app", "rectype": "CNAME", "content": "old.example.com", "prio": "0"}]
				}]}
			}`))
			return
		}

		// Verify atomic remove+add
		domains := data["domains"].([]interface{})
		actions := domains[0].(map[string]interface{})["action_list"].([]interface{})
		if len(actions) != 2 {
			t.Fatalf("expected 2 actions for atomic update, got %d", len(actions))
		}

		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","service_id":"100"}]}}`))
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
		_, _ = w.Write([]byte(`{
			"result": "success",
			"answer": {"domains": [{"dname": "example.com", "result": "success", "service_id": "12345",
				"rrs": [{"subname": "www", "rectype": "A", "content": "1.2.3.4", "prio": "0"}]
			}]}
		}`))
	})

	rec, err := adapter.FindRecord("example.com", "www", "A")
	if err != nil {
		t.Fatalf("FindRecord: %v", err)
	}
	if rec == nil {
		t.Fatal("expected record")
	}

	// Cache should now contain the external_id
	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}
	if got := cache.Get(key); got != "12345" {
		t.Errorf("expected cache hit 12345, got %q", got)
	}
}

func TestCacheIntegration_CreateRecord_PopulatesCache(t *testing.T) {
	callCount := 0
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","rrs":[]}]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","service_id":"99999"}]}}`))
	})

	rec := &Record{Name: "www", Type: "A", Content: "1.2.3.4"}
	if err := adapter.CreateRecord("example.com", rec); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}

	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}
	if got := cache.Get(key); got != "99999" {
		t.Errorf("expected cache hit 99999 after create, got %q", got)
	}
}

func TestCacheIntegration_DeleteRecord_EvictsCache(t *testing.T) {
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success"}]}}`))
	})

	// Pre-populate cache
	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}
	cache.Set(key, "12345")

	if err := adapter.DeleteRecord("example.com", "www:A:1.2.3.4"); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}

	if got := cache.Get(key); got != "" {
		t.Errorf("expected cache eviction after delete, got %q", got)
	}
}

func TestCacheIntegration_LookupExternalID_CacheHit(t *testing.T) {
	callCount := 0
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(200)
	})

	// Pre-populate cache
	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}
	cache.Set(key, "cached-id")

	id, err := adapter.LookupExternalID("example.com", "www", "A")
	if err != nil {
		t.Fatalf("LookupExternalID: %v", err)
	}
	if id != "cached-id" {
		t.Errorf("expected cached-id, got %q", id)
	}
	if callCount != 0 {
		t.Errorf("expected 0 API calls (cache hit), got %d", callCount)
	}
}

func TestCacheIntegration_LookupExternalID_CacheMiss_FallsBackToAPI(t *testing.T) {
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"result": "success",
			"answer": {"domains": [{"dname": "example.com", "result": "success", "service_id": 55555,
				"rrs": [{"subname": "www", "rectype": "A", "content": "1.2.3.4", "prio": "0"}]
			}]}
		}`))
	})

	id, err := adapter.LookupExternalID("example.com", "www", "A")
	if err != nil {
		t.Fatalf("LookupExternalID: %v", err)
	}
	if id != "55555" {
		t.Errorf("expected 55555 from fallback, got %q", id)
	}
	// After fallback, cache should be populated
	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}
	if got := cache.Get(key); got != "55555" {
		t.Errorf("expected cache populated after fallback, got %q", got)
	}
}

func TestCacheIntegration_ReconcileCache_EvictsStale(t *testing.T) {
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return empty rrs — record no longer exists
		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","rrs":[]}]}}`))
	})

	// Pre-populate cache with stale entry
	key := CacheKey{Zone: "example.com", Name: "gone", RecType: "A"}
	cache.Set(key, "stale-id")

	ctx := context.Background()
	adapter.ReconcileCache(ctx)

	if got := cache.Get(key); got != "" {
		t.Errorf("expected stale entry evicted, got %q", got)
	}
}

func TestCacheIntegration_ReconcileCache_OneRequestPerZone(t *testing.T) {
	callCount := 0
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		data := parseFormRequest(t, r)
		domains := data["domains"].([]interface{})
		zone := domains[0].(map[string]interface{})["dname"].(string)
		w.Header().Set("Content-Type", "application/json")
		// Return all records for the zone
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

	// Pre-populate: 3 entries in example.com, 1 in other.com = 2 API calls total
	cache.Set(CacheKey{Zone: "example.com", Name: "www", RecType: "A"}, "id1")
	cache.Set(CacheKey{Zone: "example.com", Name: "mail", RecType: "CNAME"}, "id2")
	cache.Set(CacheKey{Zone: "example.com", Name: "gone", RecType: "TXT"}, "id3") // stale
	cache.Set(CacheKey{Zone: "other.com", Name: "app", RecType: "A"}, "id4")

	ctx := context.Background()
	adapter.ReconcileCache(ctx)

	if callCount != 2 {
		t.Errorf("expected 2 API calls (one per zone), got %d", callCount)
	}

	// Live records stay
	if got := cache.Get(CacheKey{Zone: "example.com", Name: "www", RecType: "A"}); got == "" {
		t.Error("expected www/A to remain in cache")
	}
	if got := cache.Get(CacheKey{Zone: "example.com", Name: "mail", RecType: "CNAME"}); got == "" {
		t.Error("expected mail/CNAME to remain in cache")
	}
	if got := cache.Get(CacheKey{Zone: "other.com", Name: "app", RecType: "A"}); got == "" {
		t.Error("expected app/A to remain in cache")
	}
	// Stale record evicted
	if got := cache.Get(CacheKey{Zone: "example.com", Name: "gone", RecType: "TXT"}); got != "" {
		t.Errorf("expected gone/TXT evicted, got %q", got)
	}
}

func TestCacheIntegration_ReconcileCache_ZoneAPIError_SkipsZone(t *testing.T) {
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"error","error_code":"ACCESS_DENIED_TO_OBJECT","error_text":"No permission"}`))
	})

	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}
	cache.Set(key, "keep-me")

	ctx := context.Background()
	adapter.ReconcileCache(ctx)

	// On API error, entries should NOT be evicted (conservative)
	if got := cache.Get(key); got != "keep-me" {
		t.Errorf("expected cache entry preserved on API error, got %q", got)
	}
}

func TestCacheIntegration_UpdateRecord_UpdatesCache(t *testing.T) {
	callCount := 0
	adapter, cache := newTestAdapterWithCache(t, func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			_, _ = w.Write([]byte(`{
				"result": "success",
				"answer": {"domains": [{"dname": "example.com", "result": "success", "service_id": "100",
					"rrs": [{"subname": "www", "rectype": "A", "content": "1.1.1.1", "prio": "0"}]
				}]}
			}`))
			return
		}
		_, _ = w.Write([]byte(`{"result":"success","answer":{"domains":[{"dname":"example.com","result":"success","service_id":"200"}]}}`))
	})

	rec := &Record{Name: "www", Type: "A", Content: "2.2.2.2"}
	if err := adapter.UpdateRecord("example.com", rec); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}

	key := CacheKey{Zone: "example.com", Name: "www", RecType: "A"}
	if got := cache.Get(key); got != "200" {
		t.Errorf("expected cache updated to 200, got %q", got)
	}
}
