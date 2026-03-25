package adapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- RetryAfterError tests ---

func TestRetryAfterError_Error(t *testing.T) {
	wrapped := fmt.Errorf("original error")
	err := &RetryAfterError{
		StatusCode: 429,
		Wait:       3 * time.Second,
		Wrapped:    wrapped,
	}
	msg := err.Error()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
	if !contains(msg, "429") {
		t.Errorf("error message should contain status code, got: %s", msg)
	}
	if !contains(msg, "3s") {
		t.Errorf("error message should contain wait duration, got: %s", msg)
	}
}

func TestRetryAfterError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner")
	err := &RetryAfterError{StatusCode: 429, Wait: time.Second, Wrapped: inner}
	if !errors.Is(err, inner) {
		t.Error("expected Unwrap to return inner error")
	}
}

// --- parseRetryAfterDuration tests ---

func TestParseRetryAfterDuration_Seconds(t *testing.T) {
	resp := &http.Response{Header: http.Header{"Retry-After": {"5"}}}
	d := parseRetryAfterDuration(resp)
	if d != 5*time.Second {
		t.Errorf("expected 5s, got %v", d)
	}
}

func TestParseRetryAfterDuration_Empty(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	d := parseRetryAfterDuration(resp)
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestParseRetryAfterDuration_HTTPDate(t *testing.T) {
	future := time.Now().Add(10 * time.Second).UTC().Format(http.TimeFormat)
	resp := &http.Response{Header: http.Header{"Retry-After": {future}}}
	d := parseRetryAfterDuration(resp)
	if d < 5*time.Second || d > 15*time.Second {
		t.Errorf("expected ~10s for HTTP-date, got %v", d)
	}
}

func TestParseRetryAfterDuration_InvalidValue(t *testing.T) {
	resp := &http.Response{Header: http.Header{"Retry-After": {"not-a-number"}}}
	d := parseRetryAfterDuration(resp)
	if d != 0 {
		t.Errorf("expected 0 for invalid value, got %v", d)
	}
}

func TestParseRetryAfterDuration_NegativeSeconds(t *testing.T) {
	resp := &http.Response{Header: http.Header{"Retry-After": {"-5"}}}
	d := parseRetryAfterDuration(resp)
	if d != 0 {
		t.Errorf("expected 0 for negative value, got %v", d)
	}
}

// --- classifyHTTPError tests ---

func TestClassifyHTTPError_429_ReturnsRetryAfterError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(429)
		_, _ = w.Write([]byte("rate limited"))
	}))
	defer srv.Close()

	resp, err := http.Post(srv.URL, "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	httpErr := classifyHTTPError(resp)
	if httpErr == nil {
		t.Fatal("expected error for 429")
	}
	var rae *RetryAfterError
	if !errors.As(httpErr, &rae) {
		t.Fatalf("expected RetryAfterError, got %T: %v", httpErr, httpErr)
	}
	if rae.StatusCode != 429 {
		t.Errorf("expected status 429, got %d", rae.StatusCode)
	}
	if rae.Wait != 2*time.Second {
		t.Errorf("expected 2s wait, got %v", rae.Wait)
	}
}

func TestClassifyHTTPError_503_ReturnsRetryAfterError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "10")
		w.WriteHeader(503)
		_, _ = w.Write([]byte("service unavailable"))
	}))
	defer srv.Close()

	resp, err := http.Post(srv.URL, "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	httpErr := classifyHTTPError(resp)
	var rae *RetryAfterError
	if !errors.As(httpErr, &rae) {
		t.Fatalf("expected RetryAfterError for 503, got %T", httpErr)
	}
	if rae.StatusCode != 503 {
		t.Errorf("expected 503, got %d", rae.StatusCode)
	}
}

func TestClassifyHTTPError_200_ReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	resp, err := http.Post(srv.URL, "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if httpErr := classifyHTTPError(resp); httpErr != nil {
		t.Errorf("expected nil for 200, got %v", httpErr)
	}
}

// --- authParams tests ---

func TestAuthParams_NilDriver(t *testing.T) {
	adapter := &HTTPAdapter{authDriver: nil}
	params := adapter.authParams()
	if len(params) != 0 {
		t.Errorf("expected empty params for nil driver, got %v", params)
	}
}

type failingDriver struct{}

func (f *failingDriver) PrepareAuth() (map[string]string, error) {
	return nil, fmt.Errorf("auth failure")
}

func TestAuthParams_DriverError(t *testing.T) {
	adapter := &HTTPAdapter{authDriver: &failingDriver{}}
	params := adapter.authParams()
	if len(params) != 0 {
		t.Errorf("expected empty params on driver error, got %v", params)
	}
}

type successDriver struct{}

func (s *successDriver) PrepareAuth() (map[string]string, error) {
	return map[string]string{"username": "test", "password": "secret"}, nil
}

func TestAuthParams_DriverSuccess(t *testing.T) {
	adapter := &HTTPAdapter{authDriver: &successDriver{}}
	params := adapter.authParams()
	if params["username"] != "test" || params["password"] != "secret" {
		t.Errorf("expected test/secret, got %v", params)
	}
}

// --- classifyAPIError edge cases ---

func TestClassifyAPIError_InvalidJSON(t *testing.T) {
	err := classifyAPIError([]byte("not json"))
	if err != nil {
		t.Errorf("expected nil for unparseable JSON, got %v", err)
	}
}

func TestClassifyAPIError_SuccessResult(t *testing.T) {
	err := classifyAPIError([]byte(`{"result":"success"}`))
	if err != nil {
		t.Errorf("expected nil for success result, got %v", err)
	}
}

// --- ResourceRecord prio field: string vs number ---

func TestResourceRecord_UnmarshalJSON_PrioAsString(t *testing.T) {
	data := []byte(`{"subname":"www","rectype":"A","content":"1.2.3.4","prio":"10","state":"A"}`)
	var rr ResourceRecord
	if err := json.Unmarshal(data, &rr); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if rr.Priority != "10" {
		t.Errorf("expected priority '10', got %q", rr.Priority)
	}
	if rr.Subname != "www" || rr.Content != "1.2.3.4" {
		t.Errorf("unexpected fields: %+v", rr)
	}
}

func TestResourceRecord_UnmarshalJSON_PrioAsNumber(t *testing.T) {
	data := []byte(`{"subname":"mail","rectype":"MX","content":"mx.example.com","prio":5,"state":"A"}`)
	var rr ResourceRecord
	if err := json.Unmarshal(data, &rr); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if rr.Priority != "5" {
		t.Errorf("expected priority '5', got %q", rr.Priority)
	}
}

func TestResourceRecord_UnmarshalJSON_PrioZeroNumber(t *testing.T) {
	data := []byte(`{"subname":"@","rectype":"A","content":"10.0.0.1","prio":0}`)
	var rr ResourceRecord
	if err := json.Unmarshal(data, &rr); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if rr.Priority != "0" {
		t.Errorf("expected priority '0', got %q", rr.Priority)
	}
}

func TestResourceRecord_UnmarshalJSON_PrioZeroString(t *testing.T) {
	data := []byte(`{"subname":"@","rectype":"A","content":"10.0.0.1","prio":"0"}`)
	var rr ResourceRecord
	if err := json.Unmarshal(data, &rr); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if rr.Priority != "0" {
		t.Errorf("expected priority '0', got %q", rr.Priority)
	}
}

func TestResourceRecord_UnmarshalJSON_PrioMissing(t *testing.T) {
	data := []byte(`{"subname":"@","rectype":"A","content":"10.0.0.1"}`)
	var rr ResourceRecord
	if err := json.Unmarshal(data, &rr); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if rr.Priority != "" {
		t.Errorf("expected empty priority, got %q", rr.Priority)
	}
}

func TestResourceRecord_UnmarshalJSON_FullResponse(t *testing.T) {
	// Simulate real Reg.ru production response with numeric prio and service_id.
	data := []byte(`{
		"result": "success",
		"answer": {
			"domains": [{
				"dname": "example.com",
				"result": "success",
				"service_id": 12345,
				"rrs": [
					{"subname":"@","rectype":"A","content":"1.2.3.4","prio":0,"state":"A"},
					{"subname":"mail","rectype":"MX","content":"mx.example.com","prio":10,"state":"A"}
				]
			}]
		}
	}`)
	var resp ReguResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal full response: %v", err)
	}
	if len(resp.Answer.Domains) != 1 {
		t.Fatalf("expected 1 domain, got %d", len(resp.Answer.Domains))
	}
	rrs := resp.Answer.Domains[0].Rrs
	if len(rrs) != 2 {
		t.Fatalf("expected 2 rrs, got %d", len(rrs))
	}
	if rrs[0].Priority != "0" {
		t.Errorf("rrs[0] expected prio '0', got %q", rrs[0].Priority)
	}
	if rrs[1].Priority != "10" {
		t.Errorf("rrs[1] expected prio '10', got %q", rrs[1].Priority)
	}
}

func TestClassifyAPIError_UnknownErrorCode(t *testing.T) {
	err := classifyAPIError([]byte(`{"result":"error","error_code":"SOME_UNKNOWN","error_text":"something"}`))
	if err == nil {
		t.Fatal("expected error for unknown code")
	}
	if errors.Is(err, ErrPermissionDenied) || errors.Is(err, ErrAuthenticationFailed) {
		t.Error("unknown code should not map to sentinel errors")
	}
}

func TestClassifyAPIError_NoSuchUser(t *testing.T) {
	err := classifyAPIError([]byte(`{"result":"error","error_code":"NO_SUCH_USER","error_text":"user not found"}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("NO_SUCH_USER should map to ErrPermissionDenied, got %v", err)
	}
}

// --- Table-driven record type tests ---

func TestRecordTypeAction_TableDriven(t *testing.T) {
	tests := []struct {
		recType  string
		expected string
		wantErr  bool
	}{
		{"A", "add_alias", false},
		{"AAAA", "add_aaaa", false},
		{"CNAME", "add_cname", false},
		{"TXT", "add_txt", false},
		{"a", "add_alias", false},     // case-insensitive
		{"cname", "add_cname", false}, // case-insensitive
		{"MX", "", true},              // unsupported
		{"SRV", "", true},             // unsupported
		{"NS", "", true},              // unsupported
	}

	for _, tt := range tests {
		t.Run(tt.recType, func(t *testing.T) {
			action, err := recordTypeAction(tt.recType)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %s", tt.recType)
				}
				if !errors.Is(err, ErrUnsupportedRecordType) {
					t.Errorf("expected ErrUnsupportedRecordType for %s, got %v", tt.recType, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for %s: %v", tt.recType, err)
			}
			if action != tt.expected {
				t.Errorf("for %s: expected %q, got %q", tt.recType, tt.expected, action)
			}
		})
	}
}

func TestBuildActionPayload_TableDriven(t *testing.T) {
	tests := []struct {
		recType     string
		action      string
		content     string
		expectedKey string
	}{
		{"A", "add_alias", "1.2.3.4", "ipaddr"},
		{"AAAA", "add_aaaa", "::1", "ipaddr"},
		{"CNAME", "add_cname", "target.example.com", "canonical_name"},
		{"TXT", "add_txt", "v=spf1", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.recType, func(t *testing.T) {
			payload := buildActionPayload(tt.action, "www", tt.content, tt.recType)
			if payload["action"] != tt.action {
				t.Errorf("expected action %q, got %v", tt.action, payload["action"])
			}
			if payload["subdomain"] != "www" {
				t.Errorf("expected subdomain www, got %v", payload["subdomain"])
			}
			if payload[tt.expectedKey] != tt.content {
				t.Errorf("expected %s=%q, got %v", tt.expectedKey, tt.content, payload[tt.expectedKey])
			}
		})
	}
}

// --- Malformed response tests ---

func TestDoRequest_MalformedJSON(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not valid json}`))
	})

	_, err := adapter.FindRecord("example.com", "www", "A")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestDoRequest_EmptyBody(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		// empty body
	})

	_, err := adapter.FindRecord("example.com", "www", "A")
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestDoRequest_ResultNotSuccess(t *testing.T) {
	adapter := newTestAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"pending"}`))
	})

	_, err := adapter.FindRecord("example.com", "www", "A")
	if err == nil {
		t.Fatal("expected error for non-success result")
	}
}

// --- Cache Keys() test ---

func TestExternalIDCache_Keys(t *testing.T) {
	c := NewExternalIDCache()
	c.Set(CacheKey{Zone: "a.com", Name: "www", RecType: "A"}, "1")
	c.Set(CacheKey{Zone: "b.com", Name: "mail", RecType: "CNAME"}, "2")

	keys := c.Keys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

// helper
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
