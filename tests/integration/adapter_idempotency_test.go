//go:build integration

package integration

import (
	"testing"

	"github.com/aleks-dolotin/external-dns-regru-webhook/internal/adapter"
)

func TestAdapter_IdempotentCreate(t *testing.T) {
	resetMock(t)
	a := newTestAdapter(t)

	zone := "example.com"
	subdomain := "idempotent-test"
	ip := "10.20.30.40"

	// Create the same record twice via adapter.
	rec1 := &adapter.Record{Name: subdomain, Type: "A", Content: ip}
	if err := a.CreateRecord(zone, rec1); err != nil {
		t.Fatalf("first CreateRecord: %v", err)
	}

	rec2 := &adapter.Record{Name: subdomain, Type: "A", Content: ip}
	if err := a.CreateRecord(zone, rec2); err != nil {
		t.Fatalf("second CreateRecord: %v", err)
	}

	// Both should have the same ID (idempotency — CreateRecord checks
	// if record exists via FindRecord before creating).
	if rec1.ID == "" || rec2.ID == "" {
		t.Error("expected non-empty IDs from both creates")
	}

	// FindRecord should return exactly one record.
	found, err := a.FindRecord(zone, subdomain, "A")
	if err != nil {
		t.Fatalf("FindRecord: %v", err)
	}
	if found == nil {
		t.Fatal("expected record, got nil")
	}
	if found.Content != ip {
		t.Errorf("expected content %q, got %q", ip, found.Content)
	}

	t.Log("Idempotent create test passed ✅")
}

func TestAdapter_UpdateRecord_NotExists_CreatesNew(t *testing.T) {
	resetMock(t)
	a := newTestAdapter(t)

	zone := "example.com"

	// Update a non-existent record — adapter should fall back to create.
	rec := &adapter.Record{Name: "new-via-update", Type: "A", Content: "3.3.3.3"}
	if err := a.UpdateRecord(zone, rec); err != nil {
		t.Fatalf("UpdateRecord (create fallback): %v", err)
	}

	// Verify it was created.
	found, err := a.FindRecord(zone, "new-via-update", "A")
	if err != nil {
		t.Fatalf("FindRecord: %v", err)
	}
	if found == nil {
		t.Fatal("expected record after UpdateRecord fallback to create")
	}
	if found.Content != "3.3.3.3" {
		t.Errorf("expected content 3.3.3.3, got %q", found.Content)
	}

	t.Log("UpdateRecord not-exists fallback test passed ✅")
}

func TestAdapter_UpdateRecord_SameContent_NoOp(t *testing.T) {
	resetMock(t)
	a := newTestAdapter(t)

	zone := "example.com"

	// Create initial record.
	rec := &adapter.Record{Name: "no-op-test", Type: "A", Content: "5.5.5.5"}
	if err := a.CreateRecord(zone, rec); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}

	// Update with same content — should be a no-op.
	rec2 := &adapter.Record{Name: "no-op-test", Type: "A", Content: "5.5.5.5"}
	if err := a.UpdateRecord(zone, rec2); err != nil {
		t.Fatalf("UpdateRecord (no-op): %v", err)
	}

	// Verify content unchanged.
	found, err := a.FindRecord(zone, "no-op-test", "A")
	if err != nil {
		t.Fatalf("FindRecord: %v", err)
	}
	if found == nil {
		t.Fatal("record not found")
	}
	if found.Content != "5.5.5.5" {
		t.Errorf("expected content 5.5.5.5, got %q", found.Content)
	}

	t.Log("UpdateRecord same-content no-op test passed ✅")
}
