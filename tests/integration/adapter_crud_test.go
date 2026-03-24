//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
)

// newTestAdapter creates an HTTPAdapter pointing at the mock server.
// REGRU_BASE_URL must be set (e.g. http://127.0.0.1:8081/api/regru2).
func newTestAdapter(t *testing.T) *adapter.HTTPAdapter {
	t.Helper()
	base := os.Getenv("REGRU_BASE_URL")
	if base == "" {
		t.Fatal("REGRU_BASE_URL must be set to mock server URL")
	}
	// HTTPAdapter reads REGRU_BASE_URL from env in NewHTTPAdapter.
	return adapter.NewHTTPAdapter(nil)
}

// resetMock sends DELETE /reset to clear mock server state.
func resetMock(t *testing.T) {
	t.Helper()
	base := os.Getenv("REGRU_BASE_URL")
	hostURL := strings.TrimSuffix(base, "/api/regru2")

	req, err := http.NewRequest(http.MethodDelete, hostURL+"/reset", nil)
	if err != nil {
		t.Fatalf("create reset request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("reset mock: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("reset mock: unexpected status %d", resp.StatusCode)
	}
}

// seedRecord creates a record directly via HTTP POST so we can test
// adapter.FindRecord/DeleteRecord against known state.
func seedRecord(t *testing.T, zone, subdomain, action, fieldKey, value string) {
	t.Helper()
	base := os.Getenv("REGRU_BASE_URL")

	inputData := map[string]interface{}{
		"domains": []map[string]interface{}{{
			"dname": zone,
			"action_list": []map[string]interface{}{{
				"action":    action,
				"subdomain": subdomain,
				fieldKey:    value,
			}},
		}},
	}
	jsonBytes, _ := json.Marshal(inputData)
	form := url.Values{}
	form.Set("input_format", "json")
	form.Set("input_data", string(jsonBytes))

	resp, err := http.Post(base+"/zone/update_records", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("seed record: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("seed record: status %d", resp.StatusCode)
	}
}

func TestAdapter_CreateRecord_ThenFindRecord(t *testing.T) {
	resetMock(t)
	a := newTestAdapter(t)

	zone := "example.com"
	rec := &adapter.Record{Name: "www", Type: "A", Content: "1.2.3.4"}

	// Create via adapter.
	if err := a.CreateRecord(zone, rec); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}

	// ID should be populated (service_id from mock).
	if rec.ID == "" {
		t.Error("expected non-empty ID after CreateRecord")
	}

	// Find via adapter — should return the record we just created.
	found, err := a.FindRecord(zone, "www", "A")
	if err != nil {
		t.Fatalf("FindRecord: %v", err)
	}
	if found == nil {
		t.Fatal("FindRecord returned nil — expected record created above")
	}
	if found.Content != "1.2.3.4" {
		t.Errorf("expected content 1.2.3.4, got %q", found.Content)
	}

	t.Log("CreateRecord → FindRecord test passed ✅")
}

func TestAdapter_UpdateRecord_ContentChange(t *testing.T) {
	resetMock(t)
	a := newTestAdapter(t)

	zone := "example.com"

	// Create initial record via adapter.
	rec := &adapter.Record{Name: "app", Type: "A", Content: "10.0.0.1"}
	if err := a.CreateRecord(zone, rec); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}

	// Update content via adapter.
	rec.Content = "10.0.0.2"
	if err := a.UpdateRecord(zone, rec); err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}

	// Verify updated content.
	found, err := a.FindRecord(zone, "app", "A")
	if err != nil {
		t.Fatalf("FindRecord after update: %v", err)
	}
	if found == nil {
		t.Fatal("record not found after update")
	}
	if found.Content != "10.0.0.2" {
		t.Errorf("expected content 10.0.0.2, got %q", found.Content)
	}

	t.Log("UpdateRecord content change test passed ✅")
}

func TestAdapter_DeleteRecord(t *testing.T) {
	resetMock(t)
	a := newTestAdapter(t)

	zone := "example.com"

	// Create via adapter.
	rec := &adapter.Record{Name: "del-test", Type: "A", Content: "9.9.9.9"}
	if err := a.CreateRecord(zone, rec); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}

	// Verify exists.
	found, err := a.FindRecord(zone, "del-test", "A")
	if err != nil {
		t.Fatalf("FindRecord before delete: %v", err)
	}
	if found == nil {
		t.Fatal("record not found before delete")
	}

	// Delete via adapter (id format: subdomain:rectype:content).
	if err := a.DeleteRecord(zone, "del-test:A:9.9.9.9"); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}

	// Verify gone.
	found, err = a.FindRecord(zone, "del-test", "A")
	if err != nil {
		t.Fatalf("FindRecord after delete: %v", err)
	}
	if found != nil {
		t.Fatal("record still exists after DeleteRecord")
	}

	t.Log("DeleteRecord test passed ✅")
}

func TestAdapter_FullCRUDLifecycle(t *testing.T) {
	resetMock(t)
	a := newTestAdapter(t)

	zone := "example.com"

	// CREATE
	rec := &adapter.Record{Name: "lifecycle", Type: "CNAME", Content: "target.example.com"}
	if err := a.CreateRecord(zone, rec); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// READ
	found, err := a.FindRecord(zone, "lifecycle", "CNAME")
	if err != nil {
		t.Fatalf("Find after create: %v", err)
	}
	if found == nil || found.Content != "target.example.com" {
		t.Fatalf("unexpected find result: %+v", found)
	}

	// UPDATE
	rec.Content = "new-target.example.com"
	if err := a.UpdateRecord(zone, rec); err != nil {
		t.Fatalf("Update: %v", err)
	}
	found, err = a.FindRecord(zone, "lifecycle", "CNAME")
	if err != nil {
		t.Fatalf("Find after update: %v", err)
	}
	if found == nil || found.Content != "new-target.example.com" {
		t.Fatalf("unexpected find result after update: %+v", found)
	}

	// DELETE
	if err := a.DeleteRecord(zone, "lifecycle:CNAME:new-target.example.com"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	found, err = a.FindRecord(zone, "lifecycle", "CNAME")
	if err != nil {
		t.Fatalf("Find after delete: %v", err)
	}
	if found != nil {
		t.Fatal("record still exists after delete")
	}

	t.Log("Full CRUD lifecycle test passed ✅")
}

func TestAdapter_MultipleRecordTypes(t *testing.T) {
	resetMock(t)
	a := newTestAdapter(t)

	zone := "example.com"

	records := []struct {
		name    string
		recType string
		content string
	}{
		{"a-test", "A", "10.0.0.1"},
		{"aaaa-test", "AAAA", "::1"},
		{"cname-test", "CNAME", "other.example.com"},
		{"txt-test", "TXT", "v=spf1 include:example.com"},
	}

	// Create all.
	for _, r := range records {
		rec := &adapter.Record{Name: r.name, Type: r.recType, Content: r.content}
		if err := a.CreateRecord(zone, rec); err != nil {
			t.Fatalf("CreateRecord %s/%s: %v", r.name, r.recType, err)
		}
	}

	// Verify all via adapter.FindRecord.
	for _, r := range records {
		found, err := a.FindRecord(zone, r.name, r.recType)
		if err != nil {
			t.Fatalf("FindRecord %s/%s: %v", r.name, r.recType, err)
		}
		if found == nil {
			t.Errorf("FindRecord %s/%s: expected record, got nil", r.name, r.recType)
			continue
		}
		if found.Content != r.content {
			t.Errorf("FindRecord %s/%s: expected content %q, got %q", r.name, r.recType, r.content, found.Content)
		}
	}

	t.Log("Multiple record types test passed ✅")
}
