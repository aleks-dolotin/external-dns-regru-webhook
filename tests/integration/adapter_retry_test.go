//go:build integration

package integration

import (
	"errors"
	"os"
	"testing"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
)

func TestAdapter_FindRecord_Returns429Error(t *testing.T) {
	// Use ?simulate_429=true query param. The mock checks for this in the URL path,
	// but the adapter constructs the URL from baseURL + endpoint. We can set
	// REGRU_BASE_URL to include the query param for this test.
	//
	// Alternative approach: use the mock's simulate_429 query param directly.
	// Since the Go adapter does not support query params, we test via the
	// mock's env-based simulation or accept that 429 testing at the adapter
	// level is covered by unit tests (regru_http_edge_test.go).
	//
	// For this integration test, we verify that the adapter properly returns
	// an error when the mock returns 429 via env-based simulation.
	// This test is meaningful ONLY when MOCK_SIMULATE_429=true is set on the mock.
	//
	// If 429 simulation is not active, we verify normal behavior instead.

	resetMock(t)
	a := newTestAdapter(t)

	_, err := a.FindRecord("example.com", "test-429", "A")

	if os.Getenv("MOCK_SIMULATE_429") == "true" {
		// Mock is in 429 mode — adapter should return a RetryAfterError.
		if err == nil {
			t.Fatal("expected error when mock returns 429, got nil")
		}
		var rae *adapter.RetryAfterError
		if !errors.As(err, &rae) {
			t.Fatalf("expected RetryAfterError, got %T: %v", err, err)
		}
		if rae.StatusCode != 429 {
			t.Errorf("expected status 429, got %d", rae.StatusCode)
		}
		if rae.Wait <= 0 {
			t.Error("expected positive Retry-After duration")
		}
		t.Logf("Adapter correctly received 429 with Retry-After=%v ✅", rae.Wait)
	} else {
		// Normal mode — should succeed (record not found is fine).
		if err != nil {
			t.Fatalf("FindRecord in normal mode: unexpected error: %v", err)
		}
		t.Log("Adapter FindRecord in normal mode passed ✅ (429 env not set)")
	}
}

func TestAdapter_ResponseFormat_ValidReguStructure(t *testing.T) {
	// Verify that the mock returns responses the adapter can parse correctly.
	resetMock(t)
	a := newTestAdapter(t)

	zone := "format-test.com"

	// Create a record through the adapter.
	rec := &adapter.Record{Name: "verify", Type: "A", Content: "1.1.1.1"}
	if err := a.CreateRecord(zone, rec); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}

	// service_id from mock should be populated as record ID.
	if rec.ID == "" {
		t.Error("expected non-empty ID (service_id) from mock response")
	}

	// FindRecord should parse the Reg.ru response format correctly.
	found, err := a.FindRecord(zone, "verify", "A")
	if err != nil {
		t.Fatalf("FindRecord: %v", err)
	}
	if found == nil {
		t.Fatal("expected record, got nil")
	}
	if found.Name != "verify" {
		t.Errorf("expected name=verify, got %q", found.Name)
	}
	if found.Type != "A" {
		t.Errorf("expected type=A, got %q", found.Type)
	}
	if found.Content != "1.1.1.1" {
		t.Errorf("expected content=1.1.1.1, got %q", found.Content)
	}

	t.Log("Response format validation test passed ✅")
}
